# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-repeats/driver/../a-n24-r1/rep-1/nydus-o-w2

72 first turns. TTFI p50 / p95 / p99 / p100: 1.85 / 2.50 / 2.62 / 2.70 s. Window 2.7 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 9v46 | 25 | 1.83 | 2.46 |
| ckf0 | 23 | 1.92 | 2.70 |
| lp3s | 24 | 1.86 | 2.56 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-049 | ckf0 | 2.70 | 0.05 | 0.45 | 0.16 | 1.01 | 1.53 | 0.06 | 0.05 | 0.04 | 0.33 | 1.01 |
| r1-025 | ckf0 | 2.58 | 0.05 | 0.40 | 0.22 | 1.09 | 1.34 | 0.04 | 0.33 | 0.01 | 0.08 | 0.83 |
| r1-058 | ckf0 | 2.58 | 0.06 | 0.52 | 0.18 | 1.10 | 1.33 | 0.01 | 0.05 | 0.06 | 0.17 | 0.70 |
| r1-024 | lp3s | 2.56 | 0.06 | 0.45 | 0.26 | 0.91 | 1.48 | 0.06 | 0.05 | 0.01 | 0.35 | 0.98 |
| r1-046 | 9v46 | 2.46 | 0.05 | 0.40 | 0.34 | 0.83 | 1.49 | 0.14 | 0.06 | 0.06 | 0.31 | 0.89 |
| r1-048 | 9v46 | 2.44 | 0.06 | 0.41 | 0.30 | 0.81 | 1.46 | 0.13 | 0.06 | 0.06 | 0.32 | 0.87 |
| r1-063 | lp3s | 2.43 | 0.06 | 0.53 | 0.21 | 0.90 | 1.38 | 0.10 | 0.06 | 0.29 | 0.10 | 0.79 |
| r1-069 | ckf0 | 2.43 | 0.05 | 0.55 | 0.14 | 1.13 | 1.15 | 0.02 | 0.33 | 0.02 | 0.05 | 0.68 |
| r1-015 | 9v46 | 2.39 | 0.05 | 0.43 | 0.23 | 0.76 | 1.41 | 0.14 | 0.04 | 0.02 | 0.10 | 1.09 |
| r1-044 | lp3s | 2.34 | 0.06 | 0.50 | 0.16 | 0.96 | 1.23 | 0.04 | 0.07 | 0.30 | 0.08 | 0.66 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 9v46 | 133.1 | 585.5 | 21300.0 | 2.4 | 10.1 | 167.9 | 0.0 | 52.0 | 60.5 | 11.5 | 37.8 | 39.2 | 0.0 | 355.9 | 0.0 | containerd-nydu/16092/containerd-nydu |
| ckf0 | 151.0 | 694.4 | 18802.2 | 1.9 | 10.7 | 126.6 | 0.9 | 43.8 | 50.5 | 17.9 | 28.9 | 37.8 | 0.0 | 494.6 | 0.0 | containerd-nydu/15492/containerd-nydu |
| lp3s | 137.3 | 595.9 | 16087.0 | 2.4 | 11.6 | 110.2 | 0.0 | 45.3 | 52.0 | 16.7 | 31.6 | 34.0 | 0.0 | 353.6 | 7.6 | containerd-nydu/17080/containerd-nydu |

## Top processes in the window, cores per node

- 9v46: nydusd-amp 1.25, nydus-dragonfly 0.70, ateworker-gviso 0.57, objectstore-plu 0.49, containerd-nydu 0.21, fluent-bit 0.12, atelet 0.08, kubelet 0.07
- ckf0: nydusd-amp 1.15, nydus-dragonfly 0.89, objectstore-plu 0.60, ateworker-gviso 0.56, containerd-nydu 0.27, fluent-bit 0.16, containerd-shim 0.10, cilium-agent 0.10
- lp3s: nydusd-amp 1.09, nydus-dragonfly 0.99, objectstore-plu 0.59, ateworker-gviso 0.54, containerd-nydu 0.26, fluent-bit 0.18, atelet 0.10, kubelet 0.09

## Timeline, node 9v46

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 10 | 834 | 0.44 | 0 | 10 | 3 | 1.7 | 0.6 | 110 | 291.6 |
| 1 | 25 | 1 | 586 | 4121 | 2.41 | 168 | 60 | 11 | 11.5 | 24.8 | 116 | 14.0 |
| 2 | 5 | 133 | 134 | 8751 | 0.55 | 48 | 56 | 1 | 18.0 | 37.8 | 176 | 1.0 |
| 3 | 0 | 76 | 235 | 21300 | 0.44 | 64 | 46 | 5 | 39.2 | 22.1 | 313 | 0.0 |

## Timeline, node ckf0

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 349 | 2829 | 1.94 | 127 | 32 | 3 | 7.5 | 9.4 | 97 | 298.9 |
| 1 | 23 | 67 | 694 | 5986 | 1.77 | 50 | 46 | 18 | 13.9 | 19.2 | 118 | 1.0 |
| 2 | 8 | 151 | 348 | 18802 | 0.54 | 62 | 50 | 2 | 37.8 | 28.9 | 325 | 1.0 |
| 4 | 0 | 98 | 423 | 22175 | 0.57 | 48 | 37 | 5 | 47.5 | 10.8 | 151 | 0.0 |

## Timeline, node lp3s

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 18 | 1321 | 0.49 | 76 | 21 | 6 | 3.9 | 5.9 | 99 | 353.2 |
| 1 | 24 | 51 | 596 | 4660 | 2.45 | 110 | 52 | 17 | 15.3 | 19.4 | 142 | 1.6 |
| 2 | 6 | 137 | 230 | 16087 | 0.55 | 65 | 50 | 1 | 34.0 | 31.6 | 287 | 1.0 |
| 3 | 0 | 73 | 287 | 20248 | 0.52 | 45 | 30 | 5 | 44.6 | 9.5 | 231 | 0.0 |

## Seed requests (nginx)

2129 requests in the window, 692 MB. Seed time p50 / p99 / max: 0.0 / 158.4 / 411.0 ms. Size p50 / p99: 174 / 3852 KB. Status: {200: 288, 206: 1841}.
