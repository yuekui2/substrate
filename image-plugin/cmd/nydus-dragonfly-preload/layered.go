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
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/klauspost/compress/zstd"
)

// errNoParentBootstrap means the seeds have no bootstrap for the image's
// parent chain, so the image is mounted through the snapshotter instead.
var errNoParentBootstrap = errors.New("no parent bootstrap on the seeds")

// LayeredConfig configures layered mounts: the parent chain (every data layer
// but the top) is one EROFS mount shared by all images on the node with the
// same chain, and the top layer is a second EROFS mount per image. Both are
// bound directly with the shared nydusd, without the snapshotter.
type LayeredConfig struct {
	// NydusImage is the nydus-image binary. It rewrites the top layer's
	// bootstrap, whose embedded blob id is a placeholder, with the real one.
	NydusImage string
	// APISockGlob matches the shared nydusd's API socket.
	APISockGlob string
	// DaemonConfig is the nydusd config template the snapshotter also uses.
	DaemonConfig string
	// DomainID is the fscache domain the mounts share.
	DomainID string
	// WorkDir holds the parent chain bootstraps and mounts.
	WorkDir string
}

type layeredMounter struct {
	cfg       LayeredConfig
	seedURL   string
	seed      *http.Client
	workDir   string
	keychain  authn.Keychain
	mountFS   MountFSFunc
	unmountFS UnmountFSFunc
	verify    func(string) error

	mu      sync.Mutex
	api     *http.Client
	parents map[string]*parentMount
	tops    map[string][]layeredTop
}

type parentMount struct {
	done chan struct{}
	mnt  string
	err  error
}

type layeredTop struct {
	fsid string
	mnt  string
}

func newLayeredMounter(cfg LayeredConfig, seedURL string, kc authn.Keychain, mountFS MountFSFunc, unmountFS UnmountFSFunc, verify func(string) error) *layeredMounter {
	return &layeredMounter{
		cfg:       cfg,
		seedURL:   seedURL,
		seed:      &http.Client{Timeout: 30 * time.Second},
		workDir:   cfg.WorkDir,
		keychain:  kc,
		mountFS:   mountFS,
		unmountFS: unmountFS,
		verify:    verify,
		parents:   make(map[string]*parentMount),
		tops:      make(map[string][]layeredTop),
	}
}

// mount mounts the image's data layers as the parent chain and the top layer
// and returns the two mount dirs, bottom first. imageWorkDir owns the top
// mount until release.
func (lm *layeredMounter) mount(ctx context.Context, imageRef, imageWorkDir string, dataDigests []string) ([]string, error) {
	if len(dataDigests) < 2 {
		return nil, fmt.Errorf("layered mount needs a parent chain and a top layer, got %d data layers", len(dataDigests))
	}
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, err
	}
	repo := ref.Context()
	auth, err := lm.registryAuth(repo)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	parentDigests, top := dataDigests[:len(dataDigests)-1], dataDigests[len(dataDigests)-1]
	parentMnt, hit, err := lm.parent(ctx, repo, auth, parentDigests)
	if err != nil {
		return nil, err
	}
	parentDur := time.Since(start)

	t := time.Now()
	dir := filepath.Join(imageWorkDir, "layered-top")
	topHex := strings.TrimPrefix(top, "sha256:")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	boot, err := lm.layerBootstrap(ctx, repo, top)
	if err != nil {
		return nil, fmt.Errorf("reading top layer bootstrap %s: %w", top, err)
	}
	// nydus-image takes the blob id from the file name.
	if err := os.WriteFile(filepath.Join(dir, topHex), boot, 0o644); err != nil {
		return nil, err
	}
	bootPath := filepath.Join(dir, "image.boot")
	if out, err := exec.CommandContext(ctx, lm.cfg.NydusImage, "merge", "--bootstrap", bootPath, filepath.Join(dir, topHex)).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("setting the blob id of the top layer bootstrap: %w: %s", err, out)
	}
	fetchDur := time.Since(t)

	t = time.Now()
	fsid := fsidFor("top", imageRef+"\n"+top)
	topMnt := filepath.Join(dir, "mnt")
	if err := lm.bindAndMount(ctx, fsid, bootPath, dir, topMnt, repo, auth); err != nil {
		return nil, err
	}
	lm.mu.Lock()
	lm.tops[imageWorkDir] = append(lm.tops[imageWorkDir], layeredTop{fsid: fsid, mnt: topMnt})
	lm.mu.Unlock()

	slog.InfoContext(ctx, "Nydus layered mount timing",
		slog.String("image", imageRef),
		slog.Bool("parent_hit", hit),
		slog.Float64("parent_ms", ms(parentDur)),
		slog.Float64("top_fetch_ms", ms(fetchDur)),
		slog.Float64("top_mount_ms", ms(time.Since(t))),
		slog.Int("top_bootstrap_bytes", len(boot)),
		slog.Float64("total_ms", ms(time.Since(start))))
	return []string{parentMnt, topMnt}, nil
}

