# OCILayerService Live E2E Functional Test Guide & Evidence (`kuiyue-gke-dev`)

This document provides the step-by-step commands, reusable scripts, root-cause fixes, and live cluster evidence for running a clean-slate end-to-end (`E2E`) functional test of all `OCILayerService` image-streaming adapters (`nydus-dragonfly`, `riptide`, `soci`, `spegel`, and `baseline`) on the `substrate-stream-test` GKE cluster in `kuiyue-gke-dev`.

---

## 1. Root Causes & Code Fixes Verified

### Issue 1: Worker Pod Cannot See Nydus Mount (`no such file or directory` on `.../layer-0/fs`)
- **Symptom:**
  ```text
  while setting overlay lowerdir+="/var/lib/ateom-gvisor/streaming/nydus-dragonfly/<image>/layer-0/fs": no such file or directory
  ```
- **Root Cause:**
  `containerd-nydus-grpc` mounts RAFS v6 EROFS filesystems under `<SnapshotterRoot>/mnt/<snapshot_id>` (previously `/var/lib/containerd-nydus/mnt/<id>`), and the adapter creates a symlink at `/var/lib/ateom-gvisor/streaming/nydus-dragonfly/<image>/layer-0/fs -> /var/lib/containerd-nydus/mnt/<id>`. Because `ateom` runs inside the worker pod, which mounts `/var/lib/ateom-gvisor` with `HostToContainer` mount propagation—and did **not** mount `/var/lib/containerd-nydus`—the symlink target did not exist inside the worker pod's mount namespace.
- **Fix:**
  Moved `DefaultSnapshotterRoot` from `/var/lib/containerd-nydus` to `/var/lib/ateom-gvisor/containerd-nydus` across:
  - [`cmd/ocilayer-nydus-dragonfly/driver/driver.go`](../../cmd/ocilayer-nydus-dragonfly/driver/driver.go)
  - [`cmd/ocilayer-nydus-dragonfly/ocilayer-nydus-dragonfly.yaml`](../../cmd/ocilayer-nydus-dragonfly/ocilayer-nydus-dragonfly.yaml)
  - [`cmd/ocilayer-nydus/driver/driver.go`](../../cmd/ocilayer-nydus/driver/driver.go)
  - [`cmd/ocilayer-nydus/ocilayer-nydus.yaml`](../../cmd/ocilayer-nydus/ocilayer-nydus.yaml)

  Because `/var/lib/ateom-gvisor` is mounted on the worker pod with `mountPropagation: HostToContainer` (and `Bidirectional` on the adapter DaemonSet), EROFS mounts under `/var/lib/ateom-gvisor/containerd-nydus/mnt/<id>` propagate directly into the worker pod and are resolvable by `ateom` when constructing the overlayfs `lowerdir`.

### Issue 2: `nydusd` Receives `403 Forbidden` on Artifact Registry `/v2/token` Requests
- **Symptom:**
  `nydusd` fails to fetch chunks from `us-central1-docker.pkg.dev` with HTTP `403 Forbidden` on `/v2/token?scope=...&service=us-central1-docker.pkg.dev`.
- **Root Cause:**
  In [`cmd/ocilayer-nydus-dragonfly/ocilayer-nydus-dragonfly.yaml`](../../cmd/ocilayer-nydus-dragonfly/ocilayer-nydus-dragonfly.yaml), `dfdaemon` configured `proxy.rules` for `.*/blobs/sha256/.*`, and `nydusd` used `proxy.url = "http://127.0.0.1:4001"` with `proxy.fallback = true`. When `nydusd` requested an OAuth2 registry token from `https://us-central1-docker.pkg.dev/v2/token?...` before fetching blobs:
  1. `nydusd` sent a plain HTTP `GET http://us-central1-docker.pkg.dev/v2/token?...` request to `dfdaemon:4001`.
  2. Because `/v2/token` did not match `.*/blobs/sha256/.*`, `dfdaemon` forwarded it as plain **HTTP** (`http://us-central1-docker.pkg.dev/v2/token`) instead of **HTTPS**, or without `Authorization` headers when `dfdaemon` intercepted blobs, causing Google Artifact Registry to reject it with `403 Forbidden`.
