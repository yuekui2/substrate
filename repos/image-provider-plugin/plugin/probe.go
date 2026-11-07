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
	"fmt"
	"log/slog"
	"os"
	"slices"
	"time"

	"github.com/ai-on-gke/image-provider-plugin/client"
	snapshotsapi "github.com/containerd/containerd/api/services/snapshots/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// probeKey is the snapshot the health check stats. It need not exist.
const probeKey = "ate-image-provider-probe"

// CanProvide reports whether the driver can serve the image now. It returns
// true for an image that is mounted, false for one the snapshotter recently
// declined, and otherwise whether the provider passes a health check. It
// returns false while the driver is shutting down.
func (d *Driver) CanProvide(ctx context.Context, req *client.PrepareRequest) (bool, error) {
	ref := ""
	if req != nil {
		ref = req.ImageRef
	}
	d.mu.Lock()
	if d.stoppingLocked() {
		d.mu.Unlock()
		return false, nil
	}
	var layers []string
	if lease, ok := d.leases[ref]; ok && !lease.evicted {
		layers = slices.Clone(lease.layers)
	}
	declined := d.isDeclinedLocked(ref, time.Now())
	d.mu.Unlock()

	if layersExist(layers) {
		return true, nil
	}
	if declined {
		return false, nil
	}
	probeCtx, cancel := context.WithTimeout(ctx, DefaultProbeTimeout)
	defer cancel()
	if err := d.probe(probeCtx); err != nil {
		slog.DebugContext(ctx, "Provider failed its health check",
			slog.String("provider", d.name),
			slog.Any("err", err))
		return false, nil
	}
	return true, nil
}

// isDeclinedLocked reports whether the snapshotter declined ref within the
// decline TTL, and forgets expired declines.
func (d *Driver) isDeclinedLocked(ref string, now time.Time) bool {
	exp, ok := d.declined[ref]
	if !ok {
		return false
	}
	if !now.Before(exp) {
		delete(d.declined, ref)
		return false
	}
	return true
}

// probe checks that the snapshotter answers Snapshots requests and that the
// probe hook, if any, passes.
func (d *Driver) probe(ctx context.Context) error {
	d.mu.Lock()
	snapClient := d.snapshotsClient
	d.mu.Unlock()
	if snapClient == nil {
		fi, err := os.Stat(d.socket)
		if err != nil {
			return fmt.Errorf("stat snapshotter socket: %w", err)
		}
		if fi.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("snapshotter path %s is not a unix socket", d.socket)
		}
		// Until the first PrepareImage, dial for each check. A shared
		// connection that failed while the snapshotter was starting would
		// stay in reconnect backoff after the snapshotter is up.
		conn, err := d.dial()
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close() }()
		snapClient = snapshotsapi.NewSnapshotsClient(conn)
	}
	_, err := snapClient.Stat(d.WithNamespaceContext(ctx), &snapshotsapi.StatSnapshotRequest{
		Snapshotter: d.snapshotterName,
		Key:         probeKey,
	})
	if code := status.Code(err); code != codes.OK && code != codes.NotFound {
		return fmt.Errorf("checking snapshotter: %w", err)
	}
	if d.probeHook != nil {
		return d.probeHook(ctx)
	}
	return nil
}

// waitReady checks the provider's health until it passes, or until the
// startup timeout or ctx ends. A zero startup timeout skips the checks.
func (d *Driver) waitReady(ctx context.Context) error {
	if d.startupTimeout <= 0 {
		return nil
	}
	waitCtx, cancel := context.WithTimeout(ctx, d.startupTimeout)
	defer cancel()

	start := time.Now()
	for attempt := 1; ; attempt++ {
		probeCtx, probeCancel := context.WithTimeout(waitCtx, DefaultProbeTimeout)
		err := d.probe(probeCtx)
		probeCancel()
		if err == nil {
			slog.InfoContext(ctx, "Provider passed its health check",
				slog.String("provider", d.name),
				slog.Int("attempts", attempt),
				slog.Duration("elapsed", time.Since(start)))
			return nil
		}
		if attempt == 1 {
			slog.InfoContext(ctx, "Waiting for the provider to pass its health check",
				slog.String("provider", d.name),
				slog.String("socket", d.socket),
				slog.Duration("startupTimeout", d.startupTimeout),
				slog.Any("err", err))
		}
		select {
		case <-waitCtx.Done():
			if ctx.Err() != nil {
				return fmt.Errorf("waiting for provider %s: %w (last error: %v)", d.name, ctx.Err(), err)
			}
			return fmt.Errorf("provider %s not ready within %s: %w", d.name, d.startupTimeout, err)
		case <-time.After(startupPollInterval):
		}
	}
}
