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
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"go.ate.dev/substrate/image-plugin/internal/proto/snapshots"
	"go.ate.dev/substrate/image-plugin/pkg/plugin"
	"go.ate.dev/substrate/internal/imagepluginclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// ProviderNydusDragonfly identifies the cross-node Nydus RAFS v6 + Dragonfly
	// v2 P2P (Kernel EROFS + fscache) remote snapshotter provider.
	ProviderNydusDragonfly = "nydus-dragonfly"

	// DefaultNydusSocket is the default UNIX socket path for containerd-nydus-grpc.
	DefaultNydusSocket = "/run/containerd-nydus/containerd-nydus-grpc.sock"

	// DefaultNydusDockerConfigPath is the default Docker config.json path shared
	// with containerd-nydus-grpc for registry authentication.
	DefaultNydusDockerConfigPath = "/run/containerd-nydus/docker/config.json"

	// EnvDragonflyProxyAddr is the environment variable configuring the local
	// Dragonfly dfdaemon HTTP proxy endpoint used by nydusd.
	EnvDragonflyProxyAddr = "DRAGONFLY_PROXY_ADDR"

	// DefaultDragonflyProxyAddr is the default local Dragonfly dfdaemon HTTP
	// proxy address on each worker node.
	DefaultDragonflyProxyAddr = "http://127.0.0.1:4001"

	// EnvDragonflyHealthURL is the environment variable configuring the local
	// Dragonfly dfdaemon health endpoint.
	EnvDragonflyHealthURL = "DRAGONFLY_HEALTH_URL"

	// DefaultDragonflyHealthURL is the default health endpoint of the local
	// Dragonfly dfdaemon.
	DefaultDragonflyHealthURL = "http://127.0.0.1:4003/healthy"

	// EnvDragonflySchedulerAddr is the environment variable configuring the
	// cluster-wide Dragonfly Scheduler gRPC endpoint used for cross-node P2P
	// piece scheduling.
	EnvDragonflySchedulerAddr = "DRAGONFLY_SCHEDULER_ADDR"

	// DefaultDragonflySchedulerAddr is the default Kubernetes DNS address of the
	// Dragonfly Scheduler service in ate-system.
	DefaultDragonflySchedulerAddr = "dragonfly-scheduler.ate-system.svc.cluster.local:8002"

	// EnvDragonflyManagerAddr is the environment variable configuring the
	// cluster-wide Dragonfly Manager gRPC endpoint.
	EnvDragonflyManagerAddr = "DRAGONFLY_MANAGER_ADDR"

	// DefaultDragonflyManagerAddr is the default Kubernetes DNS address of the
	// Dragonfly Manager service in ate-system.
	DefaultDragonflyManagerAddr = "dragonfly-manager.ate-system.svc.cluster.local:65003"

	// DefaultProcFilesystemsPath is the procfs path used to verify that the
	// kernel EROFS filesystem driver is registered.
	DefaultProcFilesystemsPath = "/proc/filesystems"

	// DefaultCachefilesDevPath is the character device path required by Nydus
	// fscache mode to serve EROFS chunks in-kernel.
	DefaultCachefilesDevPath = "/dev/cachefiles"

	// EROFSSuperMagic is the Linux kernel statfs f_type magic number for EROFS (0xE0F5E1E2).
	EROFSSuperMagic = 0xE0F5E1E2

	// FUSESuperMagic is the Linux kernel statfs f_type magic number for FUSE (0x65735546).
	FUSESuperMagic = 0x65735546

	// LabelNydusBootstrap marks the Nydus RAFS v6 EROFS bootstrap layer.
	LabelNydusBootstrap = "containerd.io/snapshot/nydus-bootstrap"

	// LabelNydusBlob marks a Nydus RAFS data-blob layer.
	LabelNydusBlob = "containerd.io/snapshot/nydus-blob"

	// LabelNydusFSVersion records the Nydus RAFS format version ("6" for EROFS).
	LabelNydusFSVersion = "containerd.io/snapshot/nydus-fs-version"
)

// LayerReaderFunc returns an uncompressed tar stream for a specific layer (used
// when unpacking the Nydus RAFS v6 EROFS bootstrap layer).
type LayerReaderFunc func(ctx context.Context, imageRef string, auth *imagepluginclient.AuthConfig, layerIndex int, layerDigest string) (io.ReadCloser, error)

// LayerAnnotationsFunc returns the per-layer OCI manifest annotations for an image.
type LayerAnnotationsFunc func(ctx context.Context, imageRef string, auth *imagepluginclient.AuthConfig) ([]map[string]string, error)

// MountFSFunc mounts a kernel filesystem (such as -t erofs) at targetDir.
type MountFSFunc func(source, targetDir, fstype string, flags uintptr, data string) error

// UnmountFSFunc unmounts targetDir.
type UnmountFSFunc func(targetDir string, flags int) error

