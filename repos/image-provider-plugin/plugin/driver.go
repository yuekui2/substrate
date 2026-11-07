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
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/ai-on-gke/image-provider-plugin/client"
	snapshotsapi "github.com/containerd/containerd/api/services/snapshots/v1"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	// DefaultBaseWorkDir is the parent of each provider's default work dir,
	// which holds the per-image layer directories.
	DefaultBaseWorkDir = "/var/lib/ate/image-plugin-cache"

	// DefaultNamespace is the default containerd namespace sent to the
	// snapshotter.
	DefaultNamespace = "default"

	// DefaultListableTimeout is how long PrepareImage waits for a layer view
	// to become listable.
	DefaultListableTimeout = 10 * time.Second

	// DefaultLeaseIdleTTL is how long the driver keeps an image's layer views
	// mounted after its last lease is released.
	DefaultLeaseIdleTTL = 10 * time.Minute

	// DefaultDeclineTTL is how long the driver remembers that the
	// snapshotter declined an image.
	DefaultDeclineTTL = 10 * time.Minute

	// DefaultProbeTimeout bounds one health check of the snapshotter, so a
	// hung snapshotter makes callers fall back quickly.
	DefaultProbeTimeout = 500 * time.Millisecond

	// DefaultStartupTimeout is how long the plugin waits at startup for the
	// provider to pass its health check.
	DefaultStartupTimeout = 2 * time.Minute

	// DefaultDrainTimeout is how long Drain waits for the leases in use to be
	// released.
	DefaultDrainTimeout = 30 * time.Minute
)

// containerdNamespaceHeader is the gRPC metadata key for the containerd
// namespace.
const containerdNamespaceHeader = "containerd-namespace"

// startupPollInterval is the time between health checks at startup.
const startupPollInterval = 200 * time.Millisecond

// MountEnv is what a LayerMounterFunc gets to prepare an image.
type MountEnv struct {
	Driver       *Driver
	Client       snapshotsapi.SnapshotsClient
	Snapshotter  string
	ImageRef     string
	CanonicalRef string
	ImageDigest  string
	ImageWorkDir string
	AuthConfig   *client.AuthConfig
	ChainInfos   []ChainInfo
	LayerDigests []string
}

// MountedLayers is what a LayerMounterFunc returns.
type MountedLayers struct {
	// MountDirs are the read-only host directories of the layers, bottom
	// first. The driver links them as layer-0/fs .. layer-(K-1)/fs.
	MountDirs []string

	// SnapshotKeys are the snapshotter views to remove when the lease ends
	// or preparation fails.
	SnapshotKeys []string

	// CommittedKeys are snapshots the mounter committed itself. The driver
	// removes them if preparation fails.
	CommittedKeys []string
}

// LayerMounterFunc prepares and mounts an image's layers in the snapshotter.
// The default is StandardMountLayers.
type LayerMounterFunc func(ctx context.Context, env *MountEnv) (*MountedLayers, error)

// WarmHitHookFunc runs when PrepareImage reuses a mounted image, for example
// to refresh short-lived credentials.
type WarmHitHookFunc func(ctx context.Context, canonicalRef string, req *client.PrepareRequest)

// WorkDirTeardownHookFunc runs before the driver removes an image's work
// dir, for example to unmount what the mounter mounted inside it.
type WorkDirTeardownHookFunc func(workDir string)

// ImageResolverFunc resolves an image reference to its manifest digest,
// config, layer DiffIDs and layer digests.
type ImageResolverFunc func(ctx context.Context, ref string, auth *client.AuthConfig) (digest string, cfg *v1.Config, diffIDs, layerDigests []string, err error)

// Option configures a Driver.
type Option func(*Driver)

// WithSocketPath sets the snapshotter's Unix socket.
func WithSocketPath(p string) Option {
	return func(d *Driver) { d.socket = p }
}

// WithWorkDir sets the directory for the per-image layer directories.
func WithWorkDir(dir string) Option {
	return func(d *Driver) { d.workDir = dir }
}

// WithNamespace sets the containerd namespace sent to the snapshotter.
func WithNamespace(ns string) Option {
	return func(d *Driver) { d.namespace = ns }
}

// WithSnapshotterName sets the snapshotter name sent in Snapshots requests.
func WithSnapshotterName(name string) Option {
	return func(d *Driver) { d.snapshotterName = name }
}

// WithListableTimeout sets how long PrepareImage waits for a layer view to
// become listable. Zero skips the check.
func WithListableTimeout(timeout time.Duration) Option {
	return func(d *Driver) { d.listableTimeout = timeout }
}

// WithLeaseIdleTTL sets how long an image stays mounted after its last
// lease is released. Zero unmounts it right away.
func WithLeaseIdleTTL(ttl time.Duration) Option {
	return func(d *Driver) { d.leaseIdleTTL = ttl }
}

