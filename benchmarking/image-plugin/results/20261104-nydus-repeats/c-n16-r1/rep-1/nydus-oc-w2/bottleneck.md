# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-repeats/driver/../c-n16-r1/rep-1/nydus-oc-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 1.39 / 1.80 / 1.84 / 1.86 s. Window 1.9 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| dcgm | 15 | 1.44 | 1.86 |
| rfm0 | 18 | 1.45 | 1.82 |
| wkrh | 15 | 1.27 | 1.56 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-013 | dcgm | 1.86 | 0.05 | 0.35 | 0.21 | 0.62 | 1.10 | 0.11 | 0.04 | 0.02 | 0.26 | 0.63 |
| r1-015 | rfm0 | 1.82 | 0.05 | 0.37 | 0.21 | 0.74 | 0.93 | 0.03 | 0.04 | 0.23 | 0.13 | 0.42 |
| r1-027 | dcgm | 1.81 | 0.04 | 0.33 | 0.22 | 0.58 | 1.09 | 0.16 | 0.04 | 0.02 | 0.26 | 0.58 |
| r1-014 | dcgm | 1.77 | 0.04 | 0.32 | 0.16 | 0.59 | 1.02 | 0.13 | 0.04 | 0.02 | 0.28 | 0.52 |
| r1-041 | dcgm | 1.77 | 0.05 | 0.32 | 0.20 | 0.59 | 1.03 | 0.13 | 0.04 | 0.02 | 0.26 | 0.54 |
| r1-018 | rfm0 | 1.74 | 0.04 | 0.43 | 0.21 | 0.74 | 0.84 | 0.03 | 0.04 | 0.01 | 0.20 | 0.50 |
| r1-031 | rfm0 | 1.69 | 0.04 | 0.43 | 0.18 | 0.58 | 0.93 | 0.16 | 0.04 | 0.03 | 0.12 | 0.38 |
| r1-011 | dcgm | 1.67 | 0.06 | 0.40 | 0.16 | 0.61 | 0.92 | 0.13 | 0.04 | 0.02 | 0.26 | 0.44 |
| r1-033 | dcgm | 1.60 | 0.06 | 0.35 | 0.18 | 0.56 | 0.88 | 0.16 | 0.04 | 0.02 | 0.25 | 0.38 |
| r1-016 | rfm0 | 1.57 | 0.05 | 0.40 | 0.19 | 0.56 | 0.83 | 0.18 | 0.04 | 0.01 | 0.23 | 0.34 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| dcgm | 80.9 | 345.3 | 14448.4 | 1.1 | 6.5 | 88.9 | 11.3 | 29.1 | 36.4 | 1.9 | 20.6 | 26.8 | 0.0 | 245.4 | 0.0 | containerd/2806/containerd |
| rfm0 | 100.5 | 486.1 | 16930.3 | 1.0 | 7.2 | 104.9 | 0.0 | 41.8 | 47.0 | 7.4 | 25.1 | 30.2 | 0.0 | 252.0 | 0.2 | containerd-nydu/16824/containerd-nydu |
| wkrh | 132.4 | 385.5 | 13234.4 | 1.4 | 6.6 | 117.0 | 0.0 | 31.7 | 41.1 | 2.8 | 21.7 | 27.1 | 0.0 | 218.5 | 0.0 | containerd/6844/containerd |

## Top processes in the window, cores per node

- dcgm: nydusd-amp 0.77, nydus-dragonfly 0.59, objectstore-plu 0.38, ateworker-gviso 0.28, containerd-nydu 0.17, fluent-bit 0.14, kubelet 0.09, containerd-shim 0.08
- rfm0: nydusd-amp 0.94, nydus-dragonfly 0.70, objectstore-plu 0.44, ateworker-gviso 0.35, containerd-nydu 0.19, fluent-bit 0.14, containerd 0.09, kubelet 0.09
- wkrh: nydus-dragonfly 0.59, objectstore-plu 0.38, ateworker-gviso 0.25, containerd-nydu 0.17, fluent-bit 0.12, kubelet 0.07, atelet 0.06, cilium-agent 0.05

## Timeline, node dcgm

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 91 | 1561 | 0.82 | 89 | 20 | 1 | 2.9 | 4.6 | 75 | 201.9 |
| 1 | 15 | 57 | 345 | 5286 | 1.13 | 49 | 33 | 2 | 13.6 | 13.5 | 215 | 1.0 |
| 2 | 0 | 81 | 184 | 14448 | 0.44 | 38 | 36 | 2 | 26.8 | 20.6 | 199 | 0.0 |
| 3 | 0 | 41 | 131 | 14167 | 0.41 | 22 | 21 | 3 | 29.6 | 1.6 | 111 | 0.0 |

## Timeline, node rfm0

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 12 | 1038 | 0.38 | 52 | 17 | 2 | 1.8 | 2.1 | 87 | 234.6 |
| 1 | 18 | 75 | 486 | 6060 | 1.01 | 105 | 47 | 7 | 12.1 | 19.0 | 233 | 1.0 |
| 2 | 0 | 101 | 204 | 16930 | 0.42 | 44 | 42 | 2 | 30.2 | 25.1 | 230 | 0.0 |
| 3 | 0 | 54 | 198 | 19697 | 0.40 | 35 | 21 | 4 | 37.1 | 3.0 | 136 | 0.0 |

## Timeline, node wkrh

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 53 | 0.47 | 0 | 2 | 0 | 0.3 | 0.1 | 85 | 225.7 |
| 1 | 15 | 14 | 386 | 3540 | 1.36 | 117 | 38 | 3 | 9.4 | 8.9 | 208 | 1.0 |
| 2 | 0 | 132 | 173 | 13234 | 0.50 | 46 | 41 | 2 | 27.1 | 21.7 | 241 | 0.0 |
| 3 | 0 | 70 | 218 | 16770 | 0.49 | 38 | 23 | 3 | 37.4 | 5.9 | 94 | 0.0 |

## Seed requests (nginx)

1483 requests in the window, 485 MB. Seed time p50 / p99 / max: 0.0 / 117.2 / 300.0 ms. Size p50 / p99: 182 / 3864 KB. Status: {200: 192, 206: 1291}.
