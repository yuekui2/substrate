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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ai-on-gke/image-provider-plugin/client"
	"github.com/ai-on-gke/image-provider-plugin/plugin"
	snapshotsapi "github.com/containerd/containerd/api/services/snapshots/v1"
	"github.com/containerd/containerd/api/types"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// providerName is the name the plugin reports.
	providerName = "nydus"

	// defaultSnapshotterSocket is containerd-nydus-grpc's socket.
	defaultSnapshotterSocket = "/run/containerd-nydus/containerd-nydus-grpc.sock"

	// defaultDockerConfigPath is the Docker config.json that
	// containerd-nydus-grpc reads registry credentials from.
	defaultDockerConfigPath = "/run/containerd-nydus/docker/config.json"

	// defaultDragonflyProxyAddr is the registry mirror of the node's
	// dfdaemon.
	defaultDragonflyProxyAddr = "http://127.0.0.1:4001"

	// defaultDragonflyHealthURL is the health endpoint of the node's
	// dfdaemon.
	defaultDragonflyHealthURL = "http://127.0.0.1:4003/healthy"

	// erofsSuperMagic is the statfs f_type of EROFS.
	erofsSuperMagic = 0xE0F5E1E2

	// labelNydusBootstrap marks the layer that holds the RAFS v6 bootstrap,
	// the EROFS image of the whole filesystem tree.
	labelNydusBootstrap = "containerd.io/snapshot/nydus-bootstrap"

	// labelNydusBlob marks a Nydus data blob layer.
	labelNydusBlob = "containerd.io/snapshot/nydus-blob"

	// labelNydusFSVersion holds the RAFS version, "6" for EROFS.
	labelNydusFSVersion = "containerd.io/snapshot/nydus-fs-version"

	// nydusBlobMediaType is the media type of a Nydus data blob layer.
	nydusBlobMediaType = "application/vnd.oci.image.layer.nydus.blob.v1"
)

// layerReaderFunc returns the uncompressed tar of one layer of an image.
type layerReaderFunc func(ctx context.Context, imageRef string, auth *client.AuthConfig, layerIndex int, layerDigest string) (io.ReadCloser, error)

// layerAnnotationsFunc returns the annotations of each layer of an image, in
// manifest order.
type layerAnnotationsFunc func(ctx context.Context, imageRef string, auth *client.AuthConfig) ([]map[string]string, error)

// mountFunc mounts a filesystem, like syscall.Mount.
type mountFunc func(source, target, fstype string, flags uintptr, data string) error

// unmountFunc unmounts a filesystem, like syscall.Unmount.
type unmountFunc func(target string, flags int) error

// statFSFunc returns the filesystem type of path.
type statFSFunc func(path string) (int64, error)

// config configures a nydusDriver. Empty fields take their defaults.
type config struct {
	DockerConfigPath    string
	DragonflyProxyAddr  string
	DragonflyHealthURL  string
	VerifyDragonflyPing bool
	ProcFilesystemsPath string
	CachefilesDevPath   string
	LayerReader         layerReaderFunc
	LayerAnnotations    layerAnnotationsFunc
	Mount               mountFunc
	Unmount             unmountFunc
	StatFS              statFSFunc
	DriverOpts          []plugin.Option
}

// nydusDriver prepares Nydus RAFS v6 images with containerd-nydus-grpc in
// EROFS over fscache mode, with nydusd fetching data through the node's
// Dragonfly dfdaemon.
type nydusDriver struct {
	*plugin.Driver
	dockerSyncer        *dockerConfigSyncer
	dragonflyHealthURL  string
	verifyDragonflyPing bool
	httpClient          *http.Client
	blobTransport       http.RoundTripper
	layerReader         layerReaderFunc
	layerAnnotations    layerAnnotationsFunc
	mount               mountFunc
	unmount             unmountFunc
	statFS              statFSFunc

	mu sync.Mutex
	// annotations caches the layer annotations of each image ref.
	annotations map[string][]map[string]string
	// mountedDirs holds the EROFS mounts made in each image work dir.
	mountedDirs map[string][]string
}

