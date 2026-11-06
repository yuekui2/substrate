# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-repeats/driver/../a-n16-r1/rep-1/nydus-o-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 1.63 / 1.87 / 2.06 / 2.13 s. Window 2.1 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 1wp3 | 16 | 1.60 | 1.87 |
| 34bx | 20 | 1.63 | 1.98 |
| hmdl | 12 | 1.65 | 2.13 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-024 | hmdl | 2.13 | 0.03 | 0.63 | 0.36 | 0.66 | 1.05 | 0.09 | 0.09 | 0.08 | 0.08 | 0.67 |
| r1-025 | 34bx | 1.98 | 0.05 | 0.73 | 0.35 | 0.76 | 0.76 | 0.03 | 0.20 | 0.03 | 0.09 | 0.38 |
| r1-000 | 1wp3 | 1.87 | 0.03 | 0.71 | 0.85 | 0.55 | 0.18 | 0.00 | 0.02 | 0.01 | 0.08 | 0.04 |
| r1-015 | 34bx | 1.87 | 0.06 | 0.77 | 0.26 | 0.76 | 0.72 | 0.03 | 0.08 | 0.05 | 0.16 | 0.37 |
| r1-010 | hmdl | 1.85 | 0.04 | 0.63 | 0.35 | 0.57 | 0.77 | 0.02 | 0.08 | 0.02 | 0.13 | 0.47 |
| r1-044 | hmdl | 1.84 | 0.03 | 0.62 | 0.36 | 0.63 | 0.77 | 0.02 | 0.08 | 0.02 | 0.09 | 0.53 |
| r1-022 | 34bx | 1.81 | 0.05 | 0.76 | 0.34 | 0.53 | 0.59 | 0.03 | 0.21 | 0.02 | 0.11 | 0.11 |
| r1-042 | 34bx | 1.81 | 0.06 | 0.74 | 0.31 | 0.59 | 0.63 | 0.01 | 0.07 | 0.05 | 0.16 | 0.31 |
| r1-045 | 1wp3 | 1.78 | 0.03 | 0.53 | 0.40 | 0.55 | 0.72 | 0.04 | 0.03 | 0.06 | 0.13 | 0.43 |
| r1-030 | hmdl | 1.75 | 0.03 | 0.53 | 0.43 | 0.50 | 0.67 | 0.02 | 0.10 | 0.04 | 0.12 | 0.37 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1wp3 | 123.2 | 350.7 | 10656.0 | 1.5 | 6.3 | 100.8 | 0.0 | 31.8 | 37.5 | 1.9 | 24.0 | 25.6 | 0.0 | 189.5 | 0.5 | containerd/7031/containerd |
| 34bx | 145.8 | 473.1 | 15150.5 | 1.5 | 7.4 | 138.4 | 14.3 | 42.1 | 49.5 | 4.8 | 29.4 | 28.8 | 0.0 | 275.1 | 0.0 | containerd-nydu/15038/containerd-nydu |
| hmdl | 52.2 | 316.4 | 7052.8 | 2.1 | 7.6 | 67.0 | 1.0 | 20.1 | 28.4 | 7.6 | 18.9 | 36.8 | 0.0 | 195.3 | 3.4 | containerd-nydu/14980/containerd-nydu |

## Top processes in the window, cores per node

- 1wp3: nydusd-amp 0.55, nydus-dragonfly 0.51, objectstore-plu 0.40, ateworker-gviso 0.26, containerd-nydu 0.18, fluent-bit 0.11, kubelet 0.08, atelet 0.07
- 34bx: nydusd-amp 0.82, nydus-dragonfly 0.76, objectstore-plu 0.50, ateworker-gviso 0.33, containerd-nydu 0.21, fluent-bit 0.15, atelet 0.08, kubelet 0.07
- hmdl: nydusd-amp 0.47, nydus-dragonfly 0.44, objectstore-plu 0.31, ateworker-gviso 0.23, containerd-nydu 0.14, fluent-bit 0.09, kubelet 0.08, atelet 0.05

## Timeline, node 1wp3

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 5 | 354 | 0.51 | 0 | 6 | 1 | 1.5 | 0.4 | 81 | 103.0 |
| 1 | 16 | 2 | 351 | 2916 | 1.53 | 101 | 38 | 2 | 13.2 | 7.4 | 168 | 1.3 |
| 2 | 0 | 123 | 148 | 10656 | 0.58 | 44 | 36 | 2 | 25.6 | 24.0 | 155 | 0.5 |
| 3 | 0 | 50 | 198 | 14379 | 0.57 | 35 | 28 | 4 | 36.7 | 9.0 | 133 | 0.0 |

## Timeline, node 34bx

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 6 | 513 | 0.43 | 0 | 7 | 1 | 1.3 | 0.4 | 95 | 200.9 |
| 1 | 20 | 1 | 473 | 3415 | 1.51 | 138 | 45 | 5 | 12.2 | 9.2 | 207 | 3.8 |
| 2 | 0 | 146 | 204 | 15150 | 0.48 | 63 | 50 | 1 | 28.8 | 29.4 | 269 | 0.0 |
| 3 | 0 | 89 | 283 | 20007 | 0.46 | 47 | 32 | 5 | 40.2 | 10.7 | 157 | 0.0 |

## Timeline, node hmdl

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 314 | 2366 | 1.62 | 67 | 18 | 8 | 7.7 | 1.2 | 61 | 209.6 |
| 1 | 12 | 52 | 316 | 3573 | 2.10 | 54 | 28 | 1 | 36.8 | 5.9 | 136 | 1.0 |
| 2 | 1 | 40 | 224 | 7053 | 1.06 | 19 | 28 | 2 | 27.8 | 18.9 | 102 | 0.0 |
| 3 | 0 | 44 | 265 | 14049 | 0.68 | 30 | 23 | 4 | 40.3 | 0.9 | 147 | 0.0 |

## Seed requests (nginx)

1361 requests in the window, 442 MB. Seed time p50 / p99 / max: 0.0 / 99.8 / 278.0 ms. Size p50 / p99: 161 / 3849 KB. Status: {200: 192, 206: 1169}.
