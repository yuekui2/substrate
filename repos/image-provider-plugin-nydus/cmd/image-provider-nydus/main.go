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

// Command image-provider-nydus is an image provider plugin that serves Nydus
// RAFS v6 images through containerd-nydus-grpc in EROFS over fscache mode,
// with nydusd fetching data through the node's Dragonfly dfdaemon.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ai-on-gke/image-provider-plugin/plugin"
)

func main() {
	var (
		listenSocket        = flag.String("listen-socket", plugin.DefaultListenSocket, "Unix socket to serve the plugin on")
		snapshotterSocket   = flag.String("snapshotter-socket", defaultSnapshotterSocket, "Unix socket of containerd-nydus-grpc")
		dockerConfigPath    = flag.String("docker-config", defaultDockerConfigPath, "Docker config.json that containerd-nydus-grpc reads registry credentials from")
		dragonflyProxyAddr  = flag.String("dragonfly-proxy-addr", defaultDragonflyProxyAddr, "Registry mirror URL of the node's dfdaemon")
		dragonflyHealthURL  = flag.String("dragonfly-health-url", defaultDragonflyHealthURL, "Health URL of the node's dfdaemon")
		verifyDragonflyPing = flag.Bool("verify-dragonfly-ping", true, "Require a healthy dfdaemon to serve images")
		procFilesystems     = flag.String("proc-filesystems", "/proc/filesystems", "File listing the kernel's filesystems, checked for EROFS")
		cachefilesDev       = flag.String("cachefiles-dev", "/dev/cachefiles", "fscache device that EROFS over fscache needs")
		blobSeedURL         = flag.String("blob-seed-url", "", "Base URL of the blob seed servers to read manifests and bootstrap layers from before the registry; empty disables")
		workDir             = flag.String("work-dir", "", "Work directory for image layer links (default "+plugin.DefaultBaseWorkDir+"/"+providerName+")")
		namespace           = flag.String("namespace", plugin.DefaultNamespace, "containerd namespace sent to containerd-nydus-grpc")
		snapshotterName     = flag.String("snapshotter-name", "nydus", "Snapshotter name sent in snapshot requests")
		idleTTL             = flag.Duration("idle-ttl", plugin.DefaultLeaseIdleTTL, "How long to keep an unused image mounted")
		declineTTL          = flag.Duration("decline-ttl", plugin.DefaultDeclineTTL, "How long to remember an image the plugin cannot serve")
		listableTimeout     = flag.Duration("listable-timeout", plugin.DefaultListableTimeout, "How long to wait for a mounted layer to become listable")
		startupTimeout      = flag.Duration("startup-timeout", plugin.DefaultStartupTimeout, "How long to wait at startup for containerd-nydus-grpc and dfdaemon")
		drainTimeout        = flag.Duration("drain-timeout", plugin.DefaultDrainTimeout, "How long to wait after SIGTERM for image leases to end")
		drainCompleteFile   = flag.String("drain-complete-file", plugin.DefaultDrainCompleteFile, "File to write once draining finishes")
		probe               = flag.Bool("probe", false, "Check that the plugin on --listen-socket is ready and exit")
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if *probe {
		if err := plugin.ProbeUnixSocket(ctx, *listenSocket); err != nil {
			slog.Error("Plugin is not ready", slog.String("socket", *listenSocket), slog.Any("err", err))
			os.Exit(1)
		}
		return
	}

	opts := []plugin.Option{
		plugin.WithSocketPath(*snapshotterSocket),
		plugin.WithNamespace(*namespace),
		plugin.WithSnapshotterName(*snapshotterName),
		plugin.WithLeaseIdleTTL(*idleTTL),
		plugin.WithDeclineTTL(*declineTTL),
		plugin.WithListableTimeout(*listableTimeout),
		plugin.WithStartupTimeout(*startupTimeout),
		plugin.WithDrainTimeout(*drainTimeout),
	}
	if *workDir != "" {
		opts = append(opts, plugin.WithWorkDir(*workDir))
	}
	d, err := newNydusDriver(config{
		DockerConfigPath:    *dockerConfigPath,
		DragonflyProxyAddr:  *dragonflyProxyAddr,
		DragonflyHealthURL:  *dragonflyHealthURL,
		VerifyDragonflyPing: *verifyDragonflyPing,
		ProcFilesystemsPath: *procFilesystems,
		CachefilesDevPath:   *cachefilesDev,
		BlobSeedURL:         *blobSeedURL,
		DriverOpts:          opts,
	})
	if err != nil {
		slog.Error("Failed to start the Nydus plugin", slog.Any("err", err))
		os.Exit(1)
	}
	defer func() { _ = d.Close() }()

	if err := plugin.ServeUnixSocket(ctx, *listenSocket, *drainCompleteFile, d.Driver); err != nil {
		slog.Error("Plugin server failed", slog.Any("err", err))
		os.Exit(1)
	}
}
