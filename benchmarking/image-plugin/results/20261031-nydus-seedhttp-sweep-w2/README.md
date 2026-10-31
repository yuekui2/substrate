# Nydus seedhttp sweep at 2 actors per worker

Same as `../20261031-nydus-seedhttp-sweep2` but 2 actors per worker (500
millicores each). First turn, seconds, p50/p95/p100.

| Actors per node | TTFI | E2E | Placement | Busiest node TTFI max |
|---|---|---|---|---|
| 16 | 2.1/3.5/4.2 | 4.7/5.8/6.0 | 13/16/19 | 4.2 (19 actors) |
| 24 | 3.1/8.7/9.1 | 6.6/10.8/11.5 | 16/21/35 | 9.1 (35 actors) |
| 32 | 6.7/10.2/10.7 | 8.9/12.8/13.2 | 12/42/42 | 10.7 (42 actors) |

More CPU per actor shortens E2E but not TTFI. Placement stays skewed.
TTFI max per node: 2.4 at 13 actors, 2.8 at 16, 3.1 at 16, 4.7 at 21.
worker_restore p95 grows from 2.5 s at 16 per node to 8.4 s at 32.
