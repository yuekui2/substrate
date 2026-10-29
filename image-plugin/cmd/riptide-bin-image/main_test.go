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
	"archive/tar"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"sigs.k8s.io/yaml"
)

// TestDaemonSetPatch applies the patch to manifests/riptide.yaml so renaming
// a container or volume there breaks this test.
func TestDaemonSetPatch(t *testing.T) {
	const image = "example.com/riptide-bins@sha256:0000000000000000000000000000000000000000000000000000000000000000"

	manifest, err := os.ReadFile("../../manifests/riptide.yaml")
	if err != nil {
		t.Fatal(err)
	}
	origJSON, err := yaml.YAMLToJSON(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var orig appsv1.DaemonSet
	if err := json.Unmarshal(origJSON, &orig); err != nil {
		t.Fatal(err)
	}

	patch, err := daemonSetPatch(image)
	if err != nil {
		t.Fatal(err)
	}
	patchJSON, err := yaml.YAMLToJSON(patch)
	if err != nil {
		t.Fatal(err)
	}
	mergedJSON, err := strategicpatch.StrategicMergePatch(origJSON, patchJSON, appsv1.DaemonSet{})
	if err != nil {
		t.Fatal(err)
	}
	var merged appsv1.DaemonSet
	if err := json.Unmarshal(mergedJSON, &merged); err != nil {
		t.Fatal(err)
	}
	spec := merged.Spec.Template.Spec

	if got, want := len(spec.Containers), len(orig.Spec.Template.Spec.Containers); got != want {
		t.Fatalf("patch changed container count from %d to %d; riptide-provider was renamed?", want, got)
	}
	if len(spec.InitContainers) != 1 || spec.InitContainers[0].Image != image {
		t.Errorf("InitContainers = %+v, want one container with image %s", spec.InitContainers, image)
	}

	var provider *corev1.Container
	for i := range spec.Containers {
		if spec.Containers[i].Name == "riptide-provider" {
			provider = &spec.Containers[i]
		}
	}
	if provider == nil {
		t.Fatal("riptide-provider container not found")
	}
	env := map[string]string{}
	for _, e := range provider.Env {
		env[e.Name] = e.Value
	}
	if env["RIPTIDE_BIN_DIR"] != hostBinDir {
		t.Errorf("RIPTIDE_BIN_DIR = %q, want %q", env["RIPTIDE_BIN_DIR"], hostBinDir)
	}
	if _, ok := env["KUBE_HOME"]; !ok {
		t.Error("patch dropped existing KUBE_HOME env")
	}

	var hostPath string
	for _, v := range spec.Volumes {
		if v.Name == "kube-home-bin" && v.HostPath != nil {
			hostPath = v.HostPath.Path
		}
	}
	if filepath.Join(hostPath, "ate-riptide") != hostBinDir {
		t.Errorf("kube-home-bin hostPath = %q, want parent of %s", hostPath, hostBinDir)
	}
}

func TestElfArch(t *testing.T) {
	textFile := filepath.Join(t.TempDir(), "script.sh")
	if err := os.WriteFile(textFile, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	testBin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{name: "test binary", path: testBin, want: runtime.GOARCH},
		{name: "shell script", path: textFile, wantErr: true},
		{name: "missing", path: filepath.Join(t.TempDir(), "missing"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "test binary" && (runtime.GOOS != "linux" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64")) {
				t.Skipf("test binary is not a linux amd64/arm64 ELF on %s/%s", runtime.GOOS, runtime.GOARCH)
			}
			got, err := elfArch(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("elfArch() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("elfArch() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBinariesLayer(t *testing.T) {
	dir := t.TempDir()
	bins := []binary{
		{name: "gcfsd-v2", path: filepath.Join(dir, "a")},
		{name: snapshotterName, path: filepath.Join(dir, "b")},
	}
	for _, b := range bins {
		if err := os.WriteFile(b.path, []byte(b.name), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	layer, err := binariesLayer(bins)
	if err != nil {
		t.Fatal(err)
	}
	rc, err := layer.Uncompressed()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()

	got := map[string]string{}
	tr := tar.NewReader(rc)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Mode != 0o755 || hdr.ModTime.Unix() != 0 {
			t.Errorf("%s: mode %o mtime %v, want 0755 at epoch", hdr.Name, hdr.Mode, hdr.ModTime)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		got[hdr.Name] = string(data)
	}
	want := map[string]string{
		"riptide/gcfsd-v2":             "gcfsd-v2",
		"riptide/containerd-gcfs-grpc": "containerd-gcfs-grpc",
	}
	if len(got) != len(want) {
		t.Fatalf("layer files = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("layer file %s = %q, want %q", k, got[k], v)
		}
	}

	again, err := binariesLayer(bins)
	if err != nil {
		t.Fatal(err)
	}
	d1, err := layer.Digest()
	if err != nil {
		t.Fatal(err)
	}
	d2, err := again.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if d1 != d2 {
		t.Errorf("layer digest changed between builds: %s != %s", d1, d2)
	}
}
