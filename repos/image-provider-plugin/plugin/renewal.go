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
	"time"

	imageprovider "github.com/ai-on-gke/image-provider-plugin/api/imageprovider/v1alpha1"
	"github.com/ai-on-gke/image-provider-plugin/client"
)

// DefaultSelfRenewalInterval is how often a self-renewing driver renews the
// credentials of its leased images.
const DefaultSelfRenewalInterval = time.Minute

// Values for a plugin's --credential-renewal flag.
const (
	// CredentialRenewalClient makes the caller refresh credentials with
	// RefreshAuth.
	CredentialRenewalClient = "client"

	// CredentialRenewalSelf makes the plugin renew credentials with its own
	// keychain.
	CredentialRenewalSelf = "self"
)

// CredentialRenewalOptions returns the options for a lazily loading provider
// in the credential renewal mode, CredentialRenewalClient or
// CredentialRenewalSelf.
func CredentialRenewalOptions(mode string) ([]Option, error) {
	switch mode {
	case CredentialRenewalClient:
		return []Option{WithAuthRefresh(imageprovider.AuthRefresh_AUTH_REFRESH_REQUIRED)}, nil
	case CredentialRenewalSelf:
		return []Option{
			WithAuthRefresh(imageprovider.AuthRefresh_AUTH_REFRESH_NOT_NEEDED),
			WithSelfRenewal(DefaultSelfRenewalInterval),
		}, nil
	}
	return nil, fmt.Errorf("credential renewal must be %q or %q, got %q", CredentialRenewalClient, CredentialRenewalSelf, mode)
}

// startSelfRenewal starts the loop that renews the credentials of leased
// images every d.selfRenewal. Close stops it.
func (d *Driver) startSelfRenewal() {
	ctx, cancel := context.WithCancel(context.Background())
	d.stopRenewal = cancel
	d.renewalDone = make(chan struct{})
	go func() {
		defer close(d.renewalDone)
		t := time.NewTicker(d.selfRenewal)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				d.renewCredentials(ctx)
			}
		}
	}()
}

// renewCredentials calls the auth refresh hook with no credentials for each
// leased image, so the provider renews them with its keychain. It logs
// failures and tries again on the next tick.
func (d *Driver) renewCredentials(ctx context.Context) {
	type image struct{ ref, canonicalRef string }
	var images []image
	seen := make(map[string]bool)
	d.mu.Lock()
	for _, ref := range d.leaseRefs {
		lease := d.leases[ref]
		if seen[ref] || lease == nil {
			continue
		}
		seen[ref] = true
		images = append(images, image{ref: ref, canonicalRef: CanonicalImageRef(ref, lease.digest)})
	}
	d.mu.Unlock()

	for _, img := range images {
		if err := d.authRefreshHook(ctx, img.canonicalRef, &client.RefreshRequest{ImageRef: img.ref}); err != nil {
			slog.WarnContext(ctx, "Failed to renew registry credentials",
				slog.String("provider", d.name),
				slog.String("image", img.ref),
				slog.Any("err", err))
		}
	}
}
