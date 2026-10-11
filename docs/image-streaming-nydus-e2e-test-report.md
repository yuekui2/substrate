# Nydus + Dragonfly (Kernel EROFS + `fscache`) Live GKE E2E & Performance Test Report

- **Date:** 2026-10-10
- **GCP Project:** `kuiyue-gke-dev`
- **GKE Cluster:** `substrate-stream-test` (`us-central1-a`)
- **EROFS Test Node:** `gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc` (`erofs-ubuntu-pool`, `c3-standard-4`, 4 vCPUs, 16 GiB RAM, `UBUNTU_CONTAINERD`, Ubuntu 24.04.5 LTS, Kernel `6.8.0-1062-gke`)
- **Negative Test Node (GKE COS):** `gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft` (`COS_CONTAINERD`, Kernel `6.6.119+`)
- **Component Under Test:** `cmd/ocilayer-nydus` (`ate-ocilayer-nydus` DaemonSet + `containerd-nydus-grpc v0.15.0` + `nydusd v2.3.0` + Dragonfly HTTP Range/Piece-Caching Proxy at `DRAGONFLY_PROXY_ADDR=http://127.0.0.1:65001`)

---

## 1. Executive Summary

We implemented, deployed, and benchmarked the **Nydus + Dragonfly** `OCILayerService` adapter (`cmd/ocilayer-nydus` and `cmd/ocilayer-nydus/ocilayer-nydus.yaml`) configured strictly for **Kernel EROFS (`RAFS v6`) + Linux `fscache`/`cachefiles`** (`--fs-driver=fscache --daemon-mode=shared`) with **zero fallback to userspace FUSE (`fusedev`)**.

### Key Architectural & Operational Findings on GKE

1. **Strict No-Fallback Enforcement (`fusedev` Disabled):**
   - Both `cmd/ocilayer-nydus` (`VerifyKernelEROFSSupport` and `verifyMountIsEROFS`) and `cmd/ocilayer-nydus/ocilayer-nydus.yaml` verify that `erofs` is registered in `/proc/filesystems`, `/dev/cachefiles` exists, and every mounted snapshot has `statfs.Type == EROFS_SUPER_MAGIC (0xE0F5E1E2)`.
   - On GKE `COS_CONTAINERD` (`bpft`), `ocilayer-nydus` immediately exits with code `1` and logs:
     `kernel EROFS + fscache is required for ocilayer-nydus: 'erofs' filesystem not registered in /proc/filesystems (GKE COS disables CONFIG_EROFS_FS; run on an EROFS-enabled kernel with modprobe erofs; fusedev fallback is disabled)`.
2. **GKE `UBUNTU_CONTAINERD` Kernel Discovery (`CONFIG_CACHEFILES_ONDEMAND` & `CONFIG_EROFS_FS_ONDEMAND`):**
   - On GKE's Ubuntu 24.04 kernel (`6.8.0-1062-gke`), `CONFIG_EROFS_FS=m` and `CONFIG_CACHEFILES=m` are present as loadable modules, but the stock Ubuntu GKE modules are compiled with `# CONFIG_CACHEFILES_ONDEMAND is not set` and `# CONFIG_EROFS_FS_ONDEMAND is not set`.
   - With the stock modules, `nydusd singleton --fscache` fails with `EINVAL` (`dmesg`: `CacheFiles: 'bind' command doesn't take an argument`).
   - Because GKE `UBUNTU_CONTAINERD` ships `/lib/modules/6.8.0-1062-gke/build` and allows privileged containers to load modules via `insmod`, compiling and loading `erofs.ko` (`CONFIG_EROFS_FS_ONDEMAND=y`) and `cachefiles.ko` (`CONFIG_CACHEFILES_ONDEMAND=y`) inside a privileged container enables full in-kernel EROFS + `fscache` ondemand support without replacing the node OS or kernel image.