- **Fix:**
  1. Added `proxy.registryMirror` (`url: https://us-central1-docker.pkg.dev`, `dynamic: true`) and HTTPS `proxy.rules` with `useHTTPS: true` for both `/v2/token.*` and `.*/(manifests|blobs)/.*` in [`cmd/ocilayer-nydus-dragonfly/ocilayer-nydus-dragonfly.yaml`](../../cmd/ocilayer-nydus-dragonfly/ocilayer-nydus-dragonfly.yaml).
  2. Updated `nydus-snapshotter` in [`cmd/ocilayer-nydus-dragonfly/ocilayer-nydus-dragonfly.yaml`](../../cmd/ocilayer-nydus-dragonfly/ocilayer-nydus-dragonfly.yaml) and [`cmd/ocilayer-nydus/ocilayer-nydus.yaml`](../../cmd/ocilayer-nydus/ocilayer-nydus.yaml) to periodically refresh a GCP Artifact Registry OAuth2 access token from the GKE metadata server into `/root/.docker/config.json` and pass `--shared-daemon=false` so each image mount receives fresh credentials and routes chunk fetches through Dragonfly P2P (`http://127.0.0.1:4001`).
  3. Replaced `pkill -f "/home/kubernetes/bin/nydusd"` with `pkill -x nydusd` and `pkill -x containerd-nydus-grpc` so the `sh -c` startup script does not match its own command line (`exit code 143`).

---

## 2. Reusable Scripts for Teammates

| Script | Purpose |
|---|---|
| [`benchmarking/image-streaming/swap_ocilayer_adapter.sh`](swap_ocilayer_adapter.sh) | Builds and deploys all 4 `OCILayerService` adapter DaemonSets from scratch (`--deploy-all-from-scratch`) and swaps specific GKE nodes between `nydus-dragonfly`, `riptide`, `soci`, `spegel`, and `baseline` (`--adapter=<name> --nodes=<node1,node2>`). |
| [`benchmarking/image-streaming/run_e2e_functional_test.sh`](run_e2e_functional_test.sh) | Complete clean-slate E2E functional test script: tears down all existing DaemonSets, Deployments, Services, ConfigMaps, WorkerPools, and host state; deploys `ate-system`, `WorkerPool`s, and all 4 adapter DaemonSets from scratch; and runs E2E functional tests across all 5 arms while recording logs and JSONL results. |

### Quick Usage of `swap_ocilayer_adapter.sh`

```bash
# 1. Tear down any existing adapter DaemonSets and build/deploy all 4 from scratch:
./benchmarking/image-streaming/swap_ocilayer_adapter.sh --deploy-all-from-scratch

# 2. Swap Ubuntu EROFS nodes to Nydus + Dragonfly P2P:
./benchmarking/image-streaming/swap_ocilayer_adapter.sh \
  --adapter=nydus-dragonfly \
  --nodes=gke-substrate-stream-c3-std22-ubuntu--5a335c0d-5h2p,gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc

# 3. Swap COS node to Riptide / SOCI / Spegel / Baseline:
./benchmarking/image-streaming/swap_ocilayer_adapter.sh \
  --adapter=riptide \
  --nodes=gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62
```

---

## 3. Step-by-Step Commands (Clean-Slate Live Cluster Run)

To run the entire clean-slate E2E functional test in one command:

```bash
./benchmarking/image-streaming/run_e2e_functional_test.sh
```

Below are the exact step-by-step commands executed by [`run_e2e_functional_test.sh`](run_e2e_functional_test.sh):

