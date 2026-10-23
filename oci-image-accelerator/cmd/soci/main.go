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

// Command accelerator-soci runs the out-of-process ImageAccelerator gRPC adapter
// daemon for AWS SOCI (soci-snapshotter-grpc).
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go.ate.dev/substrate/oci-image-accelerator/pkg/adapter"
)

func main() {
	var (
		listenSocket      = flag.String("listen-socket", adapter.DefaultListenSocket, "Unix domain socket path for ImageAccelerator")
		snapshotterSocket = flag.String("snapshotter-socket", DefaultSOCISocket, "Unix domain socket path for soci-snapshotter-grpc")
		dockerConfigPath  = flag.String("docker-config", DefaultSOCIDockerConfigPath, "Path to Docker config.json shared with soci-snapshotter-grpc for registry credentials")
		maxLocalLayerSize = flag.Int64("max-local-layer-size", DefaultSOCIMaxLocalLayerSize, "Maximum compressed size in bytes for small layers (< min-layer-size) materialized locally")
		workDir           = flag.String("work-dir", "", "Directory for layer wrapper symlinks (defaults to /var/lib/ate/accelerator/soci)")
		namespace         = flag.String("namespace", adapter.DefaultNamespace, "containerd namespace header sent to soci-snapshotter-grpc")
		snapshotterName   = flag.String("snapshotter-name", "soci", "Snapshotter plugin name sent in Prepare/View requests")
		idleTTL           = flag.Duration("idle-ttl", adapter.DefaultLeaseIdleTTL, "Duration to keep layer views mounted after refCount drops to 0")
		declineTTL        = flag.Duration("decline-ttl", adapter.DefaultDeclineTTL, "Duration to cache declined (non-streamable) images")
		listableTimeout   = flag.Duration("listable-timeout", adapter.DefaultListableTimeout, "Maximum time to wait for a mounted layer directory to become listable")
		startupTimeout    = flag.Duration("startup-timeout", adapter.DefaultStartupTimeout, "Maximum duration to wait on startup for the underlying provider to pass readiness probe")
		drainTimeout      = flag.Duration("drain-timeout", adapter.DefaultDrainTimeout, "Maximum duration to wait in lame-duck mode after SIGTERM for active image leases to reach 0")
		drainCompleteFile = flag.String("drain-complete-file", adapter.DefaultDrainCompleteFile, "Path to the drain-complete marker file written once lame-duck draining finishes")
		probe             = flag.Bool("probe", false, "Probe the running ImageAccelerator socket for readiness and exit")
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if *probe {
		if err := adapter.ProbeUnixSocket(ctx, *listenSocket); err != nil {
			slog.Error("ImageAccelerator readiness probe failed", slog.String("socket", *listenSocket), slog.Any("err", err))
			os.Exit(1)
		}
		return
	}

	opts := []adapter.Option{
		adapter.WithSocketPath(*snapshotterSocket),
		adapter.WithNamespace(*namespace),
		adapter.WithSnapshotterName(*snapshotterName),
		adapter.WithLeaseIdleTTL(*idleTTL),
		adapter.WithDeclineTTL(*declineTTL),
		adapter.WithListableTimeout(*listableTimeout),
		adapter.WithStartupTimeout(*startupTimeout),
		adapter.WithDrainTimeout(*drainTimeout),
		adapter.WithDrainCompleteFile(*drainCompleteFile),
	}
	if *workDir != "" {
		opts = append(opts, adapter.WithWorkDir(*workDir))
	}

	driver, err := NewSOCI(SOCIConfig{
		DockerConfigPath:  *dockerConfigPath,
		MaxLocalLayerSize: *maxLocalLayerSize,
		AdapterOpts:       opts,
	})
	if err != nil {
		slog.Error("Failed to initialize AWS SOCI ImageAccelerator driver", slog.Any("err", err))
		os.Exit(1)
	}
	defer func() { _ = driver.Close() }()

	if err := adapter.ServeUnixSocket(ctx, *listenSocket, driver.Driver); err != nil {
		slog.Error("ImageAccelerator gRPC server exited with error", slog.Any("err", err))
		os.Exit(1)
	}
}
