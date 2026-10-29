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

package imagepluginclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"go.ate.dev/substrate/api/imageplugin/v1"
	"go.ate.dev/substrate/internal/imagepluginclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeServer struct {
	imageplugin.UnimplementedImagePluginServer

	canProvideResp *imageplugin.CanProvideResponse
	prepareResp    *imageplugin.PrepareImageResponse
	prepareErr     error

	lastAuth      *imageplugin.RegistryAuth
	lastRelease   *imageplugin.ReleaseImageRequest
	lastReconcile *imageplugin.ReconcileImagesRequest
}

func (s *fakeServer) CanProvide(_ context.Context, req *imageplugin.CanProvideRequest) (*imageplugin.CanProvideResponse, error) {
	s.lastAuth = req.GetAuth()
	if s.canProvideResp != nil {
		return s.canProvideResp, nil
	}
	return &imageplugin.CanProvideResponse{CanProvide: true, ProviderName: "riptide", State: imageplugin.PluginState_PLUGIN_STATE_SERVING}, nil
}

func (s *fakeServer) PrepareImage(_ context.Context, req *imageplugin.PrepareImageRequest) (*imageplugin.PrepareImageResponse, error) {
	s.lastAuth = req.GetAuth()
	if s.prepareErr != nil {
		return nil, s.prepareErr
	}
	return s.prepareResp, nil
}

func (s *fakeServer) ReleaseImage(_ context.Context, req *imageplugin.ReleaseImageRequest) (*imageplugin.ReleaseImageResponse, error) {
	s.lastRelease = req
	return &imageplugin.ReleaseImageResponse{}, nil
}

func (s *fakeServer) ReconcileImages(_ context.Context, req *imageplugin.ReconcileImagesRequest) (*imageplugin.ReconcileImagesResponse, error) {
	s.lastReconcile = req
	return &imageplugin.ReconcileImagesResponse{}, nil
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

func startFakeServer(t *testing.T, srv *fakeServer) string {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "imageplugin.sock")
	lis, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen unix %s: %v", sock, err)
	}
	grpcSrv := grpc.NewServer()
	imageplugin.RegisterImagePluginServer(grpcSrv, srv)
	go func() {
		_ = grpcSrv.Serve(lis)
	}()
	t.Cleanup(func() {
		grpcSrv.Stop()
		_ = lis.Close()
	})
	return sock
}

func TestGRPCClient_CanProvide(t *testing.T) {
	tests := []struct {
		name           string
		canProvide     bool
		state          imageplugin.PluginState
		wantCanProvide bool
	}{
		{name: "serving", canProvide: true, state: imageplugin.PluginState_PLUGIN_STATE_SERVING, wantCanProvide: true},
		{name: "serving but declines image", canProvide: false, state: imageplugin.PluginState_PLUGIN_STATE_SERVING, wantCanProvide: false},
		{name: "draining", canProvide: true, state: imageplugin.PluginState_PLUGIN_STATE_DRAINING, wantCanProvide: false},
		{name: "unspecified", canProvide: true, state: imageplugin.PluginState_PLUGIN_STATE_UNSPECIFIED, wantCanProvide: false},
		{name: "unknown value", canProvide: true, state: imageplugin.PluginState(99), wantCanProvide: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeServer{
				canProvideResp: &imageplugin.CanProvideResponse{
					CanProvide:   tc.canProvide,
					ProviderName: "spegel",
					State:        tc.state,
				},
			}
			sock := startFakeServer(t, fake)

			client, err := imagepluginclient.NewGRPCClient(sock)
			if err != nil {
				t.Fatalf("NewGRPCClient: %v", err)
			}
			defer client.Close()

			ctx := context.Background()
			ok, err := client.CanProvide(ctx, &imagepluginclient.PrepareRequest{ImageRef: "example.com/app:v1"})
			if err != nil {
				t.Fatalf("CanProvide: %v", err)
			}
			if ok != tc.wantCanProvide {
				t.Errorf("CanProvide = %v, want %v", ok, tc.wantCanProvide)
			}
			state, err := client.State(ctx)
			if err != nil {
				t.Fatalf("State: %v", err)
			}
			if state != tc.state {
				t.Errorf("State = %v, want %v", state, tc.state)
			}
			if client.Name() != "spegel" {
				t.Errorf("Name = %q, want %q", client.Name(), "spegel")
			}
		})
	}
}

