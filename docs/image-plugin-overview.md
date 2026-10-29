# Overview: Extensible Image Plugin API for Agent Substrate

**Author:** Kui Yue  
**Status:** Ready for review  
**Last update:** Oct 29, 2026  
**Go link:** [go/image-plugin-api-for-substrate](https://goto.google.com/image-plugin-api-for-substrate)

## Problem

By streaming OCI image layers lazily over FUSE instead of downloading full layer tarballs upfront, `atelet` speeds up preparing actor root filesystems by 20+X while keeping Kubernetes and `containerd` CRI out of the actor critical path.

## Terminology

* **Agent Substrate & Actor (`atelet` / `ateworker`):** Substrate multiplexes many mostly-idle agent workloads (**Actors**, analogous to lightweight **K8s Pods/containers**) onto pre-warmed K8s worker Pods without going through the K8s control plane or `containerd` CRI. On each node, **`atelet`** plays the role of **`kubelet` + `containerd` image manager** (resolving credentials and preparing OCI layers), while **`ateworker`** plays the role of **`containerd-shim` + `runsc`** (assembling the `overlayfs` rootfs and running/restoring the Actor in gVisor).
* **Provider:** The underlying image-pulling engine that FUSE-mounts OCI layers from a registry (e.g., Google Riptide, AWS SOCI, eStargz, Nydus).
* **Image Plugin:** A lightweight DaemonSet that serves as a bridge between Substrate and a specific provider. It implements the `ImagePlugin` gRPC API and talks to that provider.

## Immediate-Term Proposal

**Goal:** Define Substrate's own `ImagePlugin` gRPC API and ship functional end-to-end image streaming on GKE via a Riptide v2 Plugin DaemonSet.

### Core Substrate `ImagePlugin` gRPC API

* There are many challenges with containerd's remote snapshotter API:
  * Lacks a standard credential-passing mechanism (`imagePullSecrets`)
  * Exposes a stateful `Active` to `Committed` snapshot-tree architecture that `atelet` does not need (since `ateworker` mounts the writable `overlayfs` upperdir)
  * Exhibits provider-specific quirks across Riptide, SOCI, eStargz, and Nydus
* Substrate defines its own higher-level `ImagePlugin` gRPC service over a single Unix socket:
  * `CanProvide(CanProvideRequest) returns (CanProvideResponse)` — called by `atelet` before preparing a container's rootfs, and periodically when `atelet` polls for socket existence and the plugin's state. It reports whether the provider can serve the given OCI image and the plugin's lifecycle state (e.g., draining while it prepares to shut down).
  * `PrepareImage(PrepareImageRequest) returns (PrepareImageResponse)` — called by `atelet` on **actor start (`CreateActor` / `Run`) and actor wakeup (`RestoreActor` / `Restore`)** before `ateworker` mounts `overlayfs`. Accepts the OCI image reference and resolved registry credentials, acquires an active lease on the image for that actor sandbox, and returns the complete ordered list of read-only host `layer_dirs` (which `atelet` writes into the sandbox's `rootfs-overlay.json`).
    * *Per-Image Contract:* Strictly per-image—this choice gives a clean ownership boundary between `atelet` and the various OSS/cloud providers (the plugin either returns the full image or returns an error telling `atelet` to fall back to its local pull), which also keeps dynamic disabling, fallback, and local image GC simple. Because each plugin has full knowledge of its underlying provider's optimizations (e.g., Riptide and AWS SOCI skipping FUSE for small layers, Nydus RAFS v6 bootstrap, or Spegel shared `diffID` pools), the plugin handles hybrid layer backing internally and returns the complete image.
  * `ReleaseImage(ReleaseImageRequest) returns (ReleaseImageResponse)` — called by `atelet` whenever an actor sandbox releases its rootfs: on **actor teardown (`DeleteActor`)**, on **actor suspend/checkpoint (`Checkpoint`)**, or on **creation/restore rollback** if a subsequent sandbox setup step fails. Releases one active lease on the image. When `refCount` reaches 0, the image stays cached for the plugin's configured idle TTL.
  * `ReconcileImages(ReconcileImagesRequest) returns (ReconcileImagesResponse)` — called by `atelet` on startup/restart, and whenever the plugin socket appears, with the list of active streamed images recorded in the actors' `rootfs-overlay.json` files on disk, allowing the plugin to synchronize active leases and clean up any mounts orphaned during an `atelet` crash.
  * `LayerForm`: An enum, used by `CanProvide` and `PrepareImage`, for the form of the layers, e.g., EROFS image files. It defaults to read-only host directories of the OCI layers.

### Out-of-Process Plugin DaemonSet Architecture

* Following the **Adapter pattern**—analogous to Kubernetes CSI (Container Storage Interface) drivers—each provider implements Substrate's uniform `ImagePlugin` gRPC API as a standalone plugin deployed via a Kubernetes `DaemonSet` on worker nodes, translating Substrate's RPCs into the provider's native snapshotter protocol and auth sockets while keeping `atelet` 100% vendor-agnostic.
* **Plugin Lease & Reference-Counting Implementation:** Internally, the plugin implements the `PrepareImage` / `ReleaseImage` / `ReconcileImages` lease contract via per-image/per-layer reference counting (`refCount`) and an idle TTL (`--idle-ttl`, default 10 minutes), reusing existing mounts on warm hits and unmounting FUSE layer views when `refCount` reaches `0` (while preserving shared layers still referenced by other active images/actors).
* **Repository Strategy:** Image plugins live in a separate `agent-substrate-plugins` GitHub repository, starting with the Riptide plugin in `agent-substrate-plugins/image-plugin/`. This structure encourages future community contributions for other providers, while the core `substrate` repository retains ownership of the `ImagePlugin` proto and the `atelet` client.
* **Chunk Cache & Cross-Image Layer Deduplication:** The underlying provider (`gcfsd`) owns the node-level on-disk chunk cache; the plugin keys FUSE layer views by content-addressed `chainID`/`diffID` so shared layers across different OCI images reuse the same FUSE mount and chunk cache without streaming twice. The chunk cache also survives provider restarts and plugin upgrades. On start, the provider clears only state that died with the old daemon (FUSE views, snapshotter metadata, and the plugin's per-image directories) and keeps the downloaded chunks. It wipes the chunk cache only when the provider cache version (`riptide-gcfs/version`, a hash of the provider binaries) changes. Riptide and Spegel keep their cache by default. For a cold start, such as a benchmark, set `CLEAR_CACHE_ON_START=true` on the Riptide provider or `--clear-cache-on-start` on the Spegel plugin. SOCI and Nydus/Dragonfly still clear their provider root on every start.

### Isolating from Host `containerd` GC via the DaemonSet

* When sharing the host Riptide Snapshotter and `gcfsd` daemon with Substrate, `containerd` GC interferes with both via gRPC calls though they do not share any underlying DB. See *Appendix: [Host containerd GC Interferences](#host-containerd-gc-interferences)* for detailed explanation.
* **DaemonSet Isolation:** The `riptide-provider` container runs an isolated `gcfsd` + Riptide Snapshotter pair (using the preloaded binaries) inside a child mount namespace, marks `/run` as `rslave`, bind-mounts `/run/ate/gcfsd` onto `/run/gcfsd` (satisfying `gcfsd-v2`'s hardcoded `/run/gcfsd/keychain.sock` path without touching the host's `/run/gcfsd`), and mounts FUSE at `/var/lib/ate/image-plugin-cache/riptide-gcfs/mnt` (`Bidirectional` propagation). This works identically on both GKE Nodes with Streaming Disabled and GKE Nodes with Streaming Enabled with zero interference.

### What Changes Across Components

* **`atelet`:** Automatically detects the local `ImagePlugin` Unix socket endpoint (`/run/ate/image-plugin.sock`, configurable via `--image-plugin-socket`) and uses it whenever present and ready, falling back to local `image-cache` when absent. When applicable, resolves registry credentials via the Kubelet Credential Provider or workload `imagePullSecrets` and passes them inside `PrepareImageRequest`.
* **`ateworker`:** Mounts the `layer_dirs` listed in `rootfs-overlay.json` as read-only `overlayfs` `lowerdir` entries identically to locally unpacked layers.

### Separate On-Disk Folders & Dynamic Disabling

* **Strict Directory Separation:** Locally unpacked layers, plugin wrapper symlinks, and provider FUSE mounts / chunk caches live in completely separate directories under `/var/lib/ate`. See *Appendix: [On-Disk Folder Structure](#on-disk-folder-structure)* for details.
* **Avoiding Plugin-provided Layers on Disable:** Unlike GKE `containerd` (where disabling Riptide leaves dangling `gcfs` snapshot metadata tied to image records), when the plugin is disabled on a Substrate node:
  * **New & Resuming Actors:** `atelet` immediately stops routing new actors (`Run`) and resuming actors (`Restore`) to `/var/lib/ate/image-plugin-cache/...` and instead pulls/unpacks layers into `/var/lib/ate/image-cache/layers/...`. Even an actor that was previously streaming seamlessly restores its checkpoint `upperdir` onto local `image-cache` layers on its next wakeup.
  * **Releasing & GC-ing Streamed Layers:** Active streamed actors are evicted via `SIGTERM` to cycle through suspend/resume (`SuspendActor` -> `ResumeActor`); as each one checkpoints or terminates, its `refCount` decrements and once `refCount` reaches 0, the `/var/lib/ate/image-plugin-cache/riptide/...` wrapper directories and `/var/lib/ate/image-plugin-cache/riptide-gcfs` FUSE views are immediately unmounted and GC-ed.
* **Deployment Workflow:**
  * **Auto-Detection in `atelet`:** `atelet` automatically detects `/run/ate/image-plugin.sock` on the node and uses it as soon as a plugin binds the socket, without restarting `atelet`. Substrate has zero knowledge of any plugin DaemonSet or provider; when no plugin socket is present on a node, `atelet` uses local `image-cache`.
  * **Separate Plugin Installation:** Installing an image plugin is a single step outside Substrate's installer—apply the plugin DaemonSet manifest (`kubectl apply -f <plugin-manifest>.yaml`).
  * **Disable:** Deleting a plugin DaemonSet disables the plugin on every node it runs on. Its pods receive `SIGTERM` and drain as described in *Appendix: [Plugin DaemonSet Upgrade](#plugin-daemonset-upgrade)*: `atelet` falls back to local `image-cache` for new and resuming actors, and cycles active streamed actors until all leases are released. A delete stops the plugin on all of its nodes at once, and all of their streamed actors cycle through suspend and resume.
  * **Scope:** A plugin DaemonSet can cover the whole cluster or, with a `nodeSelector`, specific node pools. That scope is also the granularity at which the plugin can be enabled or disabled.

### Independent Release

Each image plugin is versioned and released on its own cadence, completely decoupled from `ate-setup` and Substrate's core release cycle:

1. **Plugin Binary & Container Image Release:**
   * Each release tag triggers a standalone build and publish a multi-arch distroless OCI container image.
2. **DaemonSet YAML Manifest Release:**
   * Release uploads the rendered `<provider>.yaml` as a GitHub Release artifact, allowing operators to install or upgrade with a single `kubectl apply -f <provider>.yaml`.

### Upgrading `atelet` and the Plugin DaemonSet

Stopping a userspace FUSE daemon while an actor's `overlayfs` `lowerdir` is reading from `/dev/fuse` causes reads to fail with `ENOTCONN`. Because the plugin runs in its own DaemonSet (`image-plugin-<provider>`) outside the Substrate release upgrade ([`docs/version-skew-policy.md`](version-skew-policy.md)):

* **Substrate Upgrades:** `ds/atelet` rolls in-place without restarting `image-plugin-<provider>`; active FUSE mounts stay untouched and the new `atelet` reconnects via `ReconcileImages`.
* **Plugin DaemonSet Upgrades:** FUSE-backed plugin DaemonSets use `updateStrategy.type: RollingUpdate` (`maxUnavailable: 10%`, `terminationGracePeriodSeconds: 3600`) with `SIGTERM` lame-duck self-draining (`--drain-timeout=30m`), `/run/ate/drain-complete` `preStop` synchronization, and suspends of active streamed actors through the control plane's `RequestActorSuspend`. See *Appendix: [Plugin DaemonSet Upgrade](#plugin-daemonset-upgrade)* for details.

## Medium-Term Proposal

* Optimize cold/warm latency after delivering functional Riptide v2
* Extend to other OSS remote snapshotters
* Implement `LAYER_FORM_EROFS` for microVM workers

## Appendix

### Host `containerd` GC Interferences

`containerd` GC interferes with Substrate when it shares the on-host Riptide Snapshotter and `gcfsd`.

`containerd` and Riptide Snapshotter maintain two completely separate BoltDB databases (`containerd`'s `/var/lib/containerd/.../meta.db` vs. Riptide's internal `/var/lib/containerd-gcfs-grpc/metadata.db`). Riptide never touches `containerd`'s BoltDB. However, `containerd` assumes it is the sole client of its configured `[proxy_plugins.gcfs]` gRPC socket. During `containerd`'s periodic GC pass, `containerd` calls `Snapshots.Walk()` over gRPC on Riptide Snapshotter's sock to list every snapshot stored in Riptide's BoltDB, compares that list against `containerd`'s own `meta.db`, and calls `Snapshots.Remove(key)` over gRPC on any snapshot key in Riptide's BoltDB that `containerd` itself did not create (i.e., Substrate's layers, since Riptide's internal `snapshots/storage` BoltDB bucket is not partitioned by `containerd-namespace`).

Additionally, in Riptide v2, Riptide Snapshotter delegates `CreateView` and `DeleteView` over `/run/gcfsd/grpc.sock` using the unscoped layer `chainID` hex (`views/<chainID>/fs`) as the view key; sharing the host's `gcfsd` causes `CreateView` collisions (`view "<chainID>" already exists`) for layers already mounted by host `containerd` and causes host `containerd` GC (`DeleteView(<chainID>)`) to unmount `views/<chainID>/fs` out from under Substrate.

### Plugin DaemonSet Upgrade

`atelet` and `image-plugin-<provider>` have decoupled upgrade lifecycles:

1. **Upgrading Agent Substrate (`deploy atelet`):** Like CSI drivers, `image-plugin-<provider>` is outside the Substrate system upgrade. When `ds/atelet` rolls in-place, the `image-plugin-<provider>` pod and its FUSE mounts stay running uninterrupted, and the restarted `atelet` recovers active leases via `ReconcileImages`. The `ImagePlugin` gRPC protocol evolves via additive optional protobuf fields so `atelet` and the plugin can upgrade independently.
2. **Upgrading a FUSE-Backed Plugin DaemonSet (`RollingUpdate` + `SIGTERM` Lame-Duck Drain + Actor Eviction Cycling):**
   * **Step 1 — `SIGTERM` Initiates Lame-Duck Mode (10% of Nodes per Batch):** Applying an updated `<provider>.yaml` from the `image-plugin` release triggers a standard Kubernetes `RollingUpdate` (`maxUnavailable: 10%`, `terminationGracePeriodSeconds: 3600`). When `kubelet` begins terminating a plugin pod on a node, the plugin container receives `SIGTERM` and immediately enters lame-duck mode: `CanProvide` returns `can_provide=false, state=PLUGIN_STATE_DRAINING`, `PrepareImage` returns an error so `atelet` falls back to `/var/lib/ate/image-cache` for all new and resuming actors on the node, and idle cached leases (`refCount == 0`) are unmounted immediately.
   * **Step 2 — Suspending Active Streamed Actors:** Unlike a `WorkerPool` upgrade (where Kubernetes sends `SIGTERM` directly to the `ateworker` pod), rolling `image-plugin-<provider>` only sends `SIGTERM` to the plugin pod. To move active streamed actors off the plugin without restarting `ateworker` pods or disturbing non-streamed actors, `atelet` sees `state=PLUGIN_STATE_DRAINING` from `CanProvide` and calls the control plane's `RequestActorSuspend` for each local actor whose `rootfs-overlay.json` has `streamed: true`. This is the same RPC a worker uses to ask for its actor's suspend:
     1. The control plane suspends the actor through `atelet`, which checkpoints it and calls `ReleaseImage` on the draining plugin. The plugin skips the idle TTL once `refCount` reaches `0`.
     2. The actor is `SUSPENDED`. Its next `ResumeActor` restores it either on another node or on the same node using local `/var/lib/ate/image-cache` layers, or the new plugin once it is ready.
   * **Step 3 — Active FUSE Leases Reach Zero (`preStop` Synchronization):** The provider container runs a `lifecycle.preStop` hook that blocks `kubelet` from sending `SIGTERM` to the FUSE daemon until all active leases drain, or the 30-minute drain timeout expires, and the plugin writes `/run/ate/drain-complete`.
   * **Step 4 — Replacement Pod Starts & Verifies Provider Readiness:** Once `/run/ate/drain-complete` is written, both containers exit cleanly and Kubernetes starts the replacement plugin pod on the node. On startup, `ServeUnixSocket` waits for the driver to verify the underlying provider before binding the image plugin socket, and the container's readiness probe gates Kubernetes `Ready` status before the `RollingUpdate` advances to the next 10% batch of nodes. The replacement pod reuses the node's chunk cache unless the provider cache version changed, so actors after the upgrade start warm.
3. **Stateless Non-FUSE Providers:** Because some plugins, e.g., Spegel, unpack standard files onto the host filesystem rather than holding open `/dev/fuse` mounts, they do not need to drain and can roll in-place immediately.

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
├── image-plugin-cache/                  <--- 2. PLUGIN WRAPPERS & PROVIDER ROOTS (agent:v1)
│   ├── riptide/                         (Riptide plugin --work-dir)
│   │   ├── version                      (Work dir layout version marker)
│   │   └── 9f2c41d0.../                 (Per-image dir: hash of the image ref)
│   │       ├── lease.json               (Persisted lease state for ReconcileImages)
│   │       ├── layer-0/
│   │       │   ├── fs -> /var/lib/ate/image-plugin-cache/riptide-gcfs/.../101/fs (Symlink to FUSE)
│   │       │   └── finalized
│   │       ├── layer-1/
│   │       │   ├── fs -> /var/lib/ate/image-plugin-cache/riptide-gcfs/.../102/fs (Symlink to FUSE)
│   │       │   └── finalized
│   │       └── layer-2/
│   │           ├── fs -> /var/lib/ate/image-plugin-cache/riptide-gcfs/.../103/fs (Symlink to FUSE)
│   │           └── finalized
│   │
│   └── riptide-gcfs/                    <--- ISOLATED RIPTIDE PROVIDER ROOT
│       ├── version                      (Provider cache version; kept, cache wiped on change)
│       ├── mnt/views/<chainID>/fs/      (gcfsd-v2 FUSE virtual layer mounts; cleared on restart)
│       ├── snapshotter/
│       │   ├── metadata.db              (containerd-gcfs-grpc BoltDB; cleared on restart)
│       │   ├── snapshots/<id>/fs -> ../../mnt/views/<chainID>/fs
│       │   └── layers/                  (On-disk downloaded chunk cache; kept across restarts)
│       └── gcfsd/
│           └── images_in_use.db         (gcfsd-v2 chunk cache metadata DB; kept across restarts)
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
            └── rootfs-overlay.json      <-- OverlaySpec (streamed: true, imageRef)
```

### References

* [https://github.com/agent-substrate/substrate/blob/main/docs/upgrade.md](https://github.com/agent-substrate/substrate/blob/main/docs/upgrade.md)
* [https://github.com/agent-substrate/substrate/blob/main/docs/version-skew-policy.md](https://github.com/agent-substrate/substrate/blob/main/docs/version-skew-policy.md)
