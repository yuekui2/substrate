# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-n32-none/driver/../c-zstd-n32-r1/rep-1/nydus-oc-w2

96 first turns. TTFI p50 / p95 / p99 / p100: 2.18 / 3.27 / 3.54 / 3.57 s. Window 3.6 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| hlc0 | 32 | 2.18 | 3.54 |
| jrbz | 32 | 2.18 | 2.99 |
| s1b4 | 32 | 2.16 | 3.57 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-021 | s1b4 | 3.57 | 0.05 | 0.47 | 0.20 | 1.12 | 2.22 | 0.05 | 0.19 | 0.03 | 0.17 | 1.76 |
| r1-067 | hlc0 | 3.54 | 0.06 | 0.39 | 0.11 | 1.23 | 2.15 | 0.04 | 0.10 | 0.29 | 0.26 | 1.43 |
| r1-027 | hlc0 | 3.51 | 0.06 | 0.32 | 0.12 | 1.04 | 2.29 | 0.21 | 0.12 | 0.30 | 0.24 | 1.41 |
| r1-082 | hlc0 | 3.41 | 0.06 | 0.38 | 0.16 | 1.05 | 2.21 | 0.22 | 0.06 | 0.06 | 0.31 | 1.52 |
| r1-094 | s1b4 | 3.35 | 0.05 | 0.52 | 0.41 | 1.01 | 2.13 | 0.20 | 0.21 | 0.12 | 0.26 | 1.32 |
| r1-018 | hlc0 | 3.24 | 0.08 | 0.30 | 0.30 | 1.04 | 2.03 | 0.21 | 0.08 | 0.04 | 0.36 | 1.31 |
| r1-014 | hlc0 | 3.24 | 0.06 | 0.36 | 0.13 | 1.03 | 2.04 | 0.23 | 0.12 | 0.25 | 0.26 | 1.14 |
| r1-004 | s1b4 | 3.20 | 0.08 | 0.48 | 0.22 | 1.03 | 1.98 | 0.18 | 0.19 | 0.15 | 0.24 | 1.16 |
| r1-038 | s1b4 | 3.12 | 0.06 | 0.49 | 0.23 | 1.04 | 1.90 | 0.20 | 0.20 | 0.14 | 0.26 | 1.07 |
| r1-011 | hlc0 | 3.10 | 0.06 | 0.46 | 0.14 | 1.00 | 1.93 | 0.26 | 0.39 | 0.05 | 0.08 | 0.88 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| hlc0 | 170.0 | 502.9 | 23584.6 | 1.4 | 12.1 | 220.6 | 1.0 | 52.5 | 58.4 | 20.9 | 33.4 | 44.4 | 0.0 | 475.9 | 0.3 | containerd-nydu/15229/containerd-nydu |
| jrbz | 184.4 | 390.0 | 23413.8 | 1.4 | 10.4 | 199.3 | 5.4 | 64.4 | 68.3 | 27.3 | 50.6 | 42.3 | 0.0 | 422.0 | 0.0 | containerd-nydu/14437/containerd-nydu |
| s1b4 | 146.8 | 717.9 | 24064.3 | 1.9 | 11.9 | 153.3 | 0.9 | 53.2 | 61.8 | 12.0 | 37.8 | 44.1 | 0.0 | 431.4 | 0.0 | containerd-nydu/15484/containerd-nydu |

## Top processes in the window, cores per node

- hlc0: nydusd-amp 1.88, nydus-dragonfly 1.00, objectstore-plu 0.60, ateworker-gviso 0.57, containerd-nydu 0.24, fluent-bit 0.22, kubelet 0.12, atelet 0.12
- jrbz: nydusd-amp 1.64, nydus-dragonfly 0.86, objectstore-plu 0.62, ateworker-gviso 0.50, containerd-nydu 0.24, fluent-bit 0.19, atelet 0.10, kubelet 0.08
- s1b4: nydusd-amp 1.68, nydus-dragonfly 0.87, ateworker-gviso 0.63, objectstore-plu 0.59, containerd-nydu 0.24, fluent-bit 0.20, atelet 0.10, kubelet 0.09

## Timeline, node hlc0

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 503 | 4382 | 1.38 | 221 | 46 | 2 | 8.0 | 11.1 | 129 | 305.4 |
| 2 | 22 | 52 | 343 | 3654 | 1.23 | 31 | 43 | 21 | 10.1 | 22.6 | 275 | 1.0 |
| 3 | 7 | 170 | 254 | 18557 | 0.49 | 67 | 58 | 2 | 34.3 | 33.4 | 397 | 1.0 |
| 4 | 0 | 155 | 383 | 23585 | 0.50 | 69 | 50 | 6 | 44.4 | 21.3 | 212 | 0.0 |
| 5 | 0 | 72 | 179 | 22020 | 0.42 | 34 | 27 | 5 | 43.5 | 4.9 | 130 | 0.0 |

## Timeline, node jrbz

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 390 | 3536 | 1.43 | 199 | 42 | 3 | 8.3 | 9.0 | 133 | 260.2 |
| 1 | 23 | 8 | 356 | 3178 | 1.18 | 23 | 47 | 27 | 7.8 | 19.9 | 52 | 1.0 |
| 3 | 0 | 184 | 217 | 14606 | 0.49 | 63 | 68 | 1 | 23.8 | 50.6 | 268 | 0.3 |
| 4 | 0 | 106 | 258 | 23414 | 0.44 | 62 | 50 | 6 | 42.3 | 23.4 | 270 | 0.0 |
| 5 | 0 | 53 | 263 | 28594 | 0.45 | 44 | 36 | 9 | 50.5 | 6.7 | 228 | 0.0 |

## Timeline, node s1b4

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 22 | 1640 | 0.46 | 66 | 20 | 1 | 5.3 | 5.9 | 132 | 379.0 |
| 1 | 17 | 27 | 718 | 4969 | 1.87 | 153 | 61 | 12 | 20.4 | 27.6 | 75 | 1.3 |
| 2 | 20 | 147 | 159 | 14485 | 0.46 | 44 | 62 | 1 | 26.4 | 37.8 | 234 | 1.0 |
| 3 | 4 | 122 | 318 | 24064 | 0.48 | 63 | 58 | 7 | 44.1 | 29.1 | 285 | 1.2 |
| 5 | 0 | 75 | 257 | 28779 | 0.43 | 45 | 34 | 6 | 51.5 | 4.5 | 187 | 0.0 |

## Seed requests (nginx)

2819 requests in the window, 920 MB. Seed time p50 / p99 / max: 0.0 / 194.6 / 481.0 ms. Size p50 / p99: 186 / 3861 KB. Status: {200: 384, 206: 2435}.
