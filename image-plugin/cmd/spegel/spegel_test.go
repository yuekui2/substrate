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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/klauspost/compress/zstd"
	"go.ate.dev/substrate/image-plugin/pkg/plugin"
	"go.ate.dev/substrate/internal/imagepluginclient"
)

func TestResolveSpegelEndpoints(t *testing.T) {
	t.Setenv(EnvSpegelMirrorAddr, "")
	t.Setenv(EnvSpegelRouterAddr, "")

	mirror, err := ResolveSpegelMirrorAddr("")
	if err != nil {
		t.Fatalf("ResolveSpegelMirrorAddr default failed: %v", err)
	}
	if mirror != DefaultSpegelMirrorAddr {
		t.Fatalf("got %q, want %q", mirror, DefaultSpegelMirrorAddr)
	}

	router, err := ResolveSpegelRouterAddr("")
	if err != nil {
		t.Fatalf("ResolveSpegelRouterAddr default failed: %v", err)
	}
	if router != DefaultSpegelRouterAddr {
		t.Fatalf("got %q, want %q", router, DefaultSpegelRouterAddr)
	}

	t.Setenv(EnvSpegelMirrorAddr, "http://127.0.0.1:15000/")
	mirror, err = ResolveSpegelMirrorAddr("")
	if err != nil || mirror != "http://127.0.0.1:15000" {
		t.Fatalf("env override failed: mirror=%q err=%v", mirror, err)
	}

	if _, err := ResolveSpegelMirrorAddr("ftp://bad-scheme"); err == nil {
		t.Fatalf("expected error for invalid scheme")
	}
	if _, err := ResolveSpegelRouterAddr("missing-port"); err == nil {
		t.Fatalf("expected error for missing port")
	}
}

func TestSpegelHealthProbeAndCanProvide(t *testing.T) {
	var healthy atomic.Bool
	healthy.Store(false)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			if healthy.Load() {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
				return
			}
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	sd, err := NewSpegel(SpegelConfig{
		StoreDir:          filepath.Join(tmp, "store"),
		DockerConfigPath:  filepath.Join(tmp, "docker", "config.json"),
		SpegelMirrorAddr:  srv.URL,
		VerifySpegelReady: true,
		ServerOpts: []plugin.Option{
			plugin.WithWorkDir(filepath.Join(tmp, "work")),
		},
	})
	if err != nil {
		t.Fatalf("NewSpegel failed: %v", err)
	}
	defer func() { _ = sd.Close() }()

	ctx := context.Background()
	ok, err := sd.CanProvide(ctx, &imagepluginclient.PrepareRequest{ImageRef: "us-central1-docker.pkg.dev/proj/repo/app:v1"})
	if err != nil {
		t.Fatalf("CanProvide returned unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected CanProvide=false when Spegel /healthz returns 503")
	}

	healthy.Store(true)
	ok, err = sd.CanProvide(ctx, &imagepluginclient.PrepareRequest{ImageRef: "us-central1-docker.pkg.dev/proj/repo/app:v1"})
	if err != nil {
		t.Fatalf("CanProvide returned unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected CanProvide=true when Spegel /healthz returns 200")
	}
}

