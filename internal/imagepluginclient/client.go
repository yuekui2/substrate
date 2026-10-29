// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package imagepluginclient provides the vendor-agnostic client interface and gRPC
// implementation used by atelet to communicate with an out-of-process
// ImagePlugin DaemonSet over a Unix domain socket.
package imagepluginclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"go.ate.dev/substrate/api/imageplugin/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// DefaultSocketPath is the standard Unix domain socket path where the
// ImagePlugin daemon listens.
const DefaultSocketPath = "/run/ate/image-plugin.sock"

// ErrNotSupported reports that the ImagePlugin declined to serve
// an image. Callers should fall back to local imagecache.
var ErrNotSupported = errors.New("image not supported by ImagePlugin")

type contextKeyKeychain struct{}

// WithKeychainContext returns a child context carrying the specified authn.Keychain.
func WithKeychainContext(ctx context.Context, k authn.Keychain) context.Context {
	return context.WithValue(ctx, contextKeyKeychain{}, k)
}

// KeychainFromContext extracts an authn.Keychain from the context if present.
func KeychainFromContext(ctx context.Context) authn.Keychain {
	if ctx == nil {
		return nil
	}
	if k, ok := ctx.Value(contextKeyKeychain{}).(authn.Keychain); ok {
		return k
	}
	return nil
}

// AuthConfig contains resolved registry credentials for one image request.
type AuthConfig struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Auth          string `json:"auth,omitempty"`
	ServerAddress string `json:"serveraddress,omitempty"`
	IdentityToken string `json:"identitytoken,omitempty"`
	RegistryToken string `json:"registrytoken,omitempty"`
}

// PrepareRequest encapsulates the parameters for CanProvide and PrepareImage.
type PrepareRequest struct {
	// ImageRef is the OCI image reference.
	ImageRef string

	// AuthConfig holds explicit workload credentials (e.g. imagePullSecrets).
	AuthConfig *AuthConfig
}

// PrepareResult holds the prepared read-only host layer directories for an OCI image.
type PrepareResult struct {
	// ImageDigest is the resolved manifest digest ("sha256:<hex>").
	ImageDigest string

	// Config is the decoded OCI image configuration (entrypoint, cmd, env, etc.).
	Config *v1.Config

	// LayerDirs contains the complete ordered list of read-only host layer
	// directories (bottom-most base layer first, matching OCI manifest order).
	LayerDirs []string

	// ProviderName is the underlying provider identifier reported by the plugin.
	ProviderName string
}

// ReleaseRequest encapsulates the parameters for ReleaseImage.
type ReleaseRequest struct {
	ImageRef string
}

// ActiveImageLayers models active streamed image leases reported during startup reconciliation.
type ActiveImageLayers struct {
	ImageRef    string
	ImageDigest string
	LayerDirs   []string
	RefCount    int
}

// Client is the vendor-agnostic interface through which atelet interacts
// with an ImagePlugin daemon.
type Client interface {
	Name() string
	CanProvide(ctx context.Context, req *PrepareRequest) (bool, error)
	State(ctx context.Context) (imageplugin.PluginState, error)
	PrepareImage(ctx context.Context, req *PrepareRequest) (*PrepareResult, error)
	ReleaseImage(ctx context.Context, req *ReleaseRequest) error
	ReconcileImages(ctx context.Context, active []*ActiveImageLayers) error
	Close() error
}

// GRPCClient implements Client against an out-of-process ImagePlugin daemon
// over a Unix domain socket.
type GRPCClient struct {
	mu           sync.RWMutex
	socketPath   string
	conn         *grpc.ClientConn
	client       imageplugin.ImagePluginClient
	providerName string
}

var _ Client = (*GRPCClient)(nil)

// NewGRPCClient dials the ImagePlugin gRPC server at socketPath.
func NewGRPCClient(socketPath string) (*GRPCClient, error) {
	if socketPath == "" {
		return nil, errors.New("image-plugin socket path cannot be empty")
	}
	target := socketPath
	if !strings.HasPrefix(target, "unix://") {
		target = "unix://" + target
	}
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dialing image-plugin socket %s: %w", socketPath, err)
	}
	return &GRPCClient{
		socketPath:   socketPath,
		conn:         conn,
		client:       imageplugin.NewImagePluginClient(conn),
		providerName: "image-plugin",
	}, nil
}

// Name returns the provider name reported by the plugin.
func (c *GRPCClient) Name() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.providerName
}

func (c *GRPCClient) resolveAuth(ctx context.Context, req *PrepareRequest) *imageplugin.RegistryAuth {
	if req == nil {
		return nil
	}
	if req.AuthConfig != nil {
		return &imageplugin.RegistryAuth{
			Username:      req.AuthConfig.Username,
			Password:      req.AuthConfig.Password,
			Auth:          req.AuthConfig.Auth,
			ServerAddress: req.AuthConfig.ServerAddress,
			IdentityToken: req.AuthConfig.IdentityToken,
			RegistryToken: req.AuthConfig.RegistryToken,
		}
	}
	k := KeychainFromContext(ctx)
	if k == nil {
		return nil
	}
	ref, err := name.ParseReference(req.ImageRef)
	if err != nil {
		return nil
	}
	auth, err := k.Resolve(ref.Context())
	if err != nil || auth == nil {
		return nil
	}
	cfg, err := auth.Authorization()
	if err != nil || cfg == nil {
		return nil
	}
	if cfg.Username == "" && cfg.Password == "" && cfg.Auth == "" && cfg.IdentityToken == "" && cfg.RegistryToken == "" {
		return nil
	}
	return &imageplugin.RegistryAuth{
		Username:      cfg.Username,
		Password:      cfg.Password,
		Auth:          cfg.Auth,
		ServerAddress: ref.Context().RegistryStr(),
		IdentityToken: cfg.IdentityToken,
		RegistryToken: cfg.RegistryToken,
	}
}

