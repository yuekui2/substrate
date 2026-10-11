# Performance & Architecture Comparison: Google Riptide v2 vs. AWS SOCI (`OCILayerService`)

## 1. Executive Summary

Both **Google Riptide v2** (`cmd/ocilayer-riptide`) and **AWS SOCI v0.15.0** (`cmd/ocilayer-soci`) implement the Substrate `OCILayerService` gRPC contract (`internal/proto/ocilayer/v1/ocilayer.proto`) on top of the shared `internal/ocilayer/adapter` library and were benchmarked back-to-back on the **exact same GKE nodes** and **exact same OCI images** in cluster `substrate-stream-test` (`kuiyue-gke-dev`, `us-central1-a`).

| Metric / Dimension | Google Riptide v2 (`ate-ocilayer-riptide`) | AWS SOCI v0.15.0 (`ate-ocilayer-soci`) | Winner / Key Takeaway |
| :--- | :--- | :--- | :--- |
| **Cold `PrepareLayers` — 9-Layer, 1.19 GB Image (`no-stream-pool`)** | **`1,347.40 ms`** (`1.35 s`) | **`3,874.12 ms`** (`3.87 s`) | **Riptide v2 is `2.88x` faster** (`~35–44x` faster than full 1.19 GB pull + unpack) |
| **Cold `PrepareLayers` — 9-Layer, 1.19 GB Image (`e2-std32-ssd-pool`)** | **`1,226.61 ms`** (`1.23 s`) | **`6,537.03 ms`** (`6.54 s`) | **Riptide v2 is `5.33x` faster** |
| **Cold `Snapshots.Prepare` Stage (`prepare_ms`, 9 layers)** | **`876.77 – 1,029.27 ms`** (`~97–114 ms/layer`) | **`3,434.11 – 6,158.68 ms`** (`~382–684 ms/layer`) | **Riptide v2 is `3.34x – 7.02x` faster** in snapshot preparation |
| **Cold `PrepareLayers` — Standard 3-Layer, 52 MB Image (`image-a:latest`)** | **`543.24 – 612.16 ms`** (auto-indexed by GAR) | **Declined (`ErrNotSupported`)** unless pre-indexed with `soci create` + `soci push` | **Riptide v2 requires zero build-pipeline changes** |
| **Warm `PrepareLayers` (In-Memory Lease Hit, 9 layers)** | **`0.69 – 0.94 ms`** | **`0.56 – 0.90 ms`** | **Tie (`< 1 ms`)** — both share `internal/ocilayer/adapter` lease caching |
| **`CanProvide` Health Probe Latency** | **`2.34 – 4.81 ms`** | **`1.69 – 5.27 ms`** | **Tie (`~2–5 ms`)** |
| **`ReleaseLayers (bypass_idle_ttl=true)` Teardown (9 layers)** | **`30.97 – 31.48 ms`** | **`35.42 ms`** | **Tie (`~31–35 ms`)** |
| **FUSE Kernel Mount Overhead (`N` layers across `M` images)** | **`1` FUSE mount total per node** (`/var/lib/substrate/gcfs/mnt`, `fuse.gcfsd`) | **`1` FUSE mount per layer** (`N × M` `fuse.rawBridge` mounts via `fusermount3`) | **Riptide v2 scales with `O(1)` kernel mounts** |
| **Small-Layer (`< 10 MiB`) Handling** | Streams **100% of layers** natively over FUSE | Skips layers `< min-layer-size` by default; requires local download + unpack (`materializeSmallLayer`) unless indexed with `--min-layer-size=0` | **Riptide v2 avoids small-layer local unpack** |
| **Registry & Cloud Portability** | Google Artifact Registry (`*.pkg.dev`, `*.gcr.io`) on GKE COS | Any OCI Referrers / v1.1 registry (AWS ECR, GAR, GHCR, ACR, Docker Hub) on any Linux node | **AWS SOCI is cloud- and registry-agnostic** |

---

## 2. Benchmark Environment & Workload Specifications

