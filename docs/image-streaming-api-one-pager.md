# One-Pager: Extensible Image Streaming API for Agent Substrate

*This summarizes the key design ideas. See the detailed design for additional details.*

---

## 1. Problem & Impact

Container image download and unpacking is the primary bottleneck for cold-node actor startup and restore (`~28.5s` for a 1.19 GB compressed / ~3.5 GB unpacked image). Because agent workloads touch only 5%–15% of their rootfs at startup (and restored actors load memory pages from Golden Snapshots), replacing upfront layer downloads with lazy FUSE streaming cuts cold-node actor restore (`AteomHerder/Restore`) from **28.5s down to 1.36s** (**21.0x faster** on GKE Riptide v2) and cold actor creation from **27.8s down to 1.15s** (**24.3x faster**), with **zero warm-start penalty** (`~265ms`).

## 2. Phased Strategy & Scope

- **GA Scope (Google Riptide v2 on GKE):** Production-hardened support for **Google Riptide Snapshotter v2** (`containerd-gcfs-grpc --enable-v2`) with Google Artifact Registry (GAR).
- **Multi-Cloud Extensibility (CNCF Standard):** Rather than coupling `cmd/atelet` to vendor-specific APIs, Substrate standardizes on the open **CNCF Remote Snapshotter gRPC API** (`containerd.services.snapshots.v1.Snapshots`). A single generic driver (`internal/imagestreaming/drivers/remotesnapshotter`) powers Riptide v2 at GA and extends to OSS snapshotters such as AWS SOCI (validated in PoC at **7.7x** cold-restore speedup), eStargz, and Nydus post-GA.
- **Shared gRPC Contract, Different Provider Implementation Details:** Although Riptide v2, AWS SOCI, eStargz, and Nydus all implement `Snapshots.v1` and containerd's `cri.*` label protocol, calling them directly without `containerd`'s unpacker surfaces subtle behavioral differences that motivate our phased rollout:
  - **Sub-Layer Decline vs. All-Layer Streaming:** Riptide v2 streams every layer of an imported GAR image. By contrast, AWS SOCI skips indexing layers smaller than `--min-layer-size` (default 10 MiB), declining `Prepare` on small layers and expecting the caller to unpack them locally and call `Snapshots.Commit` (our GA driver works with `--min-layer-size=0` SOCI images; hybrid local unpack + `Commit` is a post-GA driver extension).
  - **Manifest Annotations & `View` Topology:** Riptide v2, SOCI, and eStargz materialize a FUSE mount per layer using standard `cri.*` labels, whereas Nydus requires forwarding OCI layer annotations (`containerd.io/snapshot/nydus-*`) and only mounts a rootfs when `View` is called on the topmost bootstrap layer.
  - **Out-of-Band Credential Side-Channels (`imagePullSecrets`):** Because `Snapshots.v1` has no credential fields (and snapshot labels are persisted unencrypted in BoltDB), each provider uses its own auth side-channel: Riptide v2 exposes `/run/gcfsd/keychain.sock` (`UpdateCreds`, supported at GA), whereas OSS snapshotters rely on ambient node IAM or an unstandardized CRI `ImageService` proxy.
- **Zero-Disruption Fallback:** Streaming is strictly an optimization. If no daemon is present (`--image-streamer=auto`), a daemon is unhealthy, or an image is unstreamable (e.g., non-GAR image on GKE), `atelet` transparently falls back to local image download and untar (`internal/imagecache`).

## 3. Core Architecture: Direct Snapshotter gRPC (Bypassing `containerd` CRI)

To keep Kubernetes and `containerd` CRI out of the actor critical path, `atelet` communicates directly with the host snapshotter's UNIX domain socket:

