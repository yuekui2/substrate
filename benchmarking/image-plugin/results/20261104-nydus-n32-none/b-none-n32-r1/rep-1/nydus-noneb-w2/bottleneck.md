# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-n32-none/driver/../b-none-n32-r1/rep-1/nydus-noneb-w2

96 first turns. TTFI p50 / p95 / p99 / p100: 2.12 / 2.98 / 3.30 / 3.35 s. Window 3.4 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 46fz | 32 | 2.24 | 3.35 |
| 7xq1 | 32 | 2.13 | 3.29 |
| l0ls | 32 | 1.94 | 2.65 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-075 | 46fz | 3.35 | 0.06 | 0.33 | 0.18 | 0.95 | 1.86 | 0.05 | 0.23 | 0.06 | 0.32 | 1.16 |
| r1-028 | 7xq1 | 3.29 | 0.06 | 0.51 | 0.37 | 1.34 | 1.77 | 0.06 | 0.23 | 0.13 | 0.12 | 1.20 |
| r1-042 | 7xq1 | 3.18 | 0.05 | 0.46 | 0.27 | 1.29 | 1.71 | 0.12 | 0.11 | 0.08 | 0.18 | 1.17 |
| r1-076 | 7xq1 | 3.08 | 0.06 | 0.58 | 0.62 | 1.29 | 1.63 | 0.13 | 0.12 | 0.08 | 0.19 | 1.08 |
| r1-092 | 7xq1 | 2.99 | 0.05 | 0.54 | 0.54 | 1.27 | 1.53 | 0.13 | 0.12 | 0.08 | 0.18 | 1.00 |
| r1-019 | 7xq1 | 2.97 | 0.06 | 0.53 | 0.26 | 1.30 | 1.51 | 0.12 | 0.12 | 0.08 | 0.19 | 0.97 |
| r1-058 | 46fz | 2.95 | 0.06 | 0.54 | 0.34 | 1.07 | 1.72 | 0.28 | 0.14 | 0.14 | 0.38 | 0.76 |
| r1-033 | 46fz | 2.87 | 0.09 | 0.50 | 0.36 | 1.04 | 1.64 | 0.28 | 0.13 | 0.14 | 0.36 | 0.70 |
| r1-055 | 7xq1 | 2.80 | 0.07 | 0.58 | 0.52 | 1.28 | 1.35 | 0.13 | 0.12 | 0.09 | 0.20 | 0.80 |
| r1-064 | 46fz | 2.78 | 0.07 | 0.52 | 0.36 | 1.06 | 1.54 | 0.30 | 0.24 | 0.04 | 0.28 | 0.62 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 46fz | 176.2 | 679.8 | 31084.5 | 1.5 | 14.9 | 293.1 | 0.0 | 56.4 | 65.9 | 5.6 | 40.8 | 47.5 | 0.0 | 415.6 | 0.0 | containerd-nydu/17095/containerd-nydu |
| 7xq1 | 154.8 | 429.3 | 41556.9 | 1.3 | 16.5 | 260.7 | 0.9 | 62.4 | 66.0 | 10.3 | 51.6 | 54.3 | 0.0 | 476.1 | 95.9 | containerd-nydu/14795/containerd-nydu |
| l0ls | 159.5 | 681.8 | 22441.9 | 2.5 | 12.9 | 228.2 | 2.9 | 65.2 | 71.7 | 5.7 | 56.6 | 31.9 | 0.0 | 382.7 | 0.0 | containerd/7209/containerd |

## Top processes in the window, cores per node

- 46fz: nydusd-amp 1.30, nydus-dragonfly 0.82, objectstore-plu 0.61, ateworker-gviso 0.49, containerd-nydu 0.26, fluent-bit 0.16, kubelet 0.09, atelet 0.09
- 7xq1: nydusd-amp 1.52, nydus-dragonfly 0.93, objectstore-plu 0.59, ateworker-gviso 0.58, containerd-nydu 0.27, fluent-bit 0.19, kubelet 0.12, containerd 0.11
- l0ls: nydusd-amp 1.26, nydus-dragonfly 1.09, objectstore-plu 0.79, ateworker-gviso 0.68, containerd-nydu 0.35, fluent-bit 0.20, atelet 0.11, kubelet 0.10

## Timeline, node 46fz

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 18 | 1529 | 0.42 | 27 | 16 | 4 | 2.9 | 2.3 | 133 | 299.8 |
| 1 | 11 | 1 | 680 | 4349 | 1.54 | 180 | 59 | 6 | 20.6 | 21.1 | 100 | 11.0 |
| 2 | 25 | 156 | 182 | 11347 | 0.55 | 165 | 66 | 2 | 22.6 | 40.8 | 266 | 5.4 |
| 3 | 1 | 176 | 372 | 31085 | 0.47 | 293 | 60 | 4 | 47.5 | 38.7 | 508 | 0.0 |
| 5 | 0 | 104 | 352 | 36567 | 0.44 | 162 | 42 | 4 | 55.4 | 7.5 | 125 | 0.0 |

## Timeline, node 7xq1

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 52 | 0.37 | 0 | 3 | 0 | 0.2 | 0.1 | 130 | 409.5 |
| 1 | 25 | 0 | 429 | 3856 | 1.25 | 216 | 50 | 3 | 15.3 | 16.9 | 91 | 2.1 |
| 2 | 19 | 94 | 402 | 6085 | 0.89 | 66 | 51 | 10 | 24.0 | 24.4 | 297 | 4.0 |
| 3 | 3 | 155 | 279 | 17052 | 0.45 | 208 | 66 | 3 | 27.4 | 51.6 | 444 | 1.6 |
| 4 | 0 | 109 | 410 | 41557 | 0.39 | 261 | 48 | 4 | 54.3 | 15.1 | 338 | 0.0 |

## Timeline, node l0ls

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 52 | 0.37 | 0 | 3 | 1 | 0.2 | 0.1 | 133 | 236.6 |
| 1 | 32 | 0 | 682 | 5141 | 2.46 | 196 | 46 | 6 | 17.1 | 17.5 | 115 | 2.7 |
| 2 | 9 | 124 | 350 | 7280 | 1.18 | 75 | 59 | 6 | 13.3 | 35.5 | 281 | 5.2 |
| 3 | 0 | 159 | 246 | 22442 | 0.41 | 228 | 72 | 3 | 31.9 | 56.6 | 517 | 0.0 |
| 4 | 0 | 100 | 452 | 41688 | 0.41 | 248 | 44 | 4 | 57.7 | 11.5 | 322 | 0.0 |

## Seed requests (nginx)

3369 requests in the window, 2106 MB. Seed time p50 / p99 / max: 0.0 / 201.3 / 566.0 ms. Size p50 / p99: 605 / 3744 KB. Status: {200: 384, 206: 2985}.