// StatFSFunc returns the filesystem type magic (f_type) for path.
type StatFSFunc func(path string) (int64, error)

// NydusDragonflyConfig configures the cross-node Nydus + Dragonfly v2
// (EROFS/fscache) image plugin.
type NydusDragonflyConfig struct {
	DockerConfigPath       string
	DragonflyProxyAddr     string
	DragonflyHealthURL     string
	DragonflySchedulerAddr string
	DragonflyManagerAddr   string
	VerifyDragonflyPing    bool
	HTTPClient             *http.Client
	ProcFilesystemsPath    string
	CachefilesDevPath      string
	SkipKernelCheck        bool
	LayerReader            LayerReaderFunc
	LayerAnnotations       LayerAnnotationsFunc
	MountFS                MountFSFunc
	UnmountFS              UnmountFSFunc
	StatFS                 StatFSFunc
	// WarmListDir holds per-image lists of files to read after mounting, as
	// <manifest digest hex>.txt. Empty disables the warm-up.
	WarmListDir string
	WarmWorkers int
	WarmTimeout time.Duration
	// BlobSeedURL is the base URL of the blob seed servers. When set,
	// manifests and bootstrap layers are read from there first.
	BlobSeedURL string
	// ReadAheadKB, when positive, is written to read_ahead_kb of every EROFS
	// bdi after a mount resolves. Each EROFS mount has its own erofs-N bdi.
	ReadAheadKB int
	// Layered, when set, mounts the parent chain and the top layer as two
	// EROFS mounts bound with nydusd directly. Images whose parent chain
	// bootstrap is not on the seeds go through the snapshotter.
	Layered    *LayeredConfig
	ServerOpts []plugin.Option
}

// NydusDragonflyDriver wraps the shared plugin.Driver with strict Kernel EROFS
// + fscache enforcement (no fusedev fallback), cross-node Dragonfly v2 P2P
// proxy/scheduler/manager configuration and health probing, Nydus RAFS v6
// manifest label propagation, and EROFS mount handling.
type NydusDragonflyDriver struct {
	*plugin.Driver
	dockerSyncer           *plugin.DockerConfigSyncer
	dragonflyProxyAddr     string
	dragonflyHealthURL     string
	dragonflySchedulerAddr string
	dragonflyManagerAddr   string
	verifyDragonflyPing    bool
	httpClient             *http.Client
	blobTransport          http.RoundTripper
	procFilesystemsPath    string
	cachefilesDevPath      string
	skipKernelCheck        bool
	layerReader            LayerReaderFunc
	layerAnnotations       LayerAnnotationsFunc
	mountFS                MountFSFunc
	unmountFS              UnmountFSFunc
	statFS                 StatFSFunc
	warmListDir            string
	warmWorkers            int
	warmTimeout            time.Duration
	blobSeedURL            string
	seedClient             *http.Client
	readAheadKB            int
	layered                *layeredMounter

	mu                sync.Mutex
	cachedAnnotations map[string][]map[string]string
	mountedDirs       map[string][]string
}

// ResolveDragonflyProxyAddr returns the local Dragonfly dfdaemon HTTP proxy
// endpoint from the explicit flag value, the DRAGONFLY_PROXY_ADDR environment
// variable, or DefaultDragonflyProxyAddr, validating that it is a well-formed
// HTTP(S) URL.
func ResolveDragonflyProxyAddr(flagVal string) (string, error) {
	return resolveDragonflyURL(flagVal, EnvDragonflyProxyAddr, DefaultDragonflyProxyAddr, "P2P proxy")
}

// ResolveDragonflyHealthURL returns the local Dragonfly dfdaemon health
// endpoint from val, the DRAGONFLY_HEALTH_URL environment variable, or
// DefaultDragonflyHealthURL.
func ResolveDragonflyHealthURL(val string) (string, error) {
	return resolveDragonflyURL(val, EnvDragonflyHealthURL, DefaultDragonflyHealthURL, "health")
}

func resolveDragonflyURL(val, envVar, defaultVal, label string) (string, error) {
	addr := strings.TrimSpace(val)
	if addr == "" {
		addr = strings.TrimSpace(os.Getenv(envVar))
	}
	if addr == "" {
		addr = defaultVal
	}
	u, err := url.Parse(addr)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("invalid Dragonfly %s endpoint %q (expected http:// or https:// URL)", label, addr)
	}
	return strings.TrimRight(addr, "/"), nil
}

