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

// Package imagecache implements the node-local OCI image cache: a
// content-addressed pool of unpacked image layers shared by every actor on
// the node, plus the per-bundle overlay spec that tells the ateom runtimes
// how to compose an actor rootfs from cached layers.
//
// The work is split along the existing atelet/ateom privilege boundary:
//
//   - atelet (plain root, all capabilities dropped) pulls layers and unpacks
//     them into the pool (Store.EnsureImage), and writes a rootfs-overlay.json
//     into each bundle (WriteSpec). Whiteout entries are
//     recorded in per-layer metadata rather than materialized, because
//     overlayfs whiteouts are char devices (CAP_MKNOD) with trusted.* xattrs
//     for opaque dirs (CAP_SYS_ADMIN).
//   - ateom (privileged; it already owns every mount on the node) finalizes
//     layers — materializing the recorded whiteout state, once per layer —
//     and mounts the overlay rootfs (SetupBundleRootfs) just before
//     `runsc create` / staging the micro-VM virtio-fs lower.
//
// On-disk layout under the cache root (a directory on the BasePath hostPath,
// so the same absolute paths resolve in atelet and every ateom pod):
//
//	version                          layout version marker
//	layers/sha256/<diffid-hex>/
//	    fs/                          the unpacked layer tree (overlay lowerdir)
//	    whiteouts.json               whiteout state recorded at unpack time
//	    finalized                    marker written by FinalizeLayer (ateom)
//	manifests/sha256/<digest-hex>.json
//	                                 image config + ordered diffID list
//
// Layers land in the pool via unpack-into-tempdir + atomic rename, so a
// layer directory that exists is always complete; startup recovery only has
// to sweep orphaned temp dirs.
package imagecache

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

	"github.com/agent-substrate/substrate/internal/ateattr"
)

const (
	layoutVersion   = "1"
	versionFileName = "version"

	layerFSDirName           = "fs"
	layerWhiteoutsFileName   = "whiteouts.json"
	layerFinalizedMarkerName = "finalized"
	// layerSizeFileName holds the layer's byte count, recorded at unpack so
	// sizing the pool never walks a tree. Absent for layers unpacked by
	// older atelets (backfilled lazily, see layerSize). An estimate: the
	// value is the tar-stream length when written at unpack, or the summed
	// file sizes when backfilled, so it may differ from disk usage and
	// across nodes.
	layerSizeFileName = "size"

	// defaultMinAge is the default eviction minimum age (see WithMinAge).
	defaultMinAge = 2 * time.Minute

	// layerPullConcurrency bounds concurrent layer download+unpack streams per
	// image pull. Memory use is O(stream buffers) per slot, independent of
	// layer size.
	layerPullConcurrency = 8

	// streamChunkSize is the buffer size used by asyncReadCloser and
	// parallelRangeBody to decouple network download, decompression, and tar
	// extraction across goroutines.
	streamChunkSize = 256 * 1024
	// compressedStreamQueueDepth buffers up to 32 MiB of compressed blob data
	// per active layer so network reads continue while decompression or disk
	// writes briefly stall.
	compressedStreamQueueDepth = 128
	// uncompressedStreamQueueDepth buffers up to 16 MiB of uncompressed tar
	// data per active layer so decompression continues while filesystem
	// syscalls (mkdir/open/write/close) execute.
	uncompressedStreamQueueDepth = 64

	// defaultPullTimeout is the default per-pull bound (see WithPullTimeout).
	// Generous enough for multi-GiB images on a busy node.
	defaultPullTimeout = 10 * time.Minute
)

// pullTransport is a tuned HTTP transport for OCI registry pulls:
//   - DisableCompression avoids Go's net/http auto-injecting
//     "Accept-Encoding: gzip" and running a redundant single-threaded gunzip
//     wrapper on blobs that are already compressed (.tar.gz / .tar.zst).
//   - ForceAttemptHTTP2 is disabled so concurrent layer and HTTP Range chunk
//     downloads open independent HTTP/1.1 TCP connections instead of
//     multiplexing all streams over a single TCP connection (avoiding single-
//     connection TCP head-of-line blocking and flow-control window caps).
//   - ReadBufferSize is raised to 256 KiB to reduce socket read syscall
//     overhead on multi-hundred-megabyte layer blobs.
var pullTransport http.RoundTripper = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	ForceAttemptHTTP2:     false,
	TLSNextProto:          make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
	MaxIdleConns:          128,
	MaxIdleConnsPerHost:   64,
	MaxConnsPerHost:       64,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	ResponseHeaderTimeout: 30 * time.Second,
	ReadBufferSize:        256 * 1024,
	WriteBufferSize:       64 * 1024,
	DisableCompression:    true,
}

var pullRangeTransport = newParallelRangeTransport(pullTransport)

// Pull retry backoffs, chosen per registry by retryBackoffFor. Both replace
// go-containerregistry's default (3 attempts over ~4s, jitter 0.1), which is
// tuned for a single flaky client, and both apply to every request of a pull,
// including the /v2/ auth ping. The retryable status codes stay at the
// library default, which already includes 429. Vars so tests can shrink the
// waits.
//
// The backoff applies per request, so under a sustained throttle a
// multi-layer pull can accumulate far more wall clock than one request's
// worst case (~14s shared). The caller's ctx still bounds the total: every
// request carries it (remote.WithContext in remoteOpts), a canceled ctx
// fails the next attempt immediately and is never retried, so retrying
// overshoots a deadline by at most one backoff sleep. Production pulls run
// under the Run/Restore RPC ctx, which ateapi's resume path caps at its
// server-wide max RPC deadline — a throttled shared-registry pull surfaces
// as that RPC's deadline error, not a hang.
var (
	// dedicatedRegistryBackoff covers registries where the deployment has
	// its own quota (Artifact Registry, ECR, Harbor, self-hosted, …): the
	// server absorbs retry bursts and nobody else competes for the limit,
	// and pulls sit on the Run/Restore critical path, so retries start fast
	// and come often — more attempts in less total wall clock, still quick
	// enough to surface a real outage to the RPC-level retry.
	dedicatedRegistryBackoff = remote.Backoff{
		Duration: 200 * time.Millisecond,
		Factor:   2.0,
		Jitter:   1.0,
		Steps:    4,
		Cap:      2 * time.Second,
	}
	// sharedRegistryBackoff covers communal registries (registry.k8s.io,
	// Docker Hub, …), where pulls are anonymous and rate limits are shared:
	// a whole fleet can be throttled at once, so retries must outlast the
	// throttling wave and full jitter must de-synchronize the herd rather
	// than replay it.
	sharedRegistryBackoff = remote.Backoff{
		Duration: 1 * time.Second,
		Factor:   2.0,
		Jitter:   1.0,
		Steps:    4,
		Cap:      10 * time.Second,
	}
)

