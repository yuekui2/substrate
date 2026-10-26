# OCI Image Accelerator Plugin

`oci-image-accelerator` provides accelerated container image readiness (lazy streaming, P2P layer sharing, and content-addressed FUSE layer views) for high-density container runtimes, including **Agent Substrate** and **Ray**.

By moving container image preparation out of the container runtime critical path and delegating it to an out-of-process node-local daemon, workloads can achieve **sub-second cold starts** for multi-gigabyte container images.

---

## Architecture Overview

```
┌────────────────────────────────────────────────────────┐
│            Workload Orchestrator Node Engine           │
│     (Substrate atelet / Ray Node Worker Process)       │
│                                                        │
│   Calls: CanProvide, PrepareImage, ReleaseImage        │
└───────────────────────────▲────────────────────────────┘
                            │
               gRPC over Unix Domain Socket
               (/run/ate/image-accelerator.sock)
                            │
┌───────────────────────────▼────────────────────────────┐
│      OCI Image Accelerator DaemonSet Pod on Node       │
│                                                        │
│  ┌────────────────────────┐  ┌──────────────────────┐  │
│  │  accelerator-adapter   │  │  streaming-engine    │  │
│  │  (Riptide / Nydus /    │  │  (gcfsd / nydus /    │  │
│  │   SOCI / Spegel)       │  │   soci / dfdaemon)   │  │
│  └───────────┬────────────┘  └──────────┬───────────┘  │
│              │                          │              │
│              └────────────┬─────────────┘              │
│                           │                            │
│                 FUSE Layer Views Mount                 │
│          (/var/lib/ate/accelerator-image-cache)        │
└────────────────────────────────────────────────────────┘
```

### Key Design Principles

1. **Vendor-Agnostic Client Contract**: Consumers (`atelet`, Ray) only speak the uniform `ImageAccelerator` gRPC service. They have zero dependencies on provider-specific APIs, snapshotters, or credentials.
2. **Order-Independent Deployment**: The orchestrator automatically detects when the accelerator socket is present. If the accelerator is missing or declines an image, the runtime falls back gracefully to standard image download.
3. **Isolated from Host `containerd` GC**: The accelerator runs as a dedicated Kubernetes `DaemonSet` with its own isolated mount namespace and storage directories, preventing host `containerd` garbage collection from disrupting active workload layers.
4. **Full Image Ownership**: The accelerator prepares all layers for a requested image under a clean per-image contract. Workloads never need to stitch layers between streaming and standard downloads.

---

## gRPC Protocol Specification (`oci-image-accelerator/proto/v1`)

The service defines four RPCs over a Unix domain socket:

```protobuf
service ImageAccelerator {
  // CanProvide checks whether the accelerator supports serving the image.
  rpc CanProvide(CanProvideRequest) returns (CanProvideResponse);

  // PrepareImage mounts all layers for the image and returns their ordered host paths.
  rpc PrepareImage(PrepareImageRequest) returns (PrepareImageResponse);

  // ReleaseImage decrements the image lease count when a workload stops or checkpoints.
  rpc ReleaseImage(ReleaseImageRequest) returns (ReleaseImageResponse);

  // ReconcileImages synchronizes active leases after an orchestrator restart.
  rpc ReconcileImages(ReconcileImagesRequest) returns (ReconcileImagesResponse);
}
```

---

## Supported Drivers

| Driver (`cmd/`) | Acceleration Strategy | Best Suited For | Target Environments |
| :--- | :--- | :--- | :--- |
| **`cmd/riptide`** | Lazy streaming via Google Riptide v2 (`gcfsd-v2` + `containerd-gcfs-grpc`) | Sub-second cold start on Google Cloud | Google Kubernetes Engine (GKE) |
| **`cmd/soci`** | Lazy streaming via AWS SOCI snapshotter (`soci-snapshotter-grpc`) | AWS ECR & OCI ztoc-indexed images | Amazon EKS / Any K8s cluster |
| **`cmd/nydus-dragonfly`** | P2P accelerated streaming via Nydus RAFS v6 (Kernel EROFS + `fscache`) & Dragonfly v2 | Multi-node cluster distribution without registry throttling | Kubernetes nodes with Kernel EROFS + `fscache` |
| **`cmd/spegel`** | Stateless P2P layer sharing directly between node caches | Standard unconverted OCI images; cluster-local layer re-use | Any Kubernetes cluster |

---

## Manifests

Deployment manifests for all supported drivers are available under `manifests/`:
- `manifests/riptide.yaml` (GKE Riptide v2 DaemonSet)
- `manifests/soci.yaml` (AWS SOCI Snapshotter DaemonSet)
- `manifests/nydus-dragonfly.yaml` (Nydus + Dragonfly P2P DaemonSet)
- `manifests/spegel.yaml` (Spegel P2P DaemonSet)

Deploying a driver (always a separate step from Substrate installation):
```bash
ko resolve -f oci-image-accelerator/manifests/riptide.yaml | kubectl apply -f -
```

To dynamically disable acceleration on a node without terminating the DaemonSet pod (`atelet` falls back to local `image-cache` and evicts active streamed actors):
```bash
kubectl label node <node> ate.dev/image-accelerator=disabled --overwrite
```

To re-enable acceleration on the node:
```bash
kubectl label node <node> ate.dev/image-accelerator-
```

Once ready, the driver publishes the accelerator socket at `/run/ate/image-accelerator.sock`, and connected runtimes begin accelerating container startup immediately.

### Custom Riptide binaries

By default, `riptide.yaml` runs the `gcfsd-v2` and `containerd-gcfs-grpc` that GKE preloads under `/home/kubernetes/bin`. To run your own builds, set `RIPTIDE_BIN_DIR` on the `riptide-provider` container to a host directory that holds both binaries. The pod fails to start if either is missing or not executable. Use `gcfsd` instead of `gcfsd-v2` when `ENABLE_GCFS_V2_API=false`.

To replace only one binary, copy the other from `/home/kubernetes/bin` on a node so the two always ship together.

`cmd/riptide-bin-image` packages both binaries into an image, pushes it, and writes a DaemonSet patch. The patch adds an init container that copies the binaries to `/home/kubernetes/bin/ate-riptide` and sets `RIPTIDE_BIN_DIR` to it. GKE mounts `/home/kubernetes/bin` exec, while `/var` is noexec on COS. Docker is not needed.

```bash
go run ./oci-image-accelerator/cmd/riptide-bin-image \
  --gcfsd=./out/gcfsd-v2 \
  --snapshotter=./out/containerd-gcfs-grpc \
  --repo=REGION-docker.pkg.dev/PROJECT/REPO/riptide-bins:TAG \
  --patch-out=riptide-custom-bins.yaml
kubectl -n ate-system patch ds ate-accelerator-riptide --patch-file riptide-custom-bins.yaml
```

The tool prints the pushed image pinned by digest, and the patch uses that reference. Pass `--v1` when `ENABLE_GCFS_V2_API=false`. Both binaries must be Linux ELF executables for the same architecture as the nodes. The nodes must be able to pull from the repository. For Artifact Registry, grant the node service account `roles/artifactregistry.reader`.

To roll out new binaries, run the tool again and apply the new patch.
