# Nydus and Dragonfly recipes for Agent Substrate (draft, Nov 5 2026)

Four recipes, from least to most setup. Each one builds on the previous.
The numbers come from existing run directories under this directory. Rows
marked "not comparable" used an older setup. See "Gaps" at the end.

## Common setup

- 3 × c3-standard-22 bench nodes, Hyperdisk Balanced, gVisor workers of
  1 vCPU. Cold bench pool per point.
- One distinct SWE-bench image per actor. 5 turns of 1,000 file reads.
- Nydus RAFS v6 images on kernel EROFS + fscache
  (`image-plugin/manifests/nydus-dragonfly*.yaml`).
- Metric: first-turn TTFI p50 / p95 / p99 / p100 in seconds, including
  queue wait. E2E is resume request to work done.
- Target: TTFI p95 < 3 s at 32 actors per node.
- Baseline without any plugin (pull) on the same setup as recipe 4: TTFI
  p95 75.8 / 96.2 / 126.0 s at 16 / 24 / 32 per node
  (`20261103-confirm-sweep/summary.md`).

## Summary

| Recipe | Data path | Custom patches | Best TTFI at about 20 per node | TTFI at 32 per node |
|---|---|---|---|---|
| 1. No preload or prefetch | nydusd → dfdaemon P2P → registry | None | p100 24.8 (not comparable) | Not measured |
| 2. Preload, Dragonfly P2P | nydusd → dfdaemon P2P → preloaded seed peers | None | p100 25.9 (not comparable) | Not measured |
| 3. Preload, HTTP seeds, stock | nydusd → nginx on 2 seed nodes | None | p100 3.0 at 16 | p95 6.7, p100 7.2 |
| 4. Preload, HTTP seeds, patched | Same as 3 | nydusd + substrate fixes | p95 1.70 at 16 | p95 3.10 (mean of 6 runs) |

## Recipe 1: no image preload or prefetch

Setup:
- `image-plugin/manifests/nydus-dragonfly.yaml`: Dragonfly v2.5 manager,
  scheduler, and dfdaemon on every node as a registry mirror for nydusd.
- Blobs are fetched lazily from the registry on first read. Peers share
  pieces through Dragonfly. Nothing is preloaded.
- nydusd prefetch off. 8 actors per worker.

Numbers (old 64-image set, 8 actors per worker,
`20261030-node-sweep-nydus/n*`):

| Actors per node | TTFI p50 / p95 / p99 / p100 | E2E p50 / p95 / p100 |
|---|---|---|
| about 3 (n10 total) | 10.0 / 19.4 / 21.4 / 21.5 | 13.7 / 22.9 / 27.3 |
| about 7 (n20 total) | 14.2 / 24.4 / 24.7 / 24.8 | 19.6 / 27.9 / 29.7 |
| about 13 (n40 total) | 23.3 / 43.2 / 49.3 / 50.7 | 27.5 / 46.6 / 57.7 |
| 160 (`20261030-sweep`) | 546.5 / 636.4 / 640.7 / 640.9 | 834 of 2,400 turns failed |

Why it is slow: every first read of a blob goes through Dragonfly task
setup. A first 4 KiB read in a blob takes 440–525 ms through dfdaemon
(`20261031-nydus-seedhttp/README.md`). With nydusd prefetch on, nydusd
fetches whole blobs and p100 reaches 91 s.

## Recipe 2: preload with Dragonfly P2P

Setup, on top of recipe 1:
- `image-plugin/manifests/nydus-dragonfly-seeds.yaml`: 2 nodes labeled
  `ate.dev/dragonfly-seed=true` run dfdaemon as super seed peers.
- The seeds are preloaded with every image's blobs before the run
  (`seed-preload.sh`). `schedulingPolicy: always`.
- Bench nodes fetch pieces from the seeds instead of the registry.

Numbers (8 actors per worker):

