# 5-Way `ImageAccelerator` Benchmark & E2E Verification Guide

This directory contains the reusable scripts, Go benchmark/probe CLIs, image environment definitions, and captured live cluster results for the **45-run (`5 adapters × 3 scenarios × 3 workloads`) `ImageAccelerator` benchmark suite** and **decoupled upgrade verification suite**:

1. **Google Riptide v2 (`riptide`)** — `oci-image-accelerator/cmd/riptide` streaming OCI layers from Artifact Registry via `gcfsd-v2` FUSE (verified on both COS with GKE Image Streaming enabled and COS with GKE Image Streaming disabled).
2. **AWS SOCI v0.15.0 (`soci`)** — `oci-image-accelerator/cmd/soci` streaming OCI layers from Artifact Registry via OCI Referrers zTOC (`fuse.rawBridge`).
3. **Spegel Stateless P2P (`spegel`)** — `oci-image-accelerator/cmd/spegel` pulling OCI layer blobs over in-cluster P2P from a seed peer node.
4. **Baseline (`baseline`)** — `ate.dev/image-accelerator=disabled`, using `atelet`'s built-in local `internal/imagecache` tarball pull and parallel unpack.
5. **Nydus RAFS v6 + Dragonfly v2 P2P (`nydus-dragonfly`)** — `oci-image-accelerator/cmd/nydus-dragonfly` mounting kernel EROFS over `fscache` (`CONFIG_EROFS_FS_ONDEMAND=y`) backed by Dragonfly `dfdaemon` P2P.

---

## 1. Directory Contents

| File | Description |
| :--- | :--- |
| [`built_images.env`](./built_images.env) | Pinned Artifact Registry image digests for `atelet`, `ateworker-gvisor`, the 4 `ImageAccelerator` adapters, the 10-layer SWE-bench OCI image (`SWEBENCH_IMG`), and the Nydus RAFS v6 image (`SWEBENCH_NYDUS_IMG`). |
| [`build_and_prepare_all.sh`](./build_and_prepare_all.sh) | Builds/pushes all 4 `ImageAccelerator` adapter images with `ko`, updates `built_images.env`, and invokes `setup_ubuntu_c3_and_index.sh`. |
| [`setup_ubuntu_c3_and_index.sh`](./setup_ubuntu_c3_and_index.sh) | Compiles `bench-runner` and `e2e-stream-probe`, deploys fresh `bench-helper` pods in `ate-system`, copies the binaries into `/home/kubernetes/bin`, and installs `soci` v0.15.0 on the target nodes. |
| [`run_full_benchmark_matrix.sh`](./run_full_benchmark_matrix.sh) | Deploys all 4 adapter DaemonSets (`oci-image-accelerator/manifests/*.yaml`), provisions `1 CPU / 2 GiB` `WorkerPool`s and `ActorTemplate`s, verifies Riptide on both COS modes, and executes the complete 45-run benchmark matrix. |
| [`run_upgrade_tests.sh`](./run_upgrade_tests.sh) | Executes live E2E verification from fresh DaemonSets/WorkerPools: (A1) `ds/atelet` rolling upgrade with an active streamed actor (`ReconcileImages`), (A2) dynamic node opt-out (`ate.dev/image-accelerator=disabled`) with `EvictWorkload` (`SIGTERM`) without terminating the adapter DaemonSet, and (B.1–B.4) `RollingUpdate` lame-duck drain and `EvictWorkload` verification across all 4 adapters (`riptide`, `soci`, `nydus-dragonfly`, `spegel`). |
| [`bench-runner/main.go`](./bench-runner/main.go) | Standalone Go CLI that drives `ateapi` (`CreateActorTemplate`, `CreateActor`, `ResumeActor`, `SuspendActor`, `RevertActor`, `DeleteActor`), `atenet` HTTP workload requests, and `ImageAccelerator.ReleaseImage` cache eviction. |
| [`e2e-stream-probe/main.go`](./e2e-stream-probe/main.go) | Standalone Go CLI that probes `/run/ate/image-accelerator.sock` (`CanProvide`, `PrepareImage`, layer directory verification, and `ReleaseImage`). |
| [`results/benchmark_results.jsonl`](./results/benchmark_results.jsonl) | Structured JSONL output containing all 45 benchmark runs with nanosecond timestamps and latency breakdowns. |
| [`results/benchmark_evidence.log`](./results/benchmark_evidence.log) | Detailed execution log including Riptide dual-mode probe output, `ateapi` `"Picked worker"` timestamps, and per-run adapter / `atelet` logs. |
| [`results/upgrade_verification.log`](./results/upgrade_verification.log) | Step-by-step live cluster execution log (commands, outputs, and logs) for Scenarios A1, A2, and B.1–B.4. |

