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

package adapter

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"go.ate.dev/substrate/oci-image-accelerator/internal/proto/snapshots"
	"go.ate.dev/substrate/oci-image-accelerator/pkg/client"
	imageaccelv1 "go.ate.dev/substrate/oci-image-accelerator/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type fakeSnapshotsClient struct {
	mu          sync.Mutex
	mountDirs   map[string]string
	committed   map[string]bool
	declinePrep bool
	prepareReqs []*snapshots.PrepareSnapshotRequest
	viewReqs    []*snapshots.ViewSnapshotRequest
	removeKeys  []string
}

func newFakeSnapshotsClient(t *testing.T, diffIDs []string) *fakeSnapshotsClient {
	t.Helper()
	f := &fakeSnapshotsClient{
		mountDirs: make(map[string]string),
		committed: make(map[string]bool),
	}
	for _, c := range ComputeChainIDs(diffIDs) {
		dir := filepath.Join(t.TempDir(), SanitizePathKey(c.ChainID))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("streamed"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		f.mountDirs[c.ChainID] = dir
	}
	return f
}

func (f *fakeSnapshotsClient) Prepare(_ context.Context, in *snapshots.PrepareSnapshotRequest, _ ...grpc.CallOption) (*snapshots.PrepareSnapshotResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prepareReqs = append(f.prepareReqs, in)
	if f.declinePrep {
		return &snapshots.PrepareSnapshotResponse{}, nil
	}
	chainID := in.GetLabels()["containerd.io/snapshot.ref"]
	if chainID != "" {
		f.committed[chainID] = true
	}
	return nil, status.Error(codes.AlreadyExists, "already exists")
}

func (f *fakeSnapshotsClient) View(_ context.Context, in *snapshots.ViewSnapshotRequest, _ ...grpc.CallOption) (*snapshots.ViewSnapshotResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.viewReqs = append(f.viewReqs, in)
	dir := f.mountDirs[in.GetParent()]
	return &snapshots.ViewSnapshotResponse{
		Mounts: []*snapshots.Mount{{
			Type:   "bind",
			Source: dir,
		}},
	}, nil
}

func (f *fakeSnapshotsClient) Mounts(_ context.Context, _ *snapshots.MountsRequest, _ ...grpc.CallOption) (*snapshots.MountsResponse, error) {
	return &snapshots.MountsResponse{}, nil
}

func (f *fakeSnapshotsClient) Commit(_ context.Context, _ *snapshots.CommitSnapshotRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (f *fakeSnapshotsClient) Remove(_ context.Context, in *snapshots.RemoveSnapshotRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removeKeys = append(f.removeKeys, in.GetKey())
	return &emptypb.Empty{}, nil
}

func (f *fakeSnapshotsClient) Stat(_ context.Context, in *snapshots.StatSnapshotRequest, _ ...grpc.CallOption) (*snapshots.StatSnapshotResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.committed[in.GetKey()] {
		return &snapshots.StatSnapshotResponse{}, nil
	}
	return nil, status.Error(codes.NotFound, "not found")
}

func startTestAcceleratorServer(t *testing.T, driver *Driver) *imageaccel.GRPCClient {
	t.Helper()
	sockPath := filepath.Join(t.TempDir(), "image-accelerator.sock")
	lis, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	srv := grpc.NewServer()
	imageaccelv1.RegisterImageAcceleratorServer(srv, NewServer(driver))
	go func() {
		_ = srv.Serve(lis)
	}()
	t.Cleanup(func() {
		srv.Stop()
		_ = lis.Close()
		_ = driver.Close()
	})
	client, err := imageaccel.NewGRPCClient(sockPath)
	if err != nil {
		t.Fatalf("NewGRPCClient: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestServerAndDriverLifecycle(t *testing.T) {
	ctx := context.Background()
	diffIDs := []string{
		"sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"sha256:2222222222222222222222222222222222222222222222222222222222222222",
	}
	layerDigests := []string{
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	imgDigest := "sha256:9999999999999999999999999999999999999999999999999999999999999999"

	snapClient := newFakeSnapshotsClient(t, diffIDs)
	workDir := filepath.Join(t.TempDir(), "work")

	driver, err := NewDriver("test-provider", "/run/test.sock", "test-snap",
		WithWorkDir(workDir),
		WithSnapshotsClient(snapClient),
		WithLeaseIdleTTL(10*time.Minute),
		WithImageResolver(func(_ context.Context, _ string, _ *imageaccel.AuthConfig) (string, *v1.Config, []string, []string, error) {
			return imgDigest, &v1.Config{Entrypoint: []string{"/bin/agent"}}, diffIDs, layerDigests, nil
		}),
	)
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}

	client := startTestAcceleratorServer(t, driver)
	imageRef := "us-central1-docker.pkg.dev/my-proj/repo/agent:v1"
	req := &imageaccel.PrepareRequest{ImageRef: imageRef}

	ok, err := client.CanProvide(ctx, req)
	if err != nil || !ok {
		t.Fatalf("CanProvide = (%v, %v), want (true, nil)", ok, err)
	}
	if client.Name() != "test-provider" {
		t.Errorf("client.Name() = %q, want %q", client.Name(), "test-provider")
	}

	// 1. First PrepareImage mounts the 2 layers.
	res1, err := client.PrepareImage(ctx, req)
	if err != nil {
		t.Fatalf("PrepareImage(1) failed: %v", err)
	}
	if len(res1.LayerDirs) != 2 {
		t.Fatalf("len(LayerDirs) = %d, want 2", len(res1.LayerDirs))
	}
	for i, dir := range res1.LayerDirs {
		if _, err := os.Stat(filepath.Join(dir, "fs", "hello.txt")); err != nil {
			t.Errorf("layer %d missing streamed file: %v", i, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "finalized")); err != nil {
			t.Errorf("layer %d missing finalized marker: %v", i, err)
		}
	}

	// 2. Second PrepareImage reuses warm lease and increments refCount to 2 without new View RPCs.
	res2, err := client.PrepareImage(ctx, req)
	if err != nil {
		t.Fatalf("PrepareImage(2) failed: %v", err)
	}
	snapClient.mu.Lock()
	if len(snapClient.viewReqs) != 2 {
		t.Errorf("expected 2 View RPCs total after warm hit, got %d", len(snapClient.viewReqs))
	}
	snapClient.mu.Unlock()
	if len(res2.LayerDirs) != 2 {
		t.Fatalf("len(res2.LayerDirs) = %d, want 2", len(res2.LayerDirs))
	}

	// 3. First ReleaseImage decrements refCount from 2 -> 1 (views stay mounted).
	if err := client.ReleaseImage(ctx, &imageaccel.ReleaseRequest{ImageRef: imageRef, BypassTTL: true}); err != nil {
		t.Fatalf("ReleaseImage(1) failed: %v", err)
	}
	snapClient.mu.Lock()
	if len(snapClient.removeKeys) != 0 {
		t.Errorf("expected 0 Remove calls while refCount=1, got %d", len(snapClient.removeKeys))
	}
	snapClient.mu.Unlock()

	// 4. Second ReleaseImage with BypassTTL=true decrements refCount 1 -> 0 and immediately removes views.
	if err := client.ReleaseImage(ctx, &imageaccel.ReleaseRequest{ImageRef: imageRef, BypassTTL: true}); err != nil {
		t.Fatalf("ReleaseImage(2) failed: %v", err)
	}
	snapClient.mu.Lock()
	if len(snapClient.removeKeys) != 2 {
		t.Errorf("expected 2 Remove calls after refCount reached 0 with BypassTTL, got %d", len(snapClient.removeKeys))
	}
	snapClient.mu.Unlock()
}

func TestServerDeclineReturnsErrNotSupported(t *testing.T) {
	ctx := context.Background()
	diffIDs := []string{"sha256:1111111111111111111111111111111111111111111111111111111111111111"}
	snapClient := newFakeSnapshotsClient(t, diffIDs)
	snapClient.declinePrep = true

	driver, err := NewDriver("test-provider", "/run/test.sock", "test-snap",
		WithWorkDir(filepath.Join(t.TempDir(), "work")),
		WithSnapshotsClient(snapClient),
		WithImageResolver(func(_ context.Context, _ string, _ *imageaccel.AuthConfig) (string, *v1.Config, []string, []string, error) {
			return "sha256:1234", &v1.Config{}, diffIDs, diffIDs, nil
		}),
	)
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}

	client := startTestAcceleratorServer(t, driver)
	_, err = client.PrepareImage(ctx, &imageaccel.PrepareRequest{ImageRef: "docker.io/library/alpine:latest"})
	if !errors.Is(err, imageaccel.ErrNotSupported) {
		t.Fatalf("PrepareImage error = %v, want ErrNotSupported", err)
	}
}

func TestServerReconcileImagesRestoresAndSweepsOrphans(t *testing.T) {
	ctx := context.Background()
	diffIDs := []string{"sha256:1111111111111111111111111111111111111111111111111111111111111111"}
	snapClient := newFakeSnapshotsClient(t, diffIDs)
	workDir := filepath.Join(t.TempDir(), "work")

	driver, err := NewDriver("test-provider", "/run/test.sock", "test-snap",
		WithWorkDir(workDir),
		WithSnapshotsClient(snapClient),
		WithLeaseIdleTTL(0),
		WithImageResolver(func(_ context.Context, _ string, _ *imageaccel.AuthConfig) (string, *v1.Config, []string, []string, error) {
			return "sha256:1234", &v1.Config{}, diffIDs, diffIDs, nil
		}),
	)
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}

	client := startTestAcceleratorServer(t, driver)
	activeImg := "example.com/active:v1"
	orphanImg := "example.com/orphan:v1"

	activeRes, err := client.PrepareImage(ctx, &imageaccel.PrepareRequest{ImageRef: activeImg})
	if err != nil {
		t.Fatalf("PrepareImage(active): %v", err)
	}
	orphanRes, err := client.PrepareImage(ctx, &imageaccel.PrepareRequest{ImageRef: orphanImg})
	if err != nil {
		t.Fatalf("PrepareImage(orphan): %v", err)
	}

	// Reconcile only activeImg with RefCount=3; orphanImg should be swept immediately.
	if err := client.ReconcileImages(ctx, []*imageaccel.ActiveImageLayers{
		{
			ImageRef:    activeImg,
			ImageDigest: activeRes.ImageDigest,
			LayerDirs:   activeRes.LayerDirs,
			RefCount:    3,
		},
	}); err != nil {
		t.Fatalf("ReconcileImages: %v", err)
	}

	if _, err := os.Stat(orphanRes.LayerDirs[0]); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("orphan layer dir still exists after ReconcileImages: err=%v", err)
	}
	if _, err := os.Stat(activeRes.LayerDirs[0]); err != nil {
		t.Errorf("active layer dir missing after ReconcileImages: %v", err)
	}
}

func TestLameDuckDrainAndDrainCompleteMarker(t *testing.T) {
	ctx := context.Background()
	diffIDs := []string{"sha256:1111111111111111111111111111111111111111111111111111111111111111"}
	snapClient := newFakeSnapshotsClient(t, diffIDs)
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "work")
	sockPath := filepath.Join(tmpDir, "image-accelerator.sock")
	drainMarker := filepath.Join(tmpDir, "drain-complete")

	// Pre-create a stale drain-complete marker to verify ServeUnixSocket removes it on startup.
	if err := os.WriteFile(drainMarker, []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("WriteFile stale marker: %v", err)
	}

	driver, err := NewDriver("test-provider", "/run/test.sock", "test-snap",
		WithWorkDir(workDir),
		WithSnapshotsClient(snapClient),
		WithLeaseIdleTTL(10*time.Minute),
		WithDrainTimeout(5*time.Second),
		WithDrainCompleteFile(drainMarker),
		WithImageResolver(func(_ context.Context, _ string, _ *imageaccel.AuthConfig) (string, *v1.Config, []string, []string, error) {
			return "sha256:1234", &v1.Config{}, diffIDs, diffIDs, nil
		}),
	)
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}
	t.Cleanup(func() { _ = driver.Close() })

	srvCtx, cancelSrv := context.WithCancel(ctx)
	defer cancelSrv()
	srvDone := make(chan error, 1)
	go func() {
		srvDone <- ServeUnixSocket(srvCtx, sockPath, driver)
	}()

	// Wait for the unix socket to appear and verify stale drain marker was removed.
	for _, err := os.Stat(sockPath); err != nil; _, err = os.Stat(sockPath) {
		time.Sleep(5 * time.Millisecond)
	}
	if _, err := os.Stat(drainMarker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale drain-complete marker was not removed on startup: err=%v", err)
	}

	client, err := imageaccel.NewGRPCClient(sockPath)
	if err != nil {
		t.Fatalf("NewGRPCClient: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	activeImg := "example.com/active:v1"
	idleImg := "example.com/idle:v1"

	// 1. Prepare activeImg (refCount=1) and idleImg (released so refCount=0, sitting in 10m idleTTL).
	activeRes, err := client.PrepareImage(ctx, &imageaccel.PrepareRequest{ImageRef: activeImg})
	if err != nil {
		t.Fatalf("PrepareImage(activeImg): %v", err)
	}
	idleRes, err := client.PrepareImage(ctx, &imageaccel.PrepareRequest{ImageRef: idleImg})
	if err != nil {
		t.Fatalf("PrepareImage(idleImg): %v", err)
	}
	if err := client.ReleaseImage(ctx, &imageaccel.ReleaseRequest{ImageRef: idleImg, BypassTTL: false}); err != nil {
		t.Fatalf("ReleaseImage(idleImg): %v", err)
	}
	if _, err := os.Stat(idleRes.LayerDirs[0]); err != nil {
		t.Fatalf("expected idleImg to remain cached before drain: %v", err)
	}

	if evicting, err := client.IsEvicting(ctx); err != nil || evicting {
		t.Fatalf("IsEvicting before drain = (%v, %v), want (false, nil)", evicting, err)
	}

	// 2. Cancel srvCtx (simulating SIGTERM). Adapter enters lame-duck mode:
	//    - idleImg (refCount=0) is evicted immediately
	//    - activeImg (refCount=1) keeps the server alive
	//    - CanProvide returns false, IsEvicting returns true, and PrepareImage returns ErrNotSupported
	cancelSrv()
	for !driver.IsDraining() {
		time.Sleep(2 * time.Millisecond)
	}

	canProvide, err := client.CanProvide(ctx, &imageaccel.PrepareRequest{ImageRef: activeImg})
	if err != nil || canProvide {
		t.Errorf("CanProvide during lame-duck drain = (%v, %v), want (false, nil)", canProvide, err)
	}
	if evicting, err := client.IsEvicting(ctx); err != nil || !evicting {
		t.Errorf("IsEvicting during lame-duck drain = (%v, %v), want (true, nil)", evicting, err)
	}
	if _, err := client.PrepareImage(ctx, &imageaccel.PrepareRequest{ImageRef: "example.com/new:v1"}); !errors.Is(err, imageaccel.ErrNotSupported) {
		t.Errorf("PrepareImage during lame-duck drain = %v, want ErrNotSupported", err)
	}

	// Idle lease should be evicted while active lease remains mounted.
	for _, err := os.Stat(idleRes.LayerDirs[0]); !errors.Is(err, os.ErrNotExist); _, err = os.Stat(idleRes.LayerDirs[0]) {
		time.Sleep(2 * time.Millisecond)
	}
	if _, err := os.Stat(activeRes.LayerDirs[0]); err != nil {
		t.Fatalf("activeImg layer dir was removed while refCount=1: %v", err)
	}
	if _, err := os.Stat(drainMarker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("drain-complete marker written before active lease released")
	}

	// 3. Release activeImg (even with BypassTTL=false, lame-duck mode auto-bypasses idleTTL).
	if err := client.ReleaseImage(ctx, &imageaccel.ReleaseRequest{ImageRef: activeImg, BypassTTL: false}); err != nil {
		t.Fatalf("ReleaseImage(activeImg): %v", err)
	}

	// ServeUnixSocket should now exit cleanly and write the drain-complete marker.
	if err := <-srvDone; err != nil {
		t.Fatalf("ServeUnixSocket returned error: %v", err)
	}
	if _, err := os.Stat(activeRes.LayerDirs[0]); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("activeImg layer dir still exists after release: err=%v", err)
	}
	if _, err := os.Stat(drainMarker); err != nil {
		t.Errorf("expected drain-complete marker to be written on shutdown: %v", err)
	}
}

