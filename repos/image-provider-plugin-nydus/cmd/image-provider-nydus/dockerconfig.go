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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ai-on-gke/image-provider-plugin/client"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
)

// dockerCredsCacheTTL is how long Sync skips rewriting unchanged credentials
// for a registry.
const dockerCredsCacheTTL = 5 * time.Minute

type dockerConfigFile struct {
	Auths map[string]dockerAuthEntry `json:"auths"`
}

type dockerAuthEntry struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Auth          string `json:"auth,omitempty"`
	IdentityToken string `json:"identitytoken,omitempty"`
	RegistryToken string `json:"registrytoken,omitempty"`
}

type pushedDockerCreds struct {
	digest   [32]byte
	pushedAt time.Time
}

// dockerConfigSyncer writes registry credentials into a Docker config.json
// that a snapshotter reads through DOCKER_CONFIG, such as
// containerd-nydus-grpc.
type dockerConfigSyncer struct {
	path string
	mu   sync.Mutex
	sent map[string]pushedDockerCreds
}

// newDockerConfigSyncer returns a dockerConfigSyncer that writes configPath.
func newDockerConfigSyncer(configPath string) *dockerConfigSyncer {
	return &dockerConfigSyncer{
		path: configPath,
		sent: make(map[string]pushedDockerCreds),
	}
}

// Sync resolves the credentials for imageRef from reqAuth, or from keychain
// when reqAuth has none, and writes them to the config file. Unless force is
// set, it skips the write when it wrote the same credentials for the
// registry within the last few minutes. It logs failures, since a missing
// credential shows up as a pull error in the snapshotter.
func (s *dockerConfigSyncer) Sync(ctx context.Context, keychain authn.Keychain, imageRef string, reqAuth *client.AuthConfig, force bool) {
	if s.path == "" || imageRef == "" {
		return
	}
	serverAddr, entry := resolveDockerAuth(keychain, imageRef, reqAuth)
	if serverAddr == "" || entry == nil {
		return
	}
	digest := sha256.Sum256([]byte(strings.Join([]string{
		serverAddr,
		entry.Username,
		entry.Password,
		entry.Auth,
		entry.IdentityToken,
		entry.RegistryToken,
	}, "\x00")))
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	if prev, ok := s.sent[serverAddr]; ok && !force && prev.digest == digest && now.Sub(prev.pushedAt) < dockerCredsCacheTTL {
		return
	}
	if err := s.write(serverAddr, entry); err != nil {
		slog.WarnContext(ctx, "Failed to write registry credentials to docker config",
			slog.String("path", s.path),
			slog.String("server", serverAddr),
			slog.Any("err", err))
		return
	}
	s.sent[serverAddr] = pushedDockerCreds{digest: digest, pushedAt: now}
	slog.InfoContext(ctx, "Wrote registry credentials to docker config",
		slog.String("path", s.path),
		slog.String("image", imageRef),
		slog.String("server", serverAddr))
}

// write sets the entry for serverAddr in the config file and keeps the other
// entries. It replaces the file atomically.
func (s *dockerConfigSyncer) write(serverAddr string, entry *dockerAuthEntry) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	var cfg dockerConfigFile
	b, err := os.ReadFile(s.path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return err
	case len(b) > 0:
		if err := json.Unmarshal(b, &cfg); err != nil {
			return fmt.Errorf("parsing %s: %w", s.path, err)
		}
	}
	if cfg.Auths == nil {
		cfg.Auths = make(map[string]dockerAuthEntry)
	}
	cfg.Auths[serverAddr] = *entry

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// resolveDockerAuth returns the registry host and config entry for refStr.
// It returns "" and nil when there are no credentials.
func resolveDockerAuth(keychain authn.Keychain, refStr string, reqAuth *client.AuthConfig) (string, *dockerAuthEntry) {
	ref, err := name.ParseReference(refStr)
	var serverAddr string
	if err == nil {
		serverAddr = ref.Context().RegistryStr()
	}
	var entry *dockerAuthEntry
	switch {
	case reqAuth != nil && (reqAuth.Username != "" || reqAuth.Password != "" || reqAuth.Auth != "" || reqAuth.IdentityToken != "" || reqAuth.RegistryToken != ""):
		if reqAuth.ServerAddress != "" {
			serverAddr = reqAuth.ServerAddress
		}
		entry = &dockerAuthEntry{
			Username:      reqAuth.Username,
			Password:      reqAuth.Password,
			Auth:          reqAuth.Auth,
			IdentityToken: reqAuth.IdentityToken,
			RegistryToken: reqAuth.RegistryToken,
		}
	case keychain != nil && err == nil:
		a, err := keychain.Resolve(ref.Context())
		if err != nil || a == nil || a == authn.Anonymous {
			return "", nil
		}
		cfg, err := a.Authorization()
		if err != nil || cfg == nil {
			return "", nil
		}
		if cfg.Username == "" && cfg.Password == "" && cfg.Auth == "" && cfg.IdentityToken == "" && cfg.RegistryToken == "" {
			return "", nil
		}
		entry = &dockerAuthEntry{
			Username:      cfg.Username,
			Password:      cfg.Password,
			Auth:          cfg.Auth,
			IdentityToken: cfg.IdentityToken,
			RegistryToken: cfg.RegistryToken,
		}
	default:
		return "", nil
	}
	if serverAddr == "" {
		return "", nil
	}
	if entry.Auth == "" && (entry.Username != "" || entry.Password != "") {
		entry.Auth = base64.StdEncoding.EncodeToString([]byte(entry.Username + ":" + entry.Password))
	}
	return serverAddr, entry
}
