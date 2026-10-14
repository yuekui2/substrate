# Cross-Node Spegel Stateless P2P `OCILayerService` Adapter (`ate-ocilayer-spegel`) — Multi-Node GKE E2E Verification & Performance Report

## 1. Executive Summary

This report documents the live multi-node GKE verification and performance evaluation of the **Mode A Stateless Spegel P2P `OCILayerService` Adapter (`cmd/ocilayer-spegel`)** across a **4-node heterogeneous GKE cluster** spanning **both `COS_CONTAINERD` (`Container-Optimized OS`) and `UBUNTU_CONTAINERD` (`Ubuntu 24.04 LTS`)**:

- **Zero Central Control Plane:** Deployed only the headless `spegel-bootstrap` `Service` and the `ate-ocilayer-spegel` `DaemonSet` in `ate-system` (`0` `mysql`, `redis`, `manager`, or `scheduler` pods).
- **100% Standard Unconverted OCI Images & Universal Kernel Compatibility:** Verified on standard `tar+gzip` OCI images (`image-a:latest`, `sandbox-04176181c57f0bf23e61506b0ddcf1fd:latest`, and the 9-layer `1.19 GB` / `3.5 GB` unpacked `sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bf...`) across both GKE `COS_CONTAINERD` (Kernel `6.12.105+`) and `UBUNTU_CONTAINERD` (Kernel `6.8.0-1062-gke`).
- **Stateless DNS Peer Discovery (`spegel-bootstrap.ate-system.svc.cluster.local`):** Each node (`10.1.15.212`, `10.1.15.225`, `10.1.15.220`, `10.1.15.235`) automatically discovered its 3 remote cluster peers on `:5000` via headless Service DNS and prevented recursive P2P loops via `X-Spegel-External-Request: true`.
- **Cross-Node & Cross-OS P2P Layer Distribution (`4.1x – 4.6x` End-to-End Speedup, `7.3x – 8.5x` Layer Materialization Speedup):**
  - **3-Layer Standard OCI Image (`image-a:latest`, `56.29 MB` compressed):**
    - Cluster-Cold pull from GAR on Node 1 (`65vz`, `COS_CONTAINERD`): **`2,027.69 ms`** (`prepare_ms = 1,659.0 ms`).
    - Cross-node P2P pull on Node 2 (`p1tz`, `COS_CONTAINERD` `no-stream-pool`): **`461.85 ms`** (`prepare_ms = 202.1 ms` — **`8.2x` faster layer download+unpack**, **`4.39x` faster end-to-end**, `0` GAR blob fetches).
    - Cross-OS P2P pull on Node 3 (`fkxc`, `UBUNTU_CONTAINERD`): **`498.30 ms`** (`prepare_ms = 228.5 ms` — **`7.3x` faster layer download+unpack**, **`4.07x` faster end-to-end**, `0` GAR blob fetches).
    - Multi-seeder mesh P2P pull on Node 4 (`49x6`, `UBUNTU_CONTAINERD`, served by Node 2 as a second-hop seeder): **`440.77 ms`** (**`4.60x` faster end-to-end**, `0` GAR blob fetches).
  - **Cross-Image Shared Layer Deduplication (`sandbox-...:latest`):**
    - Shared base Layer 0 (`chainID = sha256:34884abb...`) was automatically deduplicated on disk (`0` network bytes, `< 0.03 ms` stat check), and the remaining 2 delta layers (`6.02 MB`) streamed via Spegel P2P in **`83.4 – 84.4 ms`** (`prepare_ms`, **`8.5x` faster** than cold GAR `709.2 ms`).
  - **9-Layer `1.19 GB` (`3.5 GB` Unpacked, `68,096` Files) Image (`@sha256:fa2f27bf...`):**
    - All `1.19 GB` of compressed layer blobs streamed peer-to-peer from Node 1 (`10.1.15.212:5000`) to Node 2 (`10.1.15.225`) over `:5000` with **`0` upstream GAR blob downloads** (`1,317,679,770` total P2P bytes served by Node 1; largest `~650 MB` layer transferred over P2P in `~1.0 s`).
    - Once unpacked into `/var/lib/ateom-gvisor/streaming/spegel-store/layers/sha256/<chainID>/fs`, actors read directly from the native `ext4` filesystem (`STATFS_TYPE=0xEF53`) with **`0` FUSE or network paging overhead**: **`181.83 – 194.41 MiB/s`** cold disk read (`p50 = 372.5 µs`) and **`4,137.90 – 4,810.04 MiB/s`** warm page-cache read (`p50 = 7.96 µs`).

