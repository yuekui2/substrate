# 5-Way Image Acceleration Benchmark Report (`ImageAccelerator`)

This report documents the methodology, testbed configuration, and live cluster measurements for the **45-run (`5 adapters × 3 scenarios × 3 workloads`) `ImageAccelerator` benchmark suite** executed on GKE (`kuiyue-gke-dev` / `substrate-stream-test`).

All scripts, helper CLIs, raw JSONL measurements, and execution logs are stored under [`benchmarking/image-accelerator/`](../benchmarking/image-accelerator/README.md):
- Reusable benchmark matrix script: [`benchmarking/image-accelerator/run_full_benchmark_matrix.sh`](../benchmarking/image-accelerator/run_full_benchmark_matrix.sh)
- Raw 45-run JSONL output: [`benchmarking/image-accelerator/results/benchmark_results.jsonl`](../benchmarking/image-accelerator/results/benchmark_results.jsonl)
- Full execution & adapter log evidence: [`benchmarking/image-accelerator/results/benchmark_evidence.log`](../benchmarking/image-accelerator/results/benchmark_evidence.log)

---

## 1. Benchmark Scope & Evaluated Solutions

We evaluate five image provisioning modes for Agent Substrate (`atelet` + `ateworker-gvisor`) from freshly deployed DaemonSets and WorkerPools (`1 CPU`, `2 GiB` memory per worker pod on `c3-standard-22` nodes):

| # | Adapter / Mode | Manifest / Configuration | Image Format & Backing Mechanism | Target Node OS |
| :--- | :--- | :--- | :--- | :--- |
| **1** | **Google Riptide v2 (`riptide`)** | [`oci-image-accelerator/manifests/riptide.yaml`](../oci-image-accelerator/manifests/riptide.yaml) | Standard OCI image (`tar+gzip`); lazy FUSE streaming from Artifact Registry via isolated `gcfsd-v2` + `containerd-gcfs-grpc` in a child mount namespace (works on COS with host `gcfsd.service` enabled or disabled). | GKE COS (`c3-standard-22`) |
| **2** | **AWS SOCI v0.15.0 (`soci`)** | [`oci-image-accelerator/manifests/soci.yaml`](../oci-image-accelerator/manifests/soci.yaml) | Standard OCI image (`tar+gzip`) + OCI Referrers zTOC index; lazy FUSE streaming via `soci-snapshotter-grpc` (`fuse.rawBridge`). | GKE COS (`c3-standard-22`) |
| **3** | **Spegel Stateless P2P (`spegel`)** | [`oci-image-accelerator/manifests/spegel.yaml`](../oci-image-accelerator/manifests/spegel.yaml) | Standard OCI image (`tar+gzip`); stateless in-cluster P2P blob fetch (`:5000`/`:5001`) from a seed peer node + local tar extraction into `/var/lib/ate/accelerator-image-cache/spegel-store`. | GKE COS (`c3-standard-22`) |
| **4** | **Baseline (`baseline`)** | `ate.dev/image-accelerator=disabled` (no adapter DaemonSet) | Standard OCI image (`tar+gzip`); `atelet`'s built-in `internal/imagecache` pulling layer tarballs from Artifact Registry and unpacking to `/var/lib/ate/image-cache`. | GKE COS (`c3-standard-22`) |
| **5** | **Nydus RAFS v6 + Dragonfly v2 P2P (`nydus-dragonfly`)** | [`oci-image-accelerator/manifests/nydus-dragonfly.yaml`](../oci-image-accelerator/manifests/nydus-dragonfly.yaml) | Nydus RAFS v6 chunked Zstd blobs + bootstrap; in-kernel EROFS over `fscache` (`CONFIG_EROFS_FS_ONDEMAND=y`, `0xE0F5E1E2`) demand-paged via `nydusd` and Dragonfly `dfdaemon` P2P (`:65001`). | GKE Ubuntu 24.04 (`c3-standard-22`) |

