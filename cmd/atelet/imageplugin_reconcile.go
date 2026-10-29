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
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"go.ate.dev/substrate/api/ateapi/v1"
	"go.ate.dev/substrate/api/imageplugin/v1"
	"go.ate.dev/substrate/internal/imagecache"
	"go.ate.dev/substrate/internal/imagepluginclient"
	"go.ate.dev/substrate/internal/proto/ateworkerpb"
)

const (
	imagePluginMonitorPeriod = 2 * time.Second
	// drainSuspendTimeout bounds one suspend of an actor off a draining
	// image plugin, checkpoint included.
	drainSuspendTimeout = 10 * time.Minute
)

// scanActiveStreamedLeases scans the actors directory for active bundle overlay specs
// and returns all streamed image leases currently mounted by resident actors.
func scanActiveStreamedLeases(actorsDir string) ([]*imagepluginclient.ActiveImageLayers, error) {
	if actorsDir == "" {
		return nil, nil
	}
	actorEntries, err := os.ReadDir(actorsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing actors dir %q: %w", actorsDir, err)
	}

	activeMap := make(map[string]*imagepluginclient.ActiveImageLayers)
	recordLease := func(imageRef, imageDigest string, streamed bool, layers []string) {
		if len(layers) == 0 || !streamed {
			return
		}
		key := imageRef
		if key == "" {
			key = imageDigest
		}
		if key == "" {
			return
		}
		lease, ok := activeMap[key]
		if !ok {
			lease = &imagepluginclient.ActiveImageLayers{
				ImageRef:    imageRef,
				ImageDigest: imageDigest,
				LayerDirs:   append([]string(nil), layers...),
				RefCount:    0,
			}
			activeMap[key] = lease
		}
		lease.RefCount++
	}

	for _, actor := range actorEntries {
		if !actor.IsDir() {
			continue
		}
		bundlesDir := filepath.Join(actorsDir, actor.Name(), "bundles")
		bundles, err := os.ReadDir(bundlesDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			slog.Warn("Failed to list bundles for actor during streaming reconciliation",
				slog.String("actor", actor.Name()), slog.Any("err", err))
			continue
		}
		for _, bundle := range bundles {
			if !bundle.IsDir() {
				continue
			}
			spec, err := imagecache.ReadSpec(filepath.Join(bundlesDir, bundle.Name()))
			if err != nil || spec == nil {
				continue
			}
			recordLease(spec.ImageRef, spec.ImageDigest, spec.Streamed, spec.Layers)
			for _, vol := range spec.ImageVolumes {
				recordLease(vol.ImageRef, vol.ImageDigest, vol.Streamed, vol.Layers)
			}
		}
	}

	result := make([]*imagepluginclient.ActiveImageLayers, 0, len(activeMap))
	for _, l := range activeMap {
		result = append(result, l)
	}
	return result, nil
}

// scanActiveStreamedActorUIDs returns the set of actor UIDs under actorsDir
// whose bundle overlay specs have Streamed: true.
func scanActiveStreamedActorUIDs(actorsDir string) (map[string]bool, error) {
	if actorsDir == "" {
		return nil, nil
	}
	actorEntries, err := os.ReadDir(actorsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing actors dir %q: %w", actorsDir, err)
	}

	streamed := make(map[string]bool)
	for _, actor := range actorEntries {
		if !actor.IsDir() {
			continue
		}
		bundlesDir := filepath.Join(actorsDir, actor.Name(), "bundles")
		bundles, err := os.ReadDir(bundlesDir)
		if err != nil {
			continue
		}
		for _, bundle := range bundles {
			if !bundle.IsDir() {
				continue
			}
			spec, err := imagecache.ReadSpec(filepath.Join(bundlesDir, bundle.Name()))
			if err != nil || spec == nil {
				continue
			}
			if spec.Streamed {
				streamed[actor.Name()] = true
				break
			}
			for _, vol := range spec.ImageVolumes {
				if vol.Streamed {
					streamed[actor.Name()] = true
					break
				}
			}
			if streamed[actor.Name()] {
				break
			}
		}
	}
	return streamed, nil
}

// reconcileStreamingLeases discovers all active streamed images mounted by
// running or sleeping actors on this node, restores their leases in the
// image plugin, and sweeps orphaned streaming workdirs.
func reconcileStreamingLeases(ctx context.Context, client imagepluginclient.Client, actorsDir string) error {
	if client == nil {
		return nil
	}
	leases, err := scanActiveStreamedLeases(actorsDir)
	if err != nil {
		return fmt.Errorf("scanning active streamed leases: %w", err)
	}
	if err := client.ReconcileImages(ctx, leases); err != nil {
		return fmt.Errorf("reconciling leases with %s: %w", client.Name(), err)
	}
	if len(leases) == 0 {
		slog.DebugContext(ctx, "No active streamed image leases to reconcile on startup", slog.String("provider", client.Name()))
		return nil
	}
	slog.InfoContext(ctx, "Successfully reconciled streamed image leases on startup",
		slog.String("provider", client.Name()),
		slog.Int("reconciledImages", len(leases)))
	return nil
}

