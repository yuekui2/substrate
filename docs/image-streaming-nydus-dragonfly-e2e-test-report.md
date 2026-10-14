# Cross-Node Nydus RAFS v6 + Dragonfly v2 P2P (Kernel EROFS + `fscache`) Live Multi-Node GKE E2E & Performance Test Report

- **Date:** 2026-10-14
- **GCP Project:** `kuiyue-gke-dev`
- **GKE Cluster:** `substrate-stream-test` (`us-central1-a`)
- **Multi-Node EROFS Test Pool (`erofs-ubuntu-pool`):**
  - **Node A (Peer Seed / First Puller):** `gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc` (`10.1.15.220`, `c3-standard-4`, 4 vCPUs, 16 GiB RAM, `UBUNTU_CONTAINERD`, Ubuntu 24.04.5 LTS, Kernel `6.8.0-1062-gke`)
  - **Node B (Cross-Node P2P Receiver):** `gke-substrate-stream-erofs-ubuntu-poo-ea32e702-49x6` (`10.1.15.235`, `c3-standard-4`, 4 vCPUs, 16 GiB RAM, `UBUNTU_CONTAINERD`, Ubuntu 24.04.5 LTS, Kernel `6.8.0-1062-gke`)
- **Negative Test Node (GKE COS):** `gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft` (`10.1.15.221`, `COS_CONTAINERD`, Kernel `6.6.119+`)
- **Components Under Test:**
  - Adapter: [`cmd/ocilayer-nydus-dragonfly`](../cmd/ocilayer-nydus-dragonfly) (`provider="nydus-dragonfly"`, `--setup-image-streaming=nydus-dragonfly`)
  - Manifest: [`cmd/ocilayer-nydus-dragonfly/ocilayer-nydus-dragonfly.yaml`](../cmd/ocilayer-nydus-dragonfly/ocilayer-nydus-dragonfly.yaml)
  - Control Plane (`ate-system`): `dragonfly-mysql` (`mysql:8.0`), `dragonfly-redis` (`redis:7-alpine`), `dragonfly-manager` (`dragonflyoss/manager:v2.1.49`), `dragonfly-scheduler` (`dragonflyoss/scheduler:v2.1.49`)
  - Per-Node DaemonSet (`ate-ocilayer-nydus-dragonfly`): `dfdaemon` (`dragonflyoss/dfdaemon:v2.1.49`), `nydus-snapshotter` (`containerd-nydus-grpc v0.15.0` + `nydusd v2.3.0` in singleton `fscache` mode), and `ocilayer-nydus-dragonfly`

---

## 1. Executive Summary

We deployed and benchmarked the full cross-node **Nydus RAFS v6 + Dragonfly v2 P2P** `OCILayerService` stack ([`cmd/ocilayer-nydus-dragonfly`](../cmd/ocilayer-nydus-dragonfly)) across a 2-node Ubuntu EROFS GKE node pool (`fkxc` and `49x6`) alongside a GKE COS negative-test node (`bpft`).

Unlike the single-node [`cmd/ocilayer-nydus`](../cmd/ocilayer-nydus) adapter (`--setup-image-streaming=nydus`), which pairs `nydusd` with a node-local HTTP Range piece cache, [`cmd/ocilayer-nydus-dragonfly`](../cmd/ocilayer-nydus-dragonfly) (`--setup-image-streaming=nydus-dragonfly`) deploys the full production Dragonfly v2 control plane (`dragonfly-manager`, `dragonfly-scheduler`, `dragonfly-mysql`, `dragonfly-redis`) and per-node `dfdaemon` P2P peers (`:65001` HTTP proxy, `:40901` health check, `:65000` peer gRPC, `:65002` piece upload).

### Key Multi-Node Findings

