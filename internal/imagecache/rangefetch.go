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
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// rangeFetchThreshold is the minimum compressed layer blob size that
	// triggers parallel HTTP Range chunk fetching, matching containerd's
	// concurrent_layer_fetch_buffer concept (16 MiB). Smaller layers are
	// fetched in a single HTTP GET request.
	rangeFetchThreshold = 16 * 1024 * 1024

	// rangeChunkSize is the byte range size requested by each parallel chunk
	// worker (16 MiB).
	rangeChunkSize = 16 * 1024 * 1024

	// rangeLayerConcurrency is the maximum number of concurrent HTTP Range
	// requests per large layer.
	rangeLayerConcurrency = 16

	// rangeGlobalConcurrency bounds the total number of concurrent HTTP Range
	// chunk requests across all layers being pulled simultaneously on the node.
	rangeGlobalConcurrency = 24

	// rangeWindowChunks bounds how many 16 MiB chunks of a single layer may be
	// buffered or in flight ahead of the decompressor at once (96 * 16 MiB =
	// 1.5 GiB ceiling), allowing multi-hundred-megabyte layers to download at
	// full multi-Gbps line rate without stalling on single-layer decompression.
	rangeWindowChunks = 96

	// rangeChunkMaxRetries is the number of retry attempts for an individual
	// byte-range request before failing the layer stream.
	rangeChunkMaxRetries = 3

	// rangeProbeHeader marks internal Range requests issued by
	// parallelRangeTransport so they are never intercepted recursively.
	rangeProbeHeader = "X-Substrate-Range-Fetch"
)

var (
	// largeBlobSizes maps a layer blob digest ("sha256:<hex>") to its known
	// compressed byte size for blobs >= rangeFetchThreshold.
	largeBlobSizes sync.Map

	// globalRangeSem limits total concurrent HTTP Range chunk downloads across
	// all simultaneous layer pulls.
	globalRangeSem = make(chan struct{}, rangeGlobalConcurrency)
)

func registerLargeBlob(digest string, size int64) {
	if size >= rangeFetchThreshold && digest != "" {
		largeBlobSizes.Store(digest, size)
	}
}

func unregisterLargeBlob(digest string) {
	if digest != "" {
		largeBlobSizes.Delete(digest)
	}
}

// parallelRangeTransport wraps an underlying http.RoundTripper and upgrades
// large OCI layer blob GET requests (>= rangeFetchThreshold) into concurrent
// HTTP Range chunk downloads that stream into an ordered sliding window of
// pooled 256 KiB buffers without writing compressed blobs to disk.
//
// If a registry or storage backend ignores the Range header and responds with
// 200 OK (or rejects Range), parallelRangeTransport falls back transparently to
// a standard single-stream GET response.
type parallelRangeTransport struct {
	base http.RoundTripper
}

func newParallelRangeTransport(base http.RoundTripper) http.RoundTripper {
	return &parallelRangeTransport{base: base}
}

func (t *parallelRangeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet || req.Header.Get("Range") != "" || req.Header.Get(rangeProbeHeader) != "" {
		return t.base.RoundTrip(req)
	}

	totalSize, ok := lookupLargeBlobSize(req)
	if !ok || totalSize < rangeFetchThreshold {
		return t.base.RoundTrip(req)
	}

	firstEnd := int64(rangeChunkSize) - 1
	if firstEnd >= totalSize {
		return t.base.RoundTrip(req)
	}

	probeReq := req.Clone(req.Context())
	probeReq.Header.Set("Range", fmt.Sprintf("bytes=0-%d", firstEnd))
	probeReq.Header.Set(rangeProbeHeader, "1")

	resp, err := t.base.RoundTrip(probeReq)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		// Let go-containerregistry's http.Client follow the redirect (and run
		// SSRF validation); RoundTrip will be invoked again on the target URL
		// with req.Response pointing back to the original blob request.
		return resp, nil

	case http.StatusOK:
		// Registry ignored the Range header and is streaming the full blob.
		return resp, nil

	case http.StatusPartialContent:
		body := newParallelRangeBody(req, resp.Body, totalSize, t.base)
		resp.StatusCode = http.StatusOK
		resp.Status = "200 OK"
		resp.ContentLength = totalSize
		resp.Header = resp.Header.Clone()
		resp.Header.Set("Content-Length", strconv.FormatInt(totalSize, 10))
		resp.Header.Del("Content-Range")
		resp.Body = body
		return resp, nil

	default:
		// If the server rejected the Range header (e.g. 400, 416, 501) rather
		// than an auth challenge (401/403), retry once as a plain GET.
		if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
			_ = resp.Body.Close()
			return t.base.RoundTrip(req)
		}
		return resp, nil
	}
}

