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

// Package adapter provides the shared ImageAccelerator gRPC server, reference
// count and lease lifecycle manager, and containerd Snapshots.v1 engine used by
// provider-specific adapter binaries (such as cmd/accelerator-riptide and
// cmd/accelerator-soci).
package adapter

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

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"go.ate.dev/substrate/oci-image-accelerator/pkg/client"
	imageaccelv1 "go.ate.dev/substrate/oci-image-accelerator/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DefaultListenSocket is the standard Unix domain socket path where an
// ImageAccelerator adapter listens on the node.
const DefaultListenSocket = "/run/ate/image-accelerator.sock"

// Server implements substrate.imageaccel.v1.ImageAcceleratorServer by delegating
// to a provider Driver.
type Server struct {
	imageaccelv1.UnimplementedImageAcceleratorServer
	driver *Driver
}

// NewServer creates a new ImageAccelerator gRPC server backed by driver.
func NewServer(driver *Driver) *Server {
	return &Server{driver: driver}
}

// CanProvide checks whether the underlying snapshotter is reachable and able
// to serve the requested image.
func (s *Server) CanProvide(ctx context.Context, req *imageaccelv1.CanProvideRequest) (*imageaccelv1.CanProvideResponse, error) {
	if s.driver == nil {
		return &imageaccelv1.CanProvideResponse{CanProvide: false}, nil
	}
	ok, err := s.driver.CanProvide(ctx, &imageaccel.PrepareRequest{
		ImageRef: req.GetImageRef(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "CanProvide: %v", err)
	}
	return &imageaccelv1.CanProvideResponse{
		CanProvide:   ok,
		ProviderName: s.driver.Name(),
		Evicting:     s.driver.IsDraining(),
	}, nil
}

// PrepareImage prepares and mounts all layers of the image, increments the
// image's reference count, and returns the ordered read-only host LayerDirs.
func (s *Server) PrepareImage(ctx context.Context, req *imageaccelv1.PrepareImageRequest) (*imageaccelv1.PrepareImageResponse, error) {
	if s.driver == nil {
		return nil, status.Error(codes.Unavailable, "no provider driver configured")
	}
	prepReq := &imageaccel.PrepareRequest{
		ImageRef: req.GetImageRef(),
	}
	if a := req.GetAuth(); a != nil {
		prepReq.AuthConfig = &imageaccel.AuthConfig{
			Username:      a.GetUsername(),
			Password:      a.GetPassword(),
			Auth:          a.GetAuth(),
			ServerAddress: a.GetServerAddress(),
			IdentityToken: a.GetIdentityToken(),
			RegistryToken: a.GetRegistryToken(),
		}
	}

	res, err := s.driver.PrepareImage(ctx, prepReq)
	if err != nil {
		if errors.Is(err, imageaccel.ErrNotSupported) {
			return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	var cfgBytes []byte
	if res.Config != nil {
		raw, err := json.Marshal(res.Config)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "serializing image config: %v", err)
		}
		cfgBytes = raw
	} else if len(res.LayerDirs) > 0 {
		// Provide default container config if the driver did not populate one.
		raw, _ := json.Marshal(&v1.Config{})
		cfgBytes = raw
	}

	provider := s.driver.Name()
	if res.ProviderName != "" {
		provider = res.ProviderName
	}

	return &imageaccelv1.PrepareImageResponse{
		ImageDigest:  res.ImageDigest,
		ConfigJson:   cfgBytes,
		LayerDirs:    res.LayerDirs,
		ProviderName: provider,
	}, nil
}

// ReleaseImage decrements the reference count for the image's layers.
func (s *Server) ReleaseImage(ctx context.Context, req *imageaccelv1.ReleaseImageRequest) (*imageaccelv1.ReleaseImageResponse, error) {
	if s.driver == nil {
		return &imageaccelv1.ReleaseImageResponse{}, nil
	}
	if err := s.driver.ReleaseImage(ctx, &imageaccel.ReleaseRequest{
		ImageRef:  req.GetImageRef(),
		BypassTTL: req.GetBypassTtl(),
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &imageaccelv1.ReleaseImageResponse{}, nil
}

// ReconcileImages restores active reference counts from surviving actor
// bundles on atelet startup and sweeps orphaned views.
func (s *Server) ReconcileImages(ctx context.Context, req *imageaccelv1.ReconcileImagesRequest) (*imageaccelv1.ReconcileImagesResponse, error) {
	if s.driver == nil {
		return &imageaccelv1.ReconcileImagesResponse{}, nil
	}
	active := make([]*imageaccel.ActiveImageLayers, 0, len(req.GetActiveImages()))
	for _, img := range req.GetActiveImages() {
		if img == nil {
			continue
		}
		active = append(active, &imageaccel.ActiveImageLayers{
			ImageRef:    img.GetImageRef(),
			ImageDigest: img.GetImageDigest(),
			LayerDirs:   append([]string(nil), img.GetLayerDirs()...),
			RefCount:    int(img.GetRefCount()),
		})
	}
	if err := s.driver.ReconcileImages(ctx, active); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &imageaccelv1.ReconcileImagesResponse{}, nil
}

func removeStaleDrainMarker(path string) {
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("Failed to remove stale drain-complete marker", slog.String("path", path), slog.Any("err", err))
	}
}

func writeDrainMarker(path string) {
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		slog.Warn("Failed to create drain-complete marker directory", slog.String("path", path), slog.Any("err", err))
		return
	}
	if err := os.WriteFile(path, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o644); err != nil {
		slog.Warn("Failed to write drain-complete marker", slog.String("path", path), slog.Any("err", err))
		return
	}
	slog.Info("Drain-complete marker written", slog.String("path", path))
}

// ServeUnixSocket starts a gRPC server for driver on listenSocket and blocks
// until ctx is cancelled or the server returns an error. On ctx cancellation
// (SIGTERM), it enters lame-duck drain mode (returning CanProvide=false while
// keeping the gRPC server open for ReleaseImage) until active leases reach 0
// or driver.DrainTimeout() expires, then stops the gRPC server and writes the
// drain-complete marker file.
func ServeUnixSocket(ctx context.Context, listenSocket string, driver *Driver) error {
	sockPath := strings.TrimPrefix(listenSocket, "unix://")
	if err := os.MkdirAll(filepath.Dir(sockPath), 0o755); err != nil {
		return fmt.Errorf("creating socket parent directory %s: %w", sockPath, err)
	}
	var drainMarkerPath string
	if driver != nil {
		drainMarkerPath = driver.DrainCompleteFile()
	}
	removeStaleDrainMarker(drainMarkerPath)
	_ = os.Remove(sockPath)

	if driver != nil {
		if err := driver.WaitReady(ctx); err != nil {
			return fmt.Errorf("waiting for provider %s readiness: %w", driver.Name(), err)
		}
	}

	lis, err := net.Listen("unix", sockPath)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", sockPath, err)
	}
	if ul, ok := lis.(*net.UnixListener); ok {
		ul.SetUnlinkOnClose(false)
	}
	createdFi, _ := os.Stat(sockPath)
	defer func() {
		_ = lis.Close()
		if curFi, err := os.Stat(sockPath); err == nil && createdFi != nil && os.SameFile(createdFi, curFi) {
			_ = os.Remove(sockPath)
		}
		writeDrainMarker(drainMarkerPath)
	}()

	grpcServer := grpc.NewServer()
	imageaccelv1.RegisterImageAcceleratorServer(grpcServer, NewServer(driver))

	go func() {
		<-ctx.Done()
		slog.Info("Shutdown signal received; entering lame-duck drain mode", slog.String("socket", sockPath))
		if driver != nil {
			driver.Drain(context.Background())
		}
		slog.Info("Stopping ImageAccelerator gRPC server", slog.String("socket", sockPath))
		stopped := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-time.After(5 * time.Second):
			grpcServer.Stop()
		}
	}()

	var providerName, snapshotterSock string
	if driver != nil {
		providerName = driver.Name()
		snapshotterSock = driver.SocketPath()
	}
	slog.Info("ImageAccelerator adapter listening",
		slog.String("provider", providerName),
		slog.String("listenSocket", sockPath),
		slog.String("snapshotterSocket", snapshotterSock))

	return grpcServer.Serve(lis)
}

// ProbeUnixSocket dials the ImageAccelerator Unix domain socket at listenSocket
// and verifies that CanProvide returns true.
func ProbeUnixSocket(ctx context.Context, listenSocket string) error {
	sockPath := strings.TrimPrefix(listenSocket, "unix://")
	fi, err := os.Stat(sockPath)
	if err != nil {
		return fmt.Errorf("stat adapter socket %s: %w", sockPath, err)
	}
	if fi.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("adapter path %s is not a unix socket", sockPath)
	}
	probeCtx, cancel := context.WithTimeout(ctx, 2*DefaultProbeTimeout)
	defer cancel()

	client, err := imageaccel.NewGRPCClient(sockPath)
	if err != nil {
		return fmt.Errorf("dialing adapter socket %s: %w", sockPath, err)
	}
	defer func() { _ = client.Close() }()

	ok, err := client.CanProvide(probeCtx, &imageaccel.PrepareRequest{})
	if err != nil {
		return fmt.Errorf("probing adapter %s: %w", sockPath, err)
	}
	if !ok {
		return fmt.Errorf("adapter %s reported CanProvide=false (provider=%s)", sockPath, client.Name())
	}
	return nil
}
