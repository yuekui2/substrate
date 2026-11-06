# Bottleneck: /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/results/20261104-nydus-repeats/driver/../c-n16-r2/rep-1/nydus-oc-w2

48 first turns. TTFI p50 / p95 / p99 / p100: 1.41 / 1.67 / 1.77 / 1.82 s. Window 1.8 s.

## Per node

| node | actors | TTFI p50 | TTFI max |
|---|---|---|---|
| 0rtm | 16 | 1.49 | 1.82 |
| 1jq5 | 16 | 1.36 | 1.67 |
| pft5 | 16 | 1.41 | 1.66 |

## Slowest 10 first turns

| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| r1-015 | 0rtm | 1.82 | 0.03 | 0.72 | 0.19 | 0.75 | 0.79 | 0.01 | 0.04 | 0.08 | 0.20 | 0.43 |
| r1-004 | 0rtm | 1.71 | 0.03 | 0.57 | 0.26 | 0.71 | 0.76 | 0.07 | 0.04 | 0.09 | 0.19 | 0.34 |
| r1-025 | 0rtm | 1.67 | 0.03 | 0.59 | 0.25 | 0.57 | 0.72 | 0.07 | 0.04 | 0.10 | 0.11 | 0.21 |
| r1-014 | 1jq5 | 1.67 | 0.03 | 0.27 | 0.11 | 0.74 | 0.80 | 0.01 | 0.03 | 0.11 | 0.07 | 0.34 |
| r1-018 | pft5 | 1.66 | 0.02 | 0.67 | 0.20 | 0.68 | 0.66 | 0.09 | 0.07 | 0.01 | 0.09 | 0.37 |
| r1-020 | 0rtm | 1.62 | 0.02 | 0.59 | 0.25 | 0.57 | 0.67 | 0.07 | 0.04 | 0.11 | 0.23 | 0.19 |
| r1-021 | 1jq5 | 1.59 | 0.03 | 0.27 | 0.14 | 0.62 | 0.85 | 0.14 | 0.03 | 0.01 | 0.10 | 0.54 |
| r1-012 | 0rtm | 1.57 | 0.03 | 0.72 | 0.26 | 0.56 | 0.48 | 0.03 | 0.18 | 0.02 | 0.07 | 0.12 |
| r1-022 | pft5 | 1.55 | 0.03 | 0.66 | 0.14 | 0.52 | 0.62 | 0.01 | 0.11 | 0.03 | 0.09 | 0.28 |
| r1-031 | pft5 | 1.55 | 0.03 | 0.74 | 0.16 | 0.48 | 0.53 | 0.12 | 0.09 | 0.01 | 0.12 | 0.11 |

## Peaks in the window, per node

| node | rd_mb | wr_mb | iops | await_ms | queue | rx_mb | retrans | cpu_busy | top_core | top_thread | psi_cpu | psi_io | psi_memory | dirty_mb | wb_mb | busiest thread |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0rtm | 122.9 | 411.4 | 12064.2 | 1.5 | 6.9 | 123.2 | 0.0 | 32.8 | 40.4 | 5.7 | 19.8 | 27.8 | 0.0 | 240.8 | 0.3 | containerd-nydu/15508/containerd-nydu |
| 1jq5 | 105.2 | 399.3 | 19273.1 | 1.5 | 8.8 | 111.1 | 0.0 | 35.5 | 44.3 | 10.3 | 20.2 | 35.8 | 0.0 | 251.6 | 0.0 | containerd-nydu/17382/containerd-nydu |
| pft5 | 124.6 | 370.0 | 12448.2 | 4.7 | 7.1 | 106.5 | 0.0 | 33.9 | 39.1 | 3.8 | 24.0 | 26.9 | 0.0 | 199.7 | 0.0 | containerd-nydu/15569/containerd-nydu |

## Top processes in the window, cores per node

- 0rtm: nydusd-amp 0.69, nydus-dragonfly 0.66, objectstore-plu 0.43, ateworker-gviso 0.27, containerd-nydu 0.19, fluent-bit 0.12, kubelet 0.07, atelet 0.07
- 1jq5: nydusd-amp 1.00, nydus-dragonfly 0.69, objectstore-plu 0.43, ateworker-gviso 0.35, containerd-nydu 0.18, fluent-bit 0.15, atelet 0.08, kubelet 0.07
- pft5: nydusd-amp 0.65, nydus-dragonfly 0.60, objectstore-plu 0.42, ateworker-gviso 0.29, containerd-nydu 0.18, fluent-bit 0.12, kworker 0.07, atelet 0.07

## Timeline, node 0rtm

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 3 | 152 | 0.57 | 0 | 6 | 1 | 0.9 | 0.3 | 84 | 206.2 |
| 1 | 16 | 8 | 411 | 3443 | 1.48 | 123 | 38 | 6 | 9.3 | 6.9 | 225 | 1.0 |
| 2 | 0 | 123 | 195 | 12064 | 0.57 | 57 | 40 | 1 | 27.8 | 19.8 | 195 | 0.0 |
| 3 | 0 | 74 | 216 | 17590 | 0.50 | 38 | 26 | 3 | 38.9 | 4.0 | 109 | 0.0 |

## Timeline, node 1jq5

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 358 | 2639 | 1.51 | 111 | 30 | 1 | 6.4 | 5.7 | 83 | 286.5 |
| 1 | 16 | 83 | 399 | 7551 | 0.70 | 50 | 44 | 10 | 14.3 | 20.2 | 215 | 1.0 |
| 2 | 0 | 105 | 348 | 19273 | 0.45 | 47 | 35 | 2 | 35.8 | 17.6 | 282 | 0.0 |

## Timeline, node pft5

| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0 | 0 | 0 | 240 | 1500 | 4.73 | 0 | 6 | 1 | 0.5 | 0.2 | 88 | 147.8 |
| 1 | 15 | 2 | 370 | 3383 | 1.33 | 107 | 39 | 4 | 8.2 | 8.2 | 230 | 1.0 |
| 2 | 0 | 125 | 197 | 12448 | 0.54 | 55 | 39 | 1 | 26.9 | 24.0 | 183 | 0.0 |
| 3 | 0 | 59 | 199 | 15512 | 0.48 | 33 | 26 | 3 | 33.4 | 5.0 | 100 | 0.0 |

## Seed requests (nginx)

1480 requests in the window, 483 MB. Seed time p50 / p99 / max: 0.0 / 129.1 / 307.0 ms. Size p50 / p99: 168 / 3864 KB. Status: {200: 192, 206: 1288}.
