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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/ai-on-gke/image-provider-plugin/client"
)

// listableMaxInterval caps the backoff between listability checks.
const listableMaxInterval = 50 * time.Millisecond

// stoppingLocked reports whether the driver takes no new leases.
func (d *Driver) stoppingLocked() bool {
	return d.closed
}

// PrepareImage mounts the image's layers and takes a lease on them.
// Concurrent calls for one image share a single preparation. A mounted image
// is reused. It returns an error wrapping client.ErrNotSupported when the
// snapshotter declines the image, declined it within the decline TTL, or
// the driver is shutting down.
func (d *Driver) PrepareImage(ctx context.Context, req *client.PrepareRequest) (*client.PrepareResult, error) {
	if req == nil || req.ImageRef == "" {
		return nil, errors.New("image reference is required")
	}
	ref := req.ImageRef

	d.mu.Lock()
	for {
		if d.stoppingLocked() {
			d.mu.Unlock()
			return nil, fmt.Errorf("plugin is shutting down: %w", client.ErrNotSupported)
		}
		if flight, ok := d.inflight[ref]; ok {
			d.mu.Unlock()
			select {
			case <-flight.done:
				if flight.err != nil {
					return nil, flight.err
				}
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			d.mu.Lock()
			continue
		}
		if lease, ok := d.leases[ref]; ok && !lease.evicted && lease.config != nil {
			if layersExist(lease.layers) {
				res := d.acquireLeaseLocked(lease)
				hook := d.warmHitHook
				d.mu.Unlock()
				if hook != nil {
					hook(ctx, CanonicalImageRef(ref, res.ImageDigest), req)
				}
				return res, nil
			}
			stopIdleTimer(lease)
			slog.WarnContext(ctx, "Layers of a leased image are missing; preparing it again",
				slog.String("provider", d.name),
				slog.String("image", ref))
		}
		if d.isDeclinedLocked(ref, time.Now()) {
			d.mu.Unlock()
			return nil, fmt.Errorf("recently declined by the snapshotter: %w", client.ErrNotSupported)
		}
		break
	}
	flight := &inflightPrep{done: make(chan struct{})}
	d.inflight[ref] = flight
	d.mu.Unlock()

	res, err := d.prepareImageCold(ctx, req)

	d.mu.Lock()
	if errors.Is(err, client.ErrNotSupported) && d.declineTTL > 0 {
		d.declined[ref] = time.Now().Add(d.declineTTL)
	}
	flight.err = err
	delete(d.inflight, ref)
	close(flight.done)
	d.mu.Unlock()
	return res, err
}

// acquireLeaseLocked takes one more lease on a mounted image.
func (d *Driver) acquireLeaseLocked(lease *imageLease) *client.PrepareResult {
	stopIdleTimer(lease)
	lease.refCount++
	cfg := *lease.config
	return &client.PrepareResult{
		ImageDigest:  lease.digest,
		Config:       &cfg,
		LayerDirs:    slices.Clone(lease.layers),
		ProviderName: d.name,
	}
}

func stopIdleTimer(lease *imageLease) {
	if lease.idleTimer != nil {
		lease.idleTimer.Stop()
		lease.idleTimer = nil
	}
}

// layersExist reports whether every layer directory, and the view its fs
// link points to, still exists.
func layersExist(layers []string) bool {
	if len(layers) == 0 {
		return false
	}
	for _, l := range layers {
		if _, err := os.Stat(filepath.Join(l, "fs")); err != nil {
			return false
		}
	}
	return true
}

func (d *Driver) prepareImageCold(ctx context.Context, req *client.PrepareRequest) (*client.PrepareResult, error) {
	start := time.Now()
	ref := req.ImageRef
	snapClient, err := d.getClient()
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
	meta := d.resolvedMeta[ref]
	d.mu.Unlock()
	if meta == nil {
		digest, cfg, diffIDs, layerDigests, err := d.imageResolver(ctx, ref, req.AuthConfig)
		if err != nil {
			return nil, fmt.Errorf("resolving image %s: %w", ref, err)
		}
		if cfg == nil {
			return nil, fmt.Errorf("resolving image %s: no image config", ref)
		}
		meta = &resolvedImageMeta{digest: digest, config: cfg, diffIDs: diffIDs, layerDigests: layerDigests}
	}
	if len(meta.diffIDs) == 0 {
		return nil, fmt.Errorf("image %s has no layers", ref)
	}

	canonicalRef := CanonicalImageRef(ref, meta.digest)
	imageWorkDir := filepath.Join(d.workDir, SanitizePathKey(ref))
	env := &MountEnv{
		Driver:       d,
		Client:       snapClient,
		Snapshotter:  d.snapshotterName,
		ImageRef:     ref,
		CanonicalRef: canonicalRef,
		ImageDigest:  meta.digest,
		ImageWorkDir: imageWorkDir,
		AuthConfig:   req.AuthConfig,
		ChainInfos:   ComputeChainIDs(meta.diffIDs),
		LayerDigests: meta.layerDigests,
	}

	mounted, err := d.layerMounter(ctx, env)
	rollback := func() {
		d.teardownWorkDir(context.WithoutCancel(ctx), imageWorkDir, mounted)
	}
	if err != nil {
		rollback()
		return nil, err
	}
	if mounted == nil || len(mounted.MountDirs) == 0 {
		rollback()
		return nil, fmt.Errorf("provider %s mounted no layers for %s", d.name, ref)
	}
	layerDirs, err := d.linkLayers(ctx, imageWorkDir, mounted.MountDirs)
	if err != nil {
		rollback()
		return nil, err
	}

	d.mu.Lock()
	refCount := 1
	var staleKeys []string
	if prev, ok := d.leases[ref]; ok {
		// The previous lease lost its layers. Its holders now hold this one.
		stopIdleTimer(prev)
		prev.evicted = true
		refCount += prev.refCount
		staleKeys = prev.snapshotKeys
	}
	d.leases[ref] = &imageLease{
		imageRef:     ref,
		digest:       meta.digest,
		config:       meta.config,
		snapshotKeys: mounted.SnapshotKeys,
		layers:       layerDirs,
		workDir:      imageWorkDir,
		refCount:     refCount,
	}
	d.resolvedMeta[ref] = meta
	d.resolvedMeta[canonicalRef] = meta
	d.mu.Unlock()

	nsCtx := d.WithNamespaceContext(context.WithoutCancel(ctx))
	for _, key := range staleKeys {
		RemoveSnapshot(nsCtx, snapClient, d.snapshotterName, key)
	}

	slog.InfoContext(ctx, "Prepared image",
		slog.String("provider", d.name),
		slog.String("image", ref),
		slog.String("digest", meta.digest),
		slog.Int("layers", len(layerDirs)),
		slog.Duration("duration", time.Since(start)))

	cfg := *meta.config
	return &client.PrepareResult{
		ImageDigest:  meta.digest,
		Config:       &cfg,
		LayerDirs:    slices.Clone(layerDirs),
		ProviderName: d.name,
	}, nil
}

// linkLayers waits for each mount dir to become listable, then links it as
// <imageWorkDir>/layer-<i>/fs and marks the layer finalized. It returns the
// layer directories.
func (d *Driver) linkLayers(ctx context.Context, imageWorkDir string, mountDirs []string) ([]string, error) {
	layerDirs := make([]string, len(mountDirs))
	errs := make([]error, len(mountDirs))
	if i := slices.Index(mountDirs, ""); i >= 0 {
		return nil, fmt.Errorf("empty mount directory for layer %d", i)
	}
	var wg sync.WaitGroup
	for i, mountDir := range mountDirs {
		layerDir := filepath.Join(imageWorkDir, fmt.Sprintf("layer-%d", i))
		layerDirs[i] = layerDir
		wg.Go(func() {
			errs[i] = d.linkLayer(ctx, mountDir, layerDir)
			if errs[i] != nil {
				errs[i] = fmt.Errorf("layer %d: %w", i, errs[i])
			}
		})
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return layerDirs, nil
}

func (d *Driver) linkLayer(ctx context.Context, mountDir, layerDir string) error {
	if err := d.probeListable(ctx, mountDir); err != nil {
		return err
	}
	if err := os.MkdirAll(layerDir, 0o755); err != nil {
		return fmt.Errorf("creating layer directory: %w", err)
	}
	fsLink := filepath.Join(layerDir, "fs")
	if err := os.Remove(fsLink); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("removing old layer link: %w", err)
	}
	if err := os.Symlink(mountDir, fsLink); err != nil {
		return fmt.Errorf("linking layer view: %w", err)
	}
	if err := os.WriteFile(filepath.Join(layerDir, "finalized"), nil, 0o600); err != nil {
		return fmt.Errorf("writing finalized marker: %w", err)
	}
	return nil
}

// probeListable waits until mountDir can be listed. A new FUSE or kernel
// mount can take a moment to show up in this mount namespace, so it polls
// with a short backoff.
func (d *Driver) probeListable(ctx context.Context, mountDir string) error {
	if d.listableTimeout <= 0 {
		return nil
	}
	deadline := time.Now().Add(d.listableTimeout)
	interval := 2 * time.Millisecond
	for {
		_, err := os.ReadDir(mountDir)
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("mount %q not listable within %s: %w", mountDir, d.listableTimeout, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
		interval = min(2*interval, listableMaxInterval)
	}
}

// ReleaseImage releases one lease on the image. When the last lease goes,
// the image stays mounted for the idle TTL, or is unmounted right away when
// the TTL is zero or the driver is shutting down. Releasing an image without
// leases does nothing.
func (d *Driver) ReleaseImage(ctx context.Context, req *client.ReleaseRequest) error {
	if req == nil || req.ImageRef == "" {
		return errors.New("image reference is required")
	}
	ref := req.ImageRef

	d.mu.Lock()
	var lease *imageLease
	for {
		lease = d.leases[ref]
		if lease == nil || lease.evicted {
			d.mu.Unlock()
			return nil
		}
		if flight, ok := d.inflight[ref]; ok {
			d.mu.Unlock()
			select {
			case <-flight.done:
			case <-ctx.Done():
				return ctx.Err()
			}
			d.mu.Lock()
			continue
		}
		if lease.refCount <= 0 {
			d.mu.Unlock()
			return nil
		}
		lease.refCount--
		if lease.refCount > 0 {
			d.mu.Unlock()
			return nil
		}
		if d.leaseIdleTTL > 0 && !d.stoppingLocked() {
			stopIdleTimer(lease)
			idle := lease
			lease.idleTimer = time.AfterFunc(d.leaseIdleTTL, func() {
				d.evictIdleLease(idle)
			})
			d.mu.Unlock()
			return nil
		}
		break
	}
	flight := d.beginLeaseTeardownLocked(lease)
	d.mu.Unlock()
	d.finishLeaseTeardown(ctx, lease, flight)
	return nil
}

// beginLeaseTeardownLocked removes lease from the driver and marks its image
// in flight, so PrepareImage and ReleaseImage wait for the teardown.
func (d *Driver) beginLeaseTeardownLocked(lease *imageLease) *inflightPrep {
	lease.evicted = true
	stopIdleTimer(lease)
	for key, l := range d.leases {
		if l == lease {
			delete(d.leases, key)
		}
	}
	flight := &inflightPrep{done: make(chan struct{})}
	d.inflight[lease.imageRef] = flight
	return flight
}

// finishLeaseTeardown removes the lease's views and work dir.
func (d *Driver) finishLeaseTeardown(ctx context.Context, lease *imageLease, flight *inflightPrep) {
	d.teardownWorkDir(context.WithoutCancel(ctx), lease.workDir, &MountedLayers{SnapshotKeys: lease.snapshotKeys})

	d.mu.Lock()
	defer d.mu.Unlock()
	if d.inflight[lease.imageRef] == flight {
		delete(d.inflight, lease.imageRef)
	}
	close(flight.done)
}

// teardownWorkDir runs the teardown hook, removes the snapshots in mounted,
// and removes workDir.
func (d *Driver) teardownWorkDir(ctx context.Context, workDir string, mounted *MountedLayers) {
	if d.workDirTeardownHook != nil {
		d.workDirTeardownHook(workDir)
	}
	if mounted != nil && (len(mounted.SnapshotKeys) > 0 || len(mounted.CommittedKeys) > 0) {
		if snapClient, err := d.getClient(); err != nil {
			slog.WarnContext(ctx, "Failed to connect to snapshotter to remove snapshots",
				slog.String("provider", d.name),
				slog.Any("err", err))
		} else {
			nsCtx := d.WithNamespaceContext(ctx)
			for _, key := range mounted.SnapshotKeys {
				RemoveSnapshot(nsCtx, snapClient, d.snapshotterName, key)
			}
			// Remove committed snapshots child first.
			for _, key := range slices.Backward(mounted.CommittedKeys) {
				RemoveSnapshot(nsCtx, snapClient, d.snapshotterName, key)
			}
		}
	}
	if err := os.RemoveAll(workDir); err != nil {
		slog.WarnContext(ctx, "Failed to remove image work dir",
			slog.String("provider", d.name),
			slog.String("workDir", workDir),
			slog.Any("err", err))
	}
}

// evictIdleLease unmounts lease when its idle TTL expires, unless it was
// leased again or replaced.
func (d *Driver) evictIdleLease(lease *imageLease) {
	d.mu.Lock()
	for {
		if d.closed || lease.evicted || lease.refCount > 0 || d.leases[lease.imageRef] != lease {
			d.mu.Unlock()
			return
		}
		flight, ok := d.inflight[lease.imageRef]
		if !ok {
			break
		}
		d.mu.Unlock()
		<-flight.done
		d.mu.Lock()
	}
	flight := d.beginLeaseTeardownLocked(lease)
	d.mu.Unlock()
	d.finishLeaseTeardown(context.Background(), lease, flight)
}

// Close stops the idle timers and closes the snapshotter connection. It
// leaves images mounted, so a restarted plugin can reconcile them.
func (d *Driver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	for _, l := range d.leases {
		stopIdleTimer(l)
	}
	if d.conn != nil {
		err := d.conn.Close()
		d.conn = nil
		return err
	}
	return nil
}
