# 5-Way Image Streaming Benchmark Report: Riptide v2, AWS SOCI, Spegel P2P, Nydus + Dragonfly P2P, and Baseline (`OCILayerService`)

## 1. Executive Summary

This report presents the end-to-end **45-run benchmark matrix** (`5 configurations × 3 scenarios × 3 workloads`) evaluating all four `OCILayerService` streaming/distribution adapters against `atelet`'s local `imagecache` baseline on **identical `c3-standard-22` (`300 GB pd-ssd`) GKE nodes** and **`1 CPU / 2 GiB` WorkerPools and ActorTemplates** in cluster `substrate-stream-test` (`kuiyue-gke-dev`, `us-central1-a`).

### 1.1 Configurations Tested
1. **Google Riptide v2 (`riptide`):** `cmd/ocilayer-riptide` on COS (`gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62`) streaming 10-layer OCI image from Artifact Registry via `gcfsd-v2` FUSE.
2. **AWS SOCI v0.15.0 (`soci`):** `cmd/ocilayer-soci` on COS (`gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62`) streaming 10-layer OCI image from Artifact Registry via OCI Referrers zTOC (`--min-layer-size=0`, `fuse.rawBridge`).
3. **Spegel Stateless P2P (`spegel`):** `cmd/ocilayer-spegel` on COS (`gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62`) pulling OCI layer blobs over in-cluster Kademlia DHT P2P from peer node `gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc`.
4. **Baseline (`baseline`):** `ate.dev/oci-layer-service=disabled` on COS (`gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62`) using `atelet`'s built-in local `internal/imagecache` tarball pull + unpack.
5. **Nydus RAFS v6 + Dragonfly v2 P2P (`nydus-dragonfly`):** `cmd/ocilayer-nydus-dragonfly` on Ubuntu 24.04 (`gke-substrate-stream-c3-std22-ubuntu--5a335c0d-5h2p`, Linux `6.8.0-1046-gke` with `CONFIG_EROFS_FS_ONDEMAND=y`) mounting kernel EROFS over `fscache` backed by Dragonfly `dfdaemon` P2P (`gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc` peer seed).

---

## 2. Benchmark Environment & Artifacts

### 2.1 Hardware & Resource Configuration
- **GKE Cluster:** `substrate-stream-test` (`us-central1-a`, `v1.36.4-gke.1495000`)
- **Target Benchmark Nodes (`c3-standard-22`, 22 vCPU, 88 GiB RAM, `300 GB pd-ssd` boot disk):**
  - **COS Node:** `gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62` (`wp-c3-cos-644f9566dd-96zcz`)
  - **Ubuntu Node:** `gke-substrate-stream-c3-std22-ubuntu--5a335c0d-5h2p` (`wp-c3-ubuntu-799449f5f6-lv268`)
  - **P2P Seed Peer Node:** `gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc`
- **WorkerPool & ActorTemplate Sizing:** `1 CPU`, `2 GiB` memory (`requests` and `limits` on both WorkerPods and ActorTemplates `tmpl-cos-oci` and `tmpl-ubuntu-nydus`).
- **Golden Snapshots:**
  - `ate-bench-stream/tmpl-cos-oci`: `gs://snapshot-substrate-test-kuiyue-gke-dev/ate-bench-stream/atespaces/ate-golden/actors/bde71309-ab7e-474a-b195-146dd7352a7b/snapshots/daebb7dc-9dad-4f50-94bd-7d41414bb788`
  - `ate-bench-stream/tmpl-ubuntu-nydus`: `gs://snapshot-substrate-test-kuiyue-gke-dev/ate-bench-stream/atespaces/ate-golden/actors/1864816f-fd95-403d-9d04-deacea37d429/snapshots/4f8c9707-25ea-4e1a-8425-311f5612b227`

