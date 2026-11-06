# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-repeats/driver/../placement-2/rep-2/nydus-o-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 0.73 / 1.01 / 1.10 / 1.17 s. Window 1.2 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 9f08 | 14 | 0.44 | 1.17 |
| dgm4 | 16 | 0.82 | 1.03 |
| xtg2 | 18 | 0.59 | 0.94 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r2-003 | 9f08 | 1.17 | 0.04 | 0.70 | 0.21 | 0.20 | 0.17 | 0.00 | 0.02 | 0.01 | 0.07 | 0.04 |
| r2-013 | dgm4 | 1.03 | 0.02 | 0.32 | 0.35 | 0.28 | 0.28 | 0.03 | 0.04 | 0.03 | 0.09 | 0.05 |
| r2-014 | dgm4 | 1.01 | 0.03 | 0.25 | 0.38 | 0.27 | 0.28 | 0.00 | 0.04 | 0.02 | 0.06 | 0.05 |
| r2-041 | dgm4 | 1.00 | 0.02 | 0.24 | 0.34 | 0.29 | 0.32 | 0.01 | 0.04 | 0.01 | 0.12 | 0.04 |
| r2-028 | 9f08 | 0.97 | 0.03 | 0.34 | 0.23 | 0.31 | 0.31 | 0.00 | 0.03 | 0.01 | 0.10 | 0.10 |
| r2-044 | dgm4 | 0.96 | 0.02 | 0.30 | 0.42 | 0.59 | 0.18 | 0.00 | 0.03 | 0.01 | 0.06 | 0.05 |
| r2-017 | xtg2 | 0.94 | 0.02 | 0.25 | 0.36 | 0.32 | 0.24 | 0.01 | 0.06 | 0.01 | 0.07 | 0.03 |
| r2-009 | xtg2 | 0.93 | 0.02 | 0.26 | 0.34 | 0.34 | 0.22 | 0.01 | 0.06 | 0.01 | 0.08 | 0.03 |
| r2-037 | xtg2 | 0.93 | 0.02 | 0.30 | 0.24 | 0.33 | 0.32 | 0.00 | 0.08 | 0.01 | 0.12 | 0.04 |
| r2-025 | 9f08 | 0.88 | 0.02 | 0.30 | 0.20 | 0.48 | 0.26 | 0.02 | 0.03 | 0.01 | 0.07 | 0.11 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 9f08 | — | — | — | — | — | — | — | — | — | — | — | — | — | — | — | — |
| dgm4 | 106.1 | 187.3 | 16203.8 | 0.9 | 8.0 | 37.7 | 0.9 | 61.4 | 69.9 | 11.0 | 37.9 | 26.4 | 0.0 | 132.4 | 0.0 | containerd-nydu/14659/containerd-nydu |
| xtg2 | — | — | — | — | — | — | — | — | — | — | — | — | — | — | — | — |

## Top processes in the window, cores per node

- dgm4: nydus-dragonfly 3.53, nydusd-amp 0.39, objectstore-plu 0.37, ateworker-gviso 0.33, fluent-bit 0.19, containerd-nydu 0.14, atelet 0.09, kworker 0.09

## Timeline, node 9f08

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|

## Timeline, node dgm4

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 16 | 53 | 104 | 1489 | 0.90 | 35 | 25 | 11 | 2.7 | 8.3 | 65 | 199.8 |
| 1 | 1 | 106 | 187 | 16204 | 0.48 | 38 | 70 | 3 | 26.4 | 37.9 | 100 | 1.0 |
| 2 | 0 | 79 | 162 | 15754 | 0.43 | 28 | 37 | 4 | 30.0 | 13.6 | 132 | 0.0 |

## Timeline, node xtg2

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|

## Seed requests (nginx)

450 requests in the window, 172 MB. Seed time p50 / p99 / max: 0.0 / 126.0 / 331.0 ms. Size p50 / p99: 161 / 3852 KB. Status: {200: 100, 206: 350}.
