# Bottleneck: benchmarking/image-plugin/results/20261103-confirm-sweep/set-b/n24/rep-1/riptide-ms-pfb-w2

72 first turns. TTFI p50 / p95 / p99 / p100: 3.82 / 5.59 / 5.65 / 5.67 s. Window 5.7 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| fxnx | 23 | 3.90 | 4.16 |
| n0mb | 26 | 3.87 | 5.67 |
| vdhc | 23 | 2.95 | 4.14 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-028 | n0mb | 5.67 | 0.03 | 0.49 | 0.11 | 3.72 | 1.84 | 0.01 | 0.03 | 0.01 | 0.23 | 1.50 |
| r1-063 | n0mb | 5.65 | 0.09 | 0.39 | 0.18 | 3.76 | 1.75 | 0.00 | 0.03 | 0.01 | 0.19 | 1.45 |
| r1-010 | n0mb | 5.65 | 0.04 | 0.51 | 0.13 | 3.85 | 1.69 | 0.01 | 0.03 | 0.01 | 0.16 | 1.42 |
| r1-046 | n0mb | 5.60 | 0.11 | 0.34 | 0.17 | 3.72 | 1.73 | 0.03 | 0.03 | 0.01 | 0.19 | 1.43 |
| r1-002 | n0mb | 5.59 | 0.11 | 0.35 | 0.25 | 3.24 | 2.19 | 0.02 | 0.05 | 0.02 | 0.24 | 1.74 |
| r1-056 | n0mb | 4.49 | 0.04 | 0.37 | 0.13 | 3.20 | 1.09 | 0.02 | 0.03 | 0.01 | 0.25 | 0.70 |
| r1-042 | fxnx | 4.16 | 0.04 | 0.43 | 0.30 | 3.65 | 0.33 | 0.02 | 0.03 | 0.02 | 0.14 | 0.05 |
| r1-055 | vdhc | 4.14 | 0.04 | 0.43 | 0.16 | 3.40 | 0.55 | 0.01 | 0.03 | 0.04 | 0.19 | 0.23 |
| r1-019 | fxnx | 4.14 | 0.04 | 0.40 | 0.15 | 3.58 | 0.35 | 0.04 | 0.06 | 0.01 | 0.17 | 0.02 |
| r1-057 | fxnx | 4.11 | 0.11 | 0.39 | 0.19 | 3.46 | 0.40 | 0.01 | 0.08 | 0.02 | 0.20 | 0.01 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| fxnx | 0.2 | 591.2 | 3706.8 | 1.8 | 5.8 | 117.0 | 3.7 | 82.3 | 84.8 | 55.5 | 41.0 | 11.5 | 0.0 | 13.2 | 0.2 | gcfsd-v2/8177/gcfsd-v2 |
| n0mb | 0.1 | 459.4 | 4038.1 | 3.3 | 5.9 | 159.5 | 1.9 | 78.1 | 80.4 | 48.3 | 35.0 | 4.8 | 0.0 | 13.9 | 8.1 | gcfsd-v2/8458/gcfsd-v2 |
| vdhc | 0.2 | 532.5 | 3774.4 | 1.6 | 5.4 | 151.8 | 6.5 | 82.1 | 87.4 | 47.4 | 40.1 | 17.3 | 0.0 | 18.9 | 0.0 | gcfsd-v2/8235/gcfsd-v2 |

## Top processes in the window, cores per node

- fxnx: gcfsd-v2 7.22, riptide 2.11, objectstore-plu 0.25, ateworker-gviso 0.18, fluent-bit 0.10, kubelet 0.07, containerd-gcfs 0.07, cilium-agent 0.05
- n0mb: gcfsd-v2 7.13, riptide 1.94, objectstore-plu 0.34, ateworker-gviso 0.23, fluent-bit 0.11, containerd-gcfs 0.09, atelet 0.05, containerd 0.05
- vdhc: gcfsd-v2 7.83, riptide 2.08, objectstore-plu 0.28, ateworker-gviso 0.20, fluent-bit 0.10, containerd-gcfs 0.08, kubelet 0.05, atelet 0.05

## Timeline, node fxnx

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 218 | 881 | 1.33 | 0 | 2 | 1 | 3.6 | 0.1 | 0 | — |
| 1 | 3 | 0 | 505 | 2596 | 1.81 | 87 | 31 | 12 | 2.8 | 2.4 | 0 | — |
| 2 | 3 | 0 | 460 | 2846 | 1.43 | 117 | 65 | 51 | 3.7 | 11.9 | 0 | — |
| 3 | 11 | 0 | 591 | 3707 | 1.56 | 89 | 80 | 55 | 8.0 | 24.0 | 0 | — |
| 4 | 7 | 0 | 125 | 2486 | 0.71 | 26 | 84 | 36 | 7.1 | 41.0 | 0 | — |
| 5 | 0 | 0 | 9 | 57 | 1.61 | 11 | 82 | 47 | 2.6 | 38.2 | 0 | — |
| 6 | 0 | 0 | 31 | 404 | 1.19 | 38 | 85 | 47 | 11.5 | 39.5 | 0 | — |
| 7 | 0 | 0 | 3 | 116 | 0.73 | 21 | 84 | 46 | 17.2 | 39.0 | 0 | — |

## Timeline, node n0mb

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 255 | 1311 | 3.28 | 70 | 22 | 6 | 2.4 | 2.7 | 0 | — |
| 1 | 1 | 0 | 369 | 2648 | 1.56 | 160 | 48 | 46 | 3.5 | 5.2 | 0 | — |
| 2 | 0 | 0 | 459 | 3065 | 1.90 | 83 | 50 | 44 | 2.4 | 4.6 | 0 | — |
| 3 | 21 | 0 | 200 | 4038 | 0.79 | 19 | 74 | 30 | 4.8 | 31.2 | 0 | — |
| 4 | 6 | 0 | 4 | 175 | 0.54 | 12 | 80 | 48 | 0.5 | 35.0 | 0 | — |
| 6 | 0 | 0 | 13 | 204 | 0.90 | 4 | 80 | 47 | 1.3 | 34.0 | 0 | — |
| 7 | 0 | 0 | 1 | 115 | 0.60 | 13 | 84 | 49 | 3.6 | 38.6 | 0 | — |

## Timeline, node vdhc

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 361 | 1638 | 1.56 | 62 | 19 | 3 | 3.4 | 1.5 | 0 | — |
| 1 | 2 | 0 | 533 | 3774 | 1.41 | 152 | 44 | 39 | 3.3 | 3.9 | 0 | — |
| 2 | 13 | 0 | 411 | 3527 | 1.08 | 112 | 76 | 45 | 5.1 | 25.3 | 0 | — |
| 3 | 9 | 0 | 244 | 2313 | 1.53 | 31 | 87 | 42 | 4.3 | 39.0 | 0 | — |
| 5 | 0 | 0 | 190 | 901 | 1.22 | 19 | 84 | 47 | 5.3 | 39.6 | 0 | — |
| 6 | 0 | 0 | 168 | 694 | 1.22 | 40 | 84 | 45 | 17.3 | 40.1 | 0 | — |
| 7 | 0 | 0 | 192 | 878 | 1.28 | 37 | 86 | 49 | 18.7 | 39.1 | 0 | — |

## Seed requests (nginx)

No seed access log.