// ResolveDragonflyHostPort resolves a host:port endpoint from flagVal, envVar,
// or defaultVal and validates that it has both a non-empty host and port.
func ResolveDragonflyHostPort(flagVal, envVar, defaultVal, label string) (string, error) {
	addr := strings.TrimSpace(flagVal)
	if addr == "" && envVar != "" {
		addr = strings.TrimSpace(os.Getenv(envVar))
	}
	if addr == "" {
		addr = defaultVal
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil || host == "" || port == "" {
		return "", fmt.Errorf("invalid Dragonfly %s endpoint %q (expected host:port)", label, addr)
	}
	return addr, nil
}

// VerifyKernelEROFSSupport checks that the Linux kernel has the EROFS
// filesystem registered in /proc/filesystems and that /dev/cachefiles exists.
// It returns an explicit error if either is missing instead of falling back to
// userspace FUSE (fusedev).
func VerifyKernelEROFSSupport(procFilesystemsPath, cachefilesDevPath string) error {
	if procFilesystemsPath == "" {
		procFilesystemsPath = DefaultProcFilesystemsPath
	}
	if cachefilesDevPath == "" {
		cachefilesDevPath = DefaultCachefilesDevPath
	}
	b, err := os.ReadFile(procFilesystemsPath)
	if err != nil {
		return fmt.Errorf("kernel EROFS + fscache is required for image-plugin-dragonfly: reading %s failed (fusedev fallback is disabled): %w", procFilesystemsPath, err)
	}
	hasEROFS := false
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[len(fields)-1] == "erofs" {
			hasEROFS = true
			break
		}
	}
	if !hasEROFS {
		return fmt.Errorf("kernel EROFS + fscache is required for image-plugin-dragonfly: 'erofs' filesystem not registered in %s (GKE COS disables CONFIG_EROFS_FS; run on an EROFS-enabled kernel with modprobe erofs; fusedev fallback is disabled)", procFilesystemsPath)
	}
	if _, err := os.Stat(cachefilesDevPath); err != nil {
		return fmt.Errorf("kernel EROFS + fscache is required for image-plugin-dragonfly: %s is not available (run in a privileged container with modprobe cachefiles; fusedev fallback is disabled): %w", cachefilesDevPath, err)
	}
	return nil
}

// NewNydusDragonfly creates a cross-node Nydus + Dragonfly v2 ImagePlugin
// driver with strict Kernel EROFS + fscache verification and Dragonfly dfdaemon
// health probing.
func NewNydusDragonfly(cfg NydusDragonflyConfig) (*NydusDragonflyDriver, error) {
	dfAddr, err := ResolveDragonflyProxyAddr(cfg.DragonflyProxyAddr)
	if err != nil {
		return nil, err
	}
	healthURL, err := ResolveDragonflyHealthURL(cfg.DragonflyHealthURL)
	if err != nil {
		return nil, err
	}
	proxyURL, err := url.Parse(dfAddr)
	if err != nil {
		return nil, err
	}
	schedAddr, err := ResolveDragonflyHostPort(cfg.DragonflySchedulerAddr, EnvDragonflySchedulerAddr, DefaultDragonflySchedulerAddr, "scheduler")
	if err != nil {
		return nil, err
	}
	mgrAddr, err := ResolveDragonflyHostPort(cfg.DragonflyManagerAddr, EnvDragonflyManagerAddr, DefaultDragonflyManagerAddr, "manager")
	if err != nil {
		return nil, err
	}
	procPath := cfg.ProcFilesystemsPath
	if procPath == "" {
		procPath = DefaultProcFilesystemsPath
	}
	cachefilesPath := cfg.CachefilesDevPath
	if cachefilesPath == "" {
		cachefilesPath = DefaultCachefilesDevPath
	}
	if !cfg.SkipKernelCheck {
		if err := VerifyKernelEROFSSupport(procPath, cachefilesPath); err != nil {
			return nil, err
		}
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: plugin.DefaultProbeTimeout,
		}
	}

	nd := &NydusDragonflyDriver{
		dockerSyncer:           plugin.NewDockerConfigSyncer(cfg.DockerConfigPath),
		dragonflyProxyAddr:     dfAddr,
		dragonflyHealthURL:     healthURL,
		dragonflySchedulerAddr: schedAddr,
		dragonflyManagerAddr:   mgrAddr,
		verifyDragonflyPing:    cfg.VerifyDragonflyPing,
		httpClient:             httpClient,
		blobTransport:          &dragonflyBlobTransport{proxy: proxyURL, base: remote.DefaultTransport},
		procFilesystemsPath:    procPath,
		cachefilesDevPath:      cachefilesPath,
		skipKernelCheck:        cfg.SkipKernelCheck,
		layerReader:            cfg.LayerReader,
		layerAnnotations:       cfg.LayerAnnotations,
		mountFS:                cfg.MountFS,
		unmountFS:              cfg.UnmountFS,
		statFS:                 cfg.StatFS,
		warmListDir:            cfg.WarmListDir,
		warmWorkers:            cfg.WarmWorkers,
		warmTimeout:            cfg.WarmTimeout,
		blobSeedURL:            strings.TrimRight(cfg.BlobSeedURL, "/"),
		seedClient:             &http.Client{Timeout: 30 * time.Second},
		readAheadKB:            cfg.ReadAheadKB,
		cachedAnnotations:      make(map[string][]map[string]string),
		mountedDirs:            make(map[string][]string),
	}
	if nd.mountFS == nil {
		nd.mountFS = syscall.Mount
	}
	if nd.unmountFS == nil {
		nd.unmountFS = syscall.Unmount
	}
	if nd.statFS == nil {
		nd.statFS = defaultStatFS
	}

	opts := append([]plugin.Option{
		plugin.WithLayerMounter(nd.mountLayers),
		plugin.WithWarmHitHook(nd.onWarmHit),
		plugin.WithWorkDirTeardownHook(nd.unmountWorkDir),
		plugin.WithProbeHook(nd.probeDragonfly),
	}, cfg.ServerOpts...)
	if nd.blobSeedURL != "" {
		opts = append(opts, plugin.WithImageResolver(nd.resolveImage))
	}

	base, err := plugin.NewDriver(ProviderNydusDragonfly, DefaultNydusSocket, "nydus", opts...)
	if err != nil {
		return nil, err
	}
	nd.Driver = base
	if nd.layerReader == nil {
		nd.layerReader = nd.defaultLayerReader
	}
	if nd.layerAnnotations == nil {
		nd.layerAnnotations = nd.defaultLayerAnnotations
	}
	if cfg.Layered != nil {
		if nd.blobSeedURL == "" {
			return nil, errors.New("layered mounts read bootstraps from the blob seeds; set the blob seed URL")
		}
		nd.layered = newLayeredMounter(*cfg.Layered, nd.blobSeedURL, base.Keychain(), nd.mountFS, nd.unmountFS, nd.verifyMountIsEROFS)
	}
	return nd, nil
}

