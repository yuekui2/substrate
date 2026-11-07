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
	"slices"
	"time"
)

// Draining reports whether Drain or StartDrain was called.
func (d *Driver) Draining() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.draining
}

// activeLeasesLocked returns the images with leases in use, each with its
// lease count.
func (d *Driver) activeLeasesLocked() []string {
	counts := make(map[string]int)
	for _, ref := range d.leaseRefs {
		counts[ref]++
	}
	var refs []string
	for ref, n := range counts {
		refs = append(refs, fmt.Sprintf("%s (%d)", ref, n))
	}
	slices.Sort(refs)
	return refs
}

// notifyDrainedLocked closes d.drained once the driver is draining and no
// lease is in use.
func (d *Driver) notifyDrainedLocked() {
	if d.draining && !d.drainedClosed && len(d.leaseRefs) == 0 {
		d.drainedClosed = true
		close(d.drained)
	}
}

// StartDrain stops the driver from taking new leases: CanProvide returns
// false and PrepareImage declines every image. Images no workload uses are
// unmounted now, and each image in use is unmounted when its last lease is
// released.
func (d *Driver) StartDrain(ctx context.Context) {
	type teardown struct {
		lease  *imageLease
		flight *inflightPrep
	}
	var idle []teardown
	d.mu.Lock()
	d.draining = true
	for ref, l := range d.leases {
		if _, busy := d.inflight[ref]; busy || l.evicted || d.leaseCountLocked(ref) > 0 {
			continue
		}
		idle = append(idle, teardown{lease: l, flight: d.beginLeaseTeardownLocked(l)})
	}
	d.notifyDrainedLocked()
	d.mu.Unlock()

	for _, t := range idle {
		d.finishLeaseTeardown(ctx, t.lease, t.flight)
	}
}

// Drain calls StartDrain and waits until no lease is in use, the drain
// timeout passes, or ctx ends. It reports whether every lease was released.
func (d *Driver) Drain(ctx context.Context) bool {
	d.StartDrain(ctx)

	d.mu.Lock()
	active := d.activeLeasesLocked()
	d.mu.Unlock()
	if len(active) == 0 {
		slog.InfoContext(ctx, "Drained; no lease in use", slog.String("provider", d.name))
		return true
	}
	slog.InfoContext(ctx, "Draining; waiting for the leases in use to be released",
		slog.String("provider", d.name),
		slog.Duration("drainTimeout", d.drainTimeout),
		slog.Any("leases", active))

	timer := time.NewTimer(d.drainTimeout)
	defer timer.Stop()
	select {
	case <-d.drained:
		slog.InfoContext(ctx, "Drained; every lease was released", slog.String("provider", d.name))
		return true
	case <-timer.C:
	case <-ctx.Done():
	}
	d.mu.Lock()
	active = d.activeLeasesLocked()
	d.mu.Unlock()
	slog.WarnContext(ctx, "Stopped draining with leases in use",
		slog.String("provider", d.name),
		slog.Any("leases", active))
	return false
}
