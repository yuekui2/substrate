# Bottleneck: benchmarking/image-plugin/results/20261104-nydus-cpu/e2-nowalk-n32-r3/rep-1/nydus-o-w2

96 first turns. TTFI p50 / p95 / p99 / p100: 2.25 / 3.44 / 3.79 / 4.01 s. Window 4.0 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 0v83 | 32 | 2.22 | 4.01 |
| nhnj | 32 | 2.17 | 3.28 |
| sqll | 32 | 2.43 | 3.78 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-074 | 0v83 | 4.01 | 0.07 | 0.57 | 0.21 | 1.11 | 2.73 | 0.03 | 0.07 | 0.08 | 0.50 | 2.02 |
| r1-086 | sqll | 3.78 | 0.08 | 0.58 | 0.29 | 1.24 | 2.34 | 0.29 | 0.14 | 0.04 | 0.05 | 1.77 |
| r1-024 | 0v83 | 3.63 | 0.08 | 0.56 | 0.18 | 1.02 | 2.36 | 0.03 | 0.17 | 0.07 | 0.10 | 1.60 |
| r1-025 | sqll | 3.56 | 0.08 | 0.66 | 0.42 | 1.12 | 2.25 | 0.06 | 0.46 | 0.02 | 0.08 | 1.56 |
| r1-010 | sqll | 3.44 | 0.08 | 0.58 | 0.47 | 1.09 | 2.16 | 0.06 | 0.05 | 0.35 | 0.10 | 1.55 |
| r1-048 | 0v83 | 3.44 | 0.07 | 0.47 | 0.23 | 1.02 | 2.25 | 0.10 | 0.15 | 0.06 | 0.44 | 1.46 |
| r1-094 | nhnj | 3.28 | 0.06 | 0.54 | 0.30 | 1.17 | 1.94 | 0.29 | 0.11 | 0.10 | 0.15 | 1.28 |
| r1-063 | nhnj | 3.24 | 0.07 | 0.66 | 0.16 | 1.14 | 1.91 | 0.29 | 0.10 | 0.03 | 0.17 | 1.23 |
| r1-049 | 0v83 | 3.23 | 0.08 | 0.46 | 0.23 | 0.95 | 2.09 | 0.15 | 0.08 | 0.08 | 0.51 | 1.24 |
| r1-064 | sqll | 3.22 | 0.07 | 0.58 | 0.27 | 1.09 | 1.94 | 0.06 | 0.07 | 0.34 | 0.13 | 1.31 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0v83 | 149.5 | 582.0 | 23857.4 | 2.3 | 12.2 | 185.9 | 2.9 | 53.5 | 61.4 | 10.0 | 40.8 | 42.6 | 0.0 | 498.0 | 6.3 | containerd-nydu/14717/containerd-nydu |
| nhnj | 144.6 | 696.7 | 27709.6 | 2.1 | 17.0 | 188.4 | 6.7 | 55.2 | 65.1 | 8.1 | 40.8 | 51.6 | 0.0 | 647.0 | 0.4 | nydusd-amp/14821/state_machine |
| sqll | 172.3 | 563.8 | 23731.2 | 1.9 | 10.7 | 162.3 | 0.0 | 52.0 | 76.0 | 20.5 | 33.4 | 41.1 | 0.0 | 477.6 | 0.4 | containerd-nydu/16366/containerd-nydu |

## Top processes in the window, cores per node

- 0v83: nydusd-amp 1.83, nydus-dragonfly 1.05, ateworker-gviso 0.81, objectstore-plu 0.59, containerd-nydu 0.28, fluent-bit 0.16, kubelet 0.11, kworker 0.10
- nhnj: nydusd-amp 1.89, nydus-dragonfly 0.68, objectstore-plu 0.50, ateworker-gviso 0.47, containerd-nydu 0.18, fluent-bit 0.14, kworker 0.11, atelet 0.08
- sqll: nydusd-amp 1.84, nydus-dragonfly 0.89, objectstore-plu 0.59, ateworker-gviso 0.54, containerd-nydu 0.25, fluent-bit 0.16, atelet 0.10, cilium-agent 0.08

## Timeline, node 0v83

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 222 | 2775 | 1.28 | 186 | 41 | 2 | 6.9 | 17.0 | 135 | 514.9 |
| 1 | 31 | 20 | 582 | 4520 | 2.32 | 59 | 61 | 10 | 15.3 | 23.3 | 43 | 1.6 |
| 3 | 6 | 149 | 408 | 14549 | 0.83 | 52 | 58 | 2 | 27.5 | 40.8 | 345 | 1.0 |
| 4 | 0 | 136 | 358 | 23857 | 0.46 | 80 | 49 | 8 | 42.6 | 15.9 | 288 | 0.0 |
| 5 | 0 | 90 | 260 | 30074 | 0.40 | 40 | 40 | 7 | 49.3 | 4.0 | 191 | 0.0 |

## Timeline, node nhnj

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 16 | 1445 | 0.37 | 20 | 13 | 2 | 2.1 | 1.4 | 126 | 267.2 |
| 1 | 31 | 6 | 697 | 4411 | 2.14 | 188 | 65 | 5 | 29.0 | 22.2 | 91 | 5.0 |
| 2 | 17 | 145 | 165 | 11226 | 0.53 | 57 | 62 | 2 | 27.4 | 40.8 | 262 | 1.4 |
| 3 | 2 | 114 | 512 | 23893 | 0.62 | 68 | 57 | 5 | 42.2 | 30.0 | 356 | 1.0 |
| 4 | 0 | 79 | 501 | 27710 | 0.61 | 46 | 38 | 8 | 51.6 | 4.4 | 277 | 0.0 |

## Timeline, node sqll

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 220 | 3193 | 1.11 | 162 | 36 | 4 | 10.0 | 10.5 | 133 | 484.8 |
| 1 | 32 | 40 | 564 | 4674 | 1.93 | 67 | 76 | 20 | 27.1 | 22.1 | 63 | 1.0 |
| 3 | 5 | 172 | 216 | 15144 | 0.44 | 68 | 56 | 1 | 26.7 | 33.4 | 327 | 1.0 |
| 4 | 0 | 112 | 333 | 23731 | 0.44 | 67 | 55 | 7 | 41.1 | 28.7 | 245 | 0.0 |
| 5 | 0 | 71 | 244 | 30189 | 0.38 | 46 | 35 | 7 | 48.9 | 5.2 | 199 | 0.0 |

## Seed requests (nginx)

3221 requests in the window, 993 MB. Seed time p50 / p99 / max: 0.0 / 165.8 / 632.0 ms. Size p50 / p99: 178 / 3852 KB. Status: {200: 380, 206: 2841}.