// DragonflyProxyAddr returns the configured local Dragonfly dfdaemon HTTP proxy endpoint.
func (nd *NydusDragonflyDriver) DragonflyProxyAddr() string {
	return nd.dragonflyProxyAddr
}

// DragonflySchedulerAddr returns the configured cluster-wide Dragonfly Scheduler endpoint.
func (nd *NydusDragonflyDriver) DragonflySchedulerAddr() string {
	return nd.dragonflySchedulerAddr
}

// DragonflyManagerAddr returns the configured cluster-wide Dragonfly Manager endpoint.
func (nd *NydusDragonflyDriver) DragonflyManagerAddr() string {
	return nd.dragonflyManagerAddr
}

// PingDragonflyProxy probes the local dfdaemon health endpoint to ensure the
// Dragonfly P2P proxy is up before Nydus attempts to stream chunks.
func (nd *NydusDragonflyDriver) PingDragonflyProxy(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, nd.dragonflyHealthURL, nil)
	if err != nil {
		return fmt.Errorf("creating Dragonfly ping request %q: %w", nd.dragonflyHealthURL, err)
	}
	resp, err := nd.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("probing Dragonfly dfdaemon at %q: %w", nd.dragonflyHealthURL, err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dragonfly dfdaemon ping %q returned HTTP %d", nd.dragonflyHealthURL, resp.StatusCode)
	}
	return nil
}

func (nd *NydusDragonflyDriver) probeDragonfly(ctx context.Context) error {
	if !nd.verifyDragonflyPing {
		return nil
	}
	if err := nd.PingDragonflyProxy(ctx); err != nil {
		slog.WarnContext(ctx, "Dragonfly dfdaemon proxy ping failed during Probe",
			slog.String("proxyAddr", nd.dragonflyProxyAddr),
			slog.String("schedulerAddr", nd.dragonflySchedulerAddr),
			slog.Any("err", err))
		return err
	}
	return nil
}

func (nd *NydusDragonflyDriver) onWarmHit(ctx context.Context, _ string, req *imagepluginclient.PrepareRequest) {
	if req == nil {
		return
	}
	nd.dockerSyncer.Sync(ctx, nd.Driver.Keychain(), req.ImageRef, req.AuthConfig, false)
}