// sharedRegistries are the well-known communal registries whose rate limits
// are shared across all anonymous clients. Everything not listed here is
// assumed dedicated: fleet-scale pulls realistically hit either the pause
// image (registry.k8s.io) or the customer's own registry, and enumerating
// the small stable set of communal hosts beats guessing at every private
// registry vendor.
var sharedRegistries = map[string]bool{
	"registry.k8s.io": true,
	// Pulls only ever present index.docker.io here: name.ParseReference
	// normalizes docker.io (and bare refs like "ubuntu") to it before
	// retryBackoffFor runs. The docker.io entry is kept so a lookup by the
	// canonical name classifies the same way.
	"docker.io":            true,
	"index.docker.io":      true,
	"registry-1.docker.io": true,
	"quay.io":              true,
	"ghcr.io":              true,
	"public.ecr.aws":       true,
	"mcr.microsoft.com":    true,
	"registry.gitlab.com":  true,
	"cgr.dev":              true,
}

// retryBackoffFor picks the pull retry backoff for a registry host.
func retryBackoffFor(registry string) remote.Backoff {
	if sharedRegistries[registry] {
		return sharedRegistryBackoff
	}
	return dedicatedRegistryBackoff
}

// Store is atelet's handle to the on-disk layer pool. It is safe for
// concurrent use; concurrent pulls of the same image or layer are collapsed.
// The store assumes it is the only writer on the node (one atelet per node).
type Store struct {
	root string

	// keychain, when set, supplies credentials for pulls. See withCredentials.
	keychain authn.Keychain

	localhostRegistryReplacement string

	// platform overrides the default pull platform (linux/GOARCH), for
	// callers pulling on a different architecture than the images' target.
	platform *v1.Platform

	// actorsDir is scanned by InUse for bundle overlay specs; empty disables
	// the scan (the root set is then empty).
	actorsDir string

	// minAge vetoes eviction of any layer or image record younger than this,
	// covering the window between a pull (or cache-hit stat) and the bundle
	// spec write / ateom mount that roots it.
	minAge time.Duration

	// pullTimeout bounds each pull. Pulls run detached from the contexts of
	// the callers waiting on them (see EnsureImage), so this is the only
	// bound on how long one can run.
	pullTimeout time.Duration

	// meter, when set, is the meter the store reports on. See WithMeter.
	meter metric.Meter

	// requests counts EnsureImage lookups by outcome. Nil without a meter,
	// which recordRequest treats as a no-op.
	requests metric.Int64Counter

	imageSF singleflight.Group
	layerSF singleflight.Group

	// evictMu serializes EvictUnused passes (concurrent passes would fight
	// over the same candidates for no benefit).
	evictMu sync.Mutex

	// hitMu closes the hit-vs-evict window: held shared by the hit path
	// (cachedImageHit), exclusive by eviction's record removal
	// (removeStaleRecord), so a hit's last-use touch and eviction's final
	// re-check can never interleave. Uncontended except during a pass.
	hitMu sync.RWMutex
}

// Option configures a Store.
type Option func(*Store)

// CandidateKeychain returns several credentials to try in order.
type CandidateKeychain interface {
	authn.Keychain
	Candidates(ctx context.Context, target authn.Resource) ([]authn.Authenticator, error)
}

// WithKeychain attaches a keychain consulted for every pull, whatever the
// registry — the keychain itself decides which registries it has credentials
// for. A nil keychain is ignored.
func WithKeychain(k authn.Keychain) Option {
	return func(s *Store) { s.keychain = k }
}

// WithLocalhostRegistryReplacement rewrites localhost/loopback registry refs
// to the given endpoint, mirroring the containerd mirror config used by kind
// local registries (https://kind.sigs.k8s.io/docs/user/local-registry/).
func WithLocalhostRegistryReplacement(replacement string) Option {
	return func(s *Store) { s.localhostRegistryReplacement = replacement }
}

// WithPlatform overrides the pull platform (default: linux/GOARCH).
func WithPlatform(p v1.Platform) Option {
	return func(s *Store) { s.platform = &p }
}

// WithActorsDir points the eviction root-set scan at the node's actors
// directory (the per-actor state dirs under nodepath.BasePath). Each
// <actorsDir>/<actorUID>/bundles/<container>/rootfs-overlay.json roots its
// image and layers against eviction. Empty disables the scan.
func WithActorsDir(dir string) Option {
	return func(s *Store) { s.actorsDir = dir }
}

// WithMinAge overrides the eviction minimum age (default 2m): layers and
// image records younger than this are never evicted.
func WithMinAge(d time.Duration) Option {
	return func(s *Store) { s.minAge = d }
}

// WithPullTimeout overrides the per-pull timeout (default 10m).
func WithPullTimeout(d time.Duration) Option {
	return func(s *Store) { s.pullTimeout = d }
}

// WithMeter attaches the meter the store reports ate.imagecache.requests on.
// Without it the store records nothing, so a caller with no metrics pipeline
// needs no meter provider.
func WithMeter(m metric.Meter) Option {
	return func(s *Store) { s.meter = m }
}

