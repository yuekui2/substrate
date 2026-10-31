# Nydus seedhttp sweep at 16, 24 and 32 actors per node

Same variant as `../20261031-nydus-seedhttp-sweep`. Cold bench pool per
point, one image per actor, 8 actors per worker, first turn, seconds,
p50/p95/p100. `n32-authfail` died on expired credentials and was rerun.

| Actors per node | TTFI | E2E | Placement | Busiest node TTFI max |
|---|---|---|---|---|
| 16 | 2.2/3.3/3.7 | 6.8/9.6/9.8 | 12/17/19 | 3.7 (19 actors) |
| 24 | 4.2/6.8/7.1 | 9.1/13.2/13.3 | 9/29/34 | 7.1 (34 actors) |
| 32 | 5.6/9.2/9.9 | 10.1/15.7/16.3 | 13/38/45 | 9.9 (45 actors) |

First restore per actor, p50/p95:

| Actors per node | worker_restore | oci_unpack | manifest_fetch |
|---|---|---|---|
| 16 | 1.14/2.05 | 0.52/0.67 | 0.47/0.53 |
| 24 | 2.51/4.60 | 1.24/1.75 | 0.29/0.35 |
| 32 | 3.70/7.12 | 1.27/1.58 | 0.41/0.47 |

At 8 actors per worker the scheduler packs actors into few workers, so
placement is skewed and each actor gets 125 millicores. worker_restore
grows with the actors on a node. `driver/pointstat.py` makes these tables.
