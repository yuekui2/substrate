# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-repeats/driver/../b-n16-r2/rep-1/nydus-ob-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 1.27 / 1.60 / 1.78 / 1.81 s. Window 1.8 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| b7l9 | 16 | 1.26 | 1.53 |
| k7n8 | 18 | 1.37 | 1.81 |
| vg3t | 14 | 1.15 | 1.34 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-042 | k7n8 | 1.81 | 0.02 | 0.31 | 0.16 | 0.71 | 0.98 | 0.01 | 0.04 | 0.06 | 0.24 | 0.58 |
| r1-039 | k7n8 | 1.76 | 0.04 | 0.32 | 0.14 | 0.63 | 1.01 | 0.08 | 0.04 | 0.07 | 0.23 | 0.55 |
| r1-043 | k7n8 | 1.62 | 0.03 | 0.58 | 0.09 | 0.50 | 0.81 | 0.03 | 0.04 | 0.06 | 0.24 | 0.40 |
| r1-025 | k7n8 | 1.57 | 0.04 | 0.32 | 0.19 | 0.67 | 0.79 | 0.05 | 0.03 | 0.06 | 0.23 | 0.36 |
| r1-046 | k7n8 | 1.57 | 0.04 | 0.29 | 0.14 | 0.64 | 0.83 | 0.09 | 0.04 | 0.06 | 0.24 | 0.36 |
| r1-036 | k7n8 | 1.57 | 0.03 | 0.31 | 0.15 | 0.66 | 0.79 | 0.06 | 0.04 | 0.06 | 0.07 | 0.29 |
| r1-023 | b7l9 | 1.53 | 0.03 | 0.35 | 0.16 | 0.67 | 0.75 | 0.04 | 0.04 | 0.07 | 0.21 | 0.37 |
| r1-017 | b7l9 | 1.47 | 0.03 | 0.38 | 0.10 | 0.55 | 0.80 | 0.14 | 0.05 | 0.09 | 0.09 | 0.27 |
| r1-004 | k7n8 | 1.46 | 0.02 | 0.30 | 0.14 | 0.51 | 0.83 | 0.21 | 0.03 | 0.01 | 0.11 | 0.45 |
| r1-018 | b7l9 | 1.44 | 0.03 | 0.33 | 0.20 | 0.55 | 0.78 | 0.15 | 0.03 | 0.08 | 0.21 | 0.29 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| b7l9 | 74.5 | 479.4 | 11698.3 | 1.3 | 6.3 | 125.1 | 1.9 | 36.8 | 44.3 | 1.9 | 29.1 | 22.4 | 0.0 | 534.0 | 0.2 | containerd/2791/containerd |
| k7n8 | 131.7 | 370.4 | 12838.1 | 1.6 | 5.8 | 117.8 | 1.0 | 38.0 | 43.8 | 1.9 | 24.4 | 23.7 | 0.0 | 445.3 | 0.0 | containerd/2775/containerd |
| vg3t | 67.7 | 375.2 | 10835.1 | 1.0 | 4.9 | 119.7 | 3.8 | 35.9 | 46.2 | 1.9 | 27.6 | 21.7 | 0.0 | 189.1 | 0.0 | containerd/6792/containerd |

## Top processes in the window, cores per node

- b7l9: nydusd-amp 0.62, nydus-dragonfly 0.56, objectstore-plu 0.42, ateworker-gviso 0.29, containerd-nydu 0.18, fluent-bit 0.11, kubelet 0.06, atelet 0.06
- k7n8: nydus-dragonfly 0.98, objectstore-plu 0.67, ateworker-gviso 0.50, containerd-nydu 0.30, fluent-bit 0.17, cilium-agent 0.10, atelet 0.10, kubelet 0.07
- vg3t: nydusd-amp 0.57, nydus-dragonfly 0.53, objectstore-plu 0.36, ateworker-gviso 0.25, containerd-nydu 0.15, fluent-bit 0.09, atelet 0.05, cilium-agent 0.05

## Timeline, node b7l9

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 3 | 0 | 75 | 397 | 1.17 | 1 | 4 | 1 | 0.9 | 0.3 | 92 | 150.2 |
| 1 | 16 | 56 | 479 | 4883 | 1.26 | 125 | 44 | 2 | 10.5 | 14.3 | 188 | 1.0 |
| 2 | 0 | 75 | 214 | 11698 | 0.52 | 34 | 38 | 1 | 22.4 | 29.1 | 264 | 0.0 |
| 3 | 0 | 42 | 470 | 21133 | 0.53 | 40 | 22 | 4 | 39.3 | 2.4 | 139 | 0.0 |

## Timeline, node k7n8

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 2 | 1 | 120 | 1139 | 1.25 | 0 | 2 | 0 | 0.3 | 0.1 | 92 | 209.4 |
| 1 | 18 | 2 | 370 | 3207 | 1.62 | 118 | 37 | 2 | 9.4 | 14.6 | 211 | 1.0 |
| 2 | 0 | 132 | 166 | 12838 | 0.45 | 53 | 44 | 1 | 23.7 | 24.4 | 273 | 0.0 |
| 3 | 0 | 92 | 297 | 22044 | 0.43 | 49 | 33 | 6 | 38.6 | 14.9 | 162 | 0.0 |

## Timeline, node vg3t

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 2 | 0 | 1 | 52 | 0.49 | 0 | 5 | 0 | 0.2 | 0.2 | 102 | 163.9 |
| 1 | 14 | 60 | 375 | 4691 | 1.02 | 120 | 46 | 2 | 10.7 | 12.3 | 147 | 0.0 |
| 2 | 0 | 68 | 113 | 10835 | 0.45 | 24 | 37 | 1 | 21.7 | 27.6 | 206 | 0.0 |
| 3 | 0 | 44 | 191 | 19010 | 0.44 | 36 | 27 | 3 | 40.0 | 2.3 | 123 | 0.0 |

## Seed requests (nginx)

1458 requests in the window, 451 MB. Seed time p50 / p99 / max: 0.0 / 139.4 / 294.0 ms. Size p50 / p99: 181 / 3858 KB. Status: {200: 192, 206: 1266}.
