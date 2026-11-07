// Copyright 2026 The Agent Substrate Authors
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
	"os"
	"path/filepath"
	"testing"

	providerclient "github.com/ai-on-gke/image-provider-plugin/client"
	providermock "github.com/ai-on-gke/image-provider-plugin/client/mock"
	"github.com/google/go-cmp/cmp"

	"go.ate.dev/substrate/internal/imagecache"
)

// writeTestBundle writes spec as the overlay spec of actorUID's container
// bundle under actorsDir.
func writeTestBundle(t *testing.T, actorsDir, actorUID, container string, spec *imagecache.OverlaySpec) {
	t.Helper()
	dir := filepath.Join(actorsDir, actorUID, "bundles", container)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := imagecache.WriteSpec(dir, spec); err != nil {
		t.Fatal(err)
	}
}

// The plugin learns every lease the bundles record, and nothing about images
// from the cache.
func TestReconcileProviderLeases(t *testing.T) {
	actorsDir := t.TempDir()
	const app, tools, local = "registry.example/app:1", "registry.example/tools:1", "registry.example/local:1"
	writeTestBundle(t, actorsDir, "actor-1", "main", &imagecache.OverlaySpec{
		ImageRef: app, ImageDigest: "sha256:aaa", LeaseID: "actor-1/main", Layers: []string{"/plugin/app/0"},
		ImageVolumes: []imagecache.ImageVolumeOverlay{{
			Name: "tools", ImageRef: tools, ImageDigest: "sha256:bbb", LeaseID: "actor-1/main/tools", Layers: []string{"/plugin/tools/0"},
		}},
	})
	writeTestBundle(t, actorsDir, "actor-2", "main", &imagecache.OverlaySpec{
		ImageRef: app, ImageDigest: "sha256:aaa", LeaseID: "actor-2/main", Layers: []string{"/plugin/app/0"},
	})
	writeTestBundle(t, actorsDir, "actor-3", "main", &imagecache.OverlaySpec{
		ImageRef: local, ImageDigest: "sha256:ccc", Layers: []string{"/cache/ccc"},
	})

	provider := providermock.New("nydus")
	s := &Atelet{imageProvider: provider}
	if err := s.reconcileProviderLeases(t.Context(), actorsDir); err != nil {
		t.Fatalf("reconcileProviderLeases: %v", err)
	}
	want := [][]*providerclient.ActiveLease{{
		{LeaseID: "actor-1/main", ImageRef: app, ImageDigest: "sha256:aaa"},
		{LeaseID: "actor-1/main/tools", ImageRef: tools, ImageDigest: "sha256:bbb"},
		{LeaseID: "actor-2/main", ImageRef: app, ImageDigest: "sha256:aaa"},
	}}
	if diff := cmp.Diff(want, provider.ReconcileCalls); diff != "" {
		t.Errorf("ReconcileImages calls mismatch (-want +got):\n%s", diff)
	}
}