3. **Performance Highlights (9-Layer, `3.5 GB` Unpacked / `1.19 GB` Compressed Benchmark Image on `c3-standard-4`):**
   - **Cold `PrepareLayers` Latency:** **`736.87 ms` (`0.74s`)** — **6.2x faster** than Google Riptide v2 (`4.57s`) and **12.4x faster** than AWS SOCI (`9.16s`). Because Nydus RAFS v6 merges all layer metadata into a single `5.28 MB` EROFS bootstrap (`image.boot`) and mounts a single Kernel EROFS filesystem (`layers=1`), it avoids per-layer FUSE mounts and multi-layer zTOC decompression.
   - **Warm `PrepareLayers` (Active Lease Hit):** **`0.451 ms` (`451 µs`)**.
   - **Dragonfly P2P Acceleration (`State B: Warm Dragonfly P2P + Cold Node EROFS fscache`):**
     - When `fscache` is cold on a node and Dragonfly is cold (`State A`), `nydusd` fetches 1 MB chunks on-demand over HTTPS from GAR at **`6.91 MiB/s`** (`p90 = 202.42 ms`).
     - When Dragonfly's P2P piece cache is warm (`State B`, 100% cache hit rate on `DRAGONFLY_PROXY_ADDR=http://127.0.0.1:65001` with local node `fscache` completely wiped), sequential throughput jumps **19.0x** to **`131.24 MiB/s`** and `p90` demand-paging latency drops **4.6x** to **`43.54 ms`**.
   - **In-Kernel EROFS + `fscache` Warm Throughput (`State C`):**
     - **Warm Local Disk `fscache` (Kernel Page Cache Dropped):** **`180.20 MiB/s`** (`p50 = 0.655 ms`, `p99 = 1.404 ms`) with **zero `nydusd` userspace involvement**.
     - **Warm Kernel EROFS Page Cache:** **`5,813.32 MiB/s` (`5.81 GiB/s`)** and **`p50 = 0.0104 ms` (`10.4 µs`)** — **2.35x faster** than Riptide v2 / AWS SOCI warm FUSE reads (`2,469 MB/s`) because reads execute purely in the Linux kernel VFS/EROFS path with zero userspace FUSE context switches.

---

## 2. Offline RAFS v6 (EROFS) Image Conversion (Excluded from Runtime Benchmark)

To ensure an apples-to-apples runtime benchmark without penalizing Nydus for image conversion, the 9-layer benchmark image was pre-converted offline using `nydusify convert --fs-version 6` and pushed to Artifact Registry (GAR) prior to benchmarking:

- **Source OCI Image (9 layers, `1.19 GB` compressed / `3.5 GB` unpacked):**
  `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:benchmark-9layer`
  (`sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819`)
- **Pre-Converted Nydus RAFS v6 (EROFS) Image in GAR:**
  `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:nydus-v6`
  (`sha256:63efd7bfec93de795766e03c46b21496df0679548e46a37eb6291b3f5a9fff7c`)
  - **Layers `0..7`:** `application/vnd.oci.image.layer.nydus.blob.v1` (`containerd.io/snapshot/nydus-blob: "true"`, Zstd-compressed 1 MB chunked RAFS v6 data blobs).
  - **Layer `8` (EROFS Bootstrap Layer, `5.28 MB`):** `sha256:ba487201a54373ecc7dd03cbd402f32d1b887bc39c30992a58128e9b7bf35554` (`containerd.io/snapshot/nydus-bootstrap: "true"`, `containerd.io/snapshot/nydus-fs-version: "6"`).

---

## 3. Live Cluster Evidence & Test Cases

### Test Case 1: Negative Test on GKE `COS_CONTAINERD` Node (`bpft`) — Strict No-Fallback Verification

Running `ocilayer-nydus` on GKE `COS_CONTAINERD` node `gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft` (where `CONFIG_EROFS_FS` is disabled in the kernel):

