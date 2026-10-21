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
│              (/var/lib/substrate/streaming)            │
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

Deploying a driver:
```bash
ko resolve -f oci-image-accelerator/manifests/riptide.yaml | kubectl apply -f -
kubectl label nodes --all ate.dev/image-accelerator=riptide --overwrite
```
(or via `./hack/install-ate.sh --setup-image-accelerator=riptide`).

Once ready, the driver publishes the accelerator socket at `/run/ate/image-accelerator.sock`, and connected runtimes begin accelerating container startup immediately.