1. **Cross-Node P2P Piece Streaming Eliminates Cold Registry Bottlenecks (`4.21x` to `13.44x` Faster Cold Reads on Node B):**
   - When Node A (`fkxc`, `10.1.15.220`) cold-reads Nydus RAFS v6 chunks from Google Artifact Registry (GAR), its `dfdaemon` caches the 4 MiB pieces in `/var/lib/dragonfly` and registers as an active peer with `dragonfly-scheduler` (`10.96.1.32:8002`).
   - When Node B (`49x6`, `10.1.15.235`) subsequently mounts the same image with a **completely cold local `fscache`** and **cold page cache**, `dragonfly-scheduler` directs Node B's `dfdaemon` to pull pieces directly from Node A (`10.1.15.220:65000` / `10.1.15.220:65002`) over the VPC network:
     - **Cold `PrepareLayers` latency:** drops from **`1081.49 ms`** (Node A cold from GAR) to **`831.73 ms`** (Node B cross-node P2P) — **`23.1%` faster**.
     - **50-file demand-paging `mean` latency:** drops from **`285.18 ms`** (Node A cold from GAR) to **`67.66 ms`** (Node B cross-node P2P from Node A) — **`4.21x` faster**.
     - **50-file demand-paging `p90` latency:** drops from **`561.07 ms`** to **`85.34 ms`** — **`6.57x` faster**.
     - **50-file demand-paging `p99` / `max` latency:** drops from **`7014.07 ms`** to **`521.98 ms`** — **`13.44x` faster**.
2. **Strict Kernel EROFS (`0xE0F5E1E2`) + `fscache` Enforcement (`fusedev` Disabled):**
   - On GKE COS (`bpft`, Kernel `6.6.119+` where `CONFIG_EROFS_FS` is disabled), `ocilayer-nydus-dragonfly` exits immediately with code `1` and an explicit diagnostic message instead of silently degrading to userspace FUSE (`fusedev`).
   - On both Ubuntu 24.04 GKE nodes (`fkxc` and `49x6`, Kernel `6.8.0-1062-gke`), `ocilayer-nydus-dragonfly` verifies `statfs.Type == 0xE0F5E1E2` (`EROFSSuperMagic`) on every mounted snapshot before exposing `layer-0/fs` to `ateom`.
3. **Zero Host `containerd` Dependency & Fast Decline for Unconverted OCI Images:**
   - Standard unconverted OCI images (`image-a:latest`) are declined during manifest inspection with `codes.FailedPrecondition` (`ocilayer.ErrNotSupported`) and cached in `declineTTL`, returning `canProvide: false` in **`1.263 ms`** on subsequent probes.
4. **Critical Dragonfly `v2.1.49` + GAR Production Configuration Requirements:**
   - Both `- regx: blobs/sha256.*` and `- regx: us-central1-docker\.pkg\.dev` in `dfget.yaml` (`proxy.proxies`) **must** set `useHTTPS: true`. Because `nydusd` (`use_http: true`) rewrites blob URLs to `http://us-central1-docker.pkg.dev/v2/.../blobs/sha256:...` when forwarding through `dfdaemon` (`127.0.0.1:65001`), omitting `useHTTPS: true` on the `blobs/sha256.*` rule causes `dfdaemon` to contact GAR on port 80 (`http://`), which Google's frontend rejects with `403 Forbidden`.
   - Health probes from `nydusd` (`proxy.ping_url`) and `ocilayer-nydus-dragonfly` (`PingDragonflyProxy`) should target `dfdaemon`'s dedicated health listener (`http://127.0.0.1:40901/server/ping`) first so `/server/ping` is not proxied to the upstream registry mirror.

---

## 2. Multi-Node Cluster Topology & Pod Status

```text
NAME                                  READY   STATUS    RESTARTS   AGE     IP            NODE
ate-ocilayer-nydus-dragonfly-mhdj8    3/3     Running   0          5m      10.1.15.220   gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc
ate-ocilayer-nydus-dragonfly-b59bn    3/3     Running   0          5m      10.1.15.235   gke-substrate-stream-erofs-ubuntu-poo-ea32e702-49x6
dragonfly-manager-77d499ccc7-n58z4    1/1     Running   0          108m    10.96.1.31    gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft
dragonfly-mysql-77cd86849d-f4kq8      1/1     Running   0          4h      10.96.1.26    gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft
dragonfly-redis-5b94b77f76-mlpmc      1/1     Running   0          4h      10.96.1.25    gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft
dragonfly-scheduler-8c6b4587c-fjnlc   1/1     Running   0          108m    10.96.1.32    gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft
```