// WithSnapshotsClient makes the driver use c instead of dialing the
// snapshotter socket.
func WithSnapshotsClient(c snapshotsapi.SnapshotsClient) Option {
	return func(d *Driver) {
		d.snapshotsClient = c
		d.clientInjected = true
	}
}

// WithImageResolver replaces the default resolver, which reads the image
// from its registry.
func WithImageResolver(fn ImageResolverFunc) Option {
	return func(d *Driver) { d.imageResolver = fn }
}

// WithKeychain sets the keychain for registry access when a request carries
// no credentials.
func WithKeychain(k authn.Keychain) Option {
	return func(d *Driver) { d.keychain = k }
}

// WithLayerMounter replaces StandardMountLayers.
func WithLayerMounter(fn LayerMounterFunc) Option {
	return func(d *Driver) { d.layerMounter = fn }
}

// WithWarmHitHook sets a hook that runs when PrepareImage reuses a mounted
// image.
func WithWarmHitHook(fn WarmHitHookFunc) Option {
	return func(d *Driver) { d.warmHitHook = fn }
}

// WithWorkDirTeardownHook sets a hook that runs before an image's work dir is
// removed.
func WithWorkDirTeardownHook(fn WorkDirTeardownHookFunc) Option {
	return func(d *Driver) { d.workDirTeardownHook = fn }
}

// ProbeHookFunc is an extra health check for the provider, for example of a
// local P2P proxy.
type ProbeHookFunc func(ctx context.Context) error

// WithProbeHook adds a health check to the snapshotter check.
func WithProbeHook(fn ProbeHookFunc) Option {
	return func(d *Driver) { d.probeHook = fn }
}

// WithDeclineTTL sets how long the driver remembers that the snapshotter
// declined an image. Zero turns this off.
func WithDeclineTTL(ttl time.Duration) Option {
	return func(d *Driver) { d.declineTTL = ttl }
}

// WithStartupTimeout sets how long the plugin waits at startup for the
// provider to pass its health check. Zero skips the wait.
func WithStartupTimeout(timeout time.Duration) Option {
	return func(d *Driver) { d.startupTimeout = timeout }
}

// WithDrainTimeout sets how long Drain waits for the leases in use to be
// released.
func WithDrainTimeout(timeout time.Duration) Option {
	return func(d *Driver) { d.drainTimeout = timeout }
}

type imageLease struct {
	imageRef     string
	digest       string
	config       *v1.Config
	snapshotKeys []string
	layers       []string
	workDir      string
	refCount     int
	idleTimer    *time.Timer
	evicted      bool
}

type resolvedImageMeta struct {
	digest       string
	config       *v1.Config
	diffIDs      []string
	layerDigests []string
}

// inflightPrep tracks a preparation or teardown of one image. Other calls
// for the image wait for done.
type inflightPrep struct {
	done chan struct{}
	err  error
}

// Driver implements the image lease lifecycle on top of a containerd remote
// snapshotter. It coalesces concurrent PrepareImage calls, counts leases per
// image, keeps idle images mounted for an idle TTL, and links each image's
// layers under <workDir>/<image key>/layer-<i>/fs. A LayerMounterFunc does
// the provider-specific snapshotter work.
type Driver struct {
	name                string
	socket              string
	workDir             string
	namespace           string
	snapshotterName     string
	listableTimeout     time.Duration
	leaseIdleTTL        time.Duration
	imageResolver       ImageResolverFunc
	keychain            authn.Keychain
	layerMounter        LayerMounterFunc
	warmHitHook         WarmHitHookFunc
	workDirTeardownHook WorkDirTeardownHookFunc
	probeHook           ProbeHookFunc
	declineTTL          time.Duration
	startupTimeout      time.Duration
	drainTimeout        time.Duration
	clientInjected      bool

	mu       sync.Mutex
	closed   bool
	draining bool
	// drained is closed once the driver is draining and no lease is in use.
	drained         chan struct{}
	drainedClosed   bool
	conn            *grpc.ClientConn
	snapshotsClient snapshotsapi.SnapshotsClient
	leases          map[string]*imageLease
	// resolvedMeta caches image resolution by ref, so preparing an image
	// again after its lease ended skips the registry. A tag resolves once
	// per driver.
	resolvedMeta map[string]*resolvedImageMeta
	// declined maps images the snapshotter declined to when the driver
	// tries them again.
	declined   map[string]time.Time
	inflight   map[string]*inflightPrep
	chainLocks map[string]*sync.Mutex
}

