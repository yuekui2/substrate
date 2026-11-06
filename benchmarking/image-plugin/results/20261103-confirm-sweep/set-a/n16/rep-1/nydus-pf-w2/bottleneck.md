# Bottleneck: benchmarking/image-plugin/results/20261103-confirm-sweep/set-a/n16/rep-1/nydus-pf-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 1.34 / 1.71 / 1.92 / 1.95 s. Window 2.0 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 9c66 | 18 | 1.43 | 1.95 |
| kq7p | 14 | 1.17 | 1.48 |
| vn02 | 16 | 1.28 | 1.55 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-024 | 9c66 | 1.95 | 0.02 | 0.35 | 0.19 | 0.73 | 1.11 | 0.00 | 0.03 | 0.01 | 0.33 | 0.66 |
| r1-044 | 9c66 | 1.88 | 0.05 | 0.33 | 0.21 | 0.72 | 1.05 | 0.03 | 0.04 | 0.06 | 0.20 | 0.64 |
| r1-015 | 9c66 | 1.71 | 0.05 | 0.37 | 0.15 | 0.69 | 0.92 | 0.06 | 0.12 | 0.21 | 0.08 | 0.42 |
| r1-010 | 9c66 | 1.70 | 0.03 | 0.33 | 0.19 | 0.68 | 0.91 | 0.06 | 0.04 | 0.08 | 0.25 | 0.46 |
| r1-008 | 9c66 | 1.58 | 0.03 | 0.31 | 0.20 | 0.60 | 0.87 | 0.14 | 0.05 | 0.03 | 0.29 | 0.34 |
| r1-018 | vn02 | 1.55 | 0.03 | 0.31 | 0.22 | 0.64 | 0.83 | 0.05 | 0.04 | 0.01 | 0.24 | 0.42 |
| r1-016 | 9c66 | 1.52 | 0.02 | 0.30 | 0.22 | 0.54 | 0.85 | 0.19 | 0.06 | 0.03 | 0.25 | 0.28 |
| r1-025 | kq7p | 1.48 | 0.02 | 0.27 | 0.17 | 0.68 | 0.69 | 0.08 | 0.12 | 0.01 | 0.08 | 0.35 |
| r1-043 | 9c66 | 1.46 | 0.03 | 0.36 | 0.19 | 0.71 | 0.67 | 0.06 | 0.06 | 0.04 | 0.27 | 0.22 |
| r1-047 | vn02 | 1.46 | 0.02 | 0.29 | 0.28 | 0.50 | 0.78 | 0.09 | 0.04 | 0.01 | 0.26 | 0.32 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 9c66 | 186.8 | 484.2 | 27238.9 | 0.8 | 17.9 | 131.5 | 0.9 | 70.2 | 79.8 | 5.7 | 38.0 | 42.2 | 0.0 | 593.2 | 2.6 | containerd/9118/containerd |
| kq7p | 149.0 | 426.5 | 24774.4 | 0.9 | 11.0 | 95.6 | 2.8 | 63.9 | 74.5 | 10.5 | 31.7 | 37.8 | 0.0 | 203.5 | 20.0 | containerd-nydu/14457/containerd-nydu |
| vn02 | 147.4 | 347.9 | 21323.8 | 1.1 | 8.1 | 98.0 | 0.0 | 66.7 | 81.4 | 3.8 | 34.4 | 31.2 | 0.0 | 210.1 | 0.0 | containerd-nydu/14182/containerd-nydu |

## Top processes in the window, cores per node

- 9c66: nydus-dragonfly 4.04, nydusd-amp 0.81, objectstore-plu 0.46, ateworker-gviso 0.39, containerd-nydu 0.21, fluent-bit 0.12, kworker 0.09, containerd 0.08
- kq7p: nydus-dragonfly 4.06, objectstore-plu 0.35, ateworker-gviso 0.29, containerd-nydu 0.20, fluent-bit 0.12, cilium-agent 0.07, atelet 0.06, kubelet 0.06
- vn02: nydus-dragonfly 4.76, nydusd-amp 0.82, ateworker-gviso 0.77, objectstore-plu 0.62, containerd-nydu 0.27, fluent-bit 0.15, cilium-agent 0.09, atelet 0.09

## Timeline, node 9c66

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 3 | 109 | 0.45 | 0 | 5 | 1 | 0.4 | 0.3 | 89 | 312.6 |
| 1 | 18 | 39 | 445 | 12523 | 0.76 | 132 | 68 | 6 | 22.4 | 23.3 | 219 | 1.0 |
| 2 | 0 | 187 | 484 | 27239 | 0.66 | 58 | 80 | 2 | 42.2 | 38.0 | 239 | 0.0 |
| 3 | 0 | 135 | 491 | 28871 | 0.51 | 44 | 50 | 5 | 45.6 | 13.5 | 149 | 0.0 |

## Timeline, node kq7p

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 7 | 15 | 427 | 6972 | 0.88 | 96 | 37 | 11 | 10.7 | 8.7 | 84 | 159.0 |
| 1 | 14 | 149 | 253 | 23159 | 0.47 | 44 | 75 | 2 | 33.2 | 31.7 | 170 | 1.0 |
| 2 | 0 | 102 | 204 | 24774 | 0.37 | 39 | 45 | 4 | 37.8 | 14.6 | 225 | 0.0 |

## Timeline, node vn02

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 15 | 348 | 6301 | 1.06 | 98 | 41 | 4 | 11.9 | 11.3 | 87 | 130.4 |
| 1 | 16 | 147 | 110 | 21324 | 0.38 | 37 | 81 | 1 | 31.2 | 34.4 | 165 | 1.0 |
| 2 | 0 | 102 | 205 | 23683 | 0.39 | 38 | 46 | 3 | 36.2 | 24.9 | 210 | 0.0 |

## Seed requests (nginx)

1446 requests in the window, 458 MB. Seed time p50 / p99 / max: 0.0 / 113.0 / 361.0 ms. Size p50 / p99: 169 / 3825 KB. Status: {200: 192, 206: 1254}.
