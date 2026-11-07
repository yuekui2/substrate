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
	"os"
	"path/filepath"
	"testing"

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
	const used, unused = "example.com/used:v1", "example.com/unused:v1"
	usedRes := prepare(t, d, used)
	unusedRes := prepare(t, d, unused)
	_ = d.Close()

	nd := restartDriver(t, d, snap, WithLeaseIdleTTL(0))
	if err := nd.ReconcileImages(t.Context(), []*client.ActiveImageLayers{
		{ImageRef: used, RefCount: 2},
		{ImageRef: "example.com/never-prepared:v1", RefCount: 1},
	}); err != nil {
		t.Fatalf("ReconcileImages: %v", err)
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
	res := prepare(t, nd, used)
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
	for range 3 {
		release(t, nd, used)
	}
	if exists(filepath.Dir(usedRes.LayerDirs[0])) {
		t.Error("work dir still exists after the last lease was released")
	}
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
