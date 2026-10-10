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

// Command bench generates and pushes synthetic multi-scale OCI images
// (16 KB, 1 MB, 1 GB, 10 GB, and 50 GB unpacked) with realistic ~2.3x gzip
// compressibility using pre-compressed Z_SYNC_FLUSH DEFLATE blocks.
package main

import (
	"archive/tar"
	"bytes"
	"compress/flate"
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/klauspost/compress/gzip"
)

// syncFlushedBlock holds a 2 MiB uncompressed block (with ~2.3x compressibility)
// and its pre-compressed raw DEFLATE representation ending with a Z_SYNC_FLUSH
// marker (00 00 ff ff) so it can be concatenated at any byte-aligned position
// inside a single-member gzip stream.
type syncFlushedBlock struct {
	raw        []byte
	compressed []byte
}

func makeSyncFlushedBlocks(numBlocks int, blockSize int) ([]syncFlushedBlock, error) {
	blocks := make([]syncFlushedBlock, numBlocks)
	for i := 0; i < numBlocks; i++ {
		raw := make([]byte, blockSize)
		rng := rand.New(rand.NewSource(int64(1000 + i))) //nolint:gosec // Deterministic synthetic benchmark payload.
		offset := 0
		for offset < blockSize {
			chunk := 4096
			if offset+chunk > blockSize {
				chunk = blockSize - offset
			}
			// 44% random bytes, 56% structured pattern bytes -> ~43% compressed size (~2.3x ratio).
			randomPart := (chunk * 44) / 100
			rng.Read(raw[offset : offset+randomPart])
			for j := randomPart; j < chunk; j++ {
				raw[offset+j] = byte((i + j*7) & 0xff)
			}
			offset += chunk
		}

		// Compress into raw DEFLATE stream with a preceding Z_SYNC_FLUSH so the
		// block starts with a fresh byte-aligned block and ends with Z_SYNC_FLUSH.
		var warmBuf bytes.Buffer
		fw, err := flate.NewWriter(&warmBuf, flate.BestSpeed)
		if err != nil {
			return nil, err
		}
		if _, err := fw.Write([]byte("prime")); err != nil {
			return nil, err
		}
		if err := fw.Flush(); err != nil {
			return nil, err
		}
		warmBuf.Reset()

		if _, err := fw.Write(raw); err != nil {
			return nil, err
		}
		if err := fw.Flush(); err != nil {
			return nil, err
		}
		comp := append([]byte(nil), warmBuf.Bytes()...)
		blocks[i] = syncFlushedBlock{
			raw:        raw,
			compressed: comp,
		}
	}
	return blocks, nil
}

// fastGzipLayerBuilder writes a valid RFC 1952 single-member .tar.gz file while
// computing both the uncompressed SHA-256 (DiffID) and compressed SHA-256 (Digest).
type fastGzipLayerBuilder struct {
	outFile        *os.File
	compHash       hash.Hash
	uncompHash     hash.Hash
	crc            uint32
	uncompSize     int64
	compSize       int64
	compWriter     io.Writer
	scratchCompBuf bytes.Buffer
	flateWriter    *flate.Writer
}

func newFastGzipLayerBuilder(path string) (*fastGzipLayerBuilder, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	compHash := sha256.New()
	uncompHash := sha256.New()
	b := &fastGzipLayerBuilder{
		outFile:    f,
		compHash:   compHash,
		uncompHash: uncompHash,
	}
	b.compWriter = io.MultiWriter(f, compHash)

	// Standard 10-byte RFC 1952 GZIP header: ID1=0x1f, ID2=0x8b, CM=8 (deflate), FLG=0, MTIME=0, XFL=0, OS=255.
	gzipHeader := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff}
	n, err := b.compWriter.Write(gzipHeader)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	b.compSize += int64(n)

	fw, err := flate.NewWriter(&b.scratchCompBuf, flate.BestSpeed)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	b.flateWriter = fw
	return b, nil
}

