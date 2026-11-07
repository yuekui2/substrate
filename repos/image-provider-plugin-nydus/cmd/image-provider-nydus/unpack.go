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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ai-on-gke/image-provider-plugin/plugin"
	"github.com/containerd/containerd/api/types"
)

// extractActiveMountDir returns the writable directory of an active snapshot
// from Snapshots.Prepare: the overlay upperdir, or the bind source.
func extractActiveMountDir(mounts []*types.Mount) string {
	for _, m := range mounts {
		if m == nil {
			continue
		}
		for _, opt := range m.Options {
			if dir, ok := strings.CutPrefix(opt, "upperdir="); ok && dir != "" {
				return dir
			}
		}
	}
	return plugin.ExtractMountDir(mounts)
}

const (
	tarWhiteoutPrefix   = ".wh."
	tarOpaqueMarkerName = ".wh..wh..opq"
)

// validateLayerTarPath cleans a tar entry path. It reports skip for the
// archive root and an error for paths that leave it.
func validateLayerTarPath(p string) (cleaned string, skip bool, err error) {
	cleaned = strings.TrimPrefix(filepath.Clean(p), "/")
	if p == "" || cleaned == "" || cleaned == "." {
		return "", true, nil
	}
	if !filepath.IsLocal(cleaned) {
		return "", false, fmt.Errorf("non-local tar entry path %q", p)
	}
	return cleaned, false, nil
}

// unpackLayerTarToDir extracts an uncompressed OCI layer tar stream into
// targetDir. It applies whiteouts against what is already in targetDir and
// drops opaque markers. It opens targetDir with os.OpenRoot, so no entry can
// write outside it.
func unpackLayerTarToDir(ctx context.Context, r io.Reader, targetDir string) error {
	root, err := os.OpenRoot(targetDir)
	if err != nil {
		return fmt.Errorf("opening root %q: %w", targetDir, err)
	}
	defer func() { _ = root.Close() }()

	dirModes := make(map[string]os.FileMode)
	tr := tar.NewReader(r)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar header: %w", err)
		}

		entryName, skip, err := validateLayerTarPath(hdr.Name)
		if err != nil {
			return err
		}
		if skip {
			continue
		}

		base := filepath.Base(entryName)
		if base == tarOpaqueMarkerName || strings.HasPrefix(base, tarWhiteoutPrefix+tarWhiteoutPrefix) {
			continue
		}
		if deleted, ok := strings.CutPrefix(base, tarWhiteoutPrefix); ok {
			target := filepath.Join(filepath.Dir(entryName), deleted)
			if err := root.RemoveAll(target); err != nil {
				return fmt.Errorf("applying whiteout %q: %w", entryName, err)
			}
			continue
		}

		if parent := filepath.Dir(entryName); parent != "." {
			if err := root.MkdirAll(parent, 0o755); err != nil {
				return fmt.Errorf("creating parent directory %q: %w", parent, err)
			}
		}

		mode := hdr.FileInfo().Mode().Perm()
		switch hdr.Typeflag {
		case tar.TypeReg:
			if err := removeExisting(root, entryName); err != nil {
				return err
			}
			f, err := root.OpenFile(entryName, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)
			if err != nil {
				return fmt.Errorf("creating file %q: %w", entryName, err)
			}
			_, copyErr := io.Copy(f, tr)
			closeErr := f.Close()
			if copyErr != nil {
				return fmt.Errorf("writing file %q: %w", entryName, copyErr)
			}
			if closeErr != nil {
				return fmt.Errorf("closing file %q: %w", entryName, closeErr)
			}

		case tar.TypeDir:
			if err := root.Mkdir(entryName, mode|0o700); err != nil && !errors.Is(err, os.ErrExist) {
				return fmt.Errorf("creating directory %q: %w", entryName, err)
			}
			dirModes[entryName] = mode

		case tar.TypeSymlink:
			if err := removeExisting(root, entryName); err != nil {
				return err
			}
			if err := root.Symlink(hdr.Linkname, entryName); err != nil {
				return fmt.Errorf("creating symlink %q -> %q: %w", entryName, hdr.Linkname, err)
			}

		case tar.TypeLink:
			linkTarget, linkSkip, err := validateLayerTarPath(hdr.Linkname)
			if err != nil {
				return err
			}
			if linkSkip {
				return fmt.Errorf("empty hardlink target for %q", entryName)
			}
			if err := removeExisting(root, entryName); err != nil {
				return err
			}
			if err := root.Link(linkTarget, entryName); err != nil {
				return fmt.Errorf("creating hardlink %q -> %q: %w", entryName, linkTarget, err)
			}
		}
	}

	// Restore directory modes last, deepest first, so read-only directories
	// do not block writing their entries.
	dirs := make([]string, 0, len(dirModes))
	for d := range dirModes {
		dirs = append(dirs, d)
	}
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, d := range dirs {
		if err := root.Chmod(d, dirModes[d]); err != nil {
			return fmt.Errorf("restoring mode on directory %q: %w", d, err)
		}
	}
	return nil
}

// removeExisting removes whatever is at entryName, so a later layer entry
// replaces it.
func removeExisting(root *os.Root, entryName string) error {
	if _, err := root.Lstat(entryName); err != nil {
		return nil
	}
	if err := root.RemoveAll(entryName); err != nil {
		return fmt.Errorf("removing existing path %q: %w", entryName, err)
	}
	return nil
}
