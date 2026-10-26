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

// Command riptide-bin-image packages custom gcfsd and containerd-gcfs-grpc
// binaries into an image, pushes it, and optionally writes the patch that
// makes manifests/riptide.yaml run them.
package main

import (
	"archive/tar"
	"context"
	"debug/elf"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

const (
	// baseImage is busybox:1.37.0 and provides sh, cp and mv for the init
	// container.
	baseImage = "docker.io/library/busybox@sha256:bdf57e528e45e4433820e045b29b4597825a1c9e38353532d90a01445013f82e"

	// imageBinDir holds the binaries inside the image.
	imageBinDir = "riptide"

	// hostBinDir is where the init container installs the binaries. GKE
	// mounts /home/kubernetes/bin exec, while /var is noexec on COS.
	hostBinDir = "/home/kubernetes/bin/ate-riptide"

	snapshotterName = "containerd-gcfs-grpc"
)

// binary is a file to add to the image under imageBinDir.
type binary struct {
	name string // file name in the image
	path string // local source path
}

func main() {
	var (
		gcfsd       = flag.String("gcfsd", "", "Path to the gcfsd-v2 binary (gcfsd with --v1)")
		snapshotter = flag.String("snapshotter", "", "Path to the containerd-gcfs-grpc binary")
		repo        = flag.String("repo", "", "Image reference to push to, e.g. REGION-docker.pkg.dev/PROJECT/REPO/riptide-bins:TAG")
		gcfsdV1     = flag.Bool("v1", false, "Install gcfsd as gcfsd instead of gcfsd-v2, for ENABLE_GCFS_V2_API=false")
		patchOut    = flag.String("patch-out", "", "If set, write the riptide DaemonSet patch to this file after the push")
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ref, err := run(ctx, *gcfsd, *snapshotter, *repo, *gcfsdV1, *patchOut)
	if err != nil {
		slog.Error("riptide-bin-image failed", slog.Any("err", err))
		os.Exit(1)
	}
	fmt.Println(ref)
}

func run(ctx context.Context, gcfsd, snapshotter, repo string, gcfsdV1 bool, patchOut string) (string, error) {
	if gcfsd == "" || snapshotter == "" || repo == "" {
		return "", errors.New("--gcfsd, --snapshotter and --repo are required")
	}
	gcfsdName := "gcfsd-v2"
	if gcfsdV1 {
		gcfsdName = "gcfsd"
	}
	bins := []binary{
		{name: gcfsdName, path: gcfsd},
		{name: snapshotterName, path: snapshotter},
	}

	arch, err := binariesArch(bins)
	if err != nil {
		return "", err
	}
	tag, err := name.ParseReference(repo)
	if err != nil {
		return "", fmt.Errorf("parsing --repo: %w", err)
	}
	base, err := name.ParseReference(baseImage)
	if err != nil {
		return "", err
	}

	kc := authn.NewMultiKeychain(authn.DefaultKeychain, google.Keychain)
	baseImg, err := remote.Image(base,
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(kc),
		remote.WithPlatform(v1.Platform{OS: "linux", Architecture: arch}),
	)
	if err != nil {
		return "", fmt.Errorf("fetching base image %s: %w", baseImage, err)
	}
	layer, err := binariesLayer(bins)
	if err != nil {
		return "", err
	}
	img, err := mutate.AppendLayers(baseImg, layer)
	if err != nil {
		return "", err
	}
	if err := remote.Write(tag, img, remote.WithContext(ctx), remote.WithAuthFromKeychain(kc)); err != nil {
		return "", fmt.Errorf("pushing %s: %w", tag, err)
	}
	digest, err := img.Digest()
	if err != nil {
		return "", err
	}
	pinned := tag.Context().Digest(digest.String()).String()

	if patchOut != "" {
		patch, err := daemonSetPatch(pinned)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(patchOut, patch, 0o644); err != nil {
			return "", err
		}
	}
	return pinned, nil
}

// binariesArch returns the GOARCH of the binaries and fails unless they are
// all Linux executables for the same architecture.
func binariesArch(bins []binary) (string, error) {
	var arch string
	for _, b := range bins {
		a, err := elfArch(b.path)
		if err != nil {
			return "", err
		}
		if arch != "" && a != arch {
			return "", fmt.Errorf("%s is %s but %s is %s", bins[0].path, arch, b.path, a)
		}
		arch = a
	}
	return arch, nil
}

func elfArch(p string) (string, error) {
	f, err := elf.Open(p)
	if err != nil {
		return "", fmt.Errorf("%s is not an ELF binary: %w", p, err)
	}
	defer func() { _ = f.Close() }()
	if f.Type != elf.ET_EXEC && f.Type != elf.ET_DYN {
		return "", fmt.Errorf("%s is not an executable (ELF type %s)", p, f.Type)
	}
	switch f.Machine {
	case elf.EM_X86_64:
		return "amd64", nil
	case elf.EM_AARCH64:
		return "arm64", nil
	default:
		return "", fmt.Errorf("%s has unsupported architecture %s", p, f.Machine)
	}
}

// binariesLayer returns a layer with each binary at /riptide/<name>, mode
// 0755 and a fixed mtime, so the same inputs give the same digest.
func binariesLayer(bins []binary) (v1.Layer, error) {
	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		pr, pw := io.Pipe()
		go func() { pw.CloseWithError(writeTar(pw, bins)) }()
		return pr, nil
	})
}

