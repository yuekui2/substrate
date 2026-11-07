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

package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-on-gke/image-provider-plugin/client"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

const (
	// leaseMetadataFile holds an image's persistedLease in its work dir.
	leaseMetadataFile = "lease.json"

	// workDirVersionFile holds workDirLayoutVersion at the root of the work
	// dir.
	workDirVersionFile = "version"

	// workDirLayoutVersion is the layout of the per-image directories under
	// the work dir. Bump it when the layout changes, so a new plugin clears
	// directories that an older one wrote.
	workDirLayoutVersion = "1"
)

// persistedLease is what a restarted plugin needs to take over a mounted
// image.
type persistedLease struct {
	ImageRef     string     `json:"imageRef"`
	Digest       string     `json:"digest,omitempty"`
	Config       *v1.Config `json:"config,omitempty"`
	DiffIDs      []string   `json:"diffIDs,omitempty"`
	LayerDigests []string   `json:"layerDigests,omitempty"`
	SnapshotKeys []string   `json:"snapshotKeys,omitempty"`
	LayerDirs    []string   `json:"layerDirs,omitempty"`
}

func writeLeaseMetadata(dir string, meta *persistedLease) error {
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, leaseMetadataFile), b, 0o600)
}

func readLeaseMetadata(dir string) (*persistedLease, error) {
	b, err := os.ReadFile(filepath.Join(dir, leaseMetadataFile))
	if err != nil {
		return nil, err
	}
	var meta persistedLease
	if err := json.Unmarshal(b, &meta); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filepath.Join(dir, leaseMetadataFile), err)
	}
	return &meta, nil
}

// ReconcileImages replaces the driver's leases with active, the leases that
// surviving workloads hold. It restores the images of leases the driver does
// not know, for example after the plugin restarted, from their lease.json.
// It unmounts images that no lease uses, and removes work dirs that belong
// to no image. An image with a preparation or teardown in flight keeps its
// state, and the leases on it wait for the flight: they are dropped if the
// image is gone when it ends.
func (d *Driver) ReconcileImages(ctx context.Context, active []*client.ActiveLease) error {
	leaseRefs := make(map[string]string)
	for _, a := range active {
		if a != nil && a.LeaseID != "" && a.ImageRef != "" {
			leaseRefs[a.LeaseID] = a.ImageRef
		}
	}

	keep := make(map[string]bool)
	d.mu.Lock()
	for ref := range d.inflight {
		keep[d.imageWorkDir(ref)] = true
	}
	for id, ref := range d.leaseRefs {
		if _, busy := d.inflight[ref]; busy {
			leaseRefs[id] = ref
		}
	}
	restored := make(map[string]bool)
	for id, ref := range leaseRefs {
		if _, busy := d.inflight[ref]; busy || restored[ref] {
			continue
		}
		workDir := d.imageWorkDir(ref)
		lease := d.leases[ref]
		if lease == nil {
			meta, err := readLeaseMetadata(workDir)
			if err != nil {
				slog.WarnContext(ctx, "Cannot restore the lease of an image in use",
					slog.String("provider", d.name),
					slog.String("image", ref),
					slog.String("lease", id),
					slog.Any("err", err))
				delete(leaseRefs, id)
				continue
			}
			lease = &imageLease{
				imageRef:     ref,
				digest:       meta.Digest,
				config:       meta.Config,
				snapshotKeys: meta.SnapshotKeys,
				layers:       meta.LayerDirs,
				workDir:      workDir,
			}
			d.leases[ref] = lease
			if meta.Config != nil && len(meta.DiffIDs) > 0 {
				d.resolvedMeta[ref] = &resolvedImageMeta{
					digest:       meta.Digest,
					config:       meta.Config,
					diffIDs:      meta.DiffIDs,
					layerDigests: meta.LayerDigests,
				}
			}
		}
		restored[ref] = true
		keep[workDir] = true
		stopIdleTimer(lease)
	}
	d.leaseRefs = leaseRefs
	d.notifyDrainedLocked()
	for ref := range restored {
		slog.InfoContext(ctx, "Reconciled image leases",
			slog.String("provider", d.name),
			slog.String("image", ref),
			slog.Int("leases", d.leaseCountLocked(ref)))
	}

	type teardown struct {
		lease  *imageLease
		flight *inflightPrep
	}
	var orphans []teardown
	for ref, lease := range d.leases {
		if restored[ref] {
			continue
		}
		if _, busy := d.inflight[ref]; busy {
			keep[lease.workDir] = true
			continue
		}
		orphans = append(orphans, teardown{lease: lease, flight: d.beginLeaseTeardownLocked(lease)})
	}
	d.mu.Unlock()

	for _, o := range orphans {
		d.finishLeaseTeardown(ctx, o.lease, o.flight)
	}
	d.sweepOrphanWorkDirs(ctx, keep)
	return nil
}

