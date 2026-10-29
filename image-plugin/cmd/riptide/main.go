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

// Command image-plugin-riptide runs the out-of-process ImagePlugin gRPC plugin
// daemon for Google Riptide v2 (containerd-gcfs-grpc + gcfsd).
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"go.ate.dev/substrate/image-plugin/pkg/plugin"
)

func main() {
	var (
		listenSocket      = flag.String("listen-socket", plugin.DefaultListenSocket, "Unix domain socket path for ImagePlugin")
		snapshotterSocket = flag.String("snapshotter-socket", DefaultRiptideSocket, "Unix domain socket path for containerd-gcfs-grpc")
		keychainSocket    = flag.String("keychain-socket", DefaultRiptideKeychainSocket, "Unix domain socket path for Riptide gcfsd Keychain service")
		workDir           = flag.String("work-dir", "", "Directory for layer wrapper symlinks (defaults to /var/lib/ate/image-plugin-cache/riptide)")
		namespace         = flag.String("namespace", plugin.DefaultNamespace, "containerd namespace header sent to containerd-gcfs-grpc")
		snapshotterName   = flag.String("snapshotter-name", "gcfs", "Snapshotter plugin name sent in Prepare/View requests")
		idleTTL           = flag.Duration("idle-ttl", plugin.DefaultLeaseIdleTTL, "Duration to keep layer views mounted after refCount drops to 0")
		declineTTL        = flag.Duration("decline-ttl", plugin.DefaultDeclineTTL, "Duration to cache declined (non-streamable) images")
		listableTimeout   = flag.Duration("listable-timeout", plugin.DefaultListableTimeout, "Maximum time to wait for a mounted layer directory to become listable")
		startupTimeout    = flag.Duration("startup-timeout", plugin.DefaultStartupTimeout, "Maximum duration to wait on startup for the underlying provider to pass readiness probe")
		drainTimeout      = flag.Duration("drain-timeout", plugin.DefaultDrainTimeout, "Maximum duration to wait in lame-duck mode after SIGTERM for active image leases to reach 0")
		drainCompleteFile = flag.String("drain-complete-file", plugin.DefaultDrainCompleteFile, "Path to the drain-complete marker file written once lame-duck draining finishes")
		probe             = flag.Bool("probe", false, "Probe the running ImagePlugin socket for readiness and exit")
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
		plugin.WithKeychain(authn.NewMultiKeychain(authn.DefaultKeychain, google.Keychain)),
	}
	if *workDir != "" {
		opts = append(opts, plugin.WithWorkDir(*workDir))
	}

	driver, err := NewRiptide(RiptideConfig{
		KeychainSocket: *keychainSocket,
		ServerOpts:     opts,
	})
	if err != nil {
		slog.Error("Failed to initialize Riptide ImagePlugin driver", slog.Any("err", err))
		os.Exit(1)
	}
	defer func() { _ = driver.Close() }()

	if err := plugin.ServeUnixSocket(ctx, *listenSocket, driver.Driver); err != nil {
		slog.Error("ImagePlugin gRPC server exited with error", slog.Any("err", err))
		os.Exit(1)
	}
}
