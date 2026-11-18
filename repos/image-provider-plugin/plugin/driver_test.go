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
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	imageprovider "github.com/ai-on-gke/image-provider-plugin/api/imageprovider/v1alpha1"
	"github.com/ai-on-gke/image-provider-plugin/client"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

var testDiffIDs = []string{
	"sha256:1111111111111111111111111111111111111111111111111111111111111111",
	"sha256:2222222222222222222222222222222222222222222222222222222222222222",
}

const testDigest = "sha256:9999999999999999999999999999999999999999999999999999999999999999"

func staticResolver(diffIDs []string) ImageResolverFunc {
	return func(context.Context, string, *client.AuthConfig) (string, *v1.Config, []string, []string, error) {
		return testDigest, &v1.Config{Entrypoint: []string{"/bin/agent"}}, diffIDs, diffIDs, nil
	}
}

// newTestDriver returns a driver backed by a fake snapshotter that streams
// testDiffIDs.
func newTestDriver(t *testing.T, opts ...Option) (*Driver, *fakeSnapshotsClient) {
	t.Helper()
	snap := newFakeSnapshotsClient(t, testDiffIDs)
	opts = append([]Option{
		WithWorkDir(filepath.Join(t.TempDir(), "work")),
		WithSnapshotsClient(snap),
		WithImageResolver(staticResolver(testDiffIDs)),
		WithAuthRefresh(imageprovider.AuthRefresh_AUTH_REFRESH_NOT_NEEDED),
	}, opts...)
	d, err := NewDriver("test-provider", "", "test-snap", opts...)
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d, snap
}

func prepare(t *testing.T, d *Driver, ref, leaseID string) *client.PrepareResult {
	t.Helper()
	res, err := d.PrepareImage(t.Context(), &client.PrepareRequest{ImageRef: ref, LeaseID: leaseID})
	if err != nil {
		t.Fatalf("PrepareImage(%s): %v", ref, err)
	}
	return res
}

func release(t *testing.T, d *Driver, leaseID string) {
	t.Helper()
	if err := d.ReleaseImage(t.Context(), &client.ReleaseRequest{LeaseID: leaseID, ImageDigest: testDigest}); err != nil {
		t.Fatalf("ReleaseImage(%s): %v", leaseID, err)
	}
}

func leaseCount(d *Driver, ref string) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.leaseCountLocked(ref)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestDriverLeaseLifecycle(t *testing.T) {
	d, snap := newTestDriver(t, WithLeaseIdleTTL(0))
	const ref = "us-central1-docker.pkg.dev/proj/repo/agent:v1"

	res := prepare(t, d, ref, "a")
	if res.ImageDigest != testDigest || res.ProviderName != "test-provider" || res.Config.Entrypoint[0] != "/bin/agent" {
		t.Errorf("PrepareImage = %+v, want the resolved image", res)
	}
	if len(res.LayerDirs) != 2 {
		t.Fatalf("LayerDirs = %v, want 2 layers", res.LayerDirs)
	}
	for i, dir := range res.LayerDirs {
		if !exists(filepath.Join(dir, "hello.txt")) {
			t.Errorf("layer %d at %s is not the mounted view", i, dir)
		}
	}

	// A second lease reuses the mounted views, and a retry of it takes no
	// third lease.
	prepare(t, d, ref, "b")
	prepare(t, d, ref, "b")
	snap.mu.Lock()
	views := len(snap.viewReqs)
	snap.mu.Unlock()
	if views != 2 {
		t.Errorf("View calls = %d after warm prepares, want 2", views)
	}
	if got := leaseCount(d, ref); got != 2 {
		t.Errorf("leases = %d, want 2", got)
	}

	release(t, d, "a")
	if got := snap.removed(); len(got) != 0 {
		t.Errorf("removed %v with one lease left, want nothing", got)
	}
	release(t, d, "b")
	if got := snap.removed(); len(got) != 2 {
		t.Errorf("removed %v after the last release, want both views", got)
	}
	if exists(d.imageWorkDir(ref)) {
		t.Errorf("image work dir still exists after the last release")
	}

	// Releasing a lease the driver does not hold does nothing.
	release(t, d, "b")
}

func TestDriverLeaseErrors(t *testing.T) {
	d, _ := newTestDriver(t)
	const ref = "example.com/app:v1"
	prepare(t, d, ref, "a")

	if _, err := d.PrepareImage(t.Context(), &client.PrepareRequest{ImageRef: "example.com/other:v1", LeaseID: "a"}); !errors.Is(err, errLeaseConflict) {
		t.Errorf("PrepareImage of another image with a held lease = %v, want errLeaseConflict", err)
	}
	if err := d.ReleaseImage(t.Context(), &client.ReleaseRequest{LeaseID: "a", ImageDigest: "sha256:other"}); !errors.Is(err, errLeaseMismatch) {
		t.Errorf("ReleaseImage with another digest = %v, want errLeaseMismatch", err)
	}
	if got := leaseCount(d, ref); got != 1 {
		t.Errorf("leases = %d after the failed calls, want 1", got)
	}
}

