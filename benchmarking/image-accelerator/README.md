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
| [`run_upgrade_tests.sh`](./run_upgrade_tests.sh) | Executes live E2E verification of both decoupled upgrade workflows: (A) `ds/atelet` rolling upgrade with an active streamed actor (`ReconcileImages`), and (B) `ds/ate-accelerator-riptide` `OnDelete` upgrade with `ate.dev/image-accelerator=disabled` FUSE lease draining. |
| [`bench-runner/main.go`](./bench-runner/main.go) | Standalone Go CLI that drives `ateapi` (`CreateActorTemplate`, `CreateActor`, `ResumeActor`, `SuspendActor`, `RevertActor`, `DeleteActor`), `atenet` HTTP workload requests, and `ImageAccelerator.ReleaseImage` cache eviction. |
| [`e2e-stream-probe/main.go`](./e2e-stream-probe/main.go) | Standalone Go CLI that probes `/run/ate/image-accelerator.sock` (`CanProvide`, `PrepareImage`, layer directory verification, and `ReleaseImage`). |
| [`results/benchmark_results.jsonl`](./results/benchmark_results.jsonl) | Structured JSONL output containing all 45 benchmark runs with nanosecond timestamps and latency breakdowns. |
| [`results/benchmark_evidence.log`](./results/benchmark_evidence.log) | Detailed execution log including Riptide dual-mode probe output, `ateapi` `"Picked worker"` timestamps, and per-run adapter / `atelet` logs. |
| [`results/upgrade_verification.log`](./results/upgrade_verification.log) | Step-by-step live cluster execution log for Upgrade Scenarios A and B. |

---

## 2. Cluster & Node Pool Setup

The benchmark and E2E verification run on GKE cluster `substrate-stream-test` (`us-central1-a`, project `kuiyue-gke-dev`):

1. **COS Benchmark Node (`COS_NODE`)**: `gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62` (`c3-standard-22`, COS, GKE Image Streaming enabled)
   - Runs **`riptide`**, **`soci`**, **`spegel`**, and **`baseline`**, plus the live upgrade tests.
   - Labeled with `bench-node=c3-cos` and hosts `WorkerPool` `wp-c3-cos` (`1 CPU`, `2 GiB`).
2. **COS No-Streaming Node (`NO_STREAM_COS_NODE`)**: `gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz` (`c3-standard-4`, COS, GKE Image Streaming disabled)
   - Verifies that `ate-accelerator-riptide` works identically when host `gcfsd.service` is inactive by starting its own isolated `gcfsd-v2` + `containerd-gcfs-grpc` pair from `/home/kubernetes/bin`.
3. **Ubuntu 24.04 Benchmark Node (`UBUNTU_NODE`)**: `gke-substrate-stream-c3-std22-ubuntu--5a335c0d-5h2p` (`c3-standard-22`, Ubuntu 24.04)
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
   - Run `wipe_node_cache` (purges `/var/lib/ate/images/*`, calls `ImageAccelerator.ReleaseImage(bypass_ttl=true)` on `/run/ate/image-accelerator.sock`, clears Spegel store if applicable, and drops kernel page caches).
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
  - `CanProvide`: `true` (`44.19 ms`), `PrepareImage`: `10` layers (`1390.58 ms`), `ReleaseImage`: `19.65 ms`
- **COS Node with GKE Image Streaming Disabled** (`gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz`):
  - `CanProvide`: `true` (`161.36 ms`), `PrepareImage`: `10` layers (`1838.68 ms`), `ReleaseImage`: `14.13 ms`

### 5.2 `small` Workload (`cat /testbed/lib/matplotlib/pyplot.py`, `112,402` bytes)