### Step 1: Tear Down Existing DaemonSets, Control Plane, WorkerPools, and Host State
```bash
source .ate-dev-env.sh
source benchmarking/image-streaming/built_images.env

COS_NODE="gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62"
UBUNTU_NODE="gke-substrate-stream-c3-std22-ubuntu--5a335c0d-5h2p"
PEER_NODE="gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc"

# Delete test namespaces, existing adapter DaemonSets, atelet, control-plane Deployments/Services, and Dragonfly stack
kubectl delete namespace ate-bench-stream ate-bench-accel --ignore-not-found=true
for ds in $(kubectl -n ate-system get ds -o name | grep -E 'ate-ocilayer-|ate-accelerator-|atelet'); do
  kubectl -n ate-system delete "${ds}" --ignore-not-found=true
done
kubectl -n ate-system delete deployment atenet-egress atenet-router ate-controller ate-api-server \
  dragonfly-manager dragonfly-scheduler dragonfly-mysql dragonfly-redis --ignore-not-found=true
kubectl -n ate-system delete svc atenet-router atenet-egress \
  dragonfly-manager dragonfly-scheduler dragonfly-mysql dragonfly-redis --ignore-not-found=true
kubectl -n ate-system delete configmap ate-api-authentication --ignore-not-found=true
# Remove any stale NetworkPolicy objects left over from oci-image-accelerator deployments
kubectl -n ate-system delete networkpolicy --all --ignore-not-found=true

# Disable adapter node labels and clean host state on all test nodes
kubectl label nodes --all ate.dev/oci-layer-service=disabled --overwrite
kubectl label nodes --all ate.dev/image-accelerator- 2>/dev/null || true
for NODE in "${COS_NODE}" "${UBUNTU_NODE}" "${PEER_NODE}"; do
  POD="bench-helper-${NODE##*-}"
  kubectl -n ate-system exec "${POD}" -- nsenter -t 1 -m -u -i -n -- sh -c '
    pkill -x nydusd 2>/dev/null || true
    pkill -x containerd-nydus-grpc 2>/dev/null || true
    grep -E "containerd-nydus|nydus-dragonfly" /proc/mounts 2>/dev/null | awk "{print \$2}" | sort -r | xargs -r umount -l 2>/dev/null || true
    rm -rf /var/lib/containerd-nydus /var/lib/ateom-gvisor/containerd-nydus /var/lib/ateom-gvisor/streaming/* /run/ate-image-streaming/* /run/containerd-nydus/*
    rm -f /var/lib/ateom-gvisor/image-cache/manifests/sha256/c8c61a46* /var/lib/ateom-gvisor/image-cache/manifests/sha256/d89906a9* 2>/dev/null || true
    for d in /var/lib/ateom-gvisor/image-cache/layers/sha256/*; do
      if [ -d "\$d" ] && [ ! -f "\$d/fs/pause" ]; then
        chmod -R u+w "\$d" 2>/dev/null || true
        rm -rf "\$d"
      fi
    done
    mkdir -p /var/lib/ateom-gvisor/image-cache/layers/sha256 /var/lib/ateom-gvisor/image-cache/manifests/sha256 /run/ate-image-streaming
  '
done
```

### Step 2: Build and Install `bench-runner` on Helper Pods
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/bench-runner ./benchmarking/image-streaming/bench-runner
for NODE in "${COS_NODE}" "${UBUNTU_NODE}" "${PEER_NODE}"; do
  POD="bench-helper-${NODE##*-}"
  kubectl -n ate-system cp bin/bench-runner "${POD}:/home/kubernetes/bin/bench-runner.new"
  kubectl -n ate-system exec "${POD}" -- nsenter -t 1 -m -u -i -n -- sh -c \
    "chmod +x /home/kubernetes/bin/bench-runner.new && mv /home/kubernetes/bin/bench-runner.new /home/kubernetes/bin/bench-runner"