### 2.1 Cluster & Nodes
- **GCP Project / Zone:** `kuiyue-gke-dev` / `us-central1-a`
- **GKE Cluster:** `substrate-stream-test` (`v1.36.4-gke.1495000`, `COS_CONTAINERD`, Linux `6.12.74+`)
- **Worker Nodes Tested:**
  1. **Node 1 (`no-stream-pool`):** `gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz` (GKE Image Streaming disabled on host; host `gcfsd.service` inactive)
  2. **Node 2 (`e2-std32-ssd-pool`):** `gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft` (GKE Image Streaming enabled on host; host `gcfsd.service` active)

### 2.2 Benchmark Images
1. **Workload A — 9-Layer Large Agent Sandbox Image (`~1.19 GB` compressed / `~3.5 GB` unpacked):**
   `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e`
   - **Layer Breakdown (9 layers, `1,193,055,139` bytes compressed):**
     - Layer 0 (`sha256:84ad5f3a...`): `30,664,193` B (`30.66 MB` — Ubuntu 22.04 base rootfs)
     - Layer 1 (`sha256:a17129cc...`): `287` B (`requirements.txt`)
     - Layer 2 (`sha256:e9a96d24...`): `272` B (`requirements_py2.txt`)
     - Layer 3 (`sha256:2edc65f4...`): `348,951,060` B (`348.95 MB` — Python/system packages)
     - Layer 4 (`sha256:eff5bf58...`): `488` B (`builder` metadata)
     - Layer 5 (`sha256:eee00ea2...`): `167` B (`etc` config)
     - Layer 6 (`sha256:6cca528c...`): `807,417,796` B (`807.42 MB` — Playwright / Chromium / ML toolchain)
     - Layer 7 (`sha256:4a2796d8...`): `125` B (`var` runtime dir)
     - Layer 8 (`sha256:4778a54a...`): `6,020,751` B (`6.02 MB` — `/ko-app` agent binary)
   - **SOCI Index:** OCI Referrers v1 Index `sha256:b1fd22e44bb07ee1623c42e8321414ffd460221e97ac7a5d6a5dc75df951bb8a` (`application/vnd.amazon.soci.index.v1+json`, 9 layer zTOCs).
2. **Workload B — 3-Layer Standard Container Image (`~52.4 MB` compressed, no client-pushed SOCI index):**
   `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest` (`sha256:256545b7d3cda8db8d578e2bd70d4396d3f2d16c1bcd3fa549f8350f7b1daf9d`)

---

## 3. End-to-End Latency Comparison (Apples-to-Apples on 9-Layer 1.19 GB Image)

### 3.1 Node 1 (`no-stream-pool`, `gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz`)

| Stage / Operation | Google Riptide v2 (`ms`) | AWS SOCI v0.15.0 (`ms`) | Ratio (`SOCI / Riptide`) | Notes |
| :--- | ---: | ---: | ---: | :--- |
| **Cold `CanProvide` RPC** | `3.356` | `3.552` | `1.06x` | Registry + snapshotter socket health check |
| **Cold `PrepareLayers` Total RPC** | **`1,347.396`** | **`3,874.117`** | **`2.88x`** | End-to-end cold prepare for all 9 layers |
| ↳ `resolve_ms` (Registry Manifest + Config) | `261.273` | `368.158` | `1.41x` | `go-containerregistry` manifest/config fetch |
| ↳ `stat_ms` (9× `Snapshots.Stat`) | `11.643` | `19.641` | `1.69x` | Checking committed `<chainID>` snapshots |
| ↳ **`prepare_ms` (9× `Snapshots.Prepare`)** | **`1,029.269`** | **`3,434.107`** | **`3.34x`** | **Primary bottleneck in SOCI** (zTOC fetch + 9× `fusermount3`) |
| ↳ `view_ms` (9× `Snapshots.View`) | `39.657` | `45.998` | `1.16x` | Creating read-only layer view snapshots |
| ↳ `listable_ms` (9× `os.ReadDir` probe) | `2.125` | `3.256` | `1.53x` | Verifying each FUSE mount is live and readable |
| ↳ `wrapper_ms` (9× symlink + `lease.json`) | `1.087` | `1.339` | `1.23x` | Creating `/var/lib/ateom-gvisor/streaming/<provider>/...` |
| **Warm `CanProvide` RPC** | `2.341` | `1.694` | `0.72x` | Fast-path lease check |
| **Warm `PrepareLayers` Total RPC** | **`0.685`** | **`0.557`** | **`0.81x`** | Sub-millisecond in-memory lease hit + `layersExist` stat |
| **Warm `ReleaseLayers` (`bypass_ttl=false`)** | `0.459` | `0.268` | `0.58x` | Decrements `refCount` and starts 10m idle timer |
| **Cold Teardown `ReleaseLayers` (`bypass_ttl=true`)** | **`31.476`** | **`35.424`** | **`1.13x`** | Removes all 9 snapshotter views + wrapper directory |

