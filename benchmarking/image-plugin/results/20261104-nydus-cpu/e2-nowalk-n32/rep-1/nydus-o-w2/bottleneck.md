# Bottleneck: benchmarking/image-plugin/results/20261104-nydus-cpu/e2-nowalk-n32/rep-1/nydus-o-w2

96 first turns. TTFI p50 / p95 / p99 / p100: 2.23 / 3.38 / 3.89 / 4.13 s. Window 4.1 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| cvf8 | 32 | 2.21 | 4.13 |
| n78x | 32 | 2.12 | 3.36 |
| zc3w | 32 | 2.51 | 3.54 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-079 | cvf8 | 4.13 | 0.07 | 0.56 | 0.60 | 1.25 | 2.71 | 0.23 | 0.19 | 0.05 | 0.20 | 2.02 |
| r1-025 | cvf8 | 3.88 | 0.08 | 0.41 | 0.70 | 1.21 | 2.46 | 0.23 | 0.19 | 0.05 | 0.20 | 1.77 |
| r1-094 | cvf8 | 3.71 | 0.07 | 0.39 | 0.61 | 1.09 | 2.29 | 0.23 | 0.16 | 0.02 | 0.10 | 1.76 |
| r1-074 | zc3w | 3.54 | 0.08 | 0.37 | 0.15 | 1.30 | 2.04 | 0.06 | 0.24 | 0.25 | 0.12 | 1.27 |
| r1-046 | zc3w | 3.43 | 0.07 | 0.37 | 0.17 | 1.28 | 1.94 | 0.03 | 0.06 | 0.20 | 0.29 | 1.33 |
| r1-044 | n78x | 3.36 | 0.08 | 0.43 | 0.38 | 1.03 | 2.07 | 0.22 | 0.27 | 0.03 | 0.22 | 1.30 |
| r1-035 | zc3w | 3.35 | 0.07 | 0.41 | 0.18 | 1.15 | 2.00 | 0.17 | 0.07 | 0.18 | 0.26 | 1.25 |
| r1-063 | cvf8 | 3.35 | 0.07 | 0.53 | 0.51 | 1.40 | 1.75 | 0.05 | 0.18 | 0.05 | 0.20 | 1.22 |
| r1-059 | zc3w | 3.27 | 0.07 | 0.39 | 0.18 | 1.17 | 1.94 | 0.18 | 0.25 | 0.28 | 0.12 | 1.09 |
| r1-024 | n78x | 3.21 | 0.08 | 0.54 | 0.35 | 1.21 | 1.83 | 0.12 | 0.26 | 0.03 | 0.21 | 1.15 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| cvf8 | 176.5 | 752.3 | 23545.0 | 2.3 | 12.2 | 146.0 | 1.9 | 56.4 | 61.7 | 8.0 | 31.4 | 40.4 | 0.0 | 496.3 | 0.7 | nydusd-amp/15091/state_machine |
| n78x | 151.2 | 991.0 | 28805.6 | 2.7 | 16.9 | 186.6 | 0.9 | 59.5 | 64.1 | 13.9 | 44.2 | 48.4 | 0.0 | 472.4 | 0.2 | containerd-nydu/15653/containerd-nydu |
| zc3w | 112.0 | 404.8 | 24764.9 | 1.4 | 13.7 | 206.1 | 0.0 | 53.5 | 59.1 | 15.8 | 42.4 | 49.4 | 0.0 | 421.7 | 1.2 | containerd-nydu/14389/containerd-nydu |

## Top processes in the window, cores per node

- cvf8: nydusd-amp 1.83, nydus-dragonfly 0.99, objectstore-plu 0.60, ateworker-gviso 0.42, containerd-nydu 0.27, fluent-bit 0.17, kubelet 0.10, atelet 0.09
- n78x: nydusd-amp 1.93, nydus-dragonfly 0.78, ateworker-gviso 0.49, objectstore-plu 0.49, containerd-nydu 0.20, fluent-bit 0.15, kworker 0.10, atelet 0.08
- zc3w: nydusd-amp 1.80, nydus-dragonfly 0.85, ateworker-gviso 0.69, objectstore-plu 0.62, containerd-nydu 0.26, fluent-bit 0.15, atelet 0.10, cilium-agent 0.08

## Timeline, node cvf8

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 27 | 1748 | 0.45 | 92 | 25 | 2 | 4.2 | 7.7 | 130 | 463.9 |
| 1 | 20 | 41 | 752 | 5069 | 2.33 | 146 | 55 | 6 | 27.7 | 23.1 | 81 | 2.6 |
| 2 | 21 | 136 | 203 | 15769 | 0.46 | 55 | 55 | 2 | 28.5 | 30.4 | 281 | 0.2 |
| 4 | 1 | 177 | 332 | 23545 | 0.47 | 75 | 62 | 8 | 40.4 | 31.4 | 262 | 0.0 |
| 5 | 0 | 69 | 297 | 27258 | 0.43 | 51 | 36 | 8 | 48.9 | 5.6 | 197 | 0.0 |

## Timeline, node n78x

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 19 | 1655 | 0.40 | 44 | 18 | 3 | 2.8 | 4.1 | 132 | 494.6 |
| 1 | 17 | 9 | 991 | 6080 | 2.75 | 187 | 62 | 14 | 26.3 | 25.5 | 107 | 6.9 |
| 2 | 19 | 151 | 193 | 13302 | 0.47 | 62 | 64 | 2 | 22.9 | 44.2 | 256 | 1.0 |
| 3 | 2 | 106 | 344 | 22986 | 0.46 | 80 | 52 | 6 | 40.7 | 25.5 | 417 | 0.0 |
| 5 | 0 | 72 | 248 | 28806 | 0.41 | 44 | 37 | 8 | 48.4 | 5.2 | 160 | 0.0 |
| 6 | 0 | 71 | 476 | 16832 | 0.71 | 24 | 30 | 3 | 30.8 | 5.0 | 7 | 0.0 |

## Timeline, node zc3w

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 52 | 0.44 | 0 | 2 | 1 | 0.3 | 0.1 | 129 | 268.2 |
| 1 | 9 | 0 | 405 | 3830 | 1.40 | 206 | 44 | 2 | 15.1 | 12.8 | 27 | 0.0 |
| 2 | 32 | 50 | 278 | 3578 | 1.39 | 15 | 59 | 16 | 14.3 | 20.7 | 191 | 1.1 |
| 3 | 6 | 112 | 191 | 11000 | 0.60 | 49 | 59 | 2 | 24.9 | 42.4 | 275 | 1.0 |
| 4 | 0 | 104 | 315 | 24765 | 0.55 | 68 | 43 | 8 | 49.4 | 10.1 | 276 | 0.0 |
| 5 | 0 | 99 | 264 | 29427 | 0.47 | 37 | 40 | 8 | 53.3 | 6.5 | 199 | 0.0 |

## Seed requests (nginx)

3289 requests in the window, 1013 MB. Seed time p50 / p99 / max: 0.0 / 220.6 / 540.0 ms. Size p50 / p99: 178 / 3843 KB. Status: {200: 384, 206: 2905}.