done
```

### Step 3: Fresh Deploy of `ate-system` Control Plane & `atelet`
```bash
./hack/install-ate.sh --deploy-ate-system --rollout-timeout=300s
```

### Step 4: Build Fresh `ateom-gvisor` Worker Image & Deploy `WorkerPool`s
```bash
SUBSTRATE_VERSION="$(git rev-parse --short=8 HEAD)"
WORKER_IMAGE="$(./hack/run-tool.sh ko build ./cmd/ateom-gvisor --ldflags="-X=github.com/agent-substrate/substrate/internal/version.Version=${SUBSTRATE_VERSION}")"

kubectl label node "${COS_NODE}" bench-node=c3-cos "ate.dev/substrate-version=${SUBSTRATE_VERSION}" --overwrite
kubectl label node "${UBUNTU_NODE}" bench-node=c3-ubuntu "ate.dev/substrate-version=${SUBSTRATE_VERSION}" --overwrite
kubectl label node "${PEER_NODE}" "ate.dev/substrate-version=${SUBSTRATE_VERSION}" --overwrite
```
Apply the `wp-c3-cos` and `wp-c3-ubuntu` `WorkerPool` resources in namespace `ate-bench-stream` and verify that the worker pod mounts `/var/lib/ateom-gvisor` with `mountPropagation: HostToContainer` and `/run/ate-image-streaming`.

### Step 5: Build & Deploy All 4 `OCILayerService` Adapter DaemonSets from Scratch
```bash
./benchmarking/image-streaming/swap_ocilayer_adapter.sh --deploy-all-from-scratch
kubectl -n ate-system get ds,deploy -o wide
```

### Step 6: Nydus + Dragonfly P2P Functional Verification on Ubuntu Node
```bash
# Activate nydus-dragonfly on the Ubuntu worker node and peer node
./benchmarking/image-streaming/swap_ocilayer_adapter.sh --adapter=nydus-dragonfly --nodes="${UBUNTU_NODE},${PEER_NODE}"

# Seed the Nydus RAFS v6 image on the peer node via OCILayerService.PrepareImage
kubectl -n ate-system exec "bench-helper-${PEER_NODE##*-}" -- nsenter -t 1 -m -u -i -n -- \
  /home/kubernetes/bin/bench-runner \
  --mode=prepare-image \
  --socket=/run/ate-image-streaming/ocilayer.sock \
  --image="${SWEBENCH_NYDUS_IMG}"

# Ensure ActorTemplate tmpl-ubuntu-nydus and wait for golden snapshot
ATE_TOKEN="$(kubectl -n ate-system create token ate-client --audience=api.ate-system.svc --duration=2h)"
ATEAPI_IP="$(kubectl -n ate-system get pods -l app=ate-api-server -o jsonpath='{.items[0].status.podIP}')"
ATENET_ADDR="$(kubectl -n ate-system get pods -l app=atenet-router -o jsonpath='{.items[0].status.podIP}'):8080"

kubectl -n ate-system exec "bench-helper-${UBUNTU_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
  --mode=ensure-template \
  --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream \
  --template=tmpl-ubuntu-nydus --pool-label=bench-ubuntu --image="${SWEBENCH_NYDUS_IMG}" --wait-golden=true

# Run cold_boot, inspect EROFS mount inside worker pod, warm_resume, and cold_resume
kubectl -n ate-system exec "bench-helper-${UBUNTU_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
  --mode=create-boot-run --adapter=nydus-dragonfly --scenario=cold_boot --workload=functional_check \
  --cmd='cat -- "/testbed/lib/matplotlib/pyplot.py" | head -n 5' \
  --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream \
  --template=tmpl-ubuntu-nydus --actor=e2e-nydus-dragonfly-actor
```

### Step 7: Riptide, SOCI, Spegel, and Baseline Functional Verification on COS Node
```bash
# Swap COS node through riptide, soci, spegel, and baseline and execute cold_boot + cold_resume
# Note: Before each cold test, evict /var/lib/ateom-gvisor/image-cache (ateompath.ImageCacheDir)
# and call OCILayerService.ReleaseImage so atelet does not short-circuit on warm unpacked layers.
for ADAPTER in riptide soci spegel baseline; do
  ./benchmarking/image-streaming/swap_ocilayer_adapter.sh --adapter="${ADAPTER}" --nodes="${COS_NODE}"
  # ... evict /var/lib/ateom-gvisor/image-cache and run cold_boot + cold_resume via bench-runner ...
