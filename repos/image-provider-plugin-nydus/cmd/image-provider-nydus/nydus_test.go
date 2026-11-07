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
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	imageprovider "github.com/ai-on-gke/image-provider-plugin/api/imageprovider/v1alpha1"
	"github.com/ai-on-gke/image-provider-plugin/client"
	"github.com/ai-on-gke/image-provider-plugin/plugin"
	snapshotsapi "github.com/containerd/containerd/api/services/snapshots/v1"
	"github.com/containerd/containerd/api/types"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const testImageRef = "registry.example.com/sandbox:nydus"

var (
	testDiffIDs = []string{
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	testLayerDigests = []string{
		"sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"sha256:2222222222222222222222222222222222222222222222222222222222222222",
	}
)

// fakeNydusSnapshots acts like containerd-nydus-grpc: Prepare registers a
// data blob layer itself and returns AlreadyExists, and leaves any other
// layer for the plugin to fill in activeDir.
type fakeNydusSnapshots struct {
	snapshotsapi.SnapshotsClient

	activeDir  string
	viewMounts []*types.Mount

	mu         sync.Mutex
	committed  map[string]bool
	commitReqs []*snapshotsapi.CommitSnapshotRequest
}

func (f *fakeNydusSnapshots) Prepare(_ context.Context, in *snapshotsapi.PrepareSnapshotRequest, _ ...grpc.CallOption) (*snapshotsapi.PrepareSnapshotResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if in.GetLabels()[labelNydusBlob] == "true" {
		f.committed[in.GetLabels()["containerd.io/snapshot.ref"]] = true
		return nil, status.Error(codes.AlreadyExists, "already exists")
	}
	return &snapshotsapi.PrepareSnapshotResponse{
		Mounts: []*types.Mount{{Type: "bind", Source: f.activeDir, Options: []string{"rw", "rbind"}}},
	}, nil
}

func (f *fakeNydusSnapshots) Commit(_ context.Context, in *snapshotsapi.CommitSnapshotRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commitReqs = append(f.commitReqs, in)
	f.committed[in.GetName()] = true
	return &emptypb.Empty{}, nil
}

func (f *fakeNydusSnapshots) View(_ context.Context, _ *snapshotsapi.ViewSnapshotRequest, _ ...grpc.CallOption) (*snapshotsapi.ViewSnapshotResponse, error) {
	return &snapshotsapi.ViewSnapshotResponse{Mounts: f.viewMounts}, nil
}

func (f *fakeNydusSnapshots) Remove(_ context.Context, _ *snapshotsapi.RemoveSnapshotRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (f *fakeNydusSnapshots) Stat(_ context.Context, in *snapshotsapi.StatSnapshotRequest, _ ...grpc.CallOption) (*snapshotsapi.StatSnapshotResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.committed[in.GetKey()] {
		return &snapshotsapi.StatSnapshotResponse{}, nil
	}
	return nil, status.Error(codes.NotFound, "not found")
}

// kernelFiles writes stand-ins for /proc/filesystems and /dev/cachefiles.
func kernelFiles(t *testing.T, filesystems string) (procFilesystems, cachefilesDev string) {
	t.Helper()
	dir := t.TempDir()
	procFilesystems = filepath.Join(dir, "filesystems")
	cachefilesDev = filepath.Join(dir, "cachefiles")
	if err := os.WriteFile(procFilesystems, []byte(filesystems), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachefilesDev, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	return procFilesystems, cachefilesDev
}

// newTestDriver returns a nydusDriver on a fake kernel with snaps as its
// snapshotter.
func newTestDriver(t *testing.T, cfg config, snaps snapshotsapi.SnapshotsClient, opts ...plugin.Option) *nydusDriver {
	t.Helper()
	cfg.ProcFilesystemsPath, cfg.CachefilesDevPath = kernelFiles(t, "nodev\toverlay\n\terofs\n")
	cfg.DockerConfigPath = filepath.Join(t.TempDir(), "config.json")
	cfg.DriverOpts = append([]plugin.Option{
		plugin.WithWorkDir(filepath.Join(t.TempDir(), "work")),
		plugin.WithSnapshotsClient(snaps),
		plugin.WithImageResolver(func(context.Context, string, *client.AuthConfig) (string, *v1.Config, []string, []string, error) {
			return "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", &v1.Config{}, testDiffIDs, testLayerDigests, nil
		}),
	}, opts...)
	d, err := newNydusDriver(cfg)
	if err != nil {
		t.Fatalf("newNydusDriver: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

// serve serves d on a Unix socket and returns a client for it.
func serve(t *testing.T, d *nydusDriver) *client.GRPCClient {
	t.Helper()
	dir, err := os.MkdirTemp("", "ipp-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sock := filepath.Join(dir, "p.sock")
	lis, err := plugin.ListenUnixSocket(sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	imageprovider.RegisterImageProviderPluginServer(srv, plugin.NewServer(d.Driver))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	c, err := client.NewGRPCClient(sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func layerTar(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestVerifyKernelSupport(t *testing.T) {
	for _, tc := range []struct {
		name          string
		filesystems   string
		noCachefiles  bool
		wantErrSubstr string
	}{
		{name: "erofs and cachefiles", filesystems: "nodev\toverlay\n\terofs\n"},
		{name: "no erofs", filesystems: "nodev\toverlay\nnodev\tfuse\n\text4\n", wantErrSubstr: "no EROFS"},
		{name: "no cachefiles", filesystems: "\terofs\n", noCachefiles: true, wantErrSubstr: "no fscache device"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			procFilesystems, cachefilesDev := kernelFiles(t, tc.filesystems)
			if tc.noCachefiles {
				cachefilesDev += "-missing"
			}
			err := verifyKernelSupport(procFilesystems, cachefilesDev)
			if tc.wantErrSubstr == "" {
				if err != nil {
					t.Fatalf("verifyKernelSupport: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErrSubstr) {
				t.Fatalf("verifyKernelSupport = %v, want an error containing %q", err, tc.wantErrSubstr)
			}
		})
	}
}

func TestCanProvideFollowsDragonflyHealth(t *testing.T) {
	var healthy atomic.Bool
	healthy.Store(true)
	df := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if !healthy.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer df.Close()

	snaps := &fakeNydusSnapshots{committed: make(map[string]bool)}
	d := newTestDriver(t, config{DragonflyHealthURL: df.URL + "/healthy", VerifyDragonflyPing: true}, snaps)
	c := serve(t, d)

	req := &client.PrepareRequest{ImageRef: testImageRef}
	if ok, err := c.CanProvide(t.Context(), req); err != nil || !ok {
		t.Fatalf("CanProvide with a healthy dfdaemon = (%v, %v), want (true, nil)", ok, err)
	}
	healthy.Store(false)
	if ok, err := c.CanProvide(t.Context(), req); err != nil || ok {
		t.Fatalf("CanProvide with an unhealthy dfdaemon = (%v, %v), want (false, nil)", ok, err)
	}
}

func TestRegistryLayerReaderUsesDragonflyProxy(t *testing.T) {
	reg := httptest.NewServer(registry.New())
	defer reg.Close()
	repo, err := name.NewRepository(strings.TrimPrefix(reg.URL, "http://") + "/sandbox")
	if err != nil {
		t.Fatal(err)
	}
	want := layerTar(t, map[string]string{"image/image.boot": "bootstrap"})
	layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(want)), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := remote.WriteLayer(repo, layer); err != nil {
		t.Fatal(err)
	}
	digest, err := layer.Digest()
	if err != nil {
		t.Fatal(err)
	}

	// The fake proxy forwards to the registry in X-Dragonfly-Registry, like
	// the dfdaemon registry mirror.
	var mu sync.Mutex
	var proxied []string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		proxied = append(proxied, r.URL.Path)
		mu.Unlock()
		if got := r.Header.Get("X-Dragonfly-Registry"); got != reg.URL {
			http.Error(w, "unexpected X-Dragonfly-Registry "+got, http.StatusBadRequest)
			return
		}
		resp, err := http.Get(reg.URL + r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	defer proxy.Close()

	snaps := &fakeNydusSnapshots{committed: make(map[string]bool)}
	d := newTestDriver(t, config{DragonflyProxyAddr: proxy.URL}, snaps)
	rc, err := d.registryLayerReader(t.Context(), repo.Tag("v1").String(), nil, 0, digest.String())
	if err != nil {
		t.Fatalf("registryLayerReader: %v", err)
	}
	got, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("layer content: got %d bytes, want %d", len(got), len(want))
	}
	mu.Lock()
	defer mu.Unlock()
	if wantPath := "/v2/sandbox/blobs/" + digest.String(); !slices.Equal(proxied, []string{wantPath}) {
		t.Errorf("proxied requests = %v, want [%s]", proxied, wantPath)
	}
}

func TestPrepareImageMountsBootstrap(t *testing.T) {
	activeDir := t.TempDir()
	snaps := &fakeNydusSnapshots{
		committed:  make(map[string]bool),
		activeDir:  activeDir,
		viewMounts: []*types.Mount{{Type: "erofs", Source: "erofs", Options: []string{"ro", "fsid=f1", "domain_id=d1"}}},
	}
	bootstrap := layerTar(t, map[string]string{"image/image.boot": "bootstrap"})

	var mu sync.Mutex
	var mountData []string
	var unmounted []string
	d := newTestDriver(t, config{
		LayerAnnotations: func(context.Context, string, *client.AuthConfig) ([]map[string]string, error) {
			return []map[string]string{
				{labelNydusBlob: "true"},
				{labelNydusBootstrap: "true", labelNydusFSVersion: "6"},
			}, nil
		},
		LayerReader: func(_ context.Context, _ string, _ *client.AuthConfig, i int, layerDigest string) (io.ReadCloser, error) {
			if i != 1 || layerDigest != testLayerDigests[1] {
				return nil, errors.New("read a layer other than the bootstrap")
			}
			return io.NopCloser(bytes.NewReader(bootstrap)), nil
		},
		Mount: func(_, target, fstype string, _ uintptr, data string) error {
			if fstype != "erofs" {
				return errors.New("mounted " + fstype)
			}
			mu.Lock()
			mountData = append(mountData, data)
			mu.Unlock()
			return os.WriteFile(filepath.Join(target, "hello.txt"), []byte("erofs"), 0o644)
		},
		Unmount: func(target string, _ int) error {
			mu.Lock()
			unmounted = append(unmounted, target)
			mu.Unlock()
			return nil
		},
		StatFS: func(string) (int64, error) { return erofsSuperMagic, nil },
	}, snaps, plugin.WithLeaseIdleTTL(0))
	c := serve(t, d)

	res, err := c.PrepareImage(t.Context(), &client.PrepareRequest{ImageRef: testImageRef, LeaseID: "actor-1/app"})
	if err != nil {
		t.Fatalf("PrepareImage: %v", err)
	}
	if res.ProviderName != providerName {
		t.Errorf("ProviderName = %q, want %q", res.ProviderName, providerName)
	}
	if len(res.LayerDirs) != 1 {
		t.Fatalf("LayerDirs = %v, want one directory", res.LayerDirs)
	}
	if got, err := os.ReadFile(filepath.Join(res.LayerDirs[0], "fs", "hello.txt")); err != nil || string(got) != "erofs" {
		t.Errorf("reading hello.txt = (%q, %v), want erofs", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(activeDir, "image", "image.boot")); err != nil || string(got) != "bootstrap" {
		t.Errorf("reading the unpacked bootstrap = (%q, %v), want bootstrap", got, err)
	}
	chains := plugin.ComputeChainIDs(testDiffIDs)
	snaps.mu.Lock()
	var committed []string
	for _, r := range snaps.commitReqs {
		committed = append(committed, r.GetName())
	}
	snaps.mu.Unlock()
	if !slices.Equal(committed, []string{chains[1].ChainID}) {
		t.Errorf("committed %v, want the bootstrap chain [%s]", committed, chains[1].ChainID)
	}
	mu.Lock()
	if !slices.Equal(mountData, []string{"fsid=f1,domain_id=d1"}) {
		t.Errorf("mount data = %q, want [fsid=f1,domain_id=d1]", mountData)
	}
	mu.Unlock()

	if err := c.ReleaseImage(t.Context(), &client.ReleaseRequest{LeaseID: "actor-1/app", ImageDigest: res.ImageDigest}); err != nil {
		t.Fatalf("ReleaseImage: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(unmounted) != 1 {
		t.Errorf("unmounted %v, want the one EROFS mount", unmounted)
	}
}

func TestPrepareImageDeclinesNonNydusImages(t *testing.T) {
	for _, tc := range []struct {
		name        string
		annotations []map[string]string
	}{
		{name: "OCI image", annotations: []map[string]string{{}, {}}},
		{name: "RAFS v5", annotations: []map[string]string{{labelNydusBlob: "true"}, {labelNydusBootstrap: "true", labelNydusFSVersion: "5"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snaps := &fakeNydusSnapshots{committed: make(map[string]bool), activeDir: t.TempDir()}
			d := newTestDriver(t, config{
				LayerAnnotations: func(context.Context, string, *client.AuthConfig) ([]map[string]string, error) {
					return tc.annotations, nil
				},
			}, snaps)
			c := serve(t, d)
			_, err := c.PrepareImage(t.Context(), &client.PrepareRequest{ImageRef: testImageRef, LeaseID: "actor-1/app"})
			if !errors.Is(err, client.ErrNotSupported) {
				t.Fatalf("PrepareImage = %v, want ErrNotSupported", err)
			}
		})
	}
}
