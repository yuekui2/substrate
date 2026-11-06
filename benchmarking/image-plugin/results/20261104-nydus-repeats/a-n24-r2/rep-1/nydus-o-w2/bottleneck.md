# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-repeats/driver/../a-n24-r2/rep-1/nydus-o-w2

72 first turns. TTFI p50 / p95 / p99 / p100: 1.89 / 2.70 / 2.88 / 2.92 s. Window 2.9 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 0c37 | 23 | 1.71 | 2.42 |
| c0vs | 26 | 2.11 | 2.92 |
| qz7h | 23 | 1.67 | 2.23 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-008 | c0vs | 2.92 | 0.06 | 0.35 | 0.15 | 0.98 | 1.78 | 0.11 | 0.05 | 0.04 | 0.44 | 1.11 |
| r1-059 | c0vs | 2.86 | 0.08 | 0.29 | 0.19 | 1.04 | 1.63 | 0.02 | 0.06 | 0.04 | 0.41 | 1.05 |
| r1-004 | c0vs | 2.80 | 0.06 | 0.32 | 0.16 | 0.97 | 1.65 | 0.11 | 0.07 | 0.05 | 0.40 | 0.99 |
| r1-015 | c0vs | 2.72 | 0.08 | 0.32 | 0.17 | 0.97 | 1.57 | 0.11 | 0.05 | 0.05 | 0.46 | 0.88 |
| r1-049 | c0vs | 2.68 | 0.07 | 0.38 | 0.59 | 0.98 | 1.53 | 0.11 | 0.07 | 0.04 | 0.40 | 0.86 |
| r1-054 | c0vs | 2.57 | 0.06 | 0.41 | 0.17 | 0.99 | 1.41 | 0.10 | 0.07 | 0.06 | 0.41 | 0.75 |
| r1-028 | c0vs | 2.57 | 0.06 | 0.37 | 0.12 | 0.97 | 1.44 | 0.13 | 0.05 | 0.04 | 0.41 | 0.76 |
| r1-063 | 0c37 | 2.42 | 0.07 | 0.40 | 0.18 | 0.86 | 1.39 | 0.01 | 0.07 | 0.06 | 0.33 | 0.87 |
| r1-001 | c0vs | 2.40 | 0.06 | 0.52 | 0.13 | 0.78 | 1.45 | 0.30 | 0.05 | 0.04 | 0.43 | 0.59 |
| r1-055 | c0vs | 2.39 | 0.06 | 0.32 | 0.19 | 0.79 | 1.43 | 0.31 | 0.05 | 0.04 | 0.44 | 0.57 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0c37 | 163.6 | 451.2 | 22695.5 | 1.6 | 9.2 | 166.6 | 8.4 | 47.3 | 53.5 | 6.4 | 34.8 | 35.8 | 0.0 | 563.3 | 0.3 | nydusd-amp/15482/state_machine |
| c0vs | 138.7 | 591.5 | 20505.1 | 1.4 | 9.2 | 171.6 | 1.0 | 50.9 | 58.5 | 4.6 | 35.9 | 35.7 | 0.0 | 347.6 | 0.1 | nydusd-amp/15405/state_machine |
| qz7h | 117.1 | 536.4 | 24210.6 | 1.3 | 9.6 | 133.0 | 9.3 | 47.4 | 54.7 | 5.5 | 38.9 | 43.2 | 0.0 | 317.6 | 0.0 | nydusd-amp/15422/state_machine |

## Top processes in the window, cores per node

- 0c37: nydusd-amp 1.48, nydus-dragonfly 0.95, objectstore-plu 0.60, ateworker-gviso 0.52, containerd-nydu 0.26, fluent-bit 0.17, atelet 0.08, kubelet 0.07
- c0vs: nydusd-amp 1.10, nydus-dragonfly 0.72, ateworker-gviso 0.60, objectstore-plu 0.51, containerd-nydu 0.20, fluent-bit 0.16, containerd 0.09, kubelet 0.09
- qz7h: nydusd-amp 1.29, nydus-dragonfly 0.61, objectstore-plu 0.43, ateworker-gviso 0.41, containerd-nydu 0.18, fluent-bit 0.16, kubelet 0.08, atelet 0.07

## Timeline, node 0c37

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 451 | 3550 | 1.62 | 167 | 40 | 2 | 7.9 | 13.2 | 98 | 343.2 |
| 2 | 4 | 93 | 203 | 6428 | 0.57 | 41 | 51 | 1 | 12.2 | 22.6 | 352 | 1.0 |
| 3 | 0 | 164 | 269 | 22695 | 0.41 | 64 | 53 | 6 | 35.8 | 34.8 | 261 | 0.0 |
| 4 | 0 | 64 | 297 | 24015 | 0.42 | 49 | 35 | 6 | 44.3 | 7.3 | 156 | 0.0 |

## Timeline, node c0vs

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 68 | 0.45 | 1 | 7 | 2 | 0.5 | 1.1 | 107 | 283.7 |
| 1 | 26 | 0 | 592 | 4724 | 1.44 | 172 | 59 | 3 | 8.6 | 15.3 | 80 | 1.4 |
| 2 | 15 | 92 | 109 | 6252 | 0.52 | 39 | 48 | 1 | 13.1 | 24.0 | 273 | 1.0 |
| 3 | 0 | 139 | 276 | 20505 | 0.44 | 62 | 58 | 5 | 35.7 | 35.9 | 306 | 0.0 |
| 4 | 0 | 63 | 261 | 26796 | 0.40 | 51 | 36 | 6 | 44.9 | 5.0 | 244 | 0.0 |

## Timeline, node qz7h

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 12 | 1070 | 0.42 | 32 | 14 | 1 | 1.9 | 1.5 | 98 | 227.8 |
| 1 | 23 | 60 | 536 | 5343 | 1.30 | 133 | 55 | 5 | 13.1 | 20.5 | 171 | 1.6 |
| 2 | 2 | 117 | 218 | 16895 | 0.41 | 52 | 52 | 2 | 27.4 | 38.9 | 248 | 1.0 |
| 3 | 0 | 60 | 254 | 24211 | 0.39 | 46 | 30 | 5 | 43.2 | 4.6 | 197 | 0.0 |
| 4 | 0 | 57 | 195 | 13546 | 0.43 | 19 | 23 | 5 | 24.4 | 2.8 | 89 | 0.0 |

## Seed requests (nginx)

2300 requests in the window, 726 MB. Seed time p50 / p99 / max: 0.0 / 168.0 / 381.0 ms. Size p50 / p99: 185 / 3805 KB. Status: {200: 288, 206: 2012}.