// Image describes one cached, ready-to-compose image.
type Image struct {
	// Digest is the manifest digest the caller's ref resolved to (for a
	// multi-arch ref, the index digest as requested, not the per-platform
	// child).
	Digest v1.Hash
	// Config is the OCI image config (entrypoint, env, ...).
	Config v1.Config
	// LayerDirs are the absolute cached layer directories, bottom-most layer
	// first. Each contains the unpacked tree under "fs/".
	LayerDirs []string
}

// imageRecord is the persisted form of a cached image, stored under
// manifests/<algorithm>/<hex>.json.
type imageRecord struct {
	Version int       `json:"version"`
	Config  v1.Config `json:"config"`
	DiffIDs []string  `json:"diffIDs"`
}

// New opens (creating if needed) the layer pool rooted at root and runs
// startup recovery: verifying the layout version and sweeping temp dirs left
// by unpacks that were in flight when a previous atelet died.
func New(root string, opts ...Option) (*Store, error) {
	s := &Store{root: root, minAge: defaultMinAge, pullTimeout: defaultPullTimeout}
	for _, o := range opts {
		o(s)
	}

	if s.meter != nil {
		requests, err := newRequestsCounter(s.meter)
		if err != nil {
			return nil, err
		}
		s.requests = requests
	}

	for _, d := range []string{s.layersDir(), s.manifestsDir()} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return nil, fmt.Errorf("while creating image cache dir %q: %w", d, err)
		}
	}

	versionPath := filepath.Join(root, versionFileName)
	switch b, err := os.ReadFile(versionPath); {
	case err == nil:
		if got := strings.TrimSpace(string(b)); got != layoutVersion {
			// Fail loudly instead of silently mixing layouts; an operator can
			// delete the cache dir to rebuild it (it holds no unique state).
			return nil, fmt.Errorf("image cache at %q has layout version %q, this atelet supports %q", root, got, layoutVersion)
		}
	case errors.Is(err, os.ErrNotExist):
		if err := os.WriteFile(versionPath, []byte(layoutVersion+"\n"), 0o600); err != nil {
			return nil, fmt.Errorf("while writing image cache version marker: %w", err)
		}
	default:
		return nil, fmt.Errorf("while reading image cache version marker: %w", err)
	}

	if err := s.sweepTempDirs(); err != nil {
		return nil, err
	}
	// Startup-only orphan recovery (see RecoverOrphans for why it must not
	// run during normal operation). Never fatal: a corrupt record must not
	// keep atelet from serving actors. Gated means nothing was attempted
	// (orphans persist until the named file is repaired and the next
	// restart); per-item errors mean the scan ran and reclaimed what it
	// could.
	if _, err := s.RecoverOrphans(context.Background()); err != nil {
		if errors.Is(err, ErrIncompleteEnumeration) {
			slog.Error("Image cache startup orphan scan skipped", slog.Any("err", err))
		} else {
			slog.Warn("Image cache startup orphan recovery incomplete", slog.Any("err", err))
		}
	}
	return s, nil
}

func (s *Store) layersDir() string    { return filepath.Join(s.root, "layers", "sha256") }
func (s *Store) manifestsDir() string { return filepath.Join(s.root, "manifests", "sha256") }

func (s *Store) layerDir(diffID v1.Hash) string {
	return filepath.Join(s.root, "layers", diffID.Algorithm, diffID.Hex)
}

func (s *Store) recordPath(digest v1.Hash) string {
	return filepath.Join(s.root, "manifests", digest.Algorithm, digest.Hex+".json")
}

// sweepTempDirs removes unpack temp dirs, retired layer dirs, and
// manifest-record temp files orphaned by a crash. A layer dir without the
// temp/retired prefix and a record without a leading dot are always
// complete (both are moved into place with a single rename), so this is
// the only recovery the pool needs.
func (s *Store) sweepTempDirs() error {
	entries, err := os.ReadDir(s.layersDir())
	if err != nil {
		return fmt.Errorf("while listing layer pool: %w", err)
	}
	swept := 0
	for _, e := range entries {
		// ".tmp-": an unpack in flight at crash. ".rm-": a layer retirement
		// renamed aside but not yet removed. Either way, unreferenced
		// garbage.
		if !strings.HasPrefix(e.Name(), ".tmp-") && !strings.HasPrefix(e.Name(), retiredPrefix) {
			continue
		}
		p := filepath.Join(s.layersDir(), e.Name())
		slog.Info("Image cache sweeping orphaned layer dir", slog.String("dir", e.Name()))
		if err := RemoveAllWritable(p); err != nil {
			return fmt.Errorf("while sweeping orphaned layer dir %q: %w", p, err)
		}
		swept++
	}
	if swept > 0 {
		slog.Info("Image cache startup sweep removed orphaned layer dirs", slog.Int("count", swept))
	}

	// writeRecord's temp files are ".<hex>.json.tmp-<rand>"; finished records
	// are "<hex>.json", so a leading dot alone identifies an orphan.
	records, err := os.ReadDir(s.manifestsDir())
	if err != nil {
		return fmt.Errorf("while listing manifest records: %w", err)
	}
	for _, e := range records {
		if !strings.HasPrefix(e.Name(), ".") {
			continue
		}
		p := filepath.Join(s.manifestsDir(), e.Name())
		if err := os.Remove(p); err != nil {
			return fmt.Errorf("while sweeping orphaned manifest temp file %q: %w", p, err)
		}
	}
	return nil
}