// NewDriver returns a Driver for the provider name, whose snapshotter
// listens on defaultSocket under snapshotterName. An empty snapshotterName
// means name.
func NewDriver(name, defaultSocket, snapshotterName string, opts ...Option) (*Driver, error) {
	d := &Driver{
		name:            name,
		socket:          defaultSocket,
		workDir:         filepath.Join(DefaultBaseWorkDir, name),
		namespace:       DefaultNamespace,
		snapshotterName: snapshotterName,
		listableTimeout: DefaultListableTimeout,
		leaseIdleTTL:    DefaultLeaseIdleTTL,
		declineTTL:      DefaultDeclineTTL,
		startupTimeout:  DefaultStartupTimeout,
		drainTimeout:    DefaultDrainTimeout,
		drained:         make(chan struct{}),
		leases:          make(map[string]*imageLease),
		resolvedMeta:    make(map[string]*resolvedImageMeta),
		declined:        make(map[string]time.Time),
		inflight:        make(map[string]*inflightPrep),
		chainLocks:      make(map[string]*sync.Mutex),
	}
	for _, opt := range opts {
		opt(d)
	}
	if d.name == "" {
		return nil, errors.New("provider name is required")
	}
	if d.socket == "" && !d.clientInjected {
		return nil, errors.New("snapshotter socket is required")
	}
	if d.workDir == "" {
		return nil, errors.New("work dir is required")
	}
	if d.snapshotterName == "" {
		d.snapshotterName = d.name
	}
	if d.namespace == "" {
		d.namespace = DefaultNamespace
	}
	if d.imageResolver == nil {
		d.imageResolver = d.defaultImageResolver
	}
	if d.layerMounter == nil {
		d.layerMounter = StandardMountLayers
	}
	return d, nil
}

// Name returns the provider name.
func (d *Driver) Name() string {
	return d.name
}

// Keychain returns the keychain set with WithKeychain, or the Google and
// default keychains.
func (d *Driver) Keychain() authn.Keychain {
	if d.keychain != nil {
		return d.keychain
	}
	return authn.NewMultiKeychain(google.Keychain, authn.DefaultKeychain)
}

// WithNamespaceContext adds the containerd namespace header to ctx.
func (d *Driver) WithNamespaceContext(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, containerdNamespaceHeader, d.namespace)
}

// AcquireChainLock returns the mutex for chainID. Hold it around the
// Stat and Prepare of a layer, so images that share base layers do not
// prepare them twice.
func (d *Driver) AcquireChainLock(chainID string) *sync.Mutex {
	d.mu.Lock()
	defer d.mu.Unlock()
	lk, ok := d.chainLocks[chainID]
	if !ok {
		lk = &sync.Mutex{}
		d.chainLocks[chainID] = lk
	}
	return lk
}

// getClient returns the snapshots client, dialing the socket on first use.
func (d *Driver) getClient() (snapshotsapi.SnapshotsClient, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.snapshotsClient != nil {
		return d.snapshotsClient, nil
	}
	conn, err := d.dial()
	if err != nil {
		return nil, err
	}
	d.conn = conn
	d.snapshotsClient = snapshotsapi.NewSnapshotsClient(conn)
	return d.snapshotsClient, nil
}

// dial creates a connection to the snapshotter socket that sends the
// containerd namespace on every call.
func (d *Driver) dial() (*grpc.ClientConn, error) {
	interceptor := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(d.WithNamespaceContext(ctx), method, req, reply, cc, opts...)
	}
	conn, err := grpc.NewClient("unix://"+d.socket,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor),
	)
	if err != nil {
		return nil, fmt.Errorf("creating client for snapshotter socket %s: %w", d.socket, err)
	}
	return conn, nil
}

func (d *Driver) defaultImageResolver(ctx context.Context, refStr string, authConfig *client.AuthConfig) (string, *v1.Config, []string, []string, error) {
	ref, err := name.ParseReference(refStr)
	if err != nil {
		return "", nil, nil, nil, fmt.Errorf("parsing reference %q: %w", refStr, err)
	}
	img, err := remote.Image(ref, RemoteOptions(ctx, d.keychain, authConfig)...)
	if err != nil {
		return "", nil, nil, nil, fmt.Errorf("fetching image: %w", err)
	}
	digest, err := img.Digest()
	if err != nil {
		return "", nil, nil, nil, fmt.Errorf("reading image digest: %w", err)
	}
	cfg, err := img.ConfigFile()
	if err != nil {
		return "", nil, nil, nil, fmt.Errorf("reading image config: %w", err)
	}
	manifest, err := img.Manifest()
	if err != nil {
		return "", nil, nil, nil, fmt.Errorf("reading image manifest: %w", err)
	}
	layerDigests := make([]string, len(manifest.Layers))
	for i, l := range manifest.Layers {
		layerDigests[i] = l.Digest.String()
	}
	diffIDs := make([]string, len(cfg.RootFS.DiffIDs))
	for i, diff := range cfg.RootFS.DiffIDs {
		diffIDs[i] = diff.String()
	}
	return digest.String(), &cfg.Config, diffIDs, layerDigests, nil
}
