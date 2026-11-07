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

package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"path/filepath"
	"sync"
	"testing"

	imageprovider "github.com/ai-on-gke/image-provider-plugin/api/imageprovider/v1alpha1"
	"github.com/ai-on-gke/image-provider-plugin/client"
	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeServer struct {
	imageprovider.UnimplementedImageProviderPluginServer

	canProvideResp *imageprovider.CanProvideResponse
	prepareResp    *imageprovider.PrepareImageResponse
	prepareErr     error

	mu            sync.Mutex
	lastAuth      *imageprovider.RegistryAuth
	lastForm      imageprovider.LayerForm
	lastRelease   *imageprovider.ReleaseImageRequest
	lastReconcile *imageprovider.ReconcileImagesRequest
}

func (s *fakeServer) CanProvide(_ context.Context, req *imageprovider.CanProvideRequest) (*imageprovider.CanProvideResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAuth = req.GetAuth()
	s.lastForm = req.GetForm()
	return s.canProvideResp, nil
}

func (s *fakeServer) PrepareImage(_ context.Context, req *imageprovider.PrepareImageRequest) (*imageprovider.PrepareImageResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAuth = req.GetAuth()
	s.lastForm = req.GetForm()
	if s.prepareErr != nil {
		return nil, s.prepareErr
	}
	return s.prepareResp, nil
}

func (s *fakeServer) ReleaseImage(_ context.Context, req *imageprovider.ReleaseImageRequest) (*imageprovider.ReleaseImageResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastRelease = req
	return &imageprovider.ReleaseImageResponse{}, nil
}

func (s *fakeServer) ReconcileImages(_ context.Context, req *imageprovider.ReconcileImagesRequest) (*imageprovider.ReconcileImagesResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastReconcile = req
	return &imageprovider.ReconcileImagesResponse{}, nil
}

type staticKeychain struct {
	username string
	password string
}

func (k staticKeychain) Resolve(_ authn.Resource) (authn.Authenticator, error) {
	return authn.FromConfig(authn.AuthConfig{
		Username: k.username,
		Password: k.password,
	}), nil
}

// startFakeServer serves srv on a temporary socket and returns a client for it.
func startFakeServer(t *testing.T, srv *fakeServer) *client.GRPCClient {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "image-provider.sock")
	lis, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen unix %s: %v", sock, err)
	}
	grpcSrv := grpc.NewServer()
	imageprovider.RegisterImageProviderPluginServer(grpcSrv, srv)
	go func() {
		_ = grpcSrv.Serve(lis)
	}()
	t.Cleanup(grpcSrv.Stop)

	c, err := client.NewGRPCClient(sock)
	if err != nil {
		t.Fatalf("NewGRPCClient: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestGRPCClientCanProvide(t *testing.T) {
	tests := []struct {
		name           string
		canProvide     bool
		state          imageprovider.PluginState
		wantCanProvide bool
	}{
		{name: "serving", canProvide: true, state: imageprovider.PluginState_PLUGIN_STATE_SERVING, wantCanProvide: true},
		{name: "serving but declines image", canProvide: false, state: imageprovider.PluginState_PLUGIN_STATE_SERVING, wantCanProvide: false},
		{name: "draining", canProvide: true, state: imageprovider.PluginState_PLUGIN_STATE_DRAINING, wantCanProvide: false},
		{name: "unspecified", canProvide: true, state: imageprovider.PluginState_PLUGIN_STATE_UNSPECIFIED, wantCanProvide: false},
		{name: "unknown value", canProvide: true, state: imageprovider.PluginState(99), wantCanProvide: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeServer{
				canProvideResp: &imageprovider.CanProvideResponse{
					CanProvide:   tc.canProvide,
					ProviderName: "nydus",
					State:        tc.state,
				},
			}
			c := startFakeServer(t, fake)

			ok, err := c.CanProvide(t.Context(), &client.PrepareRequest{ImageRef: "example.com/app:v1"})
			if err != nil {
				t.Fatalf("CanProvide: %v", err)
			}
			if ok != tc.wantCanProvide {
				t.Errorf("CanProvide = %v, want %v", ok, tc.wantCanProvide)
			}
			if fake.lastForm != imageprovider.LayerForm_LAYER_FORM_DIRS {
				t.Errorf("CanProvide sent form %v, want LAYER_FORM_DIRS", fake.lastForm)
			}
			state, err := c.State(t.Context())
			if err != nil {
				t.Fatalf("State: %v", err)
			}
			if state != tc.state {
				t.Errorf("State = %v, want %v", state, tc.state)
			}
			if c.Name() != "nydus" {
				t.Errorf("Name = %q, want %q", c.Name(), "nydus")
			}
		})
	}
}

func TestGRPCClientCanProvideNoSocket(t *testing.T) {
	c, err := client.NewGRPCClient(filepath.Join(t.TempDir(), "missing.sock"))
	if err != nil {
		t.Fatalf("NewGRPCClient: %v", err)
	}
	defer c.Close()

	ok, err := c.CanProvide(t.Context(), &client.PrepareRequest{ImageRef: "example.com/app:v1"})
	if err != nil || ok {
		t.Errorf("CanProvide = (%v, %v), want (false, nil)", ok, err)
	}
	state, err := c.State(t.Context())
	if err != nil || state != imageprovider.PluginState_PLUGIN_STATE_UNSPECIFIED {
		t.Errorf("State = (%v, %v), want (UNSPECIFIED, nil)", state, err)
	}
}