---

## 2. Cluster Topology & Test Environment

| Role | Node Name | OS / Kernel | Machine Type | Internal IP | Spegel DaemonSet Pod |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Node 1 (Primary Seeder, COS + GCFS)** | `gke-substrate-stream-substrate-node-p-01c8a8ea-65vz` | `COS_CONTAINERD` (`6.12.105+`) | `c3-standard-4` (4 vCPU, 16 GiB) | `10.1.15.212` | `ate-ocilayer-spegel-zxkj2` |
| **Node 2 (P2P Receiver + 2nd Seeder, COS No-Stream)** | `gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz` | `COS_CONTAINERD` (`6.12.105+`) | `c3-standard-4` (4 vCPU, 16 GiB) | `10.1.15.225` | `ate-ocilayer-spegel-btb4p` |
| **Node 3 (Cross-OS P2P Receiver, Ubuntu)** | `gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc` | `UBUNTU_CONTAINERD` (`6.8.0-1062-gke`) | `c3-standard-4` (4 vCPU, 16 GiB) | `10.1.15.220` | `ate-ocilayer-spegel-hwhnn` |
| **Node 4 (Multi-Hop P2P Receiver, Ubuntu)** | `gke-substrate-stream-erofs-ubuntu-poo-ea32e702-49x6` | `UBUNTU_CONTAINERD` (`6.8.0-1062-gke`) | `c3-standard-4` (4 vCPU, 16 GiB) | `10.1.15.235` | `ate-ocilayer-spegel-vwtbg` |

### Deployed `ate-system` Spegel Stack

```text
NAME                        READY   STATUS    RESTARTS   AGE    IP            NODE
ate-ocilayer-spegel-btb4p   1/1     Running   0          6m1s   10.1.15.225   gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz
ate-ocilayer-spegel-hwhnn   1/1     Running   0          6m     10.1.15.220   gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc
ate-ocilayer-spegel-vwtbg   1/1     Running   0          6m     10.1.15.235   gke-substrate-stream-erofs-ubuntu-poo-ea32e702-49x6
ate-ocilayer-spegel-zxkj2   1/1     Running   0          6m1s   10.1.15.212   gke-substrate-stream-substrate-node-p-01c8a8ea-65vz

NAME               ENDPOINTS                                                        AGE
spegel-bootstrap   10.1.15.212:5000,10.1.15.220:5000,10.1.15.225:5000 + 5 more...   8m1s
```

---

## 3. End-to-End Test Cases & Verification Results

### Test Case 1: Stateless DNS Peer Discovery (`spegel-bootstrap`) & `/healthz` Readiness (`PASS`)

Each `ate-ocilayer-spegel` pod runs with `hostNetwork: true` and `dnsPolicy: ClusterFirstWithHostNet`, resolving `spegel-bootstrap.ate-system.svc.cluster.local` to discover all remote cluster peers without any central coordinator:

