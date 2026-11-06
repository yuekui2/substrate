# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-repeats/driver/../c-n24-r1/rep-1/nydus-oc-w2

72 first turns. TTFI p50 / p95 / p99 / p100: 1.78 / 2.70 / 2.96 / 3.03 s. Window 3.0 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 2494 | 22 | 1.56 | 1.87 |
| 482m | 24 | 1.75 | 2.45 |
| 4zpx | 26 | 2.03 | 3.03 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-068 | 4zpx | 3.03 | 0.07 | 0.44 | 0.25 | 0.91 | 1.98 | 0.09 | 0.08 | 0.08 | 0.10 | 1.27 |
| r1-018 | 4zpx | 2.94 | 0.06 | 0.44 | 0.22 | 0.91 | 1.88 | 0.09 | 0.07 | 0.08 | 0.35 | 1.25 |
| r1-021 | 4zpx | 2.93 | 0.06 | 0.48 | 0.25 | 0.97 | 1.80 | 0.02 | 0.06 | 0.07 | 0.35 | 1.24 |
| r1-027 | 4zpx | 2.83 | 0.04 | 0.41 | 0.29 | 0.90 | 1.77 | 0.08 | 0.06 | 0.08 | 0.39 | 1.13 |
| r1-061 | 4zpx | 2.59 | 0.06 | 0.42 | 0.30 | 0.92 | 1.52 | 0.08 | 0.06 | 0.07 | 0.36 | 0.89 |
| r1-046 | 4zpx | 2.51 | 0.07 | 0.46 | 0.27 | 0.73 | 1.62 | 0.26 | 0.08 | 0.08 | 0.37 | 0.81 |
| r1-017 | 482m | 2.45 | 0.07 | 0.35 | 0.18 | 0.92 | 1.38 | 0.02 | 0.04 | 0.07 | 0.37 | 0.84 |
| r1-045 | 4zpx | 2.41 | 0.04 | 0.42 | 0.21 | 0.75 | 1.53 | 0.27 | 0.06 | 0.07 | 0.41 | 0.67 |
| r1-011 | 4zpx | 2.38 | 0.05 | 0.44 | 0.24 | 0.90 | 1.35 | 0.11 | 0.08 | 0.06 | 0.39 | 0.65 |
| r1-004 | 482m | 2.35 | 0.07 | 0.33 | 0.16 | 0.91 | 1.28 | 0.03 | 0.06 | 0.03 | 0.40 | 0.74 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2494 | 152.6 | 532.6 | 22936.8 | 1.4 | 14.4 | 152.9 | 0.0 | 48.1 | 57.1 | 5.6 | 36.5 | 42.0 | 0.0 | 324.2 | 24.1 | nydusd-amp/14650/state_machine |
| 482m | 137.5 | 550.1 | 19013.9 | 1.7 | 8.7 | 148.7 | 5.7 | 48.6 | 55.3 | 8.6 | 35.7 | 33.6 | 0.0 | 544.5 | 2.4 | containerd-nydu/15465/containerd-nydu |
| 4zpx | 124.1 | 561.3 | 15978.7 | 1.1 | 8.7 | 156.1 | 0.0 | 48.2 | 52.0 | 14.9 | 35.9 | 28.2 | 0.0 | 637.1 | 27.5 | containerd-nydu/16184/containerd-nydu |

## Top processes in the window, cores per node

- 2494: nydusd-amp 1.18, nydus-dragonfly 0.63, ateworker-gviso 0.43, objectstore-plu 0.42, containerd-nydu 0.19, fluent-bit 0.12, atelet 0.07, kworker 0.06
- 482m: nydusd-amp 1.25, nydus-dragonfly 0.82, ateworker-gviso 0.82, objectstore-plu 0.64, containerd-nydu 0.25, fluent-bit 0.15, kubelet 0.13, kworker 0.11
- 4zpx: nydusd-amp 1.16, nydus-dragonfly 1.12, ateworker-gviso 0.76, objectstore-plu 0.66, containerd-nydu 0.27, fluent-bit 0.17, atelet 0.09, kworker 0.08

## Timeline, node 2494

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 7 | 243 | 0.42 | 1 | 7 | 1 | 1.0 | 0.5 | 112 | 268.6 |
| 1 | 22 | 2 | 533 | 4364 | 1.37 | 153 | 57 | 3 | 8.4 | 17.7 | 243 | 1.0 |
| 2 | 0 | 153 | 302 | 14562 | 0.52 | 66 | 53 | 2 | 24.8 | 36.5 | 187 | 0.1 |
| 3 | 0 | 76 | 433 | 22937 | 0.62 | 55 | 36 | 6 | 42.0 | 10.7 | 198 | 0.0 |
| 4 | 0 | 79 | 174 | 24469 | 0.37 | 32 | 29 | 4 | 43.9 | 2.1 | 128 | 0.0 |

## Timeline, node 482m

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 550 | 4425 | 1.74 | 149 | 38 | 6 | 7.6 | 12.1 | 100 | 193.5 |
| 1 | 24 | 90 | 370 | 6189 | 1.30 | 29 | 49 | 9 | 9.5 | 21.8 | 138 | 1.0 |
| 2 | 5 | 137 | 234 | 19014 | 0.45 | 54 | 55 | 4 | 33.6 | 35.7 | 275 | 1.0 |
| 4 | 0 | 64 | 524 | 24921 | 0.60 | 47 | 35 | 5 | 46.7 | 4.9 | 174 | 0.0 |

## Timeline, node 4zpx

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 337 | 3222 | 0.98 | 156 | 32 | 2 | 6.4 | 10.7 | 105 | 404.6 |
| 1 | 26 | 56 | 561 | 5598 | 1.12 | 51 | 49 | 15 | 11.2 | 19.6 | 113 | 1.9 |
| 2 | 13 | 124 | 442 | 15979 | 0.54 | 70 | 52 | 1 | 28.2 | 35.9 | 290 | 1.0 |
| 4 | 0 | 90 | 447 | 26236 | 0.45 | 58 | 42 | 8 | 44.9 | 11.2 | 196 | 0.0 |

## Seed requests (nginx)

2349 requests in the window, 741 MB. Seed time p50 / p99 / max: 0.0 / 148.5 / 414.0 ms. Size p50 / p99: 184 / 3861 KB. Status: {200: 288, 206: 2061}.
