# Image plugin optimizations: Nydus + Dragonfly and Spegel

Setup unless noted: 3 × c3-standard-22 bench nodes, gVisor workers of 1 vCPU,
SWE-bench images with one distinct image per actor, cold nodes per point.
Numbers are first-turn TTFI unless marked E2E (resume request to work done).
Older rows come from `wrap-up.md` and used p100 or 8 actors per worker. The
metric is given in each row.

## Nydus + Dragonfly

### Kept, in the order found

| # | Change | Why it helps | Effect |
|---|---|---|---|
| 1 | Seed HTTP: 2 seed nodes serve blobs, bootstraps, manifests and image resolution over plain HTTP (nginx), replacing Dragonfly P2P | Drops Dragonfly scheduling and peer lookups from every on-demand read | p100 at 16–20 per node: 24.8 → 3.7 s |
| 2 | 2 actors per worker instead of 8 | 500 millicores per actor instead of 125 | E2E at 32 per node: 16.3 → 13.2 s |
| 3 | One fscache domain for all images (`domain_id: ate-shared`) | Shared blobs and bootstraps are cached once per node | p100 at 32 per node: 10.7 → 7.2 s; at 16: 4.2 → 3.0 s |
| 4 | atelet starts image prep before the snapshot manifest fetch | The mount overlaps the fetch | p100 at 32 per node: 7.2 → 6.9 s |
| 5 | nydusd bootstrap fill patch (stock v2.3.0 never copies the bootstrap into the cache) | Metadata reads stop going to the seeds: 99% of wake-up reads before | p100 at 32 per node: 6.9 → 5.5–5.9 s |
| 6 | Layered mount (shared parent layers mounted once per node) | Fewer mounts when images share a base | Shared-base images: 3.4 → 2.0 s p100. No gain on distinct images |
| 7 | **Removed the plugin's metadata walk** (`warmLayerMetadata`, Nov 4) | It `lstat`ed every file of every new mount during guest wake-up: 91% of plugin CPU | 32 per node, p95: 4.7–5.0 → 3.4–3.6 s. E2E p95: 7.1 → 6.1–6.4 s. Plugin CPU: 3–5 → <1 core per node. Node busy CPU peak: 81% → 55–60% |
| 8 | **Balanced workers across nodes** (`wait-ready.sh`, Nov 4) | Worker pods piled onto nodes that came up first, e.g. 21/6/21 | Removes the 31–34-actor node at 24 per node |
| 9 | **Node-aware actor placement + cache feed after each bind** (ateapi, Nov 4) | Placement ignored nodes, and concurrent first resumes saw stale load | Busiest node at 16 per node: 21 → 16–20 |
| 10 | **Uncompressed blobs** (Nov 4) | nydusd skips decompression | See below |

### Uncompressed vs zstd at 32 per node (walk removed, Nov 4)

| Set | zstd TTFI p95 | Uncompressed TTFI p95 | zstd E2E p95 | Uncompressed E2E p95 |
|---|---|---|---|---|
| A | 3.38 / 3.59 / 3.44 | 3.37 / 3.25 | 6.1–6.4 | 5.6–5.8 |
| B | 3.66 / 3.43 | 2.98 / 3.08 | 6.9 / 6.9 | 5.7 / 5.6 |
| C | 3.27 / 3.27 | 3.12 / 2.80 | 6.4 / 6.2 | 6.0 / 5.7 |
| Mean | 3.43 | 3.10 | 6.5 | 5.7 |

Uncompressed wins on every set: about 0.3 s TTFI p95 and 0.8 s E2E p95.
nydusd drops from 1.5–1.9 to about 1.3 cores per node. Images grow about
2.7× (1.0 → 2.7 GB on average).

### Current best (walk removed, balanced, node-aware placement)

| Per node | TTFI p95 | E2E p95 | Runs |
|---|---|---|---|
| 16 | 1.56–1.87 (mean 1.70) | 4.3–5.6 | 6, sets A/B/C, zstd |
| 24 | 2.29–2.70 (mean 2.51) | 5.2–5.7 | 6, sets A/B/C, zstd |
| 32 | 2.80–3.37 (mean 3.10) | 5.6–6.0 | 6, sets A/B/C, uncompressed |

None of this uses the workload-specific prefetch list. Before these changes,
the best setup used prefetch-ordered images and got 1.8 / 4.5 / 4.9 s p95 at
16 / 24 / 32 per node.

### Tried and dropped

| Change | Result |
|---|---|
| nydusd prefetch (`NYDUS_PREFETCH`) | 91 s p100: it fetches whole blobs |
| Prefetch-ordered images (prefetch table) | About 0.2–0.5 s at 32 per node under the old setup. Dropped as workload-dependent; the fixes above beat it |
| Warm lists (heuristic, and memory-mapped files) | 6.1–7.0 vs 5.9 s: whole-file reads compete with restore |
| Dragonfly P2P, Dragonfly seed peers | 25–50 s p100 at 20–40 per node; failed turns at 160 per node |
| tmpfs cache | All turns failed |
| EROFS read-ahead 2 MB, larger fscache reads | No gain (10.8 vs 10.7 s) |
| Access-ordered images, local SSD, serving the extracted bootstrap, per-layer mounts on distinct images | No gain |
| No CPU limit on workers (Burstable, Nov 4) | No gain (3.45 vs 3.38 s p95). Workers use about 2 of their 16 cores at peak and barely throttle |

## Spegel

Spegel shares full OCI layers between nodes. Each node still downloads and
unpacks every layer before an actor on that image can start, so it does not
lazy-load.

| # | Change | Effect |
|---|---|---|
| 1 | Plugin fixes: cache the peer list, fetch each blob upstream once per cluster, harden owner fetches, count bytes by source | Made Spegel P2P work at scale. Still p50 37–62 s at 20–40 per node |
| 2 | Seed preload: 2 seed nodes as the only blob owners, preloaded before the run (`spegel-preload`, `--spegel-owner-dns`) | 0 upstream bytes and 0 peer misses. p100 56.0 / 63.2 / 72.9 s at 16 / 24 / 32 per node (8 per worker) |
| 3 | 2 actors per worker, workers unbalanced (Nov 3 confirm sweep) | p95 40.9 / 52.6 / 62.0 s at 16 / 24 / 32 per node (mean of sets A/B/C) |

Why it stays slow: each node downloads 5–20 GB of compressed layers and
unpacks them, about 250 MB/s per node. `oci_unpack` (download plus unpack)
is the whole TTFI. Restore stays under 0.7 s. The seeds serve about 800 MB/s
in total, below their NIC limits, so the registry and seeds aren't the
bottleneck. Recommendation: only for pre-warmed nodes or small images.

## Riptide + Memorystore, for reference (set A, Nov 4)

| Per node | With walk (fpt build) TTFI p95 / E2E p95 | Walk removed TTFI p95 / E2E p95 |
|---|---|---|
| 16 | 3.26 / 9.2 | 3.04 / 6.7 |
| 24 | 5.51 / 13.2 | 4.35 / 8.4 |
| 32 | 10.35 / 16.2 | 5.66 / 9.9 |

The walk also hurt Riptide: each `lstat` was a FUSE round trip to gcfsd.
gcfsd CPU drops from 6.6–8.4 to 4.6–6.3 cores per node.

Run directories: `20261104-nydus-cpu`, `20261104-nydus-repeats`,
`20261104-nydus-n32-none`, `20261104-riptide-fpt`, `20261104-riptide-nowalk`,
`20261031-spegel-prefetch-sweep`, `20261103-confirm-sweep`, `wrap-up.md`.
