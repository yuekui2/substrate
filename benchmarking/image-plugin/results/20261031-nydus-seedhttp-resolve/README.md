# Nydus seedhttp with image resolve, manifests and bootstraps from the seeds

Old images, cold bench pool, 8 actors per worker, first turn, seconds,
p50/p95/p100.

| Run | TTFI | E2E |
|---|---|---|
| 1 actor | 1.2 | 2.9 |
| 64 actors | 3.0/4.9/5.2 | 7.8/10.8/10.9 |

`../20261031-nydus-seedhttp-boot2` (manifests from the seeds, registry
resolve): 1 actor TTFI 1.3, E2E 3.0.

Placement at 64 was uneven, 26/10/28 actors per node. TTFI per node:
p50 3.1 / max 4.7 at 26, 1.6 / 1.9 at 10, 3.6 / 5.2 at 28. In boot2 it
was 30/30/4, with p50 5.1 and 4.1 on the full nodes and 1.0 on the other.

Node CPU stays low (busy p50 1%, max 29% of 22 cores). At 8 actors per
1-vCPU worker each actor gets 125 millicores, so TTFI grows with the
actors on a worker, not with node load.