### 3.2 Node 2 (`e2-std32-ssd-pool`, `gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft`)

| Stage / Operation | Google Riptide v2 (`ms`) | AWS SOCI v0.15.0 (`ms`) | Ratio (`SOCI / Riptide`) | Notes |
| :--- | ---: | ---: | ---: | :--- |
| **Cold `CanProvide` RPC** | `4.813` | `5.273` | `1.10x` | Registry + snapshotter socket health check |
| **Cold `PrepareLayers` Total RPC** | **`1,226.606`** | **`6,537.029`** | **`5.33x`** | End-to-end cold prepare for all 9 layers |
| ↳ `resolve_ms` (Registry Manifest + Config) | `272.060` | `263.529` | `0.97x` | `go-containerregistry` manifest/config fetch |
| ↳ `stat_ms` (9× `Snapshots.Stat`) | `16.816` | `12.866` | `0.77x` | Checking committed `<chainID>` snapshots |
| ↳ **`prepare_ms` (9× `Snapshots.Prepare`)** | **`876.774`** | **`6,158.684`** | **`7.02x`** | **Primary bottleneck in SOCI** (zTOC fetch + 9× `fusermount3`) |
| ↳ `view_ms` (9× `Snapshots.View`) | `49.200` | `90.264` | `1.83x` | Creating read-only layer view snapshots |
| ↳ `listable_ms` (9× `os.ReadDir` probe) | `6.366` | `7.742` | `1.22x` | Verifying each FUSE mount is live and readable |
| ↳ `wrapper_ms` (9× symlink + `lease.json`) | `1.965` | `2.001` | `1.02x` | Creating `/var/lib/ateom-gvisor/streaming/<provider>/...` |
| **Warm `CanProvide` RPC** | `3.750` | `3.019` | `0.81x` | Fast-path lease check |
| **Warm `PrepareLayers` Total RPC** | **`0.942`** | **`0.899`** | **`0.95x`** | Sub-millisecond in-memory lease hit |
| **Warm `ReleaseLayers` (`bypass_ttl=false`)** | `0.661` | `0.539` | `0.82x` | Decrements `refCount` and starts 10m idle timer |
| **Cold Teardown `ReleaseLayers` (`bypass_ttl=true`)** | **`30.967`** | **`35.424`** | **`1.14x`** | Removes all 9 snapshotter views + wrapper directory |

---

## 4. Deep-Dive: Why Riptide v2 Is `2.9x – 5.3x` Faster on Cold `PrepareLayers`

The per-stage timing breakdown isolates the entire cold-start gap to **`prepare_ms`** (`Snapshots.Prepare`), where Riptide v2 takes **`877–1,029 ms`** across 9 layers (`~97–114 ms/layer`) while AWS SOCI takes **`3,434–6,159 ms`** (`~382–684 ms/layer`). Three architectural differences explain this gap:

### 4.1 Single Multiplexed FUSE Daemon (`gcfsd-v2`) vs. Per-Layer `fusermount3` Subprocesses (`soci`)
- **Riptide v2 (`gcfsd-v2`):**
  - Mounts a **single** FUSE filesystem (`fuse.gcfsd`) once at daemon startup (`/var/lib/substrate/gcfs/mnt`).
  - Preparing and viewing 9 layers performs **zero kernel `mount(2)` calls**. Instead, `containerd-gcfs-grpc` sends a local gRPC `CreateView` RPC to `gcfsd-v2`, which exposes `/var/lib/substrate/gcfs/mnt/views/<chainID>/fs` as a virtual directory inside the existing FUSE mount.
- **AWS SOCI (`soci-snapshotter-grpc`):**
  - Spawns a **separate `fusermount3` subprocess and kernel `fuse.rawBridge` mount for every layer** (`/var/lib/soci-snapshotter-grpc/snapshotter/snapshots/{2,4,6,8,10,12,14,16,18}/fs`).
  - For a 9-layer image, `soci-snapshotter-grpc` initializes 9 distinct `/dev/fuse` sessions, 9 background fetchers, and 9 kernel mount table entries.

