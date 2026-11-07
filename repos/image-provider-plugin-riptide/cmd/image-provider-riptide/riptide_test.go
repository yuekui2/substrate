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

	"github.com/ai-on-gke/image-provider-plugin-riptide/internal/proto/riptidekeychain"
	"github.com/ai-on-gke/image-provider-plugin/client"
	"github.com/ai-on-gke/image-provider-plugin/plugin"
	snapshotsapi "github.com/containerd/containerd/api/services/snapshots/v1"
	"github.com/containerd/containerd/api/types"
	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	testImageRef    = "us-central1-docker.pkg.dev/proj/repo/agent:v1"
	testImageDigest = "sha256:9999999999999999999999999999999999999999999999999999999999999999"
	testCanonical   = "us-central1-docker.pkg.dev/proj/repo/agent@" + testImageDigest
	testRegistry    = "us-central1-docker.pkg.dev"
)

var testDiffIDs = []string{
	"sha256:1111111111111111111111111111111111111111111111111111111111111111",
	"sha256:2222222222222222222222222222222222222222222222222222222222222222",
}

// fakeGCFSSnapshots acts like containerd-gcfs-grpc: Prepare commits every
// layer remotely and returns AlreadyExists, and View binds a directory.
type fakeGCFSSnapshots struct {
	snapshotsapi.SnapshotsClient

	mountDirs map[string]string

	mu        sync.Mutex
	committed map[string]bool
}

func newFakeGCFSSnapshots(t *testing.T) *fakeGCFSSnapshots {
	t.Helper()
	f := &fakeGCFSSnapshots{
		mountDirs: make(map[string]string),
		committed: make(map[string]bool),
	}
	for _, c := range plugin.ComputeChainIDs(testDiffIDs) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("streamed"), 0o644); err != nil {
			t.Fatal(err)
		}
		f.mountDirs[c.ChainID] = dir
	}
	return f
}

func (f *fakeGCFSSnapshots) Prepare(_ context.Context, in *snapshotsapi.PrepareSnapshotRequest, _ ...grpc.CallOption) (*snapshotsapi.PrepareSnapshotResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.committed[in.GetLabels()["containerd.io/snapshot.ref"]] = true
	return nil, status.Error(codes.AlreadyExists, "already exists")
}

func (f *fakeGCFSSnapshots) View(_ context.Context, in *snapshotsapi.ViewSnapshotRequest, _ ...grpc.CallOption) (*snapshotsapi.ViewSnapshotResponse, error) {
	return &snapshotsapi.ViewSnapshotResponse{
		Mounts: []*types.Mount{{Type: "bind", Source: f.mountDirs[in.GetParent()]}},
	}, nil
}

