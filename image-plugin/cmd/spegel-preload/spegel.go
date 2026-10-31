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
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/klauspost/compress/zstd"
	"go.ate.dev/substrate/image-plugin/internal/proto/snapshots"
	"go.ate.dev/substrate/image-plugin/pkg/plugin"
	"go.ate.dev/substrate/internal/imagepluginclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	// ProviderSpegel identifies the stateless Spegel P2P OCI layer provider.
	ProviderSpegel = "spegel"

	// DefaultSpegelMirrorAddr is the default local Spegel OCI registry mirror HTTP endpoint.
	DefaultSpegelMirrorAddr = "http://127.0.0.1:5000"

	// DefaultSpegelRouterAddr is the default Spegel P2P router/DHT address.
	DefaultSpegelRouterAddr = ":5001"

	// DefaultSpegelBootstrapDNS is the default headless Service DNS domain used
	// to discover cluster Spegel peers in ate-system.
	DefaultSpegelBootstrapDNS = "spegel-bootstrap.ate-system.svc.cluster.local"

	// DefaultSpegelPeerPort is the default registry port on cluster Spegel peers.
	DefaultSpegelPeerPort = 5000

	// DefaultSpegelStoreDir is the default host directory where SpegelDriver
	// stores content-addressed OCI blobs (blobs/sha256/<digest>) and unpacked
	// layer trees (layers/sha256/<chainID>/fs). It sits under
	// /var/lib/ate/image-plugin-cache so symlinks from layer-<i>/fs resolve
	// inside atelet and ateworker pods without additional hostPath mounts.
	DefaultSpegelStoreDir = "/var/lib/ate/image-plugin-cache/spegel-store"

	// DefaultDockerConfigPath is the default Docker config.json path used for
	// credential synchronization.
	DefaultDockerConfigPath = "/run/ate/spegel/docker/config.json"

	// EnvSpegelMirrorAddr overrides the local Spegel HTTP mirror endpoint.
	EnvSpegelMirrorAddr = "SPEGEL_MIRROR_ADDR"

	// EnvSpegelRouterAddr overrides the Spegel P2P router address.
	EnvSpegelRouterAddr = "SPEGEL_ROUTER_ADDR"

	// EnvSpegelBootstrapDNS overrides the headless Service DNS domain for peer discovery.
	EnvSpegelBootstrapDNS = "SPEGEL_BOOTSTRAP_DNS"

	// EnvSpegelNodeIP sets this node's IP, which peers use to reach it.
	EnvSpegelNodeIP = "SPEGEL_NODE_IP"

	// EnvSpegelUpstreamMirrorAddr specifies an optional secondary local Spegel
	// daemon endpoint (e.g. http://127.0.0.1:5003).
	EnvSpegelUpstreamMirrorAddr = "SPEGEL_UPSTREAM_MIRROR_ADDR"

	// HeaderSpegelExternalRequest marks a peer-to-peer request. The receiving
	// peer serves only blobs it has or is fetching, and never fans out.
	HeaderSpegelExternalRequest = "X-Spegel-External-Request"

	// HeaderSpegelFetchUpstream asks the receiving peer to fetch a blob it
	// lacks from the upstream registry with its own credentials.
	HeaderSpegelFetchUpstream = "X-Spegel-Fetch-Upstream"

	// HeaderSpegelPeer reports the Spegel peer address that served a blob.
	HeaderSpegelPeer = "X-Spegel-Peer"

	// peerRefreshInterval is how often SpegelPeerServer re-resolves its peers.
	peerRefreshInterval = 5 * time.Second

	// blobFirstByteTimeout aborts a blob transfer that sends no bytes this
	// long after it starts. It covers registry token exchange.
	blobFirstByteTimeout = 60 * time.Second

	// blobStallTimeout aborts a blob transfer that sends no bytes for this
	// long after its first byte.
	blobStallTimeout = 10 * time.Second

	// maxPeerFetches caps the upstream fetches that peers can start at once.
	maxPeerFetches = 64
)

// BlobFetcherFunc returns a reader for the compressed OCI layer blob at
// layerIndex (with digest layerDigest) from the upstream registry.
type BlobFetcherFunc func(ctx context.Context, imageRef string, auth *imagepluginclient.AuthConfig, layerIndex int, layerDigest string) (io.ReadCloser, error)

// SpegelConfig configures a SpegelDriver.
type SpegelConfig struct {
	StoreDir string
	// ClearStore deletes StoreDir before use, so the node starts cold.
	// Actors still running on layers from the store lose their rootfs.
	ClearStore         bool
	DockerConfigPath   string
	SpegelMirrorAddr   string
	SpegelRouterAddr   string
	SpegelBootstrapDNS string
	SpegelPeerPort     int
	// SpegelOwnerDNS is a DNS name that resolves to the peers that own
	// blobs, e.g. dedicated seed nodes. Empty means every peer owns some.
	SpegelOwnerDNS string
	// SpegelNodeIP is this node's IP as peers see it. Defaults to
	// $SPEGEL_NODE_IP. Required with EnableP2PServer.
	SpegelNodeIP             string
	SpegelPeers              []string
	SpegelUpstreamMirrorAddr string
	EnableP2PServer          bool
	P2PListenAddr            string
	VerifySpegelReady        bool
	HTTPClient               *http.Client
	UpstreamBlobFetcher      BlobFetcherFunc
	ServerOpts               []plugin.Option
}

// SpegelStats holds counters exposed by SpegelPeerServer's /stats endpoint.
type SpegelStats struct {
	BlobsCached      int      `json:"blobs_cached"`
	BytesCached      int64    `json:"bytes_cached"`
	LocalHits        int64    `json:"local_hits"`
	LocalStoreHits   int64    `json:"local_store_hits"`
	PeerHits         int64    `json:"peer_hits"`
	PeerMisses       int64    `json:"peer_misses"`
	P2PRequestsIn    int64    `json:"p2p_requests_in"`
	P2PBytesServed   int64    `json:"p2p_bytes_served"`
	P2PBytesReceived int64    `json:"p2p_bytes_received"`
	UpstreamBytes    int64    `json:"upstream_bytes"`
	HealthPings      int64    `json:"health_pings"`
	DiscoveredPeers  []string `json:"discovered_peers"`
}

// spegelCounters holds blob counters that SpegelDriver and SpegelPeerServer
// share.
type spegelCounters struct {
	// localStoreHits counts layers whose blob was already in the store.
	localStoreHits atomic.Int64
	// p2pBytesReceived counts blob bytes read from peers or the mirror.
	p2pBytesReceived atomic.Int64
	// upstreamBytes counts blob bytes read from the upstream registry.
	upstreamBytes atomic.Int64
}

// countingReader adds the number of bytes read to n.
type countingReader struct {
	r io.Reader
	n *atomic.Int64
}

func (c countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n.Add(int64(n))
	return n, err
}

