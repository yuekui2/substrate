# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-n32-none/driver/../c-zstd-n32-r2/rep-1/nydus-oc-w2

96 first turns. TTFI p50 / p95 / p99 / p100: 2.28 / 3.27 / 3.48 / 3.95 s. Window 3.9 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 197x | 32 | 2.11 | 3.95 |
| 9tc5 | 32 | 2.13 | 3.35 |
| qg5s | 32 | 2.28 | 3.40 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-013 | 197x | 3.95 | 0.07 | 0.32 | 0.27 | 1.10 | 2.65 | 0.18 | 0.38 | 0.09 | 0.14 | 1.85 |
| r1-041 | 197x | 3.45 | 0.06 | 0.54 | 0.54 | 1.12 | 2.16 | 0.18 | 0.38 | 0.08 | 0.15 | 1.36 |
| r1-064 | qg5s | 3.40 | 0.07 | 0.56 | 0.28 | 1.10 | 2.11 | 0.19 | 0.09 | 0.08 | 0.10 | 1.20 |
| r1-004 | 9tc5 | 3.35 | 0.08 | 0.53 | 0.14 | 1.13 | 2.05 | 0.08 | 0.27 | 0.16 | 0.17 | 1.35 |
| r1-033 | qg5s | 3.32 | 0.07 | 0.30 | 0.21 | 1.08 | 2.03 | 0.22 | 0.05 | 0.05 | 0.51 | 1.18 |
| r1-082 | 9tc5 | 3.25 | 0.07 | 0.30 | 0.27 | 1.07 | 1.98 | 0.10 | 0.16 | 0.13 | 0.24 | 1.32 |
| r1-068 | 197x | 3.23 | 0.08 | 0.34 | 0.25 | 0.78 | 1.95 | 0.19 | 0.21 | 0.15 | 0.10 | 1.25 |
| r1-048 | qg5s | 3.22 | 0.08 | 0.49 | 0.39 | 1.11 | 1.92 | 0.17 | 0.07 | 0.03 | 0.54 | 1.09 |
| r1-061 | qg5s | 3.14 | 0.07 | 0.29 | 0.16 | 1.09 | 1.86 | 0.22 | 0.08 | 0.07 | 0.46 | 1.01 |
| r1-093 | 197x | 3.11 | 0.07 | 0.38 | 0.23 | 1.11 | 1.83 | 0.18 | 0.38 | 0.07 | 0.16 | 1.01 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 197x | 146.8 | 770.5 | 28228.2 | 1.6 | 11.6 | 219.3 | 7.4 | 53.5 | 68.9 | 24.6 | 39.5 | 47.3 | 0.0 | 472.9 | 8.1 | containerd-nydu/14682/containerd-nydu |
| 9tc5 | 155.0 | 646.9 | 23496.8 | 2.2 | 10.2 | 116.9 | 0.9 | 55.7 | 65.6 | 10.2 | 42.9 | 39.6 | 0.0 | 405.1 | 1.1 | containerd-nydu/14792/containerd-nydu |
| qg5s | 168.4 | 690.7 | 27668.2 | 1.3 | 11.8 | 212.5 | 8.1 | 59.5 | 69.1 | 6.3 | 44.4 | 47.6 | 0.0 | 436.8 | 0.5 | nydusd-amp/14724/state_machine |

## Top processes in the window, cores per node

- 197x: nydusd-amp 1.82, nydus-dragonfly 0.79, ateworker-gviso 0.56, objectstore-plu 0.47, containerd-nydu 0.21, fluent-bit 0.15, atelet 0.08, cilium-agent 0.07
- 9tc5: nydusd-amp 1.68, nydus-dragonfly 0.85, ateworker-gviso 0.75, objectstore-plu 0.60, containerd-nydu 0.25, fluent-bit 0.15, atelet 0.09, kubelet 0.08
- qg5s: nydusd-amp 1.74, nydus-dragonfly 0.73, objectstore-plu 0.47, ateworker-gviso 0.44, containerd-nydu 0.19, fluent-bit 0.15, atelet 0.08, cilium-agent 0.07

## Timeline, node 197x

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 15 | 1082 | 0.43 | 6 | 11 | 1 | 2.0 | 1.4 | 130 | 318.3 |
| 1 | 24 | 0 | 770 | 4947 | 1.63 | 219 | 69 | 25 | 9.1 | 20.4 | 88 | 5.0 |
| 2 | 19 | 144 | 195 | 11178 | 0.51 | 63 | 59 | 2 | 22.2 | 39.5 | 279 | 1.0 |
| 3 | 5 | 147 | 283 | 21878 | 0.43 | 71 | 59 | 7 | 36.9 | 31.7 | 336 | 1.0 |
| 4 | 0 | 61 | 269 | 28228 | 0.41 | 45 | 38 | 7 | 47.3 | 8.0 | 233 | 0.0 |

## Timeline, node 9tc5

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 76 | 1799 | 0.70 | 95 | 24 | 1 | 3.2 | 4.8 | 134 | 334.1 |
| 1 | 31 | 11 | 647 | 4344 | 2.21 | 117 | 66 | 10 | 13.4 | 25.0 | 83 | 4.4 |
| 2 | 22 | 155 | 152 | 11665 | 0.54 | 49 | 65 | 2 | 22.5 | 42.9 | 228 | 1.0 |
| 3 | 3 | 152 | 293 | 23497 | 0.43 | 70 | 57 | 7 | 39.6 | 29.6 | 334 | 0.7 |
| 5 | 0 | 72 | 286 | 29307 | 0.41 | 43 | 39 | 9 | 47.8 | 6.6 | 219 | 0.0 |

## Timeline, node qg5s

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 10 | 796 | 0.39 | 1 | 8 | 1 | 1.3 | 0.8 | 133 | 411.3 |
| 1 | 0 | 0 | 691 | 5075 | 1.28 | 212 | 57 | 2 | 21.4 | 18.0 | 41 | 16.6 |
| 2 | 32 | 125 | 138 | 8350 | 0.49 | 53 | 69 | 2 | 14.5 | 28.7 | 277 | 1.5 |
| 3 | 5 | 168 | 338 | 22247 | 0.45 | 65 | 64 | 6 | 36.6 | 44.4 | 374 | 0.3 |
| 4 | 0 | 71 | 330 | 27668 | 0.42 | 59 | 41 | 6 | 47.6 | 7.4 | 250 | 0.0 |
| 5 | 0 | 90 | 424 | 27078 | 0.55 | 33 | 32 | 5 | 44.8 | 4.1 | 191 | 0.0 |

## Seed requests (nginx)

3137 requests in the window, 979 MB. Seed time p50 / p99 / max: 0.0 / 190.9 / 428.0 ms. Size p50 / p99: 186 / 3849 KB. Status: {200: 384, 206: 2753}.