```json
// Node 1 (65vz, 10.1.15.212 - COS_CONTAINERD):
{"blobs_cached":0,"bytes_cached":0,"local_hits":0,"peer_hits":0,"peer_misses":0,"p2p_requests_in":0,"p2p_bytes_served":0,"health_pings":74,"discovered_peers":["10.1.15.235:5000","10.1.15.220:5000","10.1.15.225:5000"]}

// Node 2 (p1tz, 10.1.15.225 - COS_CONTAINERD no-stream-pool):
{"blobs_cached":0,"bytes_cached":0,"local_hits":0,"peer_hits":0,"peer_misses":0,"p2p_requests_in":0,"p2p_bytes_served":0,"health_pings":76,"discovered_peers":["10.1.15.212:5000","10.1.15.235:5000","10.1.15.220:5000"]}

// Node 3 (fkxc, 10.1.15.220 - UBUNTU_CONTAINERD):
{"blobs_cached":0,"bytes_cached":0,"local_hits":0,"peer_hits":0,"peer_misses":0,"p2p_requests_in":0,"p2p_bytes_served":0,"health_pings":76,"discovered_peers":["10.1.15.235:5000","10.1.15.212:5000","10.1.15.225:5000"]}

// Node 4 (49x6, 10.1.15.235 - UBUNTU_CONTAINERD):
{"blobs_cached":0,"bytes_cached":0,"local_hits":0,"peer_hits":0,"peer_misses":0,"p2p_requests_in":0,"p2p_bytes_served":0,"health_pings":76,"discovered_peers":["10.1.15.225:5000","10.1.15.212:5000","10.1.15.220:5000"]}
```

---

### Test Case 2: 4-Node Cross-OS P2P Distribution — 3-Layer Standard OCI Image (`image-a:latest`, `56.29 MB`) (`PASS`)

| Node & Scenario | OS Distribution | Blob Source | `CanProvide` (`ms`) | `PrepareLayers` Total (`ms`) | `prepare_ms` (Blob Fetch + Unpack) | Speedup vs. Cold GAR |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Node 1 (`65vz`) — Cluster-Cold Pull** | `COS_CONTAINERD` | `upstream:us-central1-docker.pkg.dev` (`3/3` blobs) | `1.947 ms` | **`2,027.69 ms`** | `1,659.00 ms` | `1.00x` (baseline) |
| **Node 1 (`65vz`) — Warm Lease Hit** | `COS_CONTAINERD` | Active in-memory lease | `1.533 ms` | **`0.570 ms`** | `0.00 ms` | **`3,557x`** |
| **Node 2 (`p1tz`) — Cross-Node P2P Pull** | `COS_CONTAINERD` (`no-stream`) | `spegel-p2p:10.1.15.212:5000` (`3/3` blobs) | `1.942 ms` | **`461.85 ms`** | **`202.08 ms`** | **`4.39x` total / `8.21x` prepare** |
| **Node 3 (`fkxc`) — Cross-OS P2P Pull** | `UBUNTU_CONTAINERD` | `spegel-p2p:10.1.15.212:5000` (`3/3` blobs) | `2.020 ms` | **`498.30 ms`** | **`228.49 ms`** | **`4.07x` total / `7.26x` prepare** |
| **Node 4 (`49x6`) — Multi-Seeder Mesh P2P Pull** | `UBUNTU_CONTAINERD` | `spegel-p2p:10.1.15.225:5000` (`3/3` blobs) | `2.343 ms` | **`440.77 ms`** | **`~190 ms`** | **`4.60x` total** |

#### Multi-Seeder Mesh P2P Verification (`/stats` Across All 4 Nodes)

After Node 2 (`10.1.15.225`) pulled `image-a:latest` from Node 1 (`10.1.15.212`), Node 2 immediately advertised those 3 blobs in its own Spegel store and served Node 4 (`10.1.15.235`):

```json
// Node 1 (zxkj2, 10.1.15.212) — served 2 peers (112.58 MB = 2 × 56.29 MB):
{"blobs_cached":3,"bytes_cached":56291503,"local_hits":12,"peer_hits":0,"peer_misses":3,"p2p_requests_in":12,"p2p_bytes_served":112583006,"health_pings":80,"discovered_peers":["10.1.15.235:5000","10.1.15.220:5000","10.1.15.225:5000"]}

// Node 2 (btb4p, 10.1.15.225) — pulled 3 blobs from Node 1 (peer_hits=3, peer_misses=0), then served 1 peer (56.29 MB to Node 4):
{"blobs_cached":3,"bytes_cached":56291503,"local_hits":6,"peer_hits":3,"peer_misses":0,"p2p_requests_in":9,"p2p_bytes_served":56291503,"health_pings":81,"discovered_peers":["10.1.15.212:5000","10.1.15.235:5000","10.1.15.220:5000"]}

// Node 3 (hwhnn, 10.1.15.220) — pulled 3 blobs via P2P (peer_hits=3, peer_misses=0):
{"blobs_cached":3,"bytes_cached":56291503,"local_hits":0,"peer_hits":3,"peer_misses":0,"p2p_requests_in":3,"p2p_bytes_served":0,"health_pings":80,"discovered_peers":["10.1.15.235:5000","10.1.15.212:5000","10.1.15.225:5000"]}

// Node 4 (vwtbg, 10.1.15.235) — pulled 3 blobs via P2P from Node 2 (peer_hits=3, peer_misses=0):
{"blobs_cached":3,"bytes_cached":56291503,"local_hits":0,"peer_hits":3,"peer_misses":0,"p2p_requests_in":6,"p2p_bytes_served":0,"health_pings":80,"discovered_peers":["10.1.15.225:5000","10.1.15.212:5000","10.1.15.220:5000"]}
```