```
atelet ──► imagestreaming.ImageStreamer (CanStream / PrepareLayers / ReleaseLayers / ReconcileLeases)
             ├─► [Fast Path] Local imagecache hit ──► return unpacked layers immediately
             ├─► [Stream Path] Snapshots.v1 gRPC (Stat / Prepare / View / Remove)
             │     └──► Host Snapshotter Daemon (Riptide v2 at GA; extensible to SOCI/OSS)
             │            └──► FUSE Layer Views ──► Symlinked in /var/lib/ateom-gvisor/streaming/
             │                                        └──► ateom mounts as read-only overlayfs lowerdir
             └─► [Fallback Path] imagecache.EnsureImage (full download & untar)
```

- **Minimal Interface (`ImageStreamer`):** Exposes 4 lifecycle methods (`CanStream`, `PrepareLayers`, `ReleaseLayers`, `ReconcileLeases`) plus a provider registry (`riptide`, `soci`, `auto`, `none`).
- **Unprivileged Layer Wrapper Contract (`atelet` $\leftrightarrow$ `ateom`):** `atelet` runs without `CAP_SYS_ADMIN`. For each layer, `PrepareLayers` creates a wrapper directory (`/var/lib/ateom-gvisor/streaming/<driver>/<image_key>/layer-N/`) containing an `fs` symlink to the snapshotter's read-only FUSE mount and a `finalized` marker. `ateom` consumes these directories as `overlayfs` lowerdirs identically to locally unpacked layers.

## 4. Key Engineering Decisions

| Area | Decision & Rationale |
| :--- | :--- |
| **1. Snapshotter Protocol** | For each layer $0 \dots N-1$ under a per-`chainID` lock, the driver calls `Stat(chainID)` then `Prepare` with `containerd.io/snapshot/cri.*` labels, followed by `View`. Per-`chainID` locking prevents Riptide's `CommitActive` active-snapshot leak on concurrent pulls of shared base layers. If `Prepare` declines a layer (returns writable mounts), the driver cleans up, caches the decline for 10m, and returns `ErrNotStreamable` to trigger fallback. |
| **2. Host `containerd` GC Isolation** | `containerd-gcfs-grpc` uses a flat BoltDB. If host `containerd` shares `[proxy_plugins.gcfs]`, K8s Pod GC deletes `atelet`'s active FUSE snapshots. On Substrate nodes, `atelet` detaches `gcfs` from host `containerd` at startup (switching K8s Pods to `overlayfs`), giving `atelet` exclusive, zero-contention ownership of Riptide and `gcfsd`. |
| **3. Lease RefCounting & Crash Recovery** | Active images are reference-counted across multiplexed actors (`refCount++` on Run/Wake; `ReleaseLayers` on Checkpoint/Sleep/Terminate). Views are unmounted when `refCount == 0` to bound kernel VFS/mount resources; warm re-attachment takes `<2ms`. On `atelet` restart, `ReconcileLeases` scans surviving bundle specs on disk (`rootfs-overlay.json`), restores leases via persisted `lease.json` metadata, and sweeps orphaned views. |
| **4. gVisor Cold-Start Warmup** | Two-stage warmup after `View` creation: **(1) Synchronous `os.ReadDir` probe** waits until the FUSE directory index is listable (preventing cold-start `ENOENT` races), and **(2) Background host `filepath.WalkDir` (`lstat` only)** prefetches inode metadata into the host cache so gVisor (`runsc`) Sentry/Gofer traversals do not stall on remote FUSE round-trips. |
| **5. Two-Layer Auth & `imagePullSecrets`** | **Control plane (`atelet`):** Resolves manifests/configs via the node's Kubelet Credential Provider (`authn.Keychain`) or workload `AuthConfig`. **Data plane (daemon):** Uses VM instance metadata (`169.254.169.254`) for node IAM, plus Riptide v2's `/run/gcfsd/keychain.sock` (`Keychain.UpdateCreds`) to support workload `imagePullSecrets` without touching host `containerd`. |
| **6. Observability** | Emits OpenTelemetry Weaver metrics `ate.imagestreaming.requests` (counter) and `ate.imagestreaming.duration` (histogram) labeled by `provider` and `outcome` (`success`, `fallback`, `error`). |
