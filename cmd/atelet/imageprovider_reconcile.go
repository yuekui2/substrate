// Copyright 2026 The Agent Substrate Authors
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
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	imageprovider "github.com/ai-on-gke/image-provider-plugin/api/imageprovider/v1alpha1"
	providerclient "github.com/ai-on-gke/image-provider-plugin/client"

	"go.ate.dev/substrate/api/ateapi/v1"
	"go.ate.dev/substrate/internal/imagecache"
	"go.ate.dev/substrate/internal/proto/ateworkerpb"
)

const (
	// imageProviderMonitorPeriod is how often atelet checks the image
	// provider plugin's socket and state.
	imageProviderMonitorPeriod = 2 * time.Second
	// imageProviderStateTimeout bounds one State call to the plugin.
	imageProviderStateTimeout = 2 * time.Second
	// drainSuspendTimeout bounds one suspend of an actor off a draining image
	// provider plugin, checkpoint included.
	drainSuspendTimeout = 10 * time.Minute
)

// forEachOverlaySpec calls fn with each actor UID under actorsDir and the
// overlay spec of each of its bundles. Unreadable bundles are logged and
// skipped.
func forEachOverlaySpec(actorsDir string, fn func(actorUID string, spec *imagecache.OverlaySpec)) error {
	actors, err := os.ReadDir(actorsDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("while listing actors dir %q: %w", actorsDir, err)
	}
	for _, actor := range actors {
		if !actor.IsDir() {
			continue
		}
		bundlesDir := filepath.Join(actorsDir, actor.Name(), "bundles")
		bundles, err := os.ReadDir(bundlesDir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			slog.Warn("Failed to list an actor's bundles", slog.String("dir", bundlesDir), slog.Any("err", err))
			continue
		}
		for _, bundle := range bundles {
			if !bundle.IsDir() {
				continue
			}
			spec, err := imagecache.ReadSpec(filepath.Join(bundlesDir, bundle.Name()))
			if err != nil {
				slog.Warn("Failed to read an overlay spec", slog.String("bundle", filepath.Join(bundlesDir, bundle.Name())), slog.Any("err", err))
				continue
			}
			if spec != nil {
				fn(actor.Name(), spec)
			}
		}
	}
	return nil
}

// activeProviderLeases returns the images whose provider leases the overlay
// specs under actorsDir record, one entry per image ref with a lease count.
func activeProviderLeases(actorsDir string) ([]*providerclient.ActiveImageLayers, error) {
	byRef := map[string]*providerclient.ActiveImageLayers{}
	add := func(ref, digest string, streamed bool, layers []string) {
		if !streamed {
			return
		}
		lease := byRef[ref]
		if lease == nil {
			lease = &providerclient.ActiveImageLayers{ImageRef: ref, ImageDigest: digest, LayerDirs: layers}
			byRef[ref] = lease
		}
		lease.RefCount++
	}
	err := forEachOverlaySpec(actorsDir, func(_ string, spec *imagecache.OverlaySpec) {
		add(spec.ImageRef, spec.ImageDigest, spec.Streamed, spec.Layers)
		for _, v := range spec.ImageVolumes {
			add(v.ImageRef, v.ImageDigest, v.Streamed, v.Layers)
		}
	})
	if err != nil {
		return nil, err
	}
	leases := make([]*providerclient.ActiveImageLayers, 0, len(byRef))
	for _, lease := range byRef {
		leases = append(leases, lease)
	}
	slices.SortFunc(leases, func(a, b *providerclient.ActiveImageLayers) int { return cmp.Compare(a.ImageRef, b.ImageRef) })
	return leases, nil
}

// reconcileProviderLeases replaces the plugin's leases with those the overlay
// specs on the node record. It holds providerLeaseMu, so no bundle is between
// taking a lease and recording it in its spec.
func (s *Atelet) reconcileProviderLeases(ctx context.Context, actorsDir string) error {
	s.providerLeaseMu.Lock()
	defer s.providerLeaseMu.Unlock()
	leases, err := activeProviderLeases(actorsDir)
	if err != nil {
		return err
	}
	if err := s.imageProvider.ReconcileImages(ctx, leases); err != nil {
		return fmt.Errorf("while reconciling leases with the image provider plugin: %w", err)
	}
	slog.InfoContext(ctx, "Reconciled image provider leases",
		slog.String("provider", s.imageProvider.Name()), slog.Int("images", len(leases)))
	return nil
}

// streamedActorUIDs returns the actors under actorsDir whose overlay specs
// record a provider lease.
func streamedActorUIDs(actorsDir string) (map[string]bool, error) {
	uids := map[string]bool{}
	err := forEachOverlaySpec(actorsDir, func(actorUID string, spec *imagecache.OverlaySpec) {
		if spec.Streamed || slices.ContainsFunc(spec.ImageVolumes, func(v imagecache.ImageVolumeOverlay) bool { return v.Streamed }) {
			uids[actorUID] = true
		}
	})
	return uids, err
}

