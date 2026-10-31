# Nydus with seeds serving blobs over HTTP

`nydus-dragonfly-seedhttp.yaml`: nginx on each seed serves the 202 blobs of
the 64 old images from disk. nydusd tries the seeds first and falls back to
dfdaemon. Bench pool recreated cold. 8 actors per worker, first turn,
seconds, p50/p95/p100.

| Run | TTFI | E2E | Seeds through Dragonfly (`../20261031-nydus-sched`) |
|---|---|---|---|
| 1 actor | 2.1 | 3.8 | TTFI 4.1, E2E 6.0 |
| 64 actors | 3.4/5.0/5.4 | 9.1/11.4/11.9 | TTFI 14.1/23.3/25.6, E2E 17.2/28.5/31.0 |

wakeup_probe for 1 actor drops from 2.3 s to 0.15 s.

Range reads from a bench node (`driver/seedprobe.sh`, in `driver/seedhttp2.out`):

| Read | Seed HTTP | dfdaemon |
|---|---|---|
| 4 KiB, first in a blob | 1-2 ms | 440-525 ms |
| 4 KiB, new piece | about 1 ms | 47-95 ms |
| 1 MiB | 4-9 ms | about 1 ms once the piece is local |

`../20261031-nydus-seedhttp-nginxdown` is the first attempt, where nginx
failed to start on a config error and every read fell back to dfdaemon.