func httpURL(s, what string) (*url.URL, error) {
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("invalid %s %q: want an http:// or https:// URL", what, s)
	}
	return u, nil
}

// verifyKernelSupport checks that the kernel has EROFS and /dev/cachefiles.
// The plugin has no FUSE fallback, so it fails without them.
func verifyKernelSupport(procFilesystemsPath, cachefilesDevPath string) error {
	b, err := os.ReadFile(procFilesystemsPath)
	if err != nil {
		return fmt.Errorf("checking for EROFS: %w", err)
	}
	hasEROFS := false
	for line := range strings.SplitSeq(string(b), "\n") {
		if f := strings.Fields(line); len(f) > 0 && f[len(f)-1] == "erofs" {
			hasEROFS = true
			break
		}
	}
	if !hasEROFS {
		return fmt.Errorf("the kernel has no EROFS: erofs is not in %s", procFilesystemsPath)
	}
	if _, err := os.Stat(cachefilesDevPath); err != nil {
		return fmt.Errorf("the kernel has no fscache device: %w", err)
	}
	return nil
}

// newNydusDriver returns a nydusDriver for cfg.
func newNydusDriver(cfg config) (*nydusDriver, error) {
	if cfg.DockerConfigPath == "" {
		cfg.DockerConfigPath = defaultDockerConfigPath
	}
	if cfg.DragonflyProxyAddr == "" {
		cfg.DragonflyProxyAddr = defaultDragonflyProxyAddr
	}
	if cfg.DragonflyHealthURL == "" {
		cfg.DragonflyHealthURL = defaultDragonflyHealthURL
	}
	if cfg.ProcFilesystemsPath == "" {
		cfg.ProcFilesystemsPath = "/proc/filesystems"
	}
	if cfg.CachefilesDevPath == "" {
		cfg.CachefilesDevPath = "/dev/cachefiles"
	}
	if cfg.Mount == nil {
		cfg.Mount = syscall.Mount
	}
	if cfg.Unmount == nil {
		cfg.Unmount = syscall.Unmount
	}
	if cfg.StatFS == nil {
		cfg.StatFS = statFS
	}
	proxyURL, err := httpURL(cfg.DragonflyProxyAddr, "Dragonfly proxy address")
	if err != nil {
		return nil, err
	}
	if _, err := httpURL(cfg.DragonflyHealthURL, "Dragonfly health URL"); err != nil {
		return nil, err
	}
	if err := verifyKernelSupport(cfg.ProcFilesystemsPath, cfg.CachefilesDevPath); err != nil {
		return nil, err
	}

	nd := &nydusDriver{
		dockerSyncer:        newDockerConfigSyncer(cfg.DockerConfigPath),
		dragonflyHealthURL:  cfg.DragonflyHealthURL,
		verifyDragonflyPing: cfg.VerifyDragonflyPing,
		httpClient:          &http.Client{Timeout: plugin.DefaultProbeTimeout},
		blobTransport:       &dragonflyBlobTransport{proxy: proxyURL, base: remote.DefaultTransport},
		layerReader:         cfg.LayerReader,
		layerAnnotations:    cfg.LayerAnnotations,
		mount:               cfg.Mount,
		unmount:             cfg.Unmount,
		statFS:              cfg.StatFS,
		annotations:         make(map[string][]map[string]string),
		mountedDirs:         make(map[string][]string),
	}
	opts := append([]plugin.Option{
		plugin.WithLayerMounter(nd.mountLayers),
		plugin.WithWarmHitHook(nd.onWarmHit),
		plugin.WithWorkDirTeardownHook(nd.unmountWorkDir),
		plugin.WithProbeHook(nd.probeDragonfly),
	}, cfg.DriverOpts...)
	d, err := plugin.NewDriver(providerName, defaultSnapshotterSocket, "nydus", opts...)
	if err != nil {
		return nil, err
	}
	nd.Driver = d
	if nd.layerReader == nil {
		nd.layerReader = nd.registryLayerReader
	}
	if nd.layerAnnotations == nil {
		nd.layerAnnotations = nd.registryLayerAnnotations
	}
	return nd, nil
}