// SpegelPeerServer implements a Spegel-compatible stateless P2P OCI registry
// mirror HTTP server backed by the node's content-addressed OCI blob store
// (<StoreDir>/blobs/sha256/<hex>) and DNS/static peer discovery.
type SpegelPeerServer struct {
	storeDir           string
	bootstrapDNS       string
	ownerDNS           string
	peerPort           int
	staticPeers        []string
	upstreamMirrorAddr string
	httpClient         *http.Client
	listener           net.Listener
	server             *http.Server
	counters           *spegelCounters
	// upstream fetches blobs that peers ask this peer to fetch.
	upstream BlobFetcherFunc
	// ctx is canceled by Close. It bounds peer refreshes and blob fetches.
	ctx    context.Context
	cancel context.CancelFunc
	// dnsAddrs holds the last successful bootstrap DNS answer. Only
	// refreshPeers uses it.
	dnsAddrs []string

	peersMu sync.RWMutex
	// peers is the sorted list of other peers' host:port addresses.
	peers []string
	// owners is the sorted list of owner host:port addresses from the last
	// successful ownerDNS answer.
	owners []string
	// self is this peer's address as the other peers list it.
	self string

	fetchesMu sync.Mutex
	// fetches holds the running upstream fetches by hex digest.
	fetches map[string]*blobFetch
	// fetchWG tracks runFetch goroutines.
	fetchWG sync.WaitGroup

	localHits      atomic.Int64
	peerHits       atomic.Int64
	peerMisses     atomic.Int64
	p2pRequestsIn  atomic.Int64
	p2pBytesServed atomic.Int64
	healthPings    atomic.Int64
}

// NewSpegelPeerServer creates and starts a Spegel-compatible P2P OCI registry
// mirror server on listenAddr. Peers reach it at nodeIP:peerPort. upstream
// fetches blobs on behalf of peers. If ownerDNS is set, only the peers it
// resolves to own blobs.
func NewSpegelPeerServer(listenAddr, storeDir, bootstrapDNS, ownerDNS, nodeIP string, peerPort int, staticPeers []string, upstreamMirrorAddr string, httpClient *http.Client, counters *spegelCounters, upstream BlobFetcherFunc) (*SpegelPeerServer, error) {
	if listenAddr == "" {
		listenAddr = ":5000"
	}
	if peerPort <= 0 {
		peerPort = DefaultSpegelPeerPort
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Minute}
	}
	if net.ParseIP(nodeIP) == nil {
		return nil, fmt.Errorf("invalid Spegel node IP %q (set $%s)", nodeIP, EnvSpegelNodeIP)
	}
	blobDir := filepath.Join(storeDir, "blobs", "sha256")
	if err := os.MkdirAll(blobDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating Spegel blob store directory: %w", err)
	}
	// Remove partial blobs left by a previous process.
	partials, err := filepath.Glob(filepath.Join(blobDir, "*.partial"))
	if err != nil {
		return nil, err
	}
	for _, p := range partials {
		if err := os.Remove(p); err != nil {
			return nil, fmt.Errorf("removing stale partial blob: %w", err)
		}
	}
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("listening on %s: %w", listenAddr, err)
	}
	s := &SpegelPeerServer{
		storeDir:           storeDir,
		bootstrapDNS:       strings.TrimSpace(bootstrapDNS),
		ownerDNS:           strings.TrimSpace(ownerDNS),
		peerPort:           peerPort,
		staticPeers:        append([]string(nil), staticPeers...),
		upstreamMirrorAddr: strings.TrimRight(strings.TrimSpace(upstreamMirrorAddr), "/"),
		httpClient:         httpClient,
		listener:           ln,
		counters:           counters,
		upstream:           upstream,
		fetches:            make(map[string]*blobFetch),
		self:               net.JoinHostPort(nodeIP, strconv.Itoa(peerPort)),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/stats", s.handleStats)
	mux.HandleFunc("/v2/", s.handleV2)
	s.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.refreshPeers(s.ctx)
	go s.refreshPeersLoop(s.ctx)
	go func() {
		_ = s.server.Serve(ln)
	}()
	return s, nil
}

// Addr returns the bound TCP address of the Spegel peer server.
func (s *SpegelPeerServer) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Close shuts down the Spegel peer server.
func (s *SpegelPeerServer) Close() error {
	if s == nil || s.server == nil {
		return nil
	}
	s.cancel()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := s.server.Shutdown(ctx)
	// fetchBlob starts no fetch once s.ctx is canceled, so no Add races Wait.
	s.fetchesMu.Lock()
	s.fetchesMu.Unlock() //nolint:staticcheck // Barrier for in-flight fetchBlob calls.
	s.fetchWG.Wait()
	return err
}

// Stats returns the current P2P mirror counters and on-disk blob inventory.
func (s *SpegelPeerServer) Stats() SpegelStats {
	var blobsCached int
	var bytesCached int64
	entries, err := os.ReadDir(filepath.Join(s.storeDir, "blobs", "sha256"))
	if err == nil {
		for _, e := range entries {
			if e.IsDir() || strings.Contains(e.Name(), ".tmp") || strings.HasSuffix(e.Name(), ".partial") {
				continue
			}
			if info, err := e.Info(); err == nil {
				blobsCached++
				bytesCached += info.Size()
			}
		}
	}
	return SpegelStats{
		BlobsCached:      blobsCached,
		BytesCached:      bytesCached,
		LocalHits:        s.localHits.Load(),
		LocalStoreHits:   s.counters.localStoreHits.Load(),
		PeerHits:         s.peerHits.Load(),
		PeerMisses:       s.peerMisses.Load(),
		P2PRequestsIn:    s.p2pRequestsIn.Load(),
		P2PBytesServed:   s.p2pBytesServed.Load(),
		P2PBytesReceived: s.counters.p2pBytesReceived.Load(),
		UpstreamBytes:    s.counters.upstreamBytes.Load(),
		HealthPings:      s.healthPings.Load(),
		DiscoveredPeers:  s.Peers(),
	}
}

func (s *SpegelPeerServer) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	s.healthPings.Add(1)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *SpegelPeerServer) handleStats(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.Stats())
}

