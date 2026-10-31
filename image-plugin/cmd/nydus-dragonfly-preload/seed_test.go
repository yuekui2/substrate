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
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"go.ate.dev/substrate/image-plugin/pkg/plugin"
)

func TestBlobSeedReads(t *testing.T) {
	ctx := context.Background()
	reg := httptest.NewServer(registry.New())
	defer reg.Close()
	repo, err := name.NewRepository(strings.TrimPrefix(reg.URL, "http://") + "/sandbox")
	if err != nil {
		t.Fatal(err)
	}
	bootstrapTar := makeTestTar(t, map[string]string{"image/image.boot": "rafs-v6-erofs-superblock"})
	layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bootstrapTar)), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	img, err := mutate.AppendLayers(empty.Image, layer)
	if err != nil {
		t.Fatal(err)
	}
	imgDigest, err := img.Digest()
	if err != nil {
		t.Fatal(err)
	}
	ref := repo.Digest(imgDigest.String())
	if err := remote.Write(ref, img); err != nil {
		t.Fatal(err)
	}
	layerDigest, err := layer.Digest()
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name      string
		seedHas   bool
		wantSeed  int64
		wantProxy int64
	}{
		{name: "seed serves manifest and layer", seedHas: true, wantSeed: 2, wantProxy: 0},
		{name: "seed miss falls back", seedHas: false, wantSeed: 2, wantProxy: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var seedHits, proxyHits atomic.Int64
			seed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seedHits.Add(1)
				if !tc.seedHas {
					w.WriteHeader(http.StatusServiceUnavailable)
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
			defer seed.Close()
			proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				proxyHits.Add(1)
				resp, err := http.Get(r.Header.Get("X-Dragonfly-Registry") + r.URL.Path)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadGateway)
					return
				}
				defer func() { _ = resp.Body.Close() }()
				w.WriteHeader(resp.StatusCode)
				_, _ = io.Copy(w, resp.Body)
			}))
			defer proxy.Close()

			driver, err := NewNydusDragonfly(NydusDragonflyConfig{
				SkipKernelCheck:    true,
				DragonflyProxyAddr: proxy.URL,
				BlobSeedURL:        seed.URL,
				ServerOpts: []plugin.Option{
					plugin.WithWorkDir(filepath.Join(t.TempDir(), "work")),
					plugin.WithSnapshotsClient(&fakeNydusSnapshotsClient{committed: make(map[string]bool)}),
				},
			})
			if err != nil {
				t.Fatalf("NewNydusDragonfly: %v", err)
			}
			defer func() { _ = driver.Close() }()

			anns, err := driver.defaultLayerAnnotations(ctx, ref.String(), nil)
			if err != nil {
				t.Fatalf("defaultLayerAnnotations: %v", err)
			}
			if len(anns) != 1 {
				t.Fatalf("got %d layers, want 1", len(anns))
			}
			rc, err := driver.defaultLayerReader(ctx, ref.String(), nil, 0, layerDigest.String())
			if err != nil {
				t.Fatalf("defaultLayerReader: %v", err)
			}
			got, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				t.Fatalf("reading layer: %v", err)
			}
			if !bytes.Equal(got, bootstrapTar) {
				t.Fatalf("layer content differs: got %d bytes, want %d", len(got), len(bootstrapTar))
			}
			if seedHits.Load() != tc.wantSeed || proxyHits.Load() != tc.wantProxy {
				t.Fatalf("seed requests = %d, proxy requests = %d, want %d and %d",
					seedHits.Load(), proxyHits.Load(), tc.wantSeed, tc.wantProxy)
			}
		})
	}
}