func writeTar(w io.Writer, bins []binary) error {
	tw := tar.NewWriter(w)
	epoch := time.Unix(0, 0)
	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeDir,
		Name:     imageBinDir + "/",
		Mode:     0o755,
		ModTime:  epoch,
	}); err != nil {
		return err
	}
	for _, b := range bins {
		if err := writeTarFile(tw, path.Join(imageBinDir, b.name), b.path, epoch); err != nil {
			return err
		}
	}
	return tw.Close()
}

func writeTarFile(tw *tar.Writer, name, src string, mtime time.Time) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Mode:     0o755,
		Size:     fi.Size(),
		ModTime:  mtime,
	}); err != nil {
		return err
	}
	_, err = io.Copy(tw, f)
	return err
}

// installScript copies the binaries into the host directory. Writing to a
// temporary name and renaming avoids "text file busy" if an old process
// still runs the previous file.
const installScript = `dst=/host-bin/ate-riptide
mkdir -p "${dst}"
for f in /riptide/*; do
  n="$(basename "${f}")"
  cp "${f}" "${dst}/.${n}.tmp"
  chmod 0755 "${dst}/.${n}.tmp"
  mv -f "${dst}/.${n}.tmp" "${dst}/${n}"
done
`

// daemonSetPatch returns a strategic merge patch for the
// ate-accelerator-riptide DaemonSet that installs the binaries from image and
// points riptide-provider at them.
func daemonSetPatch(image string) ([]byte, error) {
	hostPathDir := corev1.HostPathDirectory
	podSpec := map[string]any{
		"initContainers": []map[string]any{{
			"name":    "install-riptide-bins",
			"image":   image,
			"command": []string{"/bin/sh", "-ec", installScript},
			"volumeMounts": []corev1.VolumeMount{{
				Name:      "kube-home-bin",
				MountPath: "/host-bin",
			}},
		}},
		"containers": []map[string]any{{
			"name": "riptide-provider",
			"env": []corev1.EnvVar{{
				Name:  "RIPTIDE_BIN_DIR",
				Value: hostBinDir,
			}},
		}},
		"volumes": []corev1.Volume{{
			Name: "kube-home-bin",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: path.Dir(hostBinDir),
					Type: &hostPathDir,
				},
			},
		}},
	}
	return yaml.Marshal(map[string]any{
		"spec": map[string]any{"template": map[string]any{"spec": podSpec}},
	})
}