// writeSmallUncompressed writes arbitrary tar bytes (such as 512-byte headers or small files),
// flushes them as a Z_SYNC_FLUSH DEFLATE block, and updates DiffID, CRC32, and Digest.
func (b *fastGzipLayerBuilder) writeSmallUncompressed(p []byte) error {
	if len(p) == 0 {
		return nil
	}
	b.uncompHash.Write(p)
	b.crc = crc32.Update(b.crc, crc32.IEEETable, p)
	b.uncompSize += int64(len(p))

	b.scratchCompBuf.Reset()
	b.flateWriter.Reset(&b.scratchCompBuf)
	if _, err := b.flateWriter.Write(p); err != nil {
		return err
	}
	if err := b.flateWriter.Flush(); err != nil {
		return err
	}
	compBytes := b.scratchCompBuf.Bytes()
	n, err := b.compWriter.Write(compBytes)
	b.compSize += int64(n)
	return err
}

// writePrecompressedBlock appends a pre-compressed Z_SYNC_FLUSH block directly.
func (b *fastGzipLayerBuilder) writePrecompressedBlock(blk syncFlushedBlock) error {
	b.uncompHash.Write(blk.raw)
	b.crc = crc32.Update(b.crc, crc32.IEEETable, blk.raw)
	b.uncompSize += int64(len(blk.raw))

	n, err := b.compWriter.Write(blk.compressed)
	b.compSize += int64(n)
	return err
}

func (b *fastGzipLayerBuilder) Close() (diffID v1.Hash, digest v1.Hash, compSize int64, uncompSize int64, err error) {
	// Write final empty DEFLATE block with BFINAL=1 (0x03, 0x00) + 8-byte GZIP trailer (CRC32 + ISIZE).
	var trailer [10]byte
	trailer[0] = 0x03
	trailer[1] = 0x00
	binary.LittleEndian.PutUint32(trailer[2:6], b.crc)
	binary.LittleEndian.PutUint32(trailer[6:10], uint32(b.uncompSize))
	n, err := b.compWriter.Write(trailer[:])
	if err != nil {
		_ = b.outFile.Close()
		return v1.Hash{}, v1.Hash{}, 0, 0, err
	}
	b.compSize += int64(n)
	if err := b.outFile.Close(); err != nil {
		return v1.Hash{}, v1.Hash{}, 0, 0, err
	}
	diffID = v1.Hash{
		Algorithm: "sha256",
		Hex:       fmt.Sprintf("%x", b.uncompHash.Sum(nil)),
	}
	digest = v1.Hash{
		Algorithm: "sha256",
		Hex:       fmt.Sprintf("%x", b.compHash.Sum(nil)),
	}
	return diffID, digest, b.compSize, b.uncompSize, nil
}

// precomputedLayer implements v1.Layer backed by an on-disk .tar.gz file with precomputed hashes.
type precomputedLayer struct {
	path       string
	diffID     v1.Hash
	digest     v1.Hash
	compSize   int64
	uncompSize int64
}

func (l *precomputedLayer) Digest() (v1.Hash, error) { return l.digest, nil }
func (l *precomputedLayer) DiffID() (v1.Hash, error) { return l.diffID, nil }
func (l *precomputedLayer) Compressed() (io.ReadCloser, error) {
	return os.Open(l.path)
}
func (l *precomputedLayer) Uncompressed() (io.ReadCloser, error) {
	f, err := os.Open(l.path)
	if err != nil {
		return nil, err
	}
	gr, err := gzip.NewReader(f)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return struct {
		io.Reader
		io.Closer
	}{Reader: gr, Closer: f}, nil
}
func (l *precomputedLayer) Size() (int64, error) { return l.compSize, nil }
func (l *precomputedLayer) MediaType() (types.MediaType, error) {
	return types.DockerLayer, nil
}

func encodeTarHeader(hdr *tar.Header) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	return append([]byte(nil), buf.Bytes()...), nil
}