// pingDragonfly checks that the node's dfdaemon is healthy.
func (nd *nydusDriver) pingDragonfly(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, nd.dragonflyHealthURL, nil)
	if err != nil {
		return err
	}
	resp, err := nd.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("checking dfdaemon health: %w", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dfdaemon health check %s returned HTTP %d", nd.dragonflyHealthURL, resp.StatusCode)
	}
	return nil
}

func (nd *nydusDriver) probeDragonfly(ctx context.Context) error {
	if !nd.verifyDragonflyPing {
		return nil
	}
	return nd.pingDragonfly(ctx)
}

// onWarmHit refreshes the registry credentials nydusd reads for a reused
// image.
func (nd *nydusDriver) onWarmHit(ctx context.Context, _ string, req *client.PrepareRequest) {
	nd.dockerSyncer.Sync(ctx, nd.Keychain(), req.ImageRef, req.AuthConfig, false)
}

// mountLayers registers the image's data blob layers with the snapshotter,
// writes and commits its bootstrap layer, and mounts the bootstrap as EROFS.
// The bootstrap holds the whole tree, so the image has one layer directory.
func (nd *nydusDriver) mountLayers(ctx context.Context, env *plugin.MountEnv) (*plugin.MountedLayers, error) {
	if nd.verifyDragonflyPing {
		pingCtx, cancel := context.WithTimeout(ctx, plugin.DefaultProbeTimeout)
		err := nd.pingDragonfly(pingCtx)
		cancel()
		if err != nil {
			return nil, err
		}
	}
	nd.dockerSyncer.Sync(ctx, nd.Keychain(), env.ImageRef, env.AuthConfig, true)

	imgKey := plugin.SanitizePathKey(env.ImageRef)
	runID := fmt.Sprintf("%x", time.Now().UnixNano())
	nsCtx := env.Driver.WithNamespaceContext(ctx)
	res := &plugin.MountedLayers{}
	for i, c := range env.ChainInfos {
		prepKey := fmt.Sprintf("%s-%s-l%d-prep", imgKey, runID, i)
		viewKey := fmt.Sprintf("%s-%s-l%d-view", imgKey, runID, i)
		var layerDigest string
		if i < len(env.LayerDigests) {
			layerDigest = env.LayerDigests[i]
		}
		labels := plugin.StandardCRILayerLabels(env.CanonicalRef, env.ImageDigest, env.LayerDigests, i, c.ChainID)
		extra, err := nd.layerLabels(ctx, env.ImageRef, env.AuthConfig, i, layerDigest)
		if err != nil {
			return res, fmt.Errorf("layer %d: %w", i, err)
		}
		maps.Copy(labels, extra)

		if err := nd.prepareLayer(nsCtx, env, i, layerDigest, c, prepKey, labels, res); err != nil {
			return res, fmt.Errorf("layer %d (%s): %w", i, c.ChainID, err)
		}
		// The data blobs only join the parent chain. The bootstrap is the
		// image's tree.
		if labels[labelNydusBlob] == "true" && labels[labelNydusBootstrap] != "true" {
			continue
		}
		mounts, err := plugin.ViewLayerMounts(nsCtx, env.Client, env.Snapshotter, viewKey, c.ChainID)
		if err != nil {
			return res, fmt.Errorf("layer %d (%s): %w", i, c.ChainID, err)
		}
		res.SnapshotKeys = append(res.SnapshotKeys, viewKey)
		dir, err := nd.mountEROFS(env.ImageWorkDir, len(res.MountDirs), mounts)
		if err != nil {
			return res, fmt.Errorf("layer %d (%s): %w", i, c.ChainID, err)
		}
		res.MountDirs = append(res.MountDirs, dir)
	}
	return res, nil
}

