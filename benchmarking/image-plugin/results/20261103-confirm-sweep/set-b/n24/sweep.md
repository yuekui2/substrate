### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pfb | 2 | 96 | 360 | 0 | 2.2 / 5.2 / 6.5 / 6.6 | 0.14 / 0.17 / 0.19 / 0.28 | 0.0 / 0.1 / 0.1 |
| pull-pfb | 2 | 96 | 360 | 0 | 59.1 / 86.9 / 112.5 / 115.1 | 0.14 / 0.17 / 0.23 / 0.28 | 0.0 / 0.1 / 0.2 |
| riptide-ms-pfb | 2 | 96 | 360 | 0 | 3.8 / 5.6 / 5.7 / 5.7 | 0.15 / 0.21 / 0.25 / 0.27 | 0.0 / 0.1 / 0.1 |
| spegel-pfb | 2 | 96 | 360 | 0 | 25.8 / 41.4 / 52.2 / 55.6 | 0.15 / 0.18 / 0.21 / 0.31 | 0.0 / 0.1 / 0.2 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pfb | 2 | 6.8 / 7.8 / 8.4 / 9.2 | 1.08 / 1.29 / 1.34 / 1.44 |
| pull-pfb | 2 | 60.1 / 87.2 / 112.8 / 115.4 | 1.11 / 1.31 / 1.44 / 1.65 |
| riptide-ms-pfb | 2 | 11.0 / 13.1 / 13.4 / 13.4 | 1.83 / 2.32 / 2.48 / 2.57 |
| spegel-pfb | 2 | 26.4 / 41.7 / 52.5 / 56.0 | 1.10 / 1.25 / 1.34 / 1.53 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pfb | 2 | 0.99 / 4.68 | 0.13 / 0.19 | 70 / 78 / 81 | 81 | 4.4 | 0/288 (0%) | — |
| pull-pfb | 2 | 0.97 / 1.27 | 0.13 / 0.21 | 125 / 153 / 179 | 179 | 2.0 | 0/288 (0%) | — |
| riptide-ms-pfb | 2 | 1.78 / 8.28 | 0.20 / 0.48 | 76 / 85 / 91 | 91 | 3.9 | 0/288 (0%) | — |
| spegel-pfb | 2 | 0.95 / 1.27 | 0.14 / 0.21 | 93 / 108 / 112 | 112 | 3.2 | 0/288 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pfb | 2 | 849 | 185 | 31913 |
| pull-pfb | 2 | 1909 | 2 | 15900 |
| riptide-ms-pfb | 2 | 591 | 2 | 4038 |
| spegel-pfb | 2 | 1937 | 838 | 22927 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pfb | 2 | 1 / 9 / 81 | 0.2 / 37.4 | nydus-dragonfly 0.06, gvisor_sentry 0.05, nydusd-amp 0.03, kubelet 0.03, bash 0.03, ateworker-gviso 0.02 |
| pull-pfb | 2 | 2 / 32 / 75 | 0.2 / 21.8 | atelet 1.61, kworker 0.05, gvisor_sentry 0.04, kubelet 0.04, ateworker-gviso 0.03, bash 0.02 |
| riptide-ms-pfb | 2 | 1 / 54 / 82 | 0.1 / 6.1 | gcfsd-v2 0.88, riptide 0.21, gvisor_sentry 0.21, ateworker-gviso 0.04, kubelet 0.04, fluent-bit 0.02 |
| spegel-pfb | 2 | 1 / 49 / 83 | 0.2 / 63.0 | spegel-preload 1.08, kworker 0.05, gvisor_sentry 0.03, kubelet 0.03, ateworker-gviso 0.02, bash 0.02 |
