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
	"testing/synctest"

	"github.com/ai-on-gke/image-provider-plugin/client"
)

// restartDriver returns a new driver on d's work dir and snapshotter, as a
// restarted plugin would be.
func restartDriver(t *testing.T, d *Driver, snap *fakeSnapshotsClient, opts ...Option) *Driver {
	t.Helper()
	opts = append([]Option{
		WithWorkDir(d.workDir),
		WithSnapshotsClient(snap),
		WithImageResolver(staticResolver(testDiffIDs)),
	}, opts...)
	nd, err := NewDriver(d.name, "", d.snapshotterName, opts...)
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}
	t.Cleanup(func() { _ = nd.Close() })
	return nd
}

// A restarted plugin takes over the images atelet still uses from their
// lease.json, and removes the rest.
func TestDriverReconcileAfterRestart(t *testing.T) {
	d, snap := newTestDriver(t, WithLeaseIdleTTL(0))
	const used, unused, never = "example.com/used:v1", "example.com/unused:v1", "example.com/never-prepared:v1"
	usedRes := prepare(t, d, used, "u1")
	unusedRes := prepare(t, d, unused, "x")
	_ = d.Close()

	nd := restartDriver(t, d, snap, WithLeaseIdleTTL(0))
	if err := nd.ReconcileImages(t.Context(), []*client.ActiveLease{
		{LeaseID: "u1", ImageRef: used, ImageDigest: testDigest},
		{LeaseID: "u2", ImageRef: used, ImageDigest: testDigest},
		{LeaseID: "n", ImageRef: never, ImageDigest: testDigest},
	}); err != nil {
		t.Fatalf("ReconcileImages: %v", err)
	}
	if got := leaseCount(nd, never); got != 0 {
		t.Errorf("leases on an image without a work dir = %d, want 0", got)
	}
	if exists(filepath.Dir(unusedRes.LayerDirs[0])) {
		t.Error("work dir of an image no workload uses still exists")
	}
	if got := len(snap.removed()); got != 2 {
		t.Errorf("removed %d snapshots, want the 2 views of the unused image", got)
	}

	snap.mu.Lock()
	views := len(snap.viewReqs)
	snap.mu.Unlock()
	res := prepare(t, nd, used, "u3")
	snap.mu.Lock()
	viewsAfter := len(snap.viewReqs)
	snap.mu.Unlock()
	if viewsAfter != views {
		t.Errorf("PrepareImage of a reconciled image made %d View calls, want 0", viewsAfter-views)
	}
	if res.Config == nil || res.ImageDigest != testDigest {
		t.Errorf("PrepareImage of a reconciled image = %+v, want its config and digest", res)
	}

	// Two reconciled leases and the new one.
	for _, id := range []string{"u1", "u2", "u3"} {
		release(t, nd, id)
	}
	if exists(filepath.Dir(usedRes.LayerDirs[0])) {
		t.Error("work dir still exists after the last lease was released")
	}
}

// A lease that ReconcileImages keeps on an image whose preparation is in
// flight goes away when the preparation fails.
func TestDriverReconcileDuringFailedPrepare(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		block := make(chan struct{})
		d, _ := newTestDriver(t, WithLayerMounter(func(context.Context, *MountEnv) (*MountedLayers, error) {
			<-block
			return nil, errors.New("boom")
		}))
		const ref = "example.com/app:v1"
		done := make(chan error)
		go func() {
			_, err := d.PrepareImage(t.Context(), &client.PrepareRequest{ImageRef: ref, LeaseID: "a"})
			done <- err
		}()
		synctest.Wait()

		if err := d.ReconcileImages(t.Context(), []*client.ActiveLease{{LeaseID: "b", ImageRef: ref, ImageDigest: testDigest}}); err != nil {
			t.Fatalf("ReconcileImages: %v", err)
		}
		close(block)
		if err := <-done; err == nil {
			t.Fatal("PrepareImage succeeded, want error")
		}
		if got := leaseCount(d, ref); got != 0 {
			t.Fatalf("leases = %d after the preparation failed, want 0", got)
		}
		release(t, d, "b")
	})
}

// A lease that ReconcileImages keeps on an image being torn down goes away
// when the teardown ends.
func TestDriverReconcileDuringTeardown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		block := make(chan struct{})
		d, _ := newTestDriver(t, WithLeaseIdleTTL(0), WithWorkDirTeardownHook(func(string) { <-block }))
		const ref = "example.com/app:v1"
		prepare(t, d, ref, "a")
		done := make(chan struct{})
		go func() {
			if err := d.ReleaseImage(t.Context(), &client.ReleaseRequest{LeaseID: "a", ImageDigest: testDigest}); err != nil {
				t.Errorf("ReleaseImage(a): %v", err)
			}
			close(done)
		}()
		synctest.Wait()

		if err := d.ReconcileImages(t.Context(), []*client.ActiveLease{{LeaseID: "b", ImageRef: ref, ImageDigest: testDigest}}); err != nil {
			t.Fatalf("ReconcileImages: %v", err)
		}
		close(block)
		<-done
		if got := leaseCount(d, ref); got != 0 {
			t.Fatalf("leases = %d after the teardown, want 0", got)
		}
		release(t, d, "b")
	})
}

func TestDriverPrepareWorkDir(t *testing.T) {
	for _, tc := range []struct {
		name     string
		version  string
		wantKept bool
	}{
		{name: "same version", version: workDirLayoutVersion + "\n", wantKept: true},
		{name: "other version", version: "0\n"},
		{name: "no version"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, _ := newTestDriver(t)
			imageDir := filepath.Join(d.workDir, "image")
			if err := os.MkdirAll(imageDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if tc.version != "" {
				if err := os.WriteFile(filepath.Join(d.workDir, workDirVersionFile), []byte(tc.version), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := d.prepareWorkDir(t.Context()); err != nil {
				t.Fatalf("prepareWorkDir: %v", err)
			}
			if got := exists(imageDir); got != tc.wantKept {
				t.Errorf("image dir kept = %v, want %v", got, tc.wantKept)
			}
			got, err := os.ReadFile(filepath.Join(d.workDir, workDirVersionFile))
			if err != nil || string(got) != workDirLayoutVersion+"\n" {
				t.Errorf("version file = %q, %v; want %q", got, err, workDirLayoutVersion+"\n")
			}
		})
	}
}
