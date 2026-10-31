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
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestReadWarmList(t *testing.T) {
	const digest = "sha256:abc123"
	tests := []struct {
		name   string
		list   *string // nil: no list file
		wantFG []string
		wantBG []string
	}{
		{name: "no list"},
		{name: "foreground only", list: ptr("/a\n\n/b\n"), wantFG: []string{"/a", "/b"}},
		{name: "split", list: ptr("/a\n#background\n/b\n/c\n"), wantFG: []string{"/a"}, wantBG: []string{"/b", "/c"}},
		{name: "background only", list: ptr("#background\n/b\n"), wantBG: []string{"/b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.list != nil {
				mustWrite(t, warmListPath(dir, digest), *tt.list)
			}
			fg, bg, err := readWarmList(dir, digest)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(fg, tt.wantFG) || !slices.Equal(bg, tt.wantBG) {
				t.Errorf("got fg %q bg %q, want fg %q bg %q", fg, bg, tt.wantFG, tt.wantBG)
			}
		})
	}
}

func TestReadPaths(t *testing.T) {
	tests := []struct {
		name       string
		paths      []string
		cancel     bool
		wantRead   int
		wantFailed int
		wantBytes  int64
		wantErr    error
	}{
		{name: "reads files", paths: []string{"/a.py", "/lib/b.so"}, wantRead: 2, wantBytes: 3 + 5},
		{name: "counts missing files", paths: []string{"/a.py", "/missing"}, wantRead: 1, wantFailed: 1, wantBytes: 3},
		{name: "stays inside the mount", paths: []string{"../../a.py"}, wantRead: 1, wantBytes: 3},
		{name: "stops when canceled", paths: []string{"/a.py"}, cancel: true, wantErr: context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mount := t.TempDir()
			mustWrite(t, filepath.Join(mount, "a.py"), "aaa")
			mustWrite(t, filepath.Join(mount, "lib", "b.so"), "bbbbb")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.cancel {
				cancel()
			}
			st, err := readPaths(ctx, mount, tt.paths, 4)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if st.Listed != len(tt.paths) || st.Read != tt.wantRead || st.Failed != tt.wantFailed || st.Bytes != tt.wantBytes {
				t.Errorf("stats = %+v, want listed %d read %d failed %d bytes %d",
					st, len(tt.paths), tt.wantRead, tt.wantFailed, tt.wantBytes)
			}
		})
	}
}

func ptr(s string) *string { return &s }

func mustWrite(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