// suspendStreamedActors asks the control plane to suspend each actor running
// on the node that holds a provider lease. The suspend checkpoints the actor
// and releases its leases, so a draining plugin can exit, and the actor
// resumes later from its snapshot. An actor is asked for once while it holds
// a lease; a failed request is retried on the next call. It returns how many
// suspends succeeded.
func (s *Atelet) suspendStreamedActors(ctx context.Context, actorsDir, workersDir string) int {
	streamed, err := streamedActorUIDs(actorsDir)
	if err != nil {
		slog.WarnContext(ctx, "Failed to find the actors that hold image provider leases", slog.Any("err", err))
		return 0
	}
	s.drainMu.Lock()
	for uid := range s.drainSuspended {
		if !streamed[uid] {
			delete(s.drainSuspended, uid)
		}
	}
	s.drainMu.Unlock()
	if len(streamed) == 0 {
		return 0
	}

	workers, err := os.ReadDir(workersDir)
	if err != nil {
		slog.WarnContext(ctx, "Failed to list workers to suspend actors off the image provider plugin", slog.Any("err", err))
		return 0
	}
	var (
		wg        sync.WaitGroup
		suspended atomic.Int32
	)
	for _, w := range workers {
		podUID := w.Name()
		if !w.IsDir() {
			continue
		}
		if _, err := os.Stat(workerSocketPath(podUID)); err != nil {
			continue
		}
		worker, err := s.dialWorker(ctx, podUID)
		if err != nil {
			continue
		}
		rpcCtx, cancel := context.WithTimeout(ctx, statsRPCTimeout)
		resp, err := worker.GetActiveWorkloadStats(rpcCtx, &ateworkerpb.GetActiveWorkloadStatsRequest{})
		cancel()
		if err != nil {
			slog.WarnContext(ctx, "Failed to list a worker's actors", slog.String("pod_uid", podUID), slog.Any("err", err))
			continue
		}
		for _, sample := range resp.GetSamples() {
			actorUID := sample.GetActorUid()
			if !streamed[actorUID] {
				continue
			}
			s.drainMu.Lock()
			asked := s.drainSuspended[actorUID]
			s.drainSuspended[actorUID] = true
			s.drainMu.Unlock()
			if asked {
				continue
			}
			wg.Go(func() {
				suspendCtx, cancel := context.WithTimeout(ctx, drainSuspendTimeout)
				defer cancel()
				// The control plane runs the suspend back through atelet, so
				// no actor lock may be held here.
				_, err := s.workers.RequestActorSuspend(suspendCtx, &ateapi.RequestActorSuspendRequest{
					Worker:   &ateapi.ObjectRef{Name: podUID},
					Actor:    &ateapi.ObjectRef{Atespace: sample.GetAtespace(), Name: sample.GetActorName()},
					ActorUid: actorUID,
				})
				if err != nil {
					s.drainMu.Lock()
					delete(s.drainSuspended, actorUID)
					s.drainMu.Unlock()
					slog.WarnContext(ctx, "Failed to suspend an actor off the draining image provider plugin",
						slog.String("actor_uid", actorUID), slog.String("pod_uid", podUID), slog.Any("err", err))
					return
				}
				suspended.Add(1)
				slog.InfoContext(ctx, "Suspended an actor off the draining image provider plugin",
					slog.String("actor_uid", actorUID), slog.String("pod_uid", podUID))
			})
		}
	}
	wg.Wait()
	return int(suspended.Load())
}

// monitorImageProvider watches the image provider plugin until ctx ends. Each
// time the plugin's socket appears, at startup or after the plugin restarts,
// it reconciles the plugin's leases, and retries a failed reconcile on the
// next check. While the plugin reports PLUGIN_STATE_DRAINING, it suspends the
// actors that hold its leases.
func (s *Atelet) monitorImageProvider(ctx context.Context, socket, actorsDir, workersDir string) {
	available := false
	ticker := time.NewTicker(imageProviderMonitorPeriod)
	defer ticker.Stop()
	for {
		now := providerclient.IsSocketAvailable(socket)
		if now && !available {
			if err := s.reconcileProviderLeases(ctx, actorsDir); err != nil {
				slog.WarnContext(ctx, "Failed to reconcile image provider leases", slog.Any("err", err))
				now = false
			}
		}
		available = now
		if available && s.imageProviderDraining(ctx) {
			s.suspendStreamedActors(ctx, actorsDir, workersDir)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// imageProviderDraining reports whether the plugin says it is draining. Any
// other state, or a failed call, is not draining.
func (s *Atelet) imageProviderDraining(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, imageProviderStateTimeout)
	defer cancel()
	state, err := s.imageProvider.State(ctx)
	return err == nil && state == imageprovider.PluginState_PLUGIN_STATE_DRAINING
}
