# Bottleneck: benchmarking/image-plugin/results/20261104-nydus-cpu/e5-nowalk-n24/rep-1/nydus-o-w2

72 first turns. TTFI p50 / p95 / p99 / p100: 1.85 / 2.52 / 2.69 / 2.70 s. Window 2.7 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 8qrl | 24 | 1.84 | 2.48 |
| 9zkl | 24 | 1.84 | 2.70 |
| z3ln | 24 | 1.87 | 2.68 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-022 | 9zkl | 2.70 | 0.06 | 0.41 | 0.21 | 0.92 | 1.60 | 0.01 | 0.04 | 0.03 | 0.46 | 1.02 |
| r1-048 | z3ln | 2.68 | 0.05 | 0.41 | 0.25 | 0.84 | 1.66 | 0.13 | 0.06 | 0.04 | 0.42 | 1.00 |
| r1-044 | z3ln | 2.57 | 0.07 | 0.42 | 0.19 | 0.86 | 1.55 | 0.13 | 0.05 | 0.04 | 0.41 | 0.89 |
| r1-064 | 9zkl | 2.55 | 0.08 | 0.43 | 0.29 | 0.87 | 1.51 | 0.08 | 0.06 | 0.09 | 0.37 | 0.90 |
| r1-025 | z3ln | 2.50 | 0.05 | 0.47 | 0.19 | 0.83 | 1.49 | 0.13 | 0.06 | 0.04 | 0.37 | 0.86 |
| r1-015 | 9zkl | 2.49 | 0.08 | 0.50 | 0.15 | 0.84 | 1.47 | 0.10 | 0.06 | 0.08 | 0.40 | 0.81 |
| r1-049 | 8qrl | 2.48 | 0.07 | 0.42 | 0.27 | 0.92 | 1.37 | 0.06 | 0.05 | 0.05 | 0.33 | 0.86 |
| r1-024 | 8qrl | 2.45 | 0.05 | 0.38 | 0.26 | 0.90 | 1.36 | 0.08 | 0.05 | 0.04 | 0.43 | 0.73 |
| r1-063 | 8qrl | 2.40 | 0.05 | 0.43 | 0.20 | 0.98 | 1.24 | 0.02 | 0.03 | 0.03 | 0.40 | 0.72 |
| r1-016 | 9zkl | 2.35 | 0.06 | 0.45 | 0.17 | 0.86 | 1.33 | 0.09 | 0.04 | 0.03 | 0.40 | 0.71 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 8qrl | 108.8 | 741.8 | 20051.3 | 2.5 | 12.5 | 172.2 | 0.9 | 45.7 | 53.9 | 5.5 | 33.0 | 36.6 | 0.0 | 145.9 | 5.9 | nydusd-amp/14616/state_machine |
| 9zkl | 108.0 | 728.4 | 14998.1 | 2.4 | 12.6 | 110.1 | 0.9 | 47.6 | 54.6 | 10.2 | 37.8 | 28.2 | 0.0 | 537.9 | 0.0 | containerd-nydu/16264/containerd-nydu |
| z3ln | 184.8 | 670.2 | 21946.7 | 1.4 | 10.8 | 164.5 | 0.0 | 47.4 | 52.3 | 4.7 | 29.1 | 37.7 | 0.0 | 341.3 | 0.2 | containerd-nydu/14618/containerd-nydu |

## Top processes in the window, cores per node

- 8qrl: nydusd-amp 1.05, nydus-dragonfly 0.73, ateworker-gviso 0.50, objectstore-plu 0.49, containerd-nydu 0.19, fluent-bit 0.12, atelet 0.07, kworker 0.07
- 9zkl: nydusd-amp 0.95, nydus-dragonfly 0.85, objectstore-plu 0.60, ateworker-gviso 0.58, containerd-nydu 0.25, fluent-bit 0.15, kworker 0.08, atelet 0.08
- z3ln: nydusd-amp 1.44, nydus-dragonfly 0.91, objectstore-plu 0.61, ateworker-gviso 0.53, containerd-nydu 0.26, fluent-bit 0.15, atelet 0.09, kworker 0.08

## Timeline, node 8qrl

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 63 | 0.52 | 0 | 6 | 3 | 0.7 | 0.6 | 100 | 377.2 |
| 1 | 24 | 0 | 742 | 4933 | 2.49 | 172 | 54 | 2 | 14.5 | 20.2 | 119 | 1.0 |
| 2 | 9 | 109 | 144 | 7651 | 0.55 | 51 | 51 | 2 | 15.9 | 23.0 | 245 | 1.0 |
| 3 | 0 | 101 | 306 | 20051 | 0.49 | 52 | 50 | 5 | 36.6 | 33.0 | 296 | 0.1 |
| 4 | 0 | 57 | 230 | 25175 | 0.43 | 44 | 29 | 6 | 49.7 | 2.8 | 180 | 0.2 |

## Timeline, node 9zkl

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 318 | 3231 | 1.08 | 110 | 26 | 4 | 4.2 | 7.6 | 100 | 242.4 |
| 1 | 24 | 59 | 728 | 5122 | 2.43 | 54 | 44 | 10 | 11.0 | 17.1 | 106 | 2.0 |
| 2 | 9 | 108 | 265 | 14998 | 0.49 | 53 | 55 | 1 | 28.2 | 37.8 | 256 | 1.0 |
| 3 | 0 | 54 | 389 | 23303 | 0.52 | 52 | 35 | 5 | 47.8 | 10.9 | 259 | 0.0 |

## Timeline, node z3ln

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 189 | 777 | 1.33 | 0 | 3 | 1 | 1.7 | 0.1 | 100 | 376.3 |
| 1 | 24 | 0 | 670 | 4752 | 1.43 | 164 | 38 | 2 | 9.9 | 13.5 | 137 | 1.0 |
| 2 | 8 | 91 | 345 | 6323 | 0.68 | 37 | 52 | 5 | 12.4 | 18.9 | 354 | 0.0 |
| 3 | 0 | 185 | 493 | 21947 | 0.48 | 69 | 52 | 5 | 37.7 | 29.1 | 267 | 0.0 |
| 4 | 0 | 79 | 463 | 22790 | 0.48 | 47 | 31 | 5 | 44.2 | 5.1 | 154 | 0.0 |

## Seed requests (nginx)

2143 requests in the window, 696 MB. Seed time p50 / p99 / max: 0.0 / 183.6 / 401.0 ms. Size p50 / p99: 178 / 3850 KB. Status: {200: 288, 206: 1855}.