---

## 2. Cluster & Node Pool Setup

The benchmark and E2E verification run on GKE cluster `substrate-stream-test` (`us-central1-a`, project `kuiyue-gke-dev`):

1. **COS Benchmark Node (`COS_NODE`)**: `gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62` (`c3-standard-22`, COS, GKE Image Streaming enabled)
   - Runs **`riptide`**, **`soci`**, **`spegel`**, and **`baseline`**, plus the live upgrade tests.
   - Labeled with `bench-node=c3-cos` and hosts `WorkerPool` `wp-c3-cos` (`1 CPU`, `2 GiB`).
2. **COS No-Streaming Node (`NO_STREAM_COS_NODE`)**: `gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz` (`c3-standard-4`, COS, GKE Image Streaming disabled)
   - Verifies that `ate-accelerator-riptide` works identically when host `gcfsd.service` is inactive by starting its own isolated `gcfsd-v2` + `containerd-gcfs-grpc` pair from `/home/kubernetes/bin`.
3. **Ubuntu 24.04 Benchmark Node (`UBUNTU_NODE`)**: `gke-substrate-stream-c3-std22-ubuntu--5a335c0d-1261` (`c3-standard-22`, Ubuntu 24.04)
   - Runs **`nydus-dragonfly`** (Linux kernel `CONFIG_EROFS_FS_ONDEMAND=y` and `fscache` with `erofs` and `cachefiles` modules).
   - Labeled with `bench-node=c3-ubuntu` and hosts `WorkerPool` `wp-c3-ubuntu` (`1 CPU`, `2 GiB`).
4. **P2P Seed Peer Node (`PEER_NODE`)**: `gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc` (`c3-standard-4`, Ubuntu 24.04)
   - Seeds OCI/Nydus blobs over in-cluster P2P for **Spegel** (`:5000`/`:5001`) and **Dragonfly** (`dfdaemon` `:65000`–`:65002`).

---

## 3. Step-by-Step Commands (From Fresh Cluster State)

### Step 3.1: Deploy Fresh `ate-system` and Publish Worker Images
```bash
source .ate-dev-env.sh
export SNAPSHOT_LOCATION="gs://snapshot-substrate-test-kuiyue-gke-dev"

# Deploy fresh ate-system control plane and node agents
go run ./cmd/ate-setup --rollout-timeout=180s deploy ate-system \
  --credential-provider='{"enabled":false}' \
  --ingress-allowed-spiffe-ids=spiffe://cluster.local/ns/ate-system/sa/ate-client

# Build and publish ateworker-gvisor and ateworker-microvm images
go run ./cmd/ate-setup publish worker-images
```

### Step 3.2: Build All 4 Adapter Images & Prepare Helper Pods
```bash
bash benchmarking/image-accelerator/build_and_prepare_all.sh
```

### Step 3.3: Run the 45-Run 5-Way Benchmark Matrix
```bash
bash benchmarking/image-accelerator/run_full_benchmark_matrix.sh
```

### Step 3.4: Run the Live Upgrade Verification Suite (Scenarios A & B)
```bash
bash benchmarking/image-accelerator/run_upgrade_tests.sh
```

---

## 4. Benchmark Lifecycle & Cache Eviction Details

### 4.1 Workloads
Every workload is executed via `bench-runner` sending an HTTP POST to `atenet-router` (`http://<ATENET_IP>:80`) with header `X-Ate-Actor: ate-bench-stream/<actor_name>`:
- **`small` (Single-file read, `112,402` bytes)**:
  ```bash
  cat -- "/testbed/lib/matplotlib/pyplot.py"
  ```