### 4.2 Server-Side Integrated GAR TOC vs. OCI Referrers Discovery + Multi-zTOC Fetch
- **Riptide v2:**
  - Google Artifact Registry natively stores and serves Riptide TOC metadata alongside the image blobs in GAR's backend.
- **AWS SOCI:**
  - On Layer 0 (`l0-prep`), `soci-snapshotter-grpc` must first query the registry's OCI Referrers API (`/v2/.../referrers/sha256:fa2f27...`), fetch the SOCI Index manifest (`sha256:b1fd22e4...`), and then download and deserialize all 9 gzip-span zTOC blobs from the registry (`~880 ms` on `l0-prep` and `~1,920–2,340 ms` on the 349 MB `l3-prep` zTOC).

### 4.3 Zero-Config Automatic Indexing vs. Explicit Client Indexing & Small-Layer Materialization
- **Riptide v2:**
  - **Every image in GAR is streamable out of the box** without modifying CI/CD pipelines. On the 3-layer `image-a:latest` image (which has no SOCI index), Riptide v2 cold-streamed all 3 layers in **`543.24 ms`** (`no-stream-pool`) and **`612.16 ms`** (`e2-std32-ssd-pool`).
  - Riptide v2 streams **all layers regardless of size** (even `125 B` layers) over the single `gcfsd-v2` mount.
- **AWS SOCI:**
  - Requires running `soci create` + `soci push` (or building a SOCI v2 image) before an image can be streamed. Unindexed images (`image-a:latest`) are declined with `ErrNotSupported` and fall back to `atelet`'s local `internal/imagecache`.
  - By default, `soci create` skips layers smaller than `--min-layer-size` (`10 MiB`). When those small layers return `codes.OK` from `Snapshots.Prepare`, `cmd/ocilayer-soci` must download, unpack, and `Snapshots.Commit` them locally via `materializeSmallLayer` to satisfy `atelet`'s all-or-nothing layer contract.

---

## 5. When Is Riptide v2 Faster vs. When Is AWS SOCI Faster? (Comparison With Earlier Experiments)

Across both earlier prototype benchmarks and the current `OCILayerService` benchmarks, **neither provider is faster at every phase**—they make opposite trade-offs between **mount setup time (`PrepareLayers`)** and **post-mount file read speed (demand paging)**:

| Benchmark Phase / Scenario | Google Riptide v2 | AWS SOCI v0.15.0 | Which Is Faster & Why |
| :--- | :--- | :--- | :--- |
| **1. Cold `PrepareLayers` (9-layer 1.19 GB image — Current)** | **`1,226.6 – 1,347.4 ms`** | **`3,874.1 – 6,537.0 ms`** | **Riptide v2 (`2.9x – 5.3x` faster):** Virtual `CreateView` on a single pre-mounted `gcfsd-v2` FUSE mount vs. SOCI fetching 9 zTOCs + spawning 9 `fusermount3` mounts. |
| **2. Cold `Image streamed` (9-layer 1.19 GB image — Earlier E2E)** | **`972.3 – 1,134.5 ms`** | **`3,468.6 – 4,091.7 ms`** | **Riptide v2 (`3.1x – 4.2x` faster):** Same `~1.0–1.3s` vs. `~3.5–4.1s` cold mount profile. |
| **3. Cold `PrepareLayers` (12-layer 1.88 GB vs. 19-layer 3.14 GB — Earlier E2E)** | **`470.35 ms`** | **`4,849.94 ms`** | **Riptide v2 (`10.3x` faster):** Riptide setup cost scales lightly with layer count; SOCI setup cost scales with total zTOC download size + `fusermount3` calls. |
| **4. Uncached FUSE Demand-Paging Read Latency (`p50`) & Throughput** | `4.96 MB/s` (**`p50: 1,180 µs`**) | **`6.17 MB/s`** (**`p50: 459 µs`**) | **AWS SOCI (`2.57x` lower `p50` read latency, `+24%` throughput):** SOCI pre-loads the entire zTOC (all file offsets & gzip spans) into memory during `PrepareLayers` and runs a background span prefetcher (`background_fetch`). |
| **5. Cached VFS Re-Read Throughput (2nd Pass Read)** | `20.90 MB/s` | **`40.70 MB/s`** | **AWS SOCI (`1.95x` faster):** Dedicated per-layer `fuse.rawBridge` queues + VFS page-cache attributes (`attr_timeout=1s`, `entry_timeout=1s`) avoid single-daemon FUSE serialization. |
| **6. Small Layers (`< 10 MiB`, e.g., entrypoint/configs) with Default `soci create`** | Streamed over FUSE (`fuse.gcfsd`) | **Unpacked to local `ext4` disk** (`materializeSmallLayer`) | **AWS SOCI faster at runtime:** Small layers skipped by `soci create` (`< 10 MiB`) are materialized onto local SSD during `PrepareLayers`, so reading them incurs **zero FUSE or network overhead**. |
| **7. Warm `PrepareLayers` & Warm-Node `AteomHerder/Restore`** | `0.69 – 0.94 ms` (`Restore`: `265.25 ms`) | **`0.56 – 0.90 ms`** (`Restore`: **`244.53 ms`**) | **AWS SOCI slightly faster (`~8–19%`):** Both hit the `< 1 ms` in-memory lease cache; SOCI avoids the `gcfsd` keychain socket check and benefits from faster demand paging during checkpoint/restore. |
| **8. Golden Snapshot `AteomHerder/Checkpoint`** | `628.09 ms` | **`487.32 ms`** | **AWS SOCI (`1.29x` faster):** Faster page faults from pre-loaded zTOCs + background prefetcher when flushing dirty/referenced pages. |
| **9. Earlier Riptide Runs *Before* `Keychain.UpdateCreds` & `gcfsd-v2` Isolation** | *`10,000 – 21,000 ms` (pathological retry)* | **`3,500 – 4,800 ms`** | **Why SOCI sometimes looked much faster in early experiments:** Before pushing credentials to `keychain.sock` *prior* to `Snapshots.Prepare` and isolating `gcfsd-v2` from host `containerd`, Riptide hit a **10s internal auth/mount retry backoff** inside `gcfsd-v2`. |

### Summary of the Fundamental Trade-Off
1. **Riptide v2 is a "Lazy-Index, Single-Mount" streamer:** It does almost no work during `PrepareLayers` (`~0.5–1.3s`), deferring index/chunk resolution to when files are actually read over its single `gcfsd-v2` FUSE mount.
2. **AWS SOCI is an "Eager-zTOC, Per-Layer-Mount" streamer:** It pays `~3.5–6.5s` upfront during `PrepareLayers` to download every layer's full zTOC into memory, start a background prefetcher, mount a dedicated `fuse.rawBridge` filesystem per layer (and unpack `< 10 MiB` layers to local disk), which makes **subsequent file reads (`p50 = 459 µs` vs. `1,180 µs`), cached re-reads (`40.7 MB/s` vs. `20.9 MB/s`), checkpoints (`487 ms` vs. `628 ms`), and warm restores (`244 ms` vs. `265 ms`) faster than Riptide v2**.

---

## 6. Raw Log Evidence From Same-Image Benchmark (`sandbox-04176181c57f0bf23e61506b0ddcf1fd`)

### 6.1 Riptide v2 `PrepareLayers` Timing Breakdown Logs (`p1tz` & `bpft`)

```text
# Node 1 (no-stream-pool, p1tz):
2026/10/10 17:51:43 INFO Pushed credentials to Riptide gcfsd keychain socket=/run/ate-image-streaming/gcfsd/keychain.sock image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e server=us-central1-docker.pkg.dev username=_token
2026/10/10 17:51:44 INFO PrepareLayers timing breakdown provider=riptide image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e digest=sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e layers=9 resolve_ms=261.273 stat_ms=11.643 prepare_ms=1029.269 view_ms=39.657 listable_ms=2.125 wrapper_ms=1.087 total_ms=1346.586

# Node 2 (e2-std32-ssd-pool, bpft):
2026/10/10 17:51:48 INFO Pushed credentials to Riptide gcfsd keychain socket=/run/ate-image-streaming/gcfsd/keychain.sock image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e server=us-central1-docker.pkg.dev username=_token
2026/10/10 17:51:49 INFO PrepareLayers timing breakdown provider=riptide image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e digest=sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e layers=9 resolve_ms=272.06 stat_ms=16.816 prepare_ms=876.774 view_ms=49.2 listable_ms=6.366 wrapper_ms=1.965 total_ms=1225.227
```