func TestGRPCClient_PrepareImage_Success(t *testing.T) {
	cfgJSON, _ := json.Marshal(v1.Config{
		Entrypoint: []string{"/bin/sh"},
		Env:        []string{"FOO=bar"},
	})
	fake := &fakeServer{
		prepareResp: &imageplugin.PrepareImageResponse{
			ImageDigest:  "sha256:abc123",
			ConfigJson:   cfgJSON,
			LayerDirs:    []string{"/layers/1", "/layers/2"},
			ProviderName: "riptide",
		},
	}
	sock := startFakeServer(t, fake)

	client, err := imagepluginclient.NewGRPCClient(sock)
	if err != nil {
		t.Fatalf("NewGRPCClient: %v", err)
	}
	defer client.Close()

	ctx := imagepluginclient.WithKeychainContext(context.Background(), staticKeychain{
		username: "_token",
		password: "secretpassword",
	})

	res, err := client.PrepareImage(ctx, &imagepluginclient.PrepareRequest{ImageRef: "gcr.io/proj/image:v1"})
	if err != nil {
		t.Fatalf("PrepareImage: %v", err)
	}
	if res.ImageDigest != "sha256:abc123" {
		t.Fatalf("expected digest sha256:abc123, got %s", res.ImageDigest)
	}
	if len(res.LayerDirs) != 2 || res.LayerDirs[0] != "/layers/1" {
		t.Fatalf("unexpected layer dirs: %v", res.LayerDirs)
	}
	if res.Config == nil || len(res.Config.Entrypoint) != 1 || res.Config.Entrypoint[0] != "/bin/sh" {
		t.Fatalf("unexpected parsed config: %+v", res.Config)
	}

	if fake.lastAuth == nil || fake.lastAuth.Username != "_token" || fake.lastAuth.Password != "secretpassword" {
		t.Fatalf("expected registry auth forwarded from keychain, got: %+v", fake.lastAuth)
	}
}

func TestGRPCClient_PrepareImage_FallbackOnError(t *testing.T) {
	fake := &fakeServer{
		prepareErr: status.Error(codes.FailedPrecondition, "image has no streaming index"),
	}
	sock := startFakeServer(t, fake)

	client, err := imagepluginclient.NewGRPCClient(sock)
	if err != nil {
		t.Fatalf("NewGRPCClient: %v", err)
	}
	defer client.Close()

	_, err = client.PrepareImage(context.Background(), &imagepluginclient.PrepareRequest{ImageRef: "docker.io/library/alpine:latest"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, imagepluginclient.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported wrapping, got: %v", err)
	}
}

func TestGRPCClient_ReleaseImage(t *testing.T) {
	fake := &fakeServer{}
	sock := startFakeServer(t, fake)

	client, err := imagepluginclient.NewGRPCClient(sock)
	if err != nil {
		t.Fatalf("NewGRPCClient: %v", err)
	}
	defer client.Close()

	err = client.ReleaseImage(context.Background(), &imagepluginclient.ReleaseRequest{
		ImageRef: "example.com/app:v1",
	})
	if err != nil {
		t.Fatalf("ReleaseImage: %v", err)
	}
	if fake.lastRelease == nil || fake.lastRelease.GetImageRef() != "example.com/app:v1" {
		t.Fatalf("unexpected release call: %+v", fake.lastRelease)
	}
}

func TestGRPCClient_ReconcileImages(t *testing.T) {
	fake := &fakeServer{}
	sock := startFakeServer(t, fake)

	client, err := imagepluginclient.NewGRPCClient(sock)
	if err != nil {
		t.Fatalf("NewGRPCClient: %v", err)
	}
	defer client.Close()

	err = client.ReconcileImages(context.Background(), []*imagepluginclient.ActiveImageLayers{
		{
			ImageRef:    "example.com/app:v1",
			ImageDigest: "sha256:111",
			LayerDirs:   []string{"/l/1"},
			RefCount:    2,
		},
	})
	if err != nil {
		t.Fatalf("ReconcileImages: %v", err)
	}
	if fake.lastReconcile == nil || len(fake.lastReconcile.GetActiveImages()) != 1 {
		t.Fatalf("unexpected reconcile call: %+v", fake.lastReconcile)
	}
	img := fake.lastReconcile.GetActiveImages()[0]
	if img.GetImageRef() != "example.com/app:v1" || img.GetRefCount() != 2 {
		t.Fatalf("unexpected reconciled item: %+v", img)
	}
}
