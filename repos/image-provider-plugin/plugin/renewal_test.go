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
	"sync"
	"testing"
	"testing/synctest"
	"time"

	imageprovider "github.com/ai-on-gke/image-provider-plugin/api/imageprovider/v1alpha1"
	"github.com/ai-on-gke/image-provider-plugin/client"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

func TestCredentialRenewalOptions(t *testing.T) {
	noHook := func(context.Context, string, *client.RefreshRequest) error { return nil }
	for _, tc := range []struct {
		mode        string
		want        imageprovider.AuthRefresh
		wantRenewal time.Duration
		wantErr     bool
	}{
		{mode: CredentialRenewalClient, want: imageprovider.AuthRefresh_AUTH_REFRESH_REQUIRED},
		{mode: CredentialRenewalSelf, want: imageprovider.AuthRefresh_AUTH_REFRESH_NOT_NEEDED, wantRenewal: DefaultSelfRenewalInterval},
		{mode: "", wantErr: true},
		{mode: "caller", wantErr: true},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			opts, err := CredentialRenewalOptions(tc.mode)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("CredentialRenewalOptions(%q) error = %v, want error %v", tc.mode, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			d, _ := newTestDriver(t, append(opts, WithAuthRefreshHook(noHook))...)
			if d.authRefresh != tc.want || d.selfRenewal != tc.wantRenewal {
				t.Errorf("driver reports %v and renews every %v, want %v and %v", d.authRefresh, d.selfRenewal, tc.want, tc.wantRenewal)
			}
		})
	}
}

// A self-renewing driver reads the registry with its own keychain only, and
// renews the credentials of each leased image on every tick.
func TestDriverSelfRenewal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var (
			mu          sync.Mutex
			resolveAuth []*client.AuthConfig
			renewed     []*client.RefreshRequest
		)
		resolver := func(ctx context.Context, ref string, auth *client.AuthConfig) (string, *v1.Config, []string, []string, error) {
			mu.Lock()
			resolveAuth = append(resolveAuth, auth)
			mu.Unlock()
			return staticResolver(testDiffIDs)(ctx, ref, auth)
		}
		d, _ := newTestDriver(t,
			WithImageResolver(resolver),
			WithSelfRenewal(time.Minute),
			WithAuthRefreshHook(func(_ context.Context, _ string, req *client.RefreshRequest) error {
				mu.Lock()
				renewed = append(renewed, req)
				mu.Unlock()
				return nil
			}))
		const ref = "example.com/app:v1"
		ctx := t.Context()
		auth := &client.AuthConfig{Username: "caller", Password: "secret"}
		if _, err := d.PrepareImage(ctx, &client.PrepareRequest{ImageRef: ref, LeaseID: "a", AuthConfig: auth}); err != nil {
			t.Fatalf("PrepareImage: %v", err)
		}
		prepare(t, d, ref, "b")
		if got, err := d.RefreshAuth(ctx, &client.RefreshRequest{LeaseID: "a", AuthConfig: auth}); err != nil || got != imageprovider.AuthRefresh_AUTH_REFRESH_NOT_NEEDED {
			t.Errorf("RefreshAuth = %v, %v; want AUTH_REFRESH_NOT_NEEDED", got, err)
		}

		time.Sleep(time.Minute)
		synctest.Wait()
		mu.Lock()
		if len(resolveAuth) != 1 || resolveAuth[0] != nil {
			t.Errorf("resolver got credentials %v, want only the plugin's (nil)", resolveAuth)
		}
		if len(renewed) != 1 || renewed[0].ImageRef != ref || renewed[0].AuthConfig != nil {
			t.Errorf("renewed %+v after one tick, want %s once with no credentials", renewed, ref)
		}
		renewed = nil
		mu.Unlock()

		release(t, d, "a")
		release(t, d, "b")
		time.Sleep(time.Minute)
		synctest.Wait()
		mu.Lock()
		defer mu.Unlock()
		if len(renewed) != 0 {
			t.Errorf("renewed %+v after the last release, want nothing", renewed)
		}
	})
}

// A self-renewing driver declines an image its own keychain cannot read, so
// the caller pulls it instead of failing later.
func TestDriverSelfRenewalDeclinesUnreadableImage(t *testing.T) {
	errDenied := errors.New("denied")
	d, _ := newTestDriver(t,
		WithImageResolver(func(context.Context, string, *client.AuthConfig) (string, *v1.Config, []string, []string, error) {
			return "", nil, nil, nil, errDenied
		}),
		WithSelfRenewal(time.Minute),
		WithAuthRefreshHook(func(context.Context, string, *client.RefreshRequest) error { return nil }))
	_, err := d.PrepareImage(t.Context(), &client.PrepareRequest{ImageRef: "example.com/private:v1", LeaseID: "a"})
	if !errors.Is(err, client.ErrNotSupported) {
		t.Errorf("PrepareImage = %v, want ErrNotSupported", err)
	}
}