| Run | Load | TTFI p50 / p95 / p99 / p100 | E2E p50 / p95 / p100 |
|---|---|---|---|
| `20261031-nydus-sched`, 1 actor | 1 | 4.1 | 6.0 |
| `20261031-nydus-seeds/seeds` | 64 actors, about 21 per node | 14.0 / 23.1 / 25.4 / 25.9 | 18.9 / 27.8 / 29.4 |
| `20261031-nydus-seeds/seeds-warm` (plus warm lists) | same | 43.5 / 48.0 / 49.4 / 49.7 | 51.4 / 55.8 / 56.2 |
| `20261030-nydus-seeds2` | 160 per node | 59.5 / 127.1 / 138.6 / 145.7 | 0 failed turns |
| `20261030-nydus-seeds2-dfnopf` (Dragonfly prefetch off) | 160 per node | 65.1 / 124.4 / 133.9 / 142.1 | 0 failed turns |

Key optimization: the seed peers remove registry round trips.

Improvement over recipe 1:
- At 160 per node: p100 641 → 146 s, and failed turns 834 → 0.
- At about 20 per node: no gain at p100 (24.8 vs 25.9 s).

Why it stays slow: the Dragonfly scheduler is idle (peak 63m CPU). The
cost is per-piece task setup in dfdaemon under node concurrency. One actor
starts in 4.1 s, 21 at once on a node take 14 s. Larger nydusd reads halve
the dfdaemon tasks but don't change latency (`20261031-nydus-sched`).

## Recipe 3: preload with HTTP seeds, no custom patches

Setup, replacing Dragonfly in the data path:
- `image-plugin/manifests/nydus-dragonfly-seedhttp.yaml`: nginx
  (`nydus-blob-seed`) on the 2 seed nodes serves blobs, bootstraps,
  manifests and image resolution over plain HTTP. nydusd tries the seeds
  first and falls back to dfdaemon.
- Preload: `SAVE_DIR=/var/lib/ate/nydus-seed-blobs seed-preload.sh`.
- Stock nydusd v2.3.0. nydusd prefetch off.
- Config only, no code changes:
  - 2 actors per worker instead of 8.
  - One fscache domain for all images (`"domain_id": "ate-shared"`, now the
    manifest default).

Numbers, set A, prefetch-ordered images (`images-96-pf.jsonl`):

| Step | Load | TTFI p50 / p95 / p99 / p100 | E2E p50 / p95 / p100 | Run |
|---|---|---|---|---|
| Seed HTTP, 8 per worker | 64 actors | 3.4 / 5.0 / – / 5.4 | 9.1 / 11.4 / 11.9 | `20261031-nydus-seedhttp` |
| Seed HTTP, 8 per worker | 16 per node | 2.2 / 3.3 / – / 3.7 | 6.8 / 9.6 / 9.8 | `20261031-nydus-seedhttp-sweep2` |
| Seed HTTP, 8 per worker | 32 per node | 5.6 / 9.2 / – / 9.9 | 10.1 / 15.7 / 16.3 | same |
| + 2 per worker | 16 per node | 2.1 / 3.5 / – / 4.2 | 4.7 / 5.8 / 6.0 | `20261031-nydus-seedhttp-sweep-w2` |
| + 2 per worker | 32 per node | 6.7 / 10.2 / – / 10.7 | 8.9 / 12.8 / 13.2 | same |
| + one fscache domain | 16 per node | 2.0 / 2.8 / 2.9 / 3.0 | 4.6 / 5.9 / 6.0 | `20261101-nydus-seedhttp-distinct-domain/n16` |
| + one fscache domain | 32 per node | 3.1 / 6.7 / 7.0 / 7.2 | 7.4 / 8.8 / 9.6 | `20261101-nydus-seedhttp-distinct-domain/n32` |

Key optimizations:
1. Seed HTTP. A first 4 KiB read drops from 440–525 ms to 1–2 ms. One
   actor's `wakeup_probe` drops from 2.3 to 0.15 s.
2. 2 actors per worker. Each actor gets 500 millicores instead of 125.
   This mostly shortens E2E.
3. One fscache domain. Shared blobs and bootstraps are cached once per
   node.

