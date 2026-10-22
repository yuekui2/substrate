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

	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"go.ate.dev/substrate/cmd/atelet/internal/ateletpath"
	"go.ate.dev/substrate/internal/imagecache"
	"go.ate.dev/substrate/internal/nodepath"
	"go.ate.dev/substrate/internal/proto/ateletpb"
	"go.ate.dev/substrate/internal/resources"
	"go.ate.dev/substrate/oci-image-accelerator/pkg/client"
	"go.ate.dev/substrate/oci-image-accelerator/pkg/client/mock"
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
	layerClient.PrepareImageFunc = func(_ context.Context, req *imageaccel.PrepareRequest) (*imageaccel.PrepareResult, error) {
		if req.ImageRef == refStream {
			return &imageaccel.PrepareResult{
				LayerDirs:    []string{fakeLayerDir},
				ImageDigest:  "sha256:1111111111111111111111111111111111111111111111111111111111111111",
				ProviderName: "riptide",
				Config: &v1.Config{
					Entrypoint: []string{"/bin/streamed"},
				},
			}, nil
		}
		return nil, imageaccel.ErrNotSupported
	}

	kc := staticKeychain{username: "oauth2accesstoken", password: "secret-token"}

	// 1. Streamed image succeeds via ImageAccelerator and forwards credentials.
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

func TestAdapterLameDuckDrain_FallsBackToImageCacheAndReleasesActiveLeases(t *testing.T) {
	ctx := context.Background()
	tmpBase := t.TempDir()
	origActorsDir := nodepath.ActorsDir
	nodepath.ActorsDir = filepath.Join(tmpBase, "actors")
	t.Cleanup(func() { nodepath.ActorsDir = origActorsDir })

	host := imageVolumeTestRegistry(t)
	ref := host + "/drain-app:v1"
	pushTestImage(t, ref, singleFileLayer(t, "bin/run", "#!/bin/sh\n"))

	store := newImageVolumeStore(t)
	fakeLayerDir := filepath.Join(tmpBase, "accelerator", "layer-0")
	if err := os.MkdirAll(filepath.Join(fakeLayerDir, "fs"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	draining := false
	layerClient := mock.New("riptide")
	layerClient.CanProvideFunc = func(_ context.Context, _ *imageaccel.PrepareRequest) (bool, error) {
		return !draining, nil
	}
	layerClient.PrepareImageFunc = func(_ context.Context, req *imageaccel.PrepareRequest) (*imageaccel.PrepareResult, error) {
		if draining {
			return nil, imageaccel.ErrNotSupported
		}
		return &imageaccel.PrepareResult{
			LayerDirs:    []string{fakeLayerDir},
			ImageDigest:  "sha256:2222222222222222222222222222222222222222222222222222222222222222",
			ProviderName: "riptide",
			Config: &v1.Config{
				Entrypoint: []string{"/bin/run"},
			},
		}, nil
	}

	atelet := &Atelet{
		imageCache:             store,
		imageAcceleratorClient: layerClient,
	}

	// Step 1: Prepare bundle while adapter is serving normally.
	actor1 := "actor-streamed"
	if _, err := prepareOCIDirectory(ctx, store, atelet.imageAcceleratorClient, nil, nil, actor1, "app", ref, nil, nil); err != nil {
		t.Fatalf("prepareOCIDirectory(streamed) failed: %v", err)
	}
	spec1, err := imagecache.ReadSpec(ateletpath.OCIBundlePath(actor1, "app"))
	if err != nil {
		t.Fatalf("ReadSpec(actor1): %v", err)
	}
	if !spec1.Streamed {
		t.Fatalf("expected spec1.Streamed = true")
	}

	// Step 2: Adapter enters lame-duck drain mode on SIGTERM (CanProvide returns false).
	draining = true

	// New or resuming actors immediately fall back to local imageCache with zero node-label coupling in atelet.
	actor2 := "actor-drained"
	if _, err := prepareOCIDirectory(ctx, store, atelet.imageAcceleratorClient, nil, nil, actor2, "app", ref, nil, nil); err != nil {
		t.Fatalf("prepareOCIDirectory(drained) failed: %v", err)
	}
	spec2, err := imagecache.ReadSpec(ateletpath.OCIBundlePath(actor2, "app"))
	if err != nil {
		t.Fatalf("ReadSpec(actor2): %v", err)
	}
	if spec2.Streamed {
		t.Errorf("expected spec2.Streamed = false when adapter is in lame-duck drain mode")
	}

	// Step 3: When the existing streamed actor checkpoints or terminates during drain,
	// releaseStreamedLayersForActor calls ReleaseImage on the adapter so its refCount drops to 0.
	workloadSpec := &ateletpb.WorkloadSpec{
		Containers: []*ateletpb.Container{{Name: "app", Image: ref}},
	}
	atelet.releaseStreamedLayersForActor(ctx, actor1, resources.ActorRef{Atespace: "default", Name: "a1"}, workloadSpec)
	if len(layerClient.ReleaseCalls) != 1 {
		t.Fatalf("expected 1 ReleaseImage call, got %d", len(layerClient.ReleaseCalls))
	}
	if layerClient.ReleaseCalls[0].ImageRef != ref {
		t.Errorf("ReleaseCalls[0].ImageRef = %q, want %q", layerClient.ReleaseCalls[0].ImageRef, ref)
	}
}