func (s *SpegelPeerServer) handleV2(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
	if r.URL.Path == "/v2/" || r.URL.Path == "/v2" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idx := strings.LastIndex(r.URL.Path, "/blobs/sha256:")
	if idx < 0 {
		http.NotFound(w, r)
		return
	}
	hexDigest := strings.ToLower(strings.TrimPrefix(r.URL.Path[idx+len("/blobs/"):], "sha256:"))
	if !isValidSHA256Hex(hexDigest) {
		http.Error(w, "invalid digest", http.StatusBadRequest)
		return
	}

	isExternalP2P := strings.EqualFold(r.Header.Get(HeaderSpegelExternalRequest), "true")
	if isExternalP2P {
		s.p2pRequestsIn.Add(1)
	}

	blobPath := filepath.Join(s.storeDir, "blobs", "sha256", hexDigest)
	if f, err := os.Open(blobPath); err == nil {
		defer func() { _ = f.Close() }()
		if info, err := f.Stat(); err == nil && !info.IsDir() {
			s.localHits.Add(1)
			if isExternalP2P && r.Method == http.MethodGet {
				s.p2pBytesServed.Add(info.Size())
			}
			w.Header().Set("Docker-Content-Digest", "sha256:"+hexDigest)
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set(HeaderSpegelPeer, s.Addr())
			http.ServeContent(w, r, hexDigest, info.ModTime(), f)
			return
		}
	}

	if f := s.runningOrNewFetch(r, idx, hexDigest); f != nil {
		s.serveBlobFetch(w, r, f, hexDigest, isExternalP2P)
		return
	}

	if isExternalP2P {
		http.NotFound(w, r)
		return
	}

	// Query cluster Spegel peers (static list + headless DNS).
	for _, peer := range s.Peers() {
		if s.streamFromPeer(w, r, peer, hexDigest) {
			s.peerHits.Add(1)
			return
		}
	}

	// Optional fallback to a secondary local Spegel daemon (e.g. ghcr.io/spegel-org/spegel on :5003).
	if s.upstreamMirrorAddr != "" {
		if s.streamFromUpstreamMirror(w, r, hexDigest) {
			s.peerHits.Add(1)
			return
		}
	}

	s.peerMisses.Add(1)
	http.NotFound(w, r)
}

// Peers returns the sorted host:port addresses of the other peers, as of the
// last refresh.
func (s *SpegelPeerServer) Peers() []string {
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()
	return append([]string(nil), s.peers...)
}

func (s *SpegelPeerServer) refreshPeersLoop(ctx context.Context) {
	t := time.NewTicker(peerRefreshInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.refreshPeers(ctx)
		}
	}
}

// refreshPeers resolves the static and headless-DNS peers, and the owners.
// If a DNS lookup fails, it keeps the previous answer.
func (s *SpegelPeerServer) refreshPeers(ctx context.Context) {
	if s.bootstrapDNS != "" {
		lookupCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
		addrs, err := net.DefaultResolver.LookupHost(lookupCtx, s.bootstrapDNS)
		cancel()
		if err != nil {
			slog.DebugContext(ctx, "Spegel peer DNS lookup failed",
				slog.String("bootstrapDNS", s.bootstrapDNS), slog.Any("err", err))
		} else {
			s.dnsAddrs = addrs
		}
	}
	var owners []string
	if s.ownerDNS != "" {
		lookupCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
		addrs, err := net.DefaultResolver.LookupHost(lookupCtx, s.ownerDNS)
		cancel()
		if err != nil {
			slog.WarnContext(ctx, "Spegel owner DNS lookup failed",
				slog.String("ownerDNS", s.ownerDNS), slog.Any("err", err))
		}
		for _, ip := range addrs {
			owners = append(owners, net.JoinHostPort(ip, strconv.Itoa(s.peerPort)))
		}
		sort.Strings(owners)
	}
	peers := s.discoverPeers()
	s.peersMu.Lock()
	s.peers = peers
	if len(owners) > 0 {
		s.owners = owners
	}
	s.peersMu.Unlock()
}

// discoverPeers returns the static and DNS peers other than s.self.
func (s *SpegelPeerServer) discoverPeers() []string {
	seen := map[string]struct{}{s.self: {}}
	var out []string
	addPeer := func(hostPort string) {
		hostPort = strings.TrimSpace(hostPort)
		hostPort = strings.TrimPrefix(hostPort, "http://")
		hostPort = strings.TrimPrefix(hostPort, "https://")
		if hostPort == "" {
			return
		}
		h, p, err := net.SplitHostPort(hostPort)
		if err != nil {
			h = hostPort
			p = fmt.Sprintf("%d", s.peerPort)
			hostPort = net.JoinHostPort(h, p)
		}
		if _, ok := seen[hostPort]; !ok {
			seen[hostPort] = struct{}{}
			out = append(out, hostPort)
		}
	}

	for _, p := range s.staticPeers {
		addPeer(p)
	}
	for _, ip := range s.dnsAddrs {
		addPeer(net.JoinHostPort(ip, strconv.Itoa(s.peerPort)))
	}
	sort.Strings(out)
	return out
}

// owner returns the peer that fetches hexDigest from upstream, and whether it
// is this peer. With ownerDNS set, it picks among the owners, and returns ""
// until the first owner DNS answer.
func (s *SpegelPeerServer) owner(hexDigest string) (string, bool) {
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()
	members := s.owners
	if s.ownerDNS == "" {
		members = append([]string{s.self}, s.peers...)
	}
	o := rendezvousOwner(members, hexDigest)
	return o, o == s.self
}

func (s *SpegelPeerServer) runningFetch(hexDigest string) *blobFetch {
	s.fetchesMu.Lock()
	defer s.fetchesMu.Unlock()
	return s.fetches[hexDigest]
}

// rendezvousOwner returns the member with the highest hash of member and
// digest, so all peers with the same member list pick the same owner.
func rendezvousOwner(members []string, hexDigest string) string {
	var best string
	var bestScore uint64
	for _, m := range members {
		sum := sha256.Sum256([]byte(m + "/" + hexDigest))
		if score := binary.BigEndian.Uint64(sum[:8]); best == "" || score > bestScore {
			best, bestScore = m, score
		}
	}
	return best
}

// blobFetch is an upstream fetch of one blob into <digest>.partial. Readers
// stream the file while it grows. The fetch renames it to the blob path
// after verifying its digest.
type blobFetch struct {
	partialPath string
	finalPath   string
	// done is closed when the fetch ends.
	done chan struct{}

	mu       sync.Mutex
	cond     *sync.Cond
	written  int64
	finished bool
	err      error
}

func newBlobFetch(finalPath string) *blobFetch {
	f := &blobFetch{
		partialPath: finalPath + ".partial",
		finalPath:   finalPath,
		done:        make(chan struct{}),
	}
	f.cond = sync.NewCond(&f.mu)
	return f
}

func (f *blobFetch) finish(err error) {
	f.mu.Lock()
	f.finished, f.err = true, err
	f.cond.Broadcast()
	f.mu.Unlock()
	close(f.done)
}

// fetchBlob returns the running fetch of hexDigest. If none runs and the blob
// is not in the store, it starts one with fetch, unless maxRunning fetches
// already run. started reports whether it started a fetch.
func (s *SpegelPeerServer) fetchBlob(hexDigest string, maxRunning int, fetch func(context.Context) (io.ReadCloser, error)) (f *blobFetch, started bool, err error) {
	s.fetchesMu.Lock()
	defer s.fetchesMu.Unlock()
	if f := s.fetches[hexDigest]; f != nil {
		return f, false, nil
	}
	f = newBlobFetch(filepath.Join(s.storeDir, "blobs", "sha256", hexDigest))
	if _, err := os.Stat(f.finalPath); err == nil {
		f.finish(nil)
		return f, false, nil
	}
	if err := s.ctx.Err(); err != nil {
		return nil, false, err
	}
	if len(s.fetches) >= maxRunning {
		return nil, false, fmt.Errorf("%d upstream blob fetches already running", len(s.fetches))
	}
	file, err := os.OpenFile(f.partialPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, false, err
	}
	s.fetches[hexDigest] = f
	s.fetchWG.Add(1)
	go s.runFetch(f, file, hexDigest, fetch)
	return f, true, nil
}

