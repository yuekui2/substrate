# Bottleneck: benchmarking/image-plugin/results/20261104-nydus-cpu/e4-none-n32/rep-1/nydus-none-w2

96 first turns. TTFI p50 / p95 / p99 / p100: 2.32 / 3.37 / 3.75 / 3.81 s. Window 3.8 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 4j7d | 32 | 2.38 | 3.30 |
| hhsn | 32 | 2.14 | 3.06 |
| wz4x | 32 | 2.59 | 3.81 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-075 | wz4x | 3.81 | 0.07 | 0.17 | 0.16 | 1.22 | 2.24 | 0.05 | 0.10 | 0.38 | 0.11 | 1.57 |
| r1-091 | wz4x | 3.74 | 0.07 | 0.27 | 0.24 | 1.39 | 2.18 | 0.06 | 0.12 | 0.36 | 0.14 | 1.48 |
| r1-094 | wz4x | 3.70 | 0.06 | 0.26 | 0.18 | 1.37 | 2.14 | 0.05 | 0.14 | 0.35 | 0.10 | 1.40 |
| r1-025 | wz4x | 3.59 | 0.06 | 0.32 | 0.20 | 1.38 | 2.02 | 0.04 | 0.05 | 0.08 | 0.39 | 1.43 |
| r1-074 | wz4x | 3.52 | 0.06 | 0.25 | 0.20 | 1.37 | 1.96 | 0.02 | 0.05 | 0.02 | 0.46 | 1.35 |
| r1-045 | wz4x | 3.32 | 0.07 | 0.29 | 0.25 | 1.36 | 1.77 | 0.03 | 0.07 | 0.07 | 0.50 | 1.08 |
| r1-048 | wz4x | 3.31 | 0.05 | 0.25 | 0.20 | 1.38 | 1.75 | 0.07 | 0.14 | 0.33 | 0.12 | 1.06 |
| r1-092 | 4j7d | 3.30 | 0.05 | 0.32 | 0.16 | 1.09 | 2.03 | 0.19 | 0.06 | 0.12 | 0.46 | 1.17 |
| r1-059 | 4j7d | 3.27 | 0.06 | 0.31 | 0.16 | 1.06 | 1.98 | 0.19 | 0.18 | 0.03 | 0.42 | 1.11 |
| r1-024 | 4j7d | 3.24 | 0.06 | 0.36 | 0.12 | 1.11 | 1.96 | 0.19 | 0.06 | 0.10 | 0.48 | 1.10 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 4j7d | 194.6 | 544.9 | 30260.0 | 2.2 | 14.0 | 217.0 | 22.2 | 55.8 | 59.8 | 19.0 | 40.9 | 47.2 | 0.0 | 445.9 | 0.3 | containerd-nydu/18834/containerd-nydu |
| hhsn | 156.3 | 706.6 | 36629.9 | 2.3 | 16.4 | 234.5 | 0.0 | 60.6 | 65.0 | 3.6 | 45.6 | 53.6 | 0.0 | 670.2 | 0.6 | containerd/9567/containerd |
| wz4x | 176.1 | 869.3 | 21917.8 | 1.9 | 10.7 | 263.5 | 0.9 | 56.5 | 62.6 | 45.6 | 39.3 | 37.3 | 0.0 | 382.4 | 0.0 | containerd-nydu/17697/containerd-nydu |

## Top processes in the window, cores per node

- 4j7d: nydusd-amp 1.34, nydus-dragonfly 0.96, ateworker-gviso 0.87, objectstore-plu 0.62, containerd-nydu 0.24, fluent-bit 0.19, kubelet 0.10, atelet 0.10
- hhsn: nydusd-amp 1.22, nydus-dragonfly 0.68, ateworker-gviso 0.59, objectstore-plu 0.49, containerd-nydu 0.20, fluent-bit 0.14, kworker 0.09, atelet 0.08
- wz4x: nydusd-amp 1.08, nydus-dragonfly 0.97, objectstore-plu 0.60, ateworker-gviso 0.49, containerd-nydu 0.25, fluent-bit 0.15, atelet 0.10, cilium-agent 0.08

## Timeline, node 4j7d

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 52 | 0.41 | 0 | 4 | 0 | 0.3 | 0.1 | 131 | 316.7 |
| 1 | 7 | 0 | 545 | 4505 | 2.24 | 210 | 51 | 2 | 8.5 | 9.8 | 65 | 46.9 |
| 2 | 29 | 24 | 249 | 3017 | 0.94 | 37 | 50 | 19 | 6.4 | 25.6 | 281 | 1.4 |
| 3 | 6 | 195 | 252 | 18738 | 0.46 | 217 | 60 | 4 | 31.1 | 40.9 | 369 | 1.0 |
| 4 | 0 | 125 | 385 | 30260 | 0.46 | 208 | 47 | 4 | 47.2 | 24.3 | 295 | 0.0 |
| 5 | 0 | 90 | 454 | 27339 | 0.56 | 113 | 31 | 5 | 44.9 | 4.3 | 107 | 0.0 |

## Timeline, node hhsn

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 8 | 578 | 0.43 | 1 | 7 | 2 | 1.4 | 0.9 | 131 | 421.1 |
| 1 | 18 | 0 | 707 | 5173 | 2.29 | 211 | 63 | 3 | 22.2 | 12.9 | 87 | 3.8 |
| 2 | 21 | 137 | 150 | 9291 | 0.53 | 134 | 59 | 3 | 18.0 | 35.9 | 305 | 5.0 |
| 3 | 1 | 156 | 585 | 27657 | 0.59 | 235 | 65 | 4 | 41.4 | 45.6 | 423 | 0.0 |
| 4 | 0 | 96 | 353 | 36630 | 0.41 | 184 | 36 | 4 | 53.6 | 7.6 | 288 | 0.0 |

## Timeline, node wz4x

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 16 | 803 | 0.47 | 22 | 15 | 2 | 2.0 | 3.0 | 128 | 303.7 |
| 1 | 2 | 1 | 869 | 5419 | 1.91 | 202 | 54 | 46 | 17.4 | 13.1 | 27 | 0.7 |
| 2 | 31 | 92 | 87 | 5926 | 0.54 | 72 | 44 | 2 | 12.9 | 26.6 | 193 | 8.1 |
| 3 | 9 | 176 | 312 | 21918 | 0.48 | 264 | 63 | 4 | 37.3 | 39.3 | 460 | 1.0 |
| 4 | 0 | 111 | 380 | 36841 | 0.44 | 203 | 39 | 4 | 55.4 | 9.4 | 323 | 0.0 |

## Seed requests (nginx)

3354 requests in the window, 2106 MB. Seed time p50 / p99 / max: 0.0 / 117.9 / 461.0 ms. Size p50 / p99: 597 / 3713 KB. Status: {200: 384, 206: 2970}.