| Adapter | Scenario | `ResumeActor` RPC (ms) | Client TTCE (ms) | Workload Run (ms) | Client E2E (ms) |
| :--- | :--- | ---: | ---: | ---: | ---: |
| **`riptide`** | `cold_boot` | 918.02 | 1,075.88 | 16.62 | **1,092.51** |
| **`riptide`** | `warm_resume` | 142.23 | 171.13 | 8.79 | **179.92** |
| **`riptide`** | `cold_resume` | 571.29 | 712.09 | 16.71 | **728.80** |
| **`nydus-dragonfly`** | `cold_boot` | 1,762.45 | 1,975.82 | 47.08 | **2,022.89** |
| **`nydus-dragonfly`** | `warm_resume` | 127.94 | 190.60 | 5.70 | **196.31** |
| **`nydus-dragonfly`** | `cold_resume` | 564.29 | 639.50 | 9.91 | **649.41** |
| **`soci`** | `cold_boot` | 4,845.69 | 9,232.90 | 162.15 | **9,395.05** |
| **`soci`** | `warm_resume` | 140.22 | 166.40 | 8.21 | **174.61** |
| **`soci`** | `cold_resume` | 693.11 | 789.17 | 13.73 | **802.90** |
| **`spegel`** | `cold_boot` | 19,073.79 | 19,098.55 | 6.56 | **19,105.11** |
| **`spegel`** | `warm_resume` | 104.51 | 140.58 | 5.56 | **146.14** |
| **`spegel`** | `cold_resume` | 15,180.40 | 15,208.07 | 5.81 | **15,213.88** |
| **`baseline`** | `cold_boot` | 22,230.24 | 22,255.25 | 5.72 | **22,260.97** |
| **`baseline`** | `warm_resume` | 102.55 | 144.09 | 15.84 | **159.93** |
| **`baseline`** | `cold_resume` | 534.03 | 661.79 | 13.51 | **675.30** |

### 5.3 `medium` Workload (`python -m compileall -f -q /testbed/lib/matplotlib`, 245 files)

| Adapter | Scenario | `ResumeActor` RPC (ms) | Client TTCE (ms) | Workload Run (ms) | Client E2E (ms) |
| :--- | :--- | ---: | ---: | ---: | ---: |
| **`riptide`** | `cold_boot` | 398.51 | 518.95 | 2,186.80 | **2,705.75** |
| **`riptide`** | `warm_resume` | 149.52 | 173.22 | 1,602.68 | **1,775.91** |
| **`riptide`** | `cold_resume` | 534.78 | 662.29 | 2,202.25 | **2,864.54** |
| **`nydus-dragonfly`** | `cold_boot` | 369.90 | 441.67 | 1,897.37 | **2,339.04** |
| **`nydus-dragonfly`** | `warm_resume` | 182.16 | 202.22 | 1,138.86 | **1,341.09** |
| **`nydus-dragonfly`** | `cold_resume` | 498.23 | 565.09 | 1,275.14 | **1,840.23** |
| **`soci`** | `cold_boot` | 417.80 | 522.70 | 4,745.47 | **5,268.17** |
| **`soci`** | `warm_resume` | 117.82 | 169.05 | 1,511.08 | **1,680.13** |
| **`soci`** | `cold_resume` | 482.92 | 642.27 | 2,170.26 | **2,812.53** |
| **`spegel`** | `cold_boot` | 15,435.89 | 15,456.86 | 1,402.78 | **16,859.64** |
| **`spegel`** | `warm_resume` | 194.03 | 211.42 | 1,276.63 | **1,488.06** |
| **`spegel`** | `cold_resume` | 15,240.97 | 15,283.93 | 1,401.82 | **16,685.76** |
| **`baseline`** | `cold_boot` | 416.14 | 530.17 | 1,752.29 | **2,282.46** |
| **`baseline`** | `warm_resume` | 116.31 | 165.59 | 1,453.97 | **1,619.57** |
| **`baseline`** | `cold_resume` | 593.55 | 683.77 | 1,622.23 | **2,306.00** |

### 5.4 `large` Workload (`tar -cf - --one-file-system / | wc -c`, `~3.0 GB` full rootfs read)