- **`medium` (Multi-file Python bytecode compilation across 245 files)**:
  ```bash
  /opt/miniconda3/envs/testbed/bin/python -m compileall -f -q "/testbed/lib/matplotlib"
  ```
- **`large` (Full rootfs traversal reading `2,971,013,120` bytes)**:
  ```bash
  tar -cf - --one-file-system / | wc -c
  ```

### 4.2 Per-Workload Lifecycle Scenarios
For each `(adapter, workload)` pair:
1. **`cold_boot` (`--mode=create-boot-run`)**:
   - Run `wipe_node_cache` (purges `/var/lib/ate/image-cache/*`, calls `ImageAccelerator.ReleaseImage(bypass_ttl=true)` on `/run/ate/image-accelerator.sock`, clears Spegel store if applicable, and drops kernel page caches).
   - Call `CreateActor` from template, then `ResumeActor` to restore from the golden snapshot and mount the image layers on demand, then immediately run the workload command.
2. **Suspend (`--mode=suspend --clean-pycache=true`)**:
   - Clean any `__pycache__` files written to the sandbox upperdir, then call `SuspendActor(DURABILITY_MODE_SYNCHRONOUS)` to checkpoint the running gVisor sandbox both locally and to GCS.
3. **`warm_resume` (`--mode=resume-and-run`)**:
   - With both the local checkpoint and read-only image layers warm on the node, call `ResumeActor` and run the workload command.
4. **Suspend + Revert (`--mode=suspend` followed by `--mode=revert`)**:
   - Checkpoint to GCS (`SuspendActor(DURABILITY_MODE_SYNCHRONOUS)`), then call `RevertActor` to evict the node-local checkpoint via `atelet.Terminate` while preserving the durable GCS snapshot.
5. **`cold_resume` (`--mode=resume-and-run`)**:
   - Run `wipe_node_cache` to evict all node-local read-only layers and kernel page caches.
   - Call `ResumeActor` (which restores from the durable GCS snapshot and calls `ImageAccelerator.PrepareImage`) and run the workload command.
6. **Cleanup (`--mode=delete`)**:
   - Delete the actor via `DeleteActor`.

---

## 5. Live 45-Run Benchmark Results Summary (`results/benchmark_results.jsonl`)

### 5.1 Riptide Dual-Mode COS Verification (`results/benchmark_evidence.log`)
- **COS Node with GKE Image Streaming Enabled** (`gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62`):
  - `CanProvide`: `true` (`1.75 ms`), `PrepareImage`: `10` layers (`1,338.30 ms`), `ReleaseImage`: `18.13 ms`
- **COS Node with GKE Image Streaming Disabled** (`gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz`):
  - `CanProvide`: `true` (`2.56 ms`), `PrepareImage`: `10` layers (`1,291.74 ms`), `ReleaseImage`: `17.57 ms`

### 5.2 `small` Workload (`cat /testbed/lib/matplotlib/pyplot.py`, `112,402` bytes)

| Adapter | Scenario | `ResumeActor` RPC (ms) | Client TTCE (ms) | Workload Run (ms) | Client E2E (ms) |
| :--- | :--- | ---: | ---: | ---: | ---: |
| **`nydus-dragonfly`** | `cold_boot` | 633.43 | 826.63 | 48.58 | **875.21** |
| **`nydus-dragonfly`** | `warm_resume` | 145.98 | 163.63 | 6.25 | **169.89** |
| **`nydus-dragonfly`** | `cold_resume` | 487.60 | 550.51 | 10.17 | **560.67** |
| **`soci`** | `cold_boot` | 810.78 | 1,600.06 | 103.81 | **1,703.86** |
| **`soci`** | `warm_resume` | 123.69 | 166.55 | 8.28 | **174.84** |
| **`soci`** | `cold_resume` | 935.30 | 1,007.17 | 12.19 | **1,019.36** |
| **`riptide`** | `cold_boot` | 1,784.93 | 1,925.42 | 15.37 | **1,940.79** |
| **`riptide`** | `warm_resume` | 171.19 | 194.21 | 7.91 | **202.11** |
| **`riptide`** | `cold_resume` | 950.35 | 1,068.40 | 17.44 | **1,085.85** |
| **`spegel`** | `cold_boot` | 18,345.76 | 18,399.60 | 5.47 | **18,405.07** |
| **`spegel`** | `warm_resume` | 93.91 | 147.88 | 5.56 | **153.44** |
| **`spegel`** | `cold_resume` | 17,926.53 | 17,951.12 | 5.63 | **17,956.75** |
| **`baseline`** | `cold_boot` | 23,067.87 | 23,110.59 | 5.55 | **23,116.14** |
| **`baseline`** | `warm_resume` | 106.93 | 137.79 | 6.76 | **144.55** |
| **`baseline`** | `cold_resume` | 23,868.23 | 23,890.10 | 5.88 | **23,895.98** |

