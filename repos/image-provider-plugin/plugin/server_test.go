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
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	imageprovider "github.com/ai-on-gke/image-provider-plugin/api/imageprovider/v1alpha1"
	"github.com/ai-on-gke/image-provider-plugin/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// socketDir returns a directory short enough for Unix socket paths.
func socketDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ipp-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// startServer serves d on a new socket until the returned cancel is called.
// It returns a client for it, and a channel that is closed when serve
// returns, with serve's result in *serveErr.
func startServer(t *testing.T, d *Driver) (c *client.GRPCClient, cancel context.CancelFunc, done <-chan struct{}, serveErr *error) {
	t.Helper()
	sock := filepath.Join(socketDir(t), "p.sock")
	lis, err := ListenUnixSocket(sock)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan struct{})
	serveErr = new(error)
	go func() {
		*serveErr = serve(ctx, lis, d)
		close(stopped)
	}()
	t.Cleanup(func() {
		cancel()
		<-stopped
	})
	c, err = client.NewGRPCClient(sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c, cancel, stopped, serveErr
}

// waitFor polls cond until it holds, failing the test after a few seconds.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestServerLeaseLifecycle(t *testing.T) {
	d, snap := newTestDriver(t, WithLeaseIdleTTL(0), WithDrainTimeout(time.Second))
	c, _, _, _ := startServer(t, d)
	ctx := t.Context()
	const ref = "example.com/app:v1"

	// The fake snapshotter streams no layer while it declines, and every
	// layer once one image streamed, so the decline comes first.
	snap.mu.Lock()
	snap.declinePrep = true
	snap.mu.Unlock()
	if _, err := c.PrepareImage(ctx, &client.PrepareRequest{ImageRef: "example.com/plain:v1", LeaseID: "a"}); !errors.Is(err, client.ErrNotSupported) {
		t.Errorf("PrepareImage of a declined image = %v, want ErrNotSupported", err)
	}
	snap.mu.Lock()
	snap.declinePrep = false
	snap.mu.Unlock()

	ok, err := c.CanProvide(ctx, &client.PrepareRequest{ImageRef: ref})
	if err != nil || !ok {
		t.Fatalf("CanProvide = %v, %v; want true", ok, err)
	}
	if c.Name() != "test-provider" {
		t.Errorf("Name = %q, want test-provider", c.Name())
	}
	res, err := c.PrepareImage(ctx, &client.PrepareRequest{
		ImageRef:   ref,
		AuthConfig: &client.AuthConfig{Username: "u", Password: "p"},
		LeaseID:    "a",
	})
	if err != nil {
		t.Fatalf("PrepareImage: %v", err)
	}
	if res.ImageDigest != testDigest || res.Config == nil || res.Config.Entrypoint[0] != "/bin/agent" || len(res.LayerDirs) != 2 {
		t.Errorf("PrepareImage = %+v, want the prepared image", res)
	}
	if err := c.ReleaseImage(ctx, &client.ReleaseRequest{LeaseID: "a", ImageDigest: res.ImageDigest}); err != nil {
		t.Fatalf("ReleaseImage: %v", err)
	}
	if exists(filepath.Dir(res.LayerDirs[0])) {
		t.Error("image still mounted after its last lease was released")
	}

}

func TestServerReconcileImages(t *testing.T) {
	d, _ := newTestDriver(t, WithDrainTimeout(time.Second))
	c, _, _, _ := startServer(t, d)
	const used, unused = "example.com/used:v1", "example.com/unused:v1"
	usedRes := prepare(t, d, used, "u")
	unusedRes := prepare(t, d, unused, "x")

	if err := c.ReconcileImages(t.Context(), []*client.ActiveLease{{LeaseID: "u", ImageRef: used, ImageDigest: testDigest}}); err != nil {
		t.Fatalf("ReconcileImages: %v", err)
	}
	if exists(filepath.Dir(unusedRes.LayerDirs[0])) {
		t.Error("an image no workload uses is still mounted after ReconcileImages")
	}
	if !exists(usedRes.LayerDirs[0]) || !d.holdsLease("u") {
		t.Error("the image a workload uses lost its layers or lease after ReconcileImages")
	}
}