func TestNewDriverAuthRefresh(t *testing.T) {
	hook := WithAuthRefreshHook(func(context.Context, string, *client.RefreshRequest) error { return nil })
	for _, tc := range []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{name: "unset", wantErr: true},
		{name: "not needed", opts: []Option{WithAuthRefresh(imageprovider.AuthRefresh_AUTH_REFRESH_NOT_NEEDED)}},
		{name: "required with a hook", opts: []Option{WithAuthRefresh(imageprovider.AuthRefresh_AUTH_REFRESH_REQUIRED), hook}},
		{name: "required without a hook", opts: []Option{WithAuthRefresh(imageprovider.AuthRefresh_AUTH_REFRESH_REQUIRED)}, wantErr: true},
		{name: "self renewal", opts: []Option{WithAuthRefresh(imageprovider.AuthRefresh_AUTH_REFRESH_NOT_NEEDED), hook, WithSelfRenewal(time.Minute)}},
		{name: "self renewal with required", opts: []Option{WithAuthRefresh(imageprovider.AuthRefresh_AUTH_REFRESH_REQUIRED), hook, WithSelfRenewal(time.Minute)}, wantErr: true},
		{name: "self renewal without a hook", opts: []Option{WithAuthRefresh(imageprovider.AuthRefresh_AUTH_REFRESH_NOT_NEEDED), WithSelfRenewal(time.Minute)}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := NewDriver("test-provider", "snap.sock", "", append([]Option{WithWorkDir(t.TempDir())}, tc.opts...)...)
			if err == nil {
				_ = d.Close()
			}
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Errorf("NewDriver error = %v, want error %v", err, tc.wantErr)
			}
		})
	}
}

func TestDriverRefreshAuth(t *testing.T) {
	var (
		hookErr error
		gotRef  string
		gotReq  *client.RefreshRequest
	)
	d, _ := newTestDriver(t,
		WithAuthRefresh(imageprovider.AuthRefresh_AUTH_REFRESH_REQUIRED),
		WithAuthRefreshHook(func(_ context.Context, canonicalRef string, req *client.RefreshRequest) error {
			gotRef, gotReq = canonicalRef, req
			return hookErr
		}))
	const ref = "example.com/app:v1"
	prepare(t, d, ref, "a")
	ctx := t.Context()

	auth := &client.AuthConfig{Username: "u", Password: "fresh"}
	got, err := d.RefreshAuth(ctx, &client.RefreshRequest{LeaseID: "a", AuthConfig: auth})
	if err != nil || got != imageprovider.AuthRefresh_AUTH_REFRESH_REQUIRED {
		t.Fatalf("RefreshAuth = %v, %v; want AUTH_REFRESH_REQUIRED", got, err)
	}
	if gotReq.ImageRef != ref || gotReq.AuthConfig != auth || gotRef != CanonicalImageRef(ref, testDigest) {
		t.Errorf("hook got %q, %+v; want the lease's image and the new credentials", gotRef, gotReq)
	}

	hookErr = errors.New("writing credentials failed")
	if _, err := d.RefreshAuth(ctx, &client.RefreshRequest{LeaseID: "a"}); !errors.Is(err, hookErr) {
		t.Errorf("RefreshAuth with a failing hook = %v, want %v", err, hookErr)
	}

	// RefreshAuth never takes a lease, also not after a release.
	if _, err := d.RefreshAuth(ctx, &client.RefreshRequest{LeaseID: "b"}); !errors.Is(err, errLeaseNotHeld) {
		t.Errorf("RefreshAuth of an unknown lease = %v, want errLeaseNotHeld", err)
	}
	release(t, d, "a")
	if _, err := d.RefreshAuth(ctx, &client.RefreshRequest{LeaseID: "a"}); !errors.Is(err, errLeaseNotHeld) {
		t.Errorf("RefreshAuth of a released lease = %v, want errLeaseNotHeld", err)
	}
	d.mu.Lock()
	_, heldA := d.leaseRefs["a"]
	_, heldB := d.leaseRefs["b"]
	d.mu.Unlock()
	if heldA || heldB {
		t.Error("RefreshAuth took a lease")
	}
}

func TestDriverIdleTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d, snap := newTestDriver(t, WithLeaseIdleTTL(time.Minute))
		const ref = "example.com/app:v1"

		res := prepare(t, d, ref, "a")
		release(t, d, "a")
		time.Sleep(30 * time.Second)
		if !exists(res.LayerDirs[0]) {
			t.Fatal("image unmounted before the idle TTL")
		}

		// A new lease within the TTL reuses the image and stops the timer.
		prepare(t, d, ref, "b")
		time.Sleep(2 * time.Minute)
		if !exists(d.imageWorkDir(ref)) {
			t.Fatal("leased image unmounted")
		}

		release(t, d, "b")
		time.Sleep(2 * time.Minute)
		synctest.Wait()
		if exists(d.imageWorkDir(ref)) {
			t.Error("image still mounted after the idle TTL")
		}
		if got := snap.removed(); len(got) != 2 {
			t.Errorf("removed %v after the idle TTL, want both views", got)
		}
	})
}

func TestDriverCoalescesConcurrentPrepares(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		block := make(chan struct{})
		var calls atomic.Int32
		d, _ := newTestDriver(t, WithLeaseIdleTTL(0), WithLayerMounter(func(ctx context.Context, env *MountEnv) (*MountedLayers, error) {
			calls.Add(1)
			<-block
			return StandardMountLayers(ctx, env)
		}))
		const ref = "example.com/app:v1"

		const n = 5
		var wg sync.WaitGroup
		for i := range n {
			wg.Go(func() { prepare(t, d, ref, fmt.Sprint(i)) })
		}
		// One call is in the mounter and the others wait for it.
		synctest.Wait()
		close(block)
		wg.Wait()

		if got := calls.Load(); got != 1 {
			t.Errorf("mounter ran %d times, want 1", got)
		}
		if got := leaseCount(d, ref); got != n {
			t.Errorf("leases = %d, want %d", got, n)
		}
	})
}

func TestDriverPrepareRollsBack(t *testing.T) {
	for _, tc := range []struct {
		name             string
		mounter          LayerMounterFunc
		decline          bool
		wantNotSupported bool
		wantRemoved      []string
	}{
		{
			name:             "snapshotter declines",
			decline:          true,
			wantNotSupported: true,
		},
		{
			name: "mounter fails",
			mounter: func(context.Context, *MountEnv) (*MountedLayers, error) {
				return &MountedLayers{SnapshotKeys: []string{"view"}, CommittedKeys: []string{"c0", "c1"}}, errors.New("boom")
			},
			wantRemoved: []string{"view", "c1", "c0"},
		},
		{
			name: "mount not listable",
			mounter: func(context.Context, *MountEnv) (*MountedLayers, error) {
				return &MountedLayers{MountDirs: []string{"/does/not/exist"}, SnapshotKeys: []string{"view"}}, nil
			},
			wantRemoved: []string{"view"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				opts := []Option{WithListableTimeout(time.Second)}
				if tc.mounter != nil {
					opts = append(opts, WithLayerMounter(tc.mounter))
				}
				d, snap := newTestDriver(t, opts...)
				snap.declinePrep = tc.decline
				const ref = "example.com/app:v1"

				_, err := d.PrepareImage(t.Context(), &client.PrepareRequest{ImageRef: ref, LeaseID: "a"})
				if err == nil {
					t.Fatal("PrepareImage succeeded, want error")
				}
				if got := leaseCount(d, ref); got != 0 {
					t.Errorf("leases = %d after a failed prepare, want 0", got)
				}
				if got := errors.Is(err, client.ErrNotSupported); got != tc.wantNotSupported {
					t.Errorf("errors.Is(%v, ErrNotSupported) = %v, want %v", err, got, tc.wantNotSupported)
				}
				if tc.wantRemoved != nil && !slices.Equal(snap.removed(), tc.wantRemoved) {
					t.Errorf("removed %v, want %v", snap.removed(), tc.wantRemoved)
				}
				if exists(filepath.Join(d.workDir, SanitizePathKey(ref))) {
					t.Error("image work dir left behind")
				}
			})
		})
	}
}

func TestDriverRepreparesMissingLayers(t *testing.T) {
	d, snap := newTestDriver(t)
	const ref = "example.com/app:v1"

	res := prepare(t, d, ref, "a")
	// The layer mount is gone, for example after a node cleanup.
	if err := os.RemoveAll(res.LayerDirs[0]); err != nil {
		t.Fatal(err)
	}
	res2 := prepare(t, d, ref, "b")
	snap.mu.Lock()
	views := len(snap.viewReqs)
	snap.mu.Unlock()
	if views != 4 {
		t.Errorf("View calls = %d, want 4 after preparing again", views)
	}
	if !layersExist(res2.LayerDirs) {
		t.Errorf("layer dirs %v missing after preparing again", res2.LayerDirs)
	}
	if got := leaseCount(d, ref); got != 2 {
		t.Errorf("leases = %d, want 2", got)
	}
}