Improvement over recipe 2: at about 20 per node, p100 25.9 → 3.7 s.
At 32 per node, p100 10.7 → 7.2 s from the shared domain alone.

Images that share a base layer do better. With `--layered-mount`, shared-base
images reach 1.8 / 1.9 / 1.9 / 2.0 at 32 per node
(`20261101-nydus-seedhttp-shared-layered/n32`). That run also had the atelet
overlap change from recipe 4.

## Recipe 4: preload with HTTP seeds and custom patches

Setup, on top of recipe 3 (`deploy-nydus-best.sh`):
- nydusd with `nydusd-amp/bootstrap-fill.patch`
  (`PATCHES=bootstrap-fill.patch nydusd-amp/build.sh`). Stock v2.3.0 passes
  an async block to `spawn_blocking`, so it never copies the bootstrap into
  the cache file. The patch makes the copy run, at explicit offsets.
- Substrate changes made during the benchmark:
  - atelet starts image prep before the snapshot manifest fetch.
  - The plugin no longer walks every file of a new mount
    (`warmLayerMetadata` removed).
  - `wait-ready.sh` balances workers across nodes, 16/16/16.
  - ateapi places actors by node load and feeds each bind to its cache.
- Uncompressed Nydus blobs at 32 per node. Images grow from 1.0 to 2.7 GB
  on average.
- Original images, no prefetch table (`images-96*.jsonl`).

Numbers, all 3 image sets:

| Load | TTFI p95 | TTFI p100 | E2E p95 | Runs |
|---|---|---|---|---|
| 16 per node | 1.56–1.87 (mean 1.70) | 1.74–2.13 | 4.3–5.6 | 6, zstd (`20261104-nydus-repeats`) |
| 24 per node | 2.29–2.70 (mean 2.51) | 2.53–3.03 | 5.2–5.7 | 6, zstd (`20261104-nydus-repeats`) |
| 32 per node | 2.80–3.37 (mean 3.10) | 3.42–3.81 | 5.6–6.0 | 6, uncompressed (`20261104-nydus-cpu`, `optimizations-summary.md`) |

Effect of each change, in the order applied:

| Change | Load | Before → after |
|---|---|---|
| atelet overlap | 32 per node | p100 7.2 → 6.9 |
| Bootstrap fill | 32 per node | p100 6.9 → 5.5–5.9. Bootstrap reads 236K → 1K |
| Bootstrap fill, 3 sets (`20261103-confirm-sweep`) | 16 / 24 / 32 | p95 1.8 / 4.5 / 4.9 |
| Walk removed | 32 per node | p95 4.7–5.0 → 3.4–3.6. Plugin CPU 3–5 → under 1 core per node |
| Walk removed, balanced workers, node-aware placement | 24 per node | p95 4.5 → 2.51 (mean). Busiest node 31–34 → 24–26 actors |
| Uncompressed blobs | 32 per node | p95 3.43 → 3.10 (mean). E2E p95 6.5 → 5.7 |

Improvement over recipe 3 at 32 per node: p95 6.7 → 3.10 s, p100
7.2 → 3.4–3.8 s, E2E p95 8.8 → 5.6–6.0 s.

## Gaps

1. Recipes 1 and 2 never ran on the current setup: 96-image sets A/B/C,
   2 actors per worker, 16 / 24 / 32 per node, balanced placement. Their
   rows use the old 64-image set at 8 per worker.
2. Recipes 1 and 2 were never run at the same load in one sweep. The only
   overlap is about 20 per node, from different days.
3. Recipe 3 has one run per point, set A only, and no 24 per node point
   with the shared domain.
4. Recipe 3 used prefetch-ordered images, and recipe 4 didn't. Their
   numbers mix image layout with the patches.
5. Recipe 4 mixes the nydusd patch with substrate fixes. There is no run
   with the substrate fixes on stock nydusd, so the patch's own gain under
   the final setup is unknown.
6. The run configs for `20261030-node-sweep-nydus` and
   `20261030-nydus-seeds2` should be checked against their transcripts. The
   doc assumes the first had no seeds and the second had preloaded seeds.