// awaitBlob waits until hexDigest is in the store, fetching it with fetch
// unless a fetch already runs. started reports whether it started the fetch.
func (s *SpegelPeerServer) awaitBlob(ctx context.Context, hexDigest string, fetch func(context.Context) (io.ReadCloser, error)) (started bool, err error) {
	f, started, err := s.fetchBlob(hexDigest, math.MaxInt, fetch)
	if err != nil {
		return false, err
	}
	select {
	case <-f.done:
		return started, f.err
	case <-ctx.Done():
		return started, ctx.Err()
	}
}

func (s *SpegelPeerServer) runFetch(f *blobFetch, file *os.File, hexDigest string, fetch func(context.Context) (io.ReadCloser, error)) {
	defer s.fetchWG.Done()
	err := s.copyVerified(f, file, hexDigest, fetch)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	// Rename under f.mu so readers never open a partial path that is gone.
	f.mu.Lock()
	if err == nil {
		err = os.Rename(f.partialPath, f.finalPath)
	}
	if err != nil {
		_ = os.Remove(f.partialPath)
	}
	f.mu.Unlock()
	// Delete first so callers that see the failure can start a new fetch.
	s.fetchesMu.Lock()
	delete(s.fetches, hexDigest)
	s.fetchesMu.Unlock()
	f.finish(err)
	if err != nil {
		slog.WarnContext(s.ctx, "Spegel upstream blob fetch failed",
			slog.String("digest", "sha256:"+hexDigest), slog.Any("err", err))
	}
}

func (s *SpegelPeerServer) copyVerified(f *blobFetch, file *os.File, hexDigest string, fetch func(context.Context) (io.ReadCloser, error)) error {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	stall := time.AfterFunc(blobFirstByteTimeout, cancel)
	defer stall.Stop()
	rc, err := fetch(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	// Closing the body unblocks a Read stuck on a stalled connection.
	stopClose := context.AfterFunc(ctx, func() { _ = rc.Close() })
	defer stopClose()
	h := sha256.New()
	buf := make([]byte, 256<<10)
	for {
		n, readErr := rc.Read(buf)
		if ctx.Err() != nil {
			return fmt.Errorf("upstream blob fetch stalled: %w", ctx.Err())
		}
		if n > 0 {
			stall.Reset(blobStallTimeout)
			if _, err := file.Write(buf[:n]); err != nil {
				return err
			}
			_, _ = h.Write(buf[:n])
			s.counters.upstreamBytes.Add(int64(n))
			f.mu.Lock()
			f.written += int64(n)
			f.cond.Broadcast()
			f.mu.Unlock()
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if got := hex.EncodeToString(h.Sum(nil)); !strings.EqualFold(got, hexDigest) {
		return fmt.Errorf("blob digest mismatch: got sha256:%s, want sha256:%s", got, hexDigest)
	}
	return nil
}

// blobFetchReader reads a blobFetch's file, waiting for bytes the fetch has
// not written yet. Read fails once ctx is done.
type blobFetchReader struct {
	f        *blobFetch
	file     *os.File
	off      int64
	ctx      context.Context
	stopWake func() bool
}

func (f *blobFetch) open(ctx context.Context) (*blobFetchReader, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	path := f.partialPath
	if f.finished {
		if f.err != nil {
			return nil, f.err
		}
		path = f.finalPath
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	wake := func() {
		f.mu.Lock()
		f.cond.Broadcast()
		f.mu.Unlock()
	}
	return &blobFetchReader{f: f, file: file, ctx: ctx, stopWake: context.AfterFunc(ctx, wake)}, nil
}

func (r *blobFetchReader) Read(p []byte) (int, error) {
	r.f.mu.Lock()
	for r.off >= r.f.written && !r.f.finished && r.ctx.Err() == nil {
		r.f.cond.Wait()
	}
	written, finished, err := r.f.written, r.f.finished, r.f.err
	r.f.mu.Unlock()
	if err != nil {
		return 0, err
	}
	if !finished && r.off >= written {
		return 0, r.ctx.Err()
	}
	if !finished && int64(len(p)) > written-r.off {
		p = p[:written-r.off]
	}
	n, err := r.file.ReadAt(p, r.off)
	r.off += int64(n)
	if !finished && errors.Is(err, io.EOF) {
		err = nil
	}
	return n, err
}

func (r *blobFetchReader) Close() error {
	r.stopWake()
	return r.file.Close()
}

// runningOrNewFetch returns the running fetch of hexDigest. If none runs, r
// carries HeaderSpegelFetchUpstream and this peer owns hexDigest, it starts
// one with this peer's credentials. idx is the index of "/blobs/" in
// r.URL.Path.
func (s *SpegelPeerServer) runningOrNewFetch(r *http.Request, idx int, hexDigest string) *blobFetch {
	if f := s.runningFetch(hexDigest); f != nil {
		return f
	}
	if r.Method != http.MethodGet || !strings.EqualFold(r.Header.Get(HeaderSpegelFetchUpstream), "true") {
		return nil
	}
	if _, isSelf := s.owner(hexDigest); !isSelf {
		return nil
	}
	repo := strings.TrimPrefix(r.URL.Path[:idx], "/v2/")
	ns := r.URL.Query().Get("ns")
	if repo == "" || ns == "" {
		return nil
	}
	imageRef := ns + "/" + repo + "@sha256:" + hexDigest
	f, _, err := s.fetchBlob(hexDigest, maxPeerFetches, func(ctx context.Context) (io.ReadCloser, error) {
		return s.upstream(ctx, imageRef, nil, -1, "sha256:"+hexDigest)
	})
	if err != nil {
		slog.WarnContext(r.Context(), "Starting Spegel upstream blob fetch failed",
			slog.String("digest", "sha256:"+hexDigest), slog.Any("err", err))
		return nil
	}
	return f
}

// serveBlobFetch streams f's file to w as it grows.
func (s *SpegelPeerServer) serveBlobFetch(w http.ResponseWriter, r *http.Request, f *blobFetch, hexDigest string, isExternalP2P bool) {
	rd, err := f.open(r.Context())
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = rd.Close() }()
	// Wait for the first bytes, so a fetch that fails to start gets a 404.
	buf := make([]byte, 256<<10)
	n, err := rd.Read(buf)
	if n == 0 && err != nil && !errors.Is(err, io.EOF) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Docker-Content-Digest", "sha256:"+hexDigest)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set(HeaderSpegelPeer, s.Addr())
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	rc := http.NewResponseController(w)
	for {
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return
			}
			if isExternalP2P {
				s.p2pBytesServed.Add(int64(n))
			}
			_ = rc.Flush()
		}
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			// Abort so the client sees a truncated response.
			panic(http.ErrAbortHandler)
		}
		n, err = rd.Read(buf)
	}
}

