# Bottleneck: benchmarking/image-plugin/results/20261104-nydus-cpu/e4-none-n32-r2/rep-1/nydus-none-w2

96 first turns. TTFI p50 / p95 / p99 / p100: 2.24 / 3.25 / 3.38 / 3.42 s. Window 3.4 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| d5hd | 32 | 2.23 | 3.38 |
| kfbd | 32 | 2.17 | 3.42 |
| mcv2 | 32 | 2.32 | 3.25 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-063 | kfbd | 3.42 | 0.06 | 0.27 | 0.17 | 1.21 | 2.03 | 0.08 | 0.14 | 0.04 | 0.40 | 1.35 |
| r1-046 | d5hd | 3.38 | 0.06 | 0.29 | 0.12 | 1.10 | 2.09 | 0.22 | 0.04 | 0.07 | 0.47 | 1.27 |
| r1-070 | d5hd | 3.28 | 0.07 | 0.24 | 0.21 | 1.08 | 2.02 | 0.26 | 0.10 | 0.03 | 0.41 | 1.18 |
| r1-024 | d5hd | 3.28 | 0.06 | 0.25 | 0.12 | 1.11 | 1.99 | 0.22 | 0.14 | 0.39 | 0.08 | 1.11 |
| r1-044 | mcv2 | 3.25 | 0.06 | 0.27 | 0.16 | 1.12 | 1.96 | 0.05 | 0.11 | 0.42 | 0.10 | 1.27 |
| r1-079 | mcv2 | 3.25 | 0.06 | 0.25 | 0.16 | 1.03 | 2.06 | 0.15 | 0.10 | 0.05 | 0.43 | 1.30 |
| r1-094 | kfbd | 3.16 | 0.07 | 0.25 | 0.14 | 1.15 | 1.83 | 0.12 | 0.04 | 0.04 | 0.18 | 1.44 |
| r1-059 | d5hd | 3.13 | 0.06 | 0.22 | 0.17 | 1.07 | 1.87 | 0.20 | 0.04 | 0.05 | 0.13 | 1.43 |
| r1-056 | mcv2 | 3.08 | 0.06 | 0.22 | 0.21 | 0.75 | 2.11 | 0.37 | 0.14 | 0.39 | 0.07 | 1.10 |
| r1-025 | kfbd | 3.07 | 0.07 | 0.33 | 0.33 | 1.15 | 1.75 | 0.13 | 0.14 | 0.05 | 0.41 | 1.00 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| d5hd | 136.6 | 761.2 | 26261.7 | 1.6 | 16.1 | 252.8 | 1.0 | 57.5 | 62.5 | 6.5 | 39.2 | 43.2 | 0.0 | 772.9 | 23.9 | containerd-nydu/14639/containerd-nydu |
| kfbd | 145.4 | 452.6 | 36463.3 | 1.3 | 15.9 | 283.9 | 0.9 | 60.5 | 65.4 | 26.0 | 48.5 | 53.0 | 0.0 | 497.0 | 0.0 | containerd-nydu/16356/containerd-nydu |
| mcv2 | 223.0 | 468.2 | 29490.6 | 1.4 | 13.9 | 225.6 | 6.3 | 50.7 | 55.9 | 13.0 | 30.1 | 45.5 | 0.0 | 382.0 | 0.0 | containerd-nydu/16168/containerd-nydu |

## Top processes in the window, cores per node

- d5hd: nydusd-amp 1.13, nydus-dragonfly 0.96, ateworker-gviso 0.90, objectstore-plu 0.61, containerd-nydu 0.25, fluent-bit 0.16, atelet 0.09, kubelet 0.09
- kfbd: nydusd-amp 1.32, nydus-dragonfly 1.02, objectstore-plu 0.60, ateworker-gviso 0.58, containerd-nydu 0.25, fluent-bit 0.16, containerd 0.10, atelet 0.10
- mcv2: nydusd-amp 1.34, nydus-dragonfly 0.79, ateworker-gviso 0.77, objectstore-plu 0.60, containerd-nydu 0.29, fluent-bit 0.20, atelet 0.10, kworker 0.10

## Timeline, node d5hd

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 12 | 673 | 0.51 | 10 | 15 | 2 | 1.7 | 2.0 | 130 | 372.3 |
| 1 | 11 | 0 | 761 | 5445 | 1.59 | 213 | 61 | 7 | 13.5 | 17.1 | 87 | 20.6 |
| 2 | 24 | 116 | 271 | 8778 | 0.72 | 117 | 62 | 4 | 18.1 | 34.1 | 239 | 3.2 |
| 3 | 5 | 137 | 453 | 26262 | 0.61 | 253 | 60 | 3 | 43.2 | 39.2 | 443 | 2.2 |
| 4 | 0 | 123 | 643 | 40749 | 0.52 | 204 | 38 | 4 | 58.7 | 5.5 | 352 | 0.0 |

## Timeline, node kfbd

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 389 | 3651 | 1.27 | 205 | 45 | 3 | 7.2 | 8.0 | 129 | 358.2 |
| 1 | 12 | 34 | 453 | 4296 | 1.32 | 61 | 47 | 26 | 13.1 | 21.2 | 76 | 9.5 |
| 3 | 3 | 145 | 421 | 14119 | 0.82 | 150 | 65 | 2 | 26.2 | 48.5 | 470 | 1.0 |
| 4 | 0 | 116 | 406 | 36463 | 0.43 | 284 | 48 | 4 | 53.0 | 20.5 | 344 | 0.0 |
| 5 | 0 | 128 | 300 | 34731 | 0.41 | 149 | 37 | 3 | 49.8 | 9.3 | 65 | 0.0 |

## Timeline, node mcv2

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 390 | 3317 | 1.42 | 187 | 40 | 3 | 4.8 | 9.2 | 130 | 192.1 |
| 1 | 30 | 6 | 315 | 2513 | 1.25 | 16 | 53 | 13 | 6.3 | 22.2 | 41 | 5.2 |
| 2 | 25 | 223 | 468 | 19118 | 0.70 | 214 | 55 | 3 | 33.8 | 27.1 | 359 | 1.4 |
| 4 | 0 | 184 | 414 | 29491 | 0.46 | 226 | 56 | 4 | 45.5 | 30.1 | 208 | 0.0 |
| 5 | 0 | 66 | 199 | 22809 | 0.40 | 97 | 25 | 3 | 39.0 | 2.7 | 74 | 0.0 |

## Seed requests (nginx)

3127 requests in the window, 2000 MB. Seed time p50 / p99 / max: 0.0 / 128.5 / 430.0 ms. Size p50 / p99: 600 / 3742 KB. Status: {200: 384, 206: 2743}.
