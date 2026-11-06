# Bottleneck: results/20261103-confirm-sweep/smoke/n16/rep-1/nydus-pf-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 1.70 / 2.59 / 2.83 / 2.90 s. Window 2.9 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 61lz | 3 | 1.06 | 1.09 |
| g2p4 | 22 | 1.69 | 2.90 |
| m23f | 23 | 1.75 | 2.74 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-015 | g2p4 | 2.90 | 0.06 | 0.50 | 0.44 | 0.81 | 1.86 | 0.03 | 0.04 | 0.05 | 0.20 | 1.43 |
| r1-024 | m23f | 2.74 | 0.03 | 0.48 | 0.35 | 1.17 | 1.44 | 0.04 | 0.10 | 0.05 | 0.08 | 1.12 |
| r1-044 | m23f | 2.62 | 0.05 | 0.56 | 0.48 | 0.82 | 1.48 | 0.10 | 0.05 | 0.06 | 0.09 | 1.16 |
| r1-021 | g2p4 | 2.55 | 0.03 | 0.78 | 0.21 | 0.59 | 1.47 | 0.01 | 0.04 | 0.14 | 0.07 | 1.08 |
| r1-025 | m23f | 2.52 | 0.05 | 0.51 | 0.52 | 0.78 | 1.35 | 0.12 | 0.13 | 0.02 | 0.14 | 0.82 |
| r1-022 | m23f | 2.37 | 0.03 | 0.43 | 0.39 | 0.74 | 1.42 | 0.07 | 0.04 | 0.02 | 0.17 | 1.07 |
| r1-011 | g2p4 | 2.35 | 0.03 | 0.75 | 0.18 | 0.54 | 1.28 | 0.02 | 0.03 | 0.04 | 0.25 | 0.91 |
| r1-027 | m23f | 2.28 | 0.03 | 0.52 | 0.39 | 0.54 | 1.23 | 0.01 | 0.08 | 0.21 | 0.10 | 0.80 |
| r1-036 | m23f | 2.27 | 0.02 | 0.53 | 0.41 | 0.60 | 1.25 | 0.01 | 0.04 | 0.01 | 0.17 | 0.95 |
| r1-028 | m23f | 2.23 | 0.03 | 0.53 | 0.35 | 0.60 | 1.26 | 0.07 | 0.09 | 0.23 | 0.09 | 0.69 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 61lz | 42.5 | 374.0 | 9290.1 | 0.8 | 7.0 | 23.2 | 1.0 | 14.9 | 23.7 | 2.9 | 2.5 | 14.8 | 0.0 | 342.6 | 0.0 | containerd/2813/containerd |
| g2p4 | 175.1 | 480.2 | 29180.6 | 0.6 | 11.8 | 136.2 | 1.0 | 76.1 | 84.8 | 1.9 | 50.8 | 43.9 | 0.0 | 271.0 | 0.9 | containerd/7355/containerd |
| m23f | 170.6 | 564.6 | 32946.3 | 0.7 | 15.2 | 165.2 | 1.9 | 75.9 | 85.3 | 5.7 | 51.0 | 46.6 | 0.0 | 335.6 | 0.6 | containerd-nydu/15340/containerd-nydu |

## Top processes in the window, cores per node

- 61lz: nydus-dragonfly 0.46, nydusd-amp 0.15, objectstore-plu 0.06, kubelet 0.05, containerd 0.04, kworker 0.04, ateworker-gviso 0.04, cilium-agent 0.04
- g2p4: nydus-dragonfly 4.37, nydusd-amp 1.19, objectstore-plu 0.44, ateworker-gviso 0.31, containerd-nydu 0.17, fluent-bit 0.13, atelet 0.07, cilium-agent 0.07
- m23f: nydus-dragonfly 4.87, nydusd-amp 1.23, objectstore-plu 0.45, ateworker-gviso 0.31, containerd-nydu 0.19, fluent-bit 0.13, kworker 0.08, atelet 0.07

## Timeline, node 61lz

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 3 | 1 | 45 | 719 | 0.74 | 14 | 9 | 1 | 2.3 | 0.2 | 52 | 81.5 |
| 1 | 2 | 38 | 374 | 9290 | 0.76 | 23 | 24 | 2 | 14.8 | 2.5 | 115 | 0.0 |
| 2 | 0 | 42 | 126 | 5432 | 0.53 | 12 | 14 | 3 | 11.2 | 1.7 | 28 | 0.0 |
| 3 | 0 | 0 | 1 | 52 | 0.39 | 0 | 2 | 1 | 0.2 | 0.1 | 0 | — |
| 4 | 0 | 0 | 1 | 52 | 0.41 | 0 | 2 | 1 | 0.7 | 0.1 | 0 | — |

## Timeline, node g2p4

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 7 | 423 | 0.39 | 0 | 7 | 1 | 1.2 | 0.5 | 101 | 127.0 |
| 1 | 22 | 57 | 480 | 17479 | 0.56 | 136 | 64 | 2 | 26.0 | 21.4 | 142 | 2.2 |
| 2 | 6 | 175 | 135 | 23324 | 0.37 | 48 | 85 | 2 | 32.1 | 50.8 | 229 | 0.0 |
| 3 | 0 | 161 | 254 | 29181 | 0.40 | 55 | 64 | 1 | 43.9 | 31.5 | 258 | 0.0 |
| 4 | 0 | 68 | 203 | 25411 | 0.37 | 32 | 32 | 2 | 42.8 | 3.4 | 158 | 0.0 |

## Timeline, node m23f

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 10 | 875 | 0.39 | 2 | 9 | 2 | 1.7 | 0.5 | 103 | 384.8 |
| 1 | 23 | 59 | 565 | 17132 | 0.73 | 165 | 81 | 6 | 27.7 | 27.7 | 129 | 6.3 |
| 2 | 7 | 171 | 368 | 23379 | 0.65 | 48 | 85 | 2 | 36.6 | 51.0 | 195 | 0.1 |
| 3 | 0 | 139 | 242 | 32946 | 0.39 | 54 | 71 | 0 | 46.6 | 31.3 | 255 | 0.0 |
| 4 | 0 | 89 | 212 | 29342 | 0.37 | 36 | 42 | 1 | 46.1 | 3.5 | 196 | 0.0 |

## Seed requests (nginx)

1564 requests in the window, 475 MB. Seed time p50 / p99 / max: 0.0 / 116.1 / 395.0 ms. Size p50 / p99: 174 / 3802 KB. Status: {200: 192, 206: 1372}.
