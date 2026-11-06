### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pfb | 2 | 96 | 480 | 0 | 2.2 / 5.3 / 6.1 / 6.2 | 0.15 / 0.17 / 0.22 / 0.27 | 0.0 / 0.1 / 0.1 |
| pull-pfb | 2 | 96 | 480 | 0 | 86.5 / 114.0 / 140.1 / 141.1 | 0.15 / 0.20 / 0.25 / 0.28 | 0.0 / 0.1 / 0.1 |
| riptide-ms-pfb | 2 | 96 | 480 | 0 | 5.2 / 6.9 / 8.1 / 8.2 | 0.16 / 0.21 / 0.26 / 0.34 | 0.0 / 0.1 / 0.1 |
| spegel-pfb | 2 | 96 | 480 | 0 | 32.4 / 47.8 / 61.8 / 61.9 | 0.15 / 0.19 / 0.26 / 0.28 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pfb | 2 | 6.7 / 7.6 / 8.4 / 8.5 | 1.10 / 1.27 / 1.37 / 1.46 |
| pull-pfb | 2 | 87.5 / 114.4 / 140.5 / 141.5 | 1.14 / 1.47 / 1.55 / 1.68 |
| riptide-ms-pfb | 2 | 13.2 / 15.1 / 16.2 / 16.3 | 1.93 / 2.45 / 2.65 / 2.74 |
| spegel-pfb | 2 | 33.9 / 48.1 / 62.2 / 62.3 | 1.12 / 1.34 / 1.43 / 1.48 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pfb | 2 | 1.00 / 4.71 | 0.14 / 0.21 | 70 / 77 / 78 | 78 | 6.1 | 0/384 (0%) | — |
| pull-pfb | 2 | 0.99 / 1.39 | 0.14 / 0.22 | 152 / 180 / 209 | 209 | 2.3 | 0/384 (0%) | — |
| riptide-ms-pfb | 2 | 1.92 / 9.96 | 0.21 / 0.60 | 79 / 89 / 91 | 91 | 5.3 | 0/384 (0%) | — |
| spegel-pfb | 2 | 0.97 / 1.34 | 0.14 / 0.23 | 99 / 110 / 128 | 128 | 3.8 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pfb | 2 | 718 | 170 | 34671 |
| pull-pfb | 2 | 1891 | 49 | 21948 |
| riptide-ms-pfb | 2 | 517 | 0 | 3886 |
| spegel-pfb | 2 | 1928 | 858 | 25868 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pfb | 2 | 1 / 14 / 81 | 0.2 / 37.0 | nydus-dragonfly 0.09, gvisor_sentry 0.07, nydusd-amp 0.05, kubelet 0.03, ateworker-gviso 0.03, bash 0.03 |
| pull-pfb | 2 | 1 / 36 / 72 | 0.1 / 40.5 | atelet 1.72, kworker 0.06, gvisor_sentry 0.05, kubelet 0.04, ateworker-gviso 0.03, bash 0.02 |
| riptide-ms-pfb | 2 | 1 / 78 / 84 | 0.1 / 5.1 | gcfsd-v2 1.19, riptide 0.32, gvisor_sentry 0.27, ateworker-gviso 0.06, kubelet 0.04, fluent-bit 0.03 |
| spegel-pfb | 2 | 2 / 59 / 86 | 0.3 / 50.4 | spegel-preload 1.35, kworker 0.07, gvisor_sentry 0.05, kubelet 0.03, ateworker-gviso 0.03, bash 0.02 |