### 2.2 Built Images (`built_images.env`)
```bash
export ATELET_IMG="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/atelet-89dbecdd4e8d5cd4d125a2de341f399c@sha256:935bda22a6c0c33d312090207ed198b59e696e3cb9038bb29add69224b75cb7b"
export RIPTIDE_IMG="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/ocilayer-riptide-fb3bebf9d8a93a0115f1bba1f3a46c4e@sha256:45336191beb7da8b57ae99fefb58c4d4f8a8b6e255f3afabac5e034359325c76"
export SOCI_IMG="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/ocilayer-soci-7812b0903a149b0ac33f7852e91a4f48@sha256:4b516e725a970bff89c7e973521736c015c666871d5ffc276cda138ca7545734"
export NYDUS_DF_IMG="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/ocilayer-nydus-dragonfly-1e72345282d02b32b52680fec87861f5@sha256:4cd93dbed24bbf92874df9e16dff6c1783c692d3977aeeaa0e4381c4f79c465a"
export SPEGEL_IMG="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/ocilayer-spegel-29f31aaf72b63aa7a9865085196dfba0@sha256:19f11c033e09df8617d38723b82b6566d128a9921eb6d3d7dc86135110abade8"
export SWEBENCH_IMG="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:c8c61a46b939c1b801e08930a2a7cc850b6f8790672634dd26cb8a38ba2965d4"
export SWEBENCH_TAG_IMG="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:swebench-matplotlib"
export SWEBENCH_NYDUS_IMG="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:d89906a9ec2c2edd335a39f38b415dfc86ae62d73a9cfa8a2b69a0feda53a7f7"
export SWEBENCH_NYDUS_TAG_IMG="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:swebench-nydus-v6"
```

### 2.3 Scenarios & Workloads
- **Scenarios:**
  - `cold_boot`: Evicts all read-only layer caches (`ReleaseImage` + cache purge + `drop_caches=3`), creates a new Actor from template, boots from golden snapshot (`ResumeActor(boot=true)`), and runs the workload.
  - `warm_resume`: Suspends the Actor to GCS (`PauseActor`), keeps read-only layers warm in node cache, restores the Actor (`ResumeActor(boot=false)`), and runs the workload.
  - `cold_resume`: Suspends the Actor to GCS (`PauseActor`), evicts all read-only layer caches on the node (`ReleaseImage` + cache purge + `drop_caches=3`) to simulate cross-node migration, restores the Actor (`ResumeActor(boot=false)`), and runs the workload.
- **Workloads (SWE-Bench `matplotlib` 3.6.0 on 10-layer `~1.22 GB` compressed / `2.97 GB` unpacked rootfs):**
  - `small`: `cat -- "/testbed/lib/matplotlib/pyplot.py"` (`112,402` bytes single-file read)
  - `medium`: `/opt/miniconda3/envs/testbed/bin/python -m compileall -f -q "/testbed/lib/matplotlib"` (245 Python files across `/testbed/lib/matplotlib`)
  - `large`: `tar -cf - --one-file-system / | wc -c` (full rootfs traversal reading `2,971,013,120` bytes)

---

## 3. Complete 45-Run Benchmark Matrix

