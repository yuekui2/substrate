# Bottleneck: benchmarking/image-plugin/results/20261103-confirm-sweep/set-a/n32/rep-1/nydus-pf-w2

96 first turns. TTFI p50 / p95 / p99 / p100: 2.17 / 4.54 / 4.85 / 4.94 s. Window 4.9 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 82qf | 32 | 2.16 | 4.61 |
| hr6f | 32 | 2.11 | 4.71 |
| q4x1 | 32 | 2.51 | 4.94 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-063 | q4x1 | 4.94 | 0.10 | 0.32 | 0.19 | 1.26 | 3.45 | 0.14 | 0.23 | 0.16 | 0.11 | 2.77 |
| r1-025 | q4x1 | 4.84 | 0.05 | 0.43 | 0.17 | 1.27 | 3.44 | 0.08 | 0.18 | 0.05 | 0.35 | 2.77 |
| r1-094 | hr6f | 4.71 | 0.09 | 0.47 | 0.39 | 1.09 | 3.45 | 0.22 | 0.32 | 0.07 | 0.21 | 2.60 |
| r1-086 | 82qf | 4.61 | 0.06 | 0.45 | 0.31 | 1.15 | 3.23 | 0.03 | 0.08 | 0.06 | 0.15 | 2.52 |
| r1-024 | q4x1 | 4.61 | 0.08 | 0.37 | 0.11 | 1.22 | 3.23 | 0.06 | 0.21 | 0.04 | 0.25 | 2.52 |
| r1-092 | 82qf | 4.51 | 0.12 | 0.50 | 0.27 | 1.03 | 3.24 | 0.07 | 0.10 | 0.07 | 0.46 | 2.52 |
| r1-044 | hr6f | 4.45 | 0.05 | 0.67 | 0.24 | 1.12 | 3.18 | 0.25 | 0.30 | 0.08 | 0.18 | 2.35 |
| r1-074 | q4x1 | 4.41 | 0.12 | 0.39 | 0.23 | 1.16 | 3.07 | 0.15 | 0.19 | 0.06 | 0.10 | 2.27 |
| r1-048 | 82qf | 4.37 | 0.11 | 0.57 | 0.31 | 1.03 | 3.17 | 0.12 | 0.09 | 0.05 | 0.49 | 2.38 |
| r1-062 | hr6f | 4.37 | 0.06 | 0.40 | 0.23 | 1.03 | 3.11 | 0.24 | 0.32 | 0.07 | 0.20 | 2.27 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 82qf | 165.2 | 571.7 | 35784.4 | 1.5 | 20.7 | 213.3 | 9.9 | 81.3 | 90.7 | 8.1 | 66.3 | 54.9 | 0.0 | 673.0 | 0.4 | nydusd-amp/14675/state_machine |
| hr6f | 177.9 | 702.3 | 32418.7 | 0.9 | 18.1 | 158.9 | 4.5 | 80.8 | 90.9 | 14.5 | 59.6 | 51.7 | 0.0 | 658.2 | 1.3 | containerd-nydu/14304/containerd-nydu |
| q4x1 | 187.4 | 627.7 | 34585.1 | 1.4 | 15.3 | 205.1 | 0.0 | 79.2 | 89.5 | 11.0 | 50.3 | 50.7 | 0.0 | 760.0 | 0.9 | containerd-nydu/15793/containerd-nydu |

## Top processes in the window, cores per node

- 82qf: nydus-dragonfly 5.72, nydusd-amp 1.94, ateworker-gviso 0.52, objectstore-plu 0.47, containerd-nydu 0.19, fluent-bit 0.16, kworker 0.09, atelet 0.08
- hr6f: nydus-dragonfly 5.28, nydusd-amp 1.76, objectstore-plu 0.47, ateworker-gviso 0.41, containerd-nydu 0.19, fluent-bit 0.14, kubelet 0.08, atelet 0.07
- q4x1: nydus-dragonfly 5.84, nydusd-amp 1.85, ateworker-gviso 0.68, objectstore-plu 0.49, containerd-nydu 0.20, fluent-bit 0.16, kworker 0.09, atelet 0.08

## Timeline, node 82qf

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 52 | 0.39 | 0 | 3 | 1 | 0.2 | 0.1 | 132 | 534.0 |
| 1 | 30 | 8 | 572 | 6642 | 1.47 | 213 | 47 | 3 | 14.9 | 18.7 | 85 | 1.6 |
| 2 | 23 | 157 | 251 | 22694 | 0.56 | 34 | 85 | 6 | 37.1 | 39.9 | 167 | 1.3 |
| 3 | 7 | 135 | 370 | 24556 | 0.52 | 38 | 91 | 1 | 32.9 | 66.3 | 208 | 3.9 |
| 4 | 3 | 165 | 420 | 35784 | 0.57 | 56 | 82 | 5 | 54.9 | 34.4 | 296 | 0.1 |
| 5 | 0 | 102 | 286 | 31444 | 0.42 | 48 | 59 | 8 | 49.6 | 9.0 | 215 | 0.0 |
| 6 | 0 | 131 | 230 | 32191 | 0.40 | 43 | 48 | 6 | 49.6 | 5.4 | 166 | 0.0 |

## Timeline, node hr6f

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 143 | 2450 | 0.93 | 48 | 21 | 2 | 3.6 | 4.6 | 131 | 364.7 |
| 1 | 21 | 56 | 702 | 18171 | 0.76 | 159 | 77 | 15 | 37.3 | 31.6 | 91 | 6.1 |
| 2 | 31 | 178 | 127 | 22216 | 0.46 | 47 | 90 | 2 | 34.2 | 59.6 | 151 | 1.5 |
| 4 | 3 | 167 | 163 | 31171 | 0.44 | 43 | 91 | 5 | 46.9 | 52.9 | 328 | 0.0 |
| 5 | 0 | 141 | 463 | 32419 | 0.56 | 64 | 68 | 8 | 51.7 | 17.5 | 211 | 0.0 |
| 6 | 0 | 73 | 491 | 30053 | 0.61 | 33 | 51 | 7 | 53.7 | 8.2 | 224 | 0.0 |

## Timeline, node q4x1

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 52 | 0.46 | 0 | 2 | 0 | 0.6 | 0.1 | 129 | 556.6 |
| 1 | 14 | 5 | 628 | 6392 | 1.36 | 205 | 48 | 5 | 17.3 | 15.0 | 65 | 11.4 |
| 2 | 24 | 131 | 250 | 22033 | 0.54 | 47 | 84 | 11 | 36.8 | 37.4 | 197 | 1.0 |
| 3 | 14 | 168 | 168 | 32342 | 0.43 | 46 | 90 | 4 | 46.9 | 50.3 | 226 | 1.0 |
| 4 | 7 | 187 | 278 | 34585 | 0.44 | 53 | 84 | 6 | 50.7 | 42.5 | 295 | 1.1 |
| 5 | 0 | 106 | 288 | 29193 | 0.44 | 56 | 61 | 8 | 48.4 | 14.6 | 239 | 0.0 |
| 6 | 0 | 110 | 694 | 29123 | 0.76 | 32 | 43 | 5 | 51.2 | 4.7 | 105 | 0.0 |

## Seed requests (nginx)

3335 requests in the window, 1024 MB. Seed time p50 / p99 / max: 0.0 / 196.0 / 613.0 ms. Size p50 / p99: 178 / 3831 KB. Status: {200: 384, 206: 2951}.
