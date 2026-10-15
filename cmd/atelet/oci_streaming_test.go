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

	"github.com/agent-substrate/substrate/internal/ateompath"
	"github.com/agent-substrate/substrate/internal/imagecache"
	"github.com/agent-substrate/substrate/internal/ocilayer"
	"github.com/agent-substrate/substrate/internal/ocilayer/mock"
	"github.com/agent-substrate/substrate/internal/proto/ateletpb"
	"github.com/agent-substrate/substrate/internal/resources"
	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type staticKeychain struct {
	username string
	password string
}

func (k staticKeychain) Resolve(_ authn.Resource) (authn.Authenticator, error) {
	return &authn.Basic{Username: k.username, Password: k.password}, nil
}

func TestEnsureContainerImage_StreamingAndFallback(t *testing.T) {
	ctx := context.Background()
	host := imageVolumeTestRegistry(t)
	refStream := host + "/app:stream"
	refFallback := host + "/app:fallback"

	pushTestImage(t, refStream, singleFileLayer(t, "app-bin", "hello"))
	pushTestImage(t, refFallback, singleFileLayer(t, "fallback-bin", "world"))

	store := newImageVolumeStore(t)
	fakeLayerDir := filepath.Join(t.TempDir(), "streamed-layer-0")
	if err := os.MkdirAll(filepath.Join(fakeLayerDir, "fs"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	layerClient := mock.New("riptide")
	layerClient.PrepareImageFunc = func(_ context.Context, req *ocilayer.PrepareRequest) (*ocilayer.PrepareResult, error) {
		if req.ImageRef == refStream {
			return &ocilayer.PrepareResult{
				LayerDirs:    []string{fakeLayerDir},
				ImageDigest:  "sha256:1111111111111111111111111111111111111111111111111111111111111111",
				ProviderName: "riptide",
				Config: &v1.Config{
					Entrypoint: []string{"/bin/streamed"},
				},
			}, nil
		}
		return nil, ocilayer.ErrNotSupported
	}

	kc := staticKeychain{username: "oauth2accesstoken", password: "secret-token"}

	// 1. Streamed image succeeds via OCILayerService and forwards credentials.
	img, err := ensureContainerImage(ctx, store, layerClient, kc, nil, refStream)
	if err != nil {
		t.Fatalf("ensureContainerImage(stream) failed: %v", err)
	}
	if !img.Streamed {
		t.Errorf("expected img.Streamed = true")
	}
	if len(img.LayerDirs) != 1 || img.LayerDirs[0] != fakeLayerDir {
		t.Errorf("unexpected LayerDirs: %v", img.LayerDirs)
	}
	if len(layerClient.PrepareCalls) != 1 {
		t.Fatalf("expected 1 PrepareImage call, got %d", len(layerClient.PrepareCalls))
	}
	if creds := layerClient.PrepareCalls[0].AuthConfig; creds == nil || creds.Password != "secret-token" {
		t.Errorf("expected forwarded credentials with secret-token, got %+v", creds)
	}

	// 2. Non-streamable image returns ErrNotSupported and transparently falls back to imageCache.
	imgFallback, err := ensureContainerImage(ctx, store, layerClient, nil, nil, refFallback)
	if err != nil {
		t.Fatalf("ensureContainerImage(fallback) failed: %v", err)
	}
	if imgFallback.Streamed {
		t.Errorf("expected fallback image Streamed = false")
	}
	if len(imgFallback.LayerDirs) != 1 {
		t.Fatalf("expected 1 unpacked layer dir, got %v", imgFallback.LayerDirs)
	}
	if _, err := os.Stat(filepath.Join(imgFallback.LayerDirs[0], "fs", "fallback-bin")); err != nil {
		t.Errorf("expected unpacked file in fallback layer: %v", err)
	}
}

func TestDynamicDrain_SkipsStreamingAndBypassesTTLOnRelease(t *testing.T) {
	ctx := context.Background()
	tmpBase := t.TempDir()
	origActorsDir := ateompath.ActorsDir
	ateompath.ActorsDir = filepath.Join(tmpBase, "actors")
	t.Cleanup(func() { ateompath.ActorsDir = origActorsDir })

	host := imageVolumeTestRegistry(t)
	ref := host + "/drain-app:v1"
	pushTestImage(t, ref, singleFileLayer(t, "bin/run", "#!/bin/sh\n"))

	store := newImageVolumeStore(t)
	fakeLayerDir := filepath.Join(tmpBase, "streaming", "layer-0")
	if err := os.MkdirAll(filepath.Join(fakeLayerDir, "fs"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	layerClient := mock.New("riptide")
	layerClient.PrepareImageFunc = func(_ context.Context, req *ocilayer.PrepareRequest) (*ocilayer.PrepareResult, error) {
		return &ocilayer.PrepareResult{
			LayerDirs:    []string{fakeLayerDir},
			ImageDigest:  "sha256:2222222222222222222222222222222222222222222222222222222222222222",
			ProviderName: "riptide",
			Config: &v1.Config{
				Entrypoint: []string{"/bin/run"},
			},
		}, nil
	}

	herder := &AteomHerder{
		imageCache:     store,
		ociLayerClient: layerClient,
	}

	// Step 1: Prepare bundle while streaming is enabled.
	actor1 := "actor-streamed"
	if err := prepareOCIDirectory(ctx, store, herder.activeLayerClient(), nil, nil, actor1, "app", ref, []string{"/bin/run"}, nil, nil, "/dev/null", nil, nil, nil, nil); err != nil {
		t.Fatalf("prepareOCIDirectory(streamed) failed: %v", err)
	}
	spec1, err := imagecache.ReadSpec(ateompath.OCIBundlePath(actor1, "app"))
	if err != nil {
		t.Fatalf("ReadSpec(actor1): %v", err)
	}
	if !spec1.Streamed {
		t.Fatalf("expected spec1.Streamed = true")
	}

	// Step 2: Enable dynamic drain on the node (simulating ate.dev/oci-layer-service=disabled).
	herder.SetStreamingDraining(true)
	if !herder.IsStreamingDraining() {
		t.Fatalf("expected IsStreamingDraining() = true")
	}

	// New or resuming actors immediately bypass OCILayerService and unpack into local imageCache.
	actor2 := "actor-drained"
	if err := prepareOCIDirectory(ctx, store, herder.activeLayerClient(), nil, nil, actor2, "app", ref, []string{"/bin/run"}, nil, nil, "/dev/null", nil, nil, nil, nil); err != nil {
		t.Fatalf("prepareOCIDirectory(drained) failed: %v", err)
	}
	spec2, err := imagecache.ReadSpec(ateompath.OCIBundlePath(actor2, "app"))
	if err != nil {
		t.Fatalf("ReadSpec(actor2): %v", err)
	}
	if spec2.Streamed {
		t.Errorf("expected spec2.Streamed = false when node is draining")
	}

	// Step 3: When the existing streamed actor checkpoints or terminates during drain,
	// releaseStreamedLayersForActor sets BypassTTL=true so FUSE views unmount immediately.
	workloadSpec := &ateletpb.WorkloadSpec{
		Containers: []*ateletpb.Container{{Name: "app", Image: ref}},
	}
	herder.releaseStreamedLayersForActor(ctx, actor1, resources.ActorRef{Atespace: "default", Name: "a1"}, workloadSpec)
	if len(layerClient.ReleaseCalls) != 1 {
		t.Fatalf("expected 1 ReleaseImage call, got %d", len(layerClient.ReleaseCalls))
	}
	if !layerClient.ReleaseCalls[0].BypassTTL {
		t.Errorf("expected BypassTTL = true when releasing during node drain")
	}
}