---

## 2. Testbed, Image & Workload Matrix

### 2.1 Cluster & Nodes (`kuiyue-gke-dev` / `substrate-stream-test`, `us-central1-a`)
- **`COS_NODE`**: `gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62` (`c3-standard-22`, Container-Optimized OS, GKE Image Streaming enabled) — hosts `wp-c3-cos` (`wp-c3-cos-67f5475486-mm5z4`, `1 CPU / 2 GiB`).
- **`NO_STREAM_COS_NODE`**: `gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz` (`c3-standard-4`, Container-Optimized OS, GKE Image Streaming disabled) — verifies Riptide dual-mode compatibility when host `gcfsd.service` is inactive.
- **`UBUNTU_NODE`**: `gke-substrate-stream-c3-std22-ubuntu--5a335c0d-1261` (`c3-standard-22`, Ubuntu 24.04 with `erofs` and `cachefiles` on-demand kernel modules) — hosts `wp-c3-ubuntu` (`wp-c3-ubuntu-86dd786b4b-8zprf`, `1 CPU / 2 GiB`).
- **`PEER_NODE`**: `gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc` (`c3-standard-4`, Ubuntu 24.04) — P2P seeder node for Spegel and Dragonfly.

### 2.2 Container Images
- **Standard 10-Layer SWE-bench `matplotlib` OCI Image (`SWEBENCH_IMG`, ~1.22 GB compressed / ~2.97 GB unpacked)**:
  - `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:c8c61a46b939c1b801e08930a2a7cc850b6f8790672634dd26cb8a38ba2965d4`
  - Indexed with AWS SOCI v2 zTOC referrers in Artifact Registry for the `soci` suite.
- **Nydus RAFS v6 Converted Image (`SWEBENCH_NYDUS_IMG`)**:
  - `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:d89906a9ec2c2edd335a39f38b415dfc86ae62d73a9cfa8a2b69a0feda53a7f7`

### 2.3 Workloads & Lifecycle Scenarios
Each adapter is freshly deployed (all other adapter DaemonSets deleted and `/var/lib/ate/accelerator-image-cache/*` wiped) before its 9-run suite (`3 workloads × 3 scenarios`):

1. **Workloads**:
   - **`small`**: `cat -- "/testbed/lib/matplotlib/pyplot.py"` (`112,402` bytes single-file read).
   - **`medium`**: `/opt/miniconda3/envs/testbed/bin/python -m compileall -f -q "/testbed/lib/matplotlib"` (Python AST parse & bytecode compilation across 245 files in `/testbed/lib/matplotlib`).
   - **`large`**: `tar -cf - --one-file-system / | wc -c` (full rootfs traversal reading `~3.0 GB` across all layers).
2. **Scenarios**:
   - **`cold_boot`**: Evict node-local `/var/lib/ate/image-cache`, call `ImageAccelerator.ReleaseImage(bypass_ttl=true)`, clear Spegel store (for `spegel`), and drop kernel page caches (`echo 3 > /proc/sys/vm/drop_caches`); then create and boot a new actor from the `ActorTemplate` golden snapshot and execute the workload.
   - **`warm_resume`**: After suspending the actor (`SuspendActor(DURABILITY_MODE_SYNCHRONOUS)`), resume on the same node with both the local checkpoint and mounted read-only image layers warm, then execute the workload.
   - **`cold_resume`**: After suspending to GCS, call `RevertActor` (evicting the node-local checkpoint via `atelet.Terminate` while preserving the durable GCS snapshot) and run `wipe_node_cache` (evicting all node-local read-only layers and kernel page caches); then call `ResumeActor` to restore from GCS and re-prepare the image layers cold before executing the workload.

---

## 3. Riptide Dual-Mode COS Verification (GKE Image Streaming Enabled vs. Disabled)

