# Overview: Extensible Image Streaming API for Agent Substrate

* **Author:** Kui Yue
* **Status:** Ready for review
* **Last update:** Oct 15, 2026
* **Go link:** [go/image-streaming-api-for-substrate](https://goto.google.com/image-streaming-api-for-substrate)

*This document serves strictly as an overview; detailed design tabs and comments have been relocated to [`docs/image-accelerator-design.md`](image-accelerator-design.md).*

## Problem

By streaming OCI image layers lazily over FUSE instead of downloading full layer tarballs upfront, `atelet` speeds up preparing actor root filesystems by 20+X while keeping Kubernetes and `containerd` CRI out of the actor critical path.

## Terminology

* **Agent Substrate & Actor (`atelet` / `ateom`):** Substrate multiplexes many mostly-idle agent workloads (**Actors**, analogous to lightweight K8s Pods/containers) onto pre-warmed K8s worker Pods without going through the K8s control plane or `containerd` CRI. On each node, `atelet` plays the role of `kubelet` + `containerd` image manager (resolving credentials and preparing OCI layers), while `ateom` plays the role of `containerd-shim` + `runsc` (assembling the `overlayfs` rootfs and running/restoring the Actor in gVisor).
* **Provider:** The underlying image-streaming engine that FUSE-mounts OCI layers from a registry (e.g., Google Riptide, AWS SOCI, eStargz, Nydus).
* **Adapter:** A lightweight DaemonSet plugin that serves as a bridge between Substrate and a specific streaming provider. It implements the `ImageAccelerator` gRPC API and talks to the specific streaming provider.

## Immediate-Term Proposal

**Goal:** Define Substrate's own `ImageAccelerator` gRPC API and ship functional end-to-end image streaming on GKE via a Riptide v2 Adapter DaemonSet.

### Core Substrate `ImageAccelerator` gRPC API

Because CNCF's low-level `containerd.services.snapshots.v1.Snapshots` API lacks a standard credential-passing mechanism (`imagePullSecrets`), exposes a stateful `Active` to `Committed` snapshot-tree dance that `atelet` does not need (since `ateom` mounts the writable `overlayfs` `upperdir`), and exhibits provider-specific quirks across Riptide, SOCI, eStargz, and Nydus, Substrate defines its own higher-level `ImageAccelerator` gRPC service over a single Unix socket:

* `CanProvide(ctx, req) (bool, error)` — called by `atelet` before preparing a container's rootfs to check whether the provider can serve the given OCI image (e.g., GAR eligibility check for Riptide).
* `PrepareImage(ctx, req) (*PrepareImageResponse, error)` — called by `atelet` on actor start (`CreateActor` / `Run`) and actor wakeup (`RestoreActor` / `Restore`) before `ateom` mounts `overlayfs`. Accepts the OCI image reference and resolved registry credentials (`imagePullSecrets`), acquires an active lease on the image for that actor sandbox, and returns the complete ordered list of read-only host `LayerDirs` (which `atelet` writes into the sandbox's `rootfs-overlay.json`).
  * **Per-Image Contract:** Strictly per-image—this choice gives a clean ownership boundary between `atelet` and the various OSS/cloud providers (either `ImageAccelerator` returns all layers for the image, or returns `ErrNotSupported` so `atelet` falls back to `imagecache` for the whole image), which also keeps dynamic disabling, fallback, and local image GC simple. Because each adapter driver has full knowledge of its underlying provider's optimizations (e.g., Riptide and AWS SOCI skipping FUSE for small layers, Nydus RAFS v6 bootstrap, or Spegel shared `diffID` pools), the driver handles hybrid layer backing internally and returns the complete image.
* `ReleaseImage(ctx, req) error` — called by `atelet` whenever an actor sandbox releases its rootfs: on actor teardown (`DeleteActor`), on actor suspend/checkpoint (`Checkpoint`), or on creation/restore rollback if a subsequent sandbox setup step fails. Releases one active lease on the image.
* `ReconcileImages(ctx, active) error` — called once by `atelet` on startup/restart with the list of active streamed images discovered from surviving on-disk actor sandboxes (`sandboxes/*/rootfs-overlay.json`), allowing the service to synchronize active leases and clean up any mounts orphaned during an `atelet` crash.

### Out-of-Process Adapter DaemonSet Architecture

Following the Adapter pattern—analogous to Kubernetes CSI (Container Storage Interface) drivers—each provider implements Substrate's uniform `ImageAccelerator` gRPC API as a standalone Go adapter deployed via a Kubernetes DaemonSet on worker nodes, translating Substrate's RPCs into the provider's native snapshotter protocol and auth sockets while keeping `atelet` 100% vendor-agnostic.

* **Adapter Lease & Reference-Counting Implementation:** Internally, the adapter implements the `PrepareImage` / `ReleaseImage` / `ReconcileImages` lease contract via per-image/per-layer reference counting (`refCount`) and an optional `idleTTL` grace timer, reusing existing mounts on warm hits and unmounting FUSE layer views when `refCount` reaches 0 (while preserving shared layers still referenced by other active images/actors).
* **Repository Strategy:** To unblock v1, the Riptide adapter binary will initially live in a dedicated subfolder of the `substrate` repository (`oci-image-accelerator/`), structured as a standalone component so it can move to an external plugins repository long-term.
* **Chunk Cache & Cross-Image Layer Deduplication:** The underlying provider (`gcfsd`) owns the node-level on-disk chunk cache; the adapter keys FUSE layer views by content-addressed `chainID`/`diffID` so shared layers across different OCI images reuse the same FUSE mount and chunk cache without streaming twice.

### Isolating from Host `containerd` GC via the DaemonSet

* When sharing the host Riptide Snapshotter and `gcfsd` daemon with Substrate, `containerd` GC interferes with both via gRPC calls though they do not share any underlying DB. See Appendix: [Host containerd GC Interferences](#host-containerd-gc-interferences) for detailed explanation.
* **DaemonSet Isolation:** The `riptide-provider` container runs an isolated `gcfsd` + Riptide Snapshotter pair (using the preloaded binaries) inside a child mount namespace (`unshare -m --propagation unchanged`), marks `/run` as `rslave`, bind-mounts `/run/ate-image-streaming/gcfsd` onto `/run/gcfsd` (satisfying `gcfsd-v2`'s hardcoded `/run/gcfsd/keychain.sock` path without touching the host's `/run/gcfsd`), and mounts FUSE at `/var/lib/substrate/gcfs/mnt` (`Bidirectional` propagation). This works identically on both GKE Nodes with Streaming Disabled and GKE Nodes with Streaming Enabled with zero interference.

### What Changes Across Components

* **`atelet`:** Configured via a single flag: `--image-accelerator-socket=<unix-socket-path>` (empty by default = disabled; non-empty = enabled). When applicable, resolves registry credentials via the Kubelet Credential Provider or workload `imagePullSecrets` and passes them inside `PrepareImageRequest`.
* **`ateom`:** No protocol changes; mounts the `LayerDirs` listed in `rootfs-overlay.json` as read-only `overlayfs` `lowerdir` entries identically to locally unpacked layers.

### Separate On-Disk Folders & Dynamic Disabling

* **Strict Directory Separation:** Locally unpacked layers, adapter streaming wrapper symlinks, and provider FUSE mounts / chunk caches live in completely separate directories on disk. See Appendix: [On-Disk Folder Structure](#on-disk-folder-structure) for details.
* **Avoiding Streaming Layers on Disable:** Unlike GKE `containerd` (where disabling Riptide leaves dangling `gcfs` snapshot metadata tied to image records), when streaming is dynamically disabled on a Substrate node:
  * **New & Resuming Actors:** `atelet` immediately stops routing new actors (`Run`) and resuming actors (`Restore`) to `/var/lib/ateom-gvisor/streaming/...` and instead pulls/unpacks layers into `/var/lib/ateom-gvisor/image-cache/layers/...`. Even an actor that was previously streaming seamlessly restores its checkpoint `upperdir` onto local `image-cache` layers on its next wakeup.
  * **Releasing & GC-ing Streamed Layers:** Already-running streamed actors continue uninterrupted; as each one checkpoints or terminates, its `refCount` decrements and once `refCount` reaches 0, the `/var/lib/ateom-gvisor/streaming/...` wrapper directories and `/var/lib/substrate/gcfs` FUSE views are immediately unmounted and GC-ed.
* **Deployment Workflow:**
  * **Cluster Install (Disabled by Default, or Auto-Enabled with Flag):** `./hack/install-ate.sh deploy ate-system` deploys `atelet` without installing any adapter DaemonSet; adding `--setup-image-accelerator=riptide` (or `ATE_IMAGE_ACCELERATOR_MANIFEST=<path-or-url>` for out-of-tree adapters) installs the adapter DaemonSet, labels the nodes `ate.dev/image-accelerator=riptide`, and automatically enables streaming across the cluster.
  * **Disable on a Node:** `kubectl label node <node> ate.dev/image-accelerator=disabled --overwrite` keeps the adapter pod alive for currently running actors while `atelet` stops using streaming for new and resuming actors; once existing actors checkpoint or terminate, `refCount` reaches 0 and the streamed layers are GC-ed (re-enable anytime with `ate.dev/image-accelerator=riptide`).

### Upgrading the Adapter DaemonSet (Extending `atelet` Rolling Upgrades)

Stopping a userspace FUSE daemon while an actor's `overlayfs` `lowerdir` is reading from `/dev/fuse` causes reads to fail with `ENOTCONN`. The solution is to treat adapter upgrade as an `atelet` data plane upgrade, where the Adapter DaemonSet follows `atelet`'s existing version-pinned rolling upgrade process. See Appendix: [Adapter DaemonSet Upgrade](#adapter-daemonset-upgrade) for details.

## Medium-Term Proposal

* Optimize cold/warm latency after delivering functional Riptide v2
* Extend to other OSS remote snapshotters
* Dynamically disabling streaming

## Appendix

### Architecture Diagram

[http://shortn/_Q7pJPaKWZ9](http://shortn/_Q7pJPaKWZ9)

### Host `containerd` GC Interferences

`containerd` GC interferes with Substrate when it shares the on-host Riptide Snapshotter and `gcfsd`.

`containerd` and Riptide Snapshotter maintain two completely separate BoltDB databases (`containerd`'s `/var/lib/containerd/.../meta.db` vs. Riptide's internal `/var/lib/containerd-gcfs-grpc/metadata.db`). Riptide never touches `containerd`'s BoltDB. However, `containerd` assumes it is the sole client of its configured `[proxy_plugins.gcfs]` gRPC socket. During `containerd`'s periodic GC pass, `containerd` calls `Snapshots.Walk()` over gRPC on Riptide Snapshotter's sock to list every snapshot stored in Riptide's BoltDB, compares that list against `containerd`'s own `meta.db`, and calls `Snapshots.Remove(key)` over gRPC on any snapshot key in Riptide's BoltDB that `containerd` itself did not create (i.e., Substrate's layers, since Riptide's internal `snapshots/storage` BoltDB bucket is not partitioned by `containerd-namespace`).

Additionally, in Riptide v2, Riptide Snapshotter delegates `CreateView` and `DeleteView` over `/run/gcfsd/grpc.sock` using the unscoped layer `chainID` hex (`views/<chainID>/fs`) as the view key; sharing the host's `gcfsd` causes `CreateView` collisions (`view "<chainID>" already exists`) for layers already mounted by host `containerd` and causes host `containerd` GC (`DeleteView(<chainID>)`) to unmount `views/<chainID>/fs` out from under Substrate.

### Adapter DaemonSet Upgrade

**Treating Adapter Upgrade as an `atelet` Dataplane Upgrade (`docs/upgrade.md`):** Rather than introducing a separate upgrade mechanism, the Adapter DaemonSet follows `atelet`'s existing version-pinned rolling upgrade convention (`ate-accelerator-<provider>-${SUBSTRATE_VERSION_SUFFIX}`), matching both `ate.dev/image-accelerator: <provider>` and `ate.dev/substrate-version: "${SUBSTRATE_VERSION}"`:

1. **Side-by-Side Deployment:** Deploy `ate-accelerator-<provider>-${NEW_SUFFIX}` alongside `atelet-${NEW_SUFFIX}` pinned to `$NEW_VERSION`. Both sit at 0 pods while nodes still carry `$OLD_VERSION`.
2. **Per-Node Worker Drain & Actor Suspend:** Following `docs/upgrade.md` Step 6a–6c, drain workers on the node and suspend all running actors. Because suspending an actor triggers `Checkpoint` to `ReleaseImage`, zero running actors and zero active FUSE mounts remain on the node once the drain completes.
3. **Shut Down Old Daemons, Then Cut Over to New Version:** Once the node has zero actors, removing the version label (`kubectl label node $NODE ate.dev/substrate-version-`) causes Kubernetes to terminate both the old `atelet` pod and the old `ate-accelerator-<provider>` pod on the node. When the old adapter pod exits, its `image-accelerator.sock`, BoltDB locks (`metadata.db` / `images_in_use.db`), and `/dev/fuse` mounts are all cleanly closed and released. Labeling the node with the new version (`kubectl label node $NODE ate.dev/substrate-version=$NEW_VERSION`) then starts the new `atelet` and new adapter pod on a clean node with zero contention. Rollback and retirement work identically.
4. **Crash Isolation Between Upgrades:** Keeping the adapter in its own DaemonSet (rather than inside the `atelet` Pod) preserves `atelet`'s zero-capability security posture and ensures that if `atelet` crashes and restarts in-place between upgrades, the FUSE daemon and active `/dev/fuse` mounts stay alive uninterrupted while `atelet` recovers state via `ReconcileImages`.

### On-Disk Folder Structure

```
/var/lib/ateom-gvisor/
│
├── image-cache/                         <--- 1. LOCAL UNPACK POOL (pause:3.9)
│   ├── version                          (Layout version marker)
│   ├── manifests/
│   │   └── sha256/5678ef01...json       (Image config + ordered diffIDs)
│   └── layers/sha256/
│       └── 11223344.../                 (Content-addressed by layer diffID)
│           ├── fs/                      <-- REAL physical files on host disk
│           │   └── pause
│           ├── whiteouts.json
│           └── finalized
│
├── streaming/                           <--- 2. ADAPTER STREAMING WRAPPERS (agent:v1)
│   └── riptide/
│       └── sha256-abcd1234/
│           ├── lease.json               (Persisted lease state for ReconcileImages)
│           ├── layer-0/
│           │   ├── fs -> /var/lib/substrate/gcfs/snapshotter/snapshots/101/fs (Symlink to FUSE)
│           │   └── finalized
│           ├── layer-1/
│           │   ├── fs -> /var/lib/substrate/gcfs/snapshotter/snapshots/102/fs (Symlink to FUSE)
│           │   └── finalized
│           └── layer-2/
│               ├── fs -> /var/lib/substrate/gcfs/snapshotter/snapshots/103/fs (Symlink to FUSE)
│               └── finalized
│
└── actors/                              <--- 3. RUNNING ACTOR BUNDLES (Source of Truth)
    ├── actor-pause-001/
    │   └── bundles/pause/
    │       ├── config.json              (OCI container config)
    │       ├── upper/                   (Actor's private writable layer)
    │       ├── work/                    (Overlayfs work directory)
    │       ├── rootfs/                  (Overlayfs mountpoint for runsc)
    │       └── rootfs-overlay.json      <-- OverlaySpec (streamed: false)
    │
    └── actor-agent-002/
        └── bundles/agent/
            ├── config.json
            ├── upper/                   (Actor's private writable layer)
            ├── work/                    (Overlayfs work directory)
            ├── rootfs/                  (Overlayfs mountpoint for runsc)
            └── rootfs-overlay.json      <-- OverlaySpec (streamed: true)

/var/lib/substrate/gcfs/                 <--- 4. ISOLATED RIPTIDE PROVIDER ROOT
├── mnt/views/<chainID>/fs/              (gcfsd-v2 FUSE virtual layer mounts)
├── snapshotter/
│   ├── metadata.db                      (containerd-gcfs-grpc BoltDB)
│   ├── snapshots/<id>/fs -> ../../mnt/views/<chainID>/fs
│   └── layers/                          (On-disk downloaded chunk cache)
└── gcfsd/
    └── images_in_use.db                 (gcfsd-v2 chunk cache metadata DB)
```
