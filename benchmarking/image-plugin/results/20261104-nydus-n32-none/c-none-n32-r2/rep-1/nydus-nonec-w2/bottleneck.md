# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-n32-none/driver/../c-none-n32-r2/rep-1/nydus-nonec-w2

96 first turns. TTFI p50 / p95 / p99 / p100: 2.25 / 2.80 / 2.95 / 3.08 s. Window 3.1 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 1qpg | 32 | 2.29 | 2.94 |
| 2h1l | 32 | 2.18 | 3.08 |
| dcc8 | 32 | 2.22 | 2.72 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-004 | 2h1l | 3.08 | 0.09 | 0.59 | 0.63 | 0.90 | 1.61 | 0.07 | 0.05 | 0.09 | 0.21 | 1.15 |
| r1-017 | 1qpg | 2.94 | 0.08 | 0.43 | 0.57 | 0.72 | 1.57 | 0.07 | 0.05 | 0.18 | 0.34 | 0.89 |
| r1-077 | 1qpg | 2.89 | 0.06 | 0.94 | 0.35 | 0.90 | 1.42 | 0.08 | 0.06 | 0.13 | 0.39 | 0.73 |
| r1-027 | 1qpg | 2.86 | 0.07 | 0.92 | 0.42 | 0.87 | 1.33 | 0.13 | 0.09 | 0.21 | 0.16 | 0.71 |
| r1-033 | 2h1l | 2.85 | 0.09 | 1.03 | 0.23 | 1.02 | 1.34 | 0.08 | 0.07 | 0.02 | 0.30 | 0.84 |
| r1-036 | 1qpg | 2.78 | 0.06 | 0.86 | 0.43 | 0.90 | 1.34 | 0.01 | 0.04 | 0.24 | 0.39 | 0.63 |
| r1-001 | 2h1l | 2.72 | 0.06 | 1.20 | 0.19 | 0.98 | 1.18 | 0.11 | 0.21 | 0.07 | 0.14 | 0.60 |
| r1-061 | dcc8 | 2.72 | 0.07 | 1.06 | 0.16 | 0.89 | 1.32 | 0.01 | 0.13 | 0.06 | 0.14 | 0.90 |
| r1-082 | 2h1l | 2.66 | 0.08 | 1.10 | 0.28 | 0.98 | 1.12 | 0.12 | 0.29 | 0.07 | 0.09 | 0.50 |
| r1-013 | 1qpg | 2.66 | 0.06 | 0.82 | 0.49 | 0.88 | 1.17 | 0.13 | 0.07 | 0.03 | 0.23 | 0.64 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1qpg | 183.1 | 610.8 | 16073.9 | 1.8 | 8.6 | 203.8 | 0.0 | 59.8 | 63.5 | 4.7 | 42.1 | 29.0 | 0.0 | 488.3 | 0.4 | containerd-nydu/16460/containerd-nydu |
| 2h1l | 141.6 | 494.2 | 17200.4 | 1.4 | 8.2 | 160.9 | 2.8 | 62.6 | 68.0 | 3.8 | 54.7 | 28.4 | 0.0 | 412.2 | 0.0 | containerd-nydu/16019/containerd-nydu |
| dcc8 | 185.1 | 424.4 | 21252.8 | 1.1 | 9.7 | 212.3 | 2.8 | 64.1 | 72.3 | 4.7 | 51.1 | 32.2 | 0.0 | 413.3 | 8.9 | containerd-nydu/17442/containerd-nydu |

## Top processes in the window, cores per node

- 1qpg: nydus-dragonfly 1.25, nydusd-amp 1.10, objectstore-plu 0.83, ateworker-gviso 0.57, containerd-nydu 0.31, fluent-bit 0.23, atelet 0.13, kubelet 0.11
- 2h1l: nydus-dragonfly 1.12, nydusd-amp 1.02, objectstore-plu 0.84, ateworker-gviso 0.55, containerd-nydu 0.32, fluent-bit 0.22, atelet 0.12, kubelet 0.10
- dcc8: nydusd-amp 0.89, nydus-dragonfly 0.82, objectstore-plu 0.62, ateworker-gviso 0.47, containerd-nydu 0.25, fluent-bit 0.19, atelet 0.10, kubelet 0.08

## Timeline, node 1qpg

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 218 | 3179 | 0.86 | 143 | 33 | 2 | 6.0 | 6.5 | 134 | 267.3 |
| 1 | 9 | 11 | 611 | 3711 | 1.81 | 104 | 49 | 5 | 9.9 | 19.2 | 75 | 3.1 |
| 3 | 0 | 183 | 248 | 16074 | 0.53 | 204 | 63 | 2 | 29.0 | 42.1 | 390 | 0.0 |
| 4 | 0 | 167 | 396 | 32547 | 0.45 | 264 | 53 | 4 | 49.7 | 21.7 | 223 | 0.0 |

## Timeline, node 2h1l

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 297 | 3412 | 1.18 | 131 | 30 | 2 | 7.3 | 6.1 | 133 | 202.6 |
| 2 | 25 | 71 | 494 | 4574 | 1.44 | 118 | 57 | 4 | 10.5 | 25.9 | 255 | 1.5 |
| 3 | 1 | 142 | 200 | 17200 | 0.47 | 161 | 68 | 2 | 28.4 | 54.7 | 439 | 0.0 |
| 4 | 0 | 144 | 401 | 37865 | 0.46 | 267 | 51 | 5 | 57.4 | 16.8 | 256 | 0.0 |

## Timeline, node dcc8

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 52 | 0.43 | 0 | 4 | 1 | 0.5 | 0.1 | 132 | 216.6 |
| 1 | 17 | 0 | 424 | 4201 | 1.10 | 129 | 34 | 2 | 8.8 | 5.8 | 129 | 4.4 |
| 2 | 21 | 98 | 384 | 5237 | 1.14 | 149 | 72 | 5 | 11.0 | 28.4 | 283 | 1.2 |
| 3 | 0 | 185 | 283 | 21253 | 0.45 | 212 | 69 | 3 | 32.2 | 51.1 | 378 | 0.2 |
| 4 | 0 | 88 | 377 | 33270 | 0.41 | 212 | 40 | 4 | 50.6 | 12.2 | 275 | 0.0 |

## Seed requests (nginx)

2813 requests in the window, 1868 MB. Seed time p50 / p99 / max: 0.0 / 127.9 / 309.0 ms. Size p50 / p99: 635 / 3747 KB. Status: {200: 384, 206: 2429}.