// prepareLayer makes sure the snapshot of layer i exists, under the chain
// lock so images that share the layer prepare it once.
func (nd *nydusDriver) prepareLayer(nsCtx context.Context, env *plugin.MountEnv, i int, layerDigest string, c plugin.ChainInfo, prepKey string, labels map[string]string, res *plugin.MountedLayers) error {
	lk := env.Driver.AcquireChainLock(c.ChainID)
	lk.Lock()
	defer lk.Unlock()
	committed, err := plugin.HasSnapshot(nsCtx, env.Client, env.Snapshotter, c.ChainID)
	if err != nil || committed {
		return err
	}
	prep, err := env.Client.Prepare(nsCtx, &snapshotsapi.PrepareSnapshotRequest{
		Snapshotter: env.Snapshotter,
		Key:         prepKey,
		Parent:      c.ParentChainID,
		Labels:      labels,
	})
	switch status.Code(err) {
	case codes.AlreadyExists:
		// The snapshotter registered the layer itself.
		found, err := plugin.HasSnapshot(nsCtx, env.Client, env.Snapshotter, c.ChainID)
		if err != nil {
			return err
		}
		if !found {
			return errors.New("prepare returned AlreadyExists but the snapshot does not exist")
		}
		return nil
	case codes.OK:
		if err := nd.commitLayer(nsCtx, env, i, layerDigest, c, prepKey, labels, prep.GetMounts()); err != nil {
			plugin.RemoveSnapshot(nsCtx, env.Client, env.Snapshotter, prepKey)
			return err
		}
		res.CommittedKeys = append(res.CommittedKeys, c.ChainID)
		return nil
	default:
		plugin.RemoveSnapshot(nsCtx, env.Client, env.Snapshotter, prepKey)
		return fmt.Errorf("preparing snapshot: %w", err)
	}
}

// commitLayer commits a layer the snapshotter left for the plugin to fill. A
// data blob is committed as is. The bootstrap layer is unpacked into the
// snapshot first. Any other layer means the image is not a Nydus image.
func (nd *nydusDriver) commitLayer(nsCtx context.Context, env *plugin.MountEnv, i int, layerDigest string, c plugin.ChainInfo, prepKey string, labels map[string]string, mounts []*types.Mount) error {
	if labels[labelNydusBootstrap] != "true" && labels[labelNydusBlob] != "true" {
		return fmt.Errorf("the snapshotter cannot serve layer %s: %w", layerDigest, client.ErrNotSupported)
	}
	if labels[labelNydusBootstrap] == "true" {
		dir := extractActiveMountDir(mounts)
		if dir == "" {
			return errors.New("no writable mount for the bootstrap layer")
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		rc, err := nd.layerReader(nsCtx, env.ImageRef, env.AuthConfig, i, layerDigest)
		if err != nil {
			return fmt.Errorf("fetching the bootstrap layer: %w", err)
		}
		defer func() { _ = rc.Close() }()
		if err := unpackLayerTarToDir(nsCtx, rc, dir); err != nil {
			return fmt.Errorf("unpacking the bootstrap layer: %w", err)
		}
	}
	labels["containerd.io/snapshot.ref"] = c.ChainID
	_, err := env.Client.Commit(nsCtx, &snapshotsapi.CommitSnapshotRequest{
		Snapshotter: env.Snapshotter,
		Name:        c.ChainID,
		Key:         prepKey,
		Labels:      labels,
	})
	if err != nil && status.Code(err) != codes.AlreadyExists {
		return fmt.Errorf("committing: %w", err)
	}
	return nil
}

func statFS(path string) (int64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, err
	}
	return int64(st.Type), nil
}