// buildLayerFile generates a layer with the requested unpacked payload bytes,
// mixing small files (64 KiB) and large files (multiples of 2 MiB).
func buildLayerFile(outPath string, layerIdx int, targetUnpackedBytes int64, blocks []syncFlushedBlock) (*precomputedLayer, error) {
	builder, err := newFastGzipLayerBuilder(outPath)
	if err != nil {
		return nil, err
	}

	blockSize := int64(len(blocks[0].raw))
	now := time.Unix(1700000000, 0)

	// For layers smaller than one block, write small files directly via tar.Writer.
	if targetUnpackedBytes < blockSize {
		var rawTar bytes.Buffer
		tw := tar.NewWriter(&rawTar)
		fileSize := int64(4096)
		if targetUnpackedBytes >= 65536 {
			fileSize = 65536
		}
		numFiles := int(targetUnpackedBytes / fileSize)
		if numFiles < 1 {
			numFiles = 1
			fileSize = targetUnpackedBytes
		}
		rng := rand.New(rand.NewSource(int64(layerIdx*100 + 1))) //nolint:gosec // Deterministic synthetic benchmark payload.
		payload := make([]byte, fileSize)
		for fIdx := 0; fIdx < numFiles; fIdx++ {
			randomPart := (len(payload) * 44) / 100
			rng.Read(payload[:randomPart])
			for j := randomPart; j < len(payload); j++ {
				payload[j] = byte((fIdx + j*7) & 0xff)
			}
			entryName := fmt.Sprintf("layer-%02d/small/file-%05d.dat", layerIdx, fIdx)
			if err := tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeReg,
				Name:     entryName,
				Mode:     0o644,
				Size:     fileSize,
				ModTime:  now,
			}); err != nil {
				return nil, err
			}
			if _, err := tw.Write(payload); err != nil {
				return nil, err
			}
		}
		if err := tw.Close(); err != nil {
			return nil, err
		}
		if err := builder.writeSmallUncompressed(rawTar.Bytes()); err != nil {
			return nil, err
		}
		diffID, digest, compSize, uncompSize, err := builder.Close()
		if err != nil {
			return nil, err
		}
		return &precomputedLayer{
			path:       outPath,
			diffID:     diffID,
			digest:     digest,
			compSize:   compSize,
			uncompSize: uncompSize,
		}, nil
	}

	// Write a batch of 64 KiB small files (up to 32 MiB or 5% of layer) so the
	// parallel file unpacker exercises both small-file workers and large-file streaming.
	smallBudget := targetUnpackedBytes / 20
	if smallBudget > 32*1024*1024 {
		smallBudget = 32 * 1024 * 1024
	}
	smallBudget = (smallBudget / blockSize) * blockSize
	var writtenPayload int64

	if smallBudget >= blockSize && blockSize >= 64*1024 {
		const smallFileSize = 64 * 1024
		numSmall := int(smallBudget / smallFileSize)
		var smallBatch bytes.Buffer
		tw := tar.NewWriter(&smallBatch)
		for fIdx := 0; fIdx < numSmall; fIdx++ {
			dirIdx := fIdx / 64
			entryName := fmt.Sprintf("layer-%02d/pkg-%03d/mod-%05d.py", layerIdx, dirIdx, fIdx)
			if err := tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeReg,
				Name:     entryName,
				Mode:     0o644,
				Size:     smallFileSize,
				ModTime:  now,
			}); err != nil {
				return nil, err
			}
			blk := blocks[(layerIdx+fIdx)%len(blocks)]
			off := int64(fIdx*smallFileSize) % (blockSize - smallFileSize)
			if _, err := tw.Write(blk.raw[off : off+smallFileSize]); err != nil {
				return nil, err
			}
			writtenPayload += smallFileSize
			if smallBatch.Len() >= 4*1024*1024 {
				if err := builder.writeSmallUncompressed(smallBatch.Bytes()); err != nil {
					return nil, err
				}
				smallBatch.Reset()
			}
		}
		if smallBatch.Len() > 0 {
			if err := builder.writeSmallUncompressed(smallBatch.Bytes()); err != nil {
				return nil, err
			}
		}
	}

	// Fill remaining budget with medium (2 MiB) and large (32-64 MiB) files using pre-compressed blocks.
	remaining := targetUnpackedBytes - writtenPayload
	fileIdx := 0
	for remaining >= blockSize {
		blocksInFile := 1
		if targetUnpackedBytes >= 4*1024*1024*1024 && fileIdx%2 == 0 && remaining >= 32*blockSize {
			blocksInFile = 32
		} else if fileIdx%4 == 0 && remaining >= 16*blockSize {
			blocksInFile = 16
		} else if remaining < int64(blocksInFile)*blockSize {
			blocksInFile = int(remaining / blockSize)
		}
		fileSize := int64(blocksInFile) * blockSize
		dirIdx := fileIdx / 32
		entryName := fmt.Sprintf("layer-%02d/weights-%03d/shard-%05d.bin", layerIdx, dirIdx, fileIdx)
		hdrBytes, err := encodeTarHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     entryName,
			Mode:     0o644,
			Size:     fileSize,
			ModTime:  now,
		})
		if err != nil {
			return nil, err
		}
		if err := builder.writeSmallUncompressed(hdrBytes); err != nil {
			return nil, err
		}
		for bIdx := 0; bIdx < blocksInFile; bIdx++ {
			blk := blocks[(layerIdx*31+fileIdx*7+bIdx)%len(blocks)]
			if err := builder.writePrecompressedBlock(blk); err != nil {
				return nil, err
			}
		}
		remaining -= fileSize
		fileIdx++
	}

	// Write standard 1024-byte zero tar EOF trailer.
	var eofTrailer [1024]byte
	if err := builder.writeSmallUncompressed(eofTrailer[:]); err != nil {
		return nil, err
	}

	diffID, digest, compSize, uncompSize, err := builder.Close()
	if err != nil {
		return nil, err
	}
	return &precomputedLayer{
		path:       outPath,
		diffID:     diffID,
		digest:     digest,
		compSize:   compSize,
		uncompSize: uncompSize,
	}, nil
}

