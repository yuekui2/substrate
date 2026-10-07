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
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	// whiteoutPrefix marks an OCI layer entry that deletes the same-named
	// path from lower layers.
	whiteoutPrefix = ".wh."
	// opaqueMarkerName marks its directory as opaque: lower-layer contents of
	// that directory are hidden entirely.
	opaqueMarkerName = ".wh..wh..opq"

	// unpackFileWorkers is the number of concurrent file-creation workers per
	// unpacking layer. Regular files up to maxAsyncFileSize are buffered in
	// memory from the sequential tar stream and written to root concurrently
	// so openat2/write/close syscalls on 100k+-file layers do not serialize
	// on the tar reader goroutine.
	unpackFileWorkers = 8
	// maxAsyncFileSize caps the size of a regular file buffered into the
	// async file-writer pool; larger files stream directly from tarReader.
	maxAsyncFileSize = 256 * 1024
)

var unpackFileBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, maxAsyncFileSize)
		return &b
	},
}

type fileWriteJob struct {
	name   string
	mode   os.FileMode
	bufPtr *[]byte
	size   int
}

// onlyWriter hides os.File's io.ReaderFrom implementation from io.CopyBuffer.
// Without this wrapper, io.CopyBuffer calls (*os.File).ReadFrom(tarReader),
// which fails copy_file_range/splice on a tar.Reader and falls back to
// io.Copy(onlyWriter, tarReader) with a nil buffer — allocating a fresh 32 KiB
// slice on every file instead of reusing the caller's copyBuf.
type onlyWriter struct {
	io.Writer
}

// parentDir returns the parent directory of an already-cleaned relative path
// without re-running filepath.Clean.
func parentDir(cleanName string) string {
	if i := strings.LastIndexByte(cleanName, '/'); i >= 0 {
		return cleanName[:i]
	}
	return "."
}

// whiteoutSet records per-layer metadata captured at unpack time that the
// (privileged) consumer needs at compose time: whiteout state materialized
// by FinalizeLayer, and the directories this layer only created implicitly.
// Paths are clean and relative to the layer's fs/ root.
type whiteoutSet struct {
	Version int `json:"version"`
	// Whiteouts are paths that must become 0:0 char devices in the lowerdir.
	Whiteouts []string `json:"whiteouts,omitempty"`
	// Opaques are directories that must carry trusted.overlay.opaque=y.
	Opaques []string `json:"opaques,omitempty"`
	// ImplicitDirs are directories the layer tar never declared but that
	// exist in the tree because a child entry (or a whiteout materialized at
	// finalize) needed a parent. Their root:root 0755 attrs are fabricated;
	// in the composed overlay the top-most layer containing a directory
	// supplies its metadata, so an implicit dir here would shadow the real
	// attrs a lower layer declared (e.g. /tmp's 1777). SetupBundleRootfs
	// repairs the merged view from these records (see resolveImplicitDirFixups).
	ImplicitDirs []string `json:"implicitDirs,omitempty"`
}

func readWhiteouts(layerDir string) (*whiteoutSet, error) {
	b, err := os.ReadFile(filepath.Join(layerDir, layerWhiteoutsFileName))
	if errors.Is(err, os.ErrNotExist) {
		return &whiteoutSet{Version: 1}, nil
	} else if err != nil {
		return nil, fmt.Errorf("while reading layer whiteouts: %w", err)
	}
	var wh whiteoutSet
	if err := json.Unmarshal(b, &wh); err != nil {
		return nil, fmt.Errorf("while decoding layer whiteouts: %w", err)
	}
	return &wh, nil
}

func validateTarName(name string) (cleaned string, skip bool, err error) {
	if name == "" {
		return "", true, nil
	}
	cleaned = filepath.Clean(name)
	if cleaned == "." {
		return "", true, nil
	}
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || cleaned == "." {
		return "", true, nil
	}
	if !filepath.IsLocal(cleaned) {
		return "", false, fmt.Errorf("not a local path: %q", name)
	}
	return cleaned, false, nil
}

func removeExistingEntry(root *os.Root, name string, existingDirs map[string]bool, dirModes map[string]os.FileMode) error {
	if err := root.RemoveAll(name); err != nil {
		return err
	}
	if existingDirs[name] {
		delete(existingDirs, name)
		delete(dirModes, name)
		prefix := name + "/"
		for k := range existingDirs {
			if strings.HasPrefix(k, prefix) {
				delete(existingDirs, k)
				delete(dirModes, k)
			}
		}
	}
	return nil
}

