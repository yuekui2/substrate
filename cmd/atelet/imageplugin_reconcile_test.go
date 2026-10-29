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

package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"go.ate.dev/substrate/api/ateapi/v1"
	"go.ate.dev/substrate/internal/imagecache"
	"go.ate.dev/substrate/internal/imagepluginclient"
	"go.ate.dev/substrate/internal/imagepluginclient/mock"
	"go.ate.dev/substrate/internal/proto/ateworkerpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestReconcileStreamingLeases(t *testing.T) {
	ctx := context.Background()
	actorsDir := t.TempDir()

	writeBundle := func(actorUID, container string, spec *imagecache.OverlaySpec) {
		t.Helper()
		dir := filepath.Join(actorsDir, actorUID, "bundles", container)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := imagecache.WriteSpec(dir, spec); err != nil {
			t.Fatalf("WriteSpec: %v", err)
		}
	}

	// Actor 1 and Actor 2 share a streamed image; Actor 3 uses a local imagecache image.
	streamRef := "us-central1-docker.pkg.dev/proj/repo/agent:v1"
	volStreamRef := "us-central1-docker.pkg.dev/proj/repo/tools:v1"
	localRef := "us-central1-docker.pkg.dev/proj/repo/local:v1"

	writeBundle("actor-1", "main", &imagecache.OverlaySpec{
		ImageRef:    streamRef,
		ImageDigest: "sha256:aaa",
		Streamed:    true,
		Layers:      []string{"/var/lib/ate/image-plugin-cache/img1/layer-0"},
		ImageVolumes: []imagecache.ImageVolumeOverlay{{
			Name:        "tools",
			ImageRef:    volStreamRef,
			ImageDigest: "sha256:bbb",
			Streamed:    true,
			Layers:      []string{"/var/lib/ate/image-plugin-cache/vol1/layer-0"},
		}},
	})
	writeBundle("actor-2", "main", &imagecache.OverlaySpec{
		ImageRef:    streamRef,
		ImageDigest: "sha256:aaa",
		Streamed:    true,
		Layers:      []string{"/var/lib/ate/image-plugin-cache/img1/layer-0"},
	})
	writeBundle("actor-3", "main", &imagecache.OverlaySpec{
		ImageRef:    localRef,
		ImageDigest: "sha256:ccc",
		Streamed:    false,
		Layers:      []string{"/var/lib/ate/image-cache/layers/sha256/ccc"},
	})

	client := mock.New("riptide")
	if err := reconcileStreamingLeases(ctx, client, actorsDir); err != nil {
		t.Fatalf("reconcileStreamingLeases failed: %v", err)
	}

	if len(client.ReconcileCalls) != 1 {
		t.Fatalf("expected 1 ReconcileImages call, got %d", len(client.ReconcileCalls))
	}
	reconciled := make(map[string]*imagepluginclient.ActiveImageLayers)
	for _, a := range client.ReconcileCalls[0] {
		reconciled[a.ImageRef] = a
	}
	if a := reconciled[streamRef]; a == nil || a.RefCount != 2 {
		t.Errorf("reconciled[%q] = %+v, want RefCount=2", streamRef, a)
	}
	if a := reconciled[volStreamRef]; a == nil || a.RefCount != 1 {
		t.Errorf("reconciled[%q] = %+v, want RefCount=1", volStreamRef, a)
	}
	if a := reconciled[localRef]; a != nil {
		t.Errorf("expected localRef %q not to be reconciled, got %+v", localRef, a)
	}
}

type fakeEvictWorkerServer struct {
	ateworkerpb.UnimplementedWorkerServer
	mu      sync.Mutex
	samples []*ateworkerpb.WorkloadStatsSample
}

func (f *fakeEvictWorkerServer) GetActiveWorkloadStats(_ context.Context, _ *ateworkerpb.GetActiveWorkloadStatsRequest) (*ateworkerpb.GetActiveWorkloadStatsResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return &ateworkerpb.GetActiveWorkloadStatsResponse{Samples: f.samples}, nil
}

// fakeSuspendWorkers records RequestActorSuspend calls and fails the first
// failFirst of them.
type fakeSuspendWorkers struct {
	ateapi.WorkerServiceClient
	mu        sync.Mutex
	failFirst int
	requests  []*ateapi.RequestActorSuspendRequest
}