func (nd *NydusDragonflyDriver) mountLayers(ctx context.Context, env *plugin.MountEnv) (*plugin.MountedLayers, error) {
	var pingDur, syncDur, labelsDur, lockDur, materializeDur, mountDur, warmDur time.Duration
	start := time.Now()
	defer func() {
		slog.InfoContext(ctx, "Nydus mountLayers timing",
			slog.String("image", env.ImageRef),
			slog.Float64("ping_ms", ms(pingDur)),
			slog.Float64("sync_ms", ms(syncDur)),
			slog.Float64("labels_ms", ms(labelsDur)),
			slog.Float64("lock_wait_ms", ms(lockDur)),
			slog.Float64("materialize_ms", ms(materializeDur)),
			slog.Float64("erofs_mount_ms", ms(mountDur)),
			slog.Float64("warm_ms", ms(warmDur)),
			slog.Float64("total_ms", ms(time.Since(start))))
	}()
	if nd.verifyDragonflyPing {
		t := time.Now()
		pingCtx, cancel := context.WithTimeout(ctx, plugin.DefaultProbeTimeout)
		err := nd.PingDragonflyProxy(pingCtx)
		cancel()
		pingDur = time.Since(t)
		if err != nil {
			return nil, fmt.Errorf("dragonfly P2P proxy unavailable (scheduler=%s): %w", nd.dragonflySchedulerAddr, err)
		}
	}

	t := time.Now()
	nd.dockerSyncer.Sync(ctx, nd.Driver.Keychain(), env.ImageRef, env.AuthConfig, true)
	syncDur = time.Since(t)

	if nd.layered != nil {
		anns, err := nd.getLayerAnnotations(ctx, env.ImageRef, env.AuthConfig)
		if err != nil {
			return nil, err
		}
		data, err := layeredDataDigests(env.LayerDigests, anns)
		if err != nil {
			return nil, err
		}
		dirs, err := nd.layered.mount(ctx, env.ImageRef, env.ImageWorkDir, data)
		if err != nil {
			return nil, fmt.Errorf("layered mount: %w", err)
		}
		return &plugin.MountedLayers{MountDirs: dirs}, nil
	}

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
		t := time.Now()
		extraLabels, err := nd.layerLabelsForIndex(ctx, env.ImageRef, env.AuthConfig, i, layerDigest, c)
		labelsDur += time.Since(t)
		if err != nil {
			return res, fmt.Errorf("layer %d (%s) labels: %w", i, c.ChainID, err)
		}
		for k, v := range extraLabels {
			labels[k] = v
		}

		lk := env.Driver.AcquireChainLock(c.ChainID)
		t = time.Now()
		lk.Lock()
		lockDur += time.Since(t)
		statStart := time.Now()
		committed, err := plugin.HasSnapshot(nsCtx, env.Client, env.Snapshotter, c.ChainID)
		res.StatDur += time.Since(statStart)
		if err == nil && !committed {
			prepStart := time.Now()
			prepResp, prepErr := env.Client.Prepare(nsCtx, &snapshots.PrepareSnapshotRequest{
				Snapshotter: env.Snapshotter,
				Key:         prepKey,
				Parent:      c.ParentChainID,
				Labels:      labels,
			})
			res.PrepareDur += time.Since(prepStart)
			switch status.Code(prepErr) {
			case codes.AlreadyExists:
				confirmStart := time.Now()
				found, sErr := plugin.HasSnapshot(nsCtx, env.Client, env.Snapshotter, c.ChainID)
				res.StatDur += time.Since(confirmStart)
				if sErr != nil {
					err = sErr
				} else if !found {
					err = errors.New("prepare returned AlreadyExists but the chain ID snapshot does not exist")
				}
			case codes.OK:
				t := time.Now()
				matErr := nd.materializeNydusLayer(ctx, env.Client, env.Snapshotter, env.ImageRef, env.AuthConfig, i, layerDigest, c, prepKey, labels, prepResp.GetMounts())
				materializeDur += time.Since(t)
				if matErr != nil {
					plugin.RemoveSnapshot(nsCtx, env.Client, env.Snapshotter, prepKey)
					err = matErr
				} else {
					res.CommittedKeys = append(res.CommittedKeys, c.ChainID)
				}
			default:
				plugin.RemoveSnapshot(nsCtx, env.Client, env.Snapshotter, prepKey)
				err = fmt.Errorf("preparing snapshot: %w", prepErr)
			}
		}
		lk.Unlock()
		if err != nil {
			return res, fmt.Errorf("layer %d (%s): %w", i, c.ChainID, err)
		}

		// Nydus data-blob layers (0 .. N-2) are registered in the snapshotter
		// parent chain; only the final RAFS v6 EROFS bootstrap layer is viewed
		// and mounted as the unified EROFS image tree.
		if labels[LabelNydusBlob] == "true" && labels[LabelNydusBootstrap] != "true" {
			continue
		}

		viewStart := time.Now()
		mounts, err := plugin.ViewLayerMounts(nsCtx, env.Client, env.Snapshotter, viewKey, c.ChainID)
		res.ViewDur += time.Since(viewStart)
		if err != nil {
			return res, fmt.Errorf("layer %d (%s): %w", i, c.ChainID, err)
		}
		res.SnapshotKeys = append(res.SnapshotKeys, viewKey)

		t = time.Now()
		mountDir, err := nd.resolveEROFSMount(ctx, env.ImageWorkDir, len(res.MountDirs), mounts)
		mountDur += time.Since(t)
		if err != nil {
			return res, fmt.Errorf("resolving mount for layer %d (%s): %w", i, c.ChainID, err)
		}
		res.MountDirs = append(res.MountDirs, mountDir)
	}

	if nd.warmListDir != "" && len(res.MountDirs) > 0 {
		t := time.Now()
		nd.warmRootfs(ctx, env.ImageRef, env.ImageDigest, res.MountDirs[len(res.MountDirs)-1])
		warmDur = time.Since(t)
	}

	return res, nil
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

