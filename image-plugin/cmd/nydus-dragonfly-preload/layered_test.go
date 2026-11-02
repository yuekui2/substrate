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
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func gz(t *testing.T, b []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	w := gzip.NewWriter(&out)
	_, _ = w.Write(b)
	_ = w.Close()
	return out.Bytes()
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

const testParentDigests = "sha256:aaaa\nsha256:bbbb"

func newTestLayered(t *testing.T, seedURL, sock string, m *fakeMounts) *layeredMounter {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "nydusd-config.json")
	tmpl := `{"type":"bootstrap","domain_id":"ate-shared","config":{"backend_type":"registry","backend_config":{"scheme":"https","mirrors":[{"host":"http://seed"}]},"cache_type":"fscache"}}`
	if err := os.WriteFile(cfgPath, []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}
	// Stands in for `nydus-image merge --bootstrap <out> <in>`.
	merge := filepath.Join(dir, "nydus-image")
	if err := os.WriteFile(merge, []byte("#!/bin/sh\ncp \"$4\" \"$3\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return newLayeredMounter(LayeredConfig{
		NydusImage:   merge,
		APISockGlob:  sock,
		DaemonConfig: cfgPath,
		DomainID:     "ate-shared",
		WorkDir:      filepath.Join(dir, "layered"),
	}, seedURL, nil, m.mount, m.unmount, func(string) error { return nil })
}

func TestLayeredMount(t *testing.T) {
	boot := []byte("top-layer-rafs-bootstrap")
	parentBoot := []byte("parent-chain-rafs-bootstrap")
	key := parentKey(strings.Split(testParentDigests, "\n"))
	for _, tc := range []struct {
		name          string
		pad           int
		corrupt       bool
		parentMissing bool
		wantErr       error
		wantErrText   string
	}{
		{name: "bootstrap in the tail read", pad: 16},
		{name: "bootstrap needs a second range read", pad: layerTailSize},
		{name: "digest mismatch", pad: 16, corrupt: true, wantErrText: "sha256"},
		{name: "no parent bootstrap falls back", pad: 16, parentMissing: true, wantErr: errNoParentBootstrap},
	} {
		t.Run(tc.name, func(t *testing.T) {
			blob := testBlob(t, boot, tc.pad, tc.corrupt)
			var parentGets int
			var mu sync.Mutex
			seed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/parentboots/sha256:"+key:
					mu.Lock()
					parentGets++
					mu.Unlock()
					if tc.parentMissing {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					_, _ = w.Write(gz(t, parentBoot))
				case strings.HasSuffix(r.URL.Path, "/blobs/sha256:top"):
					http.ServeContent(w, r, "blob", time.Time{}, bytes.NewReader(blob))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer seed.Close()
			nydusd, sock := startFakeNydusd(t)
			m := &fakeMounts{}
			lm := newTestLayered(t, seed.URL, sock, m)
			data := append(strings.Split(testParentDigests, "\n"), "sha256:top")

			// Two images on the same parent chain.
			var dirs [2][]string
			for i := range dirs {
				work := filepath.Join(t.TempDir(), fmt.Sprintf("img%d", i))
				d, err := lm.mount(context.Background(), fmt.Sprintf("example.com/repo:v%d", i), work, data)
				if tc.wantErr != nil || tc.wantErrText != "" {
					if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
						t.Fatalf("mount err = %v, want %v", err, tc.wantErr)
					}
					if tc.wantErrText != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErrText)) {
						t.Fatalf("mount err = %v, want it to mention %q", err, tc.wantErrText)
					}
					return
				}
				if err != nil {
					t.Fatalf("mount: %v", err)
				}
				dirs[i] = d
				got, err := os.ReadFile(filepath.Join(work, "layered-top", "image.boot"))
				if err != nil || !bytes.Equal(got, boot) {
					t.Fatalf("top image.boot = %q, %v; want %q", got, err, boot)
				}
				lm.release(work)
			}
			if dirs[0][0] != dirs[1][0] {
				t.Errorf("parent mounts %q and %q differ, want one shared mount", dirs[0][0], dirs[1][0])
			}
			if parentGets != 1 {
				t.Errorf("parent bootstrap fetched %d times, want 1", parentGets)
			}
			// Parent bind, then for each image a top bind, unmount, and two unbinds.
			nydusd.mu.Lock()
			defer nydusd.mu.Unlock()
			if len(nydusd.binds) != 3 {
				t.Fatalf("got %d binds, want 3", len(nydusd.binds))
			}
			cfg := nydusd.binds[1]["config"].(map[string]any)
			backend := cfg["backend_config"].(map[string]any)
			if backend["host"] != "example.com" || backend["repo"] != "repo" || cfg["metadata_path"] == "" {
				t.Errorf("top bind config = %v, want host example.com, repo repo, and a metadata path", cfg)
			}
			if nydusd.binds[1]["domain_id"] != "ate-shared" || nydusd.binds[1]["id"] != cfg["id"] {
				t.Errorf("top bind = %v, want domain ate-shared and matching ids", nydusd.binds[1])
			}
			if len(m.unmounted) != 2 || len(nydusd.calls) != 7 {
				t.Errorf("unmounts %v, nydusd calls %v; want 2 top unmounts and 3 binds + 4 unbinds", m.unmounted, nydusd.calls)
			}
			if pf, _ := cfg["prefetch_config"].(map[string]any); pf == nil || pf["bandwidth_rate"] == nil {
				t.Errorf("top bind config = %v, want prefetch_config.bandwidth_rate", cfg)
			}
		})
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