func TestServerLeaseErrors(t *testing.T) {
	d, _ := newTestDriver(t)
	s := NewServer(d)
	const ref = "example.com/app:v1"
	prepare(t, d, ref, "a")

	for _, tc := range []struct {
		name string
		call func(context.Context) error
		want codes.Code
	}{
		{
			name: "prepare without an image ref",
			call: func(ctx context.Context) error {
				_, err := s.PrepareImage(ctx, &imageprovider.PrepareImageRequest{LeaseId: "b"})
				return err
			},
			want: codes.InvalidArgument,
		},
		{
			name: "prepare a held lease in another form",
			call: func(ctx context.Context) error {
				_, err := s.PrepareImage(ctx, &imageprovider.PrepareImageRequest{ImageRef: ref, LeaseId: "a", Form: imageprovider.LayerForm_LAYER_FORM_EROFS})
				return err
			},
			want: codes.AlreadyExists,
		},
		{
			name: "prepare a new lease in another form",
			call: func(ctx context.Context) error {
				_, err := s.PrepareImage(ctx, &imageprovider.PrepareImageRequest{ImageRef: ref, LeaseId: "b", Form: imageprovider.LayerForm_LAYER_FORM_EROFS})
				return err
			},
			want: codes.Unimplemented,
		},
		{
			name: "prepare without a lease ID",
			call: func(ctx context.Context) error {
				_, err := s.PrepareImage(ctx, &imageprovider.PrepareImageRequest{ImageRef: ref})
				return err
			},
			want: codes.InvalidArgument,
		},
		{
			name: "prepare with EROFS digests in directory form",
			call: func(ctx context.Context) error {
				_, err := s.PrepareImage(ctx, &imageprovider.PrepareImageRequest{ImageRef: ref, LeaseId: "b", RequiredErofsDigests: []string{testDigest}})
				return err
			},
			want: codes.InvalidArgument,
		},
		{
			name: "prepare another image with a held lease",
			call: func(ctx context.Context) error {
				_, err := s.PrepareImage(ctx, &imageprovider.PrepareImageRequest{ImageRef: "example.com/other:v1", LeaseId: "a"})
				return err
			},
			want: codes.AlreadyExists,
		},
		{
			name: "release without a lease ID",
			call: func(ctx context.Context) error {
				_, err := s.ReleaseImage(ctx, &imageprovider.ReleaseImageRequest{ImageDigest: testDigest})
				return err
			},
			want: codes.InvalidArgument,
		},
		{
			name: "release with another digest",
			call: func(ctx context.Context) error {
				_, err := s.ReleaseImage(ctx, &imageprovider.ReleaseImageRequest{LeaseId: "a", ImageDigest: "sha256:other"})
				return err
			},
			want: codes.InvalidArgument,
		},
		{
			name: "release a held lease in another form",
			call: func(ctx context.Context) error {
				_, err := s.ReleaseImage(ctx, &imageprovider.ReleaseImageRequest{LeaseId: "a", ImageDigest: testDigest, Form: imageprovider.LayerForm_LAYER_FORM_EROFS})
				return err
			},
			want: codes.InvalidArgument,
		},
		{
			name: "release an unknown lease in another form",
			call: func(ctx context.Context) error {
				_, err := s.ReleaseImage(ctx, &imageprovider.ReleaseImageRequest{LeaseId: "b", ImageDigest: testDigest, Form: imageprovider.LayerForm_LAYER_FORM_EROFS})
				return err
			},
			want: codes.OK,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := status.Code(tc.call(t.Context())); got != tc.want {
				t.Errorf("code = %v, want %v", got, tc.want)
			}
		})
	}
	if !d.holdsLease("a") {
		t.Error("lease released by a failed call")
	}
}

// When the server's context ends, it reports DRAINING and serves releases
// until the last lease goes, then stops.
func TestServerDrainsOnShutdown(t *testing.T) {
	d, _ := newTestDriver(t, WithDrainTimeout(time.Hour))
	c, cancel, done, serveErr := startServer(t, d)
	ctx := t.Context()
	const ref = "example.com/app:v1"
	if _, err := c.PrepareImage(ctx, &client.PrepareRequest{ImageRef: ref, LeaseID: "a"}); err != nil {
		t.Fatalf("PrepareImage: %v", err)
	}
	if state, err := c.State(ctx); err != nil || state != imageprovider.PluginState_PLUGIN_STATE_SERVING {
		t.Fatalf("State = %v, %v; want SERVING", state, err)
	}

	cancel()
	waitFor(t, "the DRAINING state", func() bool {
		state, err := c.State(ctx)
		return err == nil && state == imageprovider.PluginState_PLUGIN_STATE_DRAINING
	})
	if ok, err := c.CanProvide(ctx, &client.PrepareRequest{ImageRef: ref}); err != nil || ok {
		t.Errorf("CanProvide while draining = %v, %v; want false", ok, err)
	}
	select {
	case <-done:
		t.Fatal("server stopped with a lease in use")
	default:
	}
	if err := c.ReleaseImage(ctx, &client.ReleaseRequest{LeaseID: "a", ImageDigest: testDigest}); err != nil {
		t.Fatalf("ReleaseImage while draining: %v", err)
	}
	<-done
	if *serveErr != nil {
		t.Errorf("serve = %v, want nil", *serveErr)
	}
}

func TestServeUnixSocket(t *testing.T) {
	dir := socketDir(t)
	sock := filepath.Join(dir, "p.sock")
	drainComplete := filepath.Join(dir, "drain-complete")
	if err := os.WriteFile(drainComplete, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	d, _ := newTestDriver(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- ServeUnixSocket(ctx, sock, drainComplete, d) }()

	waitFor(t, "the plugin to serve", func() bool { return ProbeUnixSocket(t.Context(), sock) == nil })
	if exists(drainComplete) {
		t.Error("a stale drain-complete file survived startup")
	}
	if !exists(filepath.Join(d.workDir, workDirVersionFile)) {
		t.Error("ServeUnixSocket did not prepare the work dir")
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("ServeUnixSocket = %v, want nil", err)
	}
	if !exists(drainComplete) {
		t.Error("ServeUnixSocket did not write the drain-complete file")
	}
	if exists(sock) {
		t.Error("the socket file survived shutdown")
	}
}

func TestServeUnixSocketWaitsForProvider(t *testing.T) {
	d, _ := newTestDriver(t, WithStartupTimeout(50*time.Millisecond), WithProbeHook(func(context.Context) error {
		return errors.New("not ready")
	}))
	sock := filepath.Join(socketDir(t), "p.sock")
	if err := ServeUnixSocket(t.Context(), sock, "", d); err == nil {
		t.Fatal("ServeUnixSocket with a provider that is never ready = nil, want an error")
	}
	if exists(sock) {
		t.Error("ServeUnixSocket listened before the provider was ready")
	}
}
