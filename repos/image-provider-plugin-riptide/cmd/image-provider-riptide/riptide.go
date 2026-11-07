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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ai-on-gke/image-provider-plugin-riptide/internal/proto/riptidekeychain"
	"github.com/ai-on-gke/image-provider-plugin/client"
	"github.com/ai-on-gke/image-provider-plugin/plugin"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	providerName = "riptide"

	// defaultSnapshotterSocket is the socket of the DaemonSet's own
	// containerd-gcfs-grpc.
	defaultSnapshotterSocket = "/run/ate/gcfs-snapshotter.sock"

	// defaultKeychainSocket is the socket of the DaemonSet's own gcfsd
	// keychain service.
	defaultKeychainSocket = "/run/ate/gcfsd/keychain.sock"

	// credsPushTTL is how long a warm hit with unchanged credentials skips
	// UpdateCreds for the same image.
	credsPushTTL = 5 * time.Minute

	// updateCredsTimeout bounds one UpdateCreds call.
	updateCredsTimeout = 5 * time.Second
)

// config configures riptideDriver.
type config struct {
	// KeychainSocket is the gcfsd keychain socket. The driver probes it at
	// startup and dials it for UpdateCreds. Ignored if KeychainClient is set.
	KeychainSocket string

	// KeychainClient replaces the client dialed on KeychainSocket.
	KeychainClient riptidekeychain.KeychainClient

	// DriverOpts are passed to plugin.NewDriver.
	DriverOpts []plugin.Option
}

type pushedCreds struct {
	digest   [32]byte
	pushedAt time.Time
}

// riptideDriver serves images through containerd-gcfs-grpc and pushes each
// image's registry credentials to gcfsd's keychain, which gcfsd uses to
// fetch the layers.
type riptideDriver struct {
	*plugin.Driver

	keychainSocket string
	keychainConn   *grpc.ClientConn
	keychain       riptidekeychain.KeychainClient

	mu     sync.Mutex
	pushed map[string]pushedCreds
}

func newRiptideDriver(cfg config) (*riptideDriver, error) {
	d := &riptideDriver{
		keychain: cfg.KeychainClient,
		pushed:   make(map[string]pushedCreds),
	}
	if d.keychain == nil {
		if cfg.KeychainSocket == "" {
			return nil, errors.New("riptide: no keychain socket")
		}
		conn, err := grpc.NewClient("unix://"+cfg.KeychainSocket,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, fmt.Errorf("riptide: dial keychain %s: %w", cfg.KeychainSocket, err)
		}
		d.keychainSocket = cfg.KeychainSocket
		d.keychainConn = conn
		d.keychain = riptidekeychain.NewKeychainClient(conn)
	}

	opts := append([]plugin.Option{
		plugin.WithLayerMounter(d.mountLayers),
		plugin.WithWarmHitHook(d.onWarmHit),
		plugin.WithProbeHook(d.probe),
	}, cfg.DriverOpts...)
	base, err := plugin.NewDriver(providerName, defaultSnapshotterSocket, "gcfs", opts...)
	if err != nil {
		if d.keychainConn != nil {
			_ = d.keychainConn.Close()
		}
		return nil, err
	}
	d.Driver = base
	return d, nil
}

// probe checks that gcfsd is serving its keychain socket.
func (d *riptideDriver) probe(context.Context) error {
	if d.keychainSocket == "" {
		return nil
	}
	fi, err := os.Stat(d.keychainSocket)
	if err != nil {
		return fmt.Errorf("gcfsd keychain socket not ready: %w", err)
	}
	if fi.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("gcfsd keychain path %s is not a socket", d.keychainSocket)
	}
	return nil
}

func (d *riptideDriver) mountLayers(ctx context.Context, env *plugin.MountEnv) (*plugin.MountedLayers, error) {
	d.pushCreds(ctx, env.CanonicalRef, env.ImageRef, env.AuthConfig, true)
	return plugin.StandardMountLayers(ctx, env)
}

func (d *riptideDriver) onWarmHit(ctx context.Context, canonicalRef string, req *client.PrepareRequest) {
	d.pushCreds(ctx, canonicalRef, req.ImageRef, req.AuthConfig, false)
}

// Close closes the driver and the keychain connection.
func (d *riptideDriver) Close() error {
	err := d.Driver.Close()
	if d.keychainConn != nil {
		err = errors.Join(err, d.keychainConn.Close())
	}
	return err
}