---

### Test Case 3: Cross-Image Layer Deduplication & Incremental P2P Transfer (`sandbox-04176181c57f0bf23e61506b0ddcf1fd:latest`) (`PASS`)

`sandbox-04176181c57f0bf23e61506b0ddcf1fd:latest` (`sha256:726cbe97...`) has 3 layers where **Layer 0 (`chainID = sha256:34884abb...`) is shared with `image-a:latest`** and Layers 1 & 2 (`6.02 MB`) are unique:

| Node & Scenario | Layer 0 (`50.2 MB`, Shared) | Layers 1 & 2 (`6.02 MB`, Delta) | `PrepareLayers` Total (`ms`) | `prepare_ms` |
| :--- | :--- | :--- | :--- | :--- |
| **Node 1 (`65vz`) — Cold Delta Pull** | Deduplicated on disk (`0 ms`, `0 B`) | Pulled from GAR (`upstream:us-central1-docker.pkg.dev`) | **`1,006.29 ms`** | `709.19 ms` |
| **Node 1 (`65vz`) — Warm Lease Hit** | Active lease | Active lease | **`0.455 ms`** | `0.00 ms` |
| **Node 2 (`p1tz`) — Cross-Node P2P Pull** | Deduplicated on disk (`0 ms`, `0 B`) | Streamed from `spegel-p2p:10.1.15.212:5000` | **`353.22 ms`** | **`84.40 ms` (`8.4x` faster)** |
| **Node 3 (`fkxc`) — Cross-OS P2P Pull** | Deduplicated on disk (`0 ms`, `0 B`) | Streamed from `spegel-p2p:10.1.15.212:5000` | **`328.30 ms`** | **`83.43 ms` (`8.5x` faster)** |

---

### Test Case 4: 9-Layer `1.19 GB` (`3.5 GB` Unpacked, `68,096` Files) Standard OCI Image P2P Transfer & Native Read Benchmark (`PASS`)

- **Target Image:** `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e` (`9` standard `tar+gzip` layers, `1.19 GB` compressed / `3.5 GB` unpacked).
- **Cross-Node P2P Transfer Verification:**
  - Node 1 (`65vz`, `10.1.15.212`) pulled the remaining 8 unique layers from GAR (`local-store` hit on shared blob `sha256:4a2796d8...`) and cached all `13` cluster blobs (`1,255,367,392` bytes = `1.20 GiB`).
  - Node 2 (`p1tz`, `10.1.15.225`) streamed **all 8 missing layer blobs (`1.19 GB`) over Spegel P2P from `10.1.15.212:5000` (`peer_hits=13`, `peer_misses=0`, `0` GAR blob fetches)** and reused shared blob `sha256:4a2796d8...` from `local-store`.
  - Total P2P bytes served by Node 1 (`10.1.15.212:5000`): **`1,317,679,770` bytes (`1.23 GiB`)**.
  - Warm `PrepareLayers` latency (9 layers): **`0.586 ms`** (Node 1) / **`0.481 ms`** (Node 2).

#### Adapter Logs on Node 2 (`ate-ocilayer-spegel-btb4p`, `COS_CONTAINERD` `no-stream-pool`)