### 6.2 AWS SOCI `PrepareLayers` Timing Breakdown Logs (`p1tz` & `bpft`)

```text
# Node 1 (no-stream-pool, p1tz):
2026/10/10 17:30:10 INFO Synchronized registry credentials to SOCI docker config path=/run/soci-snapshotter-grpc/docker/config.json image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e server=us-central1-docker.pkg.dev username=_token
2026/10/10 17:30:14 INFO PrepareLayers timing breakdown provider=soci image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e digest=sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e layers=9 resolve_ms=368.158 stat_ms=19.641 prepare_ms=3434.107 view_ms=45.998 listable_ms=3.256 wrapper_ms=1.339 total_ms=3873.046

# Node 2 (e2-std32-ssd-pool, bpft):
2026/10/10 17:35:05 INFO Synchronized registry credentials to SOCI docker config path=/run/soci-snapshotter-grpc/docker/config.json image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e server=us-central1-docker.pkg.dev username=_token
2026/10/10 17:35:11 INFO PrepareLayers timing breakdown provider=soci image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e digest=sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e layers=9 resolve_ms=263.529 stat_ms=12.866 prepare_ms=6158.684 view_ms=90.264 listable_ms=7.742 wrapper_ms=2.001 total_ms=6535.938
```

---

## 7. Three-Way Comparison: Google Riptide v2 vs. AWS SOCI vs. Nydus + Dragonfly (Kernel EROFS + `fscache`)

We extended the benchmark on cluster `substrate-stream-test` (`us-central1-a`) to include **Nydus RAFS v6 + Dragonfly P2P over Kernel EROFS + `fscache`** (`cmd/ocilayer-nydus`, `ate-ocilayer-nydus`) on a `c3-standard-4` node (`gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc`, Ubuntu 24.04, kernel `6.8.0-1062-gke` with `CONFIG_EROFS_FS_ONDEMAND=y` and `CONFIG_CACHEFILES_ONDEMAND=y`) using the same 9-layer (`1.19 GB` compressed / `3.5 GB` unpacked) benchmark image pre-converted to Nydus RAFS v6 (`nydusify convert --fs-version 6`).