func (nd *nydusDriver) verifyEROFS(dir string) error {
	fsType, err := nd.statFS(dir)
	if err != nil {
		return fmt.Errorf("checking the filesystem of %s: %w", dir, err)
	}
	if fsType != erofsSuperMagic {
		return fmt.Errorf("%s has filesystem type 0x%x, want EROFS (0x%x)", dir, fsType, int64(erofsSuperMagic))
	}
	return nil
}

// mountEROFS mounts the EROFS view the snapshotter returned under
// imageWorkDir, or uses the snapshotter's own mount, and checks that it is
// EROFS.
func (nd *nydusDriver) mountEROFS(imageWorkDir string, index int, mounts []*types.Mount) (string, error) {
	for _, m := range mounts {
		if m.GetType() != "erofs" {
			if strings.HasPrefix(m.GetType(), "fuse") {
				return "", fmt.Errorf("the snapshotter returned a %s mount; only EROFS is supported", m.GetType())
			}
			continue
		}
		dir := filepath.Join(imageWorkDir, fmt.Sprintf("erofs-mnt-%d", index))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		var data []string
		for _, opt := range m.GetOptions() {
			switch opt {
			case "ro", "rw", "nosuid", "nodev":
			default:
				data = append(data, opt)
			}
		}
		source := m.GetSource()
		if source == "" {
			source = "erofs"
		}
		if err := nd.mount(source, dir, "erofs", syscall.MS_RDONLY, strings.Join(data, ",")); err != nil {
			return "", fmt.Errorf("mounting EROFS at %s: %w", dir, err)
		}
		if err := nd.verifyEROFS(dir); err != nil {
			_ = nd.unmount(dir, syscall.MNT_DETACH)
			return "", err
		}
		nd.mu.Lock()
		nd.mountedDirs[imageWorkDir] = append(nd.mountedDirs[imageWorkDir], dir)
		nd.mu.Unlock()
		return dir, nil
	}
	dir := plugin.ExtractMountDir(mounts)
	if dir == "" {
		return "", errors.New("no mount in the snapshotter's View response")
	}
	if err := nd.verifyEROFS(dir); err != nil {
		return "", err
	}
	return dir, nil
}

// unmountWorkDir unmounts the EROFS mounts made in workDir.
func (nd *nydusDriver) unmountWorkDir(workDir string) {
	nd.mu.Lock()
	dirs := nd.mountedDirs[workDir]
	delete(nd.mountedDirs, workDir)
	nd.mu.Unlock()
	for i := len(dirs) - 1; i >= 0; i-- {
		if err := nd.unmount(dirs[i], syscall.MNT_DETACH); err != nil {
			slog.Warn("Failed to unmount EROFS", slog.String("dir", dirs[i]), slog.Any("err", err))
		}
	}
}

// imageAnnotations returns the image's layer annotations. It returns an
// error wrapping client.ErrNotSupported unless the image is Nydus RAFS v6.
func (nd *nydusDriver) imageAnnotations(ctx context.Context, imageRef string, auth *client.AuthConfig) ([]map[string]string, error) {
	nd.mu.Lock()
	cached, ok := nd.annotations[imageRef]
	nd.mu.Unlock()
	if ok {
		return cached, nil
	}
	anns, err := nd.layerAnnotations(ctx, imageRef, auth)
	if err != nil {
		return nil, err
	}
	hasBootstrap := false
	for _, ann := range anns {
		if ann[labelNydusBootstrap] != "true" {
			continue
		}
		if v := ann[labelNydusFSVersion]; v != "" && v != "6" {
			return nil, fmt.Errorf("image %s is RAFS v%s, want v6: %w", imageRef, v, client.ErrNotSupported)
		}
		hasBootstrap = true
	}
	if !hasBootstrap {
		return nil, fmt.Errorf("image %s is not a Nydus RAFS v6 image: %w", imageRef, client.ErrNotSupported)
	}
	nd.mu.Lock()
	nd.annotations[imageRef] = anns
	nd.mu.Unlock()
	return anns, nil
}