func TestNewSpegelClearStore(t *testing.T) {
	for _, tc := range []struct {
		name       string
		clearStore bool
		wantBlob   bool
	}{
		{name: "keeps the store by default", clearStore: false, wantBlob: true},
		{name: "clears the store when asked", clearStore: true, wantBlob: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			store := filepath.Join(tmp, "store")
			blob := filepath.Join(store, "blobs", "sha256", "abc")
			if err := os.MkdirAll(filepath.Dir(blob), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(blob, []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			sd, err := NewSpegel(SpegelConfig{
				StoreDir:         store,
				ClearStore:       tc.clearStore,
				DockerConfigPath: filepath.Join(tmp, "docker", "config.json"),
				ServerOpts:       []plugin.Option{plugin.WithWorkDir(filepath.Join(tmp, "work"))},
			})
			if err != nil {
				t.Fatalf("NewSpegel failed: %v", err)
			}
			defer func() { _ = sd.Close() }()
			_, err = os.Stat(blob)
			if got := err == nil; got != tc.wantBlob {
				t.Fatalf("blob present = %v, want %v (stat err: %v)", got, tc.wantBlob, err)
			}
			if _, err := os.Stat(filepath.Join(store, "layers", "sha256")); err != nil {
				t.Fatalf("layer store missing after NewSpegel: %v", err)
			}
		})
	}
}

func TestSpegelPeerServerPeers(t *testing.T) {
	for _, tc := range []struct {
		name        string
		nodeIP      string
		peerPort    int
		staticPeers []string
		wantPeers   []string
		wantSelf    string
	}{
		{
			name:        "sorts peers and drops self",
			nodeIP:      "10.0.0.3",
			staticPeers: []string{"10.0.0.2:5000", "10.0.0.3:5000", "10.0.0.1"},
			wantPeers:   []string{"10.0.0.1:5000", "10.0.0.2:5000"},
			wantSelf:    "10.0.0.3:5000",
		},
		{
			name:        "self uses the node IP and peer port",
			nodeIP:      "10.0.0.3",
			peerPort:    5003,
			staticPeers: []string{"10.0.0.3:5000"},
			wantPeers:   []string{"10.0.0.3:5000"},
			wantSelf:    "10.0.0.3:5003",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewSpegelPeerServer("127.0.0.1:0", t.TempDir(), "", tc.nodeIP, tc.peerPort, tc.staticPeers, "", nil, &spegelCounters{}, nil)
			if err != nil {
				t.Fatalf("NewSpegelPeerServer failed: %v", err)
			}
			defer func() { _ = s.Close() }()
			if got := s.Peers(); !slices.Equal(got, tc.wantPeers) {
				t.Errorf("Peers() = %v, want %v", got, tc.wantPeers)
			}
			if s.self != tc.wantSelf {
				t.Errorf("self = %q, want %q", s.self, tc.wantSelf)
			}
		})
	}
}