func (f *fakeGCFSSnapshots) Remove(context.Context, *snapshotsapi.RemoveSnapshotRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (f *fakeGCFSSnapshots) Stat(_ context.Context, in *snapshotsapi.StatSnapshotRequest, _ ...grpc.CallOption) (*snapshotsapi.StatSnapshotResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.committed[in.GetKey()] {
		return &snapshotsapi.StatSnapshotResponse{}, nil
	}
	return nil, status.Error(codes.NotFound, "not found")
}

type fakeKeychain struct {
	riptidekeychain.KeychainClient

	mu      sync.Mutex
	updates []*riptidekeychain.UpdateCredsRequest
}

func (k *fakeKeychain) UpdateCreds(_ context.Context, in *riptidekeychain.UpdateCredsRequest, _ ...grpc.CallOption) (*riptidekeychain.UpdateCredsResponse, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.updates = append(k.updates, in)
	return &riptidekeychain.UpdateCredsResponse{}, nil
}

// staticKeychain resolves every registry to auth, or to anonymous if auth
// is nil.
type staticKeychain struct{ auth *authn.AuthConfig }

func (k staticKeychain) Resolve(authn.Resource) (authn.Authenticator, error) {
	if k.auth == nil {
		return authn.Anonymous, nil
	}
	return authn.FromConfig(*k.auth), nil
}

func TestPushCreds(t *testing.T) {
	credsA := &client.AuthConfig{Username: "oauth2accesstoken", Password: "token-a"}
	credsB := &client.AuthConfig{Username: "oauth2accesstoken", Password: "token-b"}
	authA := &riptidekeychain.AuthConfig{Username: "oauth2accesstoken", Password: "token-a", ServerAddress: testRegistry}
	authB := &riptidekeychain.AuthConfig{Username: "oauth2accesstoken", Password: "token-b", ServerAddress: testRegistry}

	for _, tc := range []struct {
		name     string
		keychain *authn.AuthConfig
		// requests are the credentials of successive PrepareImage calls
		// for the same image. All but the first are warm hits.
		requests []*client.AuthConfig
		want     []*riptidekeychain.AuthConfig
	}{
		{
			name:     "warm hit with the same credentials skips the push",
			requests: []*client.AuthConfig{credsA, credsA},
			want:     []*riptidekeychain.AuthConfig{authA},
		},
		{
			name:     "warm hit with new credentials pushes them",
			requests: []*client.AuthConfig{credsA, credsB},
			want:     []*riptidekeychain.AuthConfig{authA, authB},
		},
		{
			name:     "request credentials win over the keychain",
			keychain: &authn.AuthConfig{Username: "_json_key", Password: "key"},
			requests: []*client.AuthConfig{credsA},
			want:     []*riptidekeychain.AuthConfig{authA},
		},
		{
			name:     "keychain credentials without request credentials",
			keychain: &authn.AuthConfig{Username: "_json_key", Password: "key"},
			requests: []*client.AuthConfig{nil},
			want:     []*riptidekeychain.AuthConfig{{Username: "_json_key", Password: "key", ServerAddress: testRegistry}},
		},
		{
			name:     "anonymous pull pushes nothing",
			requests: []*client.AuthConfig{nil, {}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			kc := &fakeKeychain{}
			d, err := newRiptideDriver(config{
				KeychainClient: kc,
				DriverOpts: []plugin.Option{
					plugin.WithWorkDir(filepath.Join(t.TempDir(), "work")),
					plugin.WithSnapshotsClient(newFakeGCFSSnapshots(t)),
					plugin.WithKeychain(staticKeychain{tc.keychain}),
					plugin.WithImageResolver(func(context.Context, string, *client.AuthConfig) (string, *v1.Config, []string, []string, error) {
						return testImageDigest, &v1.Config{}, testDiffIDs, []string{"sha256:aa", "sha256:bb"}, nil
					}),
				},
			})
			if err != nil {
				t.Fatalf("newRiptideDriver: %v", err)
			}
			t.Cleanup(func() { _ = d.Close() })

			for i, auth := range tc.requests {
				res, err := d.PrepareImage(t.Context(), &client.PrepareRequest{ImageRef: testImageRef, AuthConfig: auth})
				if err != nil {
					t.Fatalf("PrepareImage %d: %v", i, err)
				}
				if len(res.LayerDirs) != len(testDiffIDs) {
					t.Fatalf("PrepareImage %d: got %d layer dirs, want %d", i, len(res.LayerDirs), len(testDiffIDs))
				}
			}

			kc.mu.Lock()
			defer kc.mu.Unlock()
			if len(kc.updates) != len(tc.want) {
				t.Fatalf("got %d UpdateCreds calls, want %d: %v", len(kc.updates), len(tc.want), kc.updates)
			}
			for i, u := range kc.updates {
				if u.GetImage() != testCanonical {
					t.Errorf("UpdateCreds %d image = %q, want %q", i, u.GetImage(), testCanonical)
				}
				if !proto.Equal(u.GetAuth(), tc.want[i]) {
					t.Errorf("UpdateCreds %d auth = %v, want %v", i, u.GetAuth(), tc.want[i])
				}
			}
		})
	}
}

func TestProbeKeychainSocket(t *testing.T) {
	dir, err := os.MkdirTemp("", "ipp-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sock := filepath.Join(dir, "keychain.sock")

	d, err := newRiptideDriver(config{
		KeychainSocket: sock,
		DriverOpts: []plugin.Option{
			plugin.WithWorkDir(filepath.Join(dir, "work")),
			plugin.WithSnapshotsClient(newFakeGCFSSnapshots(t)),
		},
	})
	if err != nil {
		t.Fatalf("newRiptideDriver: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	if err := d.probe(t.Context()); err == nil {
		t.Error("probe passed without a keychain socket")
	}

	if err := os.WriteFile(sock, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := d.probe(t.Context()); err == nil {
		t.Error("probe passed with a regular file at the keychain socket path")
	}
	if err := os.Remove(sock); err != nil {
		t.Fatal(err)
	}

	lis, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lis.Close() })
	if err := d.probe(t.Context()); err != nil {
		t.Errorf("probe with a keychain socket: %v", err)
	}
}
