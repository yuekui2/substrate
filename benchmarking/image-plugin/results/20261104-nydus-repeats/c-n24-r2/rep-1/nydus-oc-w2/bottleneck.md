# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-repeats/driver/../c-n24-r2/rep-1/nydus-oc-w2

72 first turns. TTFI p50 / p95 / p99 / p100: 1.81 / 2.29 / 2.44 / 2.53 s. Window 2.5 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 5p63 | 24 | 1.82 | 2.25 |
| 982b | 25 | 1.71 | 2.31 |
| snkg | 23 | 1.86 | 2.53 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-021 | snkg | 2.53 | 0.06 | 0.52 | 0.38 | 0.95 | 1.44 | 0.01 | 0.06 | 0.03 | 0.39 | 0.90 |
| r1-031 | snkg | 2.41 | 0.05 | 0.67 | 0.16 | 0.90 | 1.36 | 0.06 | 0.05 | 0.03 | 0.35 | 0.81 |
| r1-033 | snkg | 2.38 | 0.05 | 0.70 | 0.13 | 0.74 | 1.40 | 0.13 | 0.04 | 0.01 | 0.41 | 0.72 |
| r1-027 | 982b | 2.31 | 0.06 | 0.40 | 0.20 | 0.92 | 1.22 | 0.04 | 0.06 | 0.06 | 0.31 | 0.73 |
| r1-014 | 982b | 2.27 | 0.05 | 0.43 | 0.17 | 0.84 | 1.27 | 0.13 | 0.16 | 0.25 | 0.09 | 0.55 |
| r1-048 | snkg | 2.25 | 0.06 | 0.67 | 0.28 | 0.74 | 1.15 | 0.01 | 0.05 | 0.04 | 0.37 | 0.64 |
| r1-068 | 5p63 | 2.25 | 0.06 | 0.43 | 0.23 | 0.93 | 1.16 | 0.03 | 0.05 | 0.12 | 0.33 | 0.60 |
| r1-015 | 5p63 | 2.14 | 0.07 | 0.38 | 0.26 | 0.88 | 1.09 | 0.07 | 0.04 | 0.01 | 0.17 | 0.77 |
| r1-051 | 982b | 2.12 | 0.06 | 0.50 | 0.22 | 0.83 | 1.14 | 0.14 | 0.14 | 0.26 | 0.12 | 0.46 |
| r1-018 | snkg | 2.11 | 0.06 | 0.51 | 0.28 | 0.71 | 1.15 | 0.14 | 0.05 | 0.04 | 0.36 | 0.51 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 5p63 | 129.7 | 600.7 | 22463.3 | 1.9 | 10.5 | 178.1 | 0.9 | 44.5 | 54.7 | 4.6 | 30.5 | 40.0 | 0.0 | 371.3 | 0.1 | nydusd-amp/16502/state_machine |
| 982b | 147.5 | 553.9 | 23001.8 | 2.3 | 10.4 | 158.3 | 0.0 | 48.1 | 53.7 | 5.4 | 30.5 | 38.6 | 0.0 | 334.4 | 0.0 | nydusd-amp/17730/state_machine |
| snkg | 122.3 | 471.5 | 16771.6 | 1.3 | 8.0 | 93.7 | 0.9 | 47.6 | 51.5 | 18.8 | 35.8 | 30.3 | 0.0 | 303.8 | 0.3 | containerd-nydu/21468/containerd-nydu |

## Top processes in the window, cores per node

- 5p63: nydusd-amp 1.15, nydus-dragonfly 0.77, objectstore-plu 0.47, ateworker-gviso 0.40, containerd-nydu 0.21, fluent-bit 0.14, atelet 0.08, kubelet 0.08
- 982b: nydusd-amp 1.23, nydus-dragonfly 0.67, objectstore-plu 0.47, ateworker-gviso 0.43, containerd-nydu 0.20, fluent-bit 0.15, atelet 0.08, kubelet 0.08
- snkg: nydusd-amp 1.04, nydus-dragonfly 0.81, objectstore-plu 0.58, ateworker-gviso 0.45, containerd-nydu 0.25, fluent-bit 0.18, kubelet 0.12, atelet 0.11

## Timeline, node 5p63

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 56 | 0.42 | 0 | 7 | 3 | 0.3 | 0.5 | 107 | 419.8 |
| 1 | 24 | 0 | 601 | 4608 | 1.93 | 178 | 51 | 2 | 11.6 | 18.1 | 194 | 1.0 |
| 2 | 3 | 130 | 183 | 9529 | 0.56 | 71 | 55 | 1 | 19.1 | 30.5 | 272 | 0.0 |
| 3 | 0 | 105 | 286 | 22463 | 0.46 | 55 | 48 | 5 | 40.0 | 23.6 | 269 | 0.0 |
| 4 | 0 | 55 | 196 | 22374 | 0.45 | 34 | 25 | 5 | 48.0 | 2.3 | 169 | 0.0 |

## Timeline, node 982b

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 2 | 78 | 0.62 | 0 | 5 | 3 | 0.6 | 0.4 | 114 | 217.6 |
| 1 | 25 | 0 | 554 | 4525 | 2.25 | 158 | 53 | 2 | 11.1 | 17.8 | 196 | 1.0 |
| 2 | 5 | 148 | 180 | 11260 | 0.52 | 65 | 54 | 2 | 22.6 | 30.5 | 259 | 0.4 |
| 3 | 0 | 118 | 298 | 23002 | 0.44 | 52 | 51 | 5 | 38.6 | 26.0 | 232 | 0.0 |
| 4 | 0 | 59 | 172 | 25540 | 0.38 | 32 | 25 | 5 | 47.8 | 2.2 | 171 | 0.0 |

## Timeline, node snkg

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 117 | 2129 | 0.82 | 94 | 24 | 1 | 3.8 | 4.2 | 93 | 193.2 |
| 1 | 23 | 69 | 472 | 4491 | 1.30 | 84 | 50 | 19 | 11.0 | 18.3 | 149 | 1.5 |
| 2 | 6 | 122 | 220 | 16772 | 0.47 | 48 | 51 | 2 | 30.3 | 35.8 | 237 | 0.6 |
| 4 | 0 | 77 | 261 | 24616 | 0.42 | 51 | 31 | 6 | 44.6 | 7.3 | 142 | 0.0 |

## Seed requests (nginx)

2122 requests in the window, 702 MB. Seed time p50 / p99 / max: 0.0 / 160.3 / 436.0 ms. Size p50 / p99: 187 / 3863 KB. Status: {200: 288, 206: 1834}.