func (s *SpegelPeerServer) streamFromPeer(w http.ResponseWriter, r *http.Request, peerHostPort, hexDigest string) bool {
	peerURL := fmt.Sprintf("http://%s%s", peerHostPort, r.URL.RequestURI())
	probeCtx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
	headReq, err := http.NewRequestWithContext(probeCtx, http.MethodHead, peerURL, nil)
	if err != nil {
		cancel()
		return false
	}
	headReq.Header.Set(HeaderSpegelExternalRequest, "true")
	headResp, err := s.httpClient.Do(headReq)
	cancel()
	if err != nil {
		return false
	}
	_ = headResp.Body.Close()
	if headResp.StatusCode != http.StatusOK {
		return false
	}

	if r.Method == http.MethodHead {
		w.Header().Set("Docker-Content-Digest", "sha256:"+hexDigest)
		w.Header().Set(HeaderSpegelPeer, peerHostPort)
		if cl := headResp.Header.Get("Content-Length"); cl != "" {
			w.Header().Set("Content-Length", cl)
		}
		w.WriteHeader(http.StatusOK)
		return true
	}

	getReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, peerURL, nil)
	if err != nil {
		return false
	}
	getReq.Header.Set(HeaderSpegelExternalRequest, "true")
	if rng := r.Header.Get("Range"); rng != "" {
		getReq.Header.Set("Range", rng)
	}
	resp, err := s.httpClient.Do(getReq)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return false
	}
	w.Header().Set("Docker-Content-Digest", "sha256:"+hexDigest)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set(HeaderSpegelPeer, peerHostPort)
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		w.Header().Set("Content-Length", cl)
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
	slog.InfoContext(r.Context(), "Streamed OCI layer blob from remote Spegel P2P peer",
		slog.String("peer", peerHostPort),
		slog.String("digest", "sha256:"+hexDigest))
	return true
}

func (s *SpegelPeerServer) streamFromUpstreamMirror(w http.ResponseWriter, r *http.Request, hexDigest string) bool {
	targetURL := s.upstreamMirrorAddr + r.URL.RequestURI()
	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, nil)
	if err != nil {
		return false
	}
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return false
	}
	w.Header().Set("Docker-Content-Digest", "sha256:"+hexDigest)
	w.Header().Set(HeaderSpegelPeer, s.upstreamMirrorAddr)
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		w.Header().Set("Content-Length", cl)
	}
	w.WriteHeader(resp.StatusCode)
	if r.Method != http.MethodHead {
		_, _ = io.Copy(w, resp.Body)
	}
	return true
}

func localInterfaceIPs() map[string]struct{} {
	ips := map[string]struct{}{
		"127.0.0.1": {},
		"::1":       {},
		"localhost": {},
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, a := range addrs {
		if ipNet, ok := a.(*net.IPNet); ok && ipNet.IP != nil {
			ips[ipNet.IP.String()] = struct{}{}
		}
	}
	return ips
}

// SpegelDriver wraps plugin.Driver with stateless Spegel P2P OCI blob mirror
// lookup, SHA-256 verified blob caching, and content-addressed layer unpacking.
type SpegelDriver struct {
	*plugin.Driver
	dockerSyncer        *plugin.DockerConfigSyncer
	storeDir            string
	spegelMirrorAddr    string
	spegelRouterAddr    string
	spegelBootstrapDNS  string
	verifySpegelReady   bool
	httpClient          *http.Client
	upstreamBlobFetcher BlobFetcherFunc
	peerServer          *SpegelPeerServer
	counters            *spegelCounters

	p2pBlobHits      atomic.Int64
	upstreamBlobMiss atomic.Int64
}

// ResolveSpegelMirrorAddr returns the local Spegel HTTP mirror endpoint from
// flagVal, $SPEGEL_MIRROR_ADDR, or DefaultSpegelMirrorAddr, validating that it
// is a well-formed HTTP(S) URL.
func ResolveSpegelMirrorAddr(flagVal string) (string, error) {
	addr := strings.TrimSpace(flagVal)
	if addr == "" {
		addr = strings.TrimSpace(os.Getenv(EnvSpegelMirrorAddr))
	}
	if addr == "" {
		addr = DefaultSpegelMirrorAddr
	}
	u, err := url.Parse(addr)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("invalid Spegel mirror endpoint %q (expected http:// or https:// URL)", addr)
	}
	return strings.TrimRight(addr, "/"), nil
}

// ResolveSpegelRouterAddr resolves the Spegel P2P router/DHT address from
// flagVal, $SPEGEL_ROUTER_ADDR, or DefaultSpegelRouterAddr.
func ResolveSpegelRouterAddr(flagVal string) (string, error) {
	addr := strings.TrimSpace(flagVal)
	if addr == "" {
		addr = strings.TrimSpace(os.Getenv(EnvSpegelRouterAddr))
	}
	if addr == "" {
		addr = DefaultSpegelRouterAddr
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		return "", fmt.Errorf("invalid Spegel router address %q (expected [host]:port)", addr)
	}
	return addr, nil
}

