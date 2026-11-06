# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-n32-none/driver/../c-none-n32-r1/rep-1/nydus-nonec-w2

96 first turns. TTFI p50 / p95 / p99 / p100: 2.26 / 3.12 / 3.37 / 3.45 s. Window 3.5 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 0gs3 | 32 | 2.27 | 3.45 |
| 4p32 | 32 | 2.22 | 2.85 |
| fv2p | 32 | 2.27 | 3.23 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-021 | 0gs3 | 3.45 | 0.04 | 0.73 | 0.28 | 1.15 | 2.14 | 0.04 | 0.05 | 0.06 | 0.54 | 1.42 |
| r1-001 | 0gs3 | 3.36 | 0.05 | 0.52 | 0.48 | 1.02 | 2.20 | 0.16 | 0.08 | 0.03 | 0.56 | 1.35 |
| r1-073 | 0gs3 | 3.32 | 0.06 | 0.77 | 0.35 | 1.15 | 2.00 | 0.04 | 0.07 | 0.04 | 0.46 | 1.36 |
| r1-093 | 0gs3 | 3.25 | 0.05 | 0.72 | 0.38 | 1.13 | 1.95 | 0.02 | 0.05 | 0.04 | 0.50 | 1.30 |
| r1-004 | fv2p | 3.23 | 0.05 | 0.76 | 0.52 | 1.24 | 1.81 | 0.20 | 0.27 | 0.03 | 0.14 | 1.14 |
| r1-068 | fv2p | 3.09 | 0.06 | 0.50 | 0.29 | 1.29 | 1.63 | 0.17 | 0.29 | 0.01 | 0.13 | 0.97 |
| r1-082 | fv2p | 3.08 | 0.06 | 0.58 | 0.69 | 1.20 | 1.67 | 0.20 | 0.28 | 0.03 | 0.16 | 0.95 |
| r1-018 | fv2p | 2.96 | 0.06 | 0.81 | 0.43 | 1.28 | 1.53 | 0.22 | 0.26 | 0.02 | 0.15 | 0.86 |
| r1-041 | 0gs3 | 2.96 | 0.05 | 0.72 | 0.38 | 0.94 | 1.63 | 0.04 | 0.06 | 0.07 | 0.45 | 1.00 |
| r1-033 | fv2p | 2.95 | 0.05 | 0.54 | 0.55 | 0.86 | 1.69 | 0.10 | 0.06 | 0.16 | 0.34 | 0.99 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0gs3 | 195.9 | 641.2 | 29322.2 | 1.8 | 19.8 | 269.9 | 14.8 | 48.2 | 75.4 | 13.0 | 28.9 | 50.7 | 0.0 | 744.2 | 1.4 | containerd-nydu/17020/containerd-nydu |
| 4p32 | 147.1 | 455.7 | 21751.4 | 1.1 | 9.7 | 182.7 | 0.9 | 62.0 | 65.0 | 3.6 | 53.5 | 32.4 | 0.0 | 508.5 | 28.0 | nydusd-amp/18924/reqwest-interna |
| fv2p | 139.9 | 967.5 | 28094.1 | 2.9 | 15.9 | 259.6 | 0.9 | 58.6 | 72.3 | 6.5 | 42.0 | 41.3 | 0.0 | 378.9 | 0.5 | containerd-nydu/16978/containerd-nydu |

## Top processes in the window, cores per node

- 0gs3: nydusd-amp 1.30, nydus-dragonfly 0.82, objectstore-plu 0.59, ateworker-gviso 0.46, containerd-nydu 0.24, fluent-bit 0.20, atelet 0.10, kubelet 0.09
- 4p32: nydusd-amp 0.91, nydus-dragonfly 0.85, objectstore-plu 0.61, ateworker-gviso 0.44, containerd-nydu 0.24, fluent-bit 0.17, kubelet 0.10, atelet 0.09
- fv2p: nydusd-amp 1.15, nydus-dragonfly 0.95, objectstore-plu 0.59, ateworker-gviso 0.44, containerd-nydu 0.25, fluent-bit 0.16, atelet 0.09, kubelet 0.08

## Timeline, node 0gs3

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 143 | 2578 | 0.90 | 131 | 33 | 2 | 5.2 | 7.5 | 129 | 203.3 |
| 1 | 28 | 3 | 525 | 3830 | 1.75 | 76 | 75 | 13 | 18.0 | 25.6 | 32 | 3.7 |
| 3 | 4 | 174 | 342 | 15489 | 0.56 | 175 | 51 | 2 | 29.1 | 28.9 | 394 | 1.0 |
| 4 | 0 | 196 | 641 | 29322 | 0.67 | 270 | 50 | 4 | 50.7 | 25.5 | 251 | 0.0 |
| 5 | 0 | 90 | 260 | 28138 | 0.43 | 124 | 32 | 5 | 46.0 | 5.3 | 123 | 0.0 |

## Timeline, node 4p32

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 2 | 91 | 0.71 | 0 | 5 | 1 | 0.3 | 0.6 | 131 | 247.0 |
| 1 | 21 | 0 | 456 | 4401 | 1.14 | 179 | 49 | 2 | 9.9 | 10.6 | 89 | 19.2 |
| 2 | 22 | 129 | 382 | 6596 | 1.10 | 124 | 61 | 2 | 13.8 | 26.8 | 289 | 1.1 |
| 3 | 0 | 147 | 274 | 21751 | 0.43 | 183 | 65 | 4 | 32.4 | 53.5 | 326 | 0.0 |
| 4 | 0 | 79 | 344 | 35319 | 0.42 | 192 | 35 | 4 | 54.6 | 7.4 | 262 | 0.0 |

## Timeline, node fv2p

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 144 | 1940 | 0.70 | 22 | 12 | 1 | 3.0 | 1.5 | 130 | 329.5 |
| 1 | 24 | 1 | 967 | 5416 | 2.90 | 199 | 72 | 7 | 14.2 | 24.8 | 55 | 10.5 |
| 2 | 27 | 140 | 259 | 8981 | 0.56 | 116 | 50 | 2 | 17.2 | 34.0 | 229 | 2.0 |
| 3 | 3 | 140 | 447 | 28094 | 0.46 | 260 | 64 | 3 | 41.3 | 42.0 | 447 | 1.5 |
| 4 | 0 | 111 | 484 | 39708 | 0.44 | 181 | 39 | 4 | 58.2 | 7.3 | 331 | 0.0 |

## Seed requests (nginx)

2960 requests in the window, 1923 MB. Seed time p50 / p99 / max: 0.0 / 144.0 / 376.0 ms. Size p50 / p99: 617 / 3744 KB. Status: {200: 384, 206: 2576}.
