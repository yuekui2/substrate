# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-repeats/driver/../b-n24-r1/rep-1/nydus-ob-w2

72 first turns. TTFI p50 / p95 / p99 / p100: 1.92 / 2.45 / 2.60 / 2.65 s. Window 2.6 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 8jln | 24 | 1.99 | 2.65 |
| nqnf | 24 | 1.91 | 2.12 |
| zjjq | 24 | 1.93 | 2.44 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-019 | 8jln | 2.65 | 0.06 | 0.40 | 0.18 | 1.03 | 1.46 | 0.01 | 0.06 | 0.11 | 0.34 | 0.92 |
| r1-008 | 8jln | 2.58 | 0.05 | 0.40 | 0.17 | 0.80 | 1.63 | 0.25 | 0.05 | 0.09 | 0.35 | 0.87 |
| r1-023 | 8jln | 2.55 | 0.06 | 0.39 | 0.19 | 0.79 | 1.59 | 0.23 | 0.08 | 0.08 | 0.35 | 0.82 |
| r1-018 | 8jln | 2.45 | 0.07 | 0.43 | 0.14 | 0.80 | 1.50 | 0.24 | 0.05 | 0.09 | 0.33 | 0.74 |
| r1-055 | zjjq | 2.44 | 0.05 | 0.35 | 0.17 | 0.93 | 1.35 | 0.05 | 0.08 | 0.06 | 0.29 | 0.84 |
| r1-020 | 8jln | 2.37 | 0.06 | 0.35 | 0.16 | 0.76 | 1.42 | 0.25 | 0.05 | 0.09 | 0.36 | 0.65 |
| r1-042 | zjjq | 2.30 | 0.06 | 0.44 | 0.13 | 0.97 | 1.17 | 0.02 | 0.08 | 0.06 | 0.08 | 0.64 |
| r1-039 | 8jln | 2.26 | 0.05 | 0.38 | 0.13 | 0.77 | 1.33 | 0.26 | 0.04 | 0.01 | 0.32 | 0.55 |
| r1-051 | zjjq | 2.21 | 0.06 | 0.35 | 0.18 | 0.94 | 1.11 | 0.05 | 0.07 | 0.05 | 0.29 | 0.60 |
| r1-069 | 8jln | 2.16 | 0.05 | 0.40 | 0.15 | 0.77 | 1.19 | 0.23 | 0.07 | 0.09 | 0.31 | 0.44 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 8jln | 133.8 | 698.6 | 19451.1 | 1.7 | 9.9 | 147.5 | 5.5 | 44.6 | 50.5 | 4.6 | 32.2 | 38.0 | 0.0 | 293.5 | 0.0 | containerd/8830/containerd |
| nqnf | 120.1 | 563.6 | 19795.0 | 1.4 | 9.2 | 161.9 | 3.8 | 43.6 | 56.8 | 3.7 | 35.0 | 36.3 | 0.0 | 331.2 | 0.0 | nydusd-amp/14519/state_machine |
| zjjq | 111.5 | 470.7 | 13749.8 | 1.9 | 9.0 | 155.7 | 6.5 | 48.7 | 58.8 | 13.9 | 39.0 | 30.8 | 0.0 | 562.9 | 0.0 | containerd-nydu/15634/containerd-nydu |

## Top processes in the window, cores per node

- 8jln: nydusd-amp 0.96, nydus-dragonfly 0.61, objectstore-plu 0.48, ateworker-gviso 0.45, containerd-nydu 0.18, fluent-bit 0.12, containerd 0.07, atelet 0.07
- nqnf: nydusd-amp 0.96, nydus-dragonfly 0.66, ateworker-gviso 0.54, objectstore-plu 0.47, containerd-nydu 0.21, fluent-bit 0.12, atelet 0.07, kubelet 0.07
- zjjq: nydus-dragonfly 0.89, nydusd-amp 0.89, ateworker-gviso 0.87, objectstore-plu 0.63, containerd-nydu 0.25, fluent-bit 0.15, kubelet 0.10, atelet 0.09

## Timeline, node 8jln

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 218 | 900 | 1.31 | 0 | 3 | 0 | 3.7 | 0.1 | 99 | 176.3 |
| 1 | 24 | 0 | 699 | 4781 | 1.67 | 148 | 48 | 2 | 10.1 | 11.5 | 98 | 1.0 |
| 2 | 11 | 90 | 128 | 4967 | 0.67 | 33 | 46 | 3 | 12.4 | 22.5 | 283 | 0.0 |
| 3 | 0 | 134 | 258 | 19451 | 0.50 | 62 | 50 | 5 | 38.0 | 32.2 | 336 | 0.0 |
| 4 | 0 | 89 | 291 | 24787 | 0.48 | 49 | 35 | 7 | 49.0 | 6.2 | 202 | 0.0 |

## Timeline, node nqnf

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 61 | 0.50 | 0 | 2 | 0 | 0.6 | 0.1 | 104 | 224.8 |
| 1 | 24 | 0 | 564 | 4285 | 1.43 | 162 | 45 | 3 | 13.4 | 10.6 | 191 | 2.1 |
| 2 | 2 | 120 | 138 | 7318 | 0.61 | 58 | 57 | 2 | 16.5 | 22.1 | 226 | 0.0 |
| 3 | 0 | 97 | 207 | 19795 | 0.45 | 44 | 47 | 4 | 36.3 | 35.0 | 278 | 0.0 |
| 4 | 0 | 46 | 248 | 20202 | 0.51 | 42 | 31 | 5 | 45.6 | 5.7 | 129 | 0.0 |

## Timeline, node zjjq

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 471 | 3308 | 1.92 | 156 | 34 | 2 | 6.3 | 10.4 | 97 | 263.6 |
| 1 | 24 | 68 | 431 | 4770 | 1.37 | 32 | 59 | 14 | 16.3 | 16.6 | 134 | 1.0 |
| 2 | 8 | 112 | 291 | 13750 | 0.65 | 43 | 53 | 2 | 30.8 | 39.0 | 194 | 0.1 |
| 4 | 0 | 93 | 414 | 24912 | 0.53 | 66 | 40 | 6 | 51.6 | 8.6 | 172 | 0.0 |

## Seed requests (nginx)

2103 requests in the window, 668 MB. Seed time p50 / p99 / max: 0.0 / 157.0 / 327.0 ms. Size p50 / p99: 183 / 3860 KB. Status: {200: 288, 206: 1815}.