// lookupLargeBlobSize walks req and any prior redirect chain (via
// req.Response.Request) to check if the request originated from a registered
// large blob GET (/v2/.../blobs/sha256:<hex>).
func lookupLargeBlobSize(req *http.Request) (int64, bool) {
	for r := req; r != nil; {
		if r.URL != nil {
			if dgst := extractBlobDigest(r.URL.Path); dgst != "" {
				if v, ok := largeBlobSizes.Load(dgst); ok {
					return v.(int64), true
				}
			}
		}
		if r.Response == nil {
			break
		}
		r = r.Response.Request
	}
	return 0, false
}

func extractBlobDigest(path string) string {
	idx := strings.LastIndex(path, "/blobs/")
	if idx < 0 {
		return ""
	}
	dgst := path[idx+len("/blobs/"):]
	if strings.HasPrefix(dgst, "sha256:") && len(dgst) == len("sha256:")+64 {
		return dgst
	}
	return ""
}

type rangeChunkSlot struct {
	ch  chan streamChunk
	err error
}

// parallelRangeBody reassembles a large blob from concurrent HTTP Range
// requests into an ordered io.ReadCloser backed by pooled 256 KiB buffers.
type parallelRangeBody struct {
	ctx       context.Context
	cancel    context.CancelFunc
	tmplReq   *http.Request
	client    *http.Client
	totalSize int64
	numChunks int
	slots     []*rangeChunkSlot
	windowSem chan struct{}
	workerSem chan struct{}
	curIdx    int
	currBuf   *[]byte
	pending   []byte
	closeOnce sync.Once
	workersWG sync.WaitGroup
}

func newParallelRangeBody(tmplReq *http.Request, firstBody io.ReadCloser, totalSize int64, baseRT http.RoundTripper) *parallelRangeBody {
	ctx, cancel := context.WithCancel(tmplReq.Context())
	numChunks := int((totalSize + int64(rangeChunkSize) - 1) / int64(rangeChunkSize))
	subChunksPerSlot := (rangeChunkSize + streamChunkSize - 1) / streamChunkSize

	slots := make([]*rangeChunkSlot, numChunks)
	for i := range slots {
		slots[i] = &rangeChunkSlot{
			ch: make(chan streamChunk, subChunksPerSlot),
		}
	}

	b := &parallelRangeBody{
		ctx:       ctx,
		cancel:    cancel,
		tmplReq:   tmplReq,
		client:    &http.Client{Transport: baseRT, CheckRedirect: checkRangeRedirectSSRF},
		totalSize: totalSize,
		numChunks: numChunks,
		slots:     slots,
		windowSem: make(chan struct{}, rangeWindowChunks),
		workerSem: make(chan struct{}, rangeLayerConcurrency),
	}

	// Slot 0 is already open from the initial probe request.
	b.windowSem <- struct{}{}
	b.workersWG.Add(1)
	go func() {
		defer b.workersWG.Done()
		firstLen := min(int64(rangeChunkSize), totalSize)
		b.streamChunk0(firstBody, firstLen)
	}()

	// Dispatch remaining chunks [1 .. numChunks-1] within the sliding window.
	if numChunks > 1 {
		b.workersWG.Add(1)
		go func() {
			defer b.workersWG.Done()
			b.dispatchRemainingChunks()
		}()
	}

	return b
}

func (b *parallelRangeBody) streamChunk0(firstBody io.ReadCloser, expectedLen int64) {
	slot := b.slots[0]
	sent, err := b.pumpBodyToSlot(firstBody, expectedLen, slot)
	_ = firstBody.Close()
	if err != nil && sent < expectedLen {
		// Resume the remainder of chunk 0 via range retry if the initial stream
		// dropped mid-chunk.
		err = b.fetchChunkRange(0, sent, expectedLen-1, slot)
	}
	slot.err = err
	close(slot.ch)
}

func (b *parallelRangeBody) dispatchRemainingChunks() {
	for idx := 1; idx < b.numChunks; idx++ {
		select {
		case b.windowSem <- struct{}{}:
		case <-b.ctx.Done():
			b.failSlot(idx, b.ctx.Err())
			continue
		}

		select {
		case b.workerSem <- struct{}{}:
		case <-b.ctx.Done():
			<-b.windowSem
			b.failSlot(idx, b.ctx.Err())
			continue
		}

		chunkIdx := idx
		b.workersWG.Add(1)
		go func() {
			defer b.workersWG.Done()
			defer func() { <-b.workerSem }()

			start := int64(chunkIdx) * int64(rangeChunkSize)
			end := min(start+int64(rangeChunkSize)-1, b.totalSize-1)
			slot := b.slots[chunkIdx]

			// Acquire global range concurrency permit.
			select {
			case globalRangeSem <- struct{}{}:
			case <-b.ctx.Done():
				slot.err = b.ctx.Err()
				close(slot.ch)
				return
			}
			err := b.fetchChunkRange(chunkIdx, start, end, slot)
			<-globalRangeSem

			slot.err = err
			close(slot.ch)
		}()
	}
}

func (b *parallelRangeBody) failSlot(idx int, err error) {
	slot := b.slots[idx]
	slot.err = err
	close(slot.ch)
}

