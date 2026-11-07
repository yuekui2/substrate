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
	"time"

	providerclient "github.com/ai-on-gke/image-provider-plugin/client"

	"go.ate.dev/substrate/internal/imagecache"
)

// imageProviderMonitorPeriod is how often atelet checks the image provider
// plugin's socket.
const imageProviderMonitorPeriod = 2 * time.Second

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

// activeProviderLeases returns the provider leases that the overlay specs
// under actorsDir record.
func activeProviderLeases(actorsDir string) ([]*providerclient.ActiveLease, error) {
	var leases []*providerclient.ActiveLease
	add := func(leaseID, ref, digest string) {
		if leaseID != "" {
			leases = append(leases, &providerclient.ActiveLease{LeaseID: leaseID, ImageRef: ref, ImageDigest: digest})
		}
	}
	err := forEachOverlaySpec(actorsDir, func(_ string, spec *imagecache.OverlaySpec) {
		add(spec.LeaseID, spec.ImageRef, spec.ImageDigest)
		for _, v := range spec.ImageVolumes {
			add(v.LeaseID, v.ImageRef, v.ImageDigest)
		}
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(leases, func(a, b *providerclient.ActiveLease) int { return cmp.Compare(a.LeaseID, b.LeaseID) })
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
		slog.String("provider", s.imageProvider.Name()), slog.Int("leases", len(leases)))
	return nil
}

// monitorImageProvider reconciles the plugin's leases each time the plugin's
// socket appears, at startup or after the plugin restarts, until ctx ends. A
// failed reconcile is retried on the next check.
func (s *Atelet) monitorImageProvider(ctx context.Context, socket, actorsDir string) {
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
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