type imageSpec struct {
	tag         string
	layerSizes  []int64 // unpacked bytes per layer
	description string
}

func main() {
	repoPrefix := flag.String("repo", "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/bench-scale", "Target repository")
	workDir := flag.String("work-dir", "/tmp/bench-gen", "Temporary directory for layer files")
	onlyTag := flag.String("only", "", "If set, only build/push this tag (16kb, 1mb, 1gb, 10gb, 50gb)")
	accessToken := flag.String("access-token", "", "Optional OAuth2 bearer token for pushing to Artifact Registry")
	selfTest := flag.Bool("self-test", false, "Run local self-test of fastGzipLayerBuilder and exit")
	flag.Parse()

	if err := os.MkdirAll(*workDir, 0o755); err != nil {
		log.Fatalf("mkdir work-dir: %v", err)
	}

	log.Printf("Pre-computing 16 x 2 MiB sync-flushed DEFLATE blocks...")
	blocks, err := makeSyncFlushedBlocks(16, 2*1024*1024)
	if err != nil {
		log.Fatalf("makeSyncFlushedBlocks: %v", err)
	}

	if *selfTest {
		testPath := filepath.Join(*workDir, "selftest.tar.gz")
		defer func() { _ = os.Remove(testPath) }()
		layer, err := buildLayerFile(testPath, 0, 8*1024*1024, blocks)
		if err != nil {
			log.Fatalf("buildLayerFile: %v", err)
		}
		f, err := os.Open(testPath)
		if err != nil {
			log.Fatalf("open: %v", err)
		}
		defer func() { _ = f.Close() }()
		gr, err := gzip.NewReader(f)
		if err != nil {
			log.Fatalf("gzip.NewReader: %v", err)
		}
		tr := tar.NewReader(gr)
		count := 0
		var totalBytes int64
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("tar.Next: %v", err)
			}
			n, err := io.Copy(io.Discard, tr)
			if err != nil {
				log.Fatalf("io.Copy: %v", err)
			}
			if n != hdr.Size {
				log.Fatalf("size mismatch: got %d want %d", n, hdr.Size)
			}
			count++
			totalBytes += n
		}
		if err := gr.Close(); err != nil {
			log.Fatalf("gr.Close: %v", err)
		}
		log.Printf("SELF-TEST PASSED: entries=%d payloadBytes=%d compSize=%d uncompSize=%d diffID=%s digest=%s",
			count, totalBytes, layer.compSize, layer.uncompSize, layer.diffID, layer.digest)
		return
	}

	const (
		KiB = 1024
		MiB = 1024 * KiB
		GiB = 1024 * MiB
	)

	specs := []imageSpec{
		{
			tag:         "16kb",
			layerSizes:  []int64{16 * KiB},
			description: "16 KiB unpacked (1 layer, 4 x 4 KiB files)",
		},
		{
			tag:         "1mb",
			layerSizes:  []int64{1 * MiB},
			description: "1 MiB unpacked (1 layer, 16 x 64 KiB files)",
		},
		{
			tag:         "1gb",
			layerSizes:  []int64{512 * MiB, 256 * MiB, 192 * MiB, 64 * MiB},
			description: "1 GiB unpacked (4 layers: 512/256/192/64 MiB)",
		},
		{
			tag:         "10gb",
			layerSizes:  []int64{4 * GiB, 2560 * MiB, 2 * GiB, 1 * GiB, 512 * MiB},
			description: "10 GiB unpacked (5 layers: 4.0/2.5/2.0/1.0/0.5 GiB)",
		},
		{
			tag:         "50gb",
			layerSizes:  []int64{12 * GiB, 10 * GiB, 8 * GiB, 6 * GiB, 5 * GiB, 4 * GiB, 3 * GiB, 2 * GiB},
			description: "50 GiB unpacked (8 layers: 12/10/8/6/5/4/3/2 GiB)",
		},
	}

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.MaxConnsPerHost = 64
	tr.MaxIdleConnsPerHost = 64
	tr.WriteBufferSize = 256 * 1024
	tr.ReadBufferSize = 256 * 1024
	keychain := authn.NewMultiKeychain(authn.DefaultKeychain, google.Keychain)

	for _, sp := range specs {
		if *onlyTag != "" && *onlyTag != sp.tag {
			continue
		}
		start := time.Now()
		log.Printf("=== Building %s (%s) ===", sp.tag, sp.description)

		layers := make([]v1.Layer, len(sp.layerSizes))
		layerPaths := make([]string, len(sp.layerSizes))
		var wg sync.WaitGroup
		var buildErr error
		var errOnce sync.Once

		for i, sz := range sp.layerSizes {
			i, sz := i, sz
			layerPath := filepath.Join(*workDir, fmt.Sprintf("%s-layer-%d.tar.gz", sp.tag, i))
			layerPaths[i] = layerPath
			wg.Add(1)
			go func() {
				defer wg.Done()
				seedIdx := len(sp.tag)*100 + int(sp.tag[0])*10 + i
				l, err := buildLayerFile(layerPath, seedIdx, sz, blocks)
				if err != nil {
					errOnce.Do(func() { buildErr = err })
					return
				}
				layers[i] = l
				log.Printf("  [%s] layer %d built: uncomp=%.2f MiB comp=%.2f MiB digest=%s",
					sp.tag, i, float64(l.uncompSize)/(1024*1024), float64(l.compSize)/(1024*1024), l.digest)
			}()
		}
		wg.Wait()
		if buildErr != nil {
			log.Fatalf("Failed building layers for %s: %v", sp.tag, buildErr)
		}

		img := empty.Image
		img, err := mutate.AppendLayers(img, layers...)
		if err != nil {
			log.Fatalf("AppendLayers: %v", err)
		}
		refStr := fmt.Sprintf("%s:%s", *repoPrefix, sp.tag)
		ref, err := name.ParseReference(refStr)
		if err != nil {
			log.Fatalf("ParseReference(%s): %v", refStr, err)
		}

		pushStart := time.Now()
		log.Printf("  [%s] pushing to %s ...", sp.tag, refStr)
		remoteOpts := []remote.Option{
			remote.WithTransport(tr),
			remote.WithJobs(8),
		}
		if *accessToken != "" {
			remoteOpts = append(remoteOpts, remote.WithAuth(&authn.Bearer{Token: *accessToken}))
		} else {
			remoteOpts = append(remoteOpts, remote.WithAuthFromKeychain(keychain))
		}
		if err := remote.Write(ref, img, remoteOpts...); err != nil {
			log.Fatalf("remote.Write(%s): %v", refStr, err)
		}
		digest, err := img.Digest()
		if err != nil {
			log.Fatalf("img.Digest: %v", err)
		}
		var totalComp, totalUncomp int64
		for _, l := range layers {
			pl := l.(*precomputedLayer)
			totalComp += pl.compSize
			totalUncomp += pl.uncompSize
		}
		for _, p := range layerPaths {
			_ = os.Remove(p)
		}
		log.Printf("=== READY %s@%s | comp=%.2f MiB (%.2f GiB) | uncomp=%.2f MiB (%.2f GiB) | build=%s push=%s ===",
			refStr, digest,
			float64(totalComp)/(1024*1024), float64(totalComp)/(1024*1024*1024),
			float64(totalUncomp)/(1024*1024), float64(totalUncomp)/(1024*1024*1024),
			pushStart.Sub(start).Round(time.Millisecond),
			time.Since(pushStart).Round(time.Millisecond))
	}
}
