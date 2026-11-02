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
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
)

// testBlob builds a Nydus data blob tail: the zstd image.boot entry, then the
// ToC and its rafs.blob.toc tar header. pad goes between them, so a large pad
// puts the bootstrap outside the first tail read.
func testBlob(t *testing.T, boot []byte, pad int, corruptDigest bool) []byte {
	t.Helper()
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatal(err)
	}
	comp := enc.EncodeAll(boot, nil)
	_ = enc.Close()
	var b bytes.Buffer
	b.WriteString("blob-data")
	off := b.Len()
	b.Write(comp)
	b.Write(make([]byte, pad))
	entry := make([]byte, tocEntrySize)
	binary.LittleEndian.PutUint32(entry[0:4], 0x2)
	copy(entry[8:24], "image.boot")
	sum := sha256.Sum256(boot)
	if corruptDigest {
		sum[0] ^= 0xff
	}
	copy(entry[24:56], sum[:])
	binary.LittleEndian.PutUint64(entry[56:64], uint64(off))
	binary.LittleEndian.PutUint64(entry[64:72], uint64(len(comp)))
	binary.LittleEndian.PutUint64(entry[72:80], uint64(len(boot)))
	b.Write(entry)
	hdr := make([]byte, 512)
	copy(hdr, "rafs.blob.toc")
	copy(hdr[124:136], fmt.Sprintf("%011o\x00", tocEntrySize))
	b.Write(hdr)
	return b.Bytes()
}

// testLayerBoot is a layer bootstrap with the placeholder blob id in its
// device table and blob table.
func testLayerBoot(name string) []byte {
	return []byte("devtable:" + string(blobIDPlaceholder) + " blobtable:" + string(blobIDPlaceholder) + " " + name)
}

// fakeNydusd records the API calls made on a unix socket.
type fakeNydusd struct {
	mu    sync.Mutex
	calls []string
	binds []map[string]any
}

func startFakeNydusd(t *testing.T) (*fakeNydusd, string) {
	t.Helper()
	f := &fakeNydusd{}
	sock := filepath.Join(t.TempDir(), "api.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls = append(f.calls, r.Method+" "+r.URL.RawQuery)
		if r.Method == http.MethodPut {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			f.binds = append(f.binds, body)
		}
		w.WriteHeader(http.StatusNoContent)
	})}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	return f, sock
}

type fakeMounts struct {
	mu        sync.Mutex
	mounted   []string
	unmounted []string
}

func (m *fakeMounts) mount(_, target, _ string, _ uintptr, data string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mounted = append(m.mounted, target+" "+data)
	return nil
}

func (m *fakeMounts) unmount(target string, _ int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unmounted = append(m.unmounted, target)
	return nil
}

func newTestLayered(t *testing.T, seedURL, sock string, m *fakeMounts) *layeredMounter {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "nydusd-config.json")
	tmpl := `{"type":"bootstrap","domain_id":"ate-shared","config":{"backend_type":"registry","backend_config":{"scheme":"https","mirrors":[{"host":"http://seed"}]},"cache_type":"fscache"}}`
	if err := os.WriteFile(cfgPath, []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}
	return newLayeredMounter(LayeredConfig{
		APISockGlob:  sock,
		DaemonConfig: cfgPath,
		DomainID:     "ate-shared",
		WorkDir:      filepath.Join(dir, "layered"),
	}, seedURL, nil, m.mount, m.unmount, func(string) error { return nil })
}

func digestOf(c string) string { return "sha256:" + strings.Repeat(c, 64) }