// resolveAuth returns the credentials for imageRef: the request's if it
// has any, else the driver keychain's. It returns nil for anonymous pulls.
func (d *riptideDriver) resolveAuth(ctx context.Context, imageRef string, reqAuth *client.AuthConfig) *riptidekeychain.AuthConfig {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		slog.WarnContext(ctx, "Cannot parse image reference", slog.String("image", imageRef), slog.Any("err", err))
		return nil
	}
	if reqAuth != nil && hasCreds(authn.AuthConfig{
		Username:      reqAuth.Username,
		Password:      reqAuth.Password,
		Auth:          reqAuth.Auth,
		IdentityToken: reqAuth.IdentityToken,
		RegistryToken: reqAuth.RegistryToken,
	}) {
		server := reqAuth.ServerAddress
		if server == "" {
			server = ref.Context().RegistryStr()
		}
		return &riptidekeychain.AuthConfig{
			Username:      reqAuth.Username,
			Password:      reqAuth.Password,
			Auth:          reqAuth.Auth,
			ServerAddress: server,
			IdentityToken: reqAuth.IdentityToken,
			RegistryToken: reqAuth.RegistryToken,
		}
	}
	authenticator, err := d.Keychain().Resolve(ref.Context())
	if err != nil {
		slog.WarnContext(ctx, "Cannot resolve registry credentials", slog.String("image", imageRef), slog.Any("err", err))
		return nil
	}
	if authenticator == authn.Anonymous {
		return nil
	}
	cfg, err := authenticator.Authorization()
	if err != nil {
		slog.WarnContext(ctx, "Cannot get registry credentials", slog.String("image", imageRef), slog.Any("err", err))
		return nil
	}
	if !hasCreds(*cfg) {
		return nil
	}
	return &riptidekeychain.AuthConfig{
		Username:      cfg.Username,
		Password:      cfg.Password,
		Auth:          cfg.Auth,
		ServerAddress: ref.Context().RegistryStr(),
		IdentityToken: cfg.IdentityToken,
		RegistryToken: cfg.RegistryToken,
	}
}

func hasCreds(c authn.AuthConfig) bool {
	return c.Username != "" || c.Password != "" || c.Auth != "" || c.IdentityToken != "" || c.RegistryToken != ""
}

func hashAuth(a *riptidekeychain.AuthConfig) [32]byte {
	return sha256.Sum256([]byte(strings.Join([]string{
		a.GetUsername(),
		a.GetPassword(),
		a.GetAuth(),
		a.GetServerAddress(),
		a.GetIdentityToken(),
		a.GetRegistryToken(),
	}, "\x00")))
}

// pushCreds sends the credentials for imageRef to gcfsd, keyed by
// canonicalRef. Unless force is set, it skips the call if it pushed the
// same credentials for canonicalRef within credsPushTTL. A failed push is
// logged and not returned: gcfsd can still pull public images, and a
// private image fails at read time with a registry error.
func (d *riptideDriver) pushCreds(ctx context.Context, canonicalRef, imageRef string, reqAuth *client.AuthConfig, force bool) {
	auth := d.resolveAuth(ctx, imageRef, reqAuth)
	if auth == nil {
		return
	}
	digest := hashAuth(auth)
	now := time.Now()
	if !force {
		d.mu.Lock()
		prev, ok := d.pushed[canonicalRef]
		d.mu.Unlock()
		if ok && prev.digest == digest && now.Sub(prev.pushedAt) < credsPushTTL {
			return
		}
	}

	rpcCtx, cancel := context.WithTimeout(ctx, updateCredsTimeout)
	defer cancel()
	if _, err := d.keychain.UpdateCreds(rpcCtx, &riptidekeychain.UpdateCredsRequest{
		Image: canonicalRef,
		Auth:  auth,
	}); err != nil {
		slog.WarnContext(ctx, "Failed to push credentials to gcfsd",
			slog.String("image", canonicalRef), slog.Any("err", err))
		return
	}
	d.mu.Lock()
	for ref, p := range d.pushed {
		if now.Sub(p.pushedAt) >= credsPushTTL {
			delete(d.pushed, ref)
		}
	}
	d.pushed[canonicalRef] = pushedCreds{digest: digest, pushedAt: now}
	d.mu.Unlock()
	slog.InfoContext(ctx, "Pushed credentials to gcfsd",
		slog.String("image", canonicalRef), slog.String("server", auth.GetServerAddress()))
}
