# Bottleneck: benchmarking/image-plugin/results/20261103-confirm-sweep/set-c/n16/rep-1/nydus-pfc-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 1.44 / 1.95 / 2.24 / 2.37 s. Window 2.4 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 6wcf | 19 | 1.49 | 2.37 |
| lzgj | 14 | 1.26 | 1.51 |
| v6sh | 15 | 1.28 | 1.68 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-004 | 6wcf | 2.37 | 0.04 | 0.35 | 0.20 | 0.67 | 1.56 | 0.10 | 0.05 | 0.09 | 0.25 | 1.03 |
| r1-027 | 6wcf | 2.09 | 0.04 | 0.37 | 0.18 | 0.78 | 1.21 | 0.03 | 0.06 | 0.06 | 0.28 | 0.75 |
| r1-041 | 6wcf | 2.01 | 0.04 | 0.33 | 0.20 | 0.66 | 1.22 | 0.12 | 0.05 | 0.07 | 0.28 | 0.68 |
| r1-024 | 6wcf | 1.83 | 0.04 | 0.38 | 0.22 | 0.71 | 1.02 | 0.10 | 0.05 | 0.07 | 0.29 | 0.50 |
| r1-017 | 6wcf | 1.76 | 0.03 | 0.42 | 0.24 | 0.69 | 0.96 | 0.10 | 0.04 | 0.01 | 0.11 | 0.66 |
| r1-015 | v6sh | 1.68 | 0.04 | 0.41 | 0.31 | 0.78 | 0.80 | 0.08 | 0.06 | 0.04 | 0.08 | 0.46 |
| r1-021 | v6sh | 1.68 | 0.04 | 0.42 | 0.30 | 0.78 | 0.81 | 0.09 | 0.07 | 0.04 | 0.09 | 0.49 |
| r1-031 | 6wcf | 1.65 | 0.01 | 0.38 | 0.27 | 0.68 | 0.88 | 0.15 | 0.04 | 0.08 | 0.29 | 0.31 |
| r1-038 | 6wcf | 1.60 | 0.04 | 0.36 | 0.30 | 0.62 | 0.79 | 0.09 | 0.04 | 0.03 | 0.09 | 0.52 |
| r1-033 | v6sh | 1.56 | 0.03 | 0.36 | 0.26 | 0.52 | 0.83 | 0.04 | 0.03 | 0.06 | 0.13 | 0.53 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 6wcf | 171.6 | 431.3 | 29571.4 | 1.1 | 11.6 | 122.7 | 5.6 | 69.6 | 79.8 | 4.8 | 35.1 | 43.6 | 0.0 | 254.2 | 7.8 | containerd/7124/containerd |
| lzgj | 144.0 | 540.4 | 23689.0 | 1.8 | 10.4 | 91.5 | 6.5 | 61.7 | 75.0 | 3.7 | 23.8 | 38.1 | 0.1 | 202.9 | 0.1 | nydusd-amp/14376/state_machine |
| v6sh | 148.8 | 361.3 | 24583.5 | 0.9 | 10.4 | 116.2 | 0.0 | 65.5 | 76.5 | 4.6 | 31.8 | 41.1 | 0.0 | 264.3 | 0.0 | nydusd-amp/14137/state_machine |

## Top processes in the window, cores per node

- 6wcf: nydus-dragonfly 5.32, nydusd-amp 1.35, objectstore-plu 0.48, ateworker-gviso 0.45, containerd-nydu 0.21, fluent-bit 0.15, kubelet 0.08, containerd-shim 0.08
- lzgj: nydus-dragonfly 4.02, nydusd-amp 0.97, objectstore-plu 0.34, ateworker-gviso 0.28, containerd-nydu 0.15, fluent-bit 0.13, kworker 0.08, kubelet 0.08
- v6sh: nydus-dragonfly 4.59, nydusd-amp 1.05, objectstore-plu 0.42, ateworker-gviso 0.30, containerd-nydu 0.18, kubelet 0.12, fluent-bit 0.11, containerd 0.08

## Timeline, node 6wcf

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 53 | 0.40 | 0 | 9 | 1 | 0.1 | 0.2 | 93 | 202.5 |
| 1 | 19 | 19 | 431 | 8302 | 1.09 | 123 | 54 | 5 | 15.7 | 17.1 | 198 | 1.0 |
| 2 | 2 | 172 | 142 | 24240 | 0.39 | 56 | 80 | 1 | 35.7 | 35.1 | 202 | 0.0 |
| 3 | 0 | 161 | 218 | 29571 | 0.39 | 42 | 65 | 5 | 43.6 | 26.8 | 179 | 0.0 |
| 4 | 0 | 61 | 172 | 20971 | 0.37 | 26 | 26 | 5 | 37.4 | 2.2 | 91 | 0.0 |

## Timeline, node lzgj

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 10 | 540 | 5855 | 1.75 | 92 | 30 | 1 | 9.5 | 5.8 | 81 | 172.0 |
| 1 | 14 | 144 | 152 | 23294 | 0.40 | 52 | 75 | 1 | 35.5 | 23.8 | 216 | 0.8 |
| 2 | 0 | 135 | 205 | 23689 | 0.41 | 30 | 45 | 4 | 38.1 | 13.8 | 228 | 0.0 |
| 3 | 0 | 33 | 254 | 11606 | 0.61 | 12 | 15 | 3 | 21.4 | 1.1 | 66 | 0.0 |

## Timeline, node v6sh

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 1 | 14 | 361 | 6518 | 0.89 | 116 | 37 | 2 | 12.7 | 9.4 | 82 | 325.7 |
| 1 | 15 | 149 | 149 | 22218 | 0.45 | 44 | 76 | 2 | 34.8 | 31.8 | 174 | 1.0 |
| 3 | 0 | 91 | 191 | 24583 | 0.42 | 38 | 51 | 5 | 41.1 | 16.5 | 134 | 0.0 |
| 4 | 0 | 73 | 133 | 21750 | 0.39 | 26 | 28 | 4 | 38.3 | 1.7 | 52 | 0.0 |

## Seed requests (nginx)

1637 requests in the window, 509 MB. Seed time p50 / p99 / max: 0.0 / 123.0 / 414.0 ms. Size p50 / p99: 179 / 3867 KB. Status: {200: 192, 206: 1445}.