// NewSpegel creates a new Spegel P2P ImagePlugin driver.
func NewSpegel(cfg SpegelConfig) (*SpegelDriver, error) {
	mirrorAddr, err := ResolveSpegelMirrorAddr(cfg.SpegelMirrorAddr)
	if err != nil {
		return nil, err
	}
	routerAddr, err := ResolveSpegelRouterAddr(cfg.SpegelRouterAddr)
	if err != nil {
		return nil, err
	}
	bootstrapDNS := strings.TrimSpace(cfg.SpegelBootstrapDNS)
	if bootstrapDNS == "" {
		bootstrapDNS = strings.TrimSpace(os.Getenv(EnvSpegelBootstrapDNS))
	}
	if bootstrapDNS == "" {
		bootstrapDNS = DefaultSpegelBootstrapDNS
	}
	nodeIP := strings.TrimSpace(cfg.SpegelNodeIP)
	if nodeIP == "" {
		nodeIP = strings.TrimSpace(os.Getenv(EnvSpegelNodeIP))
	}
	upstreamMirror := strings.TrimSpace(cfg.SpegelUpstreamMirrorAddr)
	if upstreamMirror == "" {
		upstreamMirror = strings.TrimSpace(os.Getenv(EnvSpegelUpstreamMirrorAddr))
	}

	storeDir := strings.TrimSpace(cfg.StoreDir)
	if storeDir == "" {
		storeDir = DefaultSpegelStoreDir
	}
	if cfg.ClearStore {
		if err := os.RemoveAll(storeDir); err != nil {
			return nil, fmt.Errorf("clearing Spegel store %s: %w", storeDir, err)
		}
		slog.Info("Cleared Spegel store on start", slog.String("storeDir", storeDir))
	}
	if err := os.MkdirAll(filepath.Join(storeDir, "blobs", "sha256"), 0o755); err != nil {
		return nil, fmt.Errorf("creating Spegel blob store directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(storeDir, "layers", "sha256"), 0o755); err != nil {
		return nil, fmt.Errorf("creating Spegel layer store directory: %w", err)
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 2 * time.Minute,
		}
	}

	sd := &SpegelDriver{
		dockerSyncer:        plugin.NewDockerConfigSyncer(cfg.DockerConfigPath),
		storeDir:            storeDir,
		spegelMirrorAddr:    mirrorAddr,
		spegelRouterAddr:    routerAddr,
		spegelBootstrapDNS:  bootstrapDNS,
		verifySpegelReady:   cfg.VerifySpegelReady,
		httpClient:          httpClient,
		upstreamBlobFetcher: cfg.UpstreamBlobFetcher,
		counters:            &spegelCounters{},
	}

	opts := append([]plugin.Option{
		plugin.WithSnapshotsClient(&noopSnapshotsClient{}),
		plugin.WithLayerMounter(sd.mountLayers),
		plugin.WithWarmHitHook(sd.onWarmHit),
		plugin.WithProbeHook(sd.probeSpegel),
	}, cfg.ServerOpts...)

	base, err := plugin.NewDriver(ProviderSpegel, "", ProviderSpegel, opts...)
	if err != nil {
		return nil, err
	}
	sd.Driver = base
	if sd.upstreamBlobFetcher == nil {
		sd.upstreamBlobFetcher = sd.defaultUpstreamBlobFetcher
	}

	// Start the peer server last: peers may ask it to fetch with
	// sd.upstreamBlobFetcher as soon as it listens.
	if cfg.EnableP2PServer {
		sd.peerServer, err = NewSpegelPeerServer(
			cfg.P2PListenAddr,
			storeDir,
			bootstrapDNS,
			cfg.SpegelOwnerDNS,
			nodeIP,
			cfg.SpegelPeerPort,
			cfg.SpegelPeers,
			upstreamMirror,
			httpClient,
			sd.counters,
			sd.upstreamBlobFetcher,
		)
		if err != nil {
			_ = base.Close()
			return nil, fmt.Errorf("starting Spegel P2P server: %w", err)
		}
		if cfg.SpegelMirrorAddr == "" && os.Getenv(EnvSpegelMirrorAddr) == "" {
			_, port, splitErr := net.SplitHostPort(sd.peerServer.Addr())
			if splitErr == nil {
				sd.spegelMirrorAddr = "http://127.0.0.1:" + port
			}
		}
	}
	return sd, nil
}

// Close closes the Spegel driver and any embedded P2P server.
func (sd *SpegelDriver) Close() error {
	if sd.peerServer != nil {
		_ = sd.peerServer.Close()
	}
	return sd.Driver.Close()
}

// SpegelMirrorAddr returns the configured local Spegel HTTP mirror endpoint.
func (sd *SpegelDriver) SpegelMirrorAddr() string {
	return sd.spegelMirrorAddr
}

// SpegelRouterAddr returns the configured Spegel P2P router/DHT address.
func (sd *SpegelDriver) SpegelRouterAddr() string {
	return sd.spegelRouterAddr
}

// SpegelBootstrapDNS returns the configured headless Service DNS domain for peer discovery.
func (sd *SpegelDriver) SpegelBootstrapDNS() string {
	return sd.spegelBootstrapDNS
}

// StoreDir returns the content-addressed OCI blob and unpacked layer store root.
func (sd *SpegelDriver) StoreDir() string {
	return sd.storeDir
}

// PeerServer returns the embedded SpegelPeerServer, if enabled.
func (sd *SpegelDriver) PeerServer() *SpegelPeerServer {
	return sd.peerServer
}

// P2PBlobHits returns the number of OCI layer blobs fetched via the Spegel P2P mirror.
func (sd *SpegelDriver) P2PBlobHits() int64 {
	return sd.p2pBlobHits.Load()
}

// UpstreamBlobMisses returns the number of OCI layer blobs fetched from the upstream registry on a P2P miss.
func (sd *SpegelDriver) UpstreamBlobMisses() int64 {
	return sd.upstreamBlobMiss.Load()
}

// PingSpegelMirror probes the local Spegel /healthz endpoint to verify the P2P
// mirror is ready.
func (sd *SpegelDriver) PingSpegelMirror(ctx context.Context) error {
	healthURL := sd.spegelMirrorAddr + "/healthz"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return fmt.Errorf("creating Spegel healthz request %q: %w", healthURL, err)
	}
	resp, err := sd.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("probing Spegel mirror at %q: %w", healthURL, err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("spegel mirror healthz %q returned HTTP %d", healthURL, resp.StatusCode)
	}
	return nil
}

func (sd *SpegelDriver) probeSpegel(ctx context.Context) error {
	if !sd.verifySpegelReady {
		return nil
	}
	if err := sd.PingSpegelMirror(ctx); err != nil {
		slog.WarnContext(ctx, "spegel mirror healthz probe failed",
			slog.String("mirrorAddr", sd.spegelMirrorAddr),
			slog.Any("err", err))
		return err
	}
	return nil
}

func (sd *SpegelDriver) onWarmHit(ctx context.Context, _ string, req *imagepluginclient.PrepareRequest) {
	if req == nil {
		return
	}
	sd.dockerSyncer.Sync(ctx, sd.Driver.Keychain(), req.ImageRef, req.AuthConfig, false)
}