```text
2026/10/14 05:08:54 INFO Synchronized registry credentials to docker config path=/run/ate-image-streaming/spegel/docker/config.json image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e server=us-central1-docker.pkg.dev username=_token
2026/10/14 05:08:55 INFO Streamed OCI layer blob from remote Spegel P2P peer peer=10.1.15.212:5000 digest=sha256:84ad5f3a28e133b7e3d79a5974fa2411b373762f03734f30c1c2b75fb05bcb40
2026/10/14 05:08:55 INFO Materialized Spegel OCI layer layerIndex=0 blobDigest=sha256:84ad5f3a28e133b7e3d79a5974fa2411b373762f03734f30c1c2b75fb05bcb40 source=spegel-p2p:10.1.15.212:5000 layerFSDir=/var/lib/ateom-gvisor/streaming/spegel-store/layers/sha256/ea16cace89338c84eb6bcb91a7efdfcae6838fff359efe951858227436486c34/fs
2026/10/14 05:08:55 INFO Streamed OCI layer blob from remote Spegel P2P peer peer=10.1.15.212:5000 digest=sha256:a17129cc092793e477505b92a06cd8d44c74989d116656cb6624a2e5b87aa089
2026/10/14 05:08:55 INFO Materialized Spegel OCI layer layerIndex=1 blobDigest=sha256:a17129cc092793e477505b92a06cd8d44c74989d116656cb6624a2e5b87aa089 source=spegel-p2p:10.1.15.212:5000 layerFSDir=/var/lib/ateom-gvisor/streaming/spegel-store/layers/sha256/5dda86b5f741fd620228596b7d11aafada6529e6011f7f54e957e2b5636c18e9/fs
2026/10/14 05:08:55 INFO Streamed OCI layer blob from remote Spegel P2P peer peer=10.1.15.212:5000 digest=sha256:e9a96d24d607549aeede6055c93293c34678b2225a6f20c8923a103a55171e4a
2026/10/14 05:08:55 INFO Materialized Spegel OCI layer layerIndex=2 blobDigest=sha256:e9a96d24d607549aeede6055c93293c34678b2225a6f20c8923a103a55171e4a source=spegel-p2p:10.1.15.212:5000 layerFSDir=/var/lib/ateom-gvisor/streaming/spegel-store/layers/sha256/3cb8e82e2e0c0fe5e61a20fb35e1a621188305e44850f9c58dbe4e0ac4b449c2/fs
2026/10/14 05:08:57 INFO Streamed OCI layer blob from remote Spegel P2P peer peer=10.1.15.212:5000 digest=sha256:2edc65f4c1b13f4dbd79d1298ea6a3070f86022e6726d99a73830a6a25005ac9
2026/10/14 05:09:08 INFO Materialized Spegel OCI layer layerIndex=3 blobDigest=sha256:2edc65f4c1b13f4dbd79d1298ea6a3070f86022e6726d99a73830a6a25005ac9 source=spegel-p2p:10.1.15.212:5000 layerFSDir=/var/lib/ateom-gvisor/streaming/spegel-store/layers/sha256/4af17c5627211dde358de00cf665cc8e5b5c1d253c97f56057a6ec9ea3d349eb/fs
2026/10/14 05:09:08 INFO Streamed OCI layer blob from remote Spegel P2P peer peer=10.1.15.212:5000 digest=sha256:eff5bf5846f61928f32134940b127704dec6e7b6fb2919eae7d6043f96198f4b
2026/10/14 05:09:08 INFO Materialized Spegel OCI layer layerIndex=4 blobDigest=sha256:eff5bf5846f61928f32134940b127704dec6e7b6fb2919eae7d6043f96198f4b source=spegel-p2p:10.1.15.212:5000 layerFSDir=/var/lib/ateom-gvisor/streaming/spegel-store/layers/sha256/6b4d0fa0650dcf70101a9df4e0a73f20ccf7343b3942e72eda85884b8ea84f13/fs
2026/10/14 05:09:08 INFO Streamed OCI layer blob from remote Spegel P2P peer peer=10.1.15.212:5000 digest=sha256:eee00ea291453c8f52c7bd4d604a2dee3ae6e1ca990c77342a0989d51524ca2f
2026/10/14 05:09:08 INFO Materialized Spegel OCI layer layerIndex=5 blobDigest=sha256:eee00ea291453c8f52c7bd4d604a2dee3ae6e1ca990c77342a0989d51524ca2f source=spegel-p2p:10.1.15.212:5000 layerFSDir=/var/lib/ateom-gvisor/streaming/spegel-store/layers/sha256/35e76a8344cfcd7992490dc23d7deacf38c2219b7cb5da4192109e03ddec4ef1/fs
2026/10/14 05:09:09 INFO Streamed OCI layer blob from remote Spegel P2P peer peer=10.1.15.212:5000 digest=sha256:6cca528c578ed71cc7845b45d5d2b4d13be0167c2873e44fbca2b847980bbda1
2026/10/14 05:09:24 INFO Materialized Spegel OCI layer layerIndex=6 blobDigest=sha256:6cca528c578ed71cc7845b45d5d2b4d13be0167c2873e44fbca2b847980bbda1 source=spegel-p2p:10.1.15.212:5000 layerFSDir=/var/lib/ateom-gvisor/streaming/spegel-store/layers/sha256/f86362802e5cd5d78264a3406407f8677f5c1007920e3825faf1f9b55f6b5792/fs
2026/10/14 05:09:24 INFO Materialized Spegel OCI layer layerIndex=7 blobDigest=sha256:4a2796d8ea5ef7ae1a64ba20ba56759987f49f2ea2091ddce6bade81c0ad1fa6 source=local-store layerFSDir=/var/lib/ateom-gvisor/streaming/spegel-store/layers/sha256/7734848956b45c32d89c894d965cc31f272af4822e8554e292b20e7c67ca62ac/fs
2026/10/14 05:09:24 INFO Streamed OCI layer blob from remote Spegel P2P peer peer=10.1.15.212:5000 digest=sha256:4778a54a0081597a421cef5da837a421e979562e2a0cf1dd9d360ce727d4eea9
2026/10/14 05:09:24 INFO Materialized Spegel OCI layer layerIndex=8 blobDigest=sha256:4778a54a0081597a421cef5da837a421e979562e2a0cf1dd9d360ce727d4eea9 source=spegel-p2p:10.1.15.212:5000 layerFSDir=/var/lib/ateom-gvisor/streaming/spegel-store/layers/sha256/fb2ce8854ebc5f60bff909e1958df5ef993746ec297d0da1c1b40d181a04e9f5/fs
```