### 5.3 `medium` Workload (`python -m compileall -f -q /testbed/lib/matplotlib`, 245 files)

| Adapter | Scenario | `ResumeActor` RPC (ms) | Client TTCE (ms) | Workload Run (ms) | Client E2E (ms) |
| :--- | :--- | ---: | ---: | ---: | ---: |
| **`nydus-dragonfly`** | `cold_boot` | 371.47 | 430.89 | 1,876.94 | **2,307.84** |
| **`nydus-dragonfly`** | `warm_resume` | 155.46 | 172.33 | 1,167.12 | **1,339.45** |
| **`nydus-dragonfly`** | `cold_resume` | 497.45 | 568.37 | 1,267.89 | **1,836.26** |
| **`riptide`** | `cold_boot` | 473.25 | 573.66 | 1,833.05 | **2,406.72** |
| **`riptide`** | `warm_resume` | 109.44 | 159.75 | 1,365.32 | **1,525.07** |
| **`riptide`** | `cold_resume` | 505.46 | 649.77 | 1,704.78 | **2,354.54** |
| **`soci`** | `cold_boot` | 518.35 | 619.62 | 3,361.91 | **3,981.53** |
| **`soci`** | `warm_resume` | 182.84 | 205.35 | 1,327.47 | **1,532.82** |
| **`soci`** | `cold_resume` | 633.55 | 745.38 | 1,917.53 | **2,662.91** |
| **`spegel`** | `cold_boot` | 18,379.75 | 18,409.31 | 1,365.73 | **19,775.04** |
| **`spegel`** | `warm_resume` | 170.21 | 214.64 | 1,223.51 | **1,438.15** |
| **`spegel`** | `cold_resume` | 17,619.81 | 17,648.71 | 1,343.12 | **18,991.83** |
| **`baseline`** | `cold_boot` | 23,229.71 | 23,255.36 | 1,228.18 | **24,483.54** |
| **`baseline`** | `warm_resume` | 118.74 | 150.27 | 1,251.10 | **1,401.36** |
| **`baseline`** | `cold_resume` | 23,840.91 | 23,874.36 | 1,228.22 | **25,102.58** |

### 5.4 `large` Workload (`tar -cf - --one-file-system / | wc -c`, `~3.0 GB` full rootfs read)

| Adapter | Scenario | `ResumeActor` RPC (ms) | Client TTCE (ms) | Workload Run (ms) | Client E2E (ms) |
| :--- | :--- | ---: | ---: | ---: | ---: |
| **`riptide`** | `cold_boot` | 816.58 | 923.25 | 96,461.45 | **97,384.70** |
| **`riptide`** | `warm_resume` | 183.60 | 221.14 | 41,927.58 | **42,148.72** |
| **`riptide`** | `cold_resume` | 546.84 | 681.31 | 95,210.09 | **95,891.40** |
| **`nydus-dragonfly`** | `cold_boot` | 392.06 | 470.50 | 75,643.16 | **76,113.66** |
| **`nydus-dragonfly`** | `warm_resume` | 211.92 | 266.23 | 37,152.16 | **37,418.40** |
| **`nydus-dragonfly`** | `cold_resume` | 537.10 | 598.71 | 40,893.92 | **41,492.63** |
| **`soci`** | `cold_boot` | 423.88 | 542.03 | 112,748.04 | **113,290.07** |
| **`soci`** | `warm_resume` | 258.48 | 294.76 | 34,309.23 | **34,603.99** |
| **`soci`** | `cold_resume` | 609.50 | 712.78 | 43,739.09 | **44,451.88** |
| **`spegel`** | `cold_boot` | 18,049.01 | 18,096.27 | 18,293.40 | **36,389.67** |
| **`spegel`** | `warm_resume` | 255.65 | 273.47 | 19,381.43 | **19,654.90** |
| **`spegel`** | `cold_resume` | 17,499.02 | 17,524.36 | 18,657.79 | **36,182.15** |
| **`baseline`** | `cold_boot` | 24,228.17 | 24,257.82 | 18,370.27 | **42,628.08** |
| **`baseline`** | `warm_resume` | 108.14 | 125.89 | 19,565.04 | **19,690.93** |
| **`baseline`** | `cold_resume` | 23,934.21 | 23,994.54 | 18,636.57 | **42,631.11** |