// unpackLayer extracts one uncompressed OCI layer tar into root. Whiteout
// entries (.wh.*) are not written to the tree; they are returned so the
// caller can persist them for later materialization by a privileged process.
//
// Unlike a flattened-image extract, cross-layer "later entry wins" semantics
// are overlayfs's job now; the handling here only needs to cope with
// duplicate entries within a single layer (real ko images repeat directory
// entries).
func unpackLayer(ctx context.Context, tarData io.Reader, root *os.Root) (*whiteoutSet, error) {
	wh := &whiteoutSet{Version: 1}

	// Directories are created owner-writable during extraction (so their children
	// can be written even when the image marks them read-only, e.g. ko ships
	// /ko-app as 0555) and their real modes are restored afterwards. This lets
	// atelet, running as plain root, unpack arbitrary actor images without
	// CAP_DAC_OVERRIDE. Keyed by name so a repeated dir entry's last mode wins.
	dirModes := map[string]os.FileMode{}

	// existingDirs tracks directories already created in root during this
	// layer unpack so repeated files in the same parent directory skip
	// redundant root.MkdirAll syscalls.
	existingDirs := map[string]bool{}
	// seenNonDirs tracks paths created as regular files, symlinks, or
	// hardlinks during this unpack so any later entry touching the same path
	// (or converting a file path into a directory) drains in-flight async
	// file writers first to preserve strict "later entry wins" ordering.
	seenNonDirs := map[string]bool{}

	var (
		jobs       = make(chan fileWriteJob, unpackFileWorkers*8)
		workersWG  sync.WaitGroup
		pendingWG  sync.WaitGroup
		errOnce    sync.Once
		workerErr  error
		setWorkErr = func(err error) {
			errOnce.Do(func() { workerErr = err })
		}
	)
	for range unpackFileWorkers {
		workersWG.Add(1)
		go func() {
			defer workersWG.Done()
			for job := range jobs {
				outFile, err := root.OpenFile(job.name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, job.mode)
				if err != nil {
					if job.bufPtr != nil {
						unpackFileBufPool.Put(job.bufPtr)
					}
					setWorkErr(fmt.Errorf("while creating file %q: %w", job.name, err))
					pendingWG.Done()
					continue
				}
				var writeErr error
				if job.size > 0 {
					_, writeErr = outFile.Write((*job.bufPtr)[:job.size])
					unpackFileBufPool.Put(job.bufPtr)
				}
				closeErr := outFile.Close()
				if writeErr != nil {
					setWorkErr(fmt.Errorf("while writing contents of %q from tar stream: %w", job.name, writeErr))
				} else if closeErr != nil {
					setWorkErr(fmt.Errorf("while closing file %q: %w", job.name, closeErr))
				}
				pendingWG.Done()
			}
		}()
	}
	defer func() {
		pendingWG.Wait()
		close(jobs)
		workersWG.Wait()
	}()

	flushWorkers := func() error {
		pendingWG.Wait()
		return workerErr
	}

	ensureDir := func(dir string) error {
		if dir == "." || existingDirs[dir] {
			return nil
		}
		for p := dir; p != "." && !existingDirs[p]; p = parentDir(p) {
			if seenNonDirs[p] {
				if err := flushWorkers(); err != nil {
					return err
				}
				break
			}
		}
		if err := root.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		for p := dir; p != "." && !existingDirs[p]; p = parentDir(p) {
			existingDirs[p] = true
		}
		return nil
	}

	// Ancestors an entry needed vs. directories the tar declared: the
	// difference is recorded as ImplicitDirs (attrs fabricated, see the
	// whiteoutSet field doc). Whiteout/opaque markers count too — their
	// parents are created by FinalizeLayer's MkdirAll with the same
	// fabricated attrs. Once an ancestor is in declared or implicit, all of
	// its own ancestors have already been marked.
	declared := map[string]bool{}
	implicit := map[string]bool{}
	markAncestors := func(parent string) {
		for p := parent; p != "."; p = parentDir(p) {
			if declared[p] || implicit[p] {
				break
			}
			implicit[p] = true
		}
	}

	copyBuf := make([]byte, 256*1024)
	tarReader := tar.NewReader(tarData)
	for {
		hdr, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("in tarReader.Next: %w", err)
		}

		name, skip, err := validateTarName(hdr.Name)
		if err != nil {
			return nil, fmt.Errorf("invalid tar entry: %w", err)
		}
		if skip {
			continue
		}

		parent := parentDir(name)
		base := name
		if parent != "." {
			base = name[len(parent)+1:]
		}

		if base == opaqueMarkerName {
			if parent != "." {
				wh.Opaques = append(wh.Opaques, parent)
			}
			markAncestors(parent)
			continue
		} else if strings.HasPrefix(base, whiteoutPrefix+whiteoutPrefix) {
			// AUFS bookkeeping entries (.wh..wh.plnk, .wh..wh.aufs, ...);
			// nothing to represent in an overlayfs lowerdir.
			continue
		} else if deleted, ok := strings.CutPrefix(base, whiteoutPrefix); ok {
			wh.Whiteouts = append(wh.Whiteouts, filepath.Join(parent, deleted))
			markAncestors(parent)
			continue
		}

		mode := hdr.FileInfo().Mode().Perm()

		// A layer tar routinely omits entries for parent directories that
		// exist in lower layers (e.g. just "etc/nsswitch.conf", with "etc/"
		// declared only in the base layer). Unpacking per layer, those
		// parents must be created here; overlayfs merges them with the
		// lower layers' directories at compose time.
		markAncestors(parent)
		if parent != "." {
			if err := ensureDir(parent); err != nil {
				return nil, fmt.Errorf("while creating parent directories for %q: %w", name, err)
			}
		}

		switch hdr.Typeflag {
		case tar.TypeReg: // Regular file
			if !seenNonDirs[name] && !existingDirs[name] && hdr.Size >= 0 && hdr.Size <= maxAsyncFileSize {
				var bufPtr *[]byte
				if hdr.Size > 0 {
					bufPtr = unpackFileBufPool.Get().(*[]byte)
					if _, err := io.ReadFull(tarReader, (*bufPtr)[:hdr.Size]); err != nil {
						unpackFileBufPool.Put(bufPtr)
						_ = flushWorkers()
						return nil, fmt.Errorf("while writing contents of %q from tar stream: %w", name, err)
					}
				}
				seenNonDirs[name] = true
				pendingWG.Add(1)
				jobs <- fileWriteJob{
					name:   name,
					mode:   mode,
					bufPtr: bufPtr,
					size:   int(hdr.Size),
				}
				continue
			}

			if err := flushWorkers(); err != nil {
				return nil, err
			}
			seenNonDirs[name] = true

			// Optimistically create the file with O_EXCL to avoid a separate
			// Lstat syscall on every new file. "Later entry wins" within the
			// layer: if an entry already exists at name (O_EXCL returns
			// ErrExist even for a dangling symlink), remove it first so we
			// never write through a symlink, truncate a shared hardlink inode,
			// or fail on an existing directory.
			outFile, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if errors.Is(err, os.ErrExist) {
				if err := removeExistingEntry(root, name, existingDirs, dirModes); err != nil {
					return nil, fmt.Errorf("while replacing existing path at %q before regular file: %w", name, err)
				}
				outFile, err = root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			}
			if err != nil {
				return nil, fmt.Errorf("while creating file %q: %w", name, err)
			}

			var copyErr error
			if hdr.Size > 0 {
				_, copyErr = io.CopyBuffer(onlyWriter{outFile}, tarReader, copyBuf)
			}
			closeErr := outFile.Close()

			if copyErr != nil {
				return nil, fmt.Errorf("while writing contents of %q from tar stream: %w", name, copyErr)
			}
			if closeErr != nil {
				return nil, fmt.Errorf("while closing file %q: %w", name, closeErr)
			}

		case tar.TypeDir:
			if seenNonDirs[name] {
				if err := flushWorkers(); err != nil {
					return nil, err
				}
				delete(seenNonDirs, name)
			}
			// Create owner-writable so children can be written even when the image
			// marks the dir read-only; the real mode is restored after extraction
			// (see dirModes / the restore pass below).
			if !existingDirs[name] {
				err := root.Mkdir(name, mode|0o700)
				if errors.Is(err, os.ErrExist) {
					if fi, lerr := root.Lstat(name); lerr == nil && !fi.IsDir() {
						if rerr := removeExistingEntry(root, name, existingDirs, dirModes); rerr != nil {
							return nil, fmt.Errorf("while replacing existing non-directory at %q before directory: %w", name, rerr)
						}
						err = root.Mkdir(name, mode|0o700)
					} else {
						err = nil
					}
				}
				if err != nil {
					return nil, fmt.Errorf("while creating directory=%q, mode=%v: %w", name, mode, err)
				}
				existingDirs[name] = true
			}
			dirModes[name] = mode
			declared[name] = true
			delete(implicit, name)

		case tar.TypeSymlink:
			if seenNonDirs[name] || existingDirs[name] {
				if err := flushWorkers(); err != nil {
					return nil, err
				}
			}
			seenNonDirs[name] = true
			// A layer may re-define the same path (e.g. declare /var/run as a dir
			// then re-declare it as a symlink). Standard tar-extract semantics are
			// "later entry wins": optimistically create the symlink first, and
			// replace any existing entry on ErrExist.
			if err := root.Symlink(hdr.Linkname, name); errors.Is(err, os.ErrExist) {
				if existing, lerr := root.Lstat(name); lerr == nil && existing.Mode()&os.ModeSymlink != 0 {
					if cur, rerr := root.Readlink(name); rerr == nil && cur == hdr.Linkname {
						continue
					}
				}
				if err := removeExistingEntry(root, name, existingDirs, dirModes); err != nil {
					return nil, fmt.Errorf("while replacing existing path at %q before symlink: %w", name, err)
				}
				if err := root.Symlink(hdr.Linkname, name); err != nil {
					return nil, fmt.Errorf("while creating symlink src=%q target=%q: %w", name, hdr.Linkname, err)
				}
			} else if err != nil {
				return nil, fmt.Errorf("while creating symlink src=%q target=%q: %w", name, hdr.Linkname, err)
			}

		case tar.TypeLink:
			linkname, linkSkip, err := validateTarName(hdr.Linkname)
			if err != nil {
				return nil, fmt.Errorf("invalid hardlink target for %q: %w", name, err)
			}
			if linkSkip {
				return nil, fmt.Errorf("invalid hardlink target for %q: empty", name)
			}
			if err := flushWorkers(); err != nil {
				return nil, err
			}
			seenNonDirs[name] = true
			// Same "later entry wins" handling as TypeSymlink: replace existing entry on ErrExist.
			if err := root.Link(linkname, name); errors.Is(err, os.ErrExist) {
				if err := removeExistingEntry(root, name, existingDirs, dirModes); err != nil {
					return nil, fmt.Errorf("while replacing existing path at %q before hardlink: %w", name, err)
				}
				if err := root.Link(linkname, name); err != nil {
					return nil, fmt.Errorf("while creating hardlink src=%q target=%q: %w", name, linkname, err)
				}
			} else if err != nil {
				return nil, fmt.Errorf("while creating hardlink src=%q target=%q: %w", name, linkname, err)
			}

		default:
			tfStr := string([]byte{hdr.Typeflag})
			slog.ErrorContext(ctx, "Unhandled tar entry typeflag", slog.String("typeflag", tfStr), slog.Any("hdr", hdr))
			return nil, fmt.Errorf("unhandled tar entry typeflag %q", tfStr)
		}
	}

	if err := flushWorkers(); err != nil {
		return nil, err
	}

	// Restore the image's intended directory modes now that every child exists.
	// Deepest paths first: a child's path is always longer than its parent's, so
	// length-descending order guarantees a directory is restored before any of its
	// ancestors — restoring a parent to a non-traversable mode then can't block
	// restoring its children.
	dirs := make([]string, 0, len(dirModes))
	for name := range dirModes {
		dirs = append(dirs, name)
	}
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, name := range dirs {
		if err := root.Chmod(name, dirModes[name]); err != nil {
			return nil, fmt.Errorf("while restoring mode %v on directory %q: %w", dirModes[name], name, err)
		}
	}

	// Keep only implicit candidates that survive in the tree as directories
	// ("later entry wins" may have replaced one with a file or symlink), plus
	// the ones FinalizeLayer will create for whiteout/opaque materialization
	// (they may not exist yet). Sorted for deterministic metadata.
	finalizeDirs := map[string]bool{}
	for _, w := range wh.Whiteouts {
		for p := parentDir(w); p != "."; p = parentDir(p) {
			finalizeDirs[p] = true
		}
	}
	for _, o := range wh.Opaques {
		for p := o; p != "."; p = parentDir(p) {
			finalizeDirs[p] = true
		}
	}
	for p := range implicit {
		if declared[p] {
			continue
		}
		if existingDirs[p] {
			wh.ImplicitDirs = append(wh.ImplicitDirs, p)
		} else if fi, err := root.Lstat(p); err == nil && fi.IsDir() {
			wh.ImplicitDirs = append(wh.ImplicitDirs, p)
		} else if finalizeDirs[p] {
			wh.ImplicitDirs = append(wh.ImplicitDirs, p)
		}
	}
	sort.Strings(wh.ImplicitDirs)

	return wh, nil
}