func TestSpegelCrossNodeP2PFlow(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()

	// Build two test layers: Layer 0 (gzip) and Layer 1 (zstd).
	tar0, diff0 := buildTestTar(t, map[string]string{
		"etc/os-release": "NAME=Ubuntu\nVERSION=24.04\n",
		"bin/base":       "base-binary-payload",
	})
	gzBlob0, blobDigest0 := compressGzip(t, tar0)

	tar1, diff1 := buildTestTar(t, map[string]string{
		"app/agent.py": "print('hello from spegel p2p')\n",
	})
	zstdBlob1, blobDigest1 := compressZstd(t, tar1)

	testImage := "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-e2e/agent:v1"
	imageDigest := "sha256:9999999999999999999999999999999999999999999999999999999999999999"

	resolver := func(_ context.Context, _ string, _ *imagepluginclient.AuthConfig) (string, *v1.Config, []string, []string, error) {
		return imageDigest, &v1.Config{Env: []string{"PATH=/usr/bin:/bin"}}, []string{diff0, diff1}, []string{blobDigest0, blobDigest1}, nil
	}

	var nodeAUpstreamCalls atomic.Int64
	nodeAFetcher := func(_ context.Context, _ string, _ *imagepluginclient.AuthConfig, layerIndex int, _ string) (io.ReadCloser, error) {
		nodeAUpstreamCalls.Add(1)
		switch layerIndex {
		case 0:
			return io.NopCloser(bytes.NewReader(gzBlob0)), nil
		case 1:
			return io.NopCloser(bytes.NewReader(zstdBlob1)), nil
		default:
			return nil, fmt.Errorf("unexpected layer index %d", layerIndex)
		}
	}

	// Node A: starts its embedded SpegelPeerServer on an ephemeral port.
	nodeA, err := NewSpegel(SpegelConfig{
		StoreDir:            filepath.Join(tmp, "nodeA-store"),
		DockerConfigPath:    filepath.Join(tmp, "nodeA-docker", "config.json"),
		EnableP2PServer:     true,
		P2PListenAddr:       "127.0.0.1:0",
		SpegelNodeIP:        "127.0.0.1",
		SpegelBootstrapDNS:  "nonexistent.invalid",
		VerifySpegelReady:   true,
		UpstreamBlobFetcher: nodeAFetcher,
		ServerOpts: []plugin.Option{
			plugin.WithWorkDir(filepath.Join(tmp, "nodeA-work")),
			plugin.WithImageResolver(resolver),
		},
	})
	if err != nil {
		t.Fatalf("NewSpegel(nodeA) failed: %v", err)
	}
	defer func() { _ = nodeA.Close() }()

	// Step 1: Cold PrepareImage on Node A pulls from upstream and populates Node A's Spegel blob store.
	resA, err := nodeA.PrepareImage(ctx, &imagepluginclient.PrepareRequest{
		ImageRef: testImage,
		AuthConfig: &imagepluginclient.AuthConfig{
			Username: "oauth2accesstoken",
			Password: "test-gar-token",
		},
	})
	if err != nil {
		t.Fatalf("nodeA.PrepareImage failed: %v", err)
	}
	if len(resA.LayerDirs) != 2 {
		t.Fatalf("got %d LayerDirs on Node A, want 2", len(resA.LayerDirs))
	}
	if nodeAUpstreamCalls.Load() != 2 || nodeA.UpstreamBlobMisses() != 2 {
		t.Fatalf("expected 2 upstream fetches on cold Node A, got calls=%d misses=%d", nodeAUpstreamCalls.Load(), nodeA.UpstreamBlobMisses())
	}

	// Node B: configured with Node A as a cluster Spegel peer, and with its
	// upstream registry fetcher disabled (failing) to prove 100% P2P transfer!
	nodeB, err := NewSpegel(SpegelConfig{
		StoreDir:           filepath.Join(tmp, "nodeB-store"),
		DockerConfigPath:   filepath.Join(tmp, "nodeB-docker", "config.json"),
		EnableP2PServer:    true,
		P2PListenAddr:      "127.0.0.1:0",
		SpegelNodeIP:       "127.0.0.1",
		SpegelBootstrapDNS: "nonexistent.invalid",
		SpegelPeers:        []string{nodeA.PeerServer().Addr()},
		VerifySpegelReady:  true,
		UpstreamBlobFetcher: func(_ context.Context, _ string, _ *imagepluginclient.AuthConfig, _ int, _ string) (io.ReadCloser, error) {
			return nil, errors.New("upstream registry intentionally offline on Node B")
		},
		ServerOpts: []plugin.Option{
			plugin.WithWorkDir(filepath.Join(tmp, "nodeB-work")),
			plugin.WithImageResolver(resolver),
		},
	})
	if err != nil {
		t.Fatalf("NewSpegel(nodeB) failed: %v", err)
	}
	defer func() { _ = nodeB.Close() }()

	// Step 2: PrepareImage on Node B streams both compressed blobs peer-to-peer from Node A!
	resB, err := nodeB.PrepareImage(ctx, &imagepluginclient.PrepareRequest{
		ImageRef: testImage,
	})
	if err != nil {
		t.Fatalf("nodeB.PrepareImage (P2P from Node A) failed: %v", err)
	}
	if len(resB.LayerDirs) != 2 {
		t.Fatalf("got %d LayerDirs on Node B, want 2", len(resB.LayerDirs))
	}
	if nodeB.P2PBlobHits() != 2 || nodeB.UpstreamBlobMisses() != 0 {
		t.Fatalf("expected P2PBlobHits=2 and UpstreamBlobMisses=0 on Node B, got hits=%d misses=%d",
			nodeB.P2PBlobHits(), nodeB.UpstreamBlobMisses())
	}

	totalBytes := int64(len(gzBlob0) + len(zstdBlob1))
	statsA := nodeA.PeerServer().Stats()
	if statsA.P2PRequestsIn < 2 || statsA.P2PBytesServed != totalBytes || statsA.UpstreamBytes != totalBytes {
		t.Fatalf("unexpected Node A P2P server stats: %+v (want P2PBytesServed=UpstreamBytes=%d)", statsA, totalBytes)
	}
	if statsB := nodeB.PeerServer().Stats(); statsB.P2PBytesReceived != totalBytes || statsB.UpstreamBytes != 0 {
		t.Fatalf("unexpected Node B P2P server stats: %+v (want P2PBytesReceived=%d)", statsB, totalBytes)
	}

	gotAgentPy, err := os.ReadFile(filepath.Join(resB.LayerDirs[1], "fs", "app", "agent.py"))
	if err != nil || string(gotAgentPy) != "print('hello from spegel p2p')\n" {
		t.Fatalf("unexpected file content on Node B layer-1: %q (err=%v)", string(gotAgentPy), err)
	}
}

