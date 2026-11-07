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
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ai-on-gke/image-provider-plugin/client"
	"github.com/google/go-containerregistry/pkg/authn"
)

type staticKeychain struct{ username, password string }

func (k staticKeychain) Resolve(authn.Resource) (authn.Authenticator, error) {
	return authn.FromConfig(authn.AuthConfig{Username: k.username, Password: k.password}), nil
}

func readDockerConfig(t *testing.T, path string) dockerConfigFile {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg dockerConfigFile
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	return cfg
}

func TestDockerConfigSyncer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docker", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"auths":{"other.example.com":{"auth":"b3RoZXI="}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s := newDockerConfigSyncer(path)
	ctx := t.Context()

	s.Sync(ctx, nil, "gcr.io/proj/app:v1", &client.AuthConfig{Username: "_token", Password: "secret"}, false)
	cfg := readDockerConfig(t, path)
	wantAuth := base64.StdEncoding.EncodeToString([]byte("_token:secret"))
	if got := cfg.Auths["gcr.io"]; got.Username != "_token" || got.Auth != wantAuth {
		t.Errorf("gcr.io entry = %+v, want username _token and auth %s", got, wantAuth)
	}
	if _, ok := cfg.Auths["other.example.com"]; !ok {
		t.Errorf("entry for other.example.com was dropped: %+v", cfg.Auths)
	}

	// The same credentials again within the TTL skip the write.
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s.Sync(ctx, nil, "gcr.io/proj/app:v1", &client.AuthConfig{Username: "_token", Password: "secret"}, false)
	if cfg := readDockerConfig(t, path); len(cfg.Auths) != 0 {
		t.Errorf("Sync rewrote unchanged credentials: %+v", cfg.Auths)
	}

	// force writes them anyway.
	s.Sync(ctx, nil, "gcr.io/proj/app:v1", &client.AuthConfig{Username: "_token", Password: "secret"}, true)
	if cfg := readDockerConfig(t, path); cfg.Auths["gcr.io"].Username != "_token" {
		t.Errorf("forced Sync did not write credentials: %+v", cfg.Auths)
	}

	// Without request credentials, the keychain supplies them.
	s.Sync(ctx, staticKeychain{"kc-user", "kc-pass"}, "us-docker.pkg.dev/proj/repo/app:v1", nil, false)
	if got := readDockerConfig(t, path).Auths["us-docker.pkg.dev"]; got.Username != "kc-user" || got.Password != "kc-pass" {
		t.Errorf("us-docker.pkg.dev entry = %+v, want keychain credentials", got)
	}
}

func TestDockerConfigSyncerAnonymous(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	newDockerConfigSyncer(path).Sync(t.Context(), authn.NewMultiKeychain(), "example.com/app:v1", nil, true)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("config written for an anonymous image (err=%v)", err)
	}
}