// warmRootfs reads the files listed for the image before returning, then the
// files after the list's background marker in a goroutine. A slow or failed
// warm-up leaves the rest to on-demand reads.
func (nd *NydusDragonflyDriver) warmRootfs(ctx context.Context, imageRef, imageDigest, mountDir string) {
	fg, bg, err := readWarmList(nd.warmListDir, imageDigest)
	if err != nil {
		slog.ErrorContext(ctx, "Reading Nydus warm list", slog.String("image", imageRef), slog.Any("err", err))
		return
	}
	log := func(ctx context.Context, part string, st warmStats, err error) {
		slog.InfoContext(ctx, "Warmed Nydus rootfs", slog.String("image", imageRef), slog.String("part", part),
			slog.Int("listed", st.Listed), slog.Int("read", st.Read), slog.Int("failed", st.Failed),
			slog.Int64("bytes", st.Bytes), slog.Duration("duration", st.Dur), slog.Any("err", err))
	}
	if len(fg) > 0 {
		fgCtx, cancel := context.WithTimeout(ctx, nd.warmTimeout)
		st, err := readPaths(fgCtx, mountDir, fg, nd.warmWorkers)
		cancel()
		log(ctx, "foreground", st, err)
	}
	if len(bg) > 0 {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), nd.warmTimeout)
			defer cancel()
			st, err := readPaths(bgCtx, mountDir, bg, warmBackgroundWorkers)
			log(bgCtx, "background", st, err)
		}()
	}
}

func defaultStatFS(path string) (int64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, err
	}
	return int64(st.Type), nil
}

func (nd *NydusDragonflyDriver) verifyMountIsEROFS(mountDir string) error {
	if nd.skipKernelCheck && nd.statFS == nil {
		return nil
	}
	fsType, err := nd.statFS(mountDir)
	if err != nil {
		return fmt.Errorf("statfs on Nydus mount %q failed: %w", mountDir, err)
	}
	if fsType == FUSESuperMagic {
		return fmt.Errorf("nydus mount %q has FUSE filesystem type (0x%x) instead of Kernel EROFS (0x%x); fusedev fallback is disabled", mountDir, fsType, int64(EROFSSuperMagic))
	}
	if fsType != EROFSSuperMagic {
		return fmt.Errorf("nydus mount %q has unexpected filesystem type 0x%x; Kernel EROFS (0x%x) is strictly required", mountDir, fsType, int64(EROFSSuperMagic))
	}
	return nil
}

func (nd *NydusDragonflyDriver) resolveEROFSMount(_ context.Context, imageWorkDir string, viewIndex int, mounts []*snapshots.Mount) (string, error) {
	for _, m := range mounts {
		if m == nil {
			continue
		}
		if m.Type == "erofs" || m.Type == "fuse.nydus-overlayfs" {
			if m.Type != "erofs" {
				return "", fmt.Errorf("snapshotter returned non-EROFS mount type %q (options=%v); Kernel EROFS (fscache) is strictly required and fusedev fallback is disabled", m.Type, m.Options)
			}
			targetDir := filepath.Join(imageWorkDir, fmt.Sprintf("erofs-mnt-%d", viewIndex))
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return "", fmt.Errorf("creating EROFS mount point %s: %w", targetDir, err)
			}
			var dataOpts []string
			for _, opt := range m.Options {
				if opt == "ro" || opt == "rw" || opt == "nosuid" || opt == "nodev" {
					continue
				}
				dataOpts = append(dataOpts, opt)
			}
			source := m.Source
			if source == "" {
				source = "erofs"
			}
			if err := nd.mountFS(source, targetDir, "erofs", syscall.MS_RDONLY, strings.Join(dataOpts, ",")); err != nil {
				return "", fmt.Errorf("mounting kernel EROFS (%s, opts=%q) at %s: %w", source, strings.Join(dataOpts, ","), targetDir, err)
			}
			if err := nd.verifyMountIsEROFS(targetDir); err != nil {
				_ = nd.unmountFS(targetDir, syscall.MNT_DETACH)
				return "", err
			}
			nd.mu.Lock()
			nd.mountedDirs[imageWorkDir] = append(nd.mountedDirs[imageWorkDir], targetDir)
			nd.mu.Unlock()
			if err := nd.setEROFSReadAhead(); err != nil {
				return "", err
			}
			return targetDir, nil
		}
	}

	mountDir := plugin.ExtractMountDir(mounts)
	if mountDir == "" {
		return "", errors.New("no mount directory found in Nydus View response")
	}
	if err := nd.verifyMountIsEROFS(mountDir); err != nil {
		return "", err
	}
	if err := nd.setEROFSReadAhead(); err != nil {
		return "", err
	}
	return mountDir, nil
}

