# Bottleneck: benchmarking/image-plugin/results/20261103-confirm-sweep/set-a/n16/rep-1/riptide-ms-pf-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 2.23 / 3.43 / 3.65 / 3.79 s. Window 3.8 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 145k | 20 | 3.00 | 3.79 |
| k5dh | 14 | 2.05 | 2.73 |
| vlpl | 14 | 1.85 | 3.50 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-004 | 145k | 3.79 | 0.03 | 0.18 | 0.19 | 3.31 | 0.32 | 0.00 | 0.03 | 0.01 | 0.15 | 0.07 |
| r1-024 | vlpl | 3.50 | 0.06 | 0.24 | 0.12 | 3.15 | 0.26 | 0.00 | 0.03 | 0.01 | 0.10 | 0.06 |
| r1-025 | 145k | 3.49 | 0.07 | 0.21 | 0.13 | 2.87 | 0.47 | 0.01 | 0.03 | 0.01 | 0.18 | 0.19 |
| r1-015 | 145k | 3.32 | 0.03 | 0.16 | 0.18 | 2.66 | 0.51 | 0.02 | 0.03 | 0.01 | 0.17 | 0.20 |
| r1-028 | 145k | 3.30 | 0.03 | 0.25 | 0.15 | 2.85 | 0.37 | 0.00 | 0.03 | 0.01 | 0.16 | 0.09 |
| r1-047 | 145k | 3.07 | 0.07 | 0.20 | 0.14 | 2.56 | 0.35 | 0.01 | 0.06 | 0.03 | 0.16 | 0.02 |
| r1-006 | 145k | 3.05 | 0.08 | 0.29 | 0.13 | 2.49 | 0.45 | 0.02 | 0.03 | 0.01 | 0.22 | 0.08 |
| r1-034 | 145k | 3.05 | 0.03 | 0.25 | 0.13 | 2.55 | 0.35 | 0.00 | 0.03 | 0.04 | 0.20 | 0.01 |
| r1-042 | 145k | 3.05 | 0.09 | 0.24 | 0.14 | 2.54 | 0.38 | 0.03 | 0.03 | 0.01 | 0.17 | 0.04 |
| r1-022 | 145k | 3.05 | 0.03 | 0.26 | 0.12 | 2.46 | 0.50 | 0.01 | 0.03 | 0.02 | 0.25 | 0.14 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 145k | 0.1 | 492.4 | 4467.0 | 1.7 | 5.1 | 169.5 | 14.9 | 80.4 | 84.5 | 49.1 | 38.1 | 6.8 | 0.0 | 13.3 | 0.6 | gcfsd-v2/9107/gcfsd-v2 |
| k5dh | 0.1 | 407.4 | 3857.6 | 1.6 | 4.1 | 137.5 | 16.0 | 79.6 | 82.5 | 49.5 | 34.0 | 5.5 | 0.0 | 6.3 | 0.3 | gcfsd-v2/9025/gcfsd-v2 |
| vlpl | 0.1 | 524.4 | 3428.0 | 1.4 | 4.9 | 117.7 | 2.8 | 81.1 | 85.4 | 51.4 | 35.0 | 14.2 | 0.0 | 7.1 | 0.0 | gcfsd-v2/9054/gcfsd-v2 |

## Top processes in the window, cores per node

- 145k: gcfsd-v2 7.62, riptide 1.65, objectstore-plu 0.38, ateworker-gviso 0.24, fluent-bit 0.13, containerd-gcfs 0.11, atelet 0.06, containerd 0.06
- k5dh: gcfsd-v2 6.86, riptide 2.11, objectstore-plu 0.25, ateworker-gviso 0.18, fluent-bit 0.09, containerd-gcfs 0.08, containerd 0.04, atelet 0.04
- vlpl: gcfsd-v2 6.98, riptide 1.94, objectstore-plu 0.20, ateworker-gviso 0.15, fluent-bit 0.07, kubelet 0.07, containerd-gcfs 0.06, cilium-agent 0.03

## Timeline, node 145k

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 352 | 1610 | 1.71 | 56 | 20 | 3 | 1.5 | 1.3 | 0 | — |
| 1 | 1 | 0 | 476 | 3270 | 1.27 | 170 | 55 | 47 | 3.8 | 5.8 | 0 | — |
| 2 | 13 | 0 | 492 | 4467 | 1.13 | 60 | 84 | 49 | 6.8 | 32.1 | 0 | — |
| 3 | 9 | 0 | 184 | 1276 | 0.94 | 56 | 84 | 45 | 3.5 | 38.1 | 0 | — |
| 4 | 0 | 0 | 181 | 800 | 1.26 | 21 | 85 | 51 | 5.3 | 37.2 | 0 | — |

## Timeline, node k5dh

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 307 | 1434 | 1.64 | 42 | 16 | 5 | 1.4 | 0.9 | 0 | — |
| 1 | 8 | 0 | 407 | 3858 | 1.05 | 138 | 56 | 38 | 3.9 | 9.8 | 0 | — |
| 2 | 8 | 0 | 52 | 929 | 0.73 | 22 | 83 | 40 | 2.0 | 34.0 | 0 | — |
| 3 | 0 | 0 | 21 | 173 | 1.30 | 18 | 83 | 49 | 5.5 | 32.8 | 0 | — |
| 4 | 0 | 0 | 19 | 161 | 1.34 | 23 | 80 | 48 | 9.9 | 31.8 | 0 | — |

## Timeline, node vlpl

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 162 | 674 | 1.26 | 2 | 9 | 2 | 0.8 | 0.4 | 0 | — |
| 1 | 9 | 0 | 524 | 3428 | 1.40 | 118 | 39 | 25 | 3.0 | 4.1 | 0 | — |
| 2 | 4 | 0 | 320 | 2716 | 1.04 | 102 | 83 | 43 | 3.3 | 35.0 | 0 | — |
| 3 | 1 | 0 | 184 | 995 | 1.13 | 25 | 84 | 49 | 10.8 | 34.1 | 0 | — |
| 4 | 0 | 0 | 158 | 653 | 1.24 | 40 | 85 | 51 | 14.2 | 34.8 | 0 | — |
| 5 | 0 | 0 | 186 | 809 | 1.35 | 13 | 81 | 43 | 8.2 | 33.9 | 0 | — |

## Seed requests (nginx)

No seed access log.
