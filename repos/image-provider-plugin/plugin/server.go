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

package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	imageprovider "github.com/ai-on-gke/image-provider-plugin/api/imageprovider/v1alpha1"
	"github.com/ai-on-gke/image-provider-plugin/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// DefaultListenSocket is the Unix socket a plugin listens on, where
	// atelet looks for it.
	DefaultListenSocket = client.DefaultSocketPath

	// DefaultDrainCompleteFile is the file ServeUnixSocket writes when it
	// stops, for a preStop hook to wait on.
	DefaultDrainCompleteFile = "/run/ate/drain-complete"

	// gracefulStopTimeout bounds how long ServeUnixSocket waits for open RPCs
	// after draining.
	gracefulStopTimeout = 5 * time.Second
)

// Server implements the ImageProviderPlugin service on a Driver.
type Server struct {
	imageprovider.UnimplementedImageProviderPluginServer
	driver *Driver
}

// NewServer returns a Server for driver.
func NewServer(driver *Driver) *Server {
	return &Server{driver: driver}
}

// CanProvide reports whether the driver serves the image, and
// PLUGIN_STATE_DRAINING once the driver drains.
func (s *Server) CanProvide(ctx context.Context, req *imageprovider.CanProvideRequest) (*imageprovider.CanProvideResponse, error) {
	ok, err := s.driver.CanProvide(ctx, &client.PrepareRequest{ImageRef: req.GetImageRef()})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	state := imageprovider.PluginState_PLUGIN_STATE_SERVING
	if s.driver.Draining() {
		state = imageprovider.PluginState_PLUGIN_STATE_DRAINING
	}
	if !isDirsForm(req.GetForm()) {
		ok = false
	}
	return &imageprovider.CanProvideResponse{
		CanProvide:   ok,
		ProviderName: s.driver.Name(),
		State:        state,
	}, nil
}

// isDirsForm reports whether form asks for layer directories.
func isDirsForm(form imageprovider.LayerForm) bool {
	return form == imageprovider.LayerForm_LAYER_FORM_UNSPECIFIED || form == imageprovider.LayerForm_LAYER_FORM_DIRS
}

// PrepareImage prepares the image's layers and takes a lease on them. It
// returns FailedPrecondition when the driver declines the image, and
// Unimplemented for a layer form other than directories.
func (s *Server) PrepareImage(ctx context.Context, req *imageprovider.PrepareImageRequest) (*imageprovider.PrepareImageResponse, error) {
	if !isDirsForm(req.GetForm()) {
		return nil, status.Error(codes.Unimplemented, "only LAYER_FORM_DIRS is supported")
	}
	prepReq := &client.PrepareRequest{ImageRef: req.GetImageRef()}
	if a := req.GetAuth(); a != nil {
		prepReq.AuthConfig = &client.AuthConfig{
			Username:      a.GetUsername(),
			Password:      a.GetPassword(),
			Auth:          a.GetAuth(),
			ServerAddress: a.GetServerAddress(),
			IdentityToken: a.GetIdentityToken(),
			RegistryToken: a.GetRegistryToken(),
		}
	}
	res, err := s.driver.PrepareImage(ctx, prepReq)
	if errors.Is(err, client.ErrNotSupported) {
		return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	cfg, err := json.Marshal(res.Config)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encoding image config: %v", err)
	}
	return &imageprovider.PrepareImageResponse{
		ImageDigest:  res.ImageDigest,
		ConfigJson:   cfg,
		ProviderName: res.ProviderName,
		Layers: &imageprovider.PrepareImageResponse_LayerDirs{
			LayerDirs: &imageprovider.LayerDirs{Dirs: res.LayerDirs},
		},
	}, nil
}

// ReleaseImage releases one lease on the image.
func (s *Server) ReleaseImage(ctx context.Context, req *imageprovider.ReleaseImageRequest) (*imageprovider.ReleaseImageResponse, error) {
	if err := s.driver.ReleaseImage(ctx, &client.ReleaseRequest{ImageRef: req.GetImageRef()}); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &imageprovider.ReleaseImageResponse{}, nil
}

