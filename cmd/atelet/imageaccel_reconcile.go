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
	"time"

	"go.ate.dev/substrate/internal/imagecache"
	"go.ate.dev/substrate/internal/proto/ateworkerpb"
	"go.ate.dev/substrate/oci-image-accelerator/pkg/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	imageAcceleratorNodeLabel     = "ate.dev/image-accelerator"
	imageAcceleratorDisabledValue = "disabled"
	imageAcceleratorMonitorPeriod = 2 * time.Second
)

// scanActiveStreamedLeases scans the actors directory for active bundle overlay specs
// and returns all streamed image leases currently mounted by resident actors.
func scanActiveStreamedLeases(actorsDir string) ([]*imageaccel.ActiveImageLayers, error) {
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

	activeMap := make(map[string]*imageaccel.ActiveImageLayers)
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
			lease = &imageaccel.ActiveImageLayers{
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

	result := make([]*imageaccel.ActiveImageLayers, 0, len(activeMap))
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
// ImageAccelerator adapter, and sweeps orphaned streaming workdirs.
func reconcileStreamingLeases(ctx context.Context, client imageaccel.Client, actorsDir string) error {
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

// evictActiveStreamedActors scans local workers and sends EvictWorkload to each
// active actor whose rootfs-overlay.json uses streamed layers, triggering the
// actor's SIGTERM handler to cycle through SuspendActor -> ResumeActor.
func (s *Atelet) evictActiveStreamedActors(ctx context.Context, actorsDir, workersDir string) int {
	if s.workerDialer == nil || workersDir == "" {
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

	evicted := 0
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
			s.evictedStreamedMu.Unlock()
			if alreadyEvicted {
				continue
			}
			evictCtx, evictCancel := context.WithTimeout(ctx, statsRPCTimeout)
			_, err := client.EvictWorkload(evictCtx, &ateworkerpb.EvictWorkloadRequest{ActorUid: actorUID})
			evictCancel()
			if err != nil {
				slog.WarnContext(ctx, "Failed to evict active streamed actor",
					slog.String("actorUID", actorUID),
					slog.String("workerPodUID", podUID),
					slog.Any("err", err))
				continue
			}
			s.evictedStreamedMu.Lock()
			s.evictedStreamedActors[actorUID] = true
			s.evictedStreamedMu.Unlock()
			evicted++
			slog.InfoContext(ctx, "Triggered SIGTERM eviction for active streamed actor",
				slog.String("actorUID", actorUID),
				slog.String("workerPodUID", podUID))
		}
	}
	return evicted
}

// startImageAcceleratorMonitor watches the local Node for the
// ate.dev/image-accelerator=disabled opt-out label and periodically polls the
// adapter socket for lame-duck eviction (evicting=true) or socket appearance.
func startImageAcceleratorMonitor(
	ctx context.Context,
	k8sClient kubernetes.Interface,
	nodeName string,
	socketPath string,
	stopCh <-chan struct{},
	wmService *Atelet,
	actorsDir string,
	workersDir string,
) {
	if wmService == nil {
		return
	}
	if k8sClient != nil && nodeName != "" {
		factory := kubeinformers.NewSharedInformerFactoryWithOptions(k8sClient, 0,
			kubeinformers.WithTweakListOptions(func(o *metav1.ListOptions) {
				o.FieldSelector = fields.OneTermEqualSelector("metadata.name", nodeName).String()
			}))
		nodeInformer := factory.Core().V1().Nodes().Informer()
		updateDrain := func(obj any) {
			node, ok := obj.(*corev1.Node)
			if !ok || node == nil {
				return
			}
			draining := node.Labels[imageAcceleratorNodeLabel] == imageAcceleratorDisabledValue
			if prev := wmService.SetStreamingDraining(draining); prev != draining {
				slog.InfoContext(ctx, "ImageAccelerator node drain state changed",
					slog.String("node", nodeName),
					slog.Bool("draining", draining))
				if draining {
					wmService.evictActiveStreamedActors(ctx, actorsDir, workersDir)
				}
			}
		}
		_, _ = nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    updateDrain,
			UpdateFunc: func(_, newObj any) { updateDrain(newObj) },
		})
		factory.Start(stopCh)
	}

	if wmService.imageAcceleratorClient == nil {
		return
	}

	go func() {
		socketAvail := imageaccel.IsSocketAvailable(socketPath)
		ticker := time.NewTicker(imageAcceleratorMonitorPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			case <-ticker.C:
				nowAvail := imageaccel.IsSocketAvailable(socketPath)
				if nowAvail && !socketAvail {
					if err := reconcileStreamingLeases(ctx, wmService.imageAcceleratorClient, actorsDir); err != nil {
						slog.WarnContext(ctx, "Failed to reconcile streamed image leases after adapter socket appeared", slog.Any("err", err))
					}
				}
				socketAvail = nowAvail

				shouldEvict := wmService.IsStreamingDraining()
				if !shouldEvict && nowAvail {
					probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
					evicting, err := wmService.imageAcceleratorClient.IsEvicting(probeCtx)
					cancel()
					if err == nil && evicting {
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
