# Bottleneck: benchmarking/image-plugin/results/20261103-confirm-sweep/set-b/n16/rep-1/riptide-ms-pfb-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 2.71 / 3.45 / 3.46 / 3.46 s. Window 3.5 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 1j1j | 11 | 2.14 | 2.35 |
| 285s | 24 | 3.18 | 3.46 |
| qn8l | 13 | 2.56 | 3.45 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-046 | 285s | 3.46 | 0.17 | 0.39 | 0.15 | 2.98 | 0.26 | 0.01 | 0.03 | 0.01 | 0.14 | 0.01 |
| r1-028 | qn8l | 3.45 | 0.03 | 0.27 | 0.29 | 2.79 | 0.30 | 0.00 | 0.03 | 0.01 | 0.12 | 0.09 |
| r1-010 | 285s | 3.45 | 0.03 | 0.22 | 0.15 | 2.63 | 0.46 | 0.00 | 0.08 | 0.03 | 0.22 | 0.08 |
| r1-017 | 285s | 3.44 | 0.03 | 0.49 | 0.15 | 2.86 | 0.50 | 0.01 | 0.11 | 0.01 | 0.23 | 0.08 |
| r1-040 | 285s | 3.44 | 0.17 | 0.19 | 0.11 | 2.57 | 0.45 | 0.01 | 0.08 | 0.03 | 0.20 | 0.07 |
| r1-025 | 285s | 3.44 | 0.03 | 0.29 | 0.14 | 2.67 | 0.42 | 0.03 | 0.03 | 0.05 | 0.20 | 0.05 |
| r1-038 | 285s | 3.28 | 0.18 | 0.21 | 0.15 | 2.47 | 0.39 | 0.01 | 0.06 | 0.03 | 0.18 | 0.01 |
| r1-022 | 285s | 3.27 | 0.03 | 0.34 | 0.21 | 2.52 | 0.39 | 0.01 | 0.03 | 0.03 | 0.23 | 0.00 |
| r1-011 | 285s | 3.27 | 0.14 | 0.34 | 0.20 | 2.69 | 0.37 | 0.01 | 0.06 | 0.03 | 0.20 | 0.01 |
| r1-018 | 285s | 3.26 | 0.03 | 0.25 | 0.15 | 2.35 | 0.55 | 0.04 | 0.04 | 0.01 | 0.25 | 0.10 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1j1j | 0.1 | 474.8 | 3027.8 | 1.3 | 4.0 | 139.3 | 8.4 | 79.3 | 81.2 | 49.5 | 29.9 | 13.1 | 0.0 | 8.7 | 0.5 | gcfsd-v2/7772/gcfsd-v2 |
| 285s | 0.1 | 335.0 | 3780.7 | 1.7 | 3.7 | 142.6 | 4.7 | 76.8 | 79.6 | 48.7 | 38.2 | 4.4 | 0.0 | 23.0 | 0.7 | gcfsd-v2/10302/gcfsd-v2 |
| qn8l | 0.1 | 393.0 | 2814.8 | 2.8 | 3.5 | 110.4 | 2.9 | 78.8 | 82.2 | 47.5 | 34.2 | 4.6 | 0.0 | 9.8 | 1.8 | gcfsd-v2/7824/gcfsd-v2 |

## Top processes in the window, cores per node

- 1j1j: gcfsd-v2 6.10, riptide 1.72, objectstore-plu 0.21, ateworker-gviso 0.13, fluent-bit 0.09, containerd-gcfs 0.07, kubelet 0.07, containerd 0.04
- 285s: gcfsd-v2 6.99, riptide 2.03, objectstore-plu 0.45, ateworker-gviso 0.31, fluent-bit 0.18, containerd-gcfs 0.12, kubelet 0.10, atelet 0.09
- qn8l: gcfsd-v2 6.19, riptide 1.84, objectstore-plu 0.25, ateworker-gviso 0.16, fluent-bit 0.10, containerd-gcfs 0.07, kubelet 0.06, cilium-agent 0.05

## Timeline, node 1j1j

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 218 | 887 | 1.28 | 0 | 5 | 2 | 1.4 | 0.2 | 0 | — |
| 1 | 5 | 0 | 475 | 3028 | 1.29 | 139 | 43 | 29 | 3.1 | 5.0 | 0 | — |
| 2 | 6 | 0 | 357 | 2895 | 1.04 | 44 | 77 | 44 | 6.2 | 26.4 | 0 | — |
| 3 | 0 | 0 | 221 | 913 | 1.29 | 26 | 81 | 50 | 13.1 | 29.9 | 0 | — |
| 4 | 0 | 0 | 97 | 496 | 1.15 | 30 | 79 | 50 | 18.9 | 25.9 | 0 | — |

## Timeline, node 285s

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 275 | 1692 | 1.66 | 143 | 32 | 15 | 2.0 | 3.6 | 0 | — |
| 1 | 3 | 0 | 335 | 3052 | 1.18 | 129 | 61 | 49 | 3.4 | 10.7 | 0 | — |
| 3 | 18 | 0 | 256 | 3781 | 0.88 | 16 | 79 | 36 | 4.4 | 31.0 | 0 | — |
| 4 | 0 | 0 | 32 | 491 | 0.96 | 3 | 80 | 38 | 0.8 | 38.2 | 0 | — |
| 5 | 0 | 0 | 12 | 118 | 1.13 | 15 | 84 | 47 | 4.0 | 39.6 | 0 | — |

## Timeline, node qn8l

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 278 | 1237 | 2.84 | 39 | 15 | 2 | 3.5 | 0.6 | 0 | — |
| 2 | 11 | 0 | 393 | 2815 | 1.25 | 110 | 40 | 33 | 3.2 | 4.1 | 0 | — |
| 3 | 2 | 0 | 305 | 2657 | 0.95 | 54 | 80 | 45 | 3.9 | 34.2 | 0 | — |
| 4 | 0 | 0 | 197 | 924 | 1.12 | 24 | 82 | 48 | 4.6 | 32.0 | 0 | — |
| 5 | 0 | 0 | 203 | 906 | 1.18 | 14 | 80 | 49 | 7.6 | 31.0 | 0 | — |

## Seed requests (nginx)

No seed access log.