---

## 6. Live Decoupled Upgrade & Node Opt-Out Verification Summary (`results/upgrade_verification.log`)

All decoupled upgrade and dynamic node opt-out workflows defined in [`docs/image-accelerator-overview.md`](../../docs/image-accelerator-overview.md) and [`oci-image-accelerator/README.md`](../../oci-image-accelerator/README.md) passed from a freshly deployed state (zero reused DaemonSets or WorkerPools):

1. **Scenario A1 — Substrate `atelet` Rolling Upgrade (`RollingUpdate`) with Active Streamed Actor**:
   - Freshly deployed `ds/atelet`, `wp-c3-cos`, `wp-c3-ubuntu`, and `ds/ate-accelerator-riptide` (`ate-accelerator-riptide-gqxh8`, `WaitReady` passed in `2.83s`).
   - Booted `ate-bench-stream/up-riptide-stream` on `riptide` (`rootfs-overlay.json` persisted `"streamed": true` and 10 `/var/lib/ate/accelerator-image-cache/riptide/.../layer-<i>` paths).
   - Rolled `ds/atelet` (`atelet-bq8p9` -> `atelet-plg6b`) while `up-riptide-stream` remained `RUNNING`.
   - Verified `ate-accelerator-riptide-gqxh8` did **not** restart (`restarts=0`, unchanged pod UID `87c87f80-92eb-40d9-af3e-28ad6cca381d`).
   - Verified `up-riptide-stream` read uncached files from the FUSE mount after `atelet` restarted (`ATELET_ROLLOUT_FUSE_READ_OK bytes= 131407`, zero `ENOTCONN`).
   - Verified the new `atelet` pod reconciled active leases on startup:
     - `atelet`: `{"level":"INFO","msg":"Successfully reconciled streamed image leases on startup","provider":"image-accelerator","reconciledImages":1}`
     - `ate-accelerator-riptide`: `INFO Reconciled active streamed image lease provider=riptide ... refCount=1`
2. **Scenario A2 — Dynamic Node Opt-Out (`ate.dev/image-accelerator=disabled`) without Terminating Adapter DaemonSet**:
   - Labeled `gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62` with `ate.dev/image-accelerator=disabled`.
   - Verified `ate-accelerator-riptide-gqxh8` remained `2/2 Running` with unchanged UID `87c87f80-92eb-40d9-af3e-28ad6cca381d`.
   - Verified `atelet` logged `"ImageAccelerator node drain state changed" draining=true` and `"Triggered SIGTERM eviction for active streamed actor" actorUID=88357fe7-6b61-45f5-8a26-4efc638184c1`, and `ateworker` executed `runsc kill container=sandbox signal=SIGTERM`.
   - Booted `up-riptide-optout-fallback` while the node was labeled `disabled` and verified `rootfs-overlay.json` fell back to `/var/lib/ate/image-cache/layers/sha256/...` (`streamed` omitted/`false`).
   - Removed the `ate.dev/image-accelerator=disabled` label (`draining=false`) and verified a new actor immediately resumed streaming on the same `ate-accelerator-riptide-gqxh8` pod (`streamed: true`, `123.77 ms` `ResumeActor`).