func TestDrainTimeoutFallback(t *testing.T) {
	ctx := context.Background()
	diffIDs := []string{"sha256:1111111111111111111111111111111111111111111111111111111111111111"}
	snapClient := newFakeSnapshotsClient(t, diffIDs)

	driver, err := NewDriver("test-provider", "/run/test.sock", "test-snap",
		WithWorkDir(filepath.Join(t.TempDir(), "work")),
		WithSnapshotsClient(snapClient),
		WithDrainTimeout(20*time.Millisecond),
		WithImageResolver(func(_ context.Context, _ string, _ *imageaccel.AuthConfig) (string, *v1.Config, []string, []string, error) {
			return "sha256:1234", &v1.Config{}, diffIDs, diffIDs, nil
		}),
	)
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}
	t.Cleanup(func() { _ = driver.Close() })

	if _, err := driver.PrepareImage(ctx, &imageaccel.PrepareRequest{ImageRef: "example.com/stuck:v1"}); err != nil {
		t.Fatalf("PrepareImage: %v", err)
	}
	if driver.ActiveLeaseCount() != 1 {
		t.Fatalf("ActiveLeaseCount() = %d, want 1", driver.ActiveLeaseCount())
	}

	// Drain should time out after 20ms and return false because refCount never reaches 0.
	if clean := driver.Drain(ctx); clean {
		t.Errorf("Drain() = true, want false when active lease exceeds drainTimeout")
	}
}

