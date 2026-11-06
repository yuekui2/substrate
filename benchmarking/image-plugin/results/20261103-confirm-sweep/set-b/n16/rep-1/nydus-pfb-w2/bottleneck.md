# Bottleneck: benchmarking/image-plugin/results/20261103-confirm-sweep/set-b/n16/rep-1/nydus-pfb-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 1.37 / 1.75 / 2.06 / 2.12 s. Window 2.1 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 3d5m | 15 | 1.26 | 1.63 |
| 7sq4 | 15 | 1.22 | 1.42 |
| s9rg | 18 | 1.50 | 2.12 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-028 | s9rg | 2.12 | 0.03 | 0.52 | 0.18 | 0.82 | 1.22 | 0.02 | 0.05 | 0.04 | 0.29 | 0.80 |
| r1-017 | s9rg | 2.00 | 0.04 | 0.38 | 0.24 | 0.78 | 1.13 | 0.05 | 0.05 | 0.04 | 0.30 | 0.67 |
| r1-046 | s9rg | 1.78 | 0.04 | 0.36 | 0.26 | 0.73 | 0.96 | 0.09 | 0.05 | 0.05 | 0.10 | 0.39 |
| r1-034 | s9rg | 1.69 | 0.03 | 0.30 | 0.22 | 0.68 | 0.86 | 0.10 | 0.04 | 0.04 | 0.28 | 0.38 |
| r1-009 | s9rg | 1.68 | 0.02 | 0.36 | 0.21 | 0.73 | 0.87 | 0.10 | 0.04 | 0.02 | 0.32 | 0.38 |
| r1-006 | s9rg | 1.64 | 0.02 | 0.29 | 0.19 | 0.66 | 0.83 | 0.10 | 0.05 | 0.04 | 0.27 | 0.32 |
| r1-010 | 3d5m | 1.63 | 0.03 | 0.25 | 0.23 | 0.47 | 0.99 | 0.15 | 0.05 | 0.07 | 0.20 | 0.51 |
| r1-019 | 3d5m | 1.62 | 0.02 | 0.31 | 0.20 | 0.48 | 0.96 | 0.11 | 0.04 | 0.09 | 0.19 | 0.51 |
| r1-033 | s9rg | 1.62 | 0.03 | 0.37 | 0.22 | 0.66 | 0.88 | 0.17 | 0.05 | 0.03 | 0.29 | 0.29 |
| r1-030 | s9rg | 1.59 | 0.04 | 0.32 | 0.24 | 0.66 | 0.77 | 0.10 | 0.05 | 0.05 | 0.28 | 0.28 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 3d5m | 168.0 | 501.1 | 26277.4 | 1.2 | 10.9 | 94.0 | 1.0 | 66.3 | 71.6 | 2.9 | 27.4 | 37.5 | 0.0 | 398.8 | 0.0 | containerd/2792/containerd |
| 7sq4 | 150.5 | 345.2 | 21420.2 | 0.5 | 8.2 | 99.8 | 0.0 | 69.2 | 77.8 | 2.9 | 40.5 | 31.4 | 0.0 | 420.5 | 0.2 | containerd-nydu/14363/containerd-nydu |
| s9rg | 136.9 | 627.8 | 25114.5 | 0.8 | 14.1 | 134.7 | 1.0 | 76.8 | 83.8 | 8.4 | 47.2 | 34.6 | 0.0 | 238.2 | 0.0 | containerd-nydu/14488/containerd-nydu |

## Top processes in the window, cores per node

- 3d5m: nydus-dragonfly 5.55, nydusd-amp 1.13, objectstore-plu 0.58, ateworker-gviso 0.42, containerd-nydu 0.28, fluent-bit 0.15, atelet 0.09, cilium-agent 0.07
- 7sq4: nydus-dragonfly 5.56, nydusd-amp 0.76, objectstore-plu 0.59, ateworker-gviso 0.39, containerd-nydu 0.25, fluent-bit 0.14, cilium-agent 0.10, atelet 0.09
- s9rg: nydus-dragonfly 4.73, nydusd-amp 0.84, objectstore-plu 0.48, ateworker-gviso 0.34, containerd-nydu 0.19, fluent-bit 0.13, kworker 0.08, atelet 0.07

## Timeline, node 3d5m

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 181 | 746 | 1.19 | 0 | 3 | 1 | 1.5 | 0.1 | 90 | 132.8 |
| 1 | 15 | 32 | 501 | 11661 | 0.62 | 94 | 52 | 3 | 17.5 | 15.8 | 214 | 1.0 |
| 2 | 0 | 168 | 352 | 26277 | 0.41 | 55 | 72 | 2 | 37.5 | 27.4 | 253 | 0.0 |
| 3 | 0 | 106 | 416 | 20401 | 0.43 | 39 | 35 | 5 | 33.8 | 14.6 | 120 | 0.0 |

## Timeline, node 7sq4

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 31 | 345 | 10841 | 0.55 | 100 | 50 | 3 | 16.0 | 18.2 | 91 | 126.8 |
| 2 | 0 | 150 | 102 | 21420 | 0.38 | 37 | 78 | 1 | 31.4 | 40.5 | 203 | 0.0 |
| 3 | 0 | 72 | 217 | 24439 | 0.37 | 48 | 43 | 4 | 39.5 | 16.3 | 197 | 0.0 |

## Timeline, node s9rg

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 199 | 1659 | 0.80 | 8 | 10 | 1 | 2.3 | 0.7 | 84 | 224.9 |
| 1 | 18 | 98 | 628 | 16739 | 0.83 | 135 | 73 | 8 | 30.8 | 25.6 | 195 | 4.1 |
| 2 | 1 | 137 | 309 | 25114 | 0.40 | 33 | 84 | 2 | 34.6 | 47.2 | 146 | 0.0 |
| 3 | 0 | 90 | 338 | 24599 | 0.41 | 45 | 43 | 5 | 39.5 | 12.8 | 221 | 0.0 |

## Seed requests (nginx)

1487 requests in the window, 461 MB. Seed time p50 / p99 / max: 0.0 / 118.1 / 332.0 ms. Size p50 / p99: 186 / 3860 KB. Status: {200: 192, 206: 1295}.
