# Bottleneck: benchmarking/image-plugin/results/20261104-nydus-cpu/e5-nowalk-n16/rep-1/nydus-o-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 1.40 / 2.11 / 2.22 / 2.29 s. Window 2.3 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 1h9s | 13 | 1.19 | 1.37 |
| 1zk7 | 14 | 1.28 | 1.51 |
| tzf8 | 21 | 1.59 | 2.29 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-008 | tzf8 | 2.29 | 0.06 | 0.28 | 0.17 | 0.79 | 1.38 | 0.07 | 0.06 | 0.09 | 0.27 | 0.84 |
| r1-044 | tzf8 | 2.15 | 0.06 | 0.28 | 0.18 | 0.79 | 1.24 | 0.07 | 0.06 | 0.10 | 0.28 | 0.70 |
| r1-004 | tzf8 | 2.13 | 0.05 | 0.31 | 0.13 | 0.81 | 1.18 | 0.05 | 0.03 | 0.03 | 0.28 | 0.67 |
| r1-038 | tzf8 | 2.07 | 0.05 | 0.29 | 0.16 | 0.85 | 1.10 | 0.01 | 0.06 | 0.11 | 0.33 | 0.56 |
| r1-041 | tzf8 | 1.91 | 0.06 | 0.30 | 0.18 | 0.62 | 1.15 | 0.24 | 0.04 | 0.01 | 0.30 | 0.44 |
| r1-001 | tzf8 | 1.88 | 0.05 | 0.32 | 0.12 | 0.63 | 1.12 | 0.24 | 0.14 | 0.04 | 0.22 | 0.43 |
| r1-035 | tzf8 | 1.88 | 0.03 | 0.26 | 0.22 | 0.62 | 1.13 | 0.24 | 0.04 | 0.11 | 0.26 | 0.43 |
| r1-040 | tzf8 | 1.70 | 0.04 | 0.28 | 0.17 | 0.61 | 0.97 | 0.25 | 0.05 | 0.11 | 0.26 | 0.24 |
| r1-013 | tzf8 | 1.64 | 0.04 | 0.26 | 0.13 | 0.57 | 0.94 | 0.29 | 0.04 | 0.01 | 0.30 | 0.19 |
| r1-011 | tzf8 | 1.61 | 0.03 | 0.27 | 0.16 | 0.53 | 0.95 | 0.34 | 0.14 | 0.05 | 0.09 | 0.10 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1h9s | 61.4 | 753.3 | 10500.5 | 2.0 | 13.0 | 121.2 | 1.0 | 33.4 | 40.6 | 4.8 | 25.5 | 21.0 | 0.0 | 235.3 | 0.0 | containerd/6465/containerd |
| 1zk7 | 111.0 | 629.6 | 14904.4 | 1.3 | 9.8 | 111.1 | 0.9 | 31.1 | 41.5 | 3.8 | 21.1 | 25.1 | 0.0 | 310.2 | 1.0 | containerd-nydu/14499/containerd-nydu |
| tzf8 | 110.9 | 460.7 | 22386.9 | 1.4 | 11.5 | 133.0 | 2.7 | 38.8 | 48.1 | 4.6 | 22.6 | 37.9 | 0.0 | 492.1 | 5.1 | containerd/2778/containerd |

## Top processes in the window, cores per node

- 1h9s: nydusd-amp 0.56, nydus-dragonfly 0.49, objectstore-plu 0.34, ateworker-gviso 0.22, containerd-nydu 0.15, kworker 0.09, fluent-bit 0.09, containerd 0.07
- 1zk7: nydusd-amp 0.62, nydus-dragonfly 0.49, objectstore-plu 0.38, ateworker-gviso 0.22, containerd-nydu 0.16, fluent-bit 0.09, kubelet 0.07, containerd 0.07
- tzf8: nydusd-amp 1.31, nydus-dragonfly 0.74, objectstore-plu 0.56, ateworker-gviso 0.52, containerd-nydu 0.23, fluent-bit 0.15, kworker 0.09, kubelet 0.08

## Timeline, node 1h9s

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 4 | 0 | 3 | 152 | 0.43 | 0 | 6 | 5 | 0.6 | 0.2 | 83 | 166.8 |
| 1 | 13 | 58 | 753 | 6501 | 1.98 | 121 | 41 | 2 | 14.6 | 9.4 | 155 | 1.0 |
| 2 | 0 | 61 | 118 | 10500 | 0.42 | 24 | 36 | 1 | 21.0 | 25.5 | 187 | 1.1 |
| 3 | 0 | 37 | 182 | 17947 | 0.41 | 35 | 21 | 3 | 35.3 | 1.9 | 124 | 0.0 |

## Timeline, node 1zk7

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 189 | 779 | 1.27 | 0 | 4 | 1 | 1.1 | 0.2 | 80 | 142.3 |
| 1 | 14 | 27 | 630 | 4895 | 1.33 | 111 | 42 | 4 | 11.9 | 9.2 | 223 | 1.0 |
| 2 | 0 | 111 | 489 | 14904 | 0.65 | 43 | 37 | 2 | 25.1 | 21.1 | 229 | 0.0 |
| 3 | 0 | 46 | 385 | 16510 | 0.44 | 34 | 25 | 4 | 29.8 | 4.7 | 96 | 0.0 |

## Timeline, node tzf8

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 461 | 3630 | 1.45 | 133 | 48 | 2 | 7.7 | 12.5 | 97 | 215.6 |
| 2 | 4 | 97 | 124 | 7583 | 0.51 | 46 | 44 | 3 | 18.7 | 22.6 | 264 | 1.0 |
| 3 | 0 | 111 | 428 | 22387 | 0.51 | 51 | 40 | 5 | 37.9 | 18.7 | 209 | 0.0 |
| 4 | 0 | 60 | 240 | 20227 | 0.45 | 28 | 25 | 5 | 39.4 | 3.7 | 99 | 0.0 |

## Seed requests (nginx)

1628 requests in the window, 493 MB. Seed time p50 / p99 / max: 0.0 / 139.2 / 279.0 ms. Size p50 / p99: 178 / 3744 KB. Status: {200: 192, 206: 1436}.