| Adapter | Scenario | `ResumeActor` RPC (ms) | Client TTCE (ms) | Workload Run (ms) | Client E2E (ms) |
| :--- | :--- | ---: | ---: | ---: | ---: |
| **`riptide`** | `cold_boot` | 465.87 | 630.51 | 108,876.69 | **109,507.20** |
| **`riptide`** | `warm_resume` | 228.40 | 255.95 | 50,258.58 | **50,514.52** |
| **`riptide`** | `cold_resume` | 752.01 | 867.95 | 110,203.33 | **111,071.28** |
| **`nydus-dragonfly`** | `cold_boot` | 382.03 | 458.63 | 74,209.40 | **74,668.02** |
| **`nydus-dragonfly`** | `warm_resume` | 247.80 | 273.15 | 36,492.10 | **36,765.24** |
| **`nydus-dragonfly`** | `cold_resume` | 574.61 | 639.48 | 39,234.54 | **39,874.01** |
| **`soci`** | `cold_boot` | 458.14 | 578.08 | 162,965.79 | **163,543.87** |
| **`soci`** | `warm_resume` | 203.74 | 227.14 | 34,757.30 | **34,984.45** |
| **`soci`** | `cold_resume` | 500.14 | 639.20 | 44,696.19 | **45,335.39** |
| **`spegel`** | `cold_boot` | 15,270.44 | 15,296.86 | 19,035.46 | **34,332.32** |
| **`spegel`** | `warm_resume` | 235.20 | 252.03 | 19,799.44 | **20,051.47** |
| **`spegel`** | `cold_resume` | 15,301.64 | 15,323.91 | 18,795.43 | **34,119.35** |
| **`baseline`** | `cold_boot` | 863.93 | 967.26 | 72,329.78 | **73,297.04** |
| **`baseline`** | `warm_resume` | 151.75 | 170.35 | 61,571.63 | **61,741.98** |
| **`baseline`** | `cold_resume` | 628.51 | 742.31 | 72,808.29 | **73,550.61** |

---

## 6. Live Decoupled Upgrade Verification Summary (`results/upgrade_verification.log`)

Both decoupled upgrade workflows defined in `docs/proposals/image-streaming/design.md` (§5.12) passed on the live cluster:

1. **Scenario A — Substrate `atelet` Rolling Upgrade (`RollingUpdate`) with Active Streamed Actor**:
   - Booted `ate-bench-stream/upgrade-actor-a` on `riptide` (`rootfs-overlay.json` persisted `"streamed": true` and 10 `/var/lib/ate/streaming/riptide/...` layer paths).
   - Rolled `ds/atelet` (`atelet-sggpj` -> `atelet-tpc5g`) while `upgrade-actor-a` remained `RUNNING`.
   - Verified `ate-accelerator-riptide-z98mc` did **not** restart (`restarts=0`, unchanged pod UID).
   - Verified `upgrade-actor-a` read uncached files from the FUSE mount after `atelet` restarted (`FUSE_READ_OK bytes= 131407`, zero `ENOTCONN`).
   - Verified the new `atelet` pod reconciled active leases on startup:
     - `atelet`: `{"level":"INFO","msg":"Successfully reconciled streamed image leases on startup","provider":"image-accelerator","reconciledImages":1}`
     - `ate-accelerator-riptide`: `INFO Reconciled active streamed image lease provider=riptide ... refCount=1`
2. **Scenario B — Adapter DaemonSet `OnDelete` Upgrade + `ate.dev/image-accelerator=disabled` FUSE Drain**:
   - Booted `ate-bench-stream/upgrade-actor-b` on `riptide`.
   - Patched `ds/ate-accelerator-riptide` pod template (`updateStrategy.type: OnDelete`) and verified `ate-accelerator-riptide-z98mc` remained running while `upgrade-actor-b` was active.
   - Labeled the node `ate.dev/image-accelerator=disabled`; verified `atelet` logged `"ImageAccelerator node drain state changed" draining=true` and `ate-accelerator-riptide-z98mc` stayed `Running` (covered by `nodeAffinity` `In: [riptide, disabled]`) with `DRAIN_FUSE_READ_OK bytes= 131407`.
   - Deleted `upgrade-actor-b`; verified `atelet` called `ReleaseImage(bypass_ttl=true)` and evicted `/var/lib/ate/streaming/riptide/*` (`Remaining workdirs: []`).
   - Booted `upgrade-actor-fallback` while the node was labeled `disabled` and verified `rootfs-overlay.json` fell back to `/var/lib/ate/image-cache/layers/sha256/...`.
   - Deleted the drained adapter pod `ate-accelerator-riptide-z98mc`, relabeled the node `ate.dev/image-accelerator=riptide`, and verified the new adapter pod `ate-accelerator-riptide-w59dq` (`rev=v2`) started and passed `e2e-stream-probe` (`verifiedLayers: 10`).

