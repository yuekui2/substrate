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

// Command e2e-stream-probe exercises an OCILayerService Adapter over its Unix
// domain socket (--socket) by calling CanProvide, PrepareLayers, verifying that
// each returned layer directory is listable, and releasing the lease via
// ReleaseLayers.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/agent-substrate/substrate/internal/ocilayer"
)

type probeReport struct {
	Provider       string   `json:"provider"`
	ImageRef       string   `json:"imageRef"`
	ImageDigest    string   `json:"imageDigest,omitempty"`
	CanProvide     bool     `json:"canProvide"`
	LayerDirs      []string `json:"layerDirs,omitempty"`
	CanProvideMs   float64  `json:"canProvideMs"`
	PrepareMs      float64  `json:"prepareMs"`
	ReleaseMs      float64  `json:"releaseMs"`
	VerifiedLayers int      `json:"verifiedLayers"`
}

func main() {
	var (
		socket    = flag.String("socket", "/run/ate-image-streaming/ocilayer.sock", "Unix domain socket path for OCILayerService")
		imageRef  = flag.String("image", "", "OCI image reference to prepare and release")
		timeout   = flag.Duration("timeout", 2*time.Minute, "Overall probe timeout")
		bypassTTL = flag.Bool("bypass-ttl", true, "Pass bypass_ttl=true in ReleaseLayers")
	)
	flag.Parse()

	if *imageRef == "" {
		fmt.Fprintln(os.Stderr, "--image is required")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	client, err := ocilayer.NewGRPCClient(*socket)
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating OCILayerService client: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = client.Close() }()

	req := &ocilayer.PrepareRequest{ImageRef: *imageRef}

	canStart := time.Now()
	ok, err := client.CanProvide(ctx, req)
	canDur := time.Since(canStart)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CanProvide error: %v\n", err)
		os.Exit(1)
	}

	report := probeReport{
		Provider:     client.Name(),
		ImageRef:     *imageRef,
		CanProvide:   ok,
		CanProvideMs: float64(canDur.Microseconds()) / 1000.0,
	}
	if !ok {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		return
	}

	prepStart := time.Now()
	res, err := client.PrepareLayers(ctx, req)
	prepDur := time.Since(prepStart)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PrepareLayers error: %v\n", err)
		os.Exit(1)
	}
	report.Provider = client.Name()
	report.ImageDigest = res.ImageDigest
	report.LayerDirs = res.LayerDirs
	report.PrepareMs = float64(prepDur.Microseconds()) / 1000.0

	for i, dir := range res.LayerDirs {
		entries, err := os.ReadDir(filepath.Join(dir, "fs"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "reading layer %d (%s/fs): %v\n", i, dir, err)
			os.Exit(1)
		}
		_ = entries
		report.VerifiedLayers++
	}

	relStart := time.Now()
	if err := client.ReleaseLayers(ctx, &ocilayer.ReleaseRequest{
		ImageRef:  *imageRef,
		BypassTTL: *bypassTTL,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "ReleaseLayers error: %v\n", err)
		os.Exit(1)
	}
	report.ReleaseMs = float64(time.Since(relStart).Microseconds()) / 1000.0

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(report)
}
