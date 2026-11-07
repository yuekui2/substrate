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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func TestBlobSeedReads(t *testing.T) {
	reg := httptest.NewServer(registry.New())
	defer reg.Close()
	repo, err := name.NewRepository(strings.TrimPrefix(reg.URL, "http://") + "/sandbox")
	if err != nil {
		t.Fatal(err)
	}
	bootstrap := layerTar(t, map[string]string{"image/image.boot": "bootstrap"})
	layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bootstrap)), nil
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
	// otherBlob is a valid gzip layer with another digest.
	other, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(layerTar(t, map[string]string{"image/image.boot": "other"}))), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	orc, err := other.Compressed()
	if err != nil {
		t.Fatal(err)
	}
	otherBlob, err := io.ReadAll(orc)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		// seed answers a seed request for path, given the registry's
		// response up.
		seed          func(w http.ResponseWriter, path string, up []byte)
		wantProxy     int64
		wantReadError bool
	}{
		{
			name:      "seed has the image",
			seed:      func(w http.ResponseWriter, _ string, up []byte) { _, _ = w.Write(up) },
			wantProxy: 0,
		},
		{
			name:      "seed miss",
			seed:      func(w http.ResponseWriter, _ string, _ []byte) { w.WriteHeader(http.StatusNotFound) },
			wantProxy: 1,
		},
		{
			name: "seed serves a corrupt manifest",
			seed: func(w http.ResponseWriter, path string, up []byte) {
				if strings.Contains(path, "/manifests/") {
					up = []byte(`{"schemaVersion":2,"layers":[]}`)
				}
				_, _ = w.Write(up)
			},
			wantProxy: 0,
		},
		{
			name: "seed serves a corrupt layer",
			seed: func(w http.ResponseWriter, path string, up []byte) {
				if strings.HasSuffix(path, layerDigest.String()) {
					up = otherBlob
				}
				_, _ = w.Write(up)
			},
			wantReadError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			seed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resp, err := http.Get(reg.URL + r.URL.Path)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadGateway)
					return
				}
				defer func() { _ = resp.Body.Close() }()
				up, err := io.ReadAll(resp.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadGateway)
					return
				}
				tc.seed(w, r.URL.Path, up)
			}))
			defer seed.Close()
			var proxyHits atomic.Int64
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

			snaps := &fakeNydusSnapshots{committed: make(map[string]bool)}
			d := newTestDriver(t, config{DragonflyProxyAddr: proxy.URL, BlobSeedURL: seed.URL}, snaps)

			digest, _, _, layerDigests, err := d.resolveImage(t.Context(), ref.String(), nil)
			if err != nil {
				t.Fatalf("resolveImage: %v", err)
			}
			if digest != imgDigest.String() || len(layerDigests) != 1 || layerDigests[0] != layerDigest.String() {
				t.Fatalf("resolveImage = (%s, %v), want (%s, [%s])", digest, layerDigests, imgDigest, layerDigest)
			}
			anns, err := d.registryLayerAnnotations(t.Context(), ref.String(), nil)
			if err != nil {
				t.Fatalf("registryLayerAnnotations: %v", err)
			}
			if len(anns) != 1 {
				t.Fatalf("got annotations for %d layers, want 1", len(anns))
			}
			rc, err := d.registryLayerReader(t.Context(), ref.String(), nil, 0, layerDigest.String())
			if err != nil {
				t.Fatalf("registryLayerReader: %v", err)
			}
			got, err := io.ReadAll(rc)
			_ = rc.Close()
			if tc.wantReadError {
				if err == nil {
					t.Fatal("reading a corrupt seed layer succeeded")
				}
				return
			}
			if err != nil {
				t.Fatalf("reading the layer: %v", err)
			}
			if !bytes.Equal(got, bootstrap) {
				t.Errorf("layer content: got %d bytes, want %d", len(got), len(bootstrap))
			}
			if got := proxyHits.Load(); got != tc.wantProxy {
				t.Errorf("proxy requests = %d, want %d", got, tc.wantProxy)
			}
		})
	}
}
