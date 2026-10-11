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
	"os"
	"path/filepath"
	"testing"

	"github.com/agent-substrate/substrate/internal/imagecache"
	"github.com/agent-substrate/substrate/internal/ocilayer"
	"github.com/agent-substrate/substrate/internal/ocilayer/mock"
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
		Layers:      []string{"/var/lib/ateom-gvisor/streaming/img1/layer-0"},
		ImageVolumes: []imagecache.ImageVolumeOverlay{{
			Name:        "tools",
			ImageRef:    volStreamRef,
			ImageDigest: "sha256:bbb",
			Streamed:    true,
			Layers:      []string{"/var/lib/ateom-gvisor/streaming/vol1/layer-0"},
		}},
	})
	writeBundle("actor-2", "main", &imagecache.OverlaySpec{
		ImageRef:    streamRef,
		ImageDigest: "sha256:aaa",
		Streamed:    true,
		Layers:      []string{"/var/lib/ateom-gvisor/streaming/img1/layer-0"},
	})
	writeBundle("actor-3", "main", &imagecache.OverlaySpec{
		ImageRef:    localRef,
		ImageDigest: "sha256:ccc",
		Streamed:    false,
		Layers:      []string{"/var/lib/ateom-gvisor/image-cache/layers/sha256/ccc"},
	})

	client := mock.New("riptide")
	if err := reconcileStreamingLeases(ctx, client, actorsDir); err != nil {
		t.Fatalf("reconcileStreamingLeases failed: %v", err)
	}

	if len(client.ReconcileCalls) != 1 {
		t.Fatalf("expected 1 ReconcileLayers call, got %d", len(client.ReconcileCalls))
	}
	reconciled := make(map[string]*ocilayer.ActiveImageLayers)
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
