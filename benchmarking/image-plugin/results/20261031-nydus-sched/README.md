# Nydus seeds variant: scheduler check and larger reads

Seeds variant (seed preload, `schedulingPolicy: always`), old images,
8 actors per worker, cold first turn. Seconds, p50/p95/p100.

| Run | TTFI | E2E | wakeup_probe p50 | dfdaemon tasks |
|---|---|---|---|---|
| 1 actor | 4.1 | 6.0 | 2.3 | 235 |
| 64 actors | 14.1/23.3/25.6 | 17.2/28.5/31.0 | 11.2 | 3440 |
| 64, nydusd reads 4 MiB | 14.1/22.3/24.0 | 18.6/27.5/30.7 | 11.6 | 2213 |
| 64, nydusd reads 16 MiB | 13.4/23.0/25.0 | 19.6/28.7/30.5 | 10.7 | 1666 |

The scheduler stays idle during the 64-actor run: peak 63m CPU, 2.5 CPU
seconds, about 600 gRPC streams (`sched-samples.log`).

Larger nydusd reads (`../20261031-nydus-amp`, patched nydusd from
`nydusd-amp/`) halve the dfdaemon tasks but leave latency unchanged. The
fscache data per run stays at about 920 MB.

One actor alone starts in 4.1 s. 21 at once on a node take 14 s. The
slowdown comes from concurrency on the node, not from the scheduler or
the request count.
