# 5-Way `OCILayerService` Image Streaming Benchmark Guide

This directory contains the scripts, Go benchmark runner, image artifacts, and raw results for the **45-run (`5 adapters × 3 scenarios × 3 workloads`) `OCILayerService` benchmark suite** comparing:

1. **Google Riptide v2 (`riptide`)** — `cmd/ocilayer-riptide` streaming OCI layers from Artifact Registry via `gcfsd-v2` FUSE.
2. **AWS SOCI v0.15.0 (`soci`)** — `cmd/ocilayer-soci` streaming OCI layers from Artifact Registry via OCI Referrers zTOC (`fuse.rawBridge`).
3. **Spegel Stateless P2P (`spegel`)** — `cmd/ocilayer-spegel` pulling OCI layer blobs over in-cluster Kademlia DHT P2P from a seed peer node.
4. **Baseline (`baseline`)** — `ate.dev/oci-layer-service=disabled`, using `atelet`'s built-in local `internal/imagecache` tarball pull and parallel unpack.
5. **Nydus RAFS v6 + Dragonfly v2 P2P (`nydus-dragonfly`)** — `cmd/ocilayer-nydus-dragonfly` mounting kernel EROFS over `fscache` (`CONFIG_EROFS_FS_ONDEMAND=y`) backed by Dragonfly `dfdaemon` P2P.

For the full benchmark report and comparison tables, see [`docs/image-streaming-5way-benchmark-report.md`](../../docs/image-streaming-5way-benchmark-report.md).

---

## 1. Directory Contents

| File | Description |
| :--- | :--- |
| [`built_images.env`](./built_images.env) | Pinned Artifact Registry image digests for `atelet`, the 4 `OCILayerService` adapters, the 10-layer SWE-bench OCI image (`SWEBENCH_IMG`), and the Nydus RAFS v6 image (`SWEBENCH_NYDUS_IMG`). |
| [`build_and_prepare_all.sh`](./build_and_prepare_all.sh) | Builds/pushes the 10-layer SWE-bench `matplotlib` image (`crane append`) and invokes `setup_ubuntu_c3_and_index.sh`. |
| [`setup_ubuntu_c3_and_index.sh`](./setup_ubuntu_c3_and_index.sh) | Compiles `bench-runner`, installs ` bench-runner` and `soci` v0.15.0 onto the cluster helper pods, builds/pushes the AWS SOCI v2 zTOC index, and records the Nydus RAFS v6 digest. |
| [`run_full_benchmark_matrix.sh`](./run_full_benchmark_matrix.sh) | Deploys all 4 adapter DaemonSets, provisions `1 CPU / 2 GiB` `WorkerPool`s and `ActorTemplate`s, and executes the complete 45-run benchmark matrix. |
| [`bench-runner/main.go`](./bench-runner/main.go) | Standalone Go CLI that drives `ateapi` (`CreateActorTemplate`, `CreateActor`, `ResumeActor`, `PauseActor`, `DeleteActor`), `atenet` HTTP workload requests, and `OCILayerService.ReleaseImage` cache eviction. |
| [`benchmark_results.jsonl`](./benchmark_results.jsonl) | Structured JSONL output containing all 45 benchmark runs with nanosecond timestamps and latency breakdowns. |
| [`benchmark_evidence.log`](./benchmark_evidence.log) | Detailed execution log including `ateapi` `"Picked worker"` timestamps and per-run adapter / `atelet` logs. |

---

## 2. Cluster & Node Pool Setup

The benchmark runs on a GKE cluster (`substrate-stream-test`, `us-central1-a`) with three nodes so that all 5 configurations are evaluated on identical hardware (`c3-standard-22`, 22 vCPUs, 88 GiB RAM, `300 GB pd-ssd` boot disk) and identical `1 CPU / 2 GiB` `WorkerPool` and `ActorTemplate` sizing:

1. **COS Benchmark Node (`COS_NODE`)**: `gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62`
   - Runs **`riptide`**, **`soci`**, **`spegel`**, and **`baseline`**.
   - Labeled with `bench-node=c3-cos` and hosts `WorkerPool` `wp-c3-cos` (`1 CPU`, `2 GiB`).
2. **Ubuntu 24.04 Benchmark Node (`UBUNTU_NODE`)**: `gke-substrate-stream-c3-std22-ubuntu--5a335c0d-5h2p`
   - Runs **`nydus-dragonfly`** (requires Linux kernel `CONFIG_EROFS_FS_ONDEMAND=y` and `fscache`, available on GKE Ubuntu `6.8.0-1046-gke` with `modprobe erofs && modprobe cachefiles`).
   - Labeled with `bench-node=c3-ubuntu` and hosts `WorkerPool` `wp-c3-ubuntu` (`1 CPU`, `2 GiB`).