func TestSpegelSharedLayerDeduplicationAndCorruptBlobFallback(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()

	tar0, diff0 := buildTestTar(t, map[string]string{
		"etc/hosts": "127.0.0.1 localhost\n",
	})
	gzBlob0, blobDigest0 := compressGzip(t, tar0)

	// Corrupt peer mirror that returns invalid bytes for blobDigest0.
	corruptMirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("corrupted-blob-bytes"))
	}))
	defer corruptMirror.Close()

	var upstreamCalls atomic.Int64
	sd, err := NewSpegel(SpegelConfig{
		StoreDir:          filepath.Join(tmp, "store"),
		DockerConfigPath:  filepath.Join(tmp, "docker", "config.json"),
		SpegelMirrorAddr:  corruptMirror.URL,
		VerifySpegelReady: true,
		UpstreamBlobFetcher: func(_ context.Context, _ string, _ *imagepluginclient.AuthConfig, _ int, _ string) (io.ReadCloser, error) {
			upstreamCalls.Add(1)
			return io.NopCloser(bytes.NewReader(gzBlob0)), nil
		},
		ServerOpts: []plugin.Option{
			plugin.WithWorkDir(filepath.Join(tmp, "work")),
			plugin.WithImageResolver(func(_ context.Context, ref string, _ *imagepluginclient.AuthConfig) (string, *v1.Config, []string, []string, error) {
				if ref == "us-central1-docker.pkg.dev/p/r/img1:v1" {
					return "sha256:1111111111111111111111111111111111111111111111111111111111111111", &v1.Config{}, []string{diff0}, []string{blobDigest0}, nil
				}
				return "sha256:2222222222222222222222222222222222222222222222222222222222222222", &v1.Config{}, []string{diff0}, []string{blobDigest0}, nil
			}),
		},
	})
	if err != nil {
		t.Fatalf("NewSpegel failed: %v", err)
	}
	defer func() { _ = sd.Close() }()

	// Image 1: corrupt P2P blob is rejected and falls back cleanly to upstream.
	if _, err := sd.PrepareImage(ctx, &imagepluginclient.PrepareRequest{ImageRef: "us-central1-docker.pkg.dev/p/r/img1:v1"}); err != nil {
		t.Fatalf("PrepareImage(img1) failed: %v", err)
	}
	if upstreamCalls.Load() != 1 {
		t.Fatalf("expected 1 upstream call after corrupt P2P fallback, got %d", upstreamCalls.Load())
	}

	// Image 2 sharing diff0: reuses already-unpacked layer on disk with 0 extra fetches.
	if _, err := sd.PrepareImage(ctx, &imagepluginclient.PrepareRequest{ImageRef: "us-central1-docker.pkg.dev/p/r/img2:v1"}); err != nil {
		t.Fatalf("PrepareImage(img2) failed: %v", err)
	}
	if upstreamCalls.Load() != 1 {
		t.Fatalf("expected shared layer to reuse on-disk unpacked layer without extra upstream calls, got %d", upstreamCalls.Load())
	}
}

func TestSpegelOneUpstreamFetchPerBlob(t *testing.T) {
	addrs := reserveAddrs(t, 3)
	blob, blobDigest, diffID := layerOwnedBy(t, addrs, addrs[0])
	up := newGatedUpstream(blob)
	// Open the gate once both non-owners get a 200 from the owner, while
	// the owner's upstream fetch is still blocked.
	var ownerResponses atomic.Int64
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err == nil && resp.StatusCode == http.StatusOK && req.Header.Get(HeaderSpegelFetchUpstream) != "" && ownerResponses.Add(1) == 2 {
			up.open()
		}
		return resp, err
	})}
	var nodes []*SpegelDriver
	for _, addr := range addrs {
		nodes = append(nodes, newTestNode(t, addr, addrs, up.fetch, client, diffID, blobDigest))
	}
	t.Cleanup(up.open)

	errs := make(chan error, len(nodes))
	for _, n := range nodes {
		go func() {
			_, err := n.PrepareImage(context.Background(), &imagepluginclient.PrepareRequest{ImageRef: testNodeImage})
			errs <- err
		}()
	}
	for range nodes {
		if err := <-errs; err != nil {
			t.Fatalf("PrepareImage failed: %v", err)
		}
	}
	if got := up.calls.Load(); got != 1 {
		t.Fatalf("upstream fetches = %d, want 1", got)
	}
}