| # | Adapter | Scenario | Workload | Resume RPC (ms) | TTCE (ms) | Workload Run (ms) | E2E (ms) |
|---:|---|---|---|---:|---:|---:|---:|
| 1 | `riptide` | `cold_boot` | `small` | 666.52 | 826.12 | 30.90 | 857.03 |
| 2 | `riptide` | `warm_resume` | `small` | 211.94 | 223.59 | 9.97 | 233.56 |
| 3 | `riptide` | `cold_resume` | `small` | 1047.25 | 1122.42 | 9.67 | 1132.09 |
| 4 | `riptide` | `cold_boot` | `medium` | 480.21 | 601.48 | 1517.40 | 2118.88 |
| 5 | `riptide` | `warm_resume` | `medium` | 205.03 | 222.50 | 1156.87 | 1379.37 |
| 6 | `riptide` | `cold_resume` | `medium` | 692.44 | 776.90 | 1314.60 | 2091.50 |
| 7 | `riptide` | `cold_boot` | `large` | 632.59 | 749.48 | 56295.68 | 57045.15 |
| 8 | `riptide` | `warm_resume` | `large` | 245.62 | 264.20 | 42454.54 | 42718.73 |
| 9 | `riptide` | `cold_resume` | `large` | 900.95 | 991.80 | 52392.28 | 53384.08 |
| 10 | `soci` | `cold_boot` | `small` | 598.88 | 757.13 | 8.59 | 765.72 |
| 11 | `soci` | `warm_resume` | `small` | 262.85 | 274.78 | 8.01 | 282.79 |
| 12 | `soci` | `cold_resume` | `small` | 675.50 | 743.55 | 9.44 | 752.99 |
| 13 | `soci` | `cold_boot` | `medium` | 493.56 | 572.11 | 1538.58 | 2110.70 |
| 14 | `soci` | `warm_resume` | `medium` | 221.94 | 236.47 | 1196.03 | 1432.50 |
| 15 | `soci` | `cold_resume` | `medium` | 625.50 | 719.30 | 1362.20 | 2081.50 |
| 16 | `soci` | `cold_boot` | `large` | 564.27 | 694.16 | 66049.56 | 66743.72 |
| 17 | `soci` | `warm_resume` | `large` | 249.76 | 265.75 | 41355.37 | 41621.12 |
| 18 | `soci` | `cold_resume` | `large` | 678.43 | 830.90 | 57981.60 | 58812.50 |
| 19 | `spegel` | `cold_boot` | `small` | 598.85 | 717.54 | 31.01 | 748.55 |
| 20 | `spegel` | `warm_resume` | `small` | 234.36 | 253.35 | 5.07 | 258.42 |
| 21 | `spegel` | `cold_resume` | `small` | 642.24 | 722.00 | 9.07 | 731.06 |
| 22 | `spegel` | `cold_boot` | `medium` | 497.53 | 632.02 | 1509.01 | 2141.03 |
| 23 | `spegel` | `warm_resume` | `medium` | 254.63 | 275.28 | 1200.97 | 1476.25 |
| 24 | `spegel` | `cold_resume` | `medium` | 697.15 | 813.36 | 1480.87 | 2294.23 |
| 25 | `spegel` | `cold_boot` | `large` | 481.89 | 631.69 | 57028.15 | 57659.83 |
| 26 | `spegel` | `warm_resume` | `large` | 264.23 | 279.33 | 41126.63 | 41405.95 |
| 27 | `spegel` | `cold_resume` | `large` | 718.69 | 801.95 | 48825.89 | 49627.84 |
| 28 | `baseline` | `cold_boot` | `small` | 886.37 | 1026.93 | 46.47 | 1073.40 |
| 29 | `baseline` | `warm_resume` | `small` | 274.53 | 283.84 | 7.02 | 290.86 |
| 30 | `baseline` | `cold_resume` | `small` | 688.42 | 917.41 | 9.94 | 927.34 |
| 31 | `baseline` | `cold_boot` | `medium` | 537.55 | 704.40 | 1628.42 | 2332.82 |
| 32 | `baseline` | `warm_resume` | `medium` | 345.21 | 362.52 | 1160.74 | 1523.26 |
| 33 | `baseline` | `cold_resume` | `medium` | 686.11 | 769.01 | 1305.13 | 2074.14 |
| 34 | `baseline` | `cold_boot` | `large` | 539.57 | 674.09 | 52457.35 | 53131.44 |
| 35 | `baseline` | `warm_resume` | `large` | 243.51 | 259.73 | 48873.69 | 49133.42 |
| 36 | `baseline` | `cold_resume` | `large` | 824.81 | 987.80 | 59619.91 | 60607.71 |
| 37 | `nydus-dragonfly` | `cold_boot` | `small` | 695.99 | 927.48 | 50.72 | 978.20 |
| 38 | `nydus-dragonfly` | `warm_resume` | `small` | 307.80 | 320.52 | 5.86 | 326.38 |
| 39 | `nydus-dragonfly` | `cold_resume` | `small` | 750.57 | 814.86 | 9.38 | 824.24 |
| 40 | `nydus-dragonfly` | `cold_boot` | `medium` | 460.75 | 571.13 | 1949.26 | 2520.38 |
| 41 | `nydus-dragonfly` | `warm_resume` | `medium` | 290.43 | 320.36 | 1094.01 | 1414.37 |
| 42 | `nydus-dragonfly` | `cold_resume` | `medium` | 622.34 | 689.66 | 1231.85 | 1921.51 |
| 43 | `nydus-dragonfly` | `cold_boot` | `large` | 473.61 | 536.19 | 74830.68 | 75366.87 |
| 44 | `nydus-dragonfly` | `warm_resume` | `large` | 325.09 | 340.13 | 36767.36 | 37107.49 |
| 45 | `nydus-dragonfly` | `cold_resume` | `large` | 672.14 | 743.70 | 40472.33 | 41216.03 |