3. **P2P Seed Peer Node (`PEER_NODE`)**: `gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc`
   - Pre-warms and seeds OCI/Nydus blobs over in-cluster P2P for **Spegel** (Kademlia DHT on `:5000`/`:5001`) and **Dragonfly** (`dfdaemon` on `:65001`).

### Provisioning `c3-standard-22` Node Pools (if starting from scratch)
```bash
# 1. Create the COS c3-standard-22 node pool
gcloud container node-pools create c3-std22-cos-pool \
  --cluster=substrate-stream-test \
  --zone=us-central1-a \
  --machine-type=c3-standard-22 \
  --disk-type=pd-ssd \
  --disk-size=300 \
  --image-type=COS_CONTAINERD \
  --num-nodes=1 \
  --scopes=https://www.googleapis.com/auth/cloud-platform

# 2. Create the Ubuntu 24.04 c3-standard-22 node pool (for EROFS fscache)
gcloud container node-pools create c3-std22-ubuntu-pool \
  --cluster=substrate-stream-test \
  --zone=us-central1-a \
  --machine-type=c3-standard-22 \
  --disk-type=pd-ssd \
  --disk-size=300 \
  --image-type=UBUNTU_CONTAINERD \
  --num-nodes=1 \
  --scopes=https://www.googleapis.com/auth/cloud-platform
```

### Deploying Host Helper Pods (`bench-helper-*`)
Each node runs a privileged helper pod `bench-helper-<node-suffix>` in namespace `ate-system` with `hostNetwork: true`, `hostPID: true`, and `/home/kubernetes/bin/bench-runner` installed on the host. This allows executing `bench-runner` inside the cluster network and running `nsenter -t 1 -m -u -i -n` for host cache eviction and `OCILayerService` socket operations.

---

## 3. Step-by-Step Image & Index Preparation

### Step 3.1: Build `atelet` and `OCILayerService` Adapter Images
From the repository root, build and push `atelet` and all four adapter binaries using `ko`:
```bash
export KO_DOCKER_REPO="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images"

ATELET_IMG=$(ko build ./cmd/atelet)
RIPTIDE_IMG=$(ko build ./cmd/ocilayer-riptide)
SOCI_IMG=$(ko build ./cmd/ocilayer-soci)
NYDUS_DF_IMG=$(ko build ./cmd/ocilayer-nydus-dragonfly)
SPEGEL_IMG=$(ko build ./cmd/ocilayer-spegel)
```
Update the `atelet` DaemonSet image in `ate-system`:
```bash
kubectl -n ate-system set image daemonset/atelet-v19129425 atelet="${ATELET_IMG}"
kubectl -n ate-system rollout status daemonset/atelet-v19129425
```

### Step 3.2: Build the 10-Layer SWE-Bench `matplotlib` Image (`SWEBENCH_IMG`)
Append the SWE-bench `matplotlib` 3.6.0 conda environment and `/testbed` git checkout layer (`swebench_matplotlib_layer.tar.gz`) onto the 9-layer `benchmark-9layer` sandbox base image using `crane` (~1.22 GB compressed / ~2.97 GB unpacked across 10 layers):
```bash
BASE_IMG="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:benchmark-9layer"
TARGET_IMG="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:swebench-matplotlib"

crane append -b "${BASE_IMG}" -f "${SCRATCH}/swebench_matplotlib_layer.tar.gz" -t "${TARGET_IMG}"
TARGET_DIGEST=$(crane digest "${TARGET_IMG}")
export SWEBENCH_IMG="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@${TARGET_DIGEST}"
```

### Step 3.3: Generate and Push the AWS SOCI v2 Index (zTOC)
On the helper node with `soci` v0.15.0 installed (`/home/kubernetes/bin/soci`), pull `SWEBENCH_IMG` into `containerd` (`k8s.io` namespace), generate a SOCI v2 zTOC index for all layers (`--min-layer-size 0`), and push it to Artifact Registry via OCI Referrers:
```bash
TOKEN=$(gcloud auth print-access-token)
kubectl -n ate-system exec "bench-helper-${PEER_NODE##*-}" -- nsenter -t 1 -m -u -i -n -- sh -c "
  export PATH=/home/kubernetes/bin:\${PATH}
  ctr -n k8s.io images pull --user 'oauth2accesstoken:${TOKEN}' '${SWEBENCH_IMG}'
  soci --namespace k8s.io create --min-layer-size 0 '${SWEBENCH_IMG}'
  soci --namespace k8s.io push --user 'oauth2accesstoken:${TOKEN}' '${SWEBENCH_IMG}'
"
```

