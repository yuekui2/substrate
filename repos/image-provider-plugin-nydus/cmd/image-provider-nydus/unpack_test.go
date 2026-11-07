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
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/containerd/containerd/api/types"
)

func TestExtractActiveMountDir(t *testing.T) {
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
			want:   "/snap/3/fs",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractActiveMountDir(tc.mounts); got != tc.want {
				t.Errorf("extractActiveMountDir = %q, want %q", got, tc.want)
			}
		})
	}
}

type tarEntry struct {
	name     string
	typeflag byte
	body     string
	linkname string
}

func entriesTar(t *testing.T, entries ...tarEntry) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Typeflag: e.typeflag, Mode: 0o644, Size: int64(len(e.body)), Linkname: e.linkname}
		if e.typeflag == tar.TypeDir {
			hdr.Mode = 0o755
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf
}

func TestUnpackLayerTarToDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "old.txt"), []byte("lower"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := entriesTar(t,
		tarEntry{name: "etc/", typeflag: tar.TypeDir},
		tarEntry{name: "etc/hosts", typeflag: tar.TypeReg, body: "127.0.0.1"},
		tarEntry{name: "etc/hosts.link", typeflag: tar.TypeLink, linkname: "etc/hosts"},
		tarEntry{name: "etc/hosts.sym", typeflag: tar.TypeSymlink, linkname: "hosts"},
		tarEntry{name: ".wh.old.txt", typeflag: tar.TypeReg},
		tarEntry{name: "etc/.wh..wh..opq", typeflag: tar.TypeReg},
	)
	if err := unpackLayerTarToDir(t.Context(), r, dir); err != nil {
		t.Fatalf("unpackLayerTarToDir: %v", err)
	}
	for path, want := range map[string]string{"etc/hosts": "127.0.0.1", "etc/hosts.link": "127.0.0.1", "etc/hosts.sym": "127.0.0.1"} {
		got, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil || string(got) != want {
			t.Errorf("%s = (%q, %v), want %q", path, got, err, want)
		}
	}
	for _, gone := range []string{"old.txt", ".wh.old.txt", "etc/.wh..wh..opq"} {
		if _, err := os.Lstat(filepath.Join(dir, gone)); !os.IsNotExist(err) {
			t.Errorf("%s exists after unpack (err=%v), want removed", gone, err)
		}
	}
}

func TestUnpackLayerTarToDirRejectsEscapes(t *testing.T) {
	for _, tc := range []struct {
		name  string
		entry tarEntry
	}{
		{"parent path", tarEntry{name: "../evil", typeflag: tar.TypeReg, body: "x"}},
		{"hardlink outside", tarEntry{name: "link", typeflag: tar.TypeLink, linkname: "../../etc/passwd"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := t.TempDir()
			dir := filepath.Join(parent, "root")
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := unpackLayerTarToDir(t.Context(), entriesTar(t, tc.entry), dir); err == nil {
				t.Error("unpackLayerTarToDir succeeded, want error")
			}
			if _, err := os.Lstat(filepath.Join(parent, "evil")); !os.IsNotExist(err) {
				t.Errorf("entry written outside the target dir (err=%v)", err)
			}
		})
	}
}
