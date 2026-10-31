# Nydus seedhttp node sweep

Best Nydus variant: seeds serve blobs, bootstraps, manifests and image
resolve over HTTP. Cold bench pool per point, one image per actor, 8 actors
per worker, first turn, seconds, p50/p95/p100. Per-point tables, including
CPU, are in `n*/sweep.md`.

| Actors per node | TTFI | E2E | Placement (actors per node) |
|---|---|---|---|
| 1 | 1.2/1.2/1.2 | 3.5/6.9/6.9 | 1/2/0 |
| 40 | 8.0/13.5/14.8 | 13.2/18.9/20.8 | 14/46/60 |
| 64 | 9.3/20.4/26.1 | 16.5/23.9/32.0 | 56/63/73 |

TTFI per node follows the actors on that node: at 40, p50/max is 2.1/2.7
with 14 actors, 7.4/9.9 with 46 and 10.9/14.8 with 60. Node CPU stays low
(busy max 5% at 40, 18% at 64). The slow phases at load are worker_restore
(p95 8-11 s, the app's first reads and its CPU share of 125 millicores) and
oci_unpack (p95 2-3 s).