func TestGRPCClientPrepareImage(t *testing.T) {
	cfgJSON, err := json.Marshal(v1.Config{
		Entrypoint: []string{"/bin/sh"},
		Env:        []string{"FOO=bar"},
	})
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeServer{
		prepareResp: &imageprovider.PrepareImageResponse{
			ImageDigest:  "sha256:abc123",
			ConfigJson:   cfgJSON,
			ProviderName: "riptide",
			Layers: &imageprovider.PrepareImageResponse_LayerDirs{
				LayerDirs: &imageprovider.LayerDirs{Dirs: []string{"/layers/1", "/layers/2"}},
			},
		},
	}
	c := startFakeServer(t, fake)

	ctx := client.WithKeychainContext(t.Context(), staticKeychain{
		username: "_token",
		password: "secretpassword",
	})
	res, err := c.PrepareImage(ctx, &client.PrepareRequest{ImageRef: "gcr.io/proj/image:v1"})
	if err != nil {
		t.Fatalf("PrepareImage: %v", err)
	}
	if res.ImageDigest != "sha256:abc123" {
		t.Errorf("ImageDigest = %q, want sha256:abc123", res.ImageDigest)
	}
	if len(res.LayerDirs) != 2 || res.LayerDirs[0] != "/layers/1" {
		t.Errorf("LayerDirs = %v, want [/layers/1 /layers/2]", res.LayerDirs)
	}
	if res.Config == nil || len(res.Config.Entrypoint) != 1 || res.Config.Entrypoint[0] != "/bin/sh" {
		t.Errorf("Config = %+v, want entrypoint /bin/sh", res.Config)
	}
	if res.ProviderName != "riptide" {
		t.Errorf("ProviderName = %q, want riptide", res.ProviderName)
	}
	if fake.lastForm != imageprovider.LayerForm_LAYER_FORM_DIRS {
		t.Errorf("PrepareImage sent form %v, want LAYER_FORM_DIRS", fake.lastForm)
	}
	if fake.lastAuth.GetUsername() != "_token" || fake.lastAuth.GetPassword() != "secretpassword" || fake.lastAuth.GetServerAddress() != "gcr.io" {
		t.Errorf("auth sent = %v, want keychain credentials for gcr.io", fake.lastAuth)
	}
}

func TestGRPCClientPrepareImageErrors(t *testing.T) {
	tests := []struct {
		name             string
		resp             *imageprovider.PrepareImageResponse
		err              error
		wantNotSupported bool
	}{
		{name: "failed precondition", err: status.Error(codes.FailedPrecondition, "no streaming index"), wantNotSupported: true},
		{name: "unimplemented", err: status.Error(codes.Unimplemented, "not served"), wantNotSupported: true},
		{name: "internal", err: status.Error(codes.Internal, "boom")},
		{name: "bad config", resp: &imageprovider.PrepareImageResponse{ConfigJson: []byte("{")}},
		{name: "no layers", resp: &imageprovider.PrepareImageResponse{ImageDigest: "sha256:abc123"}},
		{name: "wrong form", resp: &imageprovider.PrepareImageResponse{
			ImageDigest: "sha256:abc123",
			Layers:      &imageprovider.PrepareImageResponse_Rootfs{Rootfs: &imageprovider.Rootfs{Dir: "/rootfs"}},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := startFakeServer(t, &fakeServer{prepareResp: tc.resp, prepareErr: tc.err})
			_, err := c.PrepareImage(t.Context(), &client.PrepareRequest{ImageRef: "docker.io/library/alpine:latest"})
			if err == nil {
				t.Fatal("PrepareImage succeeded, want error")
			}
			if got := errors.Is(err, client.ErrNotSupported); got != tc.wantNotSupported {
				t.Errorf("errors.Is(%v, ErrNotSupported) = %v, want %v", err, got, tc.wantNotSupported)
			}
		})
	}
}

func TestGRPCClientReleaseImage(t *testing.T) {
	fake := &fakeServer{}
	c := startFakeServer(t, fake)

	if err := c.ReleaseImage(t.Context(), &client.ReleaseRequest{ImageRef: "example.com/app:v1"}); err != nil {
		t.Fatalf("ReleaseImage: %v", err)
	}
	if fake.lastRelease.GetImageRef() != "example.com/app:v1" {
		t.Errorf("released %v, want example.com/app:v1", fake.lastRelease)
	}
}

func TestGRPCClientReconcileImages(t *testing.T) {
	fake := &fakeServer{}
	c := startFakeServer(t, fake)

	err := c.ReconcileImages(t.Context(), []*client.ActiveImageLayers{{
		ImageRef:    "example.com/app:v1",
		ImageDigest: "sha256:111",
		LayerDirs:   []string{"/l/1"},
		RefCount:    2,
	}})
	if err != nil {
		t.Fatalf("ReconcileImages: %v", err)
	}
	imgs := fake.lastReconcile.GetActiveImages()
	if len(imgs) != 1 || imgs[0].GetImageRef() != "example.com/app:v1" || imgs[0].GetRefCount() != 2 {
		t.Errorf("reconciled %v, want example.com/app:v1 with ref count 2", imgs)
	}
}
