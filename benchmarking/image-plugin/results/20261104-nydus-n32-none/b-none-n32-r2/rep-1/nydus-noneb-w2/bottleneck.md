# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-n32-none/driver/../b-none-n32-r2/rep-1/nydus-noneb-w2

96 first turns. TTFI p50 / p95 / p99 / p100: 2.20 / 3.08 / 3.40 / 3.71 s. Window 3.7 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 5rxf | 32 | 2.13 | 2.86 |
| q9c0 | 32 | 2.14 | 3.07 |
| wcms | 32 | 2.37 | 3.71 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-075 | wcms | 3.71 | 0.05 | 0.33 | 0.21 | 1.23 | 2.30 | 0.02 | 0.07 | 0.07 | 0.33 | 1.79 |
| r1-055 | wcms | 3.38 | 0.05 | 0.36 | 0.15 | 1.14 | 2.06 | 0.10 | 0.07 | 0.08 | 0.32 | 1.46 |
| r1-080 | wcms | 3.30 | 0.08 | 0.31 | 0.23 | 1.07 | 1.97 | 0.10 | 0.10 | 0.04 | 0.30 | 1.38 |
| r1-042 | wcms | 3.16 | 0.08 | 0.30 | 0.19 | 1.09 | 1.82 | 0.09 | 0.07 | 0.07 | 0.33 | 1.23 |
| r1-019 | wcms | 3.11 | 0.08 | 0.34 | 0.21 | 1.11 | 1.77 | 0.10 | 0.13 | 0.28 | 0.24 | 1.00 |
| r1-086 | q9c0 | 3.07 | 0.08 | 0.26 | 0.25 | 0.97 | 1.76 | 0.07 | 0.29 | 0.07 | 0.20 | 1.09 |
| r1-061 | wcms | 3.04 | 0.06 | 0.16 | 0.10 | 0.79 | 1.69 | 0.09 | 0.06 | 0.08 | 0.32 | 1.11 |
| r1-073 | wcms | 2.89 | 0.05 | 0.38 | 0.20 | 0.89 | 1.83 | 0.38 | 0.08 | 0.06 | 0.32 | 0.94 |
| r1-012 | wcms | 2.88 | 0.07 | 0.37 | 0.17 | 0.88 | 1.82 | 0.35 | 0.15 | 0.31 | 0.20 | 0.77 |
| r1-056 | 5rxf | 2.86 | 0.05 | 0.42 | 0.19 | 0.94 | 1.76 | 0.32 | 0.05 | 0.05 | 0.14 | 1.14 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 5rxf | 144.8 | 731.9 | 45213.0 | 2.2 | 18.2 | 247.0 | 0.0 | 64.8 | 74.0 | 5.7 | 53.7 | 57.7 | 0.0 | 420.1 | 0.0 | containerd-nydu/14755/containerd-nydu |
| q9c0 | 195.0 | 785.6 | 28428.1 | 1.5 | 12.4 | 264.6 | 1.9 | 61.8 | 70.3 | 41.3 | 45.8 | 39.4 | 0.0 | 412.6 | 5.0 | containerd-nydu/16952/containerd-nydu |
| wcms | 140.6 | 841.4 | 39491.0 | 1.8 | 16.3 | 270.3 | 20.0 | 59.2 | 62.7 | 9.5 | 48.3 | 53.2 | 0.0 | 444.9 | 0.0 | containerd-nydu/16690/containerd-nydu |

## Top processes in the window, cores per node

- 5rxf: nydusd-amp 1.32, ateworker-gviso 0.72, nydus-dragonfly 0.70, objectstore-plu 0.51, containerd-nydu 0.19, fluent-bit 0.17, kubelet 0.09, atelet 0.08
- q9c0: nydusd-amp 1.27, ateworker-gviso 0.85, nydus-dragonfly 0.83, objectstore-plu 0.61, containerd-nydu 0.23, fluent-bit 0.17, kubelet 0.10, atelet 0.09
- wcms: nydusd-amp 1.14, nydus-dragonfly 0.72, ateworker-gviso 0.61, objectstore-plu 0.51, containerd-nydu 0.20, fluent-bit 0.16, atelet 0.08, containerd-shim 0.08

## Timeline, node 5rxf

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 69 | 0.56 | 1 | 8 | 1 | 0.4 | 1.0 | 132 | 199.8 |
| 1 | 13 | 0 | 732 | 5585 | 2.16 | 206 | 74 | 6 | 10.3 | 13.1 | 102 | 1.0 |
| 2 | 22 | 139 | 99 | 6199 | 0.54 | 87 | 68 | 3 | 12.6 | 36.7 | 237 | 7.9 |
| 3 | 0 | 145 | 265 | 23788 | 0.40 | 241 | 70 | 3 | 33.9 | 53.7 | 558 | 1.4 |
| 4 | 0 | 122 | 463 | 45213 | 0.40 | 247 | 47 | 4 | 57.7 | 9.1 | 325 | 0.0 |
| 5 | 0 | 94 | 322 | 20015 | 0.45 | 95 | 31 | 3 | 28.4 | 8.2 | 32 | 0.0 |

## Timeline, node q9c0

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 146 | 1535 | 0.65 | 3 | 9 | 1 | 2.3 | 0.9 | 133 | 316.9 |
| 1 | 31 | 1 | 786 | 5044 | 1.51 | 196 | 70 | 41 | 13.9 | 24.6 | 102 | 20.8 |
| 2 | 17 | 169 | 319 | 13875 | 0.50 | 172 | 57 | 3 | 24.1 | 30.8 | 354 | 1.5 |
| 3 | 1 | 195 | 489 | 28428 | 0.43 | 265 | 66 | 4 | 39.4 | 45.8 | 449 | 1.0 |
| 4 | 0 | 90 | 509 | 37984 | 0.39 | 186 | 38 | 4 | 52.1 | 6.8 | 298 | 0.0 |

## Timeline, node wcms

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 52 | 0.43 | 0 | 2 | 1 | 0.5 | 0.1 | 130 | 231.7 |
| 1 | 19 | 0 | 841 | 6118 | 1.75 | 215 | 55 | 2 | 8.6 | 12.4 | 47 | 24.3 |
| 2 | 26 | 64 | 191 | 3659 | 0.79 | 45 | 57 | 10 | 8.3 | 25.8 | 240 | 2.2 |
| 3 | 6 | 141 | 209 | 14779 | 0.44 | 175 | 63 | 2 | 25.6 | 48.3 | 436 | 1.0 |
| 4 | 0 | 116 | 405 | 39491 | 0.41 | 270 | 50 | 4 | 53.2 | 20.5 | 396 | 0.0 |
| 5 | 0 | 125 | 379 | 33241 | 0.41 | 151 | 37 | 3 | 45.7 | 9.4 | 111 | 0.0 |

## Seed requests (nginx)

3650 requests in the window, 2234 MB. Seed time p50 / p99 / max: 0.0 / 154.5 / 436.0 ms. Size p50 / p99: 599 / 3729 KB. Status: {200: 384, 206: 3266}.
