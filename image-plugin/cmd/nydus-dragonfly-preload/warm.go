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
	"bufio"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// warmBackgroundMarker separates the files read before the mount is returned
// from the files read after it, in a warm list.
const warmBackgroundMarker = "#background"

// warmBackgroundWorkers bounds the parallel reads of the background part, so
// they leave nydusd threads for the restored guest's own reads.
const warmBackgroundWorkers = 8

// warmStats reports what readPaths read.
type warmStats struct {
	Listed int
	Read   int
	Failed int
	Bytes  int64
	Dur    time.Duration
}

// warmListPath is the file in listDir naming the files to warm for the image
// with manifest digest "sha256:<hex>": <listDir>/<hex>.txt, one absolute
// image path per line. Paths after a "#background" line are read after the
// mount is returned.
func warmListPath(listDir, imageDigest string) string {
	return filepath.Join(listDir, strings.TrimPrefix(imageDigest, "sha256:")+".txt")
}

// readWarmList returns the paths listed for imageDigest before and after the
// background marker. A missing list returns no paths.
func readWarmList(listDir, imageDigest string) (fg, bg []string, err error) {
	f, err := os.Open(warmListPath(listDir, imageDigest))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	dst := &fg
	for sc.Scan() {
		switch p := strings.TrimSpace(sc.Text()); p {
		case "":
		case warmBackgroundMarker:
			dst = &bg
		default:
			*dst = append(*dst, p)
		}
	}
	return fg, bg, sc.Err()
}

// readPaths reads every path under mountDir with the given number of
// workers, so that nydusd fetches their chunks in parallel. Files that can't
// be read are counted in Failed. It stops at ctx's deadline and returns ctx's
// error with the stats so far.
func readPaths(ctx context.Context, mountDir string, paths []string, workers int) (warmStats, error) {
	start := time.Now()
	var read, failed, bytes atomic.Int64
	work := make(chan string)
	var wg sync.WaitGroup
	for range max(workers, 1) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range work {
				n, err := readAll(filepath.Join(mountDir, filepath.Clean("/"+p)))
				bytes.Add(n)
				if err != nil {
					failed.Add(1)
				} else {
					read.Add(1)
				}
			}
		}()
	}
	var ctxErr error
	for _, p := range paths {
		if ctxErr = ctx.Err(); ctxErr != nil {
			break
		}
		select {
		case work <- p:
		case <-ctx.Done():
		}
	}
	close(work)
	wg.Wait()
	if ctxErr == nil {
		ctxErr = ctx.Err()
	}
	return warmStats{
		Listed: len(paths),
		Read:   int(read.Load()),
		Failed: int(failed.Load()),
		Bytes:  bytes.Load(),
		Dur:    time.Since(start),
	}, ctxErr
}

func readAll(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(io.Discard, f)
}