// release unmounts and unbinds the top mounts imageWorkDir owns. Parent
// mounts stay for later images with the same chain.
func (lm *layeredMounter) release(imageWorkDir string) {
	lm.mu.Lock()
	tops := lm.tops[imageWorkDir]
	delete(lm.tops, imageWorkDir)
	lm.mu.Unlock()
	for _, t := range tops {
		if err := lm.unmountFS(t.mnt, syscall.MNT_DETACH); err != nil {
			slog.Warn("Unmounting layered top", slog.String("mnt", t.mnt), slog.Any("err", err))
		}
		if err := lm.unbind(context.Background(), t.fsid); err != nil {
			slog.Warn("Unbinding layered top", slog.String("fsid", t.fsid), slog.Any("err", err))
		}
	}
}

// parent returns the mount of the parent chain, mounting it on first use.
// Concurrent callers for the same chain wait for one mount.
func (lm *layeredMounter) parent(ctx context.Context, repo name.Repository, auth string, digests []string) (string, bool, error) {
	key := parentKey(digests)
	lm.mu.Lock()
	pm, ok := lm.parents[key]
	if !ok {
		pm = &parentMount{done: make(chan struct{})}
		lm.parents[key] = pm
	}
	lm.mu.Unlock()
	if ok {
		select {
		case <-pm.done:
			return pm.mnt, true, pm.err
		case <-ctx.Done():
			return "", false, ctx.Err()
		}
	}
	pm.mnt, pm.err = lm.mountParent(ctx, key, repo, auth)
	if pm.err != nil {
		lm.mu.Lock()
		delete(lm.parents, key)
		lm.mu.Unlock()
	}
	close(pm.done)
	return pm.mnt, false, pm.err
}

func (lm *layeredMounter) mountParent(ctx context.Context, key string, repo name.Repository, auth string) (string, error) {
	dir := filepath.Join(lm.workDir, "parent-"+key)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	u := lm.seedURL + "/parentboots/sha256:" + key
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	resp, err := lm.seed.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("%w: %s", errNoParentBootstrap, key)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: HTTP %d", u, resp.StatusCode)
	}
	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", u, err)
	}
	bootPath := filepath.Join(dir, "image.boot")
	if err := writeFileFrom(bootPath, zr); err != nil {
		return "", fmt.Errorf("GET %s: %w", u, err)
	}
	mnt := filepath.Join(dir, "mnt")
	if err := lm.bindAndMount(ctx, fsidFor("parent", key), bootPath, dir, mnt, repo, auth); err != nil {
		return "", err
	}
	return mnt, nil
}

// bindAndMount binds the bootstrap with nydusd as fsid in the shared domain
// and mounts it as EROFS at mnt, the way nydus-snapshotter does for fscache.
func (lm *layeredMounter) bindAndMount(ctx context.Context, fsid, bootPath, workDir, mnt string, repo name.Repository, auth string) error {
	body, err := lm.bindConfig(fsid, bootPath, workDir, repo, auth)
	if err != nil {
		return err
	}
	if err := lm.apiCall(ctx, http.MethodPut, "/api/v2/blobs", body); err != nil {
		return fmt.Errorf("binding %s with nydusd: %w", bootPath, err)
	}
	if err := os.MkdirAll(mnt, 0o755); err != nil {
		return err
	}
	opts := "domain_id=" + lm.cfg.DomainID + ",fsid=" + fsid
	if err := lm.mountFS("erofs", mnt, "erofs", syscall.MS_RDONLY, opts); err != nil {
		_ = lm.unbind(context.Background(), fsid)
		return fmt.Errorf("mounting EROFS (%s) at %s: %w", opts, mnt, err)
	}
	if err := lm.verify(mnt); err != nil {
		_ = lm.unmountFS(mnt, syscall.MNT_DETACH)
		_ = lm.unbind(context.Background(), fsid)
		return err
	}
	return nil
}