func TestLayeredMount(t *testing.T) {
	base, env := digestOf("a"), digestOf("b")
	tops := []string{digestOf("c"), digestOf("d")}
	for _, tc := range []struct {
		name          string
		pad           int
		corrupt       bool
		noPlaceholder bool
		wantErrText   string
	}{
		{name: "bootstrap in the tail read", pad: 16},
		{name: "bootstrap needs a second range read", pad: layerTailSize},
		{name: "digest mismatch", pad: 16, corrupt: true, wantErrText: "sha256"},
		{name: "no placeholder blob id", pad: 16, noPlaceholder: true, wantErrText: "placeholder"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			boots := map[string][]byte{}
			blobs := map[string][]byte{}
			for _, d := range append([]string{base, env}, tops...) {
				boots[d] = testLayerBoot(d)
				if tc.noPlaceholder {
					boots[d] = []byte("bootstrap " + d)
				}
				blobs[d] = testBlob(t, boots[d], tc.pad, tc.corrupt)
			}
			var mu sync.Mutex
			gets := map[string]int{}
			seed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				d := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
				blob, ok := blobs[d]
				if !strings.HasPrefix(r.URL.Path, "/v2/repo/blobs/") || !ok {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				mu.Lock()
				gets[d]++
				mu.Unlock()
				http.ServeContent(w, r, "blob", time.Time{}, bytes.NewReader(blob))
			}))
			defer seed.Close()
			nydusd, sock := startFakeNydusd(t)
			m := &fakeMounts{}
			lm := newTestLayered(t, seed.URL, sock, m)

			// Two images share the base and env layers.
			var works [2]string
			var dirs [2][]string
			for i := range works {
				works[i] = filepath.Join(t.TempDir(), fmt.Sprintf("img%d", i))
				d, err := lm.mount(context.Background(), fmt.Sprintf("example.com/repo:v%d", i), works[i], []string{base, env, tops[i]})
				if tc.wantErrText != "" {
					if err == nil || !strings.Contains(err.Error(), tc.wantErrText) {
						t.Fatalf("mount err = %v, want it to mention %q", err, tc.wantErrText)
					}
					return
				}
				if err != nil {
					t.Fatalf("mount: %v", err)
				}
				dirs[i] = d
			}
			if len(dirs[0]) != 3 || !slices.Equal(dirs[0][:2], dirs[1][:2]) || dirs[0][2] == dirs[1][2] {
				t.Errorf("mount dirs %q and %q, want shared base and env mounts and separate tops", dirs[0], dirs[1])
			}
			for _, d := range []string{base, env} {
				if gets[d] == 0 || gets[d] > 2 {
					t.Errorf("layer %s fetched with %d range reads, want one layer fetch", d, gets[d])
				}
			}
			// The bootstrap nydusd reads has the layer digest as its blob id.
			bootPath := filepath.Join(filepath.Dir(dirs[0][2]), strings.TrimPrefix(tops[0], "sha256:"))
			got, err := os.ReadFile(bootPath)
			want := bytes.ReplaceAll(boots[tops[0]], blobIDPlaceholder, []byte(strings.TrimPrefix(tops[0], "sha256:")))
			if err != nil || !bytes.Equal(got, want) {
				t.Errorf("top bootstrap = %q, %v; want %q", got, err, want)
			}

			// Releasing the first image unmounts only its top layer.
			lm.release(works[0])
			if !slices.Equal(m.unmounted, []string{dirs[0][2]}) {
				t.Errorf("after the first release, unmounted %q, want only %q", m.unmounted, dirs[0][2])
			}
			lm.release(works[1])
			if len(m.unmounted) != 4 {
				t.Errorf("after both releases, unmounted %q, want all 4 layers", m.unmounted)
			}

			nydusd.mu.Lock()
			defer nydusd.mu.Unlock()
			if len(nydusd.binds) != 4 {
				t.Fatalf("got %d binds, want 4", len(nydusd.binds))
			}
			for _, b := range nydusd.binds {
				cfg := b["config"].(map[string]any)
				backend := cfg["backend_config"].(map[string]any)
				if backend["host"] != "example.com" || backend["repo"] != "repo" || cfg["metadata_path"] == "" {
					t.Errorf("bind config = %v, want host example.com, repo repo, and a metadata path", cfg)
				}
				if b["domain_id"] != "ate-shared" || b["id"] != cfg["id"] {
					t.Errorf("bind = %v, want domain ate-shared and matching ids", b)
				}
				if pf, _ := cfg["prefetch_config"].(map[string]any); pf == nil || pf["bandwidth_rate"] == nil {
					t.Errorf("bind config = %v, want prefetch_config.bandwidth_rate", cfg)
				}
			}
		})
	}
}

func TestTopmostUnique(t *testing.T) {
	got := topmostUnique([]string{"a", "b", "a", "c"})
	if !slices.Equal(got, []string{"b", "a", "c"}) {
		t.Errorf("topmostUnique = %v, want [b a c]", got)
	}
}

func TestLayeredDataDigests(t *testing.T) {
	got, err := layeredDataDigests([]string{"a", "b", "boot"}, []map[string]string{
		{LabelNydusBlob: "true"},
		{LabelNydusBlob: "true"},
		{LabelNydusBootstrap: "true"},
	})
	if err != nil || strings.Join(got, ",") != "a,b" {
		t.Errorf("layeredDataDigests = %v, %v; want [a b]", got, err)
	}
}
