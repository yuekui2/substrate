# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-repeats/driver/../placement-1/rep-1/nydus-o-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 1.13 / 1.32 / 1.43 / 1.47 s. Window 1.5 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 9f08 | 17 | 1.19 | 1.28 |
| dgm4 | 16 | 1.16 | 1.47 |
| xtg2 | 15 | 0.57 | 1.32 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-025 | dgm4 | 1.47 | 0.06 | 0.72 | 0.31 | 0.65 | 0.29 | 0.00 | 0.04 | 0.01 | 0.10 | 0.10 |
| r1-037 | dgm4 | 1.39 | 0.10 | 0.73 | 0.36 | 0.44 | 0.15 | 0.00 | 0.03 | 0.01 | 0.04 | 0.03 |
| r1-013 | xtg2 | 1.32 | 0.07 | 0.71 | 0.24 | 0.40 | 0.20 | 0.00 | 0.02 | 0.01 | 0.09 | 0.03 |
| r1-047 | dgm4 | 1.32 | 0.05 | 0.53 | 0.37 | 0.30 | 0.24 | 0.02 | 0.05 | 0.01 | 0.08 | 0.04 |
| r1-026 | dgm4 | 1.31 | 0.06 | 0.55 | 0.36 | 0.34 | 0.22 | 0.01 | 0.03 | 0.01 | 0.07 | 0.07 |
| r1-030 | 9f08 | 1.28 | 0.07 | 0.62 | 0.29 | 0.50 | 0.22 | 0.00 | 0.05 | 0.01 | 0.08 | 0.04 |
| r1-028 | dgm4 | 1.25 | 0.05 | 0.53 | 0.22 | 0.40 | 0.33 | 0.02 | 0.04 | 0.01 | 0.06 | 0.17 |
| r1-021 | 9f08 | 1.24 | 0.06 | 0.58 | 0.15 | 0.47 | 0.35 | 0.02 | 0.03 | 0.02 | 0.10 | 0.12 |
| r1-035 | dgm4 | 1.23 | 0.05 | 0.57 | 0.30 | 0.34 | 0.22 | 0.01 | 0.04 | 0.01 | 0.09 | 0.04 |
| r1-045 | 9f08 | 1.23 | 0.04 | 0.53 | 0.35 | 0.38 | 0.19 | 0.00 | 0.04 | 0.01 | 0.06 | 0.04 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 9f08 | 98.5 | 157.7 | 15614.6 | 0.9 | 7.4 | 49.5 | 7.9 | 62.3 | 70.4 | 7.4 | 34.0 | 25.4 | 0.0 | 131.3 | 0.1 | containerd-nydu/14460/containerd-nydu |
| dgm4 | 100.4 | 205.9 | 15766.8 | 1.0 | 8.0 | 44.2 | 0.0 | 62.5 | 72.8 | 5.5 | 36.7 | 24.5 | 0.0 | 155.7 | 0.2 | containerd-nydu/14659/containerd-nydu |
| xtg2 | 85.4 | 243.6 | 10615.9 | 1.0 | 5.9 | 56.3 | 0.9 | 58.4 | 64.8 | 3.5 | 30.9 | 17.9 | 0.0 | 110.8 | 0.1 | containerd-nydu/14570/containerd-nydu |

## Top processes in the window, cores per node

- 9f08: nydus-dragonfly 5.18, nydusd-amp 0.43, ateworker-gviso 0.39, objectstore-plu 0.35, fluent-bit 0.19, containerd-nydu 0.16, kubelet 0.14, atelet 0.10
- dgm4: nydus-dragonfly 4.00, objectstore-plu 0.43, nydusd-amp 0.42, ateworker-gviso 0.33, containerd-nydu 0.16, fluent-bit 0.14, atelet 0.10, containerd 0.09
- xtg2: nydus-dragonfly 3.30, ateworker-gviso 0.33, nydusd-amp 0.28, objectstore-plu 0.28, fluent-bit 0.15, kubelet 0.13, containerd-nydu 0.11, atelet 0.09

## Timeline, node 9f08

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 8 | 76 | 150 | 3586 | 0.91 | 49 | 48 | 7 | 8.1 | 22.7 | 53 | 187.9 |
| 1 | 9 | 98 | 158 | 15615 | 0.47 | 26 | 70 | 3 | 25.4 | 34.0 | 108 | 6.6 |
| 3 | 0 | 47 | 158 | 12168 | 0.45 | 22 | 22 | 2 | 26.2 | 6.7 | 63 | 0.0 |

## Timeline, node dgm4

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 6 | 43 | 98 | 1361 | 1.03 | 40 | 27 | 6 | 3.5 | 6.8 | 59 | 269.0 |
| 1 | 11 | 100 | 206 | 15767 | 0.50 | 44 | 73 | 3 | 24.5 | 36.7 | 73 | 8.8 |
| 2 | 0 | 88 | 167 | 18504 | 0.43 | 33 | 42 | 3 | 32.6 | 22.3 | 193 | 0.1 |

## Timeline, node xtg2

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 10 | 44 | 6 | 627 | 0.97 | 6 | 18 | 3 | 1.9 | 4.2 | 41 | 338.4 |
| 1 | 7 | 85 | 244 | 10616 | 0.55 | 56 | 65 | 4 | 17.9 | 30.9 | 99 | 0.0 |
| 2 | 0 | 75 | 143 | 15030 | 0.41 | 20 | 34 | 4 | 27.2 | 12.0 | 96 | 0.0 |

## Seed requests (nginx)

667 requests in the window, 217 MB. Seed time p50 / p99 / max: 0.0 / 168.7 / 442.0 ms. Size p50 / p99: 122 / 3863 KB. Status: {200: 108, 206: 559}.
