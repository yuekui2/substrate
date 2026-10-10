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

package imagecache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/klauspost/compress/gzip"
)

func makeDeterministicBlob(size int) ([]byte, string) {
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = byte((i*31 + i/1024) & 0xff)
	}
	sum := sha256.Sum256(buf)
	return buf, "sha256:" + hex.EncodeToString(sum[:])
}

func TestParallelRangeTransport_ConcurrentRangeFetch(t *testing.T) {
	// 36 MiB blob -> 3 chunks (16 MiB, 16 MiB, 4 MiB).
	blobSize := 36 * 1024 * 1024
	blob, dgst := makeDeterministicBlob(blobSize)
	registerLargeBlob(dgst, int64(blobSize))
	t.Cleanup(func() { unregisterLargeBlob(dgst) })

	var rangeReqCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "" {
			rangeReqCount.Add(1)
		}
		http.ServeContent(w, r, "blob", time.Unix(0, 0), bytes.NewReader(blob))
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{
		Transport: newParallelRangeTransport(http.DefaultTransport),
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/v2/test/repo/blobs/"+dgst, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if resp.ContentLength != int64(blobSize) {
		t.Fatalf("ContentLength = %d, want %d", resp.ContentLength, blobSize)
	}

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, blob) {
		t.Fatalf("reassembled blob mismatch (len=%d, want=%d)", len(got), len(blob))
	}
	if gotReqs := rangeReqCount.Load(); gotReqs != 3 {
		t.Errorf("rangeReqCount = %d, want 3", gotReqs)
	}
}

func TestParallelRangeTransport_RedirectToStorage(t *testing.T) {
	blobSize := 34 * 1024 * 1024
	blob, dgst := makeDeterministicBlob(blobSize)
	registerLargeBlob(dgst, int64(blobSize))
	t.Cleanup(func() { unregisterLargeBlob(dgst) })

	var storageRangeReqs atomic.Int32
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v2/") {
			http.Redirect(w, r, srv.URL+"/storage/object-12345", http.StatusFound)
			return
		}
		if r.Header.Get("Range") != "" {
			storageRangeReqs.Add(1)
		}
		http.ServeContent(w, r, "blob", time.Unix(0, 0), bytes.NewReader(blob))
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{
		Transport: newParallelRangeTransport(http.DefaultTransport),
	}
	resp, err := client.Get(srv.URL + "/v2/test/repo/blobs/" + dgst)
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, blob) {
		t.Fatalf("redirected blob mismatch (len=%d, want=%d)", len(got), len(blob))
	}
	if gotReqs := storageRangeReqs.Load(); gotReqs != 3 {
		t.Errorf("storageRangeReqs = %d, want 3", gotReqs)
	}
}

func TestParallelRangeTransport_FallbackOn200OK(t *testing.T) {
	blobSize := 20 * 1024 * 1024
	blob, dgst := makeDeterministicBlob(blobSize)
	registerLargeBlob(dgst, int64(blobSize))
	t.Cleanup(func() { unregisterLargeBlob(dgst) })

	var reqCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount.Add(1)
		// Ignore Range header and return 200 OK with the full payload.
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(blob)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(blob)
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{
		Transport: newParallelRangeTransport(http.DefaultTransport),
	}
	resp, err := client.Get(srv.URL + "/v2/test/repo/blobs/" + dgst)
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, blob) {
		t.Fatalf("fallback blob mismatch")
	}
	if gotReqs := reqCount.Load(); gotReqs != 1 {
		t.Errorf("reqCount = %d, want 1", gotReqs)
	}
}

func TestParallelRangeTransport_ChunkRetryMidStream(t *testing.T) {
	blobSize := 34 * 1024 * 1024
	blob, dgst := makeDeterministicBlob(blobSize)
	registerLargeBlob(dgst, int64(blobSize))
	t.Cleanup(func() { unregisterLargeBlob(dgst) })

	var chunk1Failures atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rng := r.Header.Get("Range")
		// Simulate a mid-stream truncation on the first attempt of chunk 1.
		if strings.HasPrefix(rng, fmt.Sprintf("bytes=%d-", rangeChunkSize)) && chunk1Failures.Add(1) == 1 {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", rangeChunkSize, 2*rangeChunkSize-1, blobSize))
			w.Header().Set("Content-Length", fmt.Sprintf("%d", rangeChunkSize))
			w.WriteHeader(http.StatusPartialContent)
			// Write only 1 MiB of the promised 16 MiB, then close handler so client sees UnexpectedEOF.
			_, _ = w.Write(blob[rangeChunkSize : rangeChunkSize+1024*1024])
			return
		}
		http.ServeContent(w, r, "blob", time.Unix(0, 0), bytes.NewReader(blob))
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{
		Transport: newParallelRangeTransport(http.DefaultTransport),
	}
	resp, err := client.Get(srv.URL + "/v2/test/repo/blobs/" + dgst)
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll after mid-stream retry: %v", err)
	}
	if !bytes.Equal(got, blob) {
		t.Fatalf("reassembled blob after retry mismatch")
	}
	if chunk1Failures.Load() < 1 {
		t.Errorf("expected chunk 1 failure to be triggered")
	}
}