// evictActiveStreamedActors asks the control plane to suspend each actor on
// this node whose rootfs-overlay.json uses streamed layers. The suspend
// releases the actor's image lease, so the draining plugin can exit; the
// actor resumes later from a snapshot. It returns how many suspends ran.
func (s *Atelet) evictActiveStreamedActors(ctx context.Context, actorsDir, workersDir string) int {
	if s.workerDialer == nil || s.workers == nil || workersDir == "" {
		return 0
	}
	streamedActors, err := scanActiveStreamedActorUIDs(actorsDir)
	if err != nil || len(streamedActors) == 0 {
		s.evictedStreamedMu.Lock()
		clear(s.evictedStreamedActors)
		s.evictedStreamedMu.Unlock()
		return 0
	}

	s.evictedStreamedMu.Lock()
	if s.evictedStreamedActors == nil {
		s.evictedStreamedActors = make(map[string]bool)
	}
	for uid := range s.evictedStreamedActors {
		if !streamedActors[uid] {
			delete(s.evictedStreamedActors, uid)
		}
	}
	s.evictedStreamedMu.Unlock()

	workerEntries, err := os.ReadDir(workersDir)
	if err != nil {
		return 0
	}

	var (
		wg      sync.WaitGroup
		evicted atomic.Int32
	)
	for _, entry := range workerEntries {
		if !entry.IsDir() {
			continue
		}
		podUID := entry.Name()
		if _, err := os.Stat(workerSocketPath(podUID)); err != nil {
			continue
		}
		client, err := s.dialWorker(ctx, podUID)
		if err != nil {
			continue
		}
		rpcCtx, cancel := context.WithTimeout(ctx, statsRPCTimeout)
		statsResp, err := client.GetActiveWorkloadStats(rpcCtx, &ateworkerpb.GetActiveWorkloadStatsRequest{})
		cancel()
		if err != nil {
			continue
		}
		for _, sample := range statsResp.GetSamples() {
			actorUID := sample.GetActorUid()
			if !streamedActors[actorUID] {
				continue
			}
			s.evictedStreamedMu.Lock()
			alreadyEvicted := s.evictedStreamedActors[actorUID]
			s.evictedStreamedActors[actorUID] = true
			s.evictedStreamedMu.Unlock()
			if alreadyEvicted {
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				suspendCtx, cancel := context.WithTimeout(ctx, drainSuspendTimeout)
				defer cancel()
				// The control plane runs the suspend back through atelet,
				// so no lock may be held here.
				_, err := s.workers.RequestActorSuspend(suspendCtx, &ateapi.RequestActorSuspendRequest{
					Worker:   &ateapi.ObjectRef{Name: podUID},
					Actor:    &ateapi.ObjectRef{Atespace: sample.GetAtespace(), Name: sample.GetActorName()},
					ActorUid: actorUID,
				})
				if err != nil {
					// Retry on the next sweep.
					s.evictedStreamedMu.Lock()
					delete(s.evictedStreamedActors, actorUID)
					s.evictedStreamedMu.Unlock()
					slog.WarnContext(ctx, "Failed to suspend actor on draining image plugin",
						slog.String("actorUID", actorUID),
						slog.String("workerPodUID", podUID),
						slog.Any("err", err))
					return
				}
				evicted.Add(1)
				slog.InfoContext(ctx, "Suspended actor on draining image plugin",
					slog.String("actorUID", actorUID),
					slog.String("workerPodUID", podUID))
			}()
		}
	}
	wg.Wait()
	return int(evicted.Load())
}

// startImagePluginMonitor polls the plugin socket. When the socket appears it
// reconciles leases, and when the plugin reports PLUGIN_STATE_DRAINING it
// evicts actors that use the plugin's layers.
func startImagePluginMonitor(
	ctx context.Context,
	socketPath string,
	stopCh <-chan struct{},
	wmService *Atelet,
	actorsDir string,
	workersDir string,
) {
	if wmService == nil {
		return
	}
	if wmService.imagePluginClient == nil {
		return
	}

	go func() {
		socketAvail := imagepluginclient.IsSocketAvailable(socketPath)
		ticker := time.NewTicker(imagePluginMonitorPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			case <-ticker.C:
				nowAvail := imagepluginclient.IsSocketAvailable(socketPath)
				if nowAvail && !socketAvail {
					if err := reconcileStreamingLeases(ctx, wmService.imagePluginClient, actorsDir); err != nil {
						slog.WarnContext(ctx, "Failed to reconcile streamed image leases after plugin socket appeared", slog.Any("err", err))
					}
				}
				socketAvail = nowAvail

				shouldEvict := false
				if nowAvail {
					probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
					state, err := wmService.imagePluginClient.State(probeCtx)
					cancel()
					// Only an explicit DRAINING evicts. Unknown states never do.
					if err == nil && state == imageplugin.PluginState_PLUGIN_STATE_DRAINING {
						shouldEvict = true
					}
				}
				if shouldEvict {
					wmService.evictActiveStreamedActors(ctx, actorsDir, workersDir)
				}
			}
		}
	}()
}