func (f *fakeSuspendWorkers) RequestActorSuspend(_ context.Context, req *ateapi.RequestActorSuspendRequest, _ ...grpc.CallOption) (*ateapi.RequestActorSuspendResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, req)
	if len(f.requests) <= f.failFirst {
		return nil, status.Error(codes.Unavailable, "control plane unavailable")
	}
	return &ateapi.RequestActorSuspendResponse{}, nil
}

func TestEvictActiveStreamedActors(t *testing.T) {
	ctx := context.Background()
	actorsDir := t.TempDir()
	workersDir := t.TempDir()

	writeBundle := func(actorUID, container string, spec *imagecache.OverlaySpec) {
		t.Helper()
		dir := filepath.Join(actorsDir, actorUID, "bundles", container)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := imagecache.WriteSpec(dir, spec); err != nil {
			t.Fatalf("WriteSpec: %v", err)
		}
	}

	writeBundle("actor-streamed", "main", &imagecache.OverlaySpec{
		ImageRef:    "us-central1-docker.pkg.dev/proj/repo/agent:v1",
		ImageDigest: "sha256:aaa",
		Streamed:    true,
		Layers:      []string{"/var/lib/ate/image-plugin-cache/img1/layer-0"},
	})
	writeBundle("actor-local", "main", &imagecache.OverlaySpec{
		ImageRef:    "us-central1-docker.pkg.dev/proj/repo/local:v1",
		ImageDigest: "sha256:ccc",
		Streamed:    false,
		Layers:      []string{"/var/lib/ate/image-cache/layers/sha256/ccc"},
	})

	workerPodUID := "worker-pod-1"
	if err := os.MkdirAll(filepath.Join(workersDir, workerPodUID), 0o755); err != nil {
		t.Fatalf("MkdirAll(worker): %v", err)
	}

	sockDir, err := os.MkdirTemp("", "evict-worker-")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(sockDir) })
	sock := filepath.Join(sockDir, "worker.sock")
	lis, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	fakeWorker := &fakeEvictWorkerServer{
		samples: []*ateworkerpb.WorkloadStatsSample{
			{Atespace: "ns", ActorName: "streamed", ActorUid: "actor-streamed"},
			{Atespace: "ns", ActorName: "local", ActorUid: "actor-local"},
		},
	}
	srv := grpc.NewServer()
	ateworkerpb.RegisterWorkerServer(srv, fakeWorker)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	origSocketPath := workerSocketPath
	workerSocketPath = func(string) string { return sock }
	t.Cleanup(func() { workerSocketPath = origSocketPath })

	workers := &fakeSuspendWorkers{failFirst: 1}
	atelet := &Atelet{
		workerDialer: newWorkerDialer(1),
		workers:      workers,
	}

	// A failed suspend is retried on the next sweep.
	if evicted := atelet.evictActiveStreamedActors(ctx, actorsDir, workersDir); evicted != 0 {
		t.Fatalf("first evictActiveStreamedActors() = %d, want 0", evicted)
	}
	// The retry suspends only actor-streamed and leaves actor-local alone.
	if evicted := atelet.evictActiveStreamedActors(ctx, actorsDir, workersDir); evicted != 1 {
		t.Fatalf("second evictActiveStreamedActors() = %d, want 1", evicted)
	}
	want := &ateapi.RequestActorSuspendRequest{
		Worker:   &ateapi.ObjectRef{Name: workerPodUID},
		Actor:    &ateapi.ObjectRef{Atespace: "ns", Name: "streamed"},
		ActorUid: "actor-streamed",
	}
	workers.mu.Lock()
	got := append([]*ateapi.RequestActorSuspendRequest(nil), workers.requests...)
	workers.mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("RequestActorSuspend calls = %d, want 2", len(got))
	}
	for i, req := range got {
		if !proto.Equal(req, want) {
			t.Errorf("RequestActorSuspend call %d = %v, want %v", i, req, want)
		}
	}

	// A suspended actor is not asked for again.
	if again := atelet.evictActiveStreamedActors(ctx, actorsDir, workersDir); again != 0 {
		t.Fatalf("third evictActiveStreamedActors() = %d, want 0", again)
	}
}
