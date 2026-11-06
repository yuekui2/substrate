# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-n32-none/driver/../b-zstd-n32-r1/rep-1/nydus-ob-w2

96 first turns. TTFI p50 / p95 / p99 / p100: 2.41 / 3.66 / 4.26 / 4.68 s. Window 4.7 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| b7z7 | 32 | 2.74 | 4.68 |
| v197 | 32 | 2.25 | 3.21 |
| wn9j | 32 | 2.51 | 4.24 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-012 | b7z7 | 4.68 | 0.08 | 0.57 | 0.14 | 1.27 | 3.20 | 0.28 | 0.28 | 0.04 | 0.23 | 2.35 |
| r1-076 | wn9j | 4.24 | 0.08 | 0.79 | 0.21 | 1.38 | 2.65 | 0.24 | 0.04 | 0.07 | 0.44 | 1.82 |
| r1-003 | b7z7 | 4.10 | 0.07 | 0.59 | 0.16 | 1.29 | 2.62 | 0.26 | 0.04 | 0.23 | 0.30 | 1.77 |
| r1-092 | wn9j | 3.70 | 0.08 | 0.80 | 0.16 | 1.38 | 2.12 | 0.29 | 0.11 | 0.33 | 0.12 | 1.23 |
| r1-056 | b7z7 | 3.68 | 0.08 | 0.57 | 0.18 | 1.27 | 2.20 | 0.28 | 0.26 | 0.05 | 0.26 | 1.33 |
| r1-019 | wn9j | 3.65 | 0.08 | 0.72 | 0.24 | 1.39 | 2.06 | 0.25 | 0.10 | 0.04 | 0.39 | 1.24 |
| r1-046 | b7z7 | 3.59 | 0.07 | 0.87 | 0.17 | 1.27 | 2.12 | 0.26 | 0.06 | 0.21 | 0.32 | 1.25 |
| r1-075 | b7z7 | 3.45 | 0.08 | 0.61 | 0.20 | 1.27 | 1.99 | 0.28 | 0.08 | 0.18 | 0.28 | 1.11 |
| r1-041 | wn9j | 3.43 | 0.09 | 0.73 | 0.19 | 1.37 | 1.85 | 0.28 | 0.10 | 0.04 | 0.12 | 0.96 |
| r1-062 | wn9j | 3.38 | 0.06 | 0.79 | 0.16 | 1.33 | 1.85 | 0.27 | 0.05 | 0.03 | 0.41 | 1.01 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| b7z7 | 152.5 | 708.1 | 23888.8 | 1.4 | 11.7 | 214.8 | 0.9 | 56.5 | 59.4 | 17.0 | 44.3 | 43.9 | 0.0 | 455.3 | 0.3 | containerd-nydu/18404/containerd-nydu |
| v197 | 162.1 | 596.7 | 26872.3 | 2.3 | 11.7 | 117.5 | 0.0 | 62.8 | 71.4 | 29.5 | 52.5 | 47.7 | 0.0 | 374.8 | 0.8 | containerd-nydu/17978/containerd-nydu |
| wn9j | 150.5 | 607.2 | 23530.2 | 1.6 | 10.5 | 222.7 | 7.5 | 60.0 | 68.0 | 20.6 | 48.5 | 39.9 | 0.0 | 453.4 | 8.7 | containerd-nydu/15528/containerd-nydu |

## Top processes in the window, cores per node

- b7z7: nydusd-amp 1.75, nydus-dragonfly 0.73, objectstore-plu 0.50, ateworker-gviso 0.46, containerd-nydu 0.19, fluent-bit 0.14, atelet 0.09, containerd 0.08
- v197: nydusd-amp 1.98, nydus-dragonfly 0.62, objectstore-plu 0.48, ateworker-gviso 0.42, containerd-nydu 0.20, fluent-bit 0.16, kubelet 0.10, atelet 0.09
- wn9j: nydusd-amp 1.71, nydus-dragonfly 0.76, objectstore-plu 0.49, ateworker-gviso 0.48, containerd-nydu 0.20, fluent-bit 0.15, kubelet 0.09, atelet 0.09

## Timeline, node b7z7

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 73 | 341 | 1.34 | 0 | 5 | 1 | 1.2 | 0.6 | 107 | 109.8 |
| 1 | 0 | 0 | 708 | 5234 | 1.43 | 215 | 51 | 6 | 9.4 | 9.0 | 38 | 286.9 |
| 2 | 32 | 70 | 221 | 5059 | 0.78 | 27 | 47 | 17 | 12.3 | 25.9 | 235 | 1.0 |
| 3 | 7 | 152 | 325 | 16809 | 0.52 | 69 | 59 | 3 | 31.4 | 44.3 | 366 | 1.0 |
| 4 | 2 | 123 | 373 | 23889 | 0.48 | 68 | 50 | 9 | 43.9 | 17.7 | 327 | 0.0 |
| 5 | 0 | 85 | 363 | 27847 | 0.44 | 43 | 39 | 7 | 47.8 | 7.5 | 225 | 0.0 |

## Timeline, node v197

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 123 | 2316 | 0.78 | 117 | 25 | 2 | 4.2 | 4.4 | 130 | 147.8 |
| 1 | 10 | 23 | 597 | 4064 | 2.25 | 90 | 63 | 29 | 13.6 | 18.9 | 77 | 1.0 |
| 2 | 26 | 162 | 136 | 12254 | 0.51 | 48 | 71 | 2 | 24.0 | 52.5 | 217 | 1.0 |
| 4 | 0 | 103 | 269 | 21131 | 0.47 | 58 | 55 | 6 | 35.9 | 35.2 | 278 | 0.0 |
| 5 | 0 | 52 | 247 | 26872 | 0.43 | 47 | 35 | 9 | 47.7 | 4.1 | 224 | 0.0 |
| 6 | 0 | 109 | 456 | 31073 | 0.60 | 42 | 37 | 7 | 53.6 | 3.5 | 115 | 0.0 |

## Timeline, node wn9j

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 63 | 0.47 | 0 | 4 | 1 | 0.6 | 0.5 | 112 | 302.5 |
| 1 | 0 | 0 | 607 | 4715 | 1.64 | 223 | 52 | 3 | 9.8 | 15.9 | 38 | 312.8 |
| 2 | 32 | 94 | 204 | 5585 | 0.72 | 31 | 61 | 21 | 12.2 | 24.3 | 220 | 4.2 |
| 3 | 9 | 151 | 251 | 17118 | 0.46 | 62 | 68 | 4 | 28.5 | 48.5 | 323 | 0.0 |
| 4 | 1 | 91 | 298 | 23530 | 0.44 | 66 | 51 | 9 | 39.9 | 22.3 | 314 | 0.0 |
| 5 | 0 | 67 | 256 | 28366 | 0.42 | 43 | 37 | 10 | 48.3 | 5.8 | 264 | 0.0 |

## Seed requests (nginx)

3509 requests in the window, 1010 MB. Seed time p50 / p99 / max: 0.0 / 126.9 / 399.0 ms. Size p50 / p99: 178 / 3851 KB. Status: {200: 384, 206: 3125}.