func TestServeUnixSocketWaitReadyAndProbe(t *testing.T) {
	ctx := context.Background()
	diffIDs := []string{"sha256:1111111111111111111111111111111111111111111111111111111111111111"}
	snapClient := newFakeSnapshotsClient(t, diffIDs)
	tmpDir := t.TempDir()

	var (
		mu             sync.Mutex
		providerReady  bool
		firstProbeOnce sync.Once
		firstProbedCh  = make(chan struct{})
	)

	driver, err := NewDriver("test-provider", "/run/test.sock", "test-snap",
		WithWorkDir(filepath.Join(tmpDir, "work")),
		WithSnapshotsClient(snapClient),
		WithStartupTimeout(2*time.Second),
		WithStartupPollInterval(10*time.Millisecond),
		WithDrainTimeout(5*time.Second),
		WithDrainCompleteFile(filepath.Join(tmpDir, "drain-complete")),
		WithProbeHook(func(_ context.Context) error {
			firstProbeOnce.Do(func() { close(firstProbedCh) })
			mu.Lock()
			ready := providerReady
			mu.Unlock()
			if !ready {
				return errors.New("underlying provider not ready yet")
			}
			return nil
		}),
		WithImageResolver(func(_ context.Context, _ string, _ *imageaccel.AuthConfig) (string, *v1.Config, []string, []string, error) {
			return "sha256:1234", &v1.Config{}, diffIDs, diffIDs, nil
		}),
	)
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}
	t.Cleanup(func() { _ = driver.Close() })

	sockPath := filepath.Join(tmpDir, "image-accelerator.sock")
	srvCtx, cancelSrv := context.WithCancel(ctx)
	defer cancelSrv()

	srvDone := make(chan error, 1)
	go func() {
		srvDone <- ServeUnixSocket(srvCtx, sockPath, driver)
	}()

	// Wait for WaitReady to execute its first probe while providerReady is false.
	<-firstProbedCh
	if _, err := os.Stat(sockPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %s not to exist before provider is ready, got err=%v", sockPath, err)
	}
	if err := ProbeUnixSocket(ctx, sockPath); err == nil {
		t.Fatalf("expected ProbeUnixSocket to fail before provider is ready")
	}

	// Mark the underlying provider ready; ServeUnixSocket should bind sockPath and pass ProbeUnixSocket.
	mu.Lock()
	providerReady = true
	mu.Unlock()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if err := ProbeUnixSocket(ctx, sockPath); err == nil {
			break
		} else if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for ProbeUnixSocket to succeed: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Hold an active lease and trigger SIGTERM (cancelSrv): ProbeUnixSocket must immediately fail during lame-duck drain.
	activeImg := "example.com/active:v1"
	if _, err := driver.PrepareImage(ctx, &imageaccel.PrepareRequest{ImageRef: activeImg}); err != nil {
		t.Fatalf("PrepareImage: %v", err)
	}
	cancelSrv()

	drainDeadline := time.Now().Add(2 * time.Second)
	for !driver.IsDraining() {
		if time.Now().After(drainDeadline) {
			t.Fatalf("timed out waiting for driver to enter draining state")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := ProbeUnixSocket(ctx, sockPath); err == nil {
		t.Errorf("expected ProbeUnixSocket to fail while adapter is in lame-duck drain mode")
	}

	if err := driver.ReleaseImage(ctx, &imageaccel.ReleaseRequest{ImageRef: activeImg}); err != nil {
		t.Fatalf("ReleaseImage: %v", err)
	}
	if err := <-srvDone; err != nil {
		t.Fatalf("ServeUnixSocket returned error: %v", err)
	}
}

func TestWaitReadyTimeout(t *testing.T) {
	ctx := context.Background()
	diffIDs := []string{"sha256:1111111111111111111111111111111111111111111111111111111111111111"}
	snapClient := newFakeSnapshotsClient(t, diffIDs)
	tmpDir := t.TempDir()

	driver, err := NewDriver("test-provider", "/run/test.sock", "test-snap",
		WithWorkDir(filepath.Join(tmpDir, "work")),
		WithSnapshotsClient(snapClient),
		WithStartupTimeout(40*time.Millisecond),
		WithStartupPollInterval(10*time.Millisecond),
		WithProbeHook(func(_ context.Context) error {
			return errors.New("provider daemon offline")
		}),
	)
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}
	t.Cleanup(func() { _ = driver.Close() })

	sockPath := filepath.Join(tmpDir, "image-accelerator.sock")
	if err := ServeUnixSocket(ctx, sockPath, driver); err == nil {
		t.Fatalf("expected ServeUnixSocket to fail when provider never passes readiness probe")
	}
	if _, err := os.Stat(sockPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected %s not to be created when WaitReady fails, got err=%v", sockPath, err)
	}
}
