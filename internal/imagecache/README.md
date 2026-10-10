# imagecache — node-local OCI image layer cache

`internal/imagecache` implements substrate's node-local OCI image cache: a
content-addressed pool of **unpacked image layers**, stored once per node and
shared by every actor on it, plus the machinery that composes an actor's
rootfs from those layers as an overlayfs mount instead of extracting the
image on every run.

It replaces the previous design (an in-memory LRU of flattened image
tarballs in `atelet`, re-untarred into every bundle on every actor
start/resume) and is the Phase 1 implementation of
[#463](https://github.com/agent-substrate/substrate/issues/463), addressing
[#120](https://github.com/agent-substrate/substrate/issues/120),
[#166](https://github.com/agent-substrate/substrate/issues/166),
[#228](https://github.com/agent-substrate/substrate/issues/228) and
[#437](https://github.com/agent-substrate/substrate/issues/437).

What it buys, concretely:

- Actor start/resume composes the rootfs with **one overlay mount
  (milliseconds)** instead of a full image extraction (tens of seconds for
  GB-scale images). `Restore timing breakdown` logs show
  `ate.actor.restore.duration.oci_unpack` dropping from ~15–20 s to
  single-digit milliseconds on warm nodes.
- Layers shared between images are **downloaded and unpacked once per
  node**, not once per image; actors sharing layers also share page cache.
- The cache is **on disk and survives atelet restarts and node reboots**
  (the old in-memory cache was lost on every restart, and its unbounded heap
  retention could OOM atelet — #437).
- **Tag refs are cacheable**: a tag costs one `HEAD` request to resolve to a
  manifest digest (the only safe cache key for mutable tags); digest refs
  hit the cache with zero network I/O.
- Memory use during pulls is O(stream buffers), independent of image size
  (the old `mutate.Extract` path buffered entire flattened images — #120).

## The privilege split

The design is shaped by an existing substrate boundary: **atelet runs as
plain root with every Linux capability dropped** ("atelet does no mounts" —
see `manifests/ate-install/atelet.yaml`), while the **ateom worker pods are
privileged** and own all mounts on the node. The module is split accordingly:

| Half | Runs in | Files | Needs |
|---|---|---|---|
| Store: pull, parse, unpack, record | atelet | `imagecache.go`, `rangefetch.go`, `unpack.go`, `spec.go` (portable) | nothing but file I/O |
| Consumer: finalize, mount, unmount | ateom-gvisor / ateom-microvm | `bundle_linux.go` (`//go:build linux`) | `CAP_MKNOD`, `CAP_SYS_ADMIN` |

The two halves communicate through the filesystem only: the shared cache
directory (on the `/var/lib/ate` hostPath, so the same absolute
paths resolve in every pod) and a small per-bundle spec file.

Because the consumer mounts the overlay **in its own mount namespace** —
exactly where the workload resolves it (runsc's gofer for gVisor, virtiofsd
for the micro-VM) — no Kubernetes mount-propagation configuration is needed
anywhere.

## On-disk layout

```
<cache-root>/                        default: /var/lib/ate/image-cache
  version                            layout version marker ("1")
  layers/sha256/<diffid-hex>/
      fs/                            the unpacked layer tree (an overlay lowerdir)
      whiteouts.json                 whiteout state recorded at unpack time
      finalized                      marker written by FinalizeLayer (consumer side)
      size                           byte count recorded at unpack (lazily
                                     backfilled for older layers), so sizing
                                     the pool never walks trees
  layers/sha256/.tmp-*/              in-flight unpack (swept at startup)
  layers/sha256/.rm-*/               retired by eviction, awaiting async
                                     removal (swept at startup)
  manifests/sha256/<digest-hex>.json image config + ordered diffID list; the
                                     file's mtime doubles as the image's
                                     last-use timestamp
```

A layer directory that exists is always complete: unpack streams into a
`.tmp-*` sibling and moves it into place with a single atomic rename.
Startup recovery (`New`) sweeps leftover `.tmp-*` and `.rm-*` dirs,
verifies the layout version, and reclaims orphaned layers (see Garbage
collection below). An "image" is nothing but a manifest record listing
layer diffIDs in order — layers shared by N images exist once.

## Pull path (atelet: `Store.EnsureImage`)

1. **Resolve** the ref. Digest refs are parsed directly; tag refs cost one
   `remote.Head`. Localhost/loopback registries are rewritten for kind
   (`--localhost-registry-replacement`) and pulled over plain HTTP; gcr.io /
   pkg.dev registries get the configured GCP authenticator.
2. **Cache check**: if the manifest record exists and every layer dir is
   present, return with no network I/O. Missing layers (only) are re-pulled.
3. **Pull** by resolved digest: layers are sorted largest-first (LPT) and
   downloaded in parallel (bounded at 8), each streamed through a decoupled
   3-stage pipeline (parallel HTTP `Range` download → decompress → parallel
   untar) directly into the pool. Concurrent pulls of the same image or layer
   are collapsed with singleflight, so simultaneous actor starts never
   duplicate work — and each completed layer lands individually, so an
   interrupted pull makes incremental progress across retries.
4. **Unpack** (`unpackLayer`) is the repo's hardened untar: `os.Root`
   confinement (path traversal and symlink/hardlink escapes are refused),
   "later entry wins" within a layer, read-only-dir handling that works
   without `CAP_DAC_OVERRIDE`, and creation of parent directories that the
   layer tar omits (they may exist only in lower layers). Whiteout entries
   (`.wh.*`) are **not** written into the tree — overlayfs whiteouts are
   char devices atelet cannot create — they are recorded in
   `whiteouts.json` for the consumer to materialize.
5. **Record**: the image config + diffID list is written under the
   requested digest (and the per-platform child digest for multi-arch refs).

`prepareOCIDirectory` in atelet then writes `rootfs-overlay.json`
(`OverlaySpec`) into the bundle, listing the layer
directories bottom-first plus any `ExtraDirs` (in-rootfs bind-mount targets,
e.g. the actor identity mount at `/run/ate`), and creates the empty
bundle-local `rootfs/`, `upper/`, and `work/` directories.

### 3-Part Streaming Pull & Unpack Architecture

Unlike `containerd`, which downloads compressed layer blobs into an on-disk
content store (`/var/lib/containerd/io.containerd.content.v1.content/ingest/...`),
`fsync`s and commits each compressed blob to disk, and then sequentially
decompresses and unpacks layers in parent-to-child chain order
(`Layer 0 -> Layer 1 -> ... -> Layer N`), `imagecache` streams each layer
directly from the registry through an in-memory 3-stage pipeline into its
content-addressed target directory (`layers/sha256/<diffid-hex>/fs/`) with zero
compressed-blob disk writes and full cross-layer concurrency:

```mermaid
flowchart LR
    subgraph Stage1["Stage 1: Parallel HTTP Range Fetcher (rangefetch.go)"]
        R0["Chunk 0: HTTP Range [0..16MiB)"]
        R1["Chunk 1: HTTP Range [16MiB..32MiB)"]
        RN["Chunk k: HTTP Range [16k..16(k+1)MiB)"]
        WIN["Ordered Sliding Window (256 KiB pooled buffers)\n+ Streaming Compressed SHA-256 Verifier"]
        R0 & R1 & RN --> WIN
    end
    subgraph Stage2["Stage 2: Fast Decompressor (imagecache.go)"]
        DEC["Hardware-Accelerated igzip -d (ISA-L SIMD)\nFallback: klauspost/compress (gzip / zstd)"]
        WIN --> DEC
    end
    subgraph Stage3["Stage 3: Parallel Tar Unpacker (unpack.go)"]
        TAR["Sequential tar.Next() + DiffID SHA-256"]
        POOL["8-Worker Async File Writer Pool\n(64 KiB / 256 KiB / 2 MiB tiered pools)"]
        FS["os.Root Layer Tree + Deferred Sync"]
        DEC --> TAR --> POOL --> FS
    end
```

1. **Containerd-Style Parallel HTTP `Range` Chunk Fetcher (`rangefetch.go`)**:
   - Layers with compressed size `>= 16 MiB` (`rangeFetchThreshold`, matching `containerd`'s `concurrent_layer_fetch_buffer`) are registered by digest before `layer.Compressed()` opens the blob stream.
   - `parallelRangeTransport` intercepts the initial `GET` request and probes `Range: bytes=0-16777215` (following `302 Found` GCS/S3 CDN redirects with SSRF validation).
   - If the server responds with `206 Partial Content`, `parallelRangeBody` spawns up to `rangeLayerConcurrency = 16` concurrent chunk workers (bounded node-wide by `rangeGlobalConcurrency = 24`) over a dedicated HTTP/1.1 connection pool (`MaxConnsPerHost = 64`) to fetch `16 MiB` byte ranges in parallel.
   - Each worker reads full `256 KiB` pooled buffers via `io.ReadFull` (coalescing 16 KiB TLS records so each 64-slot channel holds the full 16 MiB chunk without blocking) into an ordered sliding window (`rangeWindowChunks = 96`, up to `1.5 GiB` in-flight). As `Read()` consumes each `256 KiB` slice, the buffer is immediately recycled to `streamChunkPool`.
   - If a chunk stream drops mid-transfer, `fetchChunkRange` retries only the remaining byte sub-range (`bytes=<offset+n>-<end>`). If a registry ignores `Range` and returns `200 OK`, `parallelRangeTransport` falls back transparently to the single-stream response.
2. **Hardware-Accelerated SIMD Decompression Fast-Path (`igzip`) + Pure-Go Fallback (`imagecache.go`)**:
   - `igzipBinaryPath` checks once per process (`sync.OnceValue`) for Intel ISA-L's `igzip` binary across container paths (`PATH`, `/usr/bin/igzip`, `/var/lib/ate/bin/igzip`) and host mounts (`/proc/1/root/usr/bin/igzip`, `/host/usr/bin/igzip`, invoked via the host's `ld-linux` ELF interpreter and `--library-path` so it works inside distroless `ko` containers when host paths are visible), validating the binary at startup against a 20-byte empty gzip stream (`verifyIgzipBinary`).
   - When available, `openLayerStream` pipes `stage1` through `igzip -d -c` (`newIgzipReader`) wrapped in `stickyEOFReader` so any upstream compressed-stream SHA-256 verification error at `io.EOF` is never masked by subprocess exit status.
   - When `igzip` is unavailable (or `ATE_DISABLE_IGZIP` is set), `openLayerStream` falls back automatically to pure-Go `klauspost/compress/gzip` (and `klauspost/compress/zstd` for `.tar.zst` layers).
   - **Availability**: Intel ISA-L (`libisal`) includes hand-tuned assembly kernels for **x86_64** (`AVX-512`, `AVX2`, `SSE4.2`) and **ARM64 / `aarch64`** (`NEON` + CRC32/Crypto extensions). `/usr/bin/igzip` (`libisal.so.2`) is **pre-installed by default on GKE Container-Optimized OS (COS)** nodes and available in standard packages on Debian/Ubuntu (`isal`), Alpine (`isal`), and RHEL/Rocky/Fedora (`libisal`).
3. **Parallel Regular-File Unpacker & LPT Scheduling (`unpack.go`, `imagecache.go`)**:
   - `Store.pullWith` sorts missing layers by compressed size descending (`layerPullConcurrency = 8`, `remote.WithJobs(8)`) so the largest layers begin downloading immediately.
   - `unpackLayer` reads tar headers sequentially (computing the uncompressed `DiffID` SHA-256 and memoizing created directories) and dispatches regular files `<= 2 MiB` to an 8-worker goroutine pool backed by three size-tiered `sync.Pool`s (`64 KiB`, `256 KiB`, `2 MiB`).
   - Files `> 2 MiB` stream directly from the tar reader using a pooled `2 MiB` copy buffer without draining in-flight small-file workers unless the large file's own path has a pending write.
   - Strict OCI `later-entry-wins` and hardlink ordering semantics are preserved via `pendingPaths` conflict tracking.

### Live-Cluster Benchmark Comparison (`kuiyue-gke-dev`)

Benchmarked on the GKE cluster `substrate-stream-test` (`us-central1-a`, GCP project `kuiyue-gke-dev`) across two node types using a **9-layer `.tar.gz` image (`1.19 GB` compressed / `~3.45 GB` unpacked)** where Layer 6 (`807.4 MB` compressed / `2.47 GB` unpacked) and Layer 3 (`349.0 MB` compressed / `1.00 GB` unpacked) account for **96.8%** of total compressed bytes:

#### Node Pool A: `e2-standard-32` (32 vCPUs AMD EPYC 7B12, 128 GB RAM, 300 GB `pd-ssd`)
*(Note: Substrate pulled all 9 layers (`1.20 GB` compressed) on every run; on `e2-standard-32`, a running pod pinned base layers 0–5 (`~380 MB`) in `containerd`, so `containerd` cold-pulled only the `807.4 MB` Layer 6 + top layers (`~823 MB`).)*

| Scenario | Total Pull + Unpack Time | Network Active Window (`>5 MB/s`) | Active Net Rx Throughput | Peak 1s Net Rx Throughput | CPU Utilization (avg / peak) | Total CPU (`core-s`) | Disk Write Total |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **4.1a Substrate (`main` baseline)** *(9 layers, 1.23 GB)* | **26.8 s** | 25.22 s | 48.8 MB/s (`0.39 Gbps`) | 80.0 MB/s (`0.64 Gbps`) | 2.30 / 3.56 cores | 62.35 | 218.9 MB *(page-cached)* |
| **4.1b Substrate (`ca29640b` + `159de77b`)** *(9 layers, 1.20 GB)* | **12.5 s** | 11.61 s | 103.3 MB/s (`0.83 Gbps`) | 205.1 MB/s (`1.64 Gbps`) | 3.18 / 7.70 cores | 41.67 | 215.8 MB *(page-cached)* |
| **4.1c Substrate (3-Part Opt + `klauspost/gzip`)** *(9 layers, 1.21 GB)* | **9.09 s** | **1.01 s** | **1,191.2 MB/s (`9.53 Gbps`)** | **1,198.0 MB/s (`9.58 Gbps`)** | **3.93 / 9.24 cores** | 37.18 | 215.2 MB *(page-cached)* |
| **4.1d Substrate (3-Part Opt + `igzip` SIMD)** *(9 layers, 1.20 GB)* | **9.40 s** | **1.52 s** | **792.4 MB/s (`6.34 Gbps`)** | **1,870.2 MB/s (`14.96 Gbps`)** | **3.51 / 12.62 cores** | **35.09** | 214.4 MB *(page-cached)* |
| **4.2 K8s `containerd` Default** (`dl=3, chunk_buf=0`, *807 MB L6*) | **27.28 s** | 8.56 s | 95.3 MB/s (`0.76 Gbps`) | 152.1 MB/s (`1.22 Gbps`) | 1.06 / 2.07 cores | 28.83 | 854.3 MB |
| **4.4a K8s `containerd` Parallel Chunks** (`dl=10, chunk_buf=16MiB`, *807 MB L6*) | **27.02 s** | 1.52 s | 545.0 MB/s (`4.36 Gbps`) | 608.7 MB/s (`4.87 Gbps`) | 1.44 / 3.31 cores | 38.78 | 909.2 MB |
| **4.4b K8s `containerd` Parallel Chunks** (`dl=16, chunk_buf=64MiB`, *807 MB L6*) | **22.27 s** | 1.01 s | 814.8 MB/s (`6.52 Gbps`) | 1,122.6 MB/s (`8.98 Gbps`) | 1.28 / 3.96 cores | 28.55 | 854.6 MB |

#### Node Pool B: `c3-standard-4` (4 vCPUs Intel Xeon Platinum 8481C Sapphire Rapids, 16 GB RAM, 100 GB `pd-balanced`, 100% Cold Pull of All 9 Layers)

| Scenario | Total Pull + Unpack Time | Network Active Window (`>5 MB/s`) | Active Net Rx Throughput | Peak 1s Net Rx Throughput | CPU Utilization (avg / peak) | Total CPU (`core-s`) | Disk Write Total (avg / peak MB/s) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **4.1a Substrate (`main` baseline)** | **23.9 s** | 23.34 s | 54.9 MB/s (`0.44 Gbps`) | 96.3 MB/s (`0.77 Gbps`) | 1.97 / 3.14 cores | 47.45 | 2,254 MB (93.8 / 217.6 MB/s) |
| **4.1b Substrate (`ca29640b` + `159de77b`)** | **14.0 s** | 13.59 s | 92.9 MB/s (`0.74 Gbps`) | 146.5 MB/s (`1.17 Gbps`) | 2.14 / 3.82 cores | 30.31 | 1,852 MB (131.1 / 210.4 MB/s) |
| **4.1c Substrate (3-Part Opt + `klauspost/gzip`)** | **12.69 s** | **1.51 s** | **836.5 MB/s (`6.69 Gbps`)** | **1,232.9 MB/s (`9.86 Gbps`)** | **2.24 / 3.90 cores** | 29.13 | 1,587 MB (122.0 / 210.5 MB/s) |
| **4.1d Substrate (3-Part Opt + `igzip` SIMD)** | **11.23 s** | **1.51 s** | **836.5 MB/s (`6.69 Gbps`)** | **1,460.5 MB/s (`11.68 Gbps`)** | **1.71 / 3.82 cores** | **21.89** | 1,684 MB (131.8 / 201.3 MB/s) |
| **4.2 K8s `containerd` Default** (`dl=3, chunk_buf=0`) | **26.77 s** | 3.02 s | 418.2 MB/s (`3.35 Gbps`) | 797.1 MB/s (`6.38 Gbps`) | 1.18 / 3.02 cores | 31.59 | 3,371 MB (125.9 / 195.7 MB/s) |
| **4.4a K8s `containerd` Parallel Chunks** (`dl=10, chunk_buf=16MiB`) | **26.47 s** | 2.52 s | 516.8 MB/s (`4.13 Gbps`) | 751.9 MB/s (`6.01 Gbps`) | 1.23 / 3.52 cores | 32.56 | 3,243 MB (122.5 / 182.0 MB/s) |
| **4.4b K8s `containerd` Parallel Chunks** (`dl=16, chunk_buf=64MiB`) | **26.16 s** | 1.52 s | 849.6 MB/s (`6.80 Gbps`) | 1,364.3 MB/s (`10.91 Gbps`) | 1.20 / 3.66 cores | 31.34 | 3,449 MB (131.9 / 180.5 MB/s) |

#### Multi-Scale Benchmark Across Image Sizes: `16 KB`, `1 MB`, `1 GB`, `3.45 GB`, `10 GB`, and `50 GB` (`e2-standard-32`, 32 vCPUs, 300 GB `pd-ssd`)

| Image Scale (Unpacked / Compressed, Layers) | Variant | Cold Pull + Unpack Time | Speedup vs `main` | Network Active Window (`>5 MB/s`) | Active Net Rx Throughput | Peak Net Rx Throughput | CPU Utilization (avg / peak) | Total CPU (`core-s`) | Disk Write Throughput (avg / peak) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **`16 KB`** (`16 KiB` / `8 KiB`, 1 layer) | **1. Baseline (`main`)** | **0.365 s** | `1.00x` | `<0.5 s` (`0.04 MB`) | — | 0.05 MB/s | 0.33 / 0.40 cores | 0.27 | — |
| | **2. 3-Part Opt (`klauspost/gzip`)** | **0.383 s** | `0.95x` | `<0.5 s` (`0.04 MB`) | — | 0.07 MB/s | 0.73 / 1.04 cores | 0.76 | — |
| | **3. 3-Part Opt (`igzip` SIMD)** | **0.412 s** | `0.89x` | `<0.5 s` (`0.06 MB`) | — | 0.05 MB/s | 0.60 / 0.74 cores | 1.02 | — |
| **`1 MB`** (`1.0 MiB` / `0.50 MiB`, 1 layer) | **1. Baseline (`main`)** | **0.398 s** | `1.00x` | `<0.5 s` (`0.59 MB`) | — | 1.92 MB/s | 0.61 / 0.76 cores | 1.07 | — |
| | **2. 3-Part Opt (`klauspost/gzip`)** | **0.374 s** | **`1.06x`** | `<0.5 s` (`0.59 MB`) | — | 1.27 MB/s | 0.67 / 1.02 cores | 1.30 | — |
| | **3. 3-Part Opt (`igzip` SIMD)** | **0.365 s** | **`1.09x`** | `<0.5 s` (`0.58 MB`) | — | 1.87 MB/s | 0.47 / 0.66 cores | 1.08 | — |
| **`1 GB`** (`1.0 GiB` / `461 MiB`, 4 layers: `512/256/192/64 MiB`) | **1. Baseline (`main`)** | **6.95 s** | `1.00x` | 6.84 s | 72.8 MB/s (`0.58 Gbps`) | 88.5 MB/s (`0.71 Gbps`) | 2.35 / 3.34 cores | 17.23 | page-cached |
| | **2. 3-Part Opt (`klauspost/gzip`)** | **1.90 s** | **`3.66x`** | **0.51 s** | **957.3 MB/s (`7.66 Gbps`)** | **957.3 MB/s (`7.66 Gbps`)** | 3.15 / 8.49 cores | **8.53** | page-cached |
| | **3. 3-Part Opt (`igzip` SIMD)** | **2.29 s** | **`3.03x`** | **1.01 s** | **482.9 MB/s (`3.86 Gbps`)** | **923.1 MB/s (`7.38 Gbps`)** | 2.85 / 8.04 cores | 8.89 | page-cached |
| **`3.45 GB`** *(Real workload: `3.45 GB` / `1.19 GB`, 9 layers: `807/349/... MB`)* | **1. Baseline (`main`)** | **26.80 s** | `1.00x` | 25.22 s | 48.8 MB/s (`0.39 Gbps`) | 80.0 MB/s (`0.64 Gbps`) | 2.30 / 3.56 cores | 62.35 | page-cached |
| | **2. 3-Part Opt (`klauspost/gzip`)** | **9.09 s** | **`2.95x`** | **1.01 s** | **1,191.2 MB/s (`9.53 Gbps`)** | **1,198.0 MB/s (`9.58 Gbps`)** | 3.93 / 9.24 cores | 37.18 | page-cached |
| | **3. 3-Part Opt (`igzip` SIMD)** | **9.40 s** | **`2.85x`** | **1.52 s** | **792.4 MB/s (`6.34 Gbps`)** | **1,870.2 MB/s (`14.96 Gbps`)** | 3.51 / 12.62 cores | **35.09** | page-cached |
| **`10 GB`** (`10.0 GiB` / `4.50 GiB`, 5 layers: `4.0/2.5/2.0/1.0/0.5 GiB`) | **1. Baseline (`main`)** | **64.52 s** | `1.00x` | 64.21 s | 77.6 MB/s (`0.62 Gbps`) | 95.1 MB/s (`0.76 Gbps`) | 2.85 / 4.18 cores | 186.05 | 86.4 / 409.0 MB/s (`5.64 GB` flushed) |
| | **2. 3-Part Opt (`klauspost/gzip`)** | **12.58 s** | **`5.13x`** | **4.05 s** | **1,205.1 MB/s (`9.64 Gbps`)** | **1,686.5 MB/s (`13.49 Gbps`)** | 5.51 / 12.98 cores | **73.00** | page-cached (`128 GB` RAM) |
| | **3. 3-Part Opt (`igzip` SIMD)** | **16.90 s** | **`3.82x`** | **4.57 s** | **1,067.1 MB/s (`8.54 Gbps`)** | **2,424.2 MB/s (`19.39 Gbps`)** | 5.04 / 15.69 cores | 88.98 | page-cached (`128 GB` RAM) |
| **`50 GB`** (`50.0 GiB` / `22.5 GiB`, 8 layers: `12/10/8/6/5/4/3/2 GiB`) | **1. Baseline (`main`)** | **292.80 s** (`4m 53s`) | `1.00x` | 292.76 s | 84.9 MB/s (`0.68 Gbps`) | 106.7 MB/s (`0.85 Gbps`) | 3.30 / 4.51 cores | 969.17 | 168.5 / 435.1 MB/s (`49.4 GB` flushed) |
| | **2. 3-Part Opt (`klauspost/gzip`)** | **128.13 s** (`2m 08s`) | **`2.29x`** | 103.21 s *(disk-bound)* | 236.3 MB/s (`1.89 Gbps`) | **2,085.5 MB/s (`16.68 Gbps`)** | 4.58 / **22.57 cores** | **596.11** | **375.3 / 432.7 MB/s** (`48.8 GB` flushed) |
| | **3. 3-Part Opt (`igzip` SIMD)** | **97.47 s** (`1m 37s`) | **`3.00x`** | 84.30 s *(disk-bound)* | 289.1 MB/s (`2.31 Gbps`) | **2,192.9 MB/s (`17.54 Gbps`)** | 6.59 / **20.75 cores** | 652.83 | **356.8 / 440.4 MB/s** (`35.3 GB` flushed) |

#### Key Takeaways & Why Substrate Outperforms `containerd`
1. **Zero Overhead on Tiny Images (`16 KB`, `1 MB`)**: Images below `rangeFetchThreshold` (`16 MiB`) bypass `parallelRangeTransport` and stream via a single HTTP `GET` in `~365–398 ms` with no regression.
2. **Network Download Saturated on `1 GB`–`50 GB` (`0.4–0.6 Gbps` → `7.7–19.4 Gbps`)**: Borrowing `containerd`'s parallel HTTP `Range` chunking into an in-memory sliding window (`parallelRangeTransport`) shrinks the network transfer window from `6.8s → 0.51s` (`1 GB`), `25.2s → 1.01s` (`3.45 GB`), and `64.2s → 4.05s` (`10 GB`), reaching **`2,424.2 MB/s` (`19.39 Gbps`) peak** and achieving a **`3.0x–5.13x` end-to-end speedup**.
3. **Smooth Backpressure at `50 GB` (Disk Hardware Ceiling)**: On `50 GB` (`22.5 GiB` compressed), the pipeline bursts to **`2,192.9 MB/s` (`17.54 Gbps`)** and **`20.8–22.6` CPU cores** for the first 8 seconds until unpacked bytes fill the Linux dirty-page cache (`~20 GB` on 128 GB RAM), then sustains the `300 GB pd-ssd` hardware write ceiling (`~357–375 MB/s` average, `440 MB/s` peak), finishing in **`97.47 s` (`3.00x` faster than `main`'s `292.80 s`)**.
4. **Why `containerd` Parallel Chunks (`4.4b`) Still Takes 22.3s–26.2s on `3.45 GB`**:
   - **Double Disk Write & `fsync` of Compressed Blobs**: `containerd` writes and `fsync`s all `1.29 GB` of compressed chunks into `/var/lib/containerd/io.containerd.content.v1.content/ingest/...` before committing them to the content store (`6–8s` of disk flush on `e2-standard-32`, and `3.45 GB` total disk writes vs Substrate's `1.52 GB` on `c3-standard-4`).
   - **Strictly Sequential Layer Unpacking**: `containerd` unpacks layers of an image strictly sequentially (`Layer 0 → ... → Layer 8`) on a single core (`~13–18s`) because `overlayfs` `Prepare(chainID, parentChainID)` requires the parent snapshot first.
5. **CPU Efficiency (`igzip` SIMD vs Pure-Go)**: On `c3-standard-4` (Intel Sapphire Rapids), the `igzip` fast-path reduces total CPU consumption from **`47.45 core-s` (`main`) / `29.13 core-s` (`klauspost/gzip`) down to `21.89 core-s`** (-25% CPU vs `klauspost/gzip`, -54% CPU vs `main`), and on `50 GB` (`e2-standard-32`) reduces wall-clock pull + unpack time from `128.13 s` (`klauspost/gzip`) to **`97.47 s`**.

## Compose path (ateom: `SetupBundleRootfs`)

Called immediately before `runsc create`/`runsc restore` (gVisor) and before
staging the virtio-fs lower (micro-VM):

1. **`FinalizeLayer`** for each referenced layer — materializes the recorded
   whiteouts as 0:0 char devices (`mknod`) and opaque dirs as
   `trusted.overlay.opaque=y` xattrs. Once per layer node-wide; idempotent
   and safe under concurrent ateom pods (`EEXIST` tolerated, marker written
   last). Paths from `whiteouts.json` are re-validated, so a crafted file
   cannot escape the layer tree.
2. **Mount** an overlay at `<bundle>/rootfs`: `lowerdir` is the layer chain
   reversed into overlayfs's top-first order (duplicate layers — images can
   legitimately list the same diffID twice — are collapsed to the topmost
   occurrence, which overlayfs otherwise rejects with `ELOOP`), `upperdir` /
   `workdir` are the bundle-local dirs, holding this actor's private writes.
   The mount uses the new mount API (`fsopen` + one `fsconfig` `lowerdir+`
   append per layer) rather than `mount(2)`, whose single-page option-string
   cap the digest-derived layer paths would hit at ~34 layers. **Minimum
   supported kernel: Linux 6.5** (`lowerdir+`); every current GKE channel
   ships ≥ 6.6 (Stable: COS 121 LTS).
3. **ExtraDirs** are created through the mount (landing in the upper), again
   under `os.Root` confinement.
4. **Implicit-parent metadata repair.** A layer tar routinely omits entries
   for parent directories that exist only in lower layers; unpack fabricates
   them (root:root 0755) and records them as `implicitDirs` in the layer
   metadata. Because overlayfs takes a merged directory's attributes from
   the **top-most** layer containing it, such a fabricated dir would shadow
   the real metadata a lower layer declared (`/tmp` losing its 1777 sticky
   bit, `/root` opening from 0700 to 0755). At compose time the consumer
   resolves each shadowed dir's true mode/ownership from the top-most
   *non-implicit* layer in this image's chain and applies it through the
   mount — the copy-up lands in the actor's private upper; the shared pool
   is never modified. Residual gaps: directory mtimes and xattrs are not
   repaired, and a dir implicit in *every* layer of the chain keeps the
   fabricated attrs.

A bundle without a spec file is left untouched (compatibility with bundles
prepared by a pre-imagecache atelet). A zero-layer spec composes an empty
rootfs with ExtraDirs and no mount.

Actor semantics are unchanged from the untar era: the upper is wiped by
atelet's `resetActorDirs` between runs, so every run still starts from a
bit-exact, pristine image rootfs — it just costs a mount instead of an
extraction. The micro-VM path is nearly untouched: it bind-mounts the (now
overlay-composed) bundle rootfs into virtiofsd's shared dir and the guest
keeps building its own tmpfs upper, as before.

**Teardown**: `UnmountAllUnder(bundleDir)` lazily detaches every mount below
an actor's bundle directory (via `/proc/self/mountinfo`) before atelet wipes
it — called from the checkpoint cleanup path in ateom-gvisor and
`teardownActor` in ateom-microvm.

## Garbage collection

`gc.go` holds the eviction engine; atelet drives it as a periodic pass
(`--image-cache-gc-period`, default 5m; `0` disables the periodic pass,
but startup orphan recovery still runs at every atelet start). Each tick
measures the cache volume with `statfs` and the pool's own size from the
per-layer `size` files, then computes a byte target: free down to
`--image-cache-low-percent` when volume usage reaches
`--image-cache-high-percent`, and/or down to `--image-cache-max-bytes` —
**capped at the pool's own size**, because this cache is one tenant of a
shared volume and an uncapped target would evict the whole cache trying
to fix disk pressure it didn't cause. `--image-cache-gc-dry-run`
computes and logs every decision while mutating nothing: the
recommended way to soak the policy on a live fleet.

Two caveats on those numbers. The cap is the pool's *total* size, not
the evictable subset (rooted and fresh layers can never be freed), so
under **sustained** foreign pressure the target stays unreachable and
every pass evicts everything unrooted and older than min-age — hit rate
goes to zero until the pressure clears. The "could not reach target"
WARNs are the signal; if this bites in practice, a retention floor
(never evict below N bytes) is the intended extension. And usage is
computed against the volume's raw capacity — kubelet's formula, so
operator intuition transfers — which counts ext4's ~5% reserved blocks
as used: eviction starts about five points below the configured
percentage as `df` reports it.

**One pass** (`Store.EvictUnused(ctx, targetBytes, dryRun)`):

1. **Root set** (`Store.InUse`): scan every bundle's
   `rootfs-overlay.json` under the actors dir (`WithActorsDir`).
   Overlay mounts live in the ateom pods' mount namespaces, so atelet
   cannot see them in its own `/proc/mounts`; the bundle specs are
   written by atelet itself before any ateom is asked to mount and
   removed only after unmount, so they are the authoritative "actively
   mounted" set. A spec roots its image digest, each layer dir it
   names, and its *exact* layer set — the last also roots the
   multi-arch twin record and records of digestless (pre-`imageDigest`)
   specs.
2. **Refcount** layers across *all* image records, and list unrooted
   records older than min-age as eviction candidates, LRU-ordered by
   last use (the record's mtime — refreshed on every cache hit and on
   every completed layer of an in-flight pull).
3. **Evict** candidates until ~targetBytes is freed: delete the record
   (after a freshness re-check under the same lock the cache-hit path
   holds), then retire each layer the removal left unreferenced. If any
   layer must be kept — still referenced by another record, rooted by a
   spec, younger than min-age, or its retirement failed — the record is
   restored byte-exact and the image simply is not evicted this pass: a
   layer is never left on disk without a record explaining it.

The pass reaches layers **only through records**: a layer is deleted
exactly when its last referencing record goes, never by an independent
scan of the pool.

**Everything fails toward retention.** If the image records or the
bundle specs cannot be fully enumerated (an unreadable file or
directory), the pass does nothing and logs at ERROR naming the culprit:
refcounts and roots computed from partial data would retire layers that
unread records still reference or running actors still mount. Dry-run
mutates nothing at all — not even the lazy size-file backfill.

**In-flight pulls need no separate protection.** `EnsureImage` writes
the image record **before** unpacking (the way Go allocates black during
GC and containerd creates its ingest record before the bytes land), so
every layer a pull produces is referenced — and kept fresh by a
per-layer progress touch — from before it exists on disk. An interrupted
pull's record is resumable progress, not garbage: the next pull of that
digest re-fetches only the missing layers, and a pull that never resumes
ages out through ordinary LRU.

**Startup recovery.** A layer no record references can only be crash
debris (eviction interrupted between record-delete and layer-rename) or
operator damage; `New` reclaims such orphans once, at startup
(`Store.RecoverOrphans`), when no pull can be racing the scan — and
skips the scan entirely, conservatively, if any record or bundle spec
fails to read. There is no online whole-pool scan (ext4's split: bounded
recovery at mount, fsck offline).

**Deletion is two-phase.** A layer is atomically renamed to `.rm-*`
inside the layer's singleflight (one `rename(2)` — eviction can never
stall a pull), then removed asynchronously; a crash in between leaves
the dir for the startup sweep. This matters because the kernel offers no
protection here: deleting a directory that is a live overlay lowerdir in
another mount namespace succeeds silently, leaves the overlay's behavior
undefined, and doesn't even free the space until the mount goes away.

Deleting the cache root by hand (while no actors are starting) remains
safe — the store re-pulls whatever is missing.

This is Phase 2 of [#463](https://github.com/agent-substrate/substrate/issues/463);
the watermark loop, flags, and cache metrics complete it. Phase 3 adds
the control-plane surface (reporting cached digests for scheduling
affinity, and a `PreloadImage` API with expiring pins). The
layer-materializer seam is also designed so a lazy-pull backend
(eStargz/SOCI-style FUSE) can replace the untar backend later without
restructuring.

## Testing

- Portable unit tests (run everywhere, including macOS): the unpack security
  suite (traversal, symlink/hardlink escape, whiteout capture, later-entry-
  wins, read-only dirs, missing parents), spec round-trips, overlay option
  assembly (including duplicate-layer dedup), mountinfo parsing, ref
  rewriting, options. End-to-end pull tests run against an in-memory
  registry (`pkg/registry`).
- Linux-tagged tests (`bundle_linux_test.go`): unprivileged ones cover
  escape rejection and specless/zero-layer compose; root-gated ones execute
  the real `mknod`/xattr materialization and a full mount → write-isolation
  → unmount round trip (the write-isolation assertion — actor writes land in
  the bundle upper, never in the shared pool — is the key safety property).
  The root-gated ones self-skip via `roottest.Require`; CI (and
  `hack/run-root-tests.sh` locally) reruns the package under `sudo` so they
  execute.
- `tools/validate-image-cache` batch-validates that arbitrary registry
  images can be pulled, parsed, and unpacked by the store half — useful for
  sweeping large image corpora before relying on them in production.
- `internal/imagecache/bench/` (`gen_scale_images.go`, `run_scale_bench.sh`,
  and `node_sampler.sh`) generates synthetic multi-scale `.tar.gz` OCI images
  (`16 KB`, `1 MB`, `1 GB`, `10 GB`, `50 GB`) and runs the live-cluster
  cold-pull + unpack benchmark suite documented in
  [`benchmark_comparison_and_design.md`](benchmark_comparison_and_design.md).