func (sd *SpegelDriver) mountLayers(ctx context.Context, env *plugin.MountEnv) (*plugin.MountedLayers, error) {
	if sd.verifySpegelReady {
		pingCtx, cancel := context.WithTimeout(ctx, plugin.DefaultProbeTimeout)
		err := sd.PingSpegelMirror(pingCtx)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("spegel P2P mirror unavailable (%s): %w", sd.spegelMirrorAddr, err)
		}
	}

	sd.dockerSyncer.Sync(ctx, sd.Driver.Keychain(), env.ImageRef, env.AuthConfig, true)

	registryHost, repoPath, err := parseRegistryAndRepo(env.ImageRef)
	if err != nil {
		return nil, err
	}

	n := len(env.ChainInfos)
	res := &plugin.MountedLayers{
		MountDirs: make([]string, n),
	}

	type layerOutcome struct {
		mountDir   string
		statDur    time.Duration
		prepareDur time.Duration
		err        error
	}
	outcomes := make([]layerOutcome, n)

	var wg sync.WaitGroup
	wg.Add(n)
	for i, c := range env.ChainInfos {
		go func(i int, c plugin.ChainInfo) {
			defer wg.Done()

			var layerDigest string
			if i < len(env.LayerDigests) {
				layerDigest = env.LayerDigests[i]
			}
			if layerDigest == "" {
				layerDigest = c.ChainID
			}

			chainHex := stripSHA256Prefix(c.ChainID)
			if !isValidSHA256Hex(chainHex) {
				outcomes[i].err = fmt.Errorf("layer %d has invalid chainID %q", i, c.ChainID)
				return
			}
			blobHex := stripSHA256Prefix(layerDigest)
			if !isValidSHA256Hex(blobHex) {
				outcomes[i].err = fmt.Errorf("layer %d has invalid digest %q", i, layerDigest)
				return
			}

			layerDir := filepath.Join(sd.storeDir, "layers", "sha256", chainHex)
			layerFSDir := filepath.Join(layerDir, "fs")
			finalizedMarker := filepath.Join(layerDir, "finalized")

			lk := env.Driver.AcquireChainLock(c.ChainID)
			lk.Lock()

			statStart := time.Now()
			alreadyUnpacked := isFinalizedLayerDir(layerFSDir, finalizedMarker)
			outcomes[i].statDur = time.Since(statStart)

			var prepErr error
			if !alreadyUnpacked {
				prepStart := time.Now()
				prepErr = sd.materializeSpegelLayer(ctx, env.ImageRef, registryHost, repoPath, env.AuthConfig, i, blobHex, layerDir, layerFSDir, finalizedMarker)
				outcomes[i].prepareDur = time.Since(prepStart)
			}
			lk.Unlock()

			if prepErr != nil {
				outcomes[i].err = fmt.Errorf("layer %d (%s): %w", i, c.ChainID, prepErr)
				return
			}
			outcomes[i].mountDir = layerFSDir
		}(i, c)
	}
	wg.Wait()

	var maxPrepDur time.Duration
	for i := 0; i < n; i++ {
		res.StatDur += outcomes[i].statDur
		if outcomes[i].prepareDur > maxPrepDur {
			maxPrepDur = outcomes[i].prepareDur
		}
		if outcomes[i].err != nil {
			return res, outcomes[i].err
		}
		res.MountDirs[i] = outcomes[i].mountDir
	}
	res.PrepareDur = maxPrepDur

	return res, nil
}

func (sd *SpegelDriver) materializeSpegelLayer(ctx context.Context, imageRef, registryHost, repoPath string, auth *imagepluginclient.AuthConfig, layerIndex int, blobHex, layerDir, layerFSDir, finalizedMarker string) error {
	blobPath, sourcePeer, err := sd.ensureBlobInStore(ctx, imageRef, registryHost, repoPath, auth, layerIndex, blobHex)
	if err != nil {
		return err
	}

	f, err := os.Open(blobPath)
	if err != nil {
		return fmt.Errorf("opening cached blob %s: %w", blobPath, err)
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat cached blob %s: %w", blobPath, err)
	}

	tarRC, err := openDecompressedTarStream(f)
	if err != nil {
		return fmt.Errorf("decompressing blob sha256:%s: %w", blobHex, err)
	}
	defer func() { _ = tarRC.Close() }()

	stagingDir := fmt.Sprintf("%s.tmp-%x", layerDir, time.Now().UnixNano())
	stagingFS := filepath.Join(stagingDir, "fs")
	if err := os.MkdirAll(stagingFS, 0o755); err != nil {
		return fmt.Errorf("creating staging layer directory %s: %w", stagingFS, err)
	}
	defer func() { _ = os.RemoveAll(stagingDir) }()

	if err := plugin.UnpackLayerTarToDir(ctx, tarRC, stagingFS); err != nil {
		return fmt.Errorf("unpacking layer %d (sha256:%s): %w", layerIndex, blobHex, err)
	}

	if err := os.WriteFile(filepath.Join(stagingDir, "finalized"), nil, 0o600); err != nil {
		return fmt.Errorf("writing staging finalized marker: %w", err)
	}

	_ = os.RemoveAll(layerDir)
	if err := os.Rename(stagingDir, layerDir); err != nil {
		return fmt.Errorf("committing unpacked layer directory %s: %w", layerDir, err)
	}

	slog.InfoContext(ctx, "Materialized Spegel OCI layer",
		slog.Int("layerIndex", layerIndex),
		slog.String("blobDigest", "sha256:"+blobHex),
		slog.Int64("blobBytes", info.Size()),
		slog.String("source", sourcePeer),
		slog.String("layerFSDir", layerFSDir))
	return nil
}

func (sd *SpegelDriver) ensureBlobInStore(ctx context.Context, imageRef, registryHost, repoPath string, auth *imagepluginclient.AuthConfig, layerIndex int, blobHex string) (string, string, error) {
	blobHex = strings.ToLower(blobHex)
	blobPath := filepath.Join(sd.storeDir, "blobs", "sha256", blobHex)
	if info, err := os.Stat(blobPath); err == nil && !info.IsDir() && info.Size() > 0 {
		sd.counters.localStoreHits.Add(1)
		return blobPath, "local-store", nil
	}
	if sd.peerServer != nil {
		return sd.fetchBlobViaOwner(ctx, imageRef, registryHost, repoPath, auth, layerIndex, blobHex, blobPath)
	}

	// 1. Query the Spegel P2P mirror first: GET <mirror>/v2/<repo>/blobs/sha256:<hex>?ns=<registry>
	mirrorURL := fmt.Sprintf("%s/v2/%s/blobs/sha256:%s?ns=%s",
		sd.spegelMirrorAddr, strings.TrimPrefix(repoPath, "/"), blobHex, url.QueryEscape(registryHost))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mirrorURL, nil)
	if err == nil {
		resp, doErr := sd.httpClient.Do(req)
		if doErr == nil {
			if resp.StatusCode == http.StatusOK {
				peer := resp.Header.Get(HeaderSpegelPeer)
				if peer == "" {
					peer = sd.spegelMirrorAddr
				}
				writeErr := writeVerifiedBlobAtomically(countingReader{resp.Body, &sd.counters.p2pBytesReceived}, blobPath, blobHex)
				_ = resp.Body.Close()
				if writeErr == nil {
					sd.p2pBlobHits.Add(1)
					return blobPath, "spegel-p2p:" + peer, nil
				}
				slog.WarnContext(ctx, "Spegel mirror blob failed SHA-256 verification; falling back to upstream registry",
					slog.String("digest", "sha256:"+blobHex),
					slog.Any("err", writeErr))
			} else {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}
		}
	}

	// 2. Cluster-cold miss: pull from the upstream OCI registry and ingest into
	// the local Spegel OCI blob store so this node advertises it to all peers.
	rc, err := sd.upstreamBlobFetcher(ctx, imageRef, auth, layerIndex, "sha256:"+blobHex)
	if err != nil {
		return "", "", fmt.Errorf("fetching blob sha256:%s from upstream registry: %w", blobHex, err)
	}
	defer func() { _ = rc.Close() }()

	if err := writeVerifiedBlobAtomically(countingReader{rc, &sd.counters.upstreamBytes}, blobPath, blobHex); err != nil {
		return "", "", fmt.Errorf("storing upstream blob sha256:%s: %w", blobHex, err)
	}
	sd.upstreamBlobMiss.Add(1)
	return blobPath, "upstream:" + registryHost, nil
}