func TestSpegelServesBlobWhileOwnerFetches(t *testing.T) {
	addrs := reserveAddrs(t, 2)
	blob, blobDigest, diffID := layerOwnedBy(t, addrs, addrs[0])
	up := newGatedUpstream(blob)
	firstBytes := make(chan struct{})
	var once sync.Once
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err == nil && req.Header.Get(HeaderSpegelFetchUpstream) != "" {
			resp.Body = notifyingBody{resp.Body, func() { once.Do(func() { close(firstBytes) }) }}
		}
		return resp, err
	})}
	var nonOwnerUpstreamCalls atomic.Int64
	failingUpstream := func(context.Context, string, *imagepluginclient.AuthConfig, int, string) (io.ReadCloser, error) {
		nonOwnerUpstreamCalls.Add(1)
		return nil, errors.New("upstream offline")
	}
	newTestNode(t, addrs[0], addrs, up.fetch, nil, diffID, blobDigest)
	nonOwner := newTestNode(t, addrs[1], addrs, failingUpstream, client, diffID, blobDigest)
	t.Cleanup(up.open)

	done := make(chan error, 1)
	go func() {
		_, err := nonOwner.PrepareImage(context.Background(), &imagepluginclient.PrepareRequest{ImageRef: testNodeImage})
		done <- err
	}()
	select {
	case <-firstBytes:
	case err := <-done:
		t.Fatalf("PrepareImage returned %v before the owner streamed any bytes", err)
	}
	up.open()
	if err := <-done; err != nil {
		t.Fatalf("PrepareImage failed: %v", err)
	}
	if up.calls.Load() != 1 || nonOwnerUpstreamCalls.Load() != 0 {
		t.Fatalf("upstream fetches: owner = %d, non-owner = %d, want 1 and 0", up.calls.Load(), nonOwnerUpstreamCalls.Load())
	}
}

func TestSpegelOwnerFailureFallsBackToUpstream(t *testing.T) {
	for _, tc := range []struct {
		name  string
		owner func(t *testing.T) string
	}{
		{
			name:  "owner unreachable",
			owner: func(t *testing.T) string { return reserveAddrs(t, 1)[0] },
		},
		{
			name: "owner sends a corrupt blob",
			owner: func(t *testing.T) string {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write([]byte("corrupt"))
				}))
				t.Cleanup(srv.Close)
				return srv.Listener.Addr().String()
			},
		},
		{
			name: "owner returns not found",
			owner: func(t *testing.T) string {
				srv := httptest.NewServer(http.NotFoundHandler())
				t.Cleanup(srv.Close)
				return srv.Listener.Addr().String()
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			owner := tc.owner(t)
			self := reserveAddrs(t, 1)[0]
			blob, blobDigest, diffID := layerOwnedBy(t, []string{self, owner}, owner)
			var askedOwner atomic.Bool
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Host == owner && req.Header.Get(HeaderSpegelFetchUpstream) != "" {
					askedOwner.Store(true)
				}
				return http.DefaultTransport.RoundTrip(req)
			})}
			var upstreamCalls atomic.Int64
			upstream := func(context.Context, string, *imagepluginclient.AuthConfig, int, string) (io.ReadCloser, error) {
				upstreamCalls.Add(1)
				return io.NopCloser(bytes.NewReader(blob)), nil
			}
			node := newTestNode(t, self, []string{owner}, upstream, client, diffID, blobDigest)
			if _, err := node.PrepareImage(context.Background(), &imagepluginclient.PrepareRequest{ImageRef: testNodeImage}); err != nil {
				t.Fatalf("PrepareImage failed: %v", err)
			}
			if !askedOwner.Load() || upstreamCalls.Load() != 1 {
				t.Fatalf("asked owner = %v, upstream fetches = %d, want true and 1", askedOwner.Load(), upstreamCalls.Load())
			}
		})
	}
}

const testNodeImage = "us-central1-docker.pkg.dev/p/r/app:v1"