3. **Scenario B.1 — Riptide (`ds/ate-accelerator-riptide`) `RollingUpdate` + `SIGTERM` Lame-Duck Drain + `EvictWorkload`**:
   - Patched `ds/ate-accelerator-riptide` (`ate.dev/upgrade-test-rev=v2`) while `up-riptide-stream` held `refCount=1`.
   - Verified `ate-accelerator-riptide-gqxh8` entered `Terminating` lame-duck drain mode (`activeLeases="[...refCount=1]"`, `/run/ate/drain-complete` held back, `e2e-stream-probe` returned `canProvide: false, evicting: true`).
   - Verified `atelet` detected `evicting=true` and triggered `EvictWorkload` (`SIGTERM`) for `up-riptide-stream` (`actorUID=c24fc25f-d092-457d-be8d-8200f8dac9e2`), while the mounted FUSE layer continued serving uncached reads (`LAME_DUCK_DRAIN_READ_OK ... bytes=320619`) and concurrent `up-riptide-fallback` fell back to `/var/lib/ate/image-cache`.
   - Released the lease (`DeleteActor`); verified `All active streamed image leases drained` and `/run/ate/drain-complete` were written, and replacement pod `ate-accelerator-riptide-9lqkk` (`rev=v2`) passed `WaitReady` and `e2e-stream-probe` (`verifiedLayers: 10`).
4. **Scenario B.2 — AWS SOCI (`ds/ate-accelerator-soci`) `RollingUpdate` + `SIGTERM` Lame-Duck Drain + `EvictWorkload`**:
   - Freshly deployed `ds/ate-accelerator-soci` (`ate-accelerator-soci-6jt6m`), booted `up-soci-stream` (`/var/lib/ate/accelerator-image-cache/soci/...`), triggered `RollingUpdate`, verified `Terminating` lame-duck drain (`canProvide: false, evicting: true`), `EvictWorkload` `SIGTERM` (`actorUID=48fc6163-8009-4cc7-8621-3e1de00ba37c`), uncached FUSE read (`LAME_DUCK_DRAIN_READ_OK ... bytes=320619`), fallback actor boot, lease release -> `/run/ate/drain-complete`, and replacement pod `ate-accelerator-soci-ztlmf` (`rev=v2`, `WaitReady` `2.81s`, `verifiedLayers: 10`).
5. **Scenario B.3 — Nydus + Dragonfly (`ds/ate-accelerator-nydus-dragonfly`) `RollingUpdate` + `SIGTERM` Lame-Duck Drain + `EvictWorkload`**:
   - Freshly deployed `ds/ate-accelerator-nydus-dragonfly` (`ate-accelerator-nydus-dragonfly-d56vq`, `WaitReady` `1.01s`), booted `up-nydus-dragonfly-stream` (`/var/lib/ate/accelerator-image-cache/nydus-dragonfly/.../layer-0`), triggered `RollingUpdate`, verified `Terminating` lame-duck drain (`canProvide: false, evicting: true`), `EvictWorkload` `SIGTERM` (`actorUID=1b1fff7e-ad04-475c-a4e3-60a5515d3077`), uncached EROFS+fscache read (`LAME_DUCK_DRAIN_READ_OK ... bytes=320619`), fallback actor boot, lease release -> `/run/ate/drain-complete`, and replacement pod `ate-accelerator-nydus-dragonfly-htggt` (`rev=v2`, `WaitReady` `1.21s`, `verifiedLayers: 1`).
6. **Scenario B.4 — Spegel P2P (`ds/ate-accelerator-spegel`) Stateless `RollingUpdate`**:
   - Freshly deployed `ds/ate-accelerator-spegel` (`ate-accelerator-spegel-mclgc` with `--drain-timeout=0s`), booted `up-spegel-actor` (`/var/lib/ate/accelerator-image-cache/spegel/...`), rolled `ds/ate-accelerator-spegel` to `ate-accelerator-spegel-8pcsj` (`rev=v2`) while `up-spegel-actor` remained `RUNNING`, verified `SPEGEL_ROLLOUT_READ_OK bytes= 131407` and `Reconciled active streamed image lease provider=spegel ... refCount=1`, and verified `e2e-stream-probe` (`verifiedLayers: 10`).