done
```

---

## 4. Live E2E Functional Test Results & Evidence (`substrate-stream-test`)

### 4.1 Summary of All 11 Live E2E Functional Test Runs (Cold Cache Evicted)

> **Important Note on Cold Cache Eviction:** `atelet` stores unpacked OCI layers under `/var/lib/ateom-gvisor/image-cache` (`ateompath.ImageCacheDir` in [`internal/ateompath/ateompath.go`](../../internal/ateompath/ateompath.go)) and checks `imageCache.CachedImage(ctx, ref)` **before** invoking `OCILayerService.PrepareImage` ([`cmd/atelet/oci.go`](../../cmd/atelet/oci.go)). Evicting `/var/lib/ateom-gvisor/image-cache` (plus calling `OCILayerService.ReleaseImage` and `drop_caches=3`) before every cold test ensures every arm runs genuinely cold.

| Adapter | Node | Scenario | Worker Pod | PrepareImage (ms) | Resume RPC (ms) | Client TTCE (ms) | Workload Run (ms) | Client E2E (ms) | Speedup vs Baseline | Exit Code |
|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|
| **`nydus-dragonfly`** | `5h2p` (Ubuntu) | `cold_boot` | `wp-c3-ubuntu-7686d7fdb6-xvvkq` | 671.39 | 599.55 | 745.87 | 10.61 | **756.49** | **29.2x** | `0` |
| **`nydus-dragonfly`** | `5h2p` (Ubuntu) | `warm_resume` | `wp-c3-ubuntu-7686d7fdb6-xvvkq` | 4.65 | 298.34 | 315.89 | 7.86 | **323.75** | — | `0` |
| **`nydus-dragonfly`** | `5h2p` (Ubuntu) | `cold_resume` | `wp-c3-ubuntu-7686d7fdb6-xvvkq` | 4.82 | 690.29 | 751.51 | 11.12 | **762.63** | **30.6x** | `0` |
| **`riptide`** | `wj62` (COS) | `cold_boot` | `wp-c3-cos-5bbd6758-cv9fr` | 371.54 | 1,246.30 | 1,526.83 | 12.55 | **1,539.38** | **14.4x** | `0` |
| **`riptide`** | `wj62` (COS) | `cold_resume` | `wp-c3-cos-5bbd6758-cv9fr` | 25.67 | 688.80 | 760.11 | 12.83 | **772.94** | **30.2x** | `0` |
| **`soci`** | `wj62` (COS) | `cold_boot` | `wp-c3-cos-5bbd6758-cv9fr` | 3,382.05 | 4,795.77 | 5,072.62 | 124.70 | **5,197.31** | **4.3x** | `0` |
| **`soci`** | `wj62` (COS) | `cold_resume` | `wp-c3-cos-5bbd6758-cv9fr` | 24.27 | 658.29 | 731.73 | 19.70 | **751.43** | **31.0x** | `0` |
| **`spegel`** | `wj62` (COS) | `cold_boot` | `wp-c3-cos-5bbd6758-cv9fr` | 16,668.57 | 19,128.59 | 19,409.89 | 8.15 | **19,418.04** | **1.14x** | `0` |
| **`spegel`** | `wj62` (COS) | `cold_resume` | `wp-c3-cos-5bbd6758-cv9fr` | 14,912.40 | 17,603.40 | 17,675.08 | 8.27 | **17,683.34** | **1.32x** | `0` |
| **`baseline`** | `wj62` (COS) | `cold_boot` | `wp-c3-cos-5bbd6758-cv9fr` | *(AR pull + unpack)* | 21,950.25 | 22,096.86 | 7.95 | **22,104.81** | **1.0x** | `0` |
| **`baseline`** | `wj62` (COS) | `cold_resume` | `wp-c3-cos-5bbd6758-cv9fr` | *(AR pull + unpack)* | 23,245.55 | 23,319.78 | 10.80 | **23,330.59** | **1.0x** | `0` |

### 4.2 Worker Pod Mount Namespace & Nydus Snapshotter Evidence

From [`e2e_functional_test_evidence.log`](e2e_functional_test_evidence.log):

```text
--- [nydus-dragonfly] 2. Verifying Nydus EROFS Mounts Inside Worker Pod (wp-c3-ubuntu-7686d7fdb6-xvvkq) ---
Inspecting mount namespace of worker pod process ateom-gvisor (host PID=2810249):
=== Symlink target of layer-0/fs inside worker pod (/proc/2810249/root) ===
total 8
drwxr-xr-x 2 root root 4096 Oct 21 20:28 .
drwxr-xr-x 3 root root 4096 Oct 21 20:28 ..
-rw------- 1 root root    0 Oct 21 20:28 finalized
lrwxrwxrwx 1 root root   55 Oct 21 20:28 fs -> /var/lib/ateom-gvisor/containerd-nydus/snapshots/10/mnt
=== Reading EROFS mount at /var/lib/ateom-gvisor/containerd-nydus/snapshots/*/mnt inside worker pod ===
total 112
drwxr-xr-x 20 root   root   472 Jan  1  1970 .
...
drwxr-xr-x  3 591210 89939   42 Oct 16 01:19 testbed
drwxrwxrwt  3 root   root    54 Sep 24 18:50 tmp
drwxr-xr-x 14 root   root   229 Sep  1 20:33 usr
dr-xr-xr-x 12 root   root   204 Jan  1  1970 var
=== Active EROFS / containerd-nydus mounts in worker pod mountinfo ===
3331 3543 0:448 / /var/lib/ateom-gvisor/containerd-nydus/snapshots/10/mnt rw,relatime master:1573 - erofs erofs ro,user_xattr,acl,cache_strategy=readaround,fsid=5fff396b3fb7a4acb15dd26e7249657c919c352e2cd75c89bbda3c0cd5a17753
```

And from the `nydus-snapshotter` container logs:
```text
time="2026-10-21T20:16:04.209366256Z" level=info msg="Mount erofs to /var/lib/ateom-gvisor/containerd-nydus/snapshots/10/mnt with options fsid=5fff396b3fb7a4acb15dd26e7249657c919c352e2cd75c89bbda3c0cd5a17753"
[2026-10-21 20:16:04.214638 +00:00] INFO backend config: ConnectionConfig { proxy: ProxyConfig { url: "http://127.0.0.1:65001", ping_url: "http://127.0.0.1:40901/server/ping", fallback: false, check_interval: 5, use_http: true, check_pause_elapsed: 300 }, mirrors: [], skip_verify: false, timeout: 10, connect_timeout: 10, retry_limit: 2 }
[2026-10-21 20:16:04.229501 +00:00] INFO Refresh token thread started.
[2026-10-21 20:16:04.229676 +00:00] INFO Will replace backend's URL's scheme with http
time="2026-10-21T20:16:04.236831159Z" level=info msg="remote mount options [lowerdir=/var/lib/ateom-gvisor/containerd-nydus/snapshots/10/mnt:/var/lib/ateom-gvisor/containerd-nydus/snapshots/11/fs]"
VERIFIED: 0 occurrences of HTTP 403 in nydus-snapshotter / nydusd logs!
```

---

## 5. Collected Evidence Files

- **Full Step-by-Step Execution Log:** [`benchmarking/image-streaming/e2e_functional_test_evidence.log`](e2e_functional_test_evidence.log)
- **Structured JSONL Latency & Status Results:** [`benchmarking/image-streaming/e2e_functional_test_results.jsonl`](e2e_functional_test_results.jsonl)