```mermaid
flowchart TB
    subgraph ControlPlane["ate-system Control Plane (bpft)"]
        MySQL["dragonfly-mysql (:3306)"]
        Redis["dragonfly-redis (:6379)"]
        Manager["dragonfly-manager (:8080 / :65003)"]
        Scheduler["dragonfly-scheduler (10.96.1.32:8002)"]
        Manager --> MySQL
        Manager --> Redis
        Scheduler --> Manager
    end

    subgraph NodeA["Node A: fkxc (10.1.15.220) — Ubuntu 24.04 EROFS"]
        OciA["ocilayer-nydus-dragonfly"]
        SnapA["nydus-snapshotter + nydusd (fscache)"]
        DfA["dfdaemon Peer A (:65001 / :65000 / :65002)"]
        ErofsA["Kernel EROFS (0xE0F5E1E2) + /dev/cachefiles"]
        OciA --> SnapA
        SnapA --> ErofsA
        SnapA -->|"HTTP Range (:65001)"| DfA
    end

    subgraph NodeB["Node B: 49x6 (10.1.15.235) — Ubuntu 24.04 EROFS"]
        OciB["ocilayer-nydus-dragonfly"]
        SnapB["nydus-snapshotter + nydusd (fscache)"]
        DfB["dfdaemon Peer B (:65001 / :65000 / :65002)"]
        ErofsB["Kernel EROFS (0xE0F5E1E2) + /dev/cachefiles"]
        OciB --> SnapB
        SnapB --> ErofsB
        SnapB -->|"HTTP Range (:65001)"| DfB
    end

    GAR["Google Artifact Registry (us-central1-docker.pkg.dev)"]

    DfA <-->|"RegisterPeerTask / AnnounceHost"| Scheduler
    DfB <-->|"RegisterPeerTask / AnnounceHost"| Scheduler
    DfA -->|"1. Cold Back-to-Source HTTPS Range"| GAR
    DfB ===>|"2. Cross-Node P2P Piece Stream (10.1.15.220:65000/65002)"| DfA
```

---

## 3. Detailed E2E Test Cases & Live Verification Results

### Test Case 1: GKE COS Negative Test — Strict No-Fallback Enforcement (`PASS`)

- **Target Node:** `gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft` (`COS_CONTAINERD`, Kernel `6.6.119+`)
- **Command:** `/home/kubernetes/bin/ocilayer-nydus-dragonfly-cos-test --listen-socket=/tmp/test-nydus-df.sock`
- **Result:** Exited immediately with **Exit Code `1`** and refused to fall back to `fusedev`:

```text
2026/10/14 01:51:07 ERROR Failed to initialize cross-node Nydus + Dragonfly EROFS/fscache OCILayerService driver err="kernel EROFS + fscache is required for ocilayer-nydus-dragonfly: 'erofs' filesystem not registered in /proc/filesystems (GKE COS disables CONFIG_EROFS_FS; run on an EROFS-enabled kernel with modprobe erofs; fusedev fallback is disabled)"
command terminated with exit code 1
```

---

### Test Case 2: Unconverted Standard OCI Image Fast Decline & `declineTTL` Cache Hit (`PASS`)

- **Target Node:** Node A (`fkxc`, pod `ate-ocilayer-nydus-dragonfly-g9hnd`)
- **Target Image:** `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest` (standard `tar+gzip` OCI image, not converted with `nydusify`)
- **Call #1 (Cold Manifest Inspection Decline):**

```text
PrepareLayers error: image not supported by OCILayerService: layer 0 (sha256:34884abbe92863fce933ed7c39c0e045631af0ed86d5cc0dfbdf9fdca426ce3c) labels: image "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest" is not in Nydus RAFS v6 (EROFS) format (missing containerd.io/snapshot/nydus-bootstrap layer; pre-convert with 'nydusify convert --fs-version 6'): image not supported by OCILayerService
```

- **Call #2 (Warm `declineTTL` Cache Hit — `1.263 ms`):**

```json
{
  "provider": "nydus-dragonfly",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest",
  "canProvide": false,
  "canProvideMs": 1.263,
  "prepareMs": 0,
  "releaseMs": 0,
  "verifiedLayers": 0
}
```

---

### Test Case 3: Node A (`fkxc`) Cold & Warm `PrepareLayers` and Kernel EROFS Superblock Verification (`PASS`)

- **Target Node:** Node A (`fkxc`, `10.1.15.220`, pod `ate-ocilayer-nydus-dragonfly-mhdj8`)
- **Target Image:** `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:nydus-v6` (`9` layers: `8` Nydus RAFS v6 data blobs + `1` EROFS bootstrap layer, `1.19 GB` compressed / `3.5 GB` unpacked / `6.2 GiB` virtual EROFS tree with `68,096` regular files)