---

## 4. Pivot Comparison Tables by Scenario & Workload

### 4.1 Time-To-Command-Execution (`TTCE ms`) Across All 5 Configurations

| Scenario | Workload | `riptide` (ms) | `soci` (ms) | `spegel` (ms) | `nydus-dragonfly` (ms) | `baseline` (ms) |
|---|---|---:|---:|---:|---:|---:|
| **`cold_boot`** | `small` | 826.12 | 757.13 | **717.54** | 927.48 | 1026.93 |
| **`cold_boot`** | `medium` | 601.48 | 572.11 | 632.02 | **571.13** | 704.40 |
| **`cold_boot`** | `large` | 749.48 | 694.16 | 631.69 | **536.19** | 674.09 |
| **`warm_resume`** | `small` | **223.59** | 274.78 | 253.35 | 320.52 | 283.84 |
| **`warm_resume`** | `medium` | **222.50** | 236.47 | 275.28 | 320.36 | 362.52 |
| **`warm_resume`** | `large` | 264.20 | 265.75 | 279.33 | 340.13 | **259.73** |
| **`cold_resume`** | `small` | 1122.42 | 743.55 | **722.00** | 814.86 | 917.41 |
| **`cold_resume`** | `medium` | 776.90 | 719.30 | 813.36 | **689.66** | 769.01 |
| **`cold_resume`** | `large` | 991.80 | 830.90 | 801.95 | **743.70** | 987.80 |

### 4.2 End-To-End Latency (`E2E ms`) Across All 5 Configurations

| Scenario | Workload | `riptide` (ms) | `soci` (ms) | `spegel` (ms) | `nydus-dragonfly` (ms) | `baseline` (ms) |
|---|---|---:|---:|---:|---:|---:|
| **`cold_boot`** | `small` | 857.03 | 765.72 | **748.55** | 978.20 | 1073.40 |
| **`cold_boot`** | `medium` | 2118.88 | **2110.70** | 2141.03 | 2520.38 | 2332.82 |
| **`cold_boot`** | `large` | 57045.15 | 66743.72 | 57659.83 | 75366.87 | **53131.44** |
| **`warm_resume`** | `small` | **233.56** | 282.79 | 258.42 | 326.38 | 290.86 |
| **`warm_resume`** | `medium` | **1379.37** | 1432.50 | 1476.25 | 1414.37 | 1523.26 |
| **`warm_resume`** | `large` | 42718.73 | 41621.12 | 41405.95 | **37107.49** | 49133.42 |
| **`cold_resume`** | `small` | 1132.09 | 752.99 | **731.06** | 824.24 | 927.34 |
| **`cold_resume`** | `medium` | 2091.50 | 2081.50 | 2294.23 | **1921.51** | 2074.14 |
| **`cold_resume`** | `large` | 53384.08 | 58812.50 | 49627.84 | **41216.03** | 60607.71 |

---

## 5. Step-by-Step Setup, Commands, and Testing Process

