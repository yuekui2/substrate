# Nydus repeats at 16 and 24 per node, sets A, B and C, Nov 4 2026

Setup as in `../20261104-nydus-cpu/README.md`, with the metadata walk
removed from the plugin. ateapi `ateapi@sha256:8b1657ba…`
(`setup/ateapi-image.txt`) places by node load and feeds each bind's Worker
to its cache (`Cache.Observe`), so resumes that start together see earlier
binds. Original zstd images: set A `images-96.jsonl`, B `images-96-b.jsonl`,
C `images-96-c.jsonl` (lines 1–96, 97–192, 193–288 of `images-320.jsonl`).
Cold pool per point, 2 actors per worker, workers 16/16/16. Driver:
`driver/all.sh`, `driver/queue.sh`.

First turn, seconds, TTFI from `rep-1/*/bottleneck.md`, E2E (resume request
to work done) from `sweep.md`:

| Run | TTFI p50 / p95 / p99 / p100 | E2E p95 | Actors per node |
|---|---|---|---|
| a-n16-r1 | 1.63 / 1.87 / 2.06 / 2.13 | 5.6 | 16 / 20 / 12 |
| a-n16-r2 | 1.37 / 1.56 / 1.69 / 1.74 | 4.3 | 17 / 16 / 15 |
| b-n16-r1 | 1.37 / 1.69 / 1.75 / 1.75 | 4.5 | 17 / 14 / 17 |
| b-n16-r2 | 1.27 / 1.60 / 1.78 / 1.81 | 4.4 | 16 / 18 / 14 |
| c-n16-r1 | 1.39 / 1.80 / 1.84 / 1.86 | 4.3 | 15 / 18 / 15 |
| c-n16-r2 | 1.41 / 1.67 / 1.77 / 1.82 | 4.5 | 16 / 16 / 16 |
| a-n24-r1 | 1.85 / 2.50 / 2.62 / 2.70 | 5.4 | 25 / 23 / 24 |
| a-n24-r2 | 1.89 / 2.70 / 2.88 / 2.92 | 5.5 | 23 / 26 / 23 |
| b-n24-r1 | 1.92 / 2.45 / 2.60 / 2.65 | 5.7 | 24 / 24 / 24 |
| b-n24-r2 | 1.80 / 2.40 / 2.54 / 2.58 | 5.2 | 24 / 24 / 24 |
| c-n24-r1 | 1.78 / 2.70 / 2.96 / 3.03 | 5.4 | 22 / 24 / 26 |
| c-n24-r2 | 1.81 / 2.29 / 2.44 / 2.53 | 5.2 | 24 / 25 / 23 |

TTFI p95 is 1.56–1.87 s at 16 per node (mean 1.70) and 2.29–2.70 s at 24
(mean 2.51). Every run is under 3 s at p95, and all but one at p100.

Placement: the busiest node held 16–20 actors at 16 per node and 24–26 at
24. Before the cache feed, both arms got 13 / 14 / 21 at 16 per node
(`../20261104-nydus-cpu/e5-nowalk-n16`, `../20261104-riptide-fpt/n16`).
Some skew remains: picks made at the same moment, and binds by the other
ateapi replica, are still unseen until the watch delivers them.
`placement-1` and `-2` are warm-pool checks of 48 actors, one turn.