#### 3.1 Cold `PrepareLayers` on Node A (`1081.491 ms`)

```json
{
  "provider": "nydus-dragonfly",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:nydus-v6",
  "imageDigest": "sha256:63efd7bfec93de795766e03c46b21496df0679548e46a37eb6291b3f5a9fff7c",
  "canProvide": true,
  "layerDirs": [
    "/var/lib/ateom-gvisor/streaming/nydus-dragonfly/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_nydus-v6/layer-0"
  ],
  "canProvideMs": 6.145,
  "prepareMs": 1081.491,
  "releaseMs": 0.975,
  "verifiedLayers": 1
}
```

#### 3.2 Warm `PrepareLayers` on Node A (Active Lease Cache Hit — `0.588 ms`)

```json
{
  "provider": "nydus-dragonfly",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:nydus-v6",
  "imageDigest": "sha256:63efd7bfec93de795766e03c46b21496df0679548e46a37eb6291b3f5a9fff7c",
  "canProvide": true,
  "layerDirs": [
    "/var/lib/ateom-gvisor/streaming/nydus-dragonfly/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_nydus-v6/layer-0"
  ],
  "canProvideMs": 1.929,
  "prepareMs": 0.588,
  "releaseMs": 0.221,
  "verifiedLayers": 1
}
```

#### 3.3 Kernel EROFS Mount & Superblock Magic Verification (`0xE0F5E1E2`)

```text
  File: "/var/lib/ateom-gvisor/streaming/nydus-dragonfly/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_nydus-v6/layer-0/fs/"
    ID: 0        Namelen: 255     Type: erofs
Block size: 4096       Fundamental block size: 4096
Blocks: Total: 1622324    Free: 0          Available: 0
Inodes: Total: 18446744073709551615 Free: -83812

Filesystem     Type   Size  Used Avail Use% Mounted on
erofs          erofs  6.2G  6.2G     0 100% /var/lib/containerd-nydus/snapshots/9/mnt

STATFS_TYPE=0xE0F5E1E2
TOTAL_REGULAR_FILES=68096 LARGE_FILES=578
```

---

### Test Case 4: Cross-Node Dragonfly v2 P2P Streaming from Node A (`fkxc`) to Node B (`49x6`) (`PASS`)

With Node A (`10.1.15.220`) holding the downloaded RAFS v6 pieces in its local Dragonfly peer storage (`/var/lib/dragonfly`) and registered in `dragonfly-scheduler` (`10.96.1.32:8002`), we ran `PrepareLayers` and `erofs-bench` on **Node B (`49x6`, `10.1.15.235`, pod `ate-ocilayer-nydus-dragonfly-b59bn`)** while Node B's local `fscache` and kernel page cache were completely cold.

#### 4.1 Node B (`49x6`) Cold `PrepareLayers` (`831.726 ms`)

```json
{
  "provider": "nydus-dragonfly",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:nydus-v6",
  "imageDigest": "sha256:63efd7bfec93de795766e03c46b21496df0679548e46a37eb6291b3f5a9fff7c",
  "canProvide": true,
  "layerDirs": [
    "/var/lib/ateom-gvisor/streaming/nydus-dragonfly/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_nydus-v6/layer-0"
  ],
  "canProvideMs": 55.199,
  "prepareMs": 831.726,
  "releaseMs": 0.38,
  "verifiedLayers": 1
}
```

#### 4.2 Node B (`49x6`) Cross-Node P2P Demand-Paging & Sequential Read Output

```text
STATFS_TYPE=0xE0F5E1E2
TOTAL_REGULAR_FILES=68096 LARGE_FILES=578
DEMAND_PAGING_FILES=50 SAMPLE_BYTES=467229 MEAN=67.660846ms P50=47.740032ms P90=85.336376ms P99=521.976417ms MAX=521.976417ms
SEQ_READ_MIB=64.32 SEQ_ELAPSED=517.959225ms SEQ_THROUGHPUT_MIB_S=124.18
```

#### 4.3 Live Evidence of Cross-Node P2P Piece Transfers (`10.1.15.220` -> `10.1.15.235`)

From `dfdaemon` logs on **Node B (`10.1.15.235`, `ate-ocilayer-nydus-dragonfly-b59bn`)**:

