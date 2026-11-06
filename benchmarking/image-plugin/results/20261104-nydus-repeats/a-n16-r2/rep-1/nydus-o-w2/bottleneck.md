# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-repeats/driver/../a-n16-r2/rep-1/nydus-o-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 1.37 / 1.56 / 1.69 / 1.74 s. Window 1.7 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 2vsd | 17 | 1.25 | 1.51 |
| 37hg | 16 | 1.39 | 1.74 |
| tfpx | 15 | 1.39 | 1.53 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-024 | 37hg | 1.74 | 0.06 | 0.51 | 0.27 | 0.67 | 0.81 | 0.01 | 0.03 | 0.01 | 0.13 | 0.61 |
| r1-015 | 37hg | 1.64 | 0.05 | 0.50 | 0.63 | 0.56 | 0.36 | 0.00 | 0.02 | 0.01 | 0.10 | 0.19 |
| r1-043 | 37hg | 1.57 | 0.05 | 0.52 | 0.27 | 0.67 | 0.65 | 0.01 | 0.04 | 0.08 | 0.20 | 0.30 |
| r1-028 | 37hg | 1.53 | 0.04 | 0.59 | 0.20 | 0.59 | 0.61 | 0.01 | 0.04 | 0.09 | 0.17 | 0.28 |
| r1-025 | tfpx | 1.53 | 0.05 | 0.57 | 0.11 | 0.58 | 0.69 | 0.04 | 0.04 | 0.01 | 0.24 | 0.30 |
| r1-022 | 2vsd | 1.51 | 0.03 | 0.28 | 0.16 | 0.54 | 0.83 | 0.14 | 0.05 | 0.12 | 0.20 | 0.30 |
| r1-021 | 37hg | 1.51 | 0.04 | 0.53 | 0.30 | 0.54 | 0.54 | 0.08 | 0.17 | 0.01 | 0.12 | 0.10 |
| r1-013 | 37hg | 1.50 | 0.06 | 0.56 | 0.25 | 0.42 | 0.54 | 0.01 | 0.24 | 0.01 | 0.13 | 0.09 |
| r1-044 | 37hg | 1.50 | 0.05 | 0.50 | 0.24 | 0.51 | 0.60 | 0.03 | 0.03 | 0.01 | 0.19 | 0.24 |
| r1-001 | 2vsd | 1.48 | 0.04 | 0.23 | 0.19 | 0.53 | 0.79 | 0.13 | 0.05 | 0.02 | 0.15 | 0.42 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 2vsd | 108.6 | 408.8 | 15780.3 | 1.3 | 6.5 | 122.8 | 7.6 | 40.6 | 51.1 | 1.9 | 29.7 | 26.8 | 0.0 | 209.9 | 0.1 | containerd/2777/containerd |
| 37hg | 110.8 | 587.9 | 10155.9 | 1.4 | 6.2 | 113.0 | 1.9 | 33.1 | 38.0 | 3.8 | 22.6 | 20.5 | 0.0 | 466.0 | 0.2 | containerd-nydu/14490/containerd-nydu |
| tfpx | 110.0 | 394.9 | 15869.4 | 1.2 | 6.0 | 123.4 | 2.8 | 32.2 | 42.7 | 1.9 | 26.0 | 26.1 | 0.0 | 213.8 | 0.0 | containerd/9410/containerd |

## Top processes in the window, cores per node

- 2vsd: nydusd-amp 0.76, nydus-dragonfly 0.58, objectstore-plu 0.46, ateworker-gviso 0.34, containerd-nydu 0.18, fluent-bit 0.12, cilium-agent 0.07, atelet 0.07
- 37hg: nydus-dragonfly 1.00, nydusd-amp 0.89, objectstore-plu 0.62, ateworker-gviso 0.39, containerd-nydu 0.25, fluent-bit 0.17, atelet 0.10, containerd 0.08
- tfpx: nydusd-amp 0.70, nydus-dragonfly 0.52, objectstore-plu 0.38, ateworker-gviso 0.25, containerd-nydu 0.17, fluent-bit 0.11, atelet 0.06, kworker 0.04

## Timeline, node 2vsd

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 4 | 228 | 0.46 | 0 | 7 | 1 | 0.8 | 0.5 | 87 | 152.2 |
| 1 | 17 | 60 | 409 | 4791 | 1.32 | 123 | 51 | 2 | 11.1 | 16.7 | 218 | 1.0 |
| 2 | 0 | 109 | 179 | 15780 | 0.40 | 44 | 40 | 2 | 26.8 | 29.7 | 268 | 0.3 |
| 3 | 0 | 43 | 214 | 20286 | 0.41 | 37 | 24 | 5 | 39.0 | 2.8 | 147 | 0.0 |

## Timeline, node 37hg

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 197 | 802 | 1.22 | 0 | 3 | 0 | 2.0 | 0.1 | 80 | 198.1 |
| 1 | 16 | 0 | 588 | 3903 | 1.44 | 113 | 36 | 4 | 7.8 | 6.1 | 214 | 1.0 |
| 2 | 0 | 111 | 367 | 10156 | 0.60 | 55 | 38 | 1 | 20.5 | 22.6 | 184 | 0.0 |
| 3 | 0 | 73 | 611 | 16756 | 0.77 | 37 | 24 | 3 | 31.6 | 9.0 | 100 | 0.0 |

## Timeline, node tfpx

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 4 | 275 | 0.40 | 0 | 7 | 1 | 0.9 | 0.3 | 83 | 129.2 |
| 1 | 15 | 35 | 395 | 3890 | 1.15 | 123 | 43 | 2 | 8.8 | 10.7 | 219 | 1.0 |
| 2 | 0 | 110 | 149 | 15869 | 0.38 | 39 | 39 | 2 | 26.1 | 26.0 | 217 | 0.0 |
| 3 | 0 | 36 | 209 | 17343 | 0.40 | 35 | 23 | 4 | 31.6 | 4.1 | 121 | 0.0 |

## Seed requests (nginx)

1411 requests in the window, 449 MB. Seed time p50 / p99 / max: 0.0 / 109.0 / 255.0 ms. Size p50 / p99: 162 / 3837 KB. Status: {200: 192, 206: 1219}.