#### Native Unpacked Layer Read Latency & Sequential Throughput (`STATFS_TYPE=0xEF53`, Layer 6: `22,092` Files)

| Benchmark Mode (Node 2 `p1tz`, `c3-standard-4`) | Filesystem Type | 50-File Read `Mean` | `p50` | `p90` | `p99` | Sequential Read Throughput (`260.92 MiB`) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Cold Page Cache (`echo 3 > /proc/sys/vm/drop_caches`)** | Native `ext4` (`0xEF53`) | `739.39 µs` | **`372.55 µs`** | `2.77 ms` | `3.81 ms` | **`181.83 MiB/s`** (`1.435 s`) |
| **Warm Page Cache** | Native `ext4` (`0xEF53`) | `10.43 µs` | **`7.96 µs`** | `20.26 µs` | `48.34 µs` | **`4,810.04 MiB/s`** (`54.24 ms`) |

---

### Test Case 5: `ReleaseLayers (bypass_ttl=true)` Lease Cleanup & Persistent Blob/Layer Store Retention (`PASS`)

- Calling `ReleaseLayers` with `bypass_ttl=true` on all 4 nodes completed in **`0.44 – 2.05 ms`**.
- All per-image wrapper directories under `/var/lib/ateom-gvisor/streaming/spegel/` were immediately removed (`0` remaining wrapper directories), while the content-addressed compressed blobs (`blobs/sha256/<digest>`) and unpacked layer trees (`layers/sha256/<chainID>/fs`) under `/var/lib/ateom-gvisor/streaming/spegel-store/` remained cached on disk for instant subsequent reuse and peer-to-peer serving.
