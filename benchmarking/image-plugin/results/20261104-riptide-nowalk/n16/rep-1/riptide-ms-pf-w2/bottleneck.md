# Bottleneck: benchmarking/image-plugin/results/20261104-riptide-nowalk/n16/rep-1/riptide-ms-pf-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 2.34 / 3.04 / 3.30 / 3.51 s. Window 3.5 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 3m8c | 16 | 2.70 | 3.51 |
| 4njf | 15 | 2.17 | 2.81 |
| blmt | 17 | 2.34 | 3.07 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-024 | 3m8c | 3.51 | 0.05 | 0.35 | 0.18 | 3.20 | 0.18 | 0.00 | 0.03 | 0.01 | 0.07 | 0.03 |
| r1-025 | blmt | 3.07 | 0.05 | 0.36 | 0.22 | 2.61 | 0.33 | 0.00 | 0.04 | 0.02 | 0.08 | 0.12 |
| r1-044 | blmt | 3.06 | 0.05 | 0.37 | 0.16 | 2.65 | 0.28 | 0.00 | 0.05 | 0.01 | 0.06 | 0.10 |
| r1-028 | blmt | 3.01 | 0.05 | 0.36 | 0.29 | 2.68 | 0.18 | 0.00 | 0.03 | 0.02 | 0.06 | 0.04 |
| r1-015 | 3m8c | 2.98 | 0.05 | 0.35 | 0.17 | 2.57 | 0.29 | 0.02 | 0.02 | 0.02 | 0.08 | 0.10 |
| r1-035 | blmt | 2.91 | 0.05 | 0.33 | 0.19 | 2.60 | 0.19 | 0.00 | 0.04 | 0.01 | 0.09 | 0.01 |
| r1-008 | 3m8c | 2.91 | 0.05 | 0.53 | 0.13 | 2.52 | 0.25 | 0.01 | 0.08 | 0.01 | 0.10 | 0.02 |
| r1-009 | blmt | 2.89 | 0.05 | 0.36 | 0.19 | 2.58 | 0.18 | 0.00 | 0.03 | 0.01 | 0.09 | 0.00 |
| r1-043 | blmt | 2.88 | 0.05 | 0.38 | 0.17 | 2.54 | 0.21 | 0.01 | 0.03 | 0.01 | 0.09 | 0.03 |
| r1-042 | 3m8c | 2.86 | 0.05 | 0.36 | 0.16 | 2.47 | 0.26 | 0.01 | 0.05 | 0.01 | 0.10 | 0.00 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 3m8c | 3.7 | 532.3 | 3460.0 | 1.7 | 5.8 | 131.4 | 2.9 | 62.7 | 64.1 | 37.1 | 18.1 | 14.8 | 0.1 | 65.8 | 4.6 | gcfsd-v2/11665/gcfsd-v2 |
| 4njf | 4.1 | 418.4 | 4068.5 | 1.4 | 4.3 | 119.9 | 3.8 | 47.3 | 52.7 | 34.7 | 15.5 | 23.7 | 0.1 | 14.8 | 2.1 | gcfsd-v2/8479/gcfsd-v2 |
| blmt | 3.9 | 610.5 | 4343.4 | 1.6 | 5.9 | 178.9 | 3.7 | 62.3 | 66.7 | 44.3 | 17.6 | 15.8 | 0.0 | 17.2 | 0.4 | gcfsd-v2/8507/gcfsd-v2 |

## Top processes in the window, cores per node

- 3m8c: gcfsd-v2 4.63, objectstore-plu 0.24, ateworker-gviso 0.15, fluent-bit 0.09, containerd-gcfs 0.07, kubelet 0.07, containerd-shim 0.06, riptide 0.06
- 4njf: gcfsd-v2 3.90, objectstore-plu 0.22, ateworker-gviso 0.16, fluent-bit 0.10, kubelet 0.08, containerd-gcfs 0.07, cilium-agent 0.06, containerd 0.06
- blmt: gcfsd-v2 5.09, objectstore-plu 0.33, ateworker-gviso 0.20, kubelet 0.14, fluent-bit 0.12, containerd-gcfs 0.10, riptide 0.08, cilium-agent 0.07

## Timeline, node 3m8c

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 4 | 173 | 823 | 1.12 | 0 | 5 | 2 | 1.4 | 0.5 | 0 | — |
| 1 | 4 | 2 | 448 | 2559 | 1.70 | 131 | 28 | 18 | 3.0 | 2.9 | 0 | — |
| 2 | 11 | 1 | 532 | 3460 | 1.66 | 95 | 42 | 35 | 4.2 | 5.4 | 0 | — |
| 3 | 1 | 1 | 269 | 2696 | 0.94 | 25 | 62 | 37 | 5.1 | 18.1 | 0 | — |
| 4 | 0 | 0 | 183 | 904 | 1.24 | 49 | 64 | 37 | 14.8 | 15.0 | 0 | — |
| 5 | 0 | 0 | 191 | 831 | 1.35 | 20 | 49 | 30 | 20.8 | 8.6 | 0 | — |

## Timeline, node 4njf

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 4 | 164 | 890 | 1.05 | 0 | 6 | 3 | 1.2 | 0.7 | 0 | — |
| 1 | 3 | 0 | 418 | 2675 | 1.45 | 120 | 30 | 15 | 2.6 | 1.7 | 0 | — |
| 2 | 13 | 0 | 394 | 4069 | 1.06 | 82 | 52 | 32 | 7.5 | 15.5 | 0 | — |
| 3 | 0 | 1 | 176 | 1081 | 1.08 | 26 | 53 | 30 | 22.9 | 12.2 | 0 | — |
| 4 | 0 | 0 | 191 | 873 | 1.40 | 45 | 50 | 35 | 23.7 | 10.1 | 0 | — |
| 5 | 0 | 0 | 191 | 856 | 1.33 | 25 | 37 | 21 | 27.9 | 4.4 | 0 | — |

## Timeline, node blmt

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 219 | 919 | 1.22 | 1 | 7 | 1 | 1.0 | 1.0 | 0 | — |
| 1 | 3 | 4 | 611 | 3667 | 1.57 | 179 | 43 | 29 | 4.7 | 3.5 | 0 | — |
| 2 | 15 | 2 | 521 | 4343 | 1.35 | 86 | 65 | 44 | 5.1 | 15.2 | 0 | — |
| 3 | 2 | 1 | 237 | 1830 | 0.92 | 52 | 67 | 34 | 15.8 | 17.6 | 0 | — |
| 4 | 0 | 0 | 230 | 1011 | 1.35 | 25 | 61 | 36 | 18.5 | 12.7 | 0 | — |

## Seed requests (nginx)

No seed access log.
