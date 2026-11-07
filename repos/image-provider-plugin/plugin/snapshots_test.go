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
	"slices"
	"testing"

	"github.com/containerd/containerd/api/types"
)

func TestExtractMountDir(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mounts []*types.Mount
		want   string
	}{
		{name: "none"},
		{
			name:   "bind",
			mounts: []*types.Mount{{Type: "bind", Source: "/snap/1/fs", Options: []string{"ro", "rbind"}}},
			want:   "/snap/1/fs",
		},
		{
			name:   "overlay view",
			mounts: []*types.Mount{{Type: "overlay", Source: "overlay", Options: []string{"lowerdir=/snap/2/fs:/snap/1/fs"}}},
			want:   "/snap/2/fs",
		},
		{
			name:   "overlay active",
			mounts: []*types.Mount{{Type: "overlay", Source: "overlay", Options: []string{"workdir=/snap/3/work", "upperdir=/snap/3/fs", "lowerdir=/snap/1/fs"}}},
			want:   "/snap/1/fs",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExtractMountDir(tc.mounts); got != tc.want {
				t.Errorf("ExtractMountDir = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStandardCRILayerLabels(t *testing.T) {
	digests := []string{"sha256:l0", "sha256:l1", "sha256:l2"}
	got := StandardCRILayerLabels("example.com/app@sha256:m", "sha256:m", digests, 1, "sha256:c1")
	want := map[string]string{
		"containerd.io/snapshot/remote":              "true",
		"containerd.io/snapshot.ref":                 "sha256:c1",
		"containerd.io/snapshot/cri.image-ref":       "example.com/app@sha256:m",
		"containerd.io/snapshot/cri.manifest-digest": "sha256:m",
		"containerd.io/snapshot/cri.layer-digest":    "sha256:l1",
		"containerd.io/snapshot/cri.image-layers":    "sha256:l1,sha256:l2",
	}
	if len(got) != len(want) {
		t.Errorf("labels = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("label %s = %q, want %q", k, got[k], v)
		}
	}
}

func TestSnapshotHelpers(t *testing.T) {
	diffIDs := []string{"sha256:1111111111111111111111111111111111111111111111111111111111111111"}
	chainID := ComputeChainIDs(diffIDs)[0].ChainID
	f := newFakeSnapshotsClient(t, diffIDs)
	ctx := t.Context()

	if ok, err := HasSnapshot(ctx, f, "snap", chainID); err != nil || ok {
		t.Fatalf("HasSnapshot before commit = (%v, %v), want (false, nil)", ok, err)
	}
	f.committed[chainID] = true
	if ok, err := HasSnapshot(ctx, f, "snap", chainID); err != nil || !ok {
		t.Fatalf("HasSnapshot after commit = (%v, %v), want (true, nil)", ok, err)
	}

	mounts, err := ViewLayerMounts(ctx, f, "snap", "view-0", chainID)
	if err != nil {
		t.Fatalf("ViewLayerMounts: %v", err)
	}
	if got := ExtractMountDir(mounts); got != f.mountDirs[chainID] {
		t.Errorf("view mount dir = %q, want %q", got, f.mountDirs[chainID])
	}

	RemoveSnapshot(ctx, f, "snap", "view-0")
	if got := f.removed(); !slices.Equal(got, []string{"view-0"}) {
		t.Errorf("removed keys = %v, want [view-0]", got)
	}
}
