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

// holdsLease reports whether the driver holds the lease leaseID.
func (d *Driver) holdsLease(leaseID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, ok := d.leaseRefs[leaseID]
	return ok
}

// PrepareImage prepares the image's layers and takes the lease on them. It
// returns FailedPrecondition when the driver declines the image,
// AlreadyExists when the lease is held for another image or form, and
// Unimplemented for a layer form other than directories.
func (s *Server) PrepareImage(ctx context.Context, req *imageprovider.PrepareImageRequest) (*imageprovider.PrepareImageResponse, error) {
	if req.GetImageRef() == "" || req.GetLeaseId() == "" {
		return nil, status.Error(codes.InvalidArgument, "image_ref and lease_id are required")
	}
	if !isDirsForm(req.GetForm()) {
		if s.driver.holdsLease(req.GetLeaseId()) {
			return nil, status.Errorf(codes.AlreadyExists, "lease %q is held in form %v", req.GetLeaseId(), imageprovider.LayerForm_LAYER_FORM_DIRS)
		}
		return nil, status.Error(codes.Unimplemented, "only LAYER_FORM_DIRS is supported")
	}
	if len(req.GetRequiredErofsDigests()) > 0 {
		return nil, status.Error(codes.InvalidArgument, "required_erofs_digests is only valid with LAYER_FORM_EROFS")
	}
	prepReq := &client.PrepareRequest{ImageRef: req.GetImageRef(), LeaseID: req.GetLeaseId()}
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
	if errors.Is(err, errLeaseConflict) {
		return nil, status.Errorf(codes.AlreadyExists, "%v", err)
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

// ReleaseImage releases a lease. It returns InvalidArgument when the image
// digest or form does not match the lease. The driver only holds leases in
// directory form, so a release in another form matches no lease it holds.
func (s *Server) ReleaseImage(ctx context.Context, req *imageprovider.ReleaseImageRequest) (*imageprovider.ReleaseImageResponse, error) {
	if req.GetLeaseId() == "" {
		return nil, status.Error(codes.InvalidArgument, "lease_id is required")
	}
	if !isDirsForm(req.GetForm()) {
		if s.driver.holdsLease(req.GetLeaseId()) {
			return nil, status.Errorf(codes.InvalidArgument, "lease %q is not in form %v", req.GetLeaseId(), req.GetForm())
		}
		return &imageprovider.ReleaseImageResponse{}, nil
	}
	err := s.driver.ReleaseImage(ctx, &client.ReleaseRequest{LeaseID: req.GetLeaseId(), ImageDigest: req.GetImageDigest()})
	if errors.Is(err, errLeaseMismatch) {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &imageprovider.ReleaseImageResponse{}, nil
}

// ReconcileImages replaces the driver's leases with the leases in use. It
// skips leases in a form other than directories, which the driver never
// takes.
func (s *Server) ReconcileImages(ctx context.Context, req *imageprovider.ReconcileImagesRequest) (*imageprovider.ReconcileImagesResponse, error) {
	active := make([]*client.ActiveLease, 0, len(req.GetActiveLeases()))
	for _, l := range req.GetActiveLeases() {
		if !isDirsForm(l.GetForm()) {
			slog.WarnContext(ctx, "Skipping a lease in a form the plugin does not serve",
				slog.String("provider", s.driver.Name()),
				slog.String("lease", l.GetLeaseId()),
				slog.String("form", l.GetForm().String()))
			continue
		}
		active = append(active, &client.ActiveLease{
			LeaseID:     l.GetLeaseId(),
			ImageRef:    l.GetImageRef(),
			ImageDigest: l.GetImageDigest(),
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
