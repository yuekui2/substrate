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
	"path/filepath"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ai-on-gke/image-provider-plugin/client"
)

func canProvide(t *testing.T, d *Driver, ref string) bool {
	t.Helper()
	ok, err := d.CanProvide(t.Context(), &client.PrepareRequest{ImageRef: ref})
	if err != nil {
		t.Fatalf("CanProvide(%s): %v", ref, err)
	}
	return ok
}

func TestDriverRemembersDeclines(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d, snap := newTestDriver(t, WithDeclineTTL(time.Minute))
		snap.declinePrep = true
		const ref = "example.com/plain:v1"
		prepares := func() int {
			snap.mu.Lock()
			defer snap.mu.Unlock()
			return len(snap.prepareReqs)
		}

		if !canProvide(t, d, ref) {
			t.Fatal("CanProvide = false before any decline")
		}
		if _, err := d.PrepareImage(t.Context(), &client.PrepareRequest{ImageRef: ref}); !errors.Is(err, client.ErrNotSupported) {
			t.Fatalf("PrepareImage = %v, want ErrNotSupported", err)
		}
		n := prepares()

		if canProvide(t, d, ref) {
			t.Error("CanProvide = true for a declined image")
		}
		if _, err := d.PrepareImage(t.Context(), &client.PrepareRequest{ImageRef: ref}); !errors.Is(err, client.ErrNotSupported) {
			t.Errorf("PrepareImage = %v, want ErrNotSupported", err)
		}
		if got := prepares(); got != n {
			t.Errorf("snapshotter got %d more Prepare calls for a declined image, want 0", got-n)
		}
		if !canProvide(t, d, "example.com/other:v1") {
			t.Error("CanProvide = false for an image that was not declined")
		}

		time.Sleep(2 * time.Minute)
		if !canProvide(t, d, ref) {
			t.Error("CanProvide = false after the decline TTL")
		}
	})
}

func TestDriverCanProvideChecksHealth(t *testing.T) {
	var healthy atomic.Bool
	d, _ := newTestDriver(t, WithProbeHook(func(context.Context) error {
		if !healthy.Load() {
			return errors.New("proxy down")
		}
		return nil
	}))
	const ref = "example.com/app:v1"

	if canProvide(t, d, ref) {
		t.Error("CanProvide = true while the provider is unhealthy")
	}
	healthy.Store(true)
	if !canProvide(t, d, ref) {
		t.Fatal("CanProvide = false while the provider is healthy")
	}
	prepare(t, d, ref)
	healthy.Store(false)
	if !canProvide(t, d, ref) {
		t.Error("CanProvide = false for a mounted image")
	}
}

func TestDriverProbeNeedsSocket(t *testing.T) {
	d, err := NewDriver("test-provider", filepath.Join(t.TempDir(), "missing.sock"), "",
		WithWorkDir(t.TempDir()))
	if err != nil {
		t.Fatalf("NewDriver: %v", err)
	}
	defer d.Close()
	if canProvide(t, d, "example.com/app:v1") {
		t.Error("CanProvide = true without a snapshotter socket")
	}
}

func TestDriverWaitReady(t *testing.T) {
	for _, tc := range []struct {
		name        string
		failures    int32
		wantErr     bool
		wantAttempt int32
	}{
		{name: "ready", failures: 0, wantAttempt: 1},
		{name: "ready after retries", failures: 3, wantAttempt: 4},
		{name: "never ready", failures: 1 << 30, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var attempts atomic.Int32
				d, _ := newTestDriver(t, WithStartupTimeout(time.Minute), WithProbeHook(func(context.Context) error {
					if attempts.Add(1) <= tc.failures {
						return errors.New("not ready")
					}
					return nil
				}))
				err := d.waitReady(t.Context())
				if (err != nil) != tc.wantErr {
					t.Fatalf("waitReady = %v, want error %v", err, tc.wantErr)
				}
				if !tc.wantErr && attempts.Load() != tc.wantAttempt {
					t.Errorf("health checks = %d, want %d", attempts.Load(), tc.wantAttempt)
				}
			})
		})
	}
}