// ReconcileImages replaces the driver's leases with the images in use.
func (s *Server) ReconcileImages(ctx context.Context, req *imageprovider.ReconcileImagesRequest) (*imageprovider.ReconcileImagesResponse, error) {
	active := make([]*client.ActiveImageLayers, 0, len(req.GetActiveImages()))
	for _, img := range req.GetActiveImages() {
		active = append(active, &client.ActiveImageLayers{
			ImageRef:    img.GetImageRef(),
			ImageDigest: img.GetImageDigest(),
			LayerDirs:   img.GetLayerDirs(),
			RefCount:    int(img.GetRefCount()),
		})
	}
	if err := s.driver.ReconcileImages(ctx, active); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &imageprovider.ReconcileImagesResponse{}, nil
}

// ServeUnixSocket serves driver on the Unix socket listenSocket until ctx
// ends. It waits for the provider to pass its health check and prepares the
// work dir before it listens. When ctx ends, it drains the driver while
// still serving, so workloads can release their leases, then stops and
// writes drainCompleteFile, if set.
func ServeUnixSocket(ctx context.Context, listenSocket, drainCompleteFile string, driver *Driver) error {
	sockPath := strings.TrimPrefix(listenSocket, "unix://")
	if drainCompleteFile != "" {
		if err := os.Remove(drainCompleteFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("removing stale drain-complete file: %w", err)
		}
	}
	if err := driver.waitReady(ctx); err != nil {
		return err
	}
	if err := driver.prepareWorkDir(ctx); err != nil {
		return err
	}
	lis, err := ListenUnixSocket(sockPath)
	if err != nil {
		return err
	}
	if err := serve(ctx, lis, driver); err != nil {
		return err
	}
	if drainCompleteFile != "" {
		if err := os.WriteFile(drainCompleteFile, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o644); err != nil {
			return fmt.Errorf("writing drain-complete file: %w", err)
		}
	}
	return nil
}

// ListenUnixSocket listens on sockPath, replacing a stale socket file. The
// listener removes the file when it closes.
func ListenUnixSocket(sockPath string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(sockPath), 0o755); err != nil {
		return nil, fmt.Errorf("creating socket directory: %w", err)
	}
	if err := os.Remove(sockPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("removing stale socket %s: %w", sockPath, err)
	}
	lis, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("listening on %s: %w", sockPath, err)
	}
	return lis, nil
}

// serve serves driver on lis until ctx ends, then drains the driver and
// stops.
func serve(ctx context.Context, lis net.Listener, driver *Driver) error {
	srv := grpc.NewServer()
	imageprovider.RegisterImageProviderPluginServer(srv, NewServer(driver))
	stopped := make(chan struct{})
	serveFailed := make(chan struct{})
	go func() {
		defer close(stopped)
		select {
		case <-ctx.Done():
		case <-serveFailed:
			return
		}
		slog.Info("Shutting down; draining leases", slog.String("provider", driver.Name()))
		driver.Drain(context.WithoutCancel(ctx))
		done := make(chan struct{})
		go func() {
			srv.GracefulStop()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(gracefulStopTimeout):
			srv.Stop()
		}
	}()
	slog.Info("Image provider plugin listening",
		slog.String("provider", driver.Name()),
		slog.String("socket", lis.Addr().String()))
	if err := srv.Serve(lis); err != nil {
		close(serveFailed)
		srv.Stop()
		return fmt.Errorf("serving: %w", err)
	}
	<-stopped
	return nil
}

// ProbeUnixSocket reports an error unless the plugin on listenSocket says it
// can serve images. A readiness probe can run it.
func ProbeUnixSocket(ctx context.Context, listenSocket string) error {
	sockPath := strings.TrimPrefix(listenSocket, "unix://")
	if !client.IsSocketAvailable(sockPath) {
		return fmt.Errorf("no plugin socket at %s", sockPath)
	}
	c, err := client.NewGRPCClient(sockPath)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	ctx, cancel := context.WithTimeout(ctx, 2*DefaultProbeTimeout)
	defer cancel()
	ok, err := c.CanProvide(ctx, &client.PrepareRequest{})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("plugin %s cannot serve images", c.Name())
	}
	return nil
}