type fakeCompressedLayer struct {
	compressed []byte
	digest     v1.Hash
	diffID     v1.Hash
	mediaType  types.MediaType
	eofErr     error
}

func (f *fakeCompressedLayer) Digest() (v1.Hash, error)            { return f.digest, nil }
func (f *fakeCompressedLayer) DiffID() (v1.Hash, error)            { return f.diffID, nil }
func (f *fakeCompressedLayer) Size() (int64, error)                { return int64(len(f.compressed)), nil }
func (f *fakeCompressedLayer) MediaType() (types.MediaType, error) { return f.mediaType, nil }
func (f *fakeCompressedLayer) Uncompressed() (io.ReadCloser, error) {
	return nil, fmt.Errorf("unexpected Uncompressed() call")
}
func (f *fakeCompressedLayer) Compressed() (io.ReadCloser, error) {
	r := bytes.NewReader(f.compressed)
	return &eofErrReadCloser{r: r, eofErr: f.eofErr}, nil
}

type eofErrReadCloser struct {
	r      io.Reader
	eofErr error
}

func (e *eofErrReadCloser) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	if err == io.EOF && e.eofErr != nil {
		return n, e.eofErr
	}
	return n, err
}

func (e *eofErrReadCloser) Close() error { return nil }

func TestOpenLayerStream_GzipAndIgzipParity(t *testing.T) {
	rawTar := buildTar(t, []tarEntry{
		{name: "dir/", typeflag: '5', mode: 0o755},
		{name: "dir/hello.txt", typeflag: '0', mode: 0o644, body: strings.Repeat("substrate-fast-pull\n", 5000)},
	})
	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	if _, err := gw.Write(rawTar); err != nil {
		t.Fatalf("gzip.Write: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip.Close: %v", err)
	}

	layer := &fakeCompressedLayer{
		compressed: gzBuf.Bytes(),
		digest:     v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("a", 64)},
		diffID:     v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("b", 64)},
		mediaType:  types.OCILayer,
	}

	// 1. Test pure-Go klauspost/compress/gzip fallback (igzipPath = "").
	rcGo, err := openLayerStreamWithIgzip(context.Background(), layer, "")
	if err != nil {
		t.Fatalf("openLayerStreamWithIgzip(go): %v", err)
	}
	gotGo, err := io.ReadAll(rcGo)
	if err != nil {
		t.Fatalf("ReadAll(go): %v", err)
	}
	if err := rcGo.Close(); err != nil {
		t.Fatalf("Close(go): %v", err)
	}
	if !bytes.Equal(gotGo, rawTar) {
		t.Fatalf("go gzip output mismatch")
	}

	// 2. Test external decompressor path (igzip if installed, or system gzip as stand-in).
	extBin := igzipBinaryPath()
	if extBin == "" {
		extBin, _ = exec.LookPath("gzip")
	}
	if extBin != "" {
		rcExt, err := openLayerStreamWithIgzip(context.Background(), layer, extBin)
		if err != nil {
			t.Fatalf("openLayerStreamWithIgzip(%s): %v", extBin, err)
		}
		gotExt, err := io.ReadAll(rcExt)
		if err != nil {
			t.Fatalf("ReadAll(%s): %v", extBin, err)
		}
		if err := rcExt.Close(); err != nil {
			t.Fatalf("Close(%s): %v", extBin, err)
		}
		if !bytes.Equal(gotExt, rawTar) {
			t.Fatalf("external decompressor (%s) output mismatch", extBin)
		}

		// Verify that a compressed-stream digest verification error at EOF is
		// surfaced by Close() even when using the external decompressor.
		wantErr := fmt.Errorf("simulated compressed sha256 mismatch at EOF")
		badLayer := *layer
		badLayer.eofErr = wantErr
		rcBad, err := openLayerStreamWithIgzip(context.Background(), &badLayer, extBin)
		if err != nil {
			t.Fatalf("openLayerStreamWithIgzip(bad): %v", err)
		}
		_, readErr := io.ReadAll(rcBad)
		closeErr := rcBad.Close()
		if readErr == nil && closeErr == nil {
			t.Fatalf("expected compressed EOF verification error to surface in Read or Close, got nil")
		}
	}
}