// layerLabels returns the snapshot labels containerd-nydus-grpc needs for
// layer i: its annotations, the image ref, the layer digest and the
// registry source.
func (nd *nydusDriver) layerLabels(ctx context.Context, imageRef string, auth *client.AuthConfig, i int, layerDigest string) (map[string]string, error) {
	anns, err := nd.imageAnnotations(ctx, imageRef, auth)
	if err != nil {
		return nil, err
	}
	if i >= len(anns) {
		return nil, fmt.Errorf("the manifest has %d layers, want more than %d", len(anns), i)
	}
	out := maps.Clone(anns[i])
	if out == nil {
		out = make(map[string]string)
	}
	out["containerd.io/snapshot/cri.image-ref"] = imageRef
	out["containerd.io/snapshot/cri.layer-digest"] = layerDigest
	if ref, err := name.ParseReference(imageRef); err == nil {
		out["containerd.io/distribution.source."+ref.Context().RegistryStr()] = ref.Context().RepositoryStr()
	}
	return out, nil
}

// registryLayerAnnotations reads the layer annotations from the image's
// manifest in its registry, and marks Nydus data blob layers.
func (nd *nydusDriver) registryLayerAnnotations(ctx context.Context, imageRef string, auth *client.AuthConfig) ([]map[string]string, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, err
	}
	img, err := remote.Image(ref, plugin.RemoteOptions(ctx, nd.Keychain(), auth)...)
	if err != nil {
		return nil, fmt.Errorf("fetching image %s: %w", imageRef, err)
	}
	m, err := img.Manifest()
	if err != nil {
		return nil, fmt.Errorf("reading the manifest of %s: %w", imageRef, err)
	}
	return manifestAnnotations(m), nil
}

func manifestAnnotations(m *v1.Manifest) []map[string]string {
	out := make([]map[string]string, len(m.Layers))
	for i, l := range m.Layers {
		ann := maps.Clone(l.Annotations)
		if ann == nil {
			ann = make(map[string]string)
		}
		if string(l.MediaType) == nydusBlobMediaType {
			ann[labelNydusBlob] = "true"
		}
		out[i] = ann
	}
	return out
}

// registryLayerReader fetches a layer by digest through the Dragonfly
// proxy, so the nodes share it.
func (nd *nydusDriver) registryLayerReader(ctx context.Context, imageRef string, auth *client.AuthConfig, i int, layerDigest string) (io.ReadCloser, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, err
	}
	if layerDigest == "" {
		return nil, fmt.Errorf("layer %d of %s has no digest", i, imageRef)
	}
	opts := append(plugin.RemoteOptions(ctx, nd.Keychain(), auth), remote.WithTransport(nd.blobTransport))
	layer, err := remote.Layer(ref.Context().Digest(layerDigest), opts...)
	if err != nil {
		return nil, fmt.Errorf("fetching layer %s of %s: %w", layerDigest, imageRef, err)
	}
	return layer.Uncompressed()
}

// dragonflyBlobTransport sends registry blob GETs to the Dragonfly proxy,
// which serves them as a mirror of the registry in X-Dragonfly-Registry.
// Other requests, such as for tokens and manifests, go to the registry.
type dragonflyBlobTransport struct {
	proxy *url.URL
	base  http.RoundTripper
}

func (t *dragonflyBlobTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet || !strings.Contains(req.URL.Path, "/blobs/sha256:") {
		return t.base.RoundTrip(req)
	}
	out := req.Clone(req.Context())
	out.Header.Set("X-Dragonfly-Registry", req.URL.Scheme+"://"+req.URL.Host)
	out.URL.Scheme = t.proxy.Scheme
	out.URL.Host = t.proxy.Host
	out.Host = ""
	return t.base.RoundTrip(out)
}