Before running the 45-run matrix, `ds/ate-accelerator-riptide` was freshly deployed onto both `COS_NODE` (`gcfsd.service` active) and `NO_STREAM_COS_NODE` (`gcfsd.service` inactive) and probed with `e2e-stream-probe -bypass-ttl=true`:

| COS Node Pool Mode | Node | `CanProvide` | Cold `PrepareImage` (10 Layers) | `ReleaseImage (bypass_ttl=true)` | Verified Layers |
| :--- | :--- | ---: | ---: | ---: | ---: |
| **GKE Image Streaming Enabled** (`gcfsd.service` active) | `gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62` | `true` (`1.75 ms`) | **`1,338.30 ms`** | `18.13 ms` | `10 / 10` |
| **GKE Image Streaming Disabled** (`gcfsd.service` inactive) | `gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz` | `true` (`2.56 ms`) | **`1,291.74 ms`** | `17.57 ms` | `10 / 10` |

---

## 4. Complete 45-Run Benchmark Results

### 4.1 `small` Workload (`cat /testbed/lib/matplotlib/pyplot.py`, `112,402` bytes)

| Adapter | Scenario | `ResumeActor` RPC (ms) | Client TTCE (ms) | Workload Run (ms) | Client E2E (ms) | Speedup vs. Baseline E2E |
| :--- | :--- | ---: | ---: | ---: | ---: | ---: |
| **`nydus-dragonfly`** | `cold_boot` | 633.43 | 826.63 | 48.58 | **875.21** | **26.4x** |
| **`soci`** | `cold_boot` | 810.78 | 1,600.06 | 103.81 | **1,703.86** | **13.6x** |
| **`riptide`** | `cold_boot` | 1,784.93 | 1,925.42 | 15.37 | **1,940.79** | **11.9x** |
| **`spegel`** | `cold_boot` | 18,345.76 | 18,399.60 | 5.47 | **18,405.07** | **1.26x** |
| **`baseline`** | `cold_boot` | 23,067.87 | 23,110.59 | 5.55 | **23,116.14** | 1.00x |
| **`baseline`** | `warm_resume` | 106.93 | 137.79 | 6.76 | **144.55** | 1.00x |
| **`spegel`** | `warm_resume` | 93.91 | 147.88 | 5.56 | **153.44** | 0.94x |
| **`nydus-dragonfly`** | `warm_resume` | 145.98 | 163.63 | 6.25 | **169.89** | 0.85x |
| **`soci`** | `warm_resume` | 123.69 | 166.55 | 8.28 | **174.84** | 0.83x |
| **`riptide`** | `warm_resume` | 171.19 | 194.21 | 7.91 | **202.11** | 0.72x |
| **`nydus-dragonfly`** | `cold_resume` | 487.60 | 550.51 | 10.17 | **560.67** | **42.6x** |
| **`soci`** | `cold_resume` | 935.30 | 1,007.17 | 12.19 | **1,019.36** | **23.4x** |
| **`riptide`** | `cold_resume` | 950.35 | 1,068.40 | 17.44 | **1,085.85** | **22.0x** |
| **`spegel`** | `cold_resume` | 17,926.53 | 17,951.12 | 5.63 | **17,956.75** | **1.33x** |
| **`baseline`** | `cold_resume` | 23,868.23 | 23,890.10 | 5.88 | **23,895.98** | 1.00x |

### 4.2 `medium` Workload (`python -m compileall -f -q /testbed/lib/matplotlib`, 245 files)