// CanProvide checks whether the plugin supports the requested image.
// If the plugin socket is not present on the node, it returns (false, nil)
// so atelet automatically detects when a plugin is installed or absent.
func (c *GRPCClient) CanProvide(ctx context.Context, req *PrepareRequest) (bool, error) {
	if req == nil {
		return false, errors.New("prepare request cannot be nil")
	}
	if !IsSocketAvailable(c.socketPath) {
		return false, nil
	}
	res, err := c.client.CanProvide(ctx, &imageplugin.CanProvideRequest{
		ImageRef: req.ImageRef,
		Auth:     c.resolveAuth(ctx, req),
	})
	if err != nil {
		return false, fmt.Errorf("calling CanProvide on image-plugin: %w", err)
	}
	if res.GetProviderName() != "" {
		c.mu.Lock()
		c.providerName = res.GetProviderName()
		c.mu.Unlock()
	}
	// A plugin that is draining, or in a state this client does not know,
	// is not used for new images.
	return res.GetCanProvide() && res.GetState() == imageplugin.PluginState_PLUGIN_STATE_SERVING, nil
}

// State returns the plugin's lifecycle state. It returns
// PLUGIN_STATE_UNSPECIFIED when the plugin socket is absent.
func (c *GRPCClient) State(ctx context.Context) (imageplugin.PluginState, error) {
	if !IsSocketAvailable(c.socketPath) {
		return imageplugin.PluginState_PLUGIN_STATE_UNSPECIFIED, nil
	}
	res, err := c.client.CanProvide(ctx, &imageplugin.CanProvideRequest{})
	if err != nil {
		return imageplugin.PluginState_PLUGIN_STATE_UNSPECIFIED, fmt.Errorf("calling CanProvide on image-plugin: %w", err)
	}
	if res.GetProviderName() != "" {
		c.mu.Lock()
		c.providerName = res.GetProviderName()
		c.mu.Unlock()
	}
	return res.GetState(), nil
}

// PrepareImage prepares the layers for the image and returns their paths.
func (c *GRPCClient) PrepareImage(ctx context.Context, req *PrepareRequest) (*PrepareResult, error) {
	if req == nil {
		return nil, errors.New("prepare request cannot be nil")
	}
	res, err := c.client.PrepareImage(ctx, &imageplugin.PrepareImageRequest{
		ImageRef: req.ImageRef,
		Auth:     c.resolveAuth(ctx, req),
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && (st.Code() == codes.FailedPrecondition || st.Code() == codes.Unimplemented) {
			return nil, fmt.Errorf("%w: %s", ErrNotSupported, st.Message())
		}
		return nil, fmt.Errorf("calling PrepareImage on image-plugin: %w", err)
	}

	provider := c.Name()
	if res.GetProviderName() != "" {
		provider = res.GetProviderName()
		c.mu.Lock()
		c.providerName = provider
		c.mu.Unlock()
	}

	var cfg *v1.Config
	if len(res.GetConfigJson()) > 0 {
		var parsed v1.Config
		if err := json.Unmarshal(res.GetConfigJson(), &parsed); err == nil {
			cfg = &parsed
		}
	}

	return &PrepareResult{
		ImageDigest:  res.GetImageDigest(),
		Config:       cfg,
		LayerDirs:    res.GetLayerDirs(),
		ProviderName: provider,
	}, nil
}

// ReleaseImage decrements the lease count for an image.
func (c *GRPCClient) ReleaseImage(ctx context.Context, req *ReleaseRequest) error {
	if req == nil {
		return errors.New("release request cannot be nil")
	}
	_, err := c.client.ReleaseImage(ctx, &imageplugin.ReleaseImageRequest{
		ImageRef: req.ImageRef,
	})
	if err != nil {
		return fmt.Errorf("calling ReleaseImage on image-plugin: %w", err)
	}
	return nil
}

// ReconcileImages synchronizes active leases after a restart.
func (c *GRPCClient) ReconcileImages(ctx context.Context, active []*ActiveImageLayers) error {
	req := &imageplugin.ReconcileImagesRequest{
		ActiveImages: make([]*imageplugin.ActiveImageLayers, 0, len(active)),
	}
	for _, a := range active {
		if a == nil {
			continue
		}
		req.ActiveImages = append(req.ActiveImages, &imageplugin.ActiveImageLayers{
			ImageRef:    a.ImageRef,
			ImageDigest: a.ImageDigest,
			LayerDirs:   a.LayerDirs,
			RefCount:    int32(a.RefCount),
		})
	}
	_, err := c.client.ReconcileImages(ctx, req)
	if err != nil {
		return fmt.Errorf("calling ReconcileImages on image-plugin: %w", err)
	}
	return nil
}

// Close closes the underlying gRPC connection.
func (c *GRPCClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// IsSocketAvailable checks if the Unix socket exists on the filesystem.
func IsSocketAvailable(socketPath string) bool {
	if socketPath == "" {
		return false
	}
	clean := strings.TrimPrefix(socketPath, "unix://")
	info, err := os.Stat(clean)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSocket != 0
}
