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
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"go.ate.dev/substrate/image-plugin/pkg/plugin"
	"go.ate.dev/substrate/internal/imagepluginclient"
)

// seedGet fetches registry content by digest from the blob seed servers,
// which serve /v2/<repo>/{blobs,manifests}/<digest> without auth. kind is
// "blobs" or "manifests".
func (nd *NydusDragonflyDriver) seedGet(ctx context.Context, repo name.Repository, kind, digest string) (io.ReadCloser, error) {
	u := nd.blobSeedURL + "/v2/" + repo.RepositoryStr() + "/" + kind + "/" + digest
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := nd.seedClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("GET %s: HTTP %d", u, resp.StatusCode)
	}
	return resp.Body, nil
}

// seedManifest reads the manifest of a digest reference from the seeds.
func (nd *NydusDragonflyDriver) seedManifest(ctx context.Context, ref name.Reference) (*v1.Manifest, error) {
	d, ok := ref.(name.Digest)
	if !ok {
		return nil, fmt.Errorf("%s is not a digest reference", ref)
	}
	rc, err := nd.seedGet(ctx, d.Context(), "manifests", d.DigestStr())
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return v1.ParseManifest(rc)
}

// seedLayer reads an uncompressed layer from the seeds. It handles gzip
// and uncompressed layers.
func (nd *NydusDragonflyDriver) seedLayer(ctx context.Context, repo name.Repository, digest string) (io.ReadCloser, error) {
	rc, err := nd.seedGet(ctx, repo, "blobs", digest)
	if err != nil {
		return nil, err
	}
	br := bufio.NewReader(rc)
	magic, err := br.Peek(2)
	if err != nil && err != io.EOF {
		_ = rc.Close()
		return nil, err
	}
	if !bytes.Equal(magic, []byte{0x1f, 0x8b}) {
		return readCloser{Reader: br, close: rc.Close}, nil
	}
	zr, err := gzip.NewReader(br)
	if err != nil {
		_ = rc.Close()
		return nil, err
	}
	return readCloser{Reader: zr, close: func() error {
		_ = zr.Close()
		return rc.Close()
	}}, nil
}

type readCloser struct {
	io.Reader
	close func() error
}

func (r readCloser) Close() error { return r.close() }

func logSeedMiss(ctx context.Context, what, ref string, err error) {
	slog.WarnContext(ctx, "Blob seed miss, falling back to the registry",
		slog.String("what", what), slog.String("ref", ref), slog.Any("err", err))
}

// resolveImage resolves an image from the seeds, then from the registry.
func (nd *NydusDragonflyDriver) resolveImage(ctx context.Context, refStr string, authConfig *imagepluginclient.AuthConfig) (string, *v1.Config, []string, []string, error) {
	ref, err := name.ParseReference(refStr)
	if err != nil {
		return "", nil, nil, nil, fmt.Errorf("parsing reference %q: %w", refStr, err)
	}
	digest, m, cfg, err := nd.seedImage(ctx, ref)
	if err != nil {
		logSeedMiss(ctx, "image", refStr, err)
		img, err := remote.Image(ref, plugin.RemoteOptions(ctx, nd.Driver.Keychain(), authConfig)...)
		if err != nil {
			return "", nil, nil, nil, fmt.Errorf("remote.Image: %w", err)
		}
		d, err := img.Digest()
		if err != nil {
			return "", nil, nil, nil, fmt.Errorf("reading image digest: %w", err)
		}
		if cfg, err = img.ConfigFile(); err != nil {
			return "", nil, nil, nil, fmt.Errorf("reading image config: %w", err)
		}
		if m, err = img.Manifest(); err != nil {
			return "", nil, nil, nil, fmt.Errorf("reading image manifest: %w", err)
		}
		digest = d.String()
	}
	layerDigests := make([]string, len(m.Layers))
	for i, l := range m.Layers {
		layerDigests[i] = l.Digest.String()
	}
	diffIDs := make([]string, len(cfg.RootFS.DiffIDs))
	for i, diff := range cfg.RootFS.DiffIDs {
		diffIDs[i] = diff.String()
	}
	return digest, &cfg.Config, diffIDs, layerDigests, nil
}

// seedImage reads the manifest and config of a digest reference from the
// seeds and checks the manifest digest.
func (nd *NydusDragonflyDriver) seedImage(ctx context.Context, ref name.Reference) (string, *v1.Manifest, *v1.ConfigFile, error) {
	d, ok := ref.(name.Digest)
	if !ok {
		return "", nil, nil, fmt.Errorf("%s is not a digest reference", ref)
	}
	rc, err := nd.seedGet(ctx, d.Context(), "manifests", d.DigestStr())
	if err != nil {
		return "", nil, nil, err
	}
	raw, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		return "", nil, nil, err
	}
	h, _, err := v1.SHA256(bytes.NewReader(raw))
	if err != nil {
		return "", nil, nil, err
	}
	if h.String() != d.DigestStr() {
		return "", nil, nil, fmt.Errorf("seed manifest digest %s, want %s", h, d.DigestStr())
	}
	m, err := v1.ParseManifest(bytes.NewReader(raw))
	if err != nil {
		return "", nil, nil, err
	}
	rc, err = nd.seedGet(ctx, d.Context(), "blobs", m.Config.Digest.String())
	if err != nil {
		return "", nil, nil, err
	}
	defer func() { _ = rc.Close() }()
	cfg, err := v1.ParseConfigFile(rc)
	if err != nil {
		return "", nil, nil, err
	}
	return d.DigestStr(), m, cfg, nil
}
