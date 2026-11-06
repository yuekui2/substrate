# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-n32-none/driver/../b-zstd-n32-r2/rep-1/nydus-ob-w2

96 first turns. TTFI p50 / p95 / p99 / p100: 2.45 / 3.43 / 3.54 / 3.63 s. Window 3.6 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 33q8 | 32 | 2.88 | 3.63 |
| 56t8 | 32 | 2.25 | 3.25 |
| hk8c | 32 | 2.48 | 3.53 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-012 | 33q8 | 3.63 | 0.08 | 0.30 | 0.20 | 1.08 | 2.18 | 0.20 | 0.08 | 0.04 | 0.49 | 1.33 |
| r1-019 | hk8c | 3.53 | 0.07 | 0.42 | 0.26 | 1.15 | 2.18 | 0.20 | 0.04 | 0.01 | 0.60 | 1.27 |
| r1-046 | 33q8 | 3.50 | 0.09 | 0.52 | 0.29 | 1.24 | 2.05 | 0.19 | 0.06 | 0.03 | 0.54 | 1.21 |
| r1-082 | 33q8 | 3.49 | 0.08 | 0.46 | 0.28 | 1.21 | 2.03 | 0.18 | 0.07 | 0.05 | 0.51 | 1.19 |
| r1-004 | 33q8 | 3.46 | 0.08 | 0.46 | 0.25 | 1.22 | 2.05 | 0.23 | 0.09 | 0.07 | 0.45 | 1.17 |
| r1-076 | 33q8 | 3.42 | 0.09 | 0.51 | 0.30 | 1.23 | 1.99 | 0.20 | 0.05 | 0.04 | 0.53 | 1.13 |
| r1-061 | hk8c | 3.42 | 0.08 | 0.27 | 0.19 | 1.01 | 2.05 | 0.21 | 0.10 | 0.04 | 0.50 | 1.17 |
| r1-006 | 33q8 | 3.33 | 0.09 | 0.43 | 0.37 | 1.25 | 1.88 | 0.19 | 0.05 | 0.04 | 0.53 | 1.05 |
| r1-042 | 33q8 | 3.28 | 0.08 | 0.50 | 0.26 | 1.23 | 1.86 | 0.22 | 0.06 | 0.09 | 0.49 | 0.97 |
| r1-028 | 56t8 | 3.25 | 0.08 | 0.46 | 0.59 | 1.19 | 1.87 | 0.16 | 0.04 | 0.04 | 0.26 | 1.34 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 33q8 | 156.0 | 391.5 | 22114.4 | 1.5 | 11.2 | 202.7 | 2.9 | 55.2 | 59.8 | 18.1 | 38.2 | 41.0 | 0.0 | 437.6 | 0.5 | containerd-nydu/14805/containerd-nydu |
| 56t8 | 164.8 | 454.8 | 24386.3 | 1.7 | 10.4 | 204.8 | 0.0 | 65.9 | 69.2 | 16.1 | 55.3 | 40.5 | 0.0 | 450.9 | 176.6 | containerd-nydu/17246/containerd-nydu |
| hk8c | 126.5 | 717.1 | 19293.9 | 2.1 | 9.7 | 207.7 | 0.0 | 63.0 | 67.3 | 24.3 | 50.8 | 31.2 | 0.0 | 432.8 | 0.5 | containerd-nydu/16009/containerd-nydu |

## Top processes in the window, cores per node

- 33q8: nydusd-amp 1.48, nydus-dragonfly 0.88, objectstore-plu 0.61, ateworker-gviso 0.53, containerd-nydu 0.24, fluent-bit 0.17, atelet 0.09, kubelet 0.09
- 56t8: nydusd-amp 1.52, nydus-dragonfly 0.69, ateworker-gviso 0.46, objectstore-plu 0.46, containerd-nydu 0.19, fluent-bit 0.14, kubelet 0.07, atelet 0.07
- hk8c: nydusd-amp 1.27, nydus-dragonfly 0.88, objectstore-plu 0.61, ateworker-gviso 0.59, containerd-nydu 0.25, fluent-bit 0.16, kubelet 0.12, atelet 0.09

## Timeline, node 33q8

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 55 | 0.54 | 0 | 4 | 0 | 0.4 | 0.1 | 127 | 246.1 |
| 1 | 0 | 0 | 391 | 3362 | 1.37 | 203 | 43 | 2 | 13.9 | 10.5 | 19 | 61.7 |
| 2 | 32 | 0 | 357 | 2441 | 1.49 | 11 | 37 | 18 | 6.8 | 26.8 | 175 | 1.0 |
| 3 | 15 | 151 | 161 | 10515 | 0.54 | 58 | 48 | 2 | 22.5 | 26.7 | 322 | 1.0 |
| 4 | 0 | 156 | 261 | 22114 | 0.50 | 66 | 60 | 5 | 41.0 | 38.2 | 290 | 0.0 |
| 5 | 0 | 83 | 312 | 26209 | 0.49 | 55 | 38 | 7 | 51.2 | 10.7 | 208 | 0.0 |

## Timeline, node 56t8

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 52 | 0.39 | 0 | 3 | 1 | 0.2 | 0.1 | 129 | 233.9 |
| 1 | 9 | 0 | 304 | 3112 | 1.66 | 205 | 39 | 2 | 19.8 | 14.1 | 54 | 1.0 |
| 2 | 32 | 51 | 455 | 4480 | 1.46 | 21 | 51 | 16 | 21.4 | 22.5 | 228 | 1.0 |
| 3 | 4 | 165 | 200 | 14784 | 0.45 | 57 | 69 | 1 | 24.1 | 55.3 | 362 | 1.0 |
| 4 | 0 | 116 | 282 | 24386 | 0.42 | 72 | 51 | 9 | 40.5 | 22.5 | 318 | 0.0 |
| 5 | 0 | 55 | 285 | 26542 | 0.42 | 42 | 37 | 10 | 46.4 | 5.5 | 208 | 0.0 |

## Timeline, node hk8c

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 17 | 1455 | 0.40 | 2 | 13 | 2 | 2.1 | 1.5 | 128 | 340.6 |
| 1 | 0 | 0 | 717 | 4487 | 2.12 | 208 | 67 | 24 | 9.7 | 25.9 | 36 | 19.8 |
| 2 | 32 | 122 | 139 | 8045 | 0.53 | 56 | 58 | 2 | 16.7 | 23.4 | 256 | 1.0 |
| 3 | 5 | 126 | 222 | 19294 | 0.45 | 56 | 67 | 5 | 31.2 | 50.8 | 305 | 0.0 |
| 5 | 0 | 83 | 300 | 26194 | 0.43 | 68 | 43 | 8 | 44.6 | 7.5 | 227 | 0.0 |

## Seed requests (nginx)

2736 requests in the window, 884 MB. Seed time p50 / p99 / max: 0.0 / 189.7 / 477.0 ms. Size p50 / p99: 182 / 3861 KB. Status: {200: 384, 206: 2352}.