### Step 3.4: Convert `SWEBENCH_IMG` to Nydus RAFS v6 (`SWEBENCH_NYDUS_IMG`)
Using `nydusify` (`v2.3.0`), convert the 10-layer OCI image into a Nydus RAFS v6 EROFS bootstrap + chunked blob image for the `nydus-dragonfly` adapter:
```bash
NYDUS_TARGET="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:swebench-nydus-v6"
nydusify convert \
  --source "${TARGET_IMG}" \
  --target "${NYDUS_TARGET}" \
  --fs-version 6
NYDUS_DIGEST=$(crane digest "${NYDUS_TARGET}")
export SWEBENCH_NYDUS_IMG="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@${NYDUS_DIGEST}"
```

Or run the automated preparation script directly:
```bash
bash benchmarking/image-streaming/build_and_prepare_all.sh
```

---

## 4. Deploying Adapters, `WorkerPool`s, and `ActorTemplate`s

[`run_full_benchmark_matrix.sh`](./run_full_benchmark_matrix.sh) automates the cluster configuration before starting the 45-run matrix:

1. **Apply All 4 Adapter DaemonSets**:
   Renders and applies:
   - [`cmd/ocilayer-riptide/ocilayer-riptide.yaml`](../../cmd/ocilayer-riptide/ocilayer-riptide.yaml) (`nodeSelector: ate.dev/oci-layer-service: riptide`)
   - [`cmd/ocilayer-soci/ocilayer-soci.yaml`](../../cmd/ocilayer-soci/ocilayer-soci.yaml) (`nodeSelector: ate.dev/oci-layer-service: soci`)
   - [`cmd/ocilayer-nydus-dragonfly/ocilayer-nydus-dragonfly.yaml`](../../cmd/ocilayer-nydus-dragonfly/ocilayer-nydus-dragonfly.yaml) (`nodeSelector: ate.dev/oci-layer-service: nydus-dragonfly`)
   - [`cmd/ocilayer-spegel/ocilayer-spegel.yaml`](../../cmd/ocilayer-spegel/ocilayer-spegel.yaml) (`nodeSelector: ate.dev/oci-layer-service: spegel`)

2. **How Adapter Switching Works**:
   Each adapter DaemonSet selects on the node label `ate.dev/oci-layer-service=<provider>` and binds the Unix domain socket `/run/ate-image-streaming/ocilayer.sock` on the host.
   - Switching adapters on a node only requires relabeling the node:
     ```bash
     kubectl label node "${COS_NODE}" ate.dev/oci-layer-service=soci --overwrite
     ```
   - For **Baseline (`baseline`)**, the node is labeled `ate.dev/oci-layer-service=disabled` and `/run/ate-image-streaming/ocilayer.sock` is removed, causing `atelet` to fall back to its built-in local `internal/imagecache`.

3. **Create `1 CPU / 2 GiB` `WorkerPool`s (`wp-c3-cos` and `wp-c3-ubuntu`)**:
   Both `WorkerPool`s run in namespace `ate-bench-stream` with `replicas: 1` and `1 CPU / 2 GiB` resource requests and limits.

4. **Initialize `ActorTemplate`s and One-Time Golden Snapshots**:
   `bench-runner --mode=ensure-template --wait-golden=true` creates two `1 CPU / 2 GiB` `ActorTemplate`s and waits for `ateapi`/`atelet` to complete the one-time golden checkpoint in GCS:
   - `tmpl-cos-oci` (pinned to `wp-c3-cos`, image `SWEBENCH_IMG`)
   - `tmpl-ubuntu-nydus` (pinned to `wp-c3-ubuntu`, image `SWEBENCH_NYDUS_IMG`)

---

## 5. Benchmark Execution Matrix (`45 Runs`)

### 5.1 Workloads
Every workload is executed via `bench-runner` sending an HTTP POST to `atenet` (`http://<ATENET_IP>:80`) with header `X-Ate-Actor: ate-bench-stream/<actor_name>`:
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