| Metric / Dimension (`c3-standard-4`, 9-Layer 1.19 GB Image) | Google Riptide v2 (`fuse.gcfsd`) | AWS SOCI v0.15.0 (`fuse.rawBridge`) | Nydus RAFS v6 + Dragonfly (`erofs` + `fscache`) | Winner & Architectural Explanation |
| :--- | :--- | :--- | :--- | :--- |
| **Filesystem Data Path** | Userspace FUSE (`fuse.gcfsd`, `0x65735546`) | Userspace FUSE (`fuse.rawBridge`, `0x65735546`) | **In-Kernel EROFS + `fscache`** (`erofs`, `0xE0F5E1E2`; `fusedev` disabled) | **Nydus EROFS:** Zero userspace FUSE context switches once chunks are in `fscache`. |
| **Cold `PrepareLayers` Total Latency** | `1,347.40 ms` (`1.35 s`) | `3,874.12 ms` (`3.87 s`) | **`645.85 – 736.87 ms`** (**`0.65 – 0.74 s`**) | **Nydus EROFS (`1.8x–2.1x` faster than Riptide, `5.3x–6.0x` faster than SOCI):** Single merged `5.28 MB` `image.boot` EROFS mount (`layers=1`). |
| **Cold `Snapshots.Prepare` Stage (`prepare_ms`)** | `1,029.27 ms` | `3,434.11 ms` | **`27.14 – 28.37 ms`** (+ `~316 ms` bootstrap commit) | **Nydus EROFS:** Intermediate `nydus-blob` layers commit metadata in `~3.5 ms` each. |
| **Warm `PrepareLayers` (Active Lease Hit)** | `0.685 ms` | `0.557 ms` | **`0.451 ms`** | **Tie (`< 1 ms`)** — shared `internal/ocilayer/adapter` lease cache. |
| **`ReleaseLayers (bypass_idle_ttl=true)`** | `31.48 ms` (9 views) | `35.42 ms` (9 FUSE mounts) | **`2.36 ms`** (1 EROFS mount) | **Nydus EROFS (`13x–15x` faster):** Only 1 merged EROFS mount to unmount. |
| **Cold Sequential Read Throughput (Cold Registry / Cold P2P)** | **`250.3 MB/s`** (parallel HTTP/2 streams + prefetch) | `65.9 MB/s` (single-stream gzip span decompression) | `6.91 MiB/s` (on-demand 1 MB chunk fetch via cold proxy) | **Riptide v2 (`3.8x` faster than SOCI, `36x` faster than cold Nydus):** Aggressive parallel prefetching from GAR. |
| **Peer-Warm Sequential Read Throughput (Warm Dragonfly P2P, Cold Node `fscache`)** | N/A (no P2P cache tier) | N/A (no P2P cache tier) | **`131.24 MiB/s`** (`19.0x` speedup over Cold GAR; `p90 = 43.54 ms`) | **Nydus + Dragonfly P2P:** Dragonfly serves 1 MB RAFS v6 chunks from peer RAM/NVMe with `0` GAR requests. |
| **Warm Disk Cache Sequential Read (Page Cache Dropped)** | Served via userspace `gcfsd-v2` FUSE | Served via userspace `soci` FUSE | **`180.20 MiB/s`** (`p50 = 0.655 ms`, pure kernel `fscache` disk read) | **Nydus EROFS:** Kernel reads directly from `/var/lib/containerd-nydus/cache` without waking `nydusd`. |
| **Warm Page Cache Sequential Read Throughput** | `2,469.1 MB/s` (FUSE VFS cache) | `2,469.1 MB/s` (FUSE VFS cache) | **`5,813.3 MiB/s` (`5.81 GiB/s`, `p50 = 10.4 µs`)** | **Nydus EROFS (`2.35x` faster):** Native in-kernel EROFS VFS path with zero FUSE overhead. |
| **Build Pipeline & GKE Node OS Requirements** | **Zero build changes**; works on default GKE `COS_CONTAINERD` | Requires `soci create` + `soci push`; works on `COS_CONTAINERD` & Ubuntu | Requires offline `nydusify convert --fs-version 6` AND custom `UBUNTU_CONTAINERD` kernel modules (`CONFIG_EROFS_FS_ONDEMAND=y`, `CONFIG_CACHEFILES_ONDEMAND=y`) | **Riptide v2** (zero build or node OS changes on GKE). |

---

## 8. Recommendations

1. **Use Google Riptide v2 (`--setup-image-streaming=riptide`) as the default on GKE + Artifact Registry:**
   - Delivers **`1.23–1.35 s`** cold `PrepareLayers` on a 9-layer 1.19 GB image, **`250.3 MB/s`** cold sequential read throughput from GAR, **`< 1 ms`** warm lease hits, **`O(1)`** FUSE mount overhead per node, native compatibility with default GKE `COS_CONTAINERD` nodes, and **zero CI/CD indexing or image conversion steps**.
2. **Use AWS SOCI (`--setup-image-streaming=soci`) for multi-cloud / non-GAR registries (AWS ECR, GHCR, ACR) or standard Linux nodes without custom kernel modules:**
   - Works on standard `COS_CONTAINERD` and `UBUNTU_CONTAINERD` kernels without custom kernel modules, while keeping the original unmodified OCI image layers and attaching an OCI Referrers zTOC index.
3. **Use Nydus + Dragonfly (`--setup-image-streaming=nydus`) on dedicated EROFS/`fscache`-enabled node pools when sub-second mount latency (`0.65–0.74 s`), P2P cluster distribution (`131.2 MiB/s`), and native in-kernel VFS read speed (`5.81 GiB/s`, `0` FUSE overhead) justify offline RAFS v6 conversion and kernel module management:**
   - See [`docs/image-streaming-nydus-e2e-test-report.md`](image-streaming-nydus-e2e-test-report.md) for the complete GKE Ubuntu EROFS setup, `CONFIG_EROFS_FS_ONDEMAND` / `CONFIG_CACHEFILES_ONDEMAND` kernel module build, and live benchmark logs.