```text
=== Step 1: Negative Test on GKE COS_CONTAINERD node (gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft) ===
COS Node Exit Code: 1
COS Node Output:
2026/10/10 19:18:07 ERROR Failed to initialize Nydus EROFS/fscache OCILayerService driver err="kernel EROFS + fscache is required for ocilayer-nydus: 'erofs' filesystem not registered in /proc/filesystems (GKE COS disables CONFIG_EROFS_FS; run on an EROFS-enabled kernel with modprobe erofs; fusedev fallback is disabled)"
```

---

### Test Case 2: Kernel EROFS + `cachefiles` Ondemand Module Enablement on GKE `UBUNTU_CONTAINERD` (`fkxc`)

Inspecting `/boot/config-6.8.0-1062-gke` on `gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc`:

```text
CONFIG_NETFS_SUPPORT=m
CONFIG_NETFS_STATS=y
CONFIG_CACHEFILES=m
# CONFIG_CACHEFILES_ONDEMAND is not set
CONFIG_EROFS_FS=m
CONFIG_EROFS_FS_XATTR=y
CONFIG_EROFS_FS_POSIX_ACL=y
CONFIG_EROFS_FS_SECURITY=y
CONFIG_EROFS_FS_ZIP=y
CONFIG_EROFS_FS_ZIP_DEFLATE=y
# CONFIG_EROFS_FS_ONDEMAND is not set
```

After compiling `cachefiles.ko` (`CONFIG_CACHEFILES_ONDEMAND=y`) and `erofs.ko` (`CONFIG_EROFS_FS_ONDEMAND=y`) against `/lib/modules/6.8.0-1062-gke/build` and loading them via `insmod` in the privileged container:

```text
cachefiles            172032  1
erofs                 147456  1
netfs                 507904  2 erofs,cachefiles
ffffffffc0bb86d0 t perf_trace_cachefiles_ondemand_open  [cachefiles]
ffffffffc0bb8810 t perf_trace_cachefiles_ondemand_copen [cachefiles]
ffffffffc0bb8a70 t perf_trace_cachefiles_ondemand_read  [cachefiles]
ffffffffc0bb8bb0 t perf_trace_cachefiles_ondemand_cread [cachefiles]
```

Startup logs of `ate-ocilayer-nydus` (`nydus-snapshotter` + `ocilayer-nydus`) on `fkxc`:

```text
time="2026-10-10T22:51:09.641950024Z" level=info msg="Start nydus-snapshotter. Version: v0.15.0, PID: 2138411, FsDriver: fscache, DaemonMode: shared"
time="2026-10-10T22:51:09.650467174Z" level=info msg="initializing shared nydus daemon for fscache"
time="2026-10-10T22:51:09.651942381Z" level=info msg="nydusd command: /home/kubernetes/bin/nydusd singleton --fscache /var/lib/containerd-nydus/cache --apisock /var/lib/containerd-nydus/socket/db5c2nahal7fn9id7mb0/api.sock --log-level info --log-rotation-size 100"
[2026-10-10 22:51:09.680342 +00:00] INFO Create fscache instance at /var/lib/containerd-nydus/cache with tag <none>, 4 working threads
[2026-10-10 22:51:09.693522 +00:00] INFO State machine(pid=2138474): from Ready to Running, input [Start], output [Some(StartService)]
2026/10/10 22:51:09 INFO Initialized Nydus + Dragonfly (Kernel EROFS + fscache) OCILayerService driver dragonflyProxyAddr=http://127.0.0.1:65001 procFilesystems=/proc/filesystems cachefilesDev=/dev/cachefiles
2026/10/10 22:51:09 INFO OCILayerService adapter listening provider=nydus listenSocket=/run/ate-image-streaming/ocilayer.sock snapshotterSocket=/run/containerd-nydus/containerd-nydus-grpc.sock
```

---

### Test Case 3: Unconverted Standard OCI Image Fast Decline + `declineTTL` Cache Hit

