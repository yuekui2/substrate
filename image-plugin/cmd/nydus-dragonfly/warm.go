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

// warmStats reports what warmRootfs read.
type warmStats struct {
	Listed int
	Read   int
	Failed int
	Bytes  int64
	Dur    time.Duration
}

// warmListPath is the file in listDir naming the files to warm for the image
// with manifest digest "sha256:<hex>": <listDir>/<hex>.txt, one absolute
// image path per line.
func warmListPath(listDir, imageDigest string) string {
	return filepath.Join(listDir, strings.TrimPrefix(imageDigest, "sha256:")+".txt")
}

// warmRootfs reads every file listed for imageDigest under mountDir with the
// given number of workers, so that nydusd fetches their chunks in parallel
// before the actor's restore reads them one after another. A missing list
// means nothing to warm. Files that can't be read are counted in Failed. It
// stops at ctx's deadline and returns ctx's error with the stats so far.
func warmRootfs(ctx context.Context, listDir, imageDigest, mountDir string, workers int) (warmStats, error) {
	start := time.Now()
	var st warmStats
	f, err := os.Open(warmListPath(listDir, imageDigest))
	if errors.Is(err, fs.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	var paths []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if p := strings.TrimSpace(sc.Text()); p != "" {
			paths = append(paths, p)
		}
	}
	_ = f.Close()
	if err := sc.Err(); err != nil {
		return st, err
	}
	st.Listed = len(paths)

	var read, failed atomic.Int64
	var bytes atomic.Int64
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
	st.Read, st.Failed, st.Bytes = int(read.Load()), int(failed.Load()), bytes.Load()
	st.Dur = time.Since(start)
	return st, ctxErr
}

func readAll(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(io.Discard, f)
}