```text
2026-10-14T01:48:51.733Z INFO grpclog/grpclog.go:37 [core][Channel #54 SubChannel #55]Subchannel picks a new address "10.1.15.220:65000" to connect
2026-10-14T01:48:51.734Z INFO grpclog/grpclog.go:37 [core][Channel #54]Channel Connectivity change to READY
2026-10-14T01:48:51.738Z INFO peer/peertask_piecetask_synchronizer.go:357 got piece 0 from 10.1.15.220:65002/10.1.15.220-3303506-832f9239-20ad-432c-a082-84617ba74249, digest: , start: 0, size: 4194304 {"targetPeerID": "10.1.15.220-3303506-832f9239-20ad-432c-a082-84617ba74249", "peer": "10.1.15.235-74515-72daef98-2715-445b-9988-0f0c84bc1104", "task": "8a7020dc37e052749c414ae6d458adf3bf201dd5019ae09f32fd8bcff539978e", "component": "PeerTask"}
2026-10-14T01:48:51.738Z INFO peer/peertask_piecetask_synchronizer.go:357 got piece 1 from 10.1.15.220:65002/10.1.15.220-3303506-832f9239-20ad-432c-a082-84617ba74249, digest: , start: 4194304, size: 4194304
2026-10-14T01:48:51.738Z INFO peer/peertask_piecetask_synchronizer.go:357 got piece 2 from 10.1.15.220:65002/10.1.15.220-3303506-832f9239-20ad-432c-a082-84617ba74249, digest: , start: 8388608, size: 4194304
2026-10-14T01:48:51.738Z INFO peer/peertask_piecetask_synchronizer.go:357 got piece 3 from 10.1.15.220:65002/10.1.15.220-3303506-832f9239-20ad-432c-a082-84617ba74249, digest: , start: 12582912, size: 4194304
2026-10-14T01:48:51.738Z INFO peer/peertask_piecetask_synchronizer.go:357 got piece 4 from 10.1.15.220:65002/10.1.15.220-3303506-832f9239-20ad-432c-a082-84617ba74249, digest: , start: 16777216, size: 4194304
2026-10-14T01:48:51.738Z INFO peer/peertask_piecetask_synchronizer.go:357 got piece 5 from 10.1.15.220:65002/10.1.15.220-3303506-832f9239-20ad-432c-a082-84617ba74249, digest: , start: 20971520, size: 4194304
2026-10-14T01:48:51.738Z INFO peer/peertask_piecetask_synchronizer.go:357 got piece 6 from 10.1.15.220:65002/10.1.15.220-3303506-832f9239-20ad-432c-a082-84617ba74249, digest: , start: 25165824, size: 4194304
2026-10-14T01:48:51.738Z INFO peer/peertask_piecetask_synchronizer.go:357 got piece 7 from 10.1.15.220:65002/10.1.15.220-3303506-832f9239-20ad-432c-a082-84617ba74249, digest: , start: 29360128, size: 3010214
```

From `dragonfly-scheduler` logs (`dragonfly-scheduler-8c6b4587c-fjnlc` at `10.96.1.32:8002`):

```text
2026-10-14T01:48:51.728Z INFO service/service_v1.go:122 register as normal task, because of task state is Running {"hostID": "54f86b3b3f7b738ba05e3c639ab8bf3b9bef75627b773a75a47039178bede406", "taskID": "6468c8a1bb659c21499d2efc505a108dcaae2d0f06c80518fc34219b21cb82b6", "peerID": "10.1.15.235-74515-2fe3d6f6-4d0d-47ca-8aed-fc54b40b576a"}
2026-10-14T01:48:51.728Z INFO zap/server_interceptors.go:39 finished unary call with code OK {"grpc.service": "scheduler.Scheduler", "grpc.method": "RegisterPeerTask", "grpc.code": "OK", "grpc.time_ms": 0.508}
```

---

### Test Case 5: Multi-Tier Cache Hierarchy Performance Summary (`PASS`)

We measured 50-file demand-paging read latency (`467,229` sample bytes across 50 files) and sequential read throughput across all four cache tiers in the 2-node cluster:

