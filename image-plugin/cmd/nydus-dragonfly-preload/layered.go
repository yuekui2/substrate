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
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/klauspost/compress/zstd"
)

// LayeredConfig configures layered mounts: each Nydus data layer is its own
// EROFS mount over the layer's bootstrap, shared by every image on the node
// that has the layer. Mounts are bound directly with the shared nydusd,
// without the snapshotter.
type LayeredConfig struct {
	// APISockGlob matches the shared nydusd's API socket.
	APISockGlob string
	// DaemonConfig is the nydusd config template the snapshotter also uses.
	DaemonConfig string
	// DomainID is the fscache domain the mounts share.
	DomainID string
	// WorkDir holds the layer bootstraps and mounts.
	WorkDir string
}

type layeredMounter struct {
	cfg       LayeredConfig
	seedURL   string
	seed      *http.Client
	keychain  authn.Keychain
	mountFS   MountFSFunc
	unmountFS UnmountFSFunc
	verify    func(string) error

	mu     sync.Mutex
	api    *http.Client
	gen    uint64
	layers map[string]*layerMount
	// images maps an image work dir to the layer mounts it holds.
	images map[string][]*layerMount
}

// layerMount is one layer's EROFS mount. done closes once mounting finishes;
// refs counts the images holding it.
type layerMount struct {
	digest string
	done   chan struct{}
	dir    string
	mnt    string
	fsid   string
	err    error
	refs   int
	bytes  int
}

func newLayeredMounter(cfg LayeredConfig, seedURL string, kc authn.Keychain, mountFS MountFSFunc, unmountFS UnmountFSFunc, verify func(string) error) *layeredMounter {
	return &layeredMounter{
		cfg:       cfg,
		seedURL:   seedURL,
		seed:      &http.Client{Timeout: 30 * time.Second},
		keychain:  kc,
		mountFS:   mountFS,
		unmountFS: unmountFS,
		verify:    verify,
		layers:    make(map[string]*layerMount),
		images:    make(map[string][]*layerMount),
	}
}