// EnsureImage makes ref's image available in the pool and returns its config
// and ordered layer directories. Digest refs hit the cache with no network
// I/O; tag refs cost one HEAD request to resolve the tag to a manifest
// digest (so tag refs are cacheable, and a moved tag is picked up on the
// next call).
func (s *Store) EnsureImage(ctx context.Context, ref string) (_ *Image, err error) {
	// A miss until a complete record proves otherwise; recordRequest
	// reclassifies a failure onto its own outcome.
	outcome := ateattr.ImageCacheOutcomeMiss
	defer func() { s.recordRequest(ctx, outcome, err) }()

	parsedRef, err := s.parseRef(ref)
	if err != nil {
		return nil, fmt.Errorf("while parsing reference: %w", err)
	}

	var digest v1.Hash
	if d, ok := parsedRef.(name.Digest); ok {
		digest, err = v1.NewHash(d.DigestStr())
		if err != nil {
			return nil, fmt.Errorf("while parsing digest of %q: %w", ref, err)
		}
	} else {
		// Tag ref: one small HEAD request pins it to an immutable manifest
		// digest, which is the only safe cache key for mutable tags.
		var desc *v1.Descriptor
		headErr := s.withCredentials(ctx, parsedRef, func(opts []remote.Option) (err error) {
			desc, err = remote.Head(parsedRef, opts...)
			return err
		})
		if headErr != nil {
			err = fmt.Errorf("while resolving tag %q to a digest: %w", ref, headErr)
			return nil, err
		}
		digest = desc.Digest
	}

	img, err := s.cachedImageHit(digest)
	if err != nil {
		return nil, err
	}
	if img != nil {
		outcome = ateattr.ImageCacheOutcomeHit
		slog.InfoContext(ctx, "Image cache hit", slog.String("ref", ref), slog.String("digest", digest.String()))
		return img, nil
	}
	slog.InfoContext(ctx, "Image cache miss", slog.String("ref", ref), slog.String("digest", digest.String()))

	// Collapse concurrent pulls of the same digest (e.g. several containers of
	// one actor, or several actors landing at once). Callers with very
	// different lifetimes share these flights — a serving Restore, a
	// best-effort prewarm bounded by its own deadline, a caller whose daemon
	// is draining — so the pull runs on a context detached from whichever
	// caller happened to start the flight, bounded only by the store's pull
	// timeout: a waiter must only ever see a real pull failure, never another
	// caller's cancellation. Each caller stops waiting when its own ctx ends,
	// while the pull runs on to warm the cache for the next attempt.
	ch := s.imageSF.DoChan(digest.String(), func() (any, error) {
		pullCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.pullTimeout)
		defer cancel()
		return s.pull(pullCtx, parsedRef, digest)
	})
	select {
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		return res.Val.(*Image), nil
	case <-ctx.Done():
		return nil, fmt.Errorf("while waiting for pull of %s: %w", digest, context.Cause(ctx))
	}
}

// cachedImageHit is the hit side of the hitMu contract: it verifies the
// cached image and records last-use for eviction's LRU ordering, atomic
// with respect to eviction's record removal (removeStaleRecord holds
// hitMu exclusive). Refreshing the mtime also renews the min-age veto, so
// an image in active use cannot age into eviction between this stat and
// the ateom's mount.
func (s *Store) cachedImageHit(digest v1.Hash) (*Image, error) {
	s.hitMu.RLock()
	defer s.hitMu.RUnlock()
	img, err := s.cachedImage(digest)
	if err == nil && img != nil {
		s.touchRecord(digest)
	}
	return img, err
}

// cachedImage returns the cached image for digest, or nil if the record or
// any of its layer dirs is missing (in which case the caller re-pulls; only
// the missing layers cost anything).
func (s *Store) cachedImage(digest v1.Hash) (*Image, error) {
	b, err := os.ReadFile(s.recordPath(digest))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("while reading image record for %s: %w", digest, err)
	}
	var rec imageRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return nil, fmt.Errorf("while decoding image record for %s: %w", digest, err)
	}

	layerDirs := make([]string, len(rec.DiffIDs))
	for i, d := range rec.DiffIDs {
		diffID, err := v1.NewHash(d)
		if err != nil {
			return nil, fmt.Errorf("invalid diffID %q in image record for %s: %w", d, digest, err)
		}
		dir := s.layerDir(diffID)
		if _, err := os.Stat(filepath.Join(dir, layerFSDirName)); err != nil {
			return nil, nil
		}
		layerDirs[i] = dir
	}
	return &Image{Digest: digest, Config: rec.Config, LayerDirs: layerDirs}, nil
}

// pull fetches the image (by its resolved digest, so what is unpacked is
// exactly what was recorded), writes the image record, and then unpacks
// every missing layer into the pool.
//
// The record comes first so that every layer is referenced — and thereby
// safe from eviction — before it can exist on disk. A record therefore
// means "known image, possibly partially present", which readers already
// handle: cachedImage verifies every layer and re-pulls what is missing,
// so an interrupted pull's record is just resumable progress.
func (s *Store) pull(ctx context.Context, parsedRef name.Reference, digest v1.Hash) (*Image, error) {
	// Re-check under the flight lock: a racing EnsureImage may have completed
	// the pull between our cache miss and winning the singleflight slot.
	if img, err := s.cachedImage(digest); err != nil {
		return nil, err
	} else if img != nil {
		return img, nil
	}

	// Retry the whole pull per credential: the config and layer blobs are
	// fetched after the manifest, and finished layers are reused.
	var out *Image
	err := s.withCredentials(ctx, parsedRef, func(opts []remote.Option) (err error) {
		out, err = s.pullWith(ctx, parsedRef, digest, opts)
		return err
	})
	return out, err
}