| Adapter | Scenario | `ResumeActor` RPC (ms) | Client TTCE (ms) | Workload Run (ms) | Client E2E (ms) | Speedup vs. Baseline E2E |
| :--- | :--- | ---: | ---: | ---: | ---: | ---: |
| **`nydus-dragonfly`** | `cold_boot` | 371.47 | 430.89 | 1,876.94 | **2,307.84** | **10.6x** |
| **`riptide`** | `cold_boot` | 473.25 | 573.66 | 1,833.05 | **2,406.72** | **10.2x** |
| **`soci`** | `cold_boot` | 518.35 | 619.62 | 3,361.91 | **3,981.53** | **6.15x** |
| **`spegel`** | `cold_boot` | 18,379.75 | 18,409.31 | 1,365.73 | **19,775.04** | **1.24x** |
| **`baseline`** | `cold_boot` | 23,229.71 | 23,255.36 | 1,228.18 | **24,483.54** | 1.00x |
| **`nydus-dragonfly`** | `warm_resume` | 155.46 | 172.33 | 1,167.12 | **1,339.45** | **1.05x** |
| **`baseline`** | `warm_resume` | 118.74 | 150.27 | 1,251.10 | **1,401.36** | 1.00x |
| **`spegel`** | `warm_resume` | 170.21 | 214.64 | 1,223.51 | **1,438.15** | 0.97x |
| **`riptide`** | `warm_resume` | 109.44 | 159.75 | 1,365.32 | **1,525.07** | 0.92x |
| **`soci`** | `warm_resume` | 182.84 | 205.35 | 1,327.47 | **1,532.82** | 0.91x |
| **`nydus-dragonfly`** | `cold_resume` | 497.45 | 568.37 | 1,267.89 | **1,836.26** | **13.7x** |
| **`riptide`** | `cold_resume` | 505.46 | 649.77 | 1,704.78 | **2,354.54** | **10.7x** |
| **`soci`** | `cold_resume` | 633.55 | 745.38 | 1,917.53 | **2,662.91** | **9.43x** |
| **`spegel`** | `cold_resume` | 17,619.81 | 17,648.71 | 1,343.12 | **18,991.83** | **1.32x** |
| **`baseline`** | `cold_resume` | 23,840.91 | 23,874.36 | 1,228.22 | **25,102.58** | 1.00x |

### 4.3 `large` Workload (`tar -cf - --one-file-system / | wc -c`, `~3.0 GB` full rootfs read)

| Adapter | Scenario | `ResumeActor` RPC (ms) | Client TTCE (ms) | Workload Run (ms) | Client E2E (ms) | TTCE Speedup vs. Baseline |
| :--- | :--- | ---: | ---: | ---: | ---: | ---: |
| **`nydus-dragonfly`** | `cold_boot` | 392.06 | 470.50 | 75,643.16 | **76,113.66** | **51.6x** |
| **`soci`** | `cold_boot` | 423.88 | 542.03 | 112,748.04 | **113,290.07** | **44.8x** |
| **`riptide`** | `cold_boot` | 816.58 | 923.25 | 96,461.45 | **97,384.70** | **26.3x** |
| **`spegel`** | `cold_boot` | 18,049.01 | 18,096.27 | 18,293.40 | **36,389.67** | **1.34x** |
| **`baseline`** | `cold_boot` | 24,228.17 | 24,257.82 | 18,370.27 | **42,628.08** | 1.00x |
| **`spegel`** | `warm_resume` | 255.65 | 273.47 | 19,381.43 | **19,654.90** | 0.46x |
| **`baseline`** | `warm_resume` | 108.14 | 125.89 | 19,565.04 | **19,690.93** | 1.00x |
| **`soci`** | `warm_resume` | 258.48 | 294.76 | 34,309.23 | **34,603.99** | 0.43x |
| **`nydus-dragonfly`** | `warm_resume` | 211.92 | 266.23 | 37,152.16 | **37,418.40** | 0.47x |
| **`riptide`** | `warm_resume` | 183.60 | 221.14 | 41,927.58 | **42,148.72** | 0.57x |
| **`nydus-dragonfly`** | `cold_resume` | 537.10 | 598.71 | 40,893.92 | **41,492.63** | **40.1x** |
| **`riptide`** | `cold_resume` | 546.84 | 681.31 | 95,210.09 | **95,891.40** | **35.2x** |
| **`soci`** | `cold_resume` | 609.50 | 712.78 | 43,739.09 | **44,451.88** | **33.7x** |
| **`spegel`** | `cold_resume` | 17,499.02 | 17,524.36 | 18,657.79 | **36,182.15** | **1.37x** |
| **`baseline`** | `cold_resume` | 23,934.21 | 23,994.54 | 18,636.57 | **42,631.11** | 1.00x |