func (b *parallelRangeBody) fetchChunkRange(chunkIdx int, start, end int64, slot *rangeChunkSlot) error {
	offset := start
	var lastErr error
	for attempt := range rangeChunkMaxRetries {
		if b.ctx.Err() != nil {
			return b.ctx.Err()
		}
		if attempt > 0 {
			timer := time.NewTimer(time.Duration(attempt*100) * time.Millisecond)
			select {
			case <-timer.C:
			case <-b.ctx.Done():
				timer.Stop()
				return b.ctx.Err()
			}
		}

		expected := end - offset + 1
		req := b.tmplReq.Clone(b.ctx)
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, end))
		req.Header.Set(rangeProbeHeader, "1")

		resp, err := b.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("chunk %d range %d-%d: %w", chunkIdx, offset, end, err)
			continue
		}
		if resp.StatusCode != http.StatusPartialContent {
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("chunk %d range %d-%d: unexpected status %s", chunkIdx, offset, end, resp.Status)
			continue
		}

		n, err := b.pumpBodyToSlot(resp.Body, expected, slot)
		_ = resp.Body.Close()
		offset += n
		if err == nil && offset > end {
			return nil
		}
		if err != nil {
			lastErr = fmt.Errorf("chunk %d range %d-%d read: %w", chunkIdx, offset-n, end, err)
		} else {
			lastErr = fmt.Errorf("chunk %d range %d-%d short read (%d/%d)", chunkIdx, offset-n, end, n, expected)
		}
	}
	return lastErr
}

// pumpBodyToSlot reads up to expected bytes from body into full 256 KiB pooled
// buffers and sends them to slot.ch, returning the number of bytes sent.
// Using io.ReadFull is critical over HTTPS because crypto/tls returns at most
// one 16 KiB TLS record per Read call; without io.ReadFull, 64 16 KiB reads
// (1 MiB) would fill slot.ch (cap 64) and stall parallel chunk workers.
func (b *parallelRangeBody) pumpBodyToSlot(body io.Reader, expected int64, slot *rangeChunkSlot) (int64, error) {
	var total int64
	for total < expected {
		bufPtr := streamChunkPool.Get().(*[]byte)
		buf := *bufPtr
		maxRead := int64(len(buf))
		if rem := expected - total; rem < maxRead {
			maxRead = rem
		}
		n, err := io.ReadFull(body, buf[:maxRead])
		if n > 0 {
			select {
			case slot.ch <- streamChunk{buf: bufPtr, n: n}:
				total += int64(n)
			case <-b.ctx.Done():
				streamChunkPool.Put(bufPtr)
				return total, b.ctx.Err()
			}
		} else {
			streamChunkPool.Put(bufPtr)
		}
		if err != nil {
			if err == io.EOF {
				return total, io.ErrUnexpectedEOF
			}
			return total, err
		}
	}
	return total, nil
}

func (b *parallelRangeBody) Read(p []byte) (int, error) {
	for len(b.pending) == 0 {
		if b.currBuf != nil {
			streamChunkPool.Put(b.currBuf)
			b.currBuf = nil
		}
		if b.curIdx >= b.numChunks {
			return 0, io.EOF
		}
		slot := b.slots[b.curIdx]
		chunk, ok := <-slot.ch
		if !ok {
			if slot.err != nil {
				b.cancel()
				return 0, slot.err
			}
			// Finished consuming this 16 MiB chunk; release its sliding-window
			// permit so the next chunk in the blob can be fetched.
			select {
			case <-b.windowSem:
			default:
			}
			b.curIdx++
			continue
		}
		b.currBuf = chunk.buf
		b.pending = (*chunk.buf)[:chunk.n]
	}

	n := copy(p, b.pending)
	b.pending = b.pending[n:]
	return n, nil
}

func (b *parallelRangeBody) Close() error {
	b.closeOnce.Do(func() {
		b.cancel()
		if b.currBuf != nil {
			streamChunkPool.Put(b.currBuf)
			b.currBuf = nil
			b.pending = nil
		}
		// Drain window permits and return any buffered chunks across all slots
		// so in-flight workers can exit and pooled buffers are recycled.
		go func() {
			for _, slot := range b.slots {
				for chunk := range slot.ch {
					streamChunkPool.Put(chunk.buf)
				}
				select {
				case <-b.windowSem:
				default:
				}
			}
			b.workersWG.Wait()
		}()
	})
	return nil
}

// checkRangeRedirectSSRF mirrors go-containerregistry's checkRedirectSSRF so
// range chunk requests reject redirects from a public host to a private or
// link-local IP literal.
func checkRangeRedirectSSRF(req *http.Request, via []*http.Request) error {
	if len(via) == 0 || req.Response == nil {
		return nil
	}
	origHost := via[0].URL.Hostname()
	destHost := req.URL.Hostname()
	if destHost == origHost {
		return nil
	}
	if ip := net.ParseIP(destHost); ip != nil {
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() || ip.IsUnspecified() {
			return fmt.Errorf("SSRF protection: redirect from %q to private/link-local host %q denied", origHost, destHost)
		}
	}
	return nil
}