// pullWith is pull with the given remote options.
func (s *Store) pullWith(ctx context.Context, parsedRef name.Reference, digest v1.Hash, opts []remote.Option) (*Image, error) {
	tStart := time.Now()
	digestRef := parsedRef.Context().Digest(digest.String())
	img, err := remote.Image(digestRef, opts...)
	if err != nil {
		return nil, fmt.Errorf("in remote.Image: %w", err)
	}

	cfgFile, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("while reading image config: %w", err)
	}

	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("while listing image layers: %w", err)
	}

	if len(cfgFile.RootFS.DiffIDs) != len(layers) {
		return nil, fmt.Errorf("image %s config lists %d diffIDs but manifest has %d layers", digest, len(cfgFile.RootFS.DiffIDs), len(layers))
	}
	diffIDs := make([]string, len(layers))
	for i, d := range cfgFile.RootFS.DiffIDs {
		diffIDs[i] = d.String()
	}

	// The full diffID list is known from the config before any unpack; make
	// every layer of this pull referenced before it can exist.
	rec := imageRecord{Version: 1, Config: cfgFile.Config, DiffIDs: diffIDs}
	if err := s.writeRecord(digest, rec); err != nil {
		return nil, err
	}
	// For a multi-arch ref the requested digest is the index digest, but the
	// layers unpacked belong to the per-platform child manifest. Record the
	// image under the child digest too, so refs pinned either way hit.
	var actualDigest *v1.Hash
	if actual, err := img.Digest(); err == nil && actual != digest {
		actualDigest = &actual
		if err := s.writeRecord(actual, rec); err != nil {
			slog.WarnContext(ctx, "Failed to record image under platform manifest digest",
				slog.String("digest", actual.String()), slog.Any("err", err))
		}
	}

	// Dispatch layers in Longest-Processing-Time (LPT) order so that the
	// heaviest layers start downloading and unpacking immediately rather than
	// getting queued behind small layers when len(layers) > layerPullConcurrency.
	order := make([]int, len(layers))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		sa, _ := layers[order[a]].Size()
		sb, _ := layers[order[b]].Size()
		return sa > sb
	})

	layerDirs := make([]string, len(layers))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(layerPullConcurrency)
	for _, idx := range order {
		i, layer := idx, layers[idx]
		g.Go(func() error {
			diffID, err := layer.DiffID()
			if err != nil {
				return fmt.Errorf("while reading layer diffID: %w", err)
			}
			if diffID.String() != diffIDs[i] {
				// The record must reference exactly what lands on disk.
				return fmt.Errorf("layer %d diffID %s does not match config rootfs diffID %s", i, diffID, diffIDs[i])
			}
			dir, err := s.ensureLayer(gctx, diffID, layer)
			if err != nil {
				return fmt.Errorf("while unpacking layer %s: %w", diffID, err)
			}
			layerDirs[i] = dir
			// Each completed layer refreshes the record's mtime: a pull
			// making progress stays fresh indefinitely; a wedged one ages
			// into ordinary LRU eviction. The twin is not touched — the
			// primary keeps the layers referenced, and the final rewrite
			// recreates the twin if it ages out mid-pull.
			s.touchRecord(digest)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Rewrite the record: eviction may legitimately remove it mid-pull
	// (no progress for min-age reads as wedged), and success must never
	// leave the just-unpacked layers unreferenced.
	if err := s.writeRecord(digest, rec); err != nil {
		return nil, fmt.Errorf("while rewriting image record after unpack: %w", err)
	}
	if actualDigest != nil {
		if err := s.writeRecord(*actualDigest, rec); err != nil {
			slog.WarnContext(ctx, "Failed to rewrite platform manifest record after unpack",
				slog.String("digest", actualDigest.String()), slog.Any("err", err))
		}
	}

	// Never return LayerDirs that are not on disk right now: a vanished
	// dir fails the pull into a clean RPC retry instead of a bundle spec
	// naming a missing lowerdir. Ordered after the rewrite so even this
	// failure path leaves the surviving layers referenced.
	for _, dir := range layerDirs {
		if _, err := os.Stat(filepath.Join(dir, layerFSDirName)); err != nil {
			return nil, fmt.Errorf("layer dir vanished during pull (evicted?): %w", err)
		}
	}

	slog.InfoContext(ctx, "Image pulled into layer cache",
		slog.String("digest", digest.String()),
		slog.Int("layers", len(layers)),
		slog.Duration("took", time.Since(tStart)))

	return &Image{Digest: digest, Config: cfgFile.Config, LayerDirs: layerDirs}, nil
}

// ensureLayer makes the unpacked tree for diffID present in the pool,
// collapsing concurrent requests for the same layer across images.
func (s *Store) ensureLayer(ctx context.Context, diffID v1.Hash, layer v1.Layer) (string, error) {
	dir := s.layerDir(diffID)
	_, err, _ := s.layerSF.Do(layerFlightKey(diffID.Hex), func() (any, error) {
		if _, err := os.Stat(filepath.Join(dir, layerFSDirName)); err == nil {
			// Refresh the dir mtime inside the flight: retireLayer re-checks
			// the mtime in this same flight, so a layer reused here can
			// never be renamed away between this stat and the image record
			// that will re-reference it.
			now := time.Now()
			if err := os.Chtimes(dir, now, now); err != nil {
				slog.WarnContext(ctx, "Failed to refresh layer mtime on reuse", slog.String("diffid", diffID.String()), slog.Any("err", err))
			}
			return nil, nil
		}
		return nil, s.unpackLayerToPool(ctx, diffID, layer)
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

// unpackLayerToPool streams the layer (download → decompress → untar) into a
// temp dir and renames it into place, so a layer dir either exists complete
// or not at all.
func (s *Store) unpackLayerToPool(ctx context.Context, diffID v1.Hash, layer v1.Layer) (retErr error) {
	tmp, err := os.MkdirTemp(filepath.Dir(s.layerDir(diffID)), ".tmp-"+diffID.Hex[:12]+"-")
	if err != nil {
		return fmt.Errorf("while creating layer temp dir: %w", err)
	}
	defer func() {
		// No-op once the rename has moved tmp into place.
		if _, err := os.Stat(tmp); err == nil {
			if rmErr := RemoveAllWritable(tmp); rmErr != nil && retErr == nil {
				retErr = fmt.Errorf("while cleaning up layer temp dir: %w", rmErr)
			}
		}
	}()

	fsDir := filepath.Join(tmp, layerFSDirName)
	if err := os.Mkdir(fsDir, 0o755); err != nil {
		return fmt.Errorf("while creating layer fs dir: %w", err)
	}

	rc, err := openLayerStream(ctx, layer)
	if err != nil {
		return fmt.Errorf("while opening layer stream: %w", err)
	}
	defer rc.Close()

	root, err := os.OpenRoot(fsDir)
	if err != nil {
		return fmt.Errorf("while opening layer fs dir as os.Root: %w", err)
	}
	defer root.Close()

	// The uncompressed tar stream is the recorded size: an optimistic
	// estimate (tar framing vs. block rounding), cheap to capture here.
	cr := &countingReader{r: rc}
	wh, err := unpackLayer(ctx, cr, root)
	if err != nil {
		return err
	}
	// Drain any trailing zero-padding blocks after the tar EOF marker and
	// close the stream so that go-containerregistry's digest verification
	// wrapper (which triggers at EOF of the underlying compressed stream)
	// runs now and catches any corrupted blob before we commit the layer.
	if _, err := io.Copy(io.Discard, cr); err != nil {
		return fmt.Errorf("while verifying layer stream at EOF: %w", err)
	}
	if err := rc.Close(); err != nil {
		return fmt.Errorf("while closing layer stream: %w", err)
	}
	// Non-fatal: a missing size file is recovered by layerSize's backfill,
	// so a metadata write must not discard a successful unpack.
	if err := os.WriteFile(filepath.Join(tmp, layerSizeFileName), []byte(strconv.FormatInt(cr.n, 10)+"\n"), 0o600); err != nil {
		slog.WarnContext(ctx, "Failed to record layer size; will backfill lazily",
			slog.String("diffid", diffID.String()), slog.Any("err", err))
	}

	whBytes, err := json.Marshal(wh)
	if err != nil {
		return fmt.Errorf("while encoding layer whiteouts: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, layerWhiteoutsFileName), whBytes, 0o600); err != nil {
		return fmt.Errorf("while writing layer whiteouts: %w", err)
	}

	if err := os.Rename(tmp, s.layerDir(diffID)); err != nil {
		// A concurrent unpack (another process sharing the pool) may have won;
		// its layer is as good as ours.
		if _, statErr := os.Stat(filepath.Join(s.layerDir(diffID), layerFSDirName)); statErr == nil {
			return nil
		}
		return fmt.Errorf("while moving layer into pool: %w", err)
	}
	return nil
}

// streamChunkPool recycles streamChunkSize byte slices used by asyncReadCloser
// and parallelRangeBody to decouple network download, decompression, and tar
// extraction without allocating tens of megabytes of short-lived buffers per
// layer.
var streamChunkPool = sync.Pool{
	New: func() any {
		b := make([]byte, streamChunkSize)
		return &b
	},
}

type streamChunk struct {
	buf *[]byte
	n   int
}

// asyncReadCloser runs a background goroutine that continuously drains src into
// a bounded queue of pooled chunks so the producer (e.g. HTTP download + sha256
// verification, or gzip/zstd decompression) stays ahead of the consumer (e.g.
// decompression or filesystem syscalls).
type asyncReadCloser struct {
	src       io.ReadCloser
	ch        chan streamChunk
	done      chan struct{}
	curr      *[]byte
	pending   []byte
	err       error
	closeOnce sync.Once
	closeErr  error
}

func newAsyncReadCloser(src io.ReadCloser, queueDepth int) *asyncReadCloser {
	a := &asyncReadCloser{
		src:  src,
		ch:   make(chan streamChunk, queueDepth),
		done: make(chan struct{}),
	}
	go a.produce()
	return a
}

func (a *asyncReadCloser) produce() {
	for {
		bufPtr := streamChunkPool.Get().(*[]byte)
		buf := *bufPtr
		n, err := io.ReadFull(a.src, buf)
		if errors.Is(err, io.ErrUnexpectedEOF) {
			err = io.EOF
		}
		if n > 0 {
			select {
			case a.ch <- streamChunk{buf: bufPtr, n: n}:
			case <-a.done:
				streamChunkPool.Put(bufPtr)
				return
			}
		} else {
			streamChunkPool.Put(bufPtr)
		}
		if err != nil {
			a.err = err
			close(a.ch)
			return
		}
	}
}

func (a *asyncReadCloser) Read(p []byte) (int, error) {
	if len(a.pending) == 0 {
		if a.curr != nil {
			streamChunkPool.Put(a.curr)
			a.curr = nil
		}
		chunk, ok := <-a.ch
		if !ok {
			if a.err != nil {
				return 0, a.err
			}
			return 0, io.EOF
		}
		a.curr = chunk.buf
		a.pending = (*chunk.buf)[:chunk.n]
	}
	n := copy(p, a.pending)
	a.pending = a.pending[n:]
	return n, nil
}

func (a *asyncReadCloser) Close() error {
	a.closeOnce.Do(func() {
		close(a.done)
		a.closeErr = a.src.Close()
		if a.curr != nil {
			streamChunkPool.Put(a.curr)
			a.curr = nil
			a.pending = nil
		}
		for chunk := range a.ch {
			streamChunkPool.Put(chunk.buf)
		}
	})
	return a.closeErr
}

// gzipEmptyStream is a valid 20-byte gzip stream encoding 0 uncompressed bytes,
// used once at startup to verify that a discovered igzip binary and its shared
// libraries execute cleanly before enabling the external fast-path.
var gzipEmptyStream = []byte{
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff,
	0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

// buildIgzipCommand constructs the exec.Cmd to run igzip -d -c for igzipPath.
// When igzipPath resides under a host root mount (/proc/1/root or /host) inside
// a container that does not carry libisal.so.2, it invokes the host's own
// ELF interpreter (ld-linux) with --library-path pointing at the host's lib
// directories so execution is self-contained.
func buildIgzipCommand(ctx context.Context, igzipPath string) *exec.Cmd {
	for _, prefix := range []string{"/proc/1/root", "/host"} {
		if strings.HasPrefix(igzipPath, prefix+"/") {
			libDirs := strings.Join([]string{
				prefix + "/usr/lib64",
				prefix + "/lib64",
				prefix + "/usr/lib/x86_64-linux-gnu",
				prefix + "/lib/x86_64-linux-gnu",
				prefix + "/usr/lib/aarch64-linux-gnu",
				prefix + "/lib/aarch64-linux-gnu",
				prefix + "/usr/lib",
				prefix + "/lib",
			}, ":")
			for _, loader := range []string{
				prefix + "/lib64/ld-linux-x86-64.so.2",
				prefix + "/lib/x86_64-linux-gnu/ld-linux-x86-64.so.2",
				prefix + "/lib/ld-linux-aarch64.so.1",
				prefix + "/lib64/ld-linux-aarch64.so.1",
			} {
				if st, err := os.Stat(loader); err == nil && !st.IsDir() {
					return exec.CommandContext(ctx, loader, "--library-path", libDirs, igzipPath, "-d", "-c")
				}
			}
		}
	}
	return exec.CommandContext(ctx, igzipPath, "-d", "-c")
}

func verifyIgzipBinary(igzipPath string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := buildIgzipCommand(ctx, igzipPath)
	cmd.Stdin = bytes.NewReader(gzipEmptyStream)
	out, err := cmd.Output()
	return err == nil && len(out) == 0
}

// igzipBinaryPath resolves the path to Intel ISA-L's hardware-accelerated
// igzip binary once per process. When present (e.g. pre-installed on GKE
// Container-Optimized OS nodes at /usr/bin/igzip, or via the isal package on
// Debian/Ubuntu/Alpine), igzip uses AVX2/AVX-512/NEON SIMD instructions to
// decompress .tar.gz streams at 300–400 MB/s compressed (~1 GB/s
// uncompressed) on a single core (~4x faster than pure-Go gzip).
var igzipBinaryPath = sync.OnceValue(func() string {
	if os.Getenv("ATE_DISABLE_IGZIP") != "" {
		return ""
	}
	var candidates []string
	if p, err := exec.LookPath("igzip"); err == nil {
		candidates = append(candidates, p)
	}
	candidates = append(candidates,
		"/usr/bin/igzip",
		"/bin/igzip",
		"/usr/local/bin/igzip",
		"/var/lib/ate/bin/igzip",
		"/proc/1/root/usr/bin/igzip",
		"/host/usr/bin/igzip",
	)
	for _, candidate := range candidates {
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			if verifyIgzipBinary(candidate) {
				return candidate
			}
		}
	}
	return ""
})

// openLayerStream opens an uncompressed tar stream for layer using a 3-stage
// concurrent pipeline:
//
//  1. Stage 1 (network + sha256): for layers >= rangeFetchThreshold (16 MiB),
//     parallelRangeTransport fetches 16 MiB chunks concurrently over HTTP
//     Range requests into a bounded sliding window; stage1 drains
//     layer.Compressed() (running sha256 verification on its own goroutine)
//     into a 32 MiB read-ahead buffer.
//  2. Stage 2 (decompression): decompresses via igzip (ISA-L SIMD fast-path
//     when available on the host/container) or klauspost/compress (gzip/zstd)
//     into a 16 MiB uncompressed buffer on its own goroutine so decompression
//     runs in parallel with both network download and tar extraction.
//
// Falls back to layer.Uncompressed() for uncompressed or unknown media types.
func openLayerStream(ctx context.Context, layer v1.Layer) (io.ReadCloser, error) {
	return openLayerStreamWithIgzip(ctx, layer, igzipBinaryPath())
}

func openLayerStreamWithIgzip(ctx context.Context, layer v1.Layer, igzipPath string) (io.ReadCloser, error) {
	if size, err := layer.Size(); err == nil && size >= rangeFetchThreshold {
		if dgst, err := layer.Digest(); err == nil {
			registerLargeBlob(dgst.String(), size)
		}
	}

	mt, err := layer.MediaType()
	if err != nil {
		return layer.Uncompressed()
	}
	switch mt {
	case types.DockerLayer, types.OCILayer:
		compRC, err := layer.Compressed()
		if err != nil {
			return nil, err
		}
		stage1 := newAsyncReadCloser(compRC, compressedStreamQueueDepth)
		if igzipPath != "" {
			if igzRC, err := newIgzipReader(ctx, igzipPath, stage1); err == nil {
				return newAsyncReadCloser(igzRC, uncompressedStreamQueueDepth), nil
			}
		}
		gz, err := gzip.NewReader(stage1)
		if err != nil {
			_ = stage1.Close()
			return nil, fmt.Errorf("while creating gzip reader: %w", err)
		}
		stage2 := newAsyncReadCloser(&chainedReadCloser{Reader: gz, closeFn: func() error {
			gzErr := gz.Close()
			s1Err := stage1.Close()
			if gzErr != nil {
				return gzErr
			}
			return s1Err
		}}, uncompressedStreamQueueDepth)
		return stage2, nil

	case types.OCILayerZStd:
		compRC, err := layer.Compressed()
		if err != nil {
			return nil, err
		}
		stage1 := newAsyncReadCloser(compRC, compressedStreamQueueDepth)
		zr, err := zstd.NewReader(stage1)
		if err != nil {
			_ = stage1.Close()
			return nil, fmt.Errorf("while creating zstd reader: %w", err)
		}
		stage2 := newAsyncReadCloser(&chainedReadCloser{Reader: zr, closeFn: func() error {
			zr.Close()
			return stage1.Close()
		}}, uncompressedStreamQueueDepth)
		return stage2, nil

	default:
		return layer.Uncompressed()
	}
}

type stickyEOFReader struct {
	r      io.Reader
	mu     sync.Mutex
	err    error
	sawEOF bool
}

func (s *stickyEOFReader) Read(p []byte) (int, error) {
	n, err := s.r.Read(p)
	if err != nil {
		s.mu.Lock()
		if err == io.EOF {
			s.sawEOF = true
		} else if s.err == nil {
			s.err = err
		}
		s.mu.Unlock()
	}
	return n, err
}

func (s *stickyEOFReader) status() (sawEOF bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sawEOF, s.err
}

func newIgzipReader(ctx context.Context, igzipPath string, stage1 io.ReadCloser) (io.ReadCloser, error) {
	sticky := &stickyEOFReader{r: stage1}
	cmd := buildIgzipCommand(ctx, igzipPath)
	cmd.Stdin = sticky
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	var closeOnce sync.Once
	var closeErr error
	return &chainedReadCloser{
		Reader: stdout,
		closeFn: func() error {
			closeOnce.Do(func() {
				_ = stdout.Close()
				waitErr := cmd.Wait()
				sawEOF, readErr := sticky.status()
				if waitErr == nil && !sawEOF && readErr == nil {
					if _, drainErr := io.Copy(io.Discard, sticky); drainErr != nil {
						readErr = drainErr
					} else {
						_, readErr = sticky.status()
					}
				}
				s1Err := stage1.Close()
				switch {
				case readErr != nil:
					closeErr = readErr
				case waitErr != nil:
					closeErr = fmt.Errorf("igzip decompression failed: %w (%s)", waitErr, strings.TrimSpace(stderr.String()))
				default:
					closeErr = s1Err
				}
			})
			return closeErr
		},
	}, nil
}

type chainedReadCloser struct {
	io.Reader
	closeFn func() error
}

func (c *chainedReadCloser) Close() error {
	return c.closeFn()
}

func (s *Store) writeRecord(digest v1.Hash, rec imageRecord) error {
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("while encoding image record: %w", err)
	}
	path := s.recordPath(digest)
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("while creating image record temp file: %w", err)
	}
	defer os.Remove(tmp.Name()) // no-op once the rename succeeds
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return fmt.Errorf("while writing image record: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("while closing image record: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("while moving image record into place: %w", err)
	}
	return nil
}

// remoteOpts assembles the go-containerregistry options for pulls from
// parsedRef's registry.
func (s *Store) remoteOpts(ctx context.Context, parsedRef name.Reference) []remote.Option {
	platform := v1.Platform{
		Architecture: runtime.GOARCH,
		OS:           "linux",
	}
	if s.platform != nil {
		platform = *s.platform
	}
	registry := parsedRef.Context().Registry.RegistryStr()
	opts := []remote.Option{
		// Propagate caller ctx into go-containerregistry so cancellation tears
		// down in-flight layer-blob HTTP requests instead of letting them run
		// to completion in background goroutines.
		remote.WithContext(ctx),
		remote.WithPlatform(platform),
		remote.WithJobs(layerPullConcurrency),
		remote.WithTransport(pullRangeTransport),
		remote.WithRetryBackoff(retryBackoffFor(registry)),
	}
	return opts
}

// withCredentials calls fetch with each credential until one succeeds, or
// once anonymously if there are none.
func (s *Store) withCredentials(ctx context.Context, parsedRef name.Reference, fetch func([]remote.Option) error) error {
	opts := s.remoteOpts(ctx, parsedRef)
	auths, err := s.credentials(ctx, parsedRef.Context())
	if err != nil {
		return fmt.Errorf("while resolving pull credentials: %w", err)
	}
	if len(auths) == 0 {
		return fetch(opts)
	}
	var errs []error
	for _, auth := range auths {
		err := fetch(append(slices.Clip(opts), remote.WithAuth(auth)))
		if err == nil {
			return nil
		}
		errs = append(errs, err)
		if ctx.Err() != nil {
			break
		}
	}
	return errors.Join(errs...)
}

func (s *Store) credentials(ctx context.Context, repo name.Repository) ([]authn.Authenticator, error) {
	switch kc := s.keychain.(type) {
	case nil:
		return nil, nil
	case CandidateKeychain:
		return kc.Candidates(ctx, repo)
	default:
		auth, err := authn.Resolve(ctx, kc, repo)
		if err != nil {
			return nil, err
		}
		return []authn.Authenticator{auth}, nil
	}
}

// parseRef applies the localhost-registry rewrite (kind local registries) and
// permits plain-HTTP pulls for localhost/loopback registries, matching docker
// behavior so local development needs no TLS certs.
func (s *Store) parseRef(ref string) (name.Reference, error) {
	rewritten := false
	if s.localhostRegistryReplacement != "" {
		if newRef := s.rewriteLocalRegistry(ref); newRef != ref {
			ref = newRef
			rewritten = true
		}
	}
	var nameOpts []name.Option
	if rewritten || isLocalRegistry(ref) {
		nameOpts = append(nameOpts, name.Insecure)
	}
	return name.ParseReference(ref, nameOpts...)
}

func registryHost(ref string) string {
	parts := strings.SplitN(ref, "/", 2)
	reg, err := name.NewRegistry(parts[0], name.Insecure)
	if err != nil {
		return ""
	}
	hostPart := reg.Name()
	if h, _, err := net.SplitHostPort(hostPart); err == nil {
		return h
	}
	return hostPart
}

func isLocalhostOrLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

func isLocalRegistry(ref string) bool {
	// By default docker permits localhost and 127.0.0.0/8; we also permit the
	// IPv6 loopback here.
	return isLocalhostOrLoopback(registryHost(ref))
}

func (s *Store) rewriteLocalRegistry(ref string) string {
	if isLocalRegistry(ref) {
		parts := strings.SplitN(ref, "/", 2)
		return s.localhostRegistryReplacement + "/" + parts[1]
	}
	return ref
}