For the complete step-by-step setup and execution guide, see [`benchmarking/image-streaming/README.md`](../benchmarking/image-streaming/README.md). Below is a summary of the workflow:

### 5.1 Cluster & Node Pools
1. **COS `c3-standard-22` Node (`COS_NODE`)**: `gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62` (`300 GB pd-ssd`) — hosts `WorkerPool` `wp-c3-cos` (`1 CPU / 2 GiB`) and runs **`riptide`**, **`soci`**, **`spegel`**, and **`baseline`**.
2. **Ubuntu 24.04 `c3-standard-22` Node (`UBUNTU_NODE`)**: `gke-substrate-stream-c3-std22-ubuntu--5a335c0d-5h2p` (`300 GB pd-ssd`, Linux `6.8.0-1046-gke` with `CONFIG_EROFS_FS_ONDEMAND=y` and `fscache`) — hosts `WorkerPool` `wp-c3-ubuntu` (`1 CPU / 2 GiB`) and runs **`nydus-dragonfly`**.
3. **P2P Seed Peer Node (`PEER_NODE`)**: `gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc` — seeds OCI/Nydus blobs over in-cluster P2P for **Spegel** and **Dragonfly**.

### 5.2 Image & Index Preparation (`build_and_prepare_all.sh` & `setup_ubuntu_c3_and_index.sh`)
1. **Build `atelet` and 4 Adapter Images**:
   ```bash
   export KO_DOCKER_REPO="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images"
   ko build ./cmd/atelet
   ko build ./cmd/ocilayer-riptide
   ko build ./cmd/ocilayer-soci
   ko build ./cmd/ocilayer-nydus-dragonfly
   ko build ./cmd/ocilayer-spegel
   ```
2. **Build the 10-Layer SWE-Bench `matplotlib` Image (`SWEBENCH_IMG`)**:
   ```bash
   crane append \
     -b "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:benchmark-9layer" \
     -f "${SCRATCH}/swebench_matplotlib_layer.tar.gz" \
     -t "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:swebench-matplotlib"
   ```
3. **Build & Push AWS SOCI v2 Index (zTOC)**:
   ```bash
   TOKEN=$(gcloud auth print-access-token)
   soci --namespace k8s.io create --min-layer-size 0 "${SWEBENCH_IMG}"
   soci --namespace k8s.io push --user "oauth2accesstoken:${TOKEN}" "${SWEBENCH_IMG}"
   ```
4. **Convert `SWEBENCH_IMG` to Nydus RAFS v6 (`SWEBENCH_NYDUS_IMG`)**:
   ```bash
   nydusify convert --source "${SWEBENCH_TAG_IMG}" --target "${SWEBENCH_NYDUS_TAG_IMG}" --fs-version 6
   ```

### 5.3 Running the 45-Run Benchmark Matrix (`run_full_benchmark_matrix.sh`)
```bash
# 1. Prepare images, SOCI index, Nydus RAFS v6 image, and bench-runner binary:
bash benchmarking/image-streaming/build_and_prepare_all.sh

# 2. Run the full 45-run benchmark matrix across all 5 adapters:
bash benchmarking/image-streaming/run_full_benchmark_matrix.sh
```
- **Adapter Switching**: Managed via node label `ate.dev/oci-layer-service=<riptide|soci|spegel|nydus-dragonfly|disabled>`, which schedules the corresponding `OCILayerService` DaemonSet pod onto the node and binds `/run/ate-image-streaming/ocilayer.sock`.
- **Cold Cache Eviction (`wipe_node_cache`)**: Before every `cold_boot` and `cold_resume` run, the script removes `/var/lib/ateom-gvisor/images/{layers,manifests}/sha256/*`, invokes `OCILayerService.ReleaseImage` on `/run/ate-image-streaming/ocilayer.sock` (which triggers each adapter's `OnReleaseImage` cache cleanup), purges Spegel's local blob store, and runs `sync && echo 3 > /proc/sys/vm/drop_caches`.