---

## 5. Key Findings & Architectural Takeaways

1. **Cold Startup & Cross-Node Migration (`cold_boot` and `cold_resume` on `small` and `medium` Agent Workloads)**:
   - Without image streaming (`baseline`), `cold_boot` and `cold_resume` must pull and unpack all 10 layers (`1.22 GB` compressed / `2.97 GB` unpacked) into `/var/lib/ate/image-cache`, taking **`23.07 s – 24.23 s`** in `ResumeActor` before the actor can execute a single command.
   - All three lazy-streaming adapters eliminate upfront layer downloads and cut `ResumeActor` latency to **`371 ms – 1,785 ms`**:
     - **`nydus-dragonfly` (Kernel EROFS + `fscache` + Dragonfly P2P)**: **`371 ms – 633 ms`** `ResumeActor` RPC, achieving a **`26.4x` E2E speedup** on `small | cold_boot` (`875.21 ms` vs `23,116.14 ms`), **`42.6x` E2E speedup** on `small | cold_resume` (`560.67 ms` vs `23,895.98 ms`), and **`10.6x – 13.7x` E2E speedup** on `medium` (`2,307.84 ms` / `1,836.26 ms` vs `24,483.54 ms` / `25,102.58 ms`).
     - **`riptide` (Google Riptide v2 isolated FUSE on GKE COS)**: **`473 ms – 1,785 ms`** `ResumeActor` RPC with **zero image conversion or external indexing**, achieving an **`11.9x – 22.0x` E2E speedup** on `small` (`1,940.79 ms` / `1,085.85 ms`) and **`10.2x – 10.7x` E2E speedup** on `medium` (`2,406.72 ms` / `2,354.54 ms`).
     - **`soci` (AWS SOCI v0.15.0 FUSE on GKE COS)**: **`424 ms – 935 ms`** `ResumeActor` RPC once zTOC descriptors are registered, achieving a **`13.6x – 23.4x` E2E speedup** on `small` (`1,703.86 ms` / `1,019.36 ms`) and **`6.15x – 9.43x` E2E speedup** on `medium` (`3,981.53 ms` / `2,662.91 ms`).
2. **Full-Rootfs Traversal (`large` Workload Reading 100% of `~3.0 GB` Rootfs)**:
   - Even when a workload reads every byte in the container rootfs (`tar -cf - --one-file-system / | wc -c`), lazy streaming still delivers **`26.3x – 51.6x` faster Time-To-Command-Execution (TTCE)** (`470 ms – 923 ms` vs `24,258 ms`).
   - For total end-to-end duration when reading 100% of all files sequentially, **Spegel P2P (`spegel`)** (`36.39 s` `cold_boot`, `36.18 s` `cold_resume`) and **Baseline (`baseline`)** (`42.63 s`) complete the `tar` traversal in `18.3 s – 18.6 s` because all files are already unpacked on the local NVMe/PD filesystem (`ext4`), whereas FUSE/on-demand chunk paging incurs per-chunk network/decompression overhead on uncached reads (`41.49 s – 113.29 s`).
3. **Warm Resume (`warm_resume`)**:
   - Across all five modes, `warm_resume` completes `ResumeActor` in **`94 ms – 258 ms`** and `small` E2E in **`145 ms – 202 ms`**, confirming that `ImageAccelerator` lease caching (`refCount` + `idleTTL`) adds near-zero overhead on warm actor wakeups.
