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
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	snapshotsapi "github.com/containerd/containerd/api/services/snapshots/v1"
	"github.com/containerd/containerd/api/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// fakeSnapshotsClient acts like a remote snapshotter that streams every
// layer: Prepare commits the layer and returns AlreadyExists, and View
// returns a bind mount of a directory holding hello.txt.
type fakeSnapshotsClient struct {
	snapshotsapi.SnapshotsClient

	mu          sync.Mutex
	mountDirs   map[string]string
	committed   map[string]bool
	declinePrep bool
	prepareReqs []*snapshotsapi.PrepareSnapshotRequest
	viewReqs    []*snapshotsapi.ViewSnapshotRequest
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
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("streamed"), 0o644); err != nil {
			t.Fatal(err)
		}
		f.mountDirs[c.ChainID] = dir
	}
	return f
}

func (f *fakeSnapshotsClient) Prepare(_ context.Context, in *snapshotsapi.PrepareSnapshotRequest, _ ...grpc.CallOption) (*snapshotsapi.PrepareSnapshotResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prepareReqs = append(f.prepareReqs, in)
	if f.declinePrep {
		return &snapshotsapi.PrepareSnapshotResponse{}, nil
	}
	if chainID := in.GetLabels()["containerd.io/snapshot.ref"]; chainID != "" {
		f.committed[chainID] = true
	}
	return nil, status.Error(codes.AlreadyExists, "already exists")
}

func (f *fakeSnapshotsClient) View(_ context.Context, in *snapshotsapi.ViewSnapshotRequest, _ ...grpc.CallOption) (*snapshotsapi.ViewSnapshotResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.viewReqs = append(f.viewReqs, in)
	return &snapshotsapi.ViewSnapshotResponse{
		Mounts: []*types.Mount{{Type: "bind", Source: f.mountDirs[in.GetParent()]}},
	}, nil
}

func (f *fakeSnapshotsClient) Remove(_ context.Context, in *snapshotsapi.RemoveSnapshotRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removeKeys = append(f.removeKeys, in.GetKey())
	return &emptypb.Empty{}, nil
}

func (f *fakeSnapshotsClient) Stat(_ context.Context, in *snapshotsapi.StatSnapshotRequest, _ ...grpc.CallOption) (*snapshotsapi.StatSnapshotResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.committed[in.GetKey()] {
		return &snapshotsapi.StatSnapshotResponse{}, nil
	}
	return nil, status.Error(codes.NotFound, "not found")
}

func (f *fakeSnapshotsClient) removed() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.removeKeys)
}
