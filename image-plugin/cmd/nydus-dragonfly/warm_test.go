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
	"testing"
)

func TestWarmRootfs(t *testing.T) {
	const digest = "sha256:abc123"
	tests := []struct {
		name       string
		list       *string // nil: no list file
		cancel     bool
		wantListed int
		wantRead   int
		wantFailed int
		wantBytes  int64
		wantErr    error
	}{
		{name: "no list", wantErr: nil},
		{
			name:       "reads listed files",
			list:       ptr("/a.py\n/lib/b.so\n\n"),
			wantListed: 2, wantRead: 2, wantBytes: 3 + 5,
		},
		{
			name:       "counts missing files",
			list:       ptr("/a.py\n/missing\n"),
			wantListed: 2, wantRead: 1, wantFailed: 1, wantBytes: 3,
		},
		{
			name:       "stays inside the mount",
			list:       ptr("../../a.py\n"),
			wantListed: 1, wantRead: 1, wantBytes: 3,
		},
		{
			name:       "stops when canceled",
			list:       ptr("/a.py\n"),
			cancel:     true,
			wantListed: 1, wantErr: context.Canceled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mount := t.TempDir()
			mustWrite(t, filepath.Join(mount, "a.py"), "aaa")
			mustWrite(t, filepath.Join(mount, "lib", "b.so"), "bbbbb")
			listDir := t.TempDir()
			if tt.list != nil {
				mustWrite(t, warmListPath(listDir, digest), *tt.list)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.cancel {
				cancel()
			}
			st, err := warmRootfs(ctx, listDir, digest, mount, 4)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if st.Listed != tt.wantListed || st.Read != tt.wantRead || st.Failed != tt.wantFailed || st.Bytes != tt.wantBytes {
				t.Errorf("stats = %+v, want listed %d read %d failed %d bytes %d",
					st, tt.wantListed, tt.wantRead, tt.wantFailed, tt.wantBytes)
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