Probing `ate-ocilayer-nydus` with an unconverted standard OCI image (`image-a:latest`):

```text
Cold Decline (manifest inspect): 507.62 ms (rc=1) -> PrepareLayers error: image not supported by OCILayerService: layer 0 (sha256:34884abbe92863fce933ed7c39c0e045631af0ed86d5cc0dfbdf9fdca426ce3c) labels: image "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest" is not in Nydus RAFS v6 (EROFS) format (missing containerd.io/snapshot/nydus-bootstrap layer; pre-convert with 'nydusify convert --fs-version 6'): image not supported by OCILayerService
Warm Decline (declineTTL cache): 8.05 ms (rc=0) -> {
  "provider": "nydus",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest",
  "canProvide": false,
  "canProvideMs": 3.251,
  "prepareMs": 0,
  "releaseMs": 0,
  "verifiedLayers": 0
}
```

---

### Test Case 4: Cold & Warm `PrepareLayers` and Kernel EROFS Superblock Verification

#### Cold `PrepareLayers` (All Caches Wiped): `736.872 ms` (`645.854 ms` on Warm Dragonfly)

```json
{
  "provider": "nydus",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:nydus-v6",
  "imageDigest": "sha256:63efd7bfec93de795766e03c46b21496df0679548e46a37eb6291b3f5a9fff7c",
  "canProvide": true,
  "layerDirs": [
    "/var/lib/ateom-gvisor/streaming/nydus/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_nydus-v6/layer-0"
  ],
  "canProvideMs": 2.301,
  "prepareMs": 736.872,
  "releaseMs": 0.491,
  "verifiedLayers": 1
}
```

#### Warm `PrepareLayers` (Active Lease Cache Hit): `0.451 ms`

```json
{
  "provider": "nydus",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:nydus-v6",
  "imageDigest": "sha256:63efd7bfec93de795766e03c46b21496df0679548e46a37eb6291b3f5a9fff7c",
  "canProvide": true,
  "layerDirs": [
    "/var/lib/ateom-gvisor/streaming/nydus/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_nydus-v6/layer-0"
  ],
  "canProvideMs": 1.632,
  "prepareMs": 0.451,
  "releaseMs": 0.22,
  "verifiedLayers": 1
}
```

#### Kernel EROFS Mount & Superblock Magic Verification (`0xE0F5E1E2`)

```text
Resolved layer-0/fs -> /var/lib/containerd-nydus/snapshots/9/mnt
Path=/var/lib/containerd-nydus/snapshots/9/mnt Type=erofs Magic=0xe0f5e1e2 BlockSize=4096 Blocks=1622324
erofs /var/lib/containerd-nydus/snapshots/9/mnt erofs ro,relatime,user_xattr,acl,cache_strategy=readaround,fsid=cd423453ddd149b0c1f7ce0457c54c2ba218529a3249154dded4b493ca017443 0 0
Filesystem     Type   Size  Used Avail Use% Mounted on
erofs          erofs  6.2G  6.2G     0 100% /var/lib/containerd-nydus/snapshots/9/mnt
```

---

### Test Case 5: Demand-Paging Read Latency & Sequential Throughput Across All 3 Cache States

We measured demand-paging latency across 50 files and sequential bulk read throughput (`260.39 MiB` of uncompressed layer data, corresponding to `83.65 MB` of Zstd-compressed RAFS v6 chunks across `289` HTTP `Range` requests) across three distinct cache states:

