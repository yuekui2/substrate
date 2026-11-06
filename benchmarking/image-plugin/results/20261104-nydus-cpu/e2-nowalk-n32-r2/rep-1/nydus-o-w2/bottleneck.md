# Bottleneck: benchmarking/image-plugin/results/20261104-nydus-cpu/e2-nowalk-n32-r2/rep-1/nydus-o-w2

96 first turns. TTFI p50 / p95 / p99 / p100: 2.21 / 3.59 / 4.15 / 4.35 s. Window 4.3 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 29l8 | 32 | 2.17 | 3.27 |
| 8gkx | 32 | 2.09 | 3.31 |
| h5vp | 32 | 2.60 | 4.35 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-022 | h5vp | 4.35 | 0.08 | 0.18 | 0.12 | 0.96 | 3.03 | 0.16 | 0.11 | 0.30 | 0.16 | 2.27 |
| r1-025 | h5vp | 4.14 | 0.07 | 0.32 | 0.11 | 1.24 | 2.72 | 0.12 | 0.34 | 0.14 | 0.05 | 2.01 |
| r1-044 | h5vp | 4.14 | 0.07 | 0.36 | 0.13 | 1.24 | 2.71 | 0.07 | 0.08 | 0.31 | 0.20 | 2.02 |
| r1-092 | h5vp | 3.88 | 0.07 | 0.38 | 0.16 | 1.23 | 2.44 | 0.13 | 0.34 | 0.12 | 0.05 | 1.76 |
| r1-094 | h5vp | 3.65 | 0.06 | 0.34 | 0.16 | 1.24 | 2.22 | 0.07 | 0.11 | 0.23 | 0.23 | 1.50 |
| r1-024 | h5vp | 3.58 | 0.07 | 0.33 | 0.14 | 1.13 | 2.27 | 0.17 | 0.10 | 0.25 | 0.17 | 1.50 |
| r1-074 | h5vp | 3.54 | 0.08 | 0.33 | 0.16 | 1.12 | 2.23 | 0.18 | 0.08 | 0.32 | 0.19 | 1.43 |
| r1-004 | h5vp | 3.49 | 0.06 | 0.33 | 0.12 | 1.12 | 2.19 | 0.16 | 0.11 | 0.32 | 0.08 | 1.37 |
| r1-015 | 8gkx | 3.31 | 0.08 | 0.30 | 0.19 | 1.17 | 1.94 | 0.05 | 0.22 | 0.10 | 0.14 | 1.40 |
| r1-048 | 29l8 | 3.27 | 0.07 | 0.34 | 0.16 | 0.93 | 2.15 | 0.30 | 0.05 | 0.05 | 0.56 | 1.16 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 29l8 | 137.8 | 668.9 | 25219.2 | 1.8 | 11.4 | 216.8 | 6.3 | 65.4 | 82.4 | 17.9 | 50.9 | 44.0 | 0.0 | 456.8 | 0.4 | containerd-nydu/14432/containerd-nydu |
| 8gkx | 132.7 | 495.4 | 28730.3 | 1.2 | 16.9 | 186.9 | 1.0 | 53.6 | 59.6 | 35.5 | 38.1 | 48.8 | 0.0 | 448.1 | 0.2 | containerd-nydu/17652/containerd-nydu |
| h5vp | 158.4 | 698.2 | 24059.4 | 1.8 | 11.8 | 233.7 | 21.2 | 49.4 | 56.3 | 12.4 | 28.1 | 42.7 | 0.0 | 509.2 | 0.7 | containerd-nydu/14718/containerd-nydu |

## Top processes in the window, cores per node

- 29l8: ateworker-gviso 0.80, nydus-dragonfly 0.73, objectstore-plu 0.50, containerd-nydu 0.20, fluent-bit 0.15, atelet 0.08, kworker 0.07, cilium-agent 0.06
- 8gkx: nydusd-amp 1.94, nydus-dragonfly 0.73, objectstore-plu 0.48, ateworker-gviso 0.48, containerd-nydu 0.19, fluent-bit 0.15, kworker 0.09, atelet 0.08
- h5vp: nydusd-amp 1.61, nydus-dragonfly 0.79, ateworker-gviso 0.57, objectstore-plu 0.48, containerd-nydu 0.19, fluent-bit 0.15, kworker 0.08, atelet 0.08

## Timeline, node 29l8

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 52 | 0.43 | 0 | 3 | 1 | 0.3 | 0.1 | 137 | 346.7 |
| 1 | 10 | 1 | 669 | 5635 | 1.85 | 217 | 55 | 2 | 14.6 | 13.8 | 75 | 1.0 |
| 2 | 26 | 105 | 98 | 4006 | 0.86 | 28 | 82 | 18 | 12.5 | 23.6 | 259 | 2.0 |
| 3 | 2 | 138 | 269 | 19926 | 0.43 | 58 | 69 | 5 | 31.2 | 50.9 | 301 | 1.0 |
| 4 | 0 | 80 | 353 | 25219 | 0.45 | 63 | 47 | 8 | 44.0 | 15.0 | 256 | 0.0 |
| 5 | 0 | 90 | 215 | 30750 | 0.37 | 43 | 35 | 7 | 47.6 | 3.9 | 203 | 0.0 |

## Timeline, node 8gkx

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 344 | 3068 | 1.21 | 187 | 38 | 2 | 4.4 | 13.3 | 131 | 371.8 |
| 1 | 22 | 52 | 382 | 4394 | 0.86 | 46 | 51 | 35 | 8.1 | 19.9 | 91 | 1.0 |
| 2 | 19 | 133 | 245 | 14903 | 0.45 | 62 | 60 | 2 | 24.5 | 38.1 | 266 | 1.4 |
| 4 | 0 | 124 | 329 | 24537 | 0.43 | 64 | 52 | 7 | 42.5 | 27.2 | 254 | 0.0 |
| 5 | 0 | 83 | 495 | 28730 | 0.59 | 43 | 38 | 7 | 48.8 | 5.7 | 189 | 0.0 |
| 6 | 0 | 61 | 238 | 16035 | 0.44 | 22 | 25 | 4 | 27.3 | 5.0 | 10 | 0.0 |

## Timeline, node h5vp

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 133 | 563 | 1.26 | 0 | 2 | 1 | 2.2 | 0.1 | 132 | 306.4 |
| 1 | 7 | 0 | 698 | 5268 | 1.78 | 234 | 48 | 2 | 9.0 | 11.4 | 24 | 0.8 |
| 2 | 31 | 52 | 390 | 4039 | 1.29 | 17 | 48 | 12 | 11.2 | 21.8 | 293 | 1.0 |
| 3 | 8 | 158 | 365 | 19070 | 0.46 | 72 | 56 | 1 | 33.1 | 28.1 | 392 | 1.0 |
| 4 | 3 | 126 | 503 | 24059 | 0.48 | 76 | 53 | 9 | 42.7 | 20.7 | 295 | 0.0 |
| 5 | 0 | 97 | 382 | 27538 | 0.42 | 47 | 35 | 7 | 45.0 | 8.3 | 194 | 0.0 |

## Seed requests (nginx)

3481 requests in the window, 1049 MB. Seed time p50 / p99 / max: 0.0 / 171.2 / 430.0 ms. Size p50 / p99: 178 / 3804 KB. Status: {200: 384, 206: 3097}.
