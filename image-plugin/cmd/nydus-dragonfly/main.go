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

// Command image-plugin-dragonfly runs the out-of-process ImagePlugin gRPC
// plugin daemon for cross-node Nydus RAFS v6 + Dragonfly v2 P2P
// (containerd-nydus-grpc over Kernel EROFS + fscache backed by dfdaemon,
// dragonfly-scheduler, and dragonfly-manager).
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go.ate.dev/substrate/image-plugin/pkg/plugin"
)

func main() {
	var (
		listenSocket           = flag.String("listen-socket", plugin.DefaultListenSocket, "Unix domain socket path for ImagePlugin")
		snapshotterSocket      = flag.String("snapshotter-socket", DefaultNydusSocket, "Unix domain socket path for containerd-nydus-grpc")
		dockerConfigPath       = flag.String("docker-config", DefaultNydusDockerConfigPath, "Path to Docker config.json shared with containerd-nydus-grpc for registry credentials")
		dragonflyProxyAddr     = flag.String("dragonfly-proxy-addr", "", "Local Dragonfly dfdaemon HTTP proxy endpoint (defaults to DRAGONFLY_PROXY_ADDR env var or http://127.0.0.1:4001)")
		dragonflySchedulerAddr = flag.String("dragonfly-scheduler-addr", "", "Cluster Dragonfly Scheduler gRPC endpoint (defaults to DRAGONFLY_SCHEDULER_ADDR env var or dragonfly-scheduler.ate-system.svc.cluster.local:8002)")
		dragonflyManagerAddr   = flag.String("dragonfly-manager-addr", "", "Cluster Dragonfly Manager gRPC endpoint (defaults to DRAGONFLY_MANAGER_ADDR env var or dragonfly-manager.ate-system.svc.cluster.local:65003)")
		verifyDragonflyPing    = flag.Bool("verify-dragonfly-ping", true, "Verify that local Dragonfly dfdaemon responds 200 OK on its health endpoint (DRAGONFLY_HEALTH_URL env var or http://127.0.0.1:4003/healthy) before serving images")
		procFilesystems        = flag.String("proc-filesystems", DefaultProcFilesystemsPath, "Path to /proc/filesystems used to verify Kernel EROFS registration")
		cachefilesDev          = flag.String("cachefiles-dev", DefaultCachefilesDevPath, "Path to /dev/cachefiles device required for Kernel EROFS fscache mode")
		workDir                = flag.String("work-dir", "", "Directory for layer wrapper symlinks (defaults to /var/lib/ate/image-plugin-cache/nydus-dragonfly)")
		namespace              = flag.String("namespace", plugin.DefaultNamespace, "containerd namespace header sent to containerd-nydus-grpc")
		snapshotterName        = flag.String("snapshotter-name", "nydus", "Snapshotter plugin name sent in Prepare/View requests")
		idleTTL                = flag.Duration("idle-ttl", plugin.DefaultLeaseIdleTTL, "Duration to keep layer views mounted after refCount drops to 0")
		declineTTL             = flag.Duration("decline-ttl", plugin.DefaultDeclineTTL, "Duration to cache declined (non-streamable) images")
		listableTimeout        = flag.Duration("listable-timeout", plugin.DefaultListableTimeout, "Maximum time to wait for a mounted EROFS layer directory to become listable")
		startupTimeout         = flag.Duration("startup-timeout", plugin.DefaultStartupTimeout, "Maximum duration to wait on startup for the underlying provider to pass readiness probe")
		drainTimeout           = flag.Duration("drain-timeout", plugin.DefaultDrainTimeout, "Maximum duration to wait in lame-duck mode after SIGTERM for active image leases to reach 0")
		drainCompleteFile      = flag.String("drain-complete-file", plugin.DefaultDrainCompleteFile, "Path to the drain-complete marker file written once lame-duck draining finishes")
		probe                  = flag.Bool("probe", false, "Probe the running ImagePlugin socket for readiness and exit")
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if *probe {
		if err := plugin.ProbeUnixSocket(ctx, *listenSocket); err != nil {
			slog.Error("ImagePlugin readiness probe failed", slog.String("socket", *listenSocket), slog.Any("err", err))
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
		plugin.WithDrainCompleteFile(*drainCompleteFile),
	}
	if *workDir != "" {
		opts = append(opts, plugin.WithWorkDir(*workDir))
	}

	driver, err := NewNydusDragonfly(NydusDragonflyConfig{
		DockerConfigPath:       *dockerConfigPath,
		DragonflyProxyAddr:     *dragonflyProxyAddr,
		DragonflySchedulerAddr: *dragonflySchedulerAddr,
		DragonflyManagerAddr:   *dragonflyManagerAddr,
		VerifyDragonflyPing:    *verifyDragonflyPing,
		ProcFilesystemsPath:    *procFilesystems,
		CachefilesDevPath:      *cachefilesDev,
		ServerOpts:             opts,
	})
	if err != nil {
		slog.Error("Failed to initialize cross-node Nydus + Dragonfly EROFS/fscache ImagePlugin driver", slog.Any("err", err))
		os.Exit(1)
	}
	defer func() { _ = driver.Close() }()

	slog.Info("Initialized cross-node Nydus + Dragonfly v2 (Kernel EROFS + fscache) ImagePlugin driver",
		slog.String("dragonflyProxyAddr", driver.DragonflyProxyAddr()),
		slog.String("dragonflySchedulerAddr", driver.DragonflySchedulerAddr()),
		slog.String("dragonflyManagerAddr", driver.DragonflyManagerAddr()),
		slog.Bool("verifyDragonflyPing", *verifyDragonflyPing),
		slog.String("procFilesystems", *procFilesystems),
		slog.String("cachefilesDev", *cachefilesDev))

	if err := plugin.ServeUnixSocket(ctx, *listenSocket, driver.Driver); err != nil {
		slog.Error("ImagePlugin gRPC server exited with error", slog.Any("err", err))
		os.Exit(1)
	}
}