// dropImagelessLeasesLocked drops the leases on ref when the driver has no
// image for it. A flight on ref calls it when it ends, since ReconcileImages
// keeps leases on an image in flight without checking it.
func (d *Driver) dropImagelessLeasesLocked(ref string) {
	if _, ok := d.leases[ref]; ok {
		return
	}
	for id, r := range d.leaseRefs {
		if r == ref {
			delete(d.leaseRefs, id)
		}
	}
}

// imageWorkDir returns the directory that holds ref's layer directories.
func (d *Driver) imageWorkDir(ref string) string {
	return filepath.Join(d.workDir, SanitizePathKey(ref))
}

// prepareWorkDir creates the work dir and records workDirLayoutVersion in
// it. When the recorded version is missing or different, an older plugin
// wrote the directories, so it removes them first. That plugin drained its
// leases before it exited, so no running workload still needs them.
func (d *Driver) prepareWorkDir(ctx context.Context) error {
	if err := os.MkdirAll(d.workDir, 0o755); err != nil {
		return fmt.Errorf("creating work dir %s: %w", d.workDir, err)
	}
	versionPath := filepath.Join(d.workDir, workDirVersionFile)
	got, err := os.ReadFile(versionPath)
	if err == nil && strings.TrimSpace(string(got)) == workDirLayoutVersion {
		return nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading work dir version %s: %w", versionPath, err)
	}
	slog.InfoContext(ctx, "Clearing a work dir from another layout version",
		slog.String("workDir", d.workDir),
		slog.String("found", strings.TrimSpace(string(got))),
		slog.String("want", workDirLayoutVersion))
	d.sweepOrphanWorkDirs(ctx, nil)
	if err := os.WriteFile(versionPath, []byte(workDirLayoutVersion+"\n"), 0o644); err != nil {
		return fmt.Errorf("writing work dir version %s: %w", versionPath, err)
	}
	return nil
}

// sweepOrphanWorkDirs removes every image directory under the work dir that
// is not in keep, with the snapshots its lease.json lists.
func (d *Driver) sweepOrphanWorkDirs(ctx context.Context, keep map[string]bool) {
	entries, err := os.ReadDir(d.workDir)
	if err != nil {
		slog.WarnContext(ctx, "Failed to list the work dir",
			slog.String("workDir", d.workDir),
			slog.Any("err", err))
		return
	}
	for _, entry := range entries {
		dir := filepath.Join(d.workDir, entry.Name())
		if !entry.IsDir() || keep[dir] {
			continue
		}
		mounted := &MountedLayers{}
		if meta, err := readLeaseMetadata(dir); err == nil {
			mounted.SnapshotKeys = meta.SnapshotKeys
		}
		d.teardownWorkDir(ctx, dir, mounted)
		slog.InfoContext(ctx, "Removed an orphaned image work dir",
			slog.String("provider", d.name),
			slog.String("workDir", dir))
	}
}
