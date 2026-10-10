// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildLayerFileValidGzipAndTar(t *testing.T) {
	blocks, err := makeSyncFlushedBlocks(4, 128*1024)
	if err != nil {
		t.Fatalf("makeSyncFlushedBlocks: %v", err)
	}

	tmpDir := t.TempDir()
	for _, targetBytes := range []int64{16 * 1024, 512 * 1024} {
		outPath := filepath.Join(tmpDir, fmt.Sprintf("layer-%d.tar.gz", targetBytes))
		layer, err := buildLayerFile(outPath, 1, targetBytes, blocks)
		if err != nil {
			t.Fatalf("buildLayerFile(%d): %v", targetBytes, err)
		}

		f, err := os.Open(outPath)
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		compHasher := sha256.New()
		gr, err := gzip.NewReader(io.TeeReader(f, compHasher))
		if err != nil {
			_ = f.Close()
			t.Fatalf("gzip.NewReader: %v", err)
		}
		uncompHasher := sha256.New()
		tr := tar.NewReader(io.TeeReader(gr, uncompHasher))
		var payloadBytes int64
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				_ = f.Close()
				t.Fatalf("tar.Next: %v", err)
			}
			n, err := io.Copy(io.Discard, tr)
			if err != nil {
				_ = f.Close()
				t.Fatalf("io.Copy: %v", err)
			}
			if n != hdr.Size {
				_ = f.Close()
				t.Fatalf("entry %s size = %d, want %d", hdr.Name, n, hdr.Size)
			}
			payloadBytes += n
		}
		// Drain trailing zero blocks through gzip reader to verify CRC32 + ISIZE.
		if _, err := io.Copy(io.Discard, io.TeeReader(gr, uncompHasher)); err != nil {
			_ = f.Close()
			t.Fatalf("drain gzip trailer: %v", err)
		}
		if err := gr.Close(); err != nil {
			_ = f.Close()
			t.Fatalf("gr.Close: %v", err)
		}
		_ = f.Close()

		if payloadBytes != targetBytes {
			t.Fatalf("payloadBytes = %d, want %d", payloadBytes, targetBytes)
		}
		gotDiffID := fmt.Sprintf("sha256:%x", uncompHasher.Sum(nil))
		if gotDiffID != layer.diffID.String() {
			t.Fatalf("diffID = %s, want %s", gotDiffID, layer.diffID)
		}
		gotDigest := fmt.Sprintf("sha256:%x", compHasher.Sum(nil))
		if gotDigest != layer.digest.String() {
			t.Fatalf("digest = %s, want %s", gotDigest, layer.digest)
		}
	}
}
