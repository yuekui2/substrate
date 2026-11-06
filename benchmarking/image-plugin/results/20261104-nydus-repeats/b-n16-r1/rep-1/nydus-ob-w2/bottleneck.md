# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-repeats/driver/../b-n16-r1/rep-1/nydus-ob-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 1.37 / 1.69 / 1.75 / 1.75 s. Window 1.8 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 28rx | 17 | 1.35 | 1.69 |
| j1qh | 14 | 1.25 | 1.51 |
| t3hf | 17 | 1.47 | 1.75 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-003 | t3hf | 1.75 | 0.05 | 0.67 | 0.17 | 0.50 | 0.78 | 0.03 | 0.05 | 0.08 | 0.18 | 0.43 |
| r1-025 | t3hf | 1.74 | 0.05 | 0.54 | 0.33 | 0.50 | 0.74 | 0.01 | 0.03 | 0.09 | 0.07 | 0.38 |
| r1-015 | t3hf | 1.70 | 0.05 | 0.67 | 0.22 | 0.51 | 0.68 | 0.02 | 0.10 | 0.14 | 0.10 | 0.29 |
| r1-042 | 28rx | 1.69 | 0.04 | 0.35 | 0.22 | 0.61 | 0.94 | 0.13 | 0.04 | 0.08 | 0.25 | 0.42 |
| r1-010 | 28rx | 1.60 | 0.05 | 0.35 | 0.21 | 0.59 | 0.88 | 0.15 | 0.04 | 0.08 | 0.22 | 0.34 |
| r1-019 | 28rx | 1.59 | 0.05 | 0.34 | 0.21 | 0.70 | 0.75 | 0.03 | 0.04 | 0.08 | 0.25 | 0.33 |
| r1-037 | t3hf | 1.58 | 0.05 | 0.67 | 0.21 | 0.54 | 0.57 | 0.01 | 0.03 | 0.08 | 0.18 | 0.25 |
| r1-040 | t3hf | 1.56 | 0.05 | 0.67 | 0.25 | 0.53 | 0.51 | 0.06 | 0.19 | 0.06 | 0.06 | 0.10 |
| r1-014 | t3hf | 1.53 | 0.04 | 0.63 | 0.22 | 0.43 | 0.51 | 0.06 | 0.19 | 0.01 | 0.09 | 0.11 |
| r1-032 | 28rx | 1.51 | 0.05 | 0.35 | 0.17 | 0.61 | 0.77 | 0.14 | 0.04 | 0.08 | 0.24 | 0.25 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 28rx | 140.3 | 395.3 | 11834.1 | 1.5 | 5.6 | 114.4 | 1.9 | 35.3 | 41.8 | 1.9 | 26.4 | 22.3 | 0.0 | 216.7 | 0.0 | containerd/9004/containerd |
| j1qh | 69.0 | 386.2 | 11782.6 | 1.1 | 5.2 | 125.0 | 1.0 | 37.1 | 43.1 | 5.6 | 24.6 | 23.1 | 0.0 | 195.0 | 0.0 | containerd/9500/containerd |
| t3hf | 79.5 | 418.6 | 12589.6 | 1.3 | 5.8 | 99.5 | 1.0 | 38.0 | 41.7 | 21.5 | 35.7 | 23.0 | 0.0 | 207.0 | 0.1 | containerd-nydu/14892/containerd-nydu |

## Top processes in the window, cores per node

- 28rx: nydusd-amp 0.68, nydus-dragonfly 0.66, objectstore-plu 0.44, ateworker-gviso 0.33, containerd-nydu 0.19, fluent-bit 0.12, atelet 0.07, cilium-agent 0.06
- j1qh: nydusd-amp 0.62, nydus-dragonfly 0.52, objectstore-plu 0.37, ateworker-gviso 0.27, containerd-nydu 0.17, kubelet 0.12, fluent-bit 0.11, containerd 0.06
- t3hf: nydusd-amp 0.69, nydus-dragonfly 0.61, objectstore-plu 0.44, ateworker-gviso 0.27, containerd-nydu 0.19, fluent-bit 0.14, kubelet 0.09, containerd 0.07

## Timeline, node 28rx

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 1 | 66 | 0.58 | 0 | 3 | 0 | 0.3 | 0.1 | 83 | 191.4 |
| 1 | 17 | 4 | 395 | 3333 | 1.48 | 114 | 40 | 2 | 7.4 | 13.3 | 213 | 1.0 |
| 2 | 0 | 140 | 176 | 11834 | 0.47 | 50 | 42 | 2 | 22.3 | 26.4 | 245 | 0.0 |
| 3 | 0 | 71 | 228 | 18922 | 0.42 | 42 | 30 | 4 | 35.7 | 9.2 | 130 | 0.0 |

## Timeline, node j1qh

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 4 | 245 | 0.45 | 0 | 7 | 1 | 1.3 | 0.3 | 80 | 160.2 |
| 1 | 14 | 63 | 386 | 4873 | 1.05 | 125 | 43 | 2 | 11.2 | 13.1 | 180 | 2.2 |
| 2 | 0 | 69 | 126 | 11783 | 0.44 | 29 | 36 | 6 | 23.1 | 24.6 | 225 | 0.0 |
| 3 | 0 | 44 | 359 | 18177 | 0.59 | 34 | 22 | 4 | 37.2 | 4.0 | 128 | 0.0 |

## Timeline, node t3hf

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 11 | 891 | 0.38 | 31 | 12 | 1 | 1.5 | 0.6 | 90 | 146.9 |
| 1 | 17 | 63 | 419 | 4515 | 1.26 | 99 | 41 | 21 | 11.1 | 13.1 | 182 | 1.0 |
| 2 | 0 | 79 | 130 | 12590 | 0.42 | 33 | 42 | 1 | 23.0 | 35.7 | 227 | 0.0 |
| 3 | 0 | 37 | 216 | 21031 | 0.43 | 45 | 27 | 5 | 43.1 | 5.0 | 160 | 0.0 |

## Seed requests (nginx)

1363 requests in the window, 431 MB. Seed time p50 / p99 / max: 0.0 / 116.0 / 274.0 ms. Size p50 / p99: 164 / 3860 KB. Status: {200: 192, 206: 1171}.