// setEROFSReadAhead writes nd.readAheadKB to every erofs-N bdi that differs.
// A bdi can vanish when another image unmounts, so ENOENT is ignored.
func (nd *NydusDragonflyDriver) setEROFSReadAhead() error {
	if nd.readAheadKB <= 0 {
		return nil
	}
	paths, err := filepath.Glob("/sys/class/bdi/erofs-*/read_ahead_kb")
	if err != nil {
		return err
	}
	want := strconv.Itoa(nd.readAheadKB)
	for _, p := range paths {
		cur, err := os.ReadFile(p)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}
		if strings.TrimSpace(string(cur)) == want {
			continue
		}
		if err := os.WriteFile(p, []byte(want), 0o644); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("setting %s: %w", p, err)
		}
	}
	return nil
}

func (nd *NydusDragonflyDriver) unmountWorkDir(workDir string) {
	nd.mu.Lock()
	dirs := nd.mountedDirs[workDir]
	delete(nd.mountedDirs, workDir)
	nd.mu.Unlock()
	for i := len(dirs) - 1; i >= 0; i-- {
		_ = nd.unmountFS(dirs[i], syscall.MNT_DETACH)
	}
	if nd.layered != nil {
		nd.layered.release(workDir)
	}
}

func (nd *NydusDragonflyDriver) getLayerAnnotations(ctx context.Context, imageRef string, auth *imagepluginclient.AuthConfig) ([]map[string]string, error) {
	nd.mu.Lock()
	if cached, ok := nd.cachedAnnotations[imageRef]; ok {
		nd.mu.Unlock()
		return cached, nil
	}
	nd.mu.Unlock()

	layerAnns, err := nd.layerAnnotations(ctx, imageRef, auth)
	if err != nil {
		return nil, err
	}
	hasBootstrap := false
	for _, ann := range layerAnns {
		if ann[LabelNydusBootstrap] == "true" {
			if v := ann[LabelNydusFSVersion]; v != "" && v != "6" {
				return nil, fmt.Errorf("nydus image %q uses RAFS version %q, but RAFS v6 (EROFS) is required: %w", imageRef, v, imagepluginclient.ErrNotSupported)
			}
			hasBootstrap = true
		}
	}
	if !hasBootstrap {
		return nil, fmt.Errorf("image %q is not in Nydus RAFS v6 (EROFS) format (missing %s layer; pre-convert with 'nydusify convert --fs-version 6'): %w",
			imageRef, LabelNydusBootstrap, imagepluginclient.ErrNotSupported)
	}

	nd.mu.Lock()
	nd.cachedAnnotations[imageRef] = layerAnns
	nd.mu.Unlock()
	return layerAnns, nil
}

func (nd *NydusDragonflyDriver) layerLabelsForIndex(ctx context.Context, imageRef string, auth *imagepluginclient.AuthConfig, layerIndex int, layerDigest string, _ plugin.ChainInfo) (map[string]string, error) {
	layerAnns, err := nd.getLayerAnnotations(ctx, imageRef, auth)
	if err != nil {
		return nil, err
	}
	if layerIndex < 0 || layerIndex >= len(layerAnns) {
		return nil, fmt.Errorf("layer index %d out of range (len=%d)", layerIndex, len(layerAnns))
	}
	out := make(map[string]string, len(layerAnns[layerIndex])+3)
	for k, v := range layerAnns[layerIndex] {
		out[k] = v
	}
	out["containerd.io/snapshot/cri.image-ref"] = imageRef
	out["containerd.io/snapshot/cri.layer-digest"] = layerDigest
	if ref, err := name.ParseReference(imageRef); err == nil {
		reg := ref.Context().RegistryStr()
		repo := ref.Context().RepositoryStr()
		if reg != "" && repo != "" {
			out["containerd.io/distribution.source."+reg] = repo
		}
	}
	return out, nil
}