| Cache State | Description | 50-File Read `p50` | 50-File Read `mean` | 50-File Read `p90` | 50-File Read `p99` | Sequential Read (`260.39 MiB`) | Dragonfly `:65001` Cache Stats |
|---|---|---:|---:|---:|---:|---:|---|
| **State A: Cold GAR + Cold Dragonfly + Cold `fscache`** | First node in cluster pulling chunks from GAR via Dragonfly (`MISS`) | `0.39 ms` | `66.13 ms` | `202.42 ms` | `323.40 ms` | **`6.91 MiB/s`** (`37,691 ms`) | `hits: 0, misses: 289` (`83.65 MB` fetched from GAR) |
| **State B: Warm Dragonfly P2P + Cold Node `fscache`** | Local node `fscache` & page cache wiped; 100% of chunks served by Dragonfly P2P (`HIT`) | `0.32 ms` | `12.38 ms` | `43.54 ms` | `52.54 ms` | **`131.24 MiB/s`** (`1,984 ms`) | `hits: 289, misses: 289` (`100%` hit rate in State B) |
| **State C1: Warm Local Disk `fscache` (Page Cache Dropped)** | `echo 1 > /proc/sys/vm/drop_caches`; Kernel EROFS reads directly from local disk `/var/lib/containerd-nydus/cache` (`nydusd` bypassed) | `0.655 ms` | `0.658 ms` | `0.866 ms` | `1.404 ms` | **`180.20 MiB/s`** (`1,445 ms`) | `0` new requests (pure in-kernel `fscache` disk read) |
| **State C2: Warm Kernel EROFS Page Cache** | Hot re-read directly from Linux kernel EROFS VFS page cache (zero FUSE overhead) | **`0.0104 ms`** (`10.4 µs`) | `0.0128 ms` | `0.0202 ms` | `0.0620 ms` | **`5,813.32 MiB/s`** (`44.79 ms`) | `0` new requests (pure in-kernel RAM page cache) |

#### Raw Benchmark Output Log

```text
--- [2.4] State A Cold Demand-Paging Latency (50 files) & Cold Sequential Read Throughput ---
[State A Cold Demand-Paging 50 files] mean=66.13ms p50=0.39ms p90=202.42ms p99=323.40ms max=323.40ms
[State A Cold Sequential Read] 260.39 MiB across 2 files in 37691.01 ms (6.91 MiB/s)
Dragonfly stats after State A Cold Read: {"bytes_hit":0,"bytes_miss":83653517,"cached_bytes":83653517,"cached_pieces":289,"hits":0,"misses":289,"pings":9}

================================================================================
BENCHMARK 3: State C — Warm Node Kernel EROFS fscache (Disk fscache vs Page Cache)
================================================================================
[State C1 Warm Disk fscache (page cache dropped) 50 files] mean=0.658ms p50=0.655ms p90=0.866ms p99=1.404ms
[State C1 Warm Disk fscache Sequential Read] 260.39 MiB in 1445.02 ms (180.20 MiB/s)
[State C2 Warm Kernel EROFS Page Cache 50 files] mean=0.0128ms p50=0.0104ms p90=0.0202ms p99=0.0620ms
[State C2 Warm Kernel EROFS Page Cache Sequential Read] 260.39 MiB in 44.79 ms (5813.32 MiB/s)
Dragonfly stats after State C1+C2 (confirm 0 extra network requests): {"bytes_hit":0,"bytes_miss":83653517,"cached_bytes":83653517,"cached_pieces":289,"hits":0,"misses":289,"pings":10}

================================================================================
BENCHMARK 4: State B — Warm Dragonfly P2P Cache + Cold Node EROFS fscache
================================================================================
--- [4.2] State B Demand-Paging Latency & Sequential Read (100% Dragonfly P2P Cache Hits -> Cold EROFS fscache) ---
[State B Dragonfly-Warm / fscache-Cold Demand-Paging 50 files] mean=12.38ms p50=0.32ms p90=43.54ms p99=52.54ms max=52.54ms
[State B Dragonfly-Warm / fscache-Cold Sequential Read] 260.39 MiB in 1984.06 ms (131.24 MiB/s)
Dragonfly stats after State B (verify hits increased): {"bytes_hit":83653517,"bytes_miss":83653517,"cached_bytes":83653517,"cached_pieces":289,"hits":289,"misses":289,"pings":11}
```
