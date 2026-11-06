# Nydus CPU and wake-up, Nov 4 2026

Setup: 3 × c3-standard-22 bench nodes, gVisor workers of 1 vCPU, 2 actors
per worker, cold pool per point, set A (first 96 lines of
`images-320.jsonl`), original zstd Nydus images without a prefetch table
unless noted. nydusd with `bootstrap-fill.patch`, seed HTTP, one fscache
domain (`deploy-nydus-best.sh`). Workers are balanced 16/16/16 by
`wait-ready.sh`, and ateapi places actors with node load
(`cmd/ateapi/internal/scheduling`). Driver: `driver/cpu.sh`,
`driver/queue*.sh`.

First-turn TTFI p50 / p95 / p99 / p100 (s), from each run's
`rep-1/*/bottleneck.md`:

| Run | Per node | Change | TTFI | Plugin cores per node |
|---|---|---|---|---|
| e1-base-n32 | 32 | baseline, metadata walk on | 2.31 / 5.04 / 5.76 / 9.33 | 2.9–3.3 |
| e1-base-n32-r2 | 32 | baseline, metadata walk on | 2.37 / 4.73 / 5.23 / 5.73 | 4.7–5.2 |
| e2-nowalk-n32 | 32 | walk off | 2.23 / 3.38 / 3.89 / 4.13 | 0.8–1.0 |
| e2-nowalk-n32-r2 | 32 | walk off | 2.21 / 3.59 / 4.15 / 4.35 | |
| e2-nowalk-n32-r3 | 32 | walk off | 2.25 / 3.44 / 3.79 / 4.01 | |
| e3-nolimit-n32 | 32 | walk off, no worker CPU limit | 2.31 / 3.45 / 4.13 / 4.17 | |
| e4-none-n32 | 32 | walk off, uncompressed blobs | 2.32 / 3.37 / 3.75 / 3.81 | |
| e4-none-n32-r2 | 32 | walk off, uncompressed blobs | 2.24 / 3.25 / 3.38 / 3.42 | |
| e5-nowalk-n24 | 24 | walk off | 1.85 / 2.52 / 2.69 / 2.70 | |
| e5-nowalk-n16 | 16 | walk off (actors placed 13/14/21) | 1.40 / 2.11 / 2.22 / 2.29 | |

Findings:
- The plugin's CPU was `warmLayerMetadata`: 91% of its profile
  (`e1-base-n32/pprof`), mostly `lstat` and `readdir` over every file of
  each new mount. Without it, p95 at 32 per node drops from 4.7–5.0 s to
  3.4–3.6 s, node busy CPU peaks at 54–60% instead of 81%, and the slowest
  `wakeup_probe` falls from 7.6 s to 2.0 s. The walk is now deleted.
- The 1-CPU worker limit is not binding: at peak the 16 workers of a node
  use 1.7–2.1 cores and are throttled 0.07–0.47 s per second in total
  (`guaranteed` in `diskstats.jsonl`). Dropping the limit (e3) changes
  nothing.
- Uncompressed blobs cut nydusd from about 1.8 to 1.1–1.3 cores per node
  and TTFI p95 by about 0.1–0.2 s (2 reps each). The app's file reads gain
  more: first-task E2E p95 is 5.6–5.8 s against 6.1–6.4 s with zstd. Blobs
  grow from 1.0 to 2.7 GB per image.
- First-task E2E p95 (resume request to work done, `sweep.md`): walk on
  7.1–7.2 s, walk off 6.1–6.4 s, uncompressed 5.6–5.8 s at 32 per node;
  5.3 s at 24 and 4.6 s at 16 with the walk off.
- At 24 per node, ateapi placed 24/24/24. At 16 per node it placed
  13/14/21 on both arms, so concurrent first placements still skew. The
  scheduler only sees committed allocations.
