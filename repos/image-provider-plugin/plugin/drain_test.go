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
	"errors"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ai-on-gke/image-provider-plugin/client"
)

func TestDriverDrain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d, _ := newTestDriver(t, WithDrainTimeout(time.Hour))
		const inUse, idle = "example.com/in-use:v1", "example.com/idle:v1"
		inUseRes := prepare(t, d, inUse)
		idleRes := prepare(t, d, idle)
		release(t, d, idle)

		drained := make(chan bool)
		go func() { drained <- d.Drain(t.Context()) }()
		synctest.Wait()

		if !d.Draining() {
			t.Error("Draining = false after Drain started")
		}
		if exists(filepath.Dir(idleRes.LayerDirs[0])) {
			t.Error("an idle image is still mounted while draining")
		}
		if canProvide(t, d, inUse) {
			t.Error("CanProvide = true while draining")
		}
		if _, err := d.PrepareImage(t.Context(), &client.PrepareRequest{ImageRef: inUse}); !errors.Is(err, client.ErrNotSupported) {
			t.Errorf("PrepareImage while draining = %v, want ErrNotSupported", err)
		}
		select {
		case <-drained:
			t.Fatal("Drain returned with a lease in use")
		default:
		}

		release(t, d, inUse)
		if !<-drained {
			t.Error("Drain = false after the last lease was released")
		}
		if exists(filepath.Dir(inUseRes.LayerDirs[0])) {
			t.Error("an image is still mounted after its last lease was released while draining")
		}
	})
}

func TestDriverDrainTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d, _ := newTestDriver(t, WithDrainTimeout(time.Minute))
		prepare(t, d, "example.com/in-use:v1")
		if d.Drain(t.Context()) {
			t.Error("Drain = true with a lease in use after the drain timeout")
		}
	})
}
