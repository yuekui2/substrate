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

package imageaccel_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"go.ate.dev/substrate/oci-image-accelerator/pkg/client"
	imageaccelv1 "go.ate.dev/substrate/oci-image-accelerator/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeServer struct {
	imageaccelv1.UnimplementedImageAcceleratorServer

	canProvideResp *imageaccelv1.CanProvideResponse
	prepareResp    *imageaccelv1.PrepareImageResponse
	prepareErr     error

	lastAuth      *imageaccelv1.RegistryAuth
	lastRelease   *imageaccelv1.ReleaseImageRequest
	lastReconcile *imageaccelv1.ReconcileImagesRequest
}

func (s *fakeServer) CanProvide(_ context.Context, req *imageaccelv1.CanProvideRequest) (*imageaccelv1.CanProvideResponse, error) {
	s.lastAuth = req.GetAuth()
	if s.canProvideResp != nil {
		return s.canProvideResp, nil
	}
	return &imageaccelv1.CanProvideResponse{CanProvide: true, ProviderName: "riptide"}, nil
}

func (s *fakeServer) PrepareImage(_ context.Context, req *imageaccelv1.PrepareImageRequest) (*imageaccelv1.PrepareImageResponse, error) {
	s.lastAuth = req.GetAuth()
	if s.prepareErr != nil {
		return nil, s.prepareErr
	}
	return s.prepareResp, nil
}

func (s *fakeServer) ReleaseImage(_ context.Context, req *imageaccelv1.ReleaseImageRequest) (*imageaccelv1.ReleaseImageResponse, error) {
	s.lastRelease = req
	return &imageaccelv1.ReleaseImageResponse{}, nil
}

func (s *fakeServer) ReconcileImages(_ context.Context, req *imageaccelv1.ReconcileImagesRequest) (*imageaccelv1.ReconcileImagesResponse, error) {
	s.lastReconcile = req
	return &imageaccelv1.ReconcileImagesResponse{}, nil
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
	sock := filepath.Join(t.TempDir(), "accelerator.sock")
	lis, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen unix %s: %v", sock, err)
	}
	grpcSrv := grpc.NewServer()
	imageaccelv1.RegisterImageAcceleratorServer(grpcSrv, srv)
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
	fake := &fakeServer{
		canProvideResp: &imageaccelv1.CanProvideResponse{
			CanProvide:   true,
			ProviderName: "spegel",
		},
	}
	sock := startFakeServer(t, fake)

	client, err := imageaccel.NewGRPCClient(sock)
	if err != nil {
		t.Fatalf("NewGRPCClient: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	ok, err := client.CanProvide(ctx, &imageaccel.PrepareRequest{ImageRef: "example.com/app:v1"})
	if err != nil {
		t.Fatalf("CanProvide: %v", err)
	}
	if !ok {
		t.Fatal("expected CanProvide=true")
	}
	if client.Name() != "spegel" {
		t.Fatalf("expected provider name 'spegel', got %q", client.Name())
	}
}

func TestGRPCClient_PrepareImage_Success(t *testing.T) {
	cfgJSON, _ := json.Marshal(v1.Config{
		Entrypoint: []string{"/bin/sh"},
		Env:        []string{"FOO=bar"},
	})
	fake := &fakeServer{
		prepareResp: &imageaccelv1.PrepareImageResponse{
			ImageDigest:  "sha256:abc123",
			ConfigJson:   cfgJSON,
			LayerDirs:    []string{"/layers/1", "/layers/2"},
			ProviderName: "riptide",
		},
	}
	sock := startFakeServer(t, fake)

	client, err := imageaccel.NewGRPCClient(sock)
	if err != nil {
		t.Fatalf("NewGRPCClient: %v", err)
	}
	defer client.Close()

	ctx := imageaccel.WithKeychainContext(context.Background(), staticKeychain{
		username: "_token",
		password: "secretpassword",
	})

	res, err := client.PrepareImage(ctx, &imageaccel.PrepareRequest{ImageRef: "gcr.io/proj/image:v1"})
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

	client, err := imageaccel.NewGRPCClient(sock)
	if err != nil {
		t.Fatalf("NewGRPCClient: %v", err)
	}
	defer client.Close()

	_, err = client.PrepareImage(context.Background(), &imageaccel.PrepareRequest{ImageRef: "docker.io/library/alpine:latest"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, imageaccel.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported wrapping, got: %v", err)
	}
}

func TestGRPCClient_ReleaseImage(t *testing.T) {
	fake := &fakeServer{}
	sock := startFakeServer(t, fake)

	client, err := imageaccel.NewGRPCClient(sock)
	if err != nil {
		t.Fatalf("NewGRPCClient: %v", err)
	}
	defer client.Close()

	err = client.ReleaseImage(context.Background(), &imageaccel.ReleaseRequest{
		ImageRef:  "example.com/app:v1",
		BypassTTL: true,
	})
	if err != nil {
		t.Fatalf("ReleaseImage: %v", err)
	}
	if fake.lastRelease == nil || fake.lastRelease.GetImageRef() != "example.com/app:v1" || !fake.lastRelease.GetBypassTtl() {
		t.Fatalf("unexpected release call: %+v", fake.lastRelease)
	}
}

func TestGRPCClient_ReconcileImages(t *testing.T) {
	fake := &fakeServer{}
	sock := startFakeServer(t, fake)

	client, err := imageaccel.NewGRPCClient(sock)
	if err != nil {
		t.Fatalf("NewGRPCClient: %v", err)
	}
	defer client.Close()

	err = client.ReconcileImages(context.Background(), []*imageaccel.ActiveImageLayers{
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
