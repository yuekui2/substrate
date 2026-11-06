# Bottleneck: benchmarking/image-plugin/results/20261103-confirm-sweep/set-a/n24/rep-1/nydus-pf-w2

72 first turns. TTFI p50 / p95 / p99 / p100: 1.96 / 3.23 / 4.89 / 5.26 s. Window 5.3 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 2j1d | 20 | 1.57 | 2.49 |
| dn7v | 31 | 2.21 | 5.26 |
| vd4g | 21 | 1.66 | 2.56 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-066 | dn7v | 5.26 | 0.09 | 0.29 | 0.17 | 1.06 | 4.01 | 0.13 | 0.15 | 0.18 | 0.22 | 3.27 |
| r1-063 | dn7v | 4.75 | 0.04 | 0.38 | 0.18 | 1.11 | 3.52 | 0.15 | 0.14 | 0.21 | 0.24 | 2.77 |
| r1-044 | dn7v | 3.44 | 0.05 | 0.28 | 0.11 | 1.08 | 2.22 | 0.15 | 0.13 | 0.20 | 0.20 | 1.47 |
| r1-025 | dn7v | 3.31 | 0.07 | 0.24 | 0.13 | 1.07 | 2.05 | 0.11 | 0.11 | 0.03 | 0.25 | 1.32 |
| r1-018 | dn7v | 3.17 | 0.05 | 0.26 | 0.11 | 1.04 | 1.95 | 0.15 | 0.13 | 0.26 | 0.20 | 1.19 |
| r1-064 | dn7v | 2.86 | 0.05 | 0.29 | 0.19 | 0.81 | 1.87 | 0.34 | 0.06 | 0.08 | 0.29 | 1.06 |
| r1-062 | dn7v | 2.70 | 0.04 | 0.33 | 0.16 | 0.80 | 1.73 | 0.37 | 0.05 | 0.06 | 0.31 | 0.90 |
| r1-049 | dn7v | 2.65 | 0.05 | 0.35 | 0.17 | 0.83 | 1.70 | 0.42 | 0.11 | 0.25 | 0.20 | 0.67 |
| r1-024 | vd4g | 2.56 | 0.05 | 0.30 | 0.12 | 0.84 | 1.54 | 0.02 | 0.05 | 0.06 | 0.32 | 1.04 |
| r1-001 | dn7v | 2.54 | 0.04 | 0.25 | 0.18 | 0.81 | 1.55 | 0.39 | 0.14 | 0.23 | 0.22 | 0.55 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2j1d | 166.1 | 639.4 | 32542.1 | 1.2 | 12.7 | 136.4 | 1.9 | 71.8 | 82.4 | 5.6 | 40.1 | 46.3 | 0.0 | 288.6 | 13.6 | nydusd-amp/14109/state_machine |
| dn7v | 157.1 | 601.2 | 36114.5 | 2.0 | 13.7 | 180.7 | 11.6 | 79.9 | 90.7 | 25.3 | 66.2 | 48.7 | 0.0 | 446.9 | 0.2 | containerd-nydu/16416/containerd-nydu |
| vd4g | 164.3 | 452.8 | 29756.9 | 0.7 | 11.6 | 139.0 | 6.4 | 69.5 | 81.7 | 5.5 | 38.9 | 42.9 | 0.0 | 294.4 | 0.3 | nydusd-amp/14072/state_machine |

## Top processes in the window, cores per node

- 2j1d: nydus-dragonfly 3.13, nydusd-amp 1.04, objectstore-plu 0.26, ateworker-gviso 0.24, containerd-nydu 0.11, fluent-bit 0.10, kubelet 0.06, containerd 0.05
- dn7v: nydus-dragonfly 5.39, nydusd-amp 1.90, ateworker-gviso 0.47, objectstore-plu 0.47, containerd-nydu 0.20, fluent-bit 0.14, kworker 0.09, atelet 0.07
- vd4g: nydus-dragonfly 3.68, nydusd-amp 1.28, objectstore-plu 0.33, ateworker-gviso 0.31, containerd-nydu 0.12, fluent-bit 0.12, kubelet 0.07, atelet 0.06

## Timeline, node 2j1d

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 1 | 173 | 738 | 1.20 | 0 | 3 | 1 | 1.7 | 0.1 | 92 | 302.8 |
| 1 | 20 | 30 | 639 | 11822 | 0.73 | 136 | 59 | 3 | 18.2 | 20.1 | 176 | 2.2 |
| 2 | 4 | 166 | 313 | 24654 | 0.40 | 57 | 82 | 1 | 34.4 | 40.1 | 224 | 0.0 |
| 3 | 0 | 143 | 263 | 32542 | 0.39 | 48 | 69 | 5 | 46.3 | 27.2 | 241 | 0.0 |
| 4 | 0 | 77 | 205 | 24102 | 0.38 | 31 | 35 | 6 | 41.4 | 3.3 | 101 | 0.0 |
| 5 | 0 | 29 | 148 | 7576 | 0.50 | 7 | 17 | 2 | 15.4 | 3.7 | 0 | — |
| 6 | 0 | 0 | 228 | 1118 | 6.79 | 0 | 4 | 0 | 1.9 | 0.1 | 0 | — |

## Timeline, node dn7v

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 601 | 4564 | 2.00 | 181 | 41 | 4 | 6.1 | 9.2 | 126 | 329.8 |
| 1 | 30 | 114 | 385 | 21443 | 0.49 | 47 | 84 | 25 | 30.1 | 33.8 | 72 | 1.3 |
| 2 | 18 | 152 | 102 | 20670 | 0.38 | 34 | 91 | 1 | 27.9 | 66.2 | 148 | 3.5 |
| 4 | 2 | 157 | 176 | 36114 | 0.38 | 50 | 86 | 5 | 48.7 | 38.0 | 272 | 0.0 |
| 5 | 1 | 115 | 265 | 29679 | 0.40 | 53 | 57 | 9 | 47.1 | 9.1 | 202 | 0.0 |
| 6 | 0 | 123 | 453 | 32743 | 0.55 | 39 | 46 | 7 | 54.0 | 4.9 | 214 | 0.0 |

## Timeline, node vd4g

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 23 | 453 | 9098 | 0.75 | 139 | 42 | 2 | 15.2 | 10.8 | 88 | 202.9 |
| 2 | 6 | 149 | 122 | 21121 | 0.39 | 37 | 82 | 1 | 30.0 | 38.9 | 249 | 1.5 |
| 3 | 0 | 164 | 226 | 29757 | 0.38 | 52 | 75 | 4 | 42.6 | 38.3 | 269 | 0.0 |
| 4 | 0 | 78 | 231 | 25955 | 0.38 | 42 | 39 | 6 | 42.9 | 6.5 | 129 | 0.0 |
| 5 | 0 | 61 | 167 | 14415 | 0.41 | 17 | 23 | 4 | 25.5 | 5.5 | 12 | 0.0 |
| 6 | 0 | 0 | 282 | 1407 | 7.11 | 0 | 6 | 1 | 3.2 | 0.4 | 0 | — |

## Seed requests (nginx)

2834 requests in the window, 812 MB. Seed time p50 / p99 / max: 0.0 / 148.4 / 392.0 ms. Size p50 / p99: 169 / 3732 KB. Status: {200: 288, 206: 2546}.