// newTestNode starts a Spegel node listening on addr with a one-layer image.
func newTestNode(t *testing.T, addr string, peers []string, upstream BlobFetcherFunc, client *http.Client, diffID, blobDigest string) *SpegelDriver {
	t.Helper()
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	sd, err := NewSpegel(SpegelConfig{
		StoreDir:            filepath.Join(dir, "store"),
		DockerConfigPath:    filepath.Join(dir, "docker", "config.json"),
		EnableP2PServer:     true,
		P2PListenAddr:       addr,
		SpegelNodeIP:        "127.0.0.1",
		SpegelPeerPort:      port,
		SpegelBootstrapDNS:  "nonexistent.invalid",
		SpegelPeers:         peers,
		HTTPClient:          client,
		UpstreamBlobFetcher: upstream,
		ServerOpts: []plugin.Option{
			plugin.WithWorkDir(filepath.Join(dir, "work")),
			plugin.WithImageResolver(func(context.Context, string, *imagepluginclient.AuthConfig) (string, *v1.Config, []string, []string, error) {
				return "sha256:3333333333333333333333333333333333333333333333333333333333333333", &v1.Config{}, []string{diffID}, []string{blobDigest}, nil
			}),
		},
	})
	if err != nil {
		t.Fatalf("NewSpegel(%s) failed: %v", addr, err)
	}
	t.Cleanup(func() { _ = sd.Close() })
	return sd
}

// reserveAddrs returns n free loopback addresses.
func reserveAddrs(t *testing.T, n int) []string {
	t.Helper()
	var addrs []string
	for range n {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = ln.Close() }()
		addrs = append(addrs, ln.Addr().String())
	}
	return addrs
}

// layerOwnedBy builds a gzip layer whose blob rendezvousOwner assigns to
// owner among members.
func layerOwnedBy(t *testing.T, members []string, owner string) (blob []byte, blobDigest, diffID string) {
	t.Helper()
	for i := range 1000 {
		tarball, diff := buildTestTar(t, map[string]string{"nonce": strconv.Itoa(i)})
		blob, blobDigest := compressGzip(t, tarball)
		if rendezvousOwner(members, stripSHA256Prefix(blobDigest)) == owner {
			return blob, blobDigest, diff
		}
	}
	t.Fatalf("no layer owned by %s", owner)
	return nil, "", ""
}

// gatedUpstream serves blob from upstream: the first half, then the rest
// once open is called. A second fetch opens the gate, so a test expecting
// one fetch fails instead of hanging.
type gatedUpstream struct {
	blob    []byte
	calls   atomic.Int64
	release chan struct{}
	once    sync.Once
}

func newGatedUpstream(blob []byte) *gatedUpstream {
	return &gatedUpstream{blob: blob, release: make(chan struct{})}
}

func (g *gatedUpstream) open() { g.once.Do(func() { close(g.release) }) }

func (g *gatedUpstream) fetch(context.Context, string, *imagepluginclient.AuthConfig, int, string) (io.ReadCloser, error) {
	if g.calls.Add(1) > 1 {
		g.open()
	}
	half := len(g.blob) / 2
	return io.NopCloser(io.MultiReader(bytes.NewReader(g.blob[:half]), gateReader(g.release), bytes.NewReader(g.blob[half:]))), nil
}

// gateReader blocks until the channel is closed, then reports EOF.
type gateReader chan struct{}

func (g gateReader) Read([]byte) (int, error) {
	<-g
	return 0, io.EOF
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// notifyingBody calls notify whenever a read returns bytes.
type notifyingBody struct {
	io.ReadCloser
	notify func()
}

func (b notifyingBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 {
		b.notify()
	}
	return n, err
}

func buildTestTar(t *testing.T, files map[string]string) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("WriteHeader(%s): %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("Write(%s): %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar.Close: %v", err)
	}
	raw := buf.Bytes()
	sum := sha256.Sum256(raw)
	return raw, "sha256:" + hex.EncodeToString(sum[:])
}

func compressGzip(t *testing.T, raw []byte) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(raw); err != nil {
		t.Fatalf("gzip.Write: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip.Close: %v", err)
	}
	compressed := buf.Bytes()
	sum := sha256.Sum256(compressed)
	return compressed, "sha256:" + hex.EncodeToString(sum[:])
}

func compressZstd(t *testing.T, raw []byte) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	zw, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatalf("zstd.NewWriter: %v", err)
	}
	if _, err := zw.Write(raw); err != nil {
		t.Fatalf("zstd.Write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zstd.Close: %v", err)
	}
	compressed := buf.Bytes()
	sum := sha256.Sum256(compressed)
	return compressed, "sha256:" + hex.EncodeToString(sum[:])
}