func (nd *NydusDragonflyDriver) materializeNydusLayer(ctx context.Context, client snapshots.SnapshotsClient, snapshotter, imageRef string, auth *imagepluginclient.AuthConfig, layerIndex int, layerDigest string, c plugin.ChainInfo, prepKey string, labels map[string]string, mounts []*snapshots.Mount) error {
	if labels[LabelNydusBootstrap] != "true" {
		if labels[LabelNydusBlob] == "true" {
			labels["containerd.io/snapshot.ref"] = c.ChainID
			_, err := client.Commit(nd.Driver.WithNamespaceContext(ctx), &snapshots.CommitSnapshotRequest{
				Snapshotter: snapshotter,
				Name:        c.ChainID,
				Key:         prepKey,
				Labels:      labels,
			})
			if err != nil && status.Code(err) != codes.AlreadyExists {
				return fmt.Errorf("committing nydus blob layer %d (%s): %w", layerIndex, c.ChainID, err)
			}
			return nil
		}
		return fmt.Errorf("nydus snapshotter declined non-bootstrap layer %d (%s): %w", layerIndex, layerDigest, imagepluginclient.ErrNotSupported)
	}

	targetDir := plugin.ExtractActiveMountDir(mounts)
	if targetDir == "" {
		return fmt.Errorf("nydus bootstrap layer %d (%s): no writable mount directory in Prepare response", layerIndex, c.ChainID)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating nydus bootstrap mount directory %s: %w", targetDir, err)
	}

	tFetch := time.Now()
	rc, err := nd.layerReader(ctx, imageRef, auth, layerIndex, layerDigest)
	if err != nil {
		return fmt.Errorf("fetching nydus EROFS bootstrap layer %d (%s): %w", layerIndex, layerDigest, err)
	}
	defer func() { _ = rc.Close() }()
	if err := plugin.UnpackLayerTarToDir(ctx, rc, targetDir); err != nil {
		return fmt.Errorf("unpacking nydus EROFS bootstrap layer %d (%s) into %s: %w", layerIndex, layerDigest, targetDir, err)
	}
	fetchDur := time.Since(tFetch)

	tCommit := time.Now()
	labels["containerd.io/snapshot.ref"] = c.ChainID
	_, err = client.Commit(nd.Driver.WithNamespaceContext(ctx), &snapshots.CommitSnapshotRequest{
		Snapshotter: snapshotter,
		Name:        c.ChainID,
		Key:         prepKey,
		Labels:      labels,
	})
	if err != nil && status.Code(err) != codes.AlreadyExists {
		return fmt.Errorf("committing nydus EROFS bootstrap layer %d (%s): %w", layerIndex, c.ChainID, err)
	}

	slog.InfoContext(ctx, "Materialized and committed Nydus RAFS v6 EROFS bootstrap layer",
		slog.Int("layerIndex", layerIndex),
		slog.String("layerDigest", layerDigest),
		slog.String("chainID", c.ChainID),
		slog.Float64("fetch_ms", ms(fetchDur)),
		slog.Float64("commit_ms", ms(time.Since(tCommit))))
	return nil
}

func (nd *NydusDragonflyDriver) defaultLayerAnnotations(ctx context.Context, imageRef string, authConfig *imagepluginclient.AuthConfig) ([]map[string]string, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, fmt.Errorf("parsing image reference %q: %w", imageRef, err)
	}
	var manifest *v1.Manifest
	if nd.blobSeedURL != "" {
		if manifest, err = nd.seedManifest(ctx, ref); err != nil {
			logSeedMiss(ctx, "manifest", imageRef, err)
		}
	}
	if manifest == nil {
		img, err := remote.Image(ref, plugin.RemoteOptions(ctx, nd.Driver.Keychain(), authConfig)...)
		if err != nil {
			return nil, fmt.Errorf("remote.Image(%q): %w", imageRef, err)
		}
		if manifest, err = img.Manifest(); err != nil {
			return nil, fmt.Errorf("reading manifest for %q: %w", imageRef, err)
		}
	}
	res := make([]map[string]string, len(manifest.Layers))
	for i, l := range manifest.Layers {
		ann := make(map[string]string, len(l.Annotations)+1)
		for k, v := range l.Annotations {
			ann[k] = v
		}
		if string(l.MediaType) == "application/vnd.oci.image.layer.nydus.blob.v1" {
			ann[LabelNydusBlob] = "true"
		}
		res[i] = ann
	}
	return res, nil
}

// defaultLayerReader fetches the layer blob by digest from the blob seeds if
// set, else through the Dragonfly proxy, so peers share it and the manifest
// is not fetched again.
func (nd *NydusDragonflyDriver) defaultLayerReader(ctx context.Context, imageRef string, authConfig *imagepluginclient.AuthConfig, layerIndex int, layerDigest string) (io.ReadCloser, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, fmt.Errorf("parsing image reference %q: %w", imageRef, err)
	}
	if layerDigest == "" {
		return nil, fmt.Errorf("layer %d of %q has no digest", layerIndex, imageRef)
	}
	if nd.blobSeedURL != "" {
		rc, err := nd.seedLayer(ctx, ref.Context(), layerDigest)
		if err == nil {
			return rc, nil
		}
		logSeedMiss(ctx, "layer "+layerDigest, imageRef, err)
	}
	opts := append(plugin.RemoteOptions(ctx, nd.Driver.Keychain(), authConfig), remote.WithTransport(nd.blobTransport))
	layer, err := remote.Layer(ref.Context().Digest(layerDigest), opts...)
	if err != nil {
		return nil, fmt.Errorf("remote.Layer(%q, %s): %w", imageRef, layerDigest, err)
	}
	return layer.Uncompressed()
}

// dragonflyBlobTransport sends registry blob GETs to the Dragonfly proxy,
// which serves them as a registry mirror of the host in X-Dragonfly-Registry.
// Other requests, such as token and manifest requests, go to the registry.
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