### 5.2 Cache Eviction Between Cold Runs (`wipe_node_cache`)
Before every `cold_boot` and `cold_resume` measurement, `wipe_node_cache` purges all node-local image layer caches and kernel page caches:
```bash
# 1. Purge atelet's local imagecache layers and manifests
rm -rf /var/lib/ateom-gvisor/images/layers/sha256/* /var/lib/ateom-gvisor/images/manifests/sha256/*
mkdir -p /var/lib/ateom-gvisor/images/layers/sha256 /var/lib/ateom-gvisor/images/manifests/sha256

# 2. Invoke OCILayerService.ReleaseImage on the active adapter socket
/home/kubernetes/bin/bench-runner \
  --mode=release-image \
  --socket=/run/ate-image-streaming/ocilayer.sock \
  --image="${IMAGE_REF}"

# 3. For Spegel, remove local Spegel blob/layer caches so layers are fetched over P2P from PEER_NODE
rm -rf /var/lib/ateom-gvisor/streaming/spegel-store/blobs/sha256/* /var/lib/ateom-gvisor/streaming/spegel-store/layers/sha256/*
mkdir -p /var/lib/ateom-gvisor/streaming/spegel-store/blobs/sha256 /var/lib/ateom-gvisor/streaming/spegel-store/layers/sha256

# 4. Flush buffers and drop Linux page/dentry/inode caches
sync
echo 3 > /proc/sys/vm/drop_caches
```

### 5.3 Per-Workload Lifecycle Scenarios
For each `(adapter, workload)` pair:
1. **`cold_boot` (`--mode=create-boot-run`)**:
   - Run `wipe_node_cache`.
   - Call `CreateActor` from template, then `ResumeActor(boot=true)` to restore from the golden snapshot and mount the image layers on demand, then immediately run the workload command.
2. **Suspend (`--mode=suspend --clean-pycache=true`)**:
   - Clean any `.pyc` files written to the sandbox upperdir (`find /testbed/lib/matplotlib -name "*.pyc" -delete`), then call `PauseActor` to checkpoint the running gVisor sandbox to GCS.
3. **`warm_resume` (`--mode=resume-and-run`)**:
   - With read-only image layers warm in the node cache, call `ResumeActor(boot=false)` from the actor's GCS checkpoint and run the workload command.
4. **Suspend (`--mode=suspend --clean-pycache=true`)**:
   - Call `PauseActor` to checkpoint the sandbox to GCS again.
5. **`cold_resume` (`--mode=resume-and-run`)**:
   - Run `wipe_node_cache` to evict all node-local read-only layers and kernel page caches (simulating cross-node actor migration to a node without cached layers).
   - Call `ResumeActor(boot=false)` and run the workload command.
6. **Cleanup (`--mode=delete`)**:
   - Delete the actor via `DeleteActor`.

---

## 6. Metrics & Timestamp Definitions

Inside [`bench-runner/main.go`](./bench-runner/main.go), the workload command is wrapped inside the sandbox container with nanosecond timestamps (`date +%s%N` before and after the command):

- **`t_start`**: `picked_worker_ns` parsed from `ateapi`'s `"Picked worker"` log entry (falling back to `resume_rpc_start_ns` if log rotation/buffering omits the line).
- **`cmd_t0_ns`**: Nanosecond timestamp inside the sandbox container immediately before the workload command executes.
- **`cmd_t1_ns`**: Nanosecond timestamp inside the sandbox container immediately after the workload command completes.
- **`resume_rpc_dur_ms`**: `(resume_rpc_done_ns - resume_rpc_start_ns) / 1e6` — Duration of the `ResumeActor` gRPC call (including `OCILayerService.PrepareImage` and gVisor container boot/restore).
- **`ttce_ms` (Time To Command Execution)**: `(cmd_t0_ns - t_start) / 1e6` — Time from worker selection / resume start until the first instruction of the user command runs inside the container.
- **`workload_run_ms`**: `(cmd_t1_ns - cmd_t0_ns) / 1e6` — Execution duration of the command inside the container (including on-demand FUSE / EROFS page faults).
- **`e2e_ms` (End-to-End Latency)**: `(cmd_t1_ns - t_start) / 1e6` (`ttce_ms + workload_run_ms`).

---

## 7. Running the Full Suite

```bash
# Step 1: Prepare images, SOCI index, Nydus RAFS v6 image, and bench-runner binary
bash benchmarking/image-streaming/build_and_prepare_all.sh

# Step 2: Execute the 45-run benchmark matrix across all 5 adapters
bash benchmarking/image-streaming/run_full_benchmark_matrix.sh
```
