# Bottleneck: benchmarking/image-plugin/results/20261103-confirm-sweep/set-c/n16/rep-1/riptide-ms-pfc-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 3.07 / 4.05 / 4.13 / 4.15 s. Window 4.2 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| fpwf | 14 | 2.30 | 2.59 |
| wt8j | 18 | 3.58 | 4.12 |
| wz2c | 16 | 3.51 | 4.15 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-015 | wz2c | 4.15 | 0.03 | 0.41 | 0.17 | 3.84 | 0.21 | 0.00 | 0.03 | 0.01 | 0.10 | 0.01 |
| r1-034 | wt8j | 4.12 | 0.06 | 0.69 | 0.17 | 3.69 | 0.27 | 0.01 | 0.03 | 0.01 | 0.15 | 0.01 |
| r1-018 | wt8j | 4.10 | 0.04 | 0.29 | 0.15 | 3.54 | 0.23 | 0.00 | 0.03 | 0.03 | 0.11 | 0.01 |
| r1-002 | wt8j | 3.95 | 0.02 | 0.42 | 0.22 | 3.52 | 0.34 | 0.01 | 0.03 | 0.04 | 0.13 | 0.06 |
| r1-040 | wt8j | 3.91 | 0.06 | 0.51 | 0.13 | 3.56 | 0.25 | 0.01 | 0.05 | 0.01 | 0.11 | 0.02 |
| r1-032 | wt8j | 3.88 | 0.07 | 0.37 | 0.11 | 3.40 | 0.25 | 0.00 | 0.07 | 0.01 | 0.11 | 0.01 |
| r1-036 | wt8j | 3.86 | 0.07 | 0.40 | 0.22 | 3.36 | 0.27 | 0.01 | 0.03 | 0.01 | 0.13 | 0.03 |
| r1-021 | wz2c | 3.80 | 0.08 | 0.39 | 0.18 | 3.36 | 0.33 | 0.00 | 0.03 | 0.01 | 0.12 | 0.11 |
| r1-028 | wt8j | 3.80 | 0.05 | 0.41 | 0.18 | 3.42 | 0.27 | 0.00 | 0.03 | 0.01 | 0.14 | 0.02 |
| r1-030 | wt8j | 3.73 | 0.07 | 0.28 | 0.15 | 3.19 | 0.30 | 0.00 | 0.03 | 0.01 | 0.15 | 0.03 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| fpwf | 0.1 | 306.7 | 3023.7 | 1.8 | 3.8 | 142.6 | 0.9 | 79.6 | 81.6 | 47.6 | 34.1 | 18.7 | 0.0 | 21.2 | 1.4 | gcfsd-v2/12777/gcfsd-v2 |
| wt8j | 0.1 | 321.3 | 2753.3 | 1.7 | 3.9 | 100.1 | 1.0 | 79.4 | 82.5 | 53.3 | 39.0 | 12.1 | 0.0 | 11.6 | 3.1 | gcfsd-v2/9950/gcfsd-v2 |
| wz2c | 0.1 | 376.6 | 3320.3 | 2.2 | 4.2 | 126.2 | 1.9 | 79.2 | 82.4 | 50.6 | 34.0 | 6.2 | 0.0 | 9.6 | 6.5 | gcfsd-v2/12995/gcfsd-v2 |

## Top processes in the window, cores per node

- fpwf: gcfsd-v2 6.54, riptide 2.01, objectstore-plu 0.21, ateworker-gviso 0.16, fluent-bit 0.08, kubelet 0.08, containerd-gcfs 0.06, containerd 0.05
- wt8j: gcfsd-v2 6.49, riptide 1.66, objectstore-plu 0.26, ateworker-gviso 0.18, fluent-bit 0.10, containerd-gcfs 0.09, kubelet 0.07, containerd 0.07
- wz2c: gcfsd-v2 5.45, riptide 1.04, objectstore-plu 0.24, ateworker-gviso 0.18, fluent-bit 0.09, containerd-gcfs 0.08, kubelet 0.05, containerd 0.05

## Timeline, node fpwf

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 52 | 0.59 | 0 | 4 | 2 | 0.2 | 0.3 | 0 | — |
| 1 | 2 | 0 | 307 | 2177 | 1.72 | 143 | 38 | 27 | 3.2 | 2.4 | 0 | — |
| 2 | 12 | 0 | 159 | 3024 | 0.84 | 59 | 71 | 39 | 4.8 | 24.6 | 0 | — |
| 3 | 0 | 0 | 9 | 61 | 1.80 | 24 | 82 | 48 | 5.3 | 34.1 | 0 | — |
| 4 | 0 | 0 | 22 | 214 | 1.27 | 35 | 80 | 48 | 18.7 | 30.6 | 0 | — |
| 5 | 0 | 0 | 1 | 64 | 0.49 | 24 | 79 | 48 | 20.4 | 27.4 | 0 | — |

## Timeline, node wt8j

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 53 | 0.45 | 0 | 6 | 2 | 0.3 | 0.3 | 0 | — |
| 1 | 6 | 0 | 321 | 2084 | 1.73 | 90 | 29 | 18 | 3.1 | 1.7 | 0 | — |
| 2 | 0 | 0 | 83 | 1331 | 0.84 | 77 | 76 | 43 | 4.6 | 21.7 | 0 | — |
| 3 | 10 | 0 | 314 | 2753 | 1.42 | 100 | 82 | 53 | 12.1 | 23.0 | 0 | — |
| 4 | 2 | 0 | 97 | 1402 | 0.98 | 36 | 83 | 46 | 11.2 | 39.0 | 0 | — |
| 5 | 0 | 0 | 18 | 106 | 1.89 | 17 | 82 | 48 | 17.1 | 36.3 | 0 | — |

## Timeline, node wz2c

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 76 | 0.52 | 0 | 4 | 1 | 0.2 | 0.3 | 0 | — |
| 1 | 0 | 0 | 177 | 870 | 2.20 | 56 | 18 | 6 | 1.3 | 0.9 | 0 | — |
| 2 | 4 | 0 | 377 | 2434 | 1.70 | 126 | 41 | 35 | 2.2 | 2.0 | 0 | — |
| 3 | 11 | 0 | 235 | 3320 | 1.02 | 78 | 77 | 49 | 6.2 | 27.4 | 0 | — |
| 4 | 1 | 0 | 64 | 619 | 1.43 | 62 | 82 | 51 | 5.8 | 34.0 | 0 | — |
| 5 | 0 | 0 | 8 | 130 | 0.97 | 19 | 83 | 49 | 8.8 | 33.7 | 0 | — |

## Seed requests (nginx)

No seed access log.
