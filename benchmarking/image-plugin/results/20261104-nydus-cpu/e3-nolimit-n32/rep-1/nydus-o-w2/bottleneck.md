# Bottleneck: benchmarking/image-plugin/results/20261104-nydus-cpu/e3-nolimit-n32/rep-1/nydus-o-w2

96 first turns. TTFI p50 / p95 / p99 / p100: 2.31 / 3.45 / 4.13 / 4.17 s. Window 4.2 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 65zb | 32 | 2.22 | 3.80 |
| k3z3 | 32 | 2.47 | 4.12 |
| w642 | 32 | 2.30 | 4.17 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-063 | w642 | 4.17 | 0.06 | 0.34 | 0.17 | 1.28 | 2.70 | 0.07 | 0.06 | 0.06 | 0.48 | 2.01 |
| r1-024 | k3z3 | 4.12 | 0.09 | 0.37 | 0.21 | 1.20 | 2.75 | 0.05 | 0.07 | 0.10 | 0.47 | 2.02 |
| r1-074 | 65zb | 3.80 | 0.09 | 0.33 | 0.17 | 1.12 | 2.42 | 0.03 | 0.14 | 0.21 | 0.24 | 1.76 |
| r1-006 | w642 | 3.66 | 0.06 | 0.40 | 0.14 | 1.18 | 2.32 | 0.19 | 0.04 | 0.06 | 0.47 | 1.50 |
| r1-094 | k3z3 | 3.51 | 0.06 | 0.40 | 0.21 | 1.27 | 2.10 | 0.03 | 0.08 | 0.06 | 0.46 | 1.41 |
| r1-016 | w642 | 3.44 | 0.09 | 0.31 | 0.26 | 1.08 | 2.11 | 0.20 | 0.08 | 0.04 | 0.46 | 1.30 |
| r1-048 | 65zb | 3.41 | 0.10 | 0.37 | 0.16 | 1.01 | 2.22 | 0.22 | 0.12 | 0.22 | 0.25 | 1.37 |
| r1-062 | k3z3 | 3.37 | 0.05 | 0.39 | 0.18 | 1.23 | 1.95 | 0.03 | 0.07 | 0.08 | 0.46 | 1.27 |
| r1-044 | w642 | 3.32 | 0.06 | 0.39 | 0.15 | 1.16 | 2.00 | 0.20 | 0.14 | 0.43 | 0.07 | 1.13 |
| r1-079 | k3z3 | 3.32 | 0.05 | 0.40 | 0.17 | 0.90 | 2.28 | 0.38 | 0.10 | 0.08 | 0.08 | 1.21 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 65zb | 140.4 | 893.8 | 26841.1 | 2.3 | 12.8 | 223.8 | 5.4 | 55.9 | 71.3 | 9.8 | 38.4 | 45.4 | 0.0 | 467.8 | 0.2 | nydusd-amp/17751/state_machine |
| k3z3 | 112.4 | 665.3 | 25352.2 | 2.1 | 11.1 | 214.4 | 2.7 | 57.1 | 62.5 | 9.0 | 44.9 | 45.1 | 0.0 | 442.3 | 0.4 | nydusd-amp/17433/state_machine |
| w642 | 156.2 | 971.9 | 24073.0 | 1.9 | 12.4 | 211.2 | 10.8 | 57.2 | 73.9 | 10.2 | 31.6 | 45.4 | 0.0 | 458.8 | 0.7 | containerd-nydu/17605/containerd-nydu |

## Top processes in the window, cores per node

- 65zb: nydusd-amp 1.80, nydus-dragonfly 0.75, ateworker-gviso 0.72, objectstore-plu 0.48, containerd-nydu 0.23, fluent-bit 0.14, kworker 0.08, atelet 0.08
- k3z3: nydusd-amp 1.61, nydus-dragonfly 0.71, ateworker-gviso 0.51, objectstore-plu 0.49, containerd-nydu 0.19, fluent-bit 0.13, atelet 0.08, containerd 0.07
- w642: nydusd-amp 1.87, nydus-dragonfly 0.75, ateworker-gviso 0.62, objectstore-plu 0.48, containerd-nydu 0.20, fluent-bit 0.15, kworker 0.09, atelet 0.08

## Timeline, node 65zb

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 174 | 1389 | 0.78 | 1 | 8 | 2 | 2.0 | 0.9 | 126 | 308.8 |
| 1 | 13 | 0 | 894 | 5568 | 2.27 | 224 | 71 | 8 | 13.5 | 19.9 | 59 | 1.0 |
| 2 | 25 | 111 | 256 | 8150 | 0.59 | 42 | 54 | 2 | 18.9 | 31.6 | 277 | 1.0 |
| 3 | 6 | 140 | 428 | 19552 | 0.47 | 74 | 59 | 5 | 35.6 | 38.4 | 332 | 2.7 |
| 4 | 0 | 78 | 466 | 26841 | 0.44 | 55 | 43 | 10 | 45.4 | 10.6 | 269 | 0.0 |
| 5 | 0 | 101 | 359 | 26482 | 0.46 | 41 | 34 | 6 | 44.4 | 3.7 | 205 | 0.0 |

## Timeline, node k3z3

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 6 | 280 | 0.46 | 1 | 7 | 1 | 0.9 | 0.8 | 130 | 259.7 |
| 1 | 4 | 0 | 665 | 5003 | 2.15 | 214 | 58 | 2 | 8.9 | 14.7 | 32 | 0.7 |
| 2 | 30 | 90 | 134 | 4167 | 0.67 | 21 | 58 | 2 | 8.6 | 28.6 | 176 | 1.2 |
| 3 | 6 | 98 | 200 | 14807 | 0.47 | 62 | 62 | 2 | 26.5 | 44.9 | 301 | 2.0 |
| 4 | 1 | 112 | 279 | 25352 | 0.43 | 65 | 46 | 9 | 45.1 | 10.3 | 321 | 0.0 |
| 5 | 0 | 84 | 467 | 29156 | 0.54 | 37 | 39 | 7 | 49.2 | 4.2 | 225 | 0.0 |

## Timeline, node w642

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 16 | 1200 | 0.41 | 5 | 12 | 1 | 2.2 | 1.9 | 129 | 308.8 |
| 1 | 4 | 0 | 972 | 6319 | 1.93 | 211 | 74 | 10 | 12.0 | 20.6 | 66 | 10.9 |
| 2 | 28 | 134 | 170 | 9303 | 0.56 | 67 | 58 | 2 | 19.7 | 25.4 | 308 | 1.0 |
| 3 | 6 | 156 | 290 | 22723 | 0.45 | 65 | 62 | 6 | 37.9 | 31.6 | 389 | 1.0 |
| 4 | 1 | 73 | 328 | 24073 | 0.46 | 51 | 43 | 6 | 45.4 | 10.3 | 226 | 0.0 |
| 6 | 0 | 76 | 439 | 21314 | 0.64 | 31 | 30 | 4 | 40.5 | 4.4 | 14 | 0.0 |

## Seed requests (nginx)

3253 requests in the window, 1006 MB. Seed time p50 / p99 / max: 0.0 / 184.9 / 480.0 ms. Size p50 / p99: 178 / 3848 KB. Status: {200: 380, 206: 2873}.
