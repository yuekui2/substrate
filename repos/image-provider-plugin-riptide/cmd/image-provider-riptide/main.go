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

// Command image-provider-riptide is an image provider plugin that streams
// images through Riptide: gcfsd serves the layers over FUSE and
// containerd-gcfs-grpc exposes them as containerd snapshots.
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
		listenSocket      = flag.String("listen-socket", plugin.DefaultListenSocket, "Unix socket to serve the plugin on")
		snapshotterSocket = flag.String("snapshotter-socket", defaultSnapshotterSocket, "Unix socket of containerd-gcfs-grpc")
		keychainSocket    = flag.String("keychain-socket", defaultKeychainSocket, "Unix socket of the gcfsd keychain service")
		workDir           = flag.String("work-dir", "", "Work directory for image layer links (default "+plugin.DefaultBaseWorkDir+"/"+providerName+")")
		namespace         = flag.String("namespace", plugin.DefaultNamespace, "containerd namespace sent to containerd-gcfs-grpc")
		snapshotterName   = flag.String("snapshotter-name", "gcfs", "Snapshotter name sent in snapshot requests")
		idleTTL           = flag.Duration("idle-ttl", plugin.DefaultLeaseIdleTTL, "How long to keep an unused image mounted")
		declineTTL        = flag.Duration("decline-ttl", plugin.DefaultDeclineTTL, "How long to remember an image the plugin cannot serve")
		listableTimeout   = flag.Duration("listable-timeout", plugin.DefaultListableTimeout, "How long to wait for a mounted layer to become listable")
		startupTimeout    = flag.Duration("startup-timeout", plugin.DefaultStartupTimeout, "How long to wait at startup for containerd-gcfs-grpc and gcfsd")
		drainTimeout      = flag.Duration("drain-timeout", plugin.DefaultDrainTimeout, "How long to wait after SIGTERM for image leases to end")
		drainCompleteFile = flag.String("drain-complete-file", plugin.DefaultDrainCompleteFile, "File to write once draining finishes")
		probe             = flag.Bool("probe", false, "Check that the plugin on --listen-socket is ready and exit")
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
	d, err := newRiptideDriver(config{
		KeychainSocket: *keychainSocket,
		DriverOpts:     opts,
	})
	if err != nil {
		slog.Error("Failed to start the Riptide plugin", slog.Any("err", err))
		os.Exit(1)
	}
	defer func() { _ = d.Close() }()

	if err := plugin.ServeUnixSocket(ctx, *listenSocket, *drainCompleteFile, d.Driver); err != nil {
		slog.Error("Plugin server failed", slog.Any("err", err))
		os.Exit(1)
	}
}