// mount mounts each of the image's data layers, reusing mounts other images
// hold, and returns the mount dirs bottom first. imageWorkDir holds a
// reference on each layer until release.
func (lm *layeredMounter) mount(ctx context.Context, imageRef, imageWorkDir string, dataDigests []string) ([]string, error) {
	if len(dataDigests) == 0 {
		return nil, errors.New("layered mount needs at least one data layer")
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
	lms := make([]*layerMount, len(dataDigests))
	hits := make([]bool, len(dataDigests))
	lm.mu.Lock()
	for i, d := range dataDigests {
		l, ok := lm.layers[d]
		if !ok {
			lm.gen++
			fsid := fsidFor(d, lm.gen)
			dir := filepath.Join(lm.cfg.WorkDir, fsid)
			l = &layerMount{digest: d, done: make(chan struct{}), dir: dir, mnt: filepath.Join(dir, "mnt"), fsid: fsid}
			lm.layers[d] = l
			go lm.mountLayer(context.WithoutCancel(ctx), l, repo, auth)
		}
		l.refs++
		lms[i], hits[i] = l, ok
	}
	lm.images[imageWorkDir] = append(lm.images[imageWorkDir], lms...)
	lm.mu.Unlock()

	dirs := make([]string, len(lms))
	var errs []error
	var fetched, hitCount int
	for i, l := range lms {
		select {
		case <-l.done:
		case <-ctx.Done():
			lm.release(imageWorkDir)
			return nil, ctx.Err()
		}
		if l.err != nil {
			errs = append(errs, fmt.Errorf("layer %s: %w", l.digest, l.err))
			continue
		}
		dirs[i] = l.mnt
		if hits[i] {
			hitCount++
		} else {
			fetched += l.bytes
		}
	}
	if err := errors.Join(errs...); err != nil {
		lm.release(imageWorkDir)
		return nil, err
	}
	slog.InfoContext(ctx, "Nydus layered mount timing",
		slog.String("image", imageRef),
		slog.Int("layers", len(dataDigests)),
		slog.Int("layer_hits", hitCount),
		slog.Int("bootstrap_bytes", fetched),
		slog.Float64("total_ms", ms(time.Since(start))))
	return topmostUnique(dirs), nil
}

// topmostUnique drops all but the topmost copy of a repeated layer dir. The
// lower copies add nothing, because the topmost copy shadows them.
func topmostUnique(dirs []string) []string {
	seen := make(map[string]bool, len(dirs))
	var out []string
	for i := len(dirs) - 1; i >= 0; i-- {
		if !seen[dirs[i]] {
			seen[dirs[i]] = true
			out = append(out, dirs[i])
		}
	}
	slices.Reverse(out)
	return out
}

// mountLayer fetches the layer's bootstrap from the seeds, binds it with
// nydusd and mounts it. A failed mount leaves the map so the next image
// retries.
func (lm *layeredMounter) mountLayer(ctx context.Context, l *layerMount, repo name.Repository, auth string) {
	defer close(l.done)
	t := time.Now()
	boot, err := lm.layerBootstrap(ctx, repo, l.digest)
	if err != nil {
		l.err = fmt.Errorf("reading the layer bootstrap: %w", err)
	} else {
		l.bytes = len(boot)
		l.err = lm.writeAndMount(ctx, l, boot, repo, auth)
	}
	if l.err != nil {
		lm.mu.Lock()
		if lm.layers[l.digest] == l {
			delete(lm.layers, l.digest)
		}
		lm.mu.Unlock()
		return
	}
	slog.InfoContext(ctx, "Nydus layer mounted",
		slog.String("layer", l.digest),
		slog.Int("bootstrap_bytes", len(boot)),
		slog.Float64("ms", ms(time.Since(t))))
}

func (lm *layeredMounter) writeAndMount(ctx context.Context, l *layerMount, boot []byte, repo name.Repository, auth string) error {
	if err := os.MkdirAll(l.dir, 0o755); err != nil {
		return err
	}
	hexID := strings.TrimPrefix(l.digest, "sha256:")
	boot, err := setBlobID(boot, hexID)
	if err != nil {
		return err
	}
	bootPath := filepath.Join(l.dir, hexID)
	if err := os.WriteFile(bootPath, boot, 0o644); err != nil {
		return err
	}
	return lm.bindAndMount(ctx, l.fsid, bootPath, l.dir, l.mnt, repo, auth)
}

// blobIDPlaceholder is the blob id nydus-image writes into a layer bootstrap
// before the blob digest is known (nydus builder/src/lib.rs).
var blobIDPlaceholder = bytes.Repeat([]byte("x"), 64)

// setBlobID replaces the placeholder blob id in a layer bootstrap with the
// layer's blob digest. The id appears in the EROFS device table, which the
// kernel uses to ask nydusd for the blob, and in the RAFS blob table.
func setBlobID(boot []byte, hexID string) ([]byte, error) {
	if len(hexID) != len(blobIDPlaceholder) {
		return nil, fmt.Errorf("blob id %q is not a sha256 hex digest", hexID)
	}
	if !bytes.Contains(boot, blobIDPlaceholder) {
		return nil, errors.New("layer bootstrap has no placeholder blob id")
	}
	return bytes.ReplaceAll(boot, blobIDPlaceholder, []byte(hexID)), nil
}

// release drops imageWorkDir's references and unmounts layers no image
// holds.
func (lm *layeredMounter) release(imageWorkDir string) {
	lm.mu.Lock()
	var drop []*layerMount
	for _, l := range lm.images[imageWorkDir] {
		l.refs--
		if l.refs == 0 {
			if lm.layers[l.digest] == l {
				delete(lm.layers, l.digest)
			}
			drop = append(drop, l)
		}
	}
	delete(lm.images, imageWorkDir)
	lm.mu.Unlock()
	for _, l := range drop {
		<-l.done
		if l.err != nil {
			continue
		}
		if err := lm.unmountFS(l.mnt, syscall.MNT_DETACH); err != nil {
			slog.Warn("Unmounting Nydus layer", slog.String("mnt", l.mnt), slog.Any("err", err))
		}
		if err := lm.unbind(context.Background(), l.fsid); err != nil {
			slog.Warn("Unbinding Nydus layer", slog.String("fsid", l.fsid), slog.Any("err", err))
		}
		if err := os.RemoveAll(l.dir); err != nil {
			slog.Warn("Removing Nydus layer dir", slog.String("dir", l.dir), slog.Any("err", err))
		}
	}
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

// fsidFor names one mount of a layer. gen keeps a remount apart from a
// previous mount of the same layer that is still being torn down.
func fsidFor(digest string, gen uint64) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("ate-layered\n%s\n%d", digest, gen)))
	return hex.EncodeToString(h[:])
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