// fetchBlobViaOwner fetches a blob through its owner peer, so one node in the
// cluster pulls it from upstream. If the owner fails, it fetches from
// upstream, then from the other peers' stores.
func (sd *SpegelDriver) fetchBlobViaOwner(ctx context.Context, imageRef, registryHost, repoPath string, auth *imagepluginclient.AuthConfig, layerIndex int, blobHex, blobPath string) (string, string, error) {
	ps := sd.peerServer
	if owner, isSelf := ps.owner(blobHex); !isSelf {
		err := sd.fetchFromPeer(ctx, owner, registryHost, repoPath, blobHex, blobPath, true)
		if err == nil {
			sd.p2pBlobHits.Add(1)
			return blobPath, "spegel-p2p:" + owner, nil
		}
		slog.WarnContext(ctx, "Fetching blob from owner peer failed; falling back to upstream registry",
			slog.String("digest", "sha256:"+blobHex),
			slog.String("owner", owner),
			slog.Any("err", err))
	}

	started, upstreamErr := ps.awaitBlob(ctx, blobHex, func(fetchCtx context.Context) (io.ReadCloser, error) {
		return sd.upstreamBlobFetcher(fetchCtx, imageRef, auth, layerIndex, "sha256:"+blobHex)
	})
	if upstreamErr == nil {
		if started {
			sd.upstreamBlobMiss.Add(1)
		}
		return blobPath, "upstream:" + registryHost, nil
	}
	for _, peer := range ps.Peers() {
		if err := sd.fetchFromPeer(ctx, peer, registryHost, repoPath, blobHex, blobPath, false); err == nil {
			sd.p2pBlobHits.Add(1)
			return blobPath, "spegel-p2p:" + peer, nil
		}
	}
	return "", "", fmt.Errorf("fetching blob sha256:%s from upstream registry: %w", blobHex, upstreamErr)
}

// fetchFromPeer downloads a blob from peer into blobPath and verifies its
// digest. With fetchUpstream, a peer that lacks the blob fetches it from
// upstream. It aborts if the first byte takes blobFirstByteTimeout or later
// bytes stall for blobStallTimeout.
func (sd *SpegelDriver) fetchFromPeer(ctx context.Context, peer, registryHost, repoPath, blobHex, blobPath string, fetchUpstream bool) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stall := time.AfterFunc(blobFirstByteTimeout, cancel)
	defer stall.Stop()

	peerURL := fmt.Sprintf("http://%s/v2/%s/blobs/sha256:%s?ns=%s",
		peer, strings.TrimPrefix(repoPath, "/"), blobHex, url.QueryEscape(registryHost))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, peerURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set(HeaderSpegelExternalRequest, "true")
	if fetchUpstream {
		req.Header.Set(HeaderSpegelFetchUpstream, "true")
	}
	resp, err := sd.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer %s returned HTTP %d", peer, resp.StatusCode)
	}
	body := stallReader{r: countingReader{resp.Body, &sd.counters.p2pBytesReceived}, timer: stall}
	return writeVerifiedBlobAtomically(body, blobPath, blobHex)
}

// stallReader resets timer to blobStallTimeout whenever r returns bytes.
type stallReader struct {
	r     io.Reader
	timer *time.Timer
}

func (s stallReader) Read(p []byte) (int, error) {
	n, err := s.r.Read(p)
	if n > 0 {
		s.timer.Reset(blobStallTimeout)
	}
	return n, err
}

func (sd *SpegelDriver) defaultUpstreamBlobFetcher(ctx context.Context, imageRef string, authConfig *imagepluginclient.AuthConfig, layerIndex int, layerDigest string) (io.ReadCloser, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, fmt.Errorf("parsing image reference %q: %w", imageRef, err)
	}
	remoteOpts := plugin.RemoteOptions(ctx, sd.Driver.Keychain(), authConfig)
	if strings.HasPrefix(layerDigest, "sha256:") {
		digestRef, dErr := name.NewDigest(fmt.Sprintf("%s@%s", ref.Context().Name(), layerDigest))
		if dErr == nil {
			layer, lErr := remote.Layer(digestRef, remoteOpts...)
			if lErr == nil {
				return layer.Compressed()
			}
			// Peer requests carry no layer index to fall back on.
			if layerIndex < 0 {
				return nil, fmt.Errorf("remote.Layer(%q): %w", digestRef, lErr)
			}
		}
	}
	img, err := remote.Image(ref, remoteOpts...)
	if err != nil {
		return nil, fmt.Errorf("remote.Image(%q): %w", imageRef, err)
	}
	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("listing layers for %q: %w", imageRef, err)
	}
	if layerIndex < 0 || layerIndex >= len(layers) {
		return nil, fmt.Errorf("layer index %d out of range (len=%d)", layerIndex, len(layers))
	}
	return layers[layerIndex].Compressed()
}

func writeVerifiedBlobAtomically(r io.Reader, destPath, expectedHex string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	tmpPath := fmt.Sprintf("%s.tmp-%x", destPath, time.Now().UnixNano())
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	cleanup := true
	defer func() {
		_ = f.Close()
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, h), r); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	gotHex := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(gotHex, expectedHex) {
		return fmt.Errorf("blob digest mismatch: got sha256:%s, want sha256:%s", gotHex, expectedHex)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func openDecompressedTarStream(r io.Reader) (io.ReadCloser, error) {
	br := bufio.NewReaderSize(r, 64*1024)
	magic, err := br.Peek(4)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		return nil, err
	}
	// Gzip magic: 0x1f 0x8b
	if len(magic) >= 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		gz, err := gzip.NewReader(br)
		if err != nil {
			return nil, err
		}
		return gz, nil
	}
	// Zstd magic: 0x28 0xb5 0x2f 0xfd
	if len(magic) >= 4 && magic[0] == 0x28 && magic[1] == 0xb5 && magic[2] == 0x2f && magic[3] == 0xfd {
		zr, err := zstd.NewReader(br)
		if err != nil {
			return nil, err
		}
		return zr.IOReadCloser(), nil
	}
	return io.NopCloser(br), nil
}

func parseRegistryAndRepo(imageRef string) (string, string, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return "", "", fmt.Errorf("parsing image reference %q: %w", imageRef, err)
	}
	return ref.Context().RegistryStr(), ref.Context().RepositoryStr(), nil
}

func stripSHA256Prefix(d string) string {
	return strings.TrimPrefix(strings.TrimSpace(d), "sha256:")
}

func isValidSHA256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func isFinalizedLayerDir(layerFSDir, finalizedMarker string) bool {
	if _, err := os.Stat(finalizedMarker); err != nil {
		return false
	}
	info, err := os.Stat(layerFSDir)
	return err == nil && info.IsDir()
}

type noopSnapshotsClient struct {
	snapshots.SnapshotsClient
}

func (n *noopSnapshotsClient) Stat(_ context.Context, _ *snapshots.StatSnapshotRequest, _ ...grpc.CallOption) (*snapshots.StatSnapshotResponse, error) {
	return nil, status.Error(codes.NotFound, "snapshot not found")
}

func (n *noopSnapshotsClient) Remove(_ context.Context, _ *snapshots.RemoveSnapshotRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