func (lm *layeredMounter) unbind(ctx context.Context, fsid string) error {
	q := url.Values{"domain_id": {lm.cfg.DomainID}, "blob_id": {fsid}}
	if err := lm.apiCall(ctx, http.MethodDelete, "/api/v2/blobs?"+q.Encode(), nil); err != nil {
		return err
	}
	// Culls the bootstrap's own fscache cookie, as the snapshotter does.
	return lm.apiCall(ctx, http.MethodDelete, "/api/v2/blobs?"+url.Values{"blob_id": {fsid}}.Encode(), nil)
}

// bindConfig fills the nydusd config template for one bootstrap.
func (lm *layeredMounter) bindConfig(fsid, bootPath, workDir string, repo name.Repository, auth string) ([]byte, error) {
	raw, err := os.ReadFile(lm.cfg.DaemonConfig)
	if err != nil {
		return nil, err
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", lm.cfg.DaemonConfig, err)
	}
	inner, ok := cfg["config"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s has no config object", lm.cfg.DaemonConfig)
	}
	backend, ok := inner["backend_config"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s has no backend_config object", lm.cfg.DaemonConfig)
	}
	cfg["id"] = fsid
	cfg["domain_id"] = lm.cfg.DomainID
	inner["id"] = fsid
	inner["metadata_path"] = bootPath
	inner["cache_config"] = map[string]any{"work_dir": workDir}
	// nydusd rejects a bind without bandwidth_rate. These match the
	// snapshotter's per-snapshot config.
	inner["prefetch_config"] = map[string]any{
		"enable":         false,
		"threads_count":  8,
		"merging_size":   1048576,
		"bandwidth_rate": 0,
	}
	backend["readahead"] = false
	backend["proxy"] = map[string]any{"fallback": false}
	backend["host"] = repo.RegistryStr()
	backend["repo"] = repo.RepositoryStr()
	if auth != "" {
		backend["auth"] = auth
	}
	return json.Marshal(cfg)
}

func (lm *layeredMounter) registryAuth(repo name.Repository) (string, error) {
	if lm.keychain == nil {
		return "", nil
	}
	a, err := lm.keychain.Resolve(repo)
	if err != nil {
		return "", err
	}
	c, err := a.Authorization()
	if err != nil {
		return "", err
	}
	if c.Username == "" && c.Password == "" {
		return "", nil
	}
	return base64.StdEncoding.EncodeToString([]byte(c.Username + ":" + c.Password)), nil
}

func (lm *layeredMounter) apiCall(ctx context.Context, method, path string, body []byte) error {
	client, err := lm.apiClient()
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://unix"+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s %s: HTTP %d: %s", method, path, resp.StatusCode, msg)
	}
	return nil
}

func (lm *layeredMounter) apiClient() (*http.Client, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if lm.api != nil {
		return lm.api, nil
	}
	socks, err := filepath.Glob(lm.cfg.APISockGlob)
	if err != nil {
		return nil, err
	}
	if len(socks) != 1 {
		return nil, fmt.Errorf("want one nydusd API socket matching %s, found %v", lm.cfg.APISockGlob, socks)
	}
	sock := socks[0]
	lm.api = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", sock)
			},
		},
	}
	return lm.api, nil
}

// tocEntrySize is the size of one entry in a Nydus blob ToC
// (nydus storage/src/meta/toc.rs TocEntry).
const tocEntrySize = 128

// layerTailSize is read from the end of a blob first. It covers the ToC and,
// for small layers, the layer's bootstrap.
const layerTailSize = 64 << 10

