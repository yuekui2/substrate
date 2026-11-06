# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-repeats/driver/../b-n24-r2/rep-1/nydus-ob-w2

72 first turns. TTFI p50 / p95 / p99 / p100: 1.80 / 2.40 / 2.54 / 2.58 s. Window 2.6 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 8pfm | 24 | 1.82 | 2.58 |
| mc5h | 24 | 1.85 | 2.53 |
| pgdw | 24 | 1.72 | 1.96 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-018 | 8pfm | 2.58 | 0.06 | 0.44 | 0.12 | 0.97 | 1.45 | 0.02 | 0.05 | 0.10 | 0.35 | 0.89 |
| r1-055 | mc5h | 2.53 | 0.07 | 0.43 | 0.13 | 0.91 | 1.45 | 0.09 | 0.04 | 0.11 | 0.33 | 0.85 |
| r1-028 | 8pfm | 2.51 | 0.07 | 0.35 | 0.11 | 0.90 | 1.44 | 0.08 | 0.05 | 0.11 | 0.34 | 0.83 |
| r1-062 | mc5h | 2.40 | 0.06 | 0.27 | 0.10 | 0.93 | 1.25 | 0.02 | 0.04 | 0.09 | 0.34 | 0.73 |
| r1-042 | 8pfm | 2.39 | 0.05 | 0.34 | 0.10 | 0.96 | 1.28 | 0.04 | 0.05 | 0.10 | 0.34 | 0.73 |
| r1-015 | mc5h | 2.33 | 0.06 | 0.34 | 0.10 | 0.70 | 1.48 | 0.30 | 0.04 | 0.02 | 0.13 | 0.97 |
| r1-056 | 8pfm | 2.32 | 0.05 | 0.43 | 0.17 | 0.92 | 1.25 | 0.08 | 0.05 | 0.09 | 0.34 | 0.65 |
| r1-070 | mc5h | 2.28 | 0.07 | 0.29 | 0.16 | 0.70 | 1.42 | 0.29 | 0.06 | 0.08 | 0.31 | 0.62 |
| r1-066 | 8pfm | 2.25 | 0.08 | 0.32 | 0.17 | 0.69 | 1.38 | 0.27 | 0.05 | 0.09 | 0.35 | 0.58 |
| r1-023 | 8pfm | 2.25 | 0.06 | 0.33 | 0.13 | 0.90 | 1.19 | 0.10 | 0.05 | 0.10 | 0.35 | 0.57 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 8pfm | 140.7 | 432.7 | 15188.7 | 1.3 | 7.1 | 99.1 | 4.8 | 46.8 | 50.5 | 24.9 | 32.7 | 28.0 | 0.0 | 323.5 | 7.7 | containerd-nydu/15704/containerd-nydu |
| mc5h | 117.2 | 470.3 | 16782.7 | 1.4 | 6.9 | 158.3 | 1.9 | 51.9 | 55.3 | 9.5 | 42.7 | 27.3 | 0.0 | 315.1 | 0.0 | containerd-nydu/16879/containerd-nydu |
| pgdw | 129.4 | 523.1 | 18932.1 | 1.6 | 8.2 | 148.3 | 5.5 | 48.9 | 51.9 | 14.2 | 37.8 | 32.9 | 0.0 | 283.8 | 0.1 | containerd-nydu/15993/containerd-nydu |

## Top processes in the window, cores per node

- 8pfm: nydusd-amp 1.06, nydus-dragonfly 0.94, ateworker-gviso 0.67, objectstore-plu 0.63, containerd-nydu 0.27, fluent-bit 0.18, atelet 0.11, kubelet 0.10
- mc5h: nydusd-amp 1.14, nydus-dragonfly 0.92, objectstore-plu 0.64, ateworker-gviso 0.46, containerd-nydu 0.26, fluent-bit 0.17, atelet 0.11, kubelet 0.09
- pgdw: nydusd-amp 1.20, nydus-dragonfly 0.82, objectstore-plu 0.63, ateworker-gviso 0.45, containerd-nydu 0.23, fluent-bit 0.18, atelet 0.10, kubelet 0.10

## Timeline, node 8pfm

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 161 | 1973 | 1.11 | 99 | 25 | 2 | 3.4 | 6.1 | 101 | 242.0 |
| 1 | 24 | 50 | 433 | 4190 | 1.34 | 77 | 50 | 25 | 10.0 | 18.5 | 133 | 1.0 |
| 2 | 9 | 141 | 240 | 15189 | 0.46 | 66 | 50 | 2 | 28.0 | 32.7 | 299 | 1.0 |
| 3 | 0 | 94 | 290 | 26271 | 0.42 | 56 | 39 | 7 | 44.7 | 8.9 | 269 | 0.0 |

## Timeline, node mc5h

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 470 | 3819 | 1.36 | 158 | 41 | 5 | 7.2 | 9.9 | 101 | 209.0 |
| 1 | 24 | 81 | 241 | 4634 | 0.80 | 34 | 43 | 9 | 11.2 | 18.8 | 147 | 1.5 |
| 3 | 0 | 117 | 193 | 16783 | 0.40 | 52 | 55 | 4 | 27.3 | 42.7 | 311 | 0.0 |
| 4 | 0 | 69 | 305 | 26742 | 0.41 | 61 | 38 | 7 | 45.8 | 7.0 | 219 | 0.0 |

## Timeline, node pgdw

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 523 | 3995 | 1.62 | 148 | 37 | 6 | 7.7 | 12.6 | 118 | 217.1 |
| 1 | 24 | 115 | 101 | 5440 | 0.57 | 41 | 52 | 14 | 11.1 | 23.4 | 172 | 1.6 |
| 2 | 0 | 129 | 191 | 18932 | 0.43 | 46 | 52 | 4 | 32.9 | 37.8 | 284 | 0.0 |
| 4 | 0 | 60 | 297 | 25079 | 0.45 | 54 | 36 | 7 | 47.5 | 4.6 | 223 | 0.0 |

## Seed requests (nginx)

2221 requests in the window, 683 MB. Seed time p50 / p99 / max: 0.0 / 158.4 / 300.0 ms. Size p50 / p99: 173 / 3859 KB. Status: {200: 288, 206: 1933}.