| Cache State | Node | Description | `PrepareLayers` Latency | 50-File Read `p50` | 50-File Read `mean` | 50-File Read `p90` | 50-File Read `p99` / `max` | Sequential Read Throughput |
|---|---|---|---:|---:|---:|---:|---:|---:|
| **State A: Cluster-Cold (GAR Back-to-Source)** | Node A (`fkxc`, `10.1.15.220`) | First node in cluster pulling chunks from GAR via `dfdaemon` (`useHTTPS: true`) | `1081.49 ms` | `49.00 ms` | `285.18 ms` | `561.07 ms` | `7014.07 ms` | `127.96 MiB/s` (`64.32 MiB` in `502.67 ms`) |
| **State B: Cross-Node P2P (Cold Node B `fscache` + Warm Node A Peer)** | Node B (`49x6`, `10.1.15.235`) | Second node with empty local `fscache` streaming 4 MiB pieces from Node A (`10.1.15.220:65000/65002`) via `dragonfly-scheduler` | `831.73 ms` | `47.74 ms` | **`67.66 ms`** (`4.21x` faster) | **`85.34 ms`** (`6.57x` faster) | **`521.98 ms`** (`13.44x` faster) | `124.18 MiB/s` (`64.32 MiB` in `517.96 ms`) / `159.51 MiB/s` (`260.61 MiB` in `1.63 s`) |
| **State C1: Warm Local Disk `fscache` (`vm.drop_caches=1`)** | Node A (`fkxc`, `10.1.15.220`) | Kernel page cache dropped; chunks served from local NVMe `/var/lib/containerd-nydus/cache` via `/dev/cachefiles` (`0` network I/O) | `0.588 ms` (warm lease) | **`0.432 ms`** (`431.85 µs`) | **`0.448 ms`** (`448.09 µs`) | **`0.644 ms`** (`644.17 µs`) | **`0.735 ms`** (`735.05 µs`) | **`243.64 MiB/s`** (`64.32 MiB` in `264.00 ms`) |
| **State C2: Warm Kernel EROFS Page Cache** | Node A (`fkxc`) & Node B (`49x6`) | Hot VFS page cache backed by Kernel EROFS (`0` userspace context switches, `0` FUSE overhead) | `0.588 ms` (warm lease) | **`0.0076 ms`** (`7.63 µs`) | **`0.0095 ms`** (`9.53 µs`) | **`0.0153 ms`** (`15.30 µs`) | **`0.0542 ms`** (`54.21 µs`) | **`7539.30 MiB/s`** (`64.32 MiB` in `8.53 ms`) |

---

## 4. Comparison: Single-Node (`nydus`) vs Cross-Node (`nydus-dragonfly`)

| Dimension | Single-Node Adapter ([`cmd/ocilayer-nydus`](../cmd/ocilayer-nydus)) | Cross-Node P2P Adapter ([`cmd/ocilayer-nydus-dragonfly`](../cmd/ocilayer-nydus-dragonfly)) |
|---|---|---|
| `setup-gcp` / `install-ate.sh` Flag | `--setup-image-streaming=nydus` | `--setup-image-streaming=nydus-dragonfly` |
| `OCILayerService` Provider Name | `nydus` | `nydus-dragonfly` |
| Control-Plane Footprint | None (DaemonSet only) | `dragonfly-mysql`, `dragonfly-redis`, `dragonfly-manager`, `dragonfly-scheduler` |
| Node-Local Piece Cache | Lightweight Go HTTP Range piece-caching proxy (`:65001`) | Full `dragonflyoss/dfdaemon:v2.1.49` peer (`:65001` proxy, `:40901` health, `:65000` gRPC, `:65002` upload) |
| Second Node Cold Pull Source | Google Artifact Registry (back-to-source per node) | Peer-to-peer from first node (`10.1.15.220:65002` -> `10.1.15.235`) scheduled by `dragonfly-scheduler` |
| Second Node 50-File Cold Read `p90` / `p99` | Same as first node cold pull (`~202 ms` – `561 ms` `p90`, multi-second `p99`) | **`85.34 ms` (`p90`) / `521.98 ms` (`p99`)** (`6.57x` – `13.44x` reduction in tail latency) |
| Warm Local `fscache` / Page Cache Speed | `0.65 ms` (`fscache`) / `0.010 ms` (page cache) | `0.43 ms` (`fscache`) / `0.0076 ms` (page cache) |
| Host `containerd` Dependency | Zero | Zero |
| Kernel Requirement | Linux `CONFIG_EROFS_FS` + `CONFIG_CACHEFILES_ONDEMAND` (`0xE0F5E1E2`) | Linux `CONFIG_EROFS_FS` + `CONFIG_CACHEFILES_ONDEMAND` (`0xE0F5E1E2`) |