// layerBootstrap reads the per-layer RAFS bootstrap embedded in a Nydus data
// blob on the seeds. The blob ends with the ToC entries followed by a tar
// header named rafs.blob.toc, whose size field is the ToC length; the
// image.boot entry locates the zstd-compressed bootstrap.
func (lm *layeredMounter) layerBootstrap(ctx context.Context, repo name.Repository, digest string) ([]byte, error) {
	u := lm.seedURL + "/v2/" + repo.RepositoryStr() + "/blobs/" + digest
	tail, start, err := lm.rangeGet(ctx, u, fmt.Sprintf("bytes=-%d", layerTailSize))
	if err != nil {
		return nil, err
	}
	if len(tail) < 512 {
		return nil, fmt.Errorf("blob tail is %d bytes", len(tail))
	}
	hdr := tail[len(tail)-512:]
	if string(bytes.TrimRight(hdr[:100], "\x00")) != "rafs.blob.toc" {
		return nil, fmt.Errorf("last tar header is %q, not rafs.blob.toc", bytes.TrimRight(hdr[:100], "\x00"))
	}
	tocLen, err := strconv.ParseInt(strings.Trim(string(hdr[124:136]), "\x00 "), 8, 64)
	if err != nil {
		return nil, fmt.Errorf("ToC size: %w", err)
	}
	if tocLen > int64(len(tail)-512) || tocLen%tocEntrySize != 0 {
		return nil, fmt.Errorf("ToC of %d bytes does not fit the %d byte tail", tocLen, len(tail))
	}
	toc := tail[int64(len(tail)-512)-tocLen : len(tail)-512]
	for i := 0; i+tocEntrySize <= len(toc); i += tocEntrySize {
		e := toc[i : i+tocEntrySize]
		if string(bytes.TrimRight(e[8:24], "\x00")) != "image.boot" {
			continue
		}
		flags := binary.LittleEndian.Uint32(e[0:4])
		want := e[24:56]
		off := int64(binary.LittleEndian.Uint64(e[56:64]))
		csize := int64(binary.LittleEndian.Uint64(e[64:72]))
		usize := int64(binary.LittleEndian.Uint64(e[72:80]))
		var data []byte
		if off >= start && off+csize <= start+int64(len(tail)) {
			data = tail[off-start : off-start+csize]
		} else if data, _, err = lm.rangeGet(ctx, u, fmt.Sprintf("bytes=%d-%d", off, off+csize-1)); err != nil {
			return nil, err
		}
		if flags&0x2 != 0 {
			dec, err := zstd.NewReader(nil)
			if err != nil {
				return nil, err
			}
			data, err = dec.DecodeAll(data, nil)
			dec.Close()
			if err != nil {
				return nil, fmt.Errorf("decompressing image.boot: %w", err)
			}
		}
		if int64(len(data)) != usize {
			return nil, fmt.Errorf("image.boot is %d bytes, ToC says %d", len(data), usize)
		}
		if got := sha256.Sum256(data); !bytes.Equal(got[:], want) {
			return nil, fmt.Errorf("image.boot sha256 %x, ToC says %x", got, want)
		}
		return data, nil
	}
	return nil, errors.New("no image.boot in the blob ToC")
}

// rangeGet GETs a byte range and returns it with its start offset in the
// object.
func (lm *layeredMounter) rangeGet(ctx context.Context, u, rng string) ([]byte, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Range", rng)
	resp, err := lm.seed.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusPartialContent {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, 0, fmt.Errorf("GET %s %s: HTTP %d", u, rng, resp.StatusCode)
	}
	// Content-Range: bytes <start>-<end>/<size>
	var startOff, end, size int64
	if _, err := fmt.Sscanf(resp.Header.Get("Content-Range"), "bytes %d-%d/%d", &startOff, &end, &size); err != nil {
		return nil, 0, fmt.Errorf("GET %s: Content-Range %q: %w", u, resp.Header.Get("Content-Range"), err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	return b, startOff, nil
}

// parentKey identifies a parent chain by its layer digests.
func parentKey(digests []string) string {
	h := sha256.Sum256([]byte(strings.Join(digests, "\n")))
	return hex.EncodeToString(h[:])
}

func fsidFor(kind, key string) string {
	h := sha256.Sum256([]byte("ate-layered-" + kind + "\n" + key))
	return hex.EncodeToString(h[:])
}

func writeFileFrom(path string, r io.Reader) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// layeredDataDigests returns the image's Nydus data layer digests, bottom
// first, from the layer annotations.
func layeredDataDigests(layerDigests []string, anns []map[string]string) ([]string, error) {
	if len(anns) != len(layerDigests) {
		return nil, fmt.Errorf("%d layer annotations for %d layers", len(anns), len(layerDigests))
	}
	var out []string
	for i, a := range anns {
		if a[LabelNydusBlob] == "true" && a[LabelNydusBootstrap] != "true" {
			out = append(out, layerDigests[i])
		}
	}
	return out, nil
}
