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

// Package imageaccel provides the vendor-agnostic client interface and gRPC
// implementation used by atelet to communicate with an out-of-process
// ImageAccelerator DaemonSet over a Unix domain socket.
package imageaccel

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
	imageaccelv1 "go.ate.dev/substrate/oci-image-accelerator/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// DefaultSocketPath is the standard Unix domain socket path where the
// ImageAccelerator daemon listens.
const DefaultSocketPath = "/run/ate/image-accelerator.sock"

// ErrNotSupported reports that the ImageAccelerator declined to serve
// an image. Callers should fall back to local imagecache.
var ErrNotSupported = errors.New("image not supported by ImageAccelerator")

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

	// ProviderName is the underlying provider identifier reported by the accelerator.
	ProviderName string
}

// ReleaseRequest encapsulates the parameters for ReleaseImage.
type ReleaseRequest struct {
	ImageRef string

	// BypassTTL bypasses idle TTL caching and cleans up mounts immediately.
	BypassTTL bool
}

// ActiveImageLayers models active streamed image leases reported during startup reconciliation.
type ActiveImageLayers struct {
	ImageRef    string
	ImageDigest string
	LayerDirs   []string
	RefCount    int
}

// Client is the vendor-agnostic interface through which atelet interacts
// with an ImageAccelerator daemon.
type Client interface {
	Name() string
	CanProvide(ctx context.Context, req *PrepareRequest) (bool, error)
	IsEvicting(ctx context.Context) (bool, error)
	PrepareImage(ctx context.Context, req *PrepareRequest) (*PrepareResult, error)
	ReleaseImage(ctx context.Context, req *ReleaseRequest) error
	ReconcileImages(ctx context.Context, active []*ActiveImageLayers) error
	Close() error
}

// GRPCClient implements Client against an out-of-process ImageAccelerator daemon
// over a Unix domain socket.
type GRPCClient struct {
	mu           sync.RWMutex
	socketPath   string
	conn         *grpc.ClientConn
	client       imageaccelv1.ImageAcceleratorClient
	providerName string
}

var _ Client = (*GRPCClient)(nil)

// NewGRPCClient dials the ImageAccelerator gRPC server at socketPath.
func NewGRPCClient(socketPath string) (*GRPCClient, error) {
	if socketPath == "" {
		return nil, errors.New("image-accelerator socket path cannot be empty")
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
		return nil, fmt.Errorf("dialing image-accelerator socket %s: %w", socketPath, err)
	}
	return &GRPCClient{
		socketPath:   socketPath,
		conn:         conn,
		client:       imageaccelv1.NewImageAcceleratorClient(conn),
		providerName: "image-accelerator",
	}, nil
}

// Name returns the provider name reported by the accelerator.
func (c *GRPCClient) Name() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.providerName
}

func (c *GRPCClient) resolveAuth(ctx context.Context, req *PrepareRequest) *imageaccelv1.RegistryAuth {
	if req == nil {
		return nil
	}
	if req.AuthConfig != nil {
		return &imageaccelv1.RegistryAuth{
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
	return &imageaccelv1.RegistryAuth{
		Username:      cfg.Username,
		Password:      cfg.Password,
		Auth:          cfg.Auth,
		ServerAddress: ref.Context().RegistryStr(),
		IdentityToken: cfg.IdentityToken,
		RegistryToken: cfg.RegistryToken,
	}
}

// CanProvide checks whether the accelerator supports the requested image.
// If the adapter socket is not present on the node, it returns (false, nil)
// so atelet automatically detects when an adapter is installed or absent.
func (c *GRPCClient) CanProvide(ctx context.Context, req *PrepareRequest) (bool, error) {
	if req == nil {
		return false, errors.New("prepare request cannot be nil")
	}
	if !IsSocketAvailable(c.socketPath) {
		return false, nil
	}
	res, err := c.client.CanProvide(ctx, &imageaccelv1.CanProvideRequest{
		ImageRef: req.ImageRef,
		Auth:     c.resolveAuth(ctx, req),
	})
	if err != nil {
		return false, fmt.Errorf("calling CanProvide on image-accelerator: %w", err)
	}
	if res.GetProviderName() != "" {
		c.mu.Lock()
		c.providerName = res.GetProviderName()
		c.mu.Unlock()
	}
	return res.GetCanProvide(), nil
}

// IsEvicting reports whether the accelerator is in lame-duck drain mode.
func (c *GRPCClient) IsEvicting(ctx context.Context) (bool, error) {
	if !IsSocketAvailable(c.socketPath) {
		return false, nil
	}
	res, err := c.client.CanProvide(ctx, &imageaccelv1.CanProvideRequest{})
	if err != nil {
		return false, fmt.Errorf("calling CanProvide on image-accelerator: %w", err)
	}
	if res.GetProviderName() != "" {
		c.mu.Lock()
		c.providerName = res.GetProviderName()
		c.mu.Unlock()
	}
	return res.GetEvicting(), nil
}

// PrepareImage prepares the layers for the image and returns their paths.
func (c *GRPCClient) PrepareImage(ctx context.Context, req *PrepareRequest) (*PrepareResult, error) {
	if req == nil {
		return nil, errors.New("prepare request cannot be nil")
	}
	res, err := c.client.PrepareImage(ctx, &imageaccelv1.PrepareImageRequest{
		ImageRef: req.ImageRef,
		Auth:     c.resolveAuth(ctx, req),
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && (st.Code() == codes.FailedPrecondition || st.Code() == codes.Unimplemented) {
			return nil, fmt.Errorf("%w: %s", ErrNotSupported, st.Message())
		}
		return nil, fmt.Errorf("calling PrepareImage on image-accelerator: %w", err)
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
	_, err := c.client.ReleaseImage(ctx, &imageaccelv1.ReleaseImageRequest{
		ImageRef:  req.ImageRef,
		BypassTtl: req.BypassTTL,
	})
	if err != nil {
		return fmt.Errorf("calling ReleaseImage on image-accelerator: %w", err)
	}
	return nil
}

// ReconcileImages synchronizes active leases after a restart.
func (c *GRPCClient) ReconcileImages(ctx context.Context, active []*ActiveImageLayers) error {
	req := &imageaccelv1.ReconcileImagesRequest{
		ActiveImages: make([]*imageaccelv1.ActiveImageLayers, 0, len(active)),
	}
	for _, a := range active {
		if a == nil {
			continue
		}
		req.ActiveImages = append(req.ActiveImages, &imageaccelv1.ActiveImageLayers{
			ImageRef:    a.ImageRef,
			ImageDigest: a.ImageDigest,
			LayerDirs:   a.LayerDirs,
			RefCount:    int32(a.RefCount),
		})
	}
	_, err := c.client.ReconcileImages(ctx, req)
	if err != nil {
		return fmt.Errorf("calling ReconcileImages on image-accelerator: %w", err)
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
