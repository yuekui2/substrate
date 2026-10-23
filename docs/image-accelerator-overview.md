# Overview: Extensible Image Streaming API for Agent Substrate

* **Author:** Kui Yue
* **Status:** Ready for review
* **Last update:** Oct 15, 2026
* **Go link:** [go/image-streaming-api-for-substrate](https://goto.google.com/image-streaming-api-for-substrate)

*This document now serves strictly as an overview; detailed design tabs and comments have been relocated to [`docs/image-accelerator-design.md`](image-accelerator-design.md).*

## Problem

By streaming OCI image layers lazily over FUSE instead of downloading full layer tarballs upfront, `atelet` speeds up preparing actor root filesystems by 20+X while keeping Kubernetes and `containerd` CRI out of the actor critical path.

## Terminology

* **Agent Substrate & Actor (`atelet` / `ateworker`):** Substrate multiplexes many mostly-idle agent workloads (**Actors**, analogous to lightweight K8s Pods/containers) onto pre-warmed K8s worker Pods without going through the K8s control plane or `containerd` CRI. On each node, `atelet` plays the role of `kubelet` + `containerd` image manager (resolving credentials and preparing OCI layers), while `ateworker` plays the role of `containerd-shim` + `runsc` (assembling the `overlayfs` rootfs and running/restoring the Actor in gVisor).
* **Provider:** The underlying image-streaming engine that FUSE-mounts OCI layers from a registry (e.g., Google Riptide, AWS SOCI, eStargz, Nydus).
* **Adapter/Driver/Accelerator:** A lightweight DaemonSet plugin that serves as a bridge between Substrate and a specific streaming provider. It implements the `ImageAccelerator` gRPC API and talks to the specific streaming provider.

## Immediate-Term Proposal

**Goal:** Define Substrate's own `ImageAccelerator` gRPC API and ship functional end-to-end image streaming on GKE via a Riptide v2 Adapter DaemonSet.

### Core Substrate `ImageAccelerator` gRPC API

Because CNCF's low-level `containerd.services.snapshots.v1.Snapshots` API lacks a standard credential-passing mechanism (`imagePullSecrets`), exposes a stateful `Active` to `Committed` snapshot-tree dance that `atelet` does not need (since `ateworker` mounts the writable `overlayfs` `upperdir`), and exhibits provider-specific quirks across Riptide, SOCI, eStargz, and Nydus, Substrate defines its own higher-level `ImageAccelerator` gRPC service over a single Unix socket:

* `CanProvide(ctx, req) (*CanProvideResponse, error)` — called by `atelet` (both before preparing a container's rootfs and via periodic node health checks) to check whether the provider can serve the given OCI image (`can_provide: true/false`, e.g., GAR eligibility check for Riptide) and whether the adapter is in lame-duck drain mode (`evicting: true/false`).
* `PrepareImage(ctx, req) (*PrepareImageResponse, error)` — called by `atelet` on actor start (`CreateActor` / `Run`) and actor wakeup (`RestoreActor` / `Restore`) before `ateworker` mounts `overlayfs`. Accepts the OCI image reference and resolved registry credentials (`imagePullSecrets`), acquires an active lease on the image for that actor sandbox, and returns the complete ordered list of read-only host `LayerDirs` (which `atelet` writes into the sandbox's `rootfs-overlay.json`).
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
* **DaemonSet Isolation:** The `riptide-provider` container runs an isolated `gcfsd` + Riptide Snapshotter pair (using the preloaded binaries) inside a child mount namespace (`unshare -m --propagation unchanged`), marks `/run` as `rslave`, bind-mounts `/run/ate/gcfsd` onto `/run/gcfsd` (satisfying `gcfsd-v2`'s hardcoded `/run/gcfsd/keychain.sock` path without touching the host's `/run/gcfsd`), and mounts FUSE at `/var/lib/ate/accelerator-image-cache/riptide-gcfs/mnt` (`Bidirectional` propagation). This works identically on both GKE Nodes with Streaming Disabled and GKE Nodes with Streaming Enabled with zero interference.

### What Changes Across Components

* **`atelet`:** Automatically detects the local `ImageAccelerator` Unix socket endpoint (`/run/ate/image-accelerator.sock`, configurable via `--image-accelerator-socket`) and uses it whenever present and ready, falling back to local `image-cache` when absent. When applicable, resolves registry credentials via the Kubelet Credential Provider or workload `imagePullSecrets` and passes them inside `PrepareImageRequest`.
* **`ateworker`:** No protocol changes; mounts the `LayerDirs` listed in `rootfs-overlay.json` as read-only `overlayfs` `lowerdir` entries identically to locally unpacked layers.

### Separate On-Disk Folders & Dynamic Disabling

* **Strict Directory Separation:** Locally unpacked layers, adapter streaming wrapper symlinks, and provider FUSE mounts / chunk caches live in completely separate directories under `/var/lib/ate`. See Appendix: [On-Disk Folder Structure](#on-disk-folder-structure) for details.
* **Avoiding Streaming Layers on Disable:** Unlike GKE `containerd` (where disabling Riptide leaves dangling `gcfs` snapshot metadata tied to image records), when streaming is dynamically disabled on a Substrate node:
  * **New & Resuming Actors:** `atelet` immediately stops routing new actors (`Run`) and resuming actors (`Restore`) to `/var/lib/ate/accelerator-image-cache/...` and instead pulls/unpacks layers into `/var/lib/ate/image-cache/layers/...`. Even an actor that was previously streaming seamlessly restores its checkpoint `upperdir` onto local `image-cache` layers on its next wakeup.
  * **Releasing & GC-ing Streamed Layers:** Active streamed actors are evicted via `SIGTERM` to cycle through suspend/resume (`SuspendActor` -> `ResumeActor`); as each one checkpoints or terminates, its `refCount` decrements and once `refCount` reaches 0, the `/var/lib/ate/accelerator-image-cache/riptide/...` wrapper directories and `/var/lib/ate/accelerator-image-cache/riptide-gcfs` FUSE views are immediately unmounted and GC-ed.
* **Deployment Workflow:**
  * **Auto-Detection in `atelet`:** `atelet` automatically detects `/run/ate/image-accelerator.sock` on the node and uses it as soon as an adapter binds the socket, without restarting `atelet`. Substrate has zero knowledge of any adapter DaemonSet or provider; when no adapter socket is present on a node, `atelet` uses local `image-cache`.
  * **Separate Adapter Installation:** Installing an `ImageAccelerator` adapter is a single step outside Substrate's installer—apply the adapter DaemonSet manifest (`kubectl apply -f <adapter-manifest>.yaml`). The DaemonSet schedules onto all worker nodes by default, with no node labeling required to enable streaming.
  * **Disable on a Node (Opt-Out Label):** Labeling a node `kubectl label node <node> ate.dev/image-accelerator=disabled --overwrite` does **not** terminate the adapter DaemonSet pod on `<node>`; instead, `atelet` watches the node label and marks the adapter on `<node>` as unused (`SetDraining(true)`). `atelet` immediately falls back to local `image-cache` for new and resuming actors and triggers `SIGTERM` eviction on active `streamed: true` actors on `<node>` so they cycle through suspend/resume (`SuspendActor` -> `ResumeActor`) and release their leases (`ReleaseImage` with `bypass_ttl: true`) until `refCount` reaches 0 and all FUSE views are unmounted, while the adapter DaemonSet pod remains running idle. Removing the label (`kubectl label node <node> ate.dev/image-accelerator-`) re-enables the adapter on `<node>` immediately without restarting the DaemonSet pod.

### Independent Release

`oci-image-accelerator` is versioned and released on its own cadence, completely decoupled from `ate-setup` and Substrate's core release cycle:

* **Adapter Binary & Container Image Release:**
  * Each release tag triggers a standalone build and publish a multi-arch distroless OCI container image.
* **DaemonSet YAML Manifest Release:**
  * Release uploads the rendered `<provider>.yaml` as a GitHub Release artifact, allowing operators to install or upgrade with a single `kubectl apply -f https://github.com/.../oci-image-accelerator%2FvX.Y.Z/<provider>.yaml`.

### Upgrading `atelet` and the Adapter DaemonSet

Stopping a userspace FUSE daemon while an actor's `overlayfs` `lowerdir` is reading from `/dev/fuse` causes reads to fail with `ENOTCONN`. Because the adapter runs in its own DaemonSet (`ate-accelerator-<provider>`) outside the Substrate release upgrade ([`docs/version-skew-policy.md`](version-skew-policy.md)):
* **Substrate Upgrades ([`docs/upgrade.md`](upgrade.md) Step 2):** `ds/atelet` rolls in-place without restarting `ate-accelerator-<provider>`; active FUSE mounts stay untouched and the new `atelet` reconnects via `ReconcileImages`.
* **Adapter DaemonSet Upgrades:** FUSE-backed adapter DaemonSets use `updateStrategy.type: RollingUpdate` (`maxUnavailable: 10%`, `terminationGracePeriodSeconds: 3600`) with `SIGTERM` lame-duck self-draining (`--drain-timeout=30m`), `/run/ate/drain-complete` `preStop` synchronization, and active actor `SIGTERM` eviction cycling (`SuspendActor` -> `ResumeActor`) that reuses Substrate's existing worker upgrade eviction pipeline. See Appendix: [Adapter DaemonSet Upgrade](#adapter-daemonset-upgrade) for details.

## Medium-Term Proposal

* Optimize cold/warm latency after delivering functional Riptide v2
* Extend to other OSS remote snapshotters
* Dynamically disabling streaming

## Appendix

### Architecture Diagram

[http://shortn/_xiCJJfZBGr](http://shortn/_xiCJJfZBGr)

### Host `containerd` GC Interferences

`containerd` GC interferes with Substrate when it shares on-host Riptide Snapshotter and `gcfsd`.

`containerd` and Riptide Snapshotter maintain two completely separate BoltDB databases (`containerd`'s `/var/lib/containerd/.../meta.db` vs. Riptide's internal `/var/lib/containerd-gcfs-grpc/metadata.db`). Riptide never touches `containerd`'s BoltDB. However, `containerd` assumes it is the sole client of its configured `[proxy_plugins.gcfs]` gRPC socket. During `containerd`'s periodic GC pass, `containerd` calls `Snapshots.Walk()` over gRPC on Riptide Snapshotter's sock to list every snapshot stored in Riptide's BoltDB, compares that list against `containerd`'s own `meta.db`, and calls `Snapshots.Remove(key)` over gRPC on any snapshot key in Riptide's BoltDB that `containerd` itself did not create (i.e., Substrate's layers, since Riptide's internal `snapshots/storage` BoltDB bucket is not partitioned by `containerd-namespace`).

Additionally, in Riptide v2, Riptide Snapshotter delegates `CreateView` and `DeleteView` over `/run/gcfsd/grpc.sock` using the unscoped layer `chainID` hex (`views/<chainID>/fs`) as the view key; sharing the host's `gcfsd` causes `CreateView` collisions (`view "<chainID>" already exists`) for layers already mounted by host `containerd` and causes host `containerd` GC (`DeleteView(<chainID>)`) to unmount `views/<chainID>/fs` out from under Substrate.

### Adapter DaemonSet Upgrade

`atelet` and `ate-accelerator-<provider>` have decoupled upgrade lifecycles:

1. **Upgrading Agent Substrate (`deploy atelet`):** Like CSI drivers, `ate-accelerator-<provider>` is outside the Substrate system upgrade. When `ds/atelet` rolls in-place, the `ate-accelerator-<provider>` pod and its FUSE mounts stay running uninterrupted, and the restarted `atelet` recovers active leases via `ReconcileImages`. The `substrate.imageaccel.v1.ImageAccelerator` gRPC protocol evolves via additive optional protobuf fields so `atelet` and the adapter can upgrade independently.
2. **Upgrading a FUSE-Backed Adapter DaemonSet (`RollingUpdate` + `SIGTERM` Lame-Duck Drain + Actor Eviction Cycling):**
   * **Step 1 — `SIGTERM` Initiates Lame-Duck Mode (10% of Nodes per Batch):** Applying an updated `oci-image-accelerator/manifests/<provider>.yaml` triggers a standard Kubernetes `RollingUpdate` (`maxUnavailable: 10%`, `terminationGracePeriodSeconds: 3600`). When `kubelet` begins terminating an adapter pod on `<node>`, the adapter container receives `SIGTERM` and immediately enters lame-duck mode (`driver.Drain`): `CanProvide` returns `can_provide=false, evicting=true` (failing `--probe` readiness checks), `PrepareImage` returns `ErrNotSupported` (`codes.FailedPrecondition`) so `atelet` falls back to `/var/lib/ate/image-cache` for all new and resuming actors on `<node>`, and idle cached leases (`refCount == 0`) are unmounted immediately.
   * **Step 2 — Triggering Active Streamed Actors to Cycle (`SuspendActor` -> `ResumeActor`):** Unlike a `WorkerPool` upgrade (where Kubernetes sends `SIGTERM` directly to the `ateworker` pod), rolling `ate-accelerator-<provider>` only sends `SIGTERM` to the adapter pod. To cycle active streamed actors without restarting `ateworker` pods or disturbing non-streamed actors, `atelet` detects `evicting=true` from `CanProvide` and calls `ateworker` (`EvictWorkload`) for each local actor whose `rootfs-overlay.json` has `streamed: true`. This reuses `ateworker`'s existing per-actor worker-upgrade eviction logic (`killContainer` on gVisor / `gracefullyStopActor` on microVM):
     1. `ateworker` sends `SIGTERM` into the actor's main container process and starts the 30-minute eviction deadline (`workloadGracePeriod`), while keeping the `ateworker` pod itself online.
     2. The actor's `SIGTERM` handler notifies its harness, which calls `SuspendActor` (`DURABILITY_MODE_SYNCHRONOUS`) and `ResumeActor`.
     3. On `SuspendActor`, `atelet` checkpoints the actor and calls `ReleaseImage` on the draining adapter (bypassing `idleTTL` when `refCount` reaches `0`). On `ResumeActor`, the actor restores either on another node or on the same node using local `/var/lib/ate/image-cache` layers.
   * **Step 3 — Active FUSE Leases Reach Zero (`preStop` Synchronization):** The provider container runs a `lifecycle.preStop` hook that blocks `kubelet` from sending `SIGTERM` to the FUSE daemon until all active leases drain (`refCount == 0` or `--drain-timeout=30m` worst-case fallback, matching `ateworker`'s 30-minute `workloadGracePeriod`) and the adapter writes `/run/ate/drain-complete`.
   * **Step 4 — Replacement Pod Starts & Verifies Provider Readiness:** Once `/run/ate/drain-complete` is written, both containers exit cleanly and Kubernetes starts the replacement adapter pod on `<node>`. On startup, `ServeUnixSocket` waits for `driver.WaitReady(ctx)` (`--startup-timeout=2m`) to verify the underlying provider (snapshotter `Snapshots.Stat` RPC + provider-specific health hook) before binding `/run/ate/image-accelerator.sock`, and the container's `--probe` `readinessProbe` gates Kubernetes `Ready` status before the `RollingUpdate` advances to the next 10% batch of nodes.
3. **Stateless Non-FUSE Providers (`ate-accelerator-spegel`):** Because Spegel unpacks standard files onto the host filesystem rather than holding open `/dev/fuse` mounts, `ate-accelerator-spegel` uses `updateStrategy.type: RollingUpdate` (`terminationGracePeriodSeconds: 5`) and rolls in-place immediately.

### On-Disk Folder Structure

```
/var/lib/ate/
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
├── accelerator-image-cache/             <--- 2. ADAPTER WRAPPERS & PROVIDER ROOTS (agent:v1)
│   ├── version                          (Layout version marker of accelerator)
│   ├── riptide/
│   │   └── sha256-abcd1234/
│   │       ├── lease.json               (Persisted lease state for ReconcileImages)
│   │       ├── layer-0/
│   │       │   ├── fs -> /var/lib/ate/accelerator-image-cache/riptide-gcfs/snapshotter/snapshots/101/fs (Symlink to FUSE)
│   │       │   └── finalized
│   │       ├── layer-1/
│   │       │   ├── fs -> /var/lib/ate/accelerator-image-cache/riptide-gcfs/snapshotter/snapshots/102/fs (Symlink to FUSE)
│   │       │   └── finalized
│   │       └── layer-2/
│   │           ├── fs -> /var/lib/ate/accelerator-image-cache/riptide-gcfs/snapshotter/snapshots/103/fs (Symlink to FUSE)
│   │           └── finalized
│   │
│   └── riptide-gcfs/                    <--- ISOLATED RIPTIDE PROVIDER ROOT
│       ├── mnt/views/<chainID>/fs/      (gcfsd-v2 FUSE virtual layer mounts)
│       ├── snapshotter/
│       │   ├── metadata.db              (containerd-gcfs-grpc BoltDB)
│       │   ├── snapshots/<id>/fs -> ../../mnt/views/<chainID>/fs
│       │   └── layers/                  (On-disk downloaded chunk cache)
│       └── gcfsd/
│           └── images_in_use.db         (gcfsd-v2 chunk cache metadata DB)
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
```

### References

* [https://github.com/agent-substrate/substrate/blob/main/docs/upgrade.md](https://github.com/agent-substrate/substrate/blob/main/docs/upgrade.md)
* [https://github.com/agent-substrate/substrate/blob/main/docs/version-skew-policy.md](https://github.com/agent-substrate/substrate/blob/main/docs/version-skew-policy.md)
