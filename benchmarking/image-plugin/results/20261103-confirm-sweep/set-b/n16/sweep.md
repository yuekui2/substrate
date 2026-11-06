### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pfb | 2 | 96 | 240 | 0 | 1.4 / 1.7 / 2.1 / 2.1 | 0.14 / 0.17 / 0.21 / 0.26 | 0.0 / 0.0 / 0.1 |
| pull-pfb | 2 | 96 | 240 | 0 | 51.8 / 61.9 / 79.9 / 84.6 | 0.14 / 0.17 / 0.20 / 0.25 | 0.0 / 0.0 / 0.1 |
| riptide-ms-pfb | 2 | 96 | 240 | 0 | 2.7 / 3.4 / 3.5 / 3.5 | 0.15 / 0.18 / 0.22 / 0.25 | 0.0 / 0.1 / 0.2 |
| spegel-pfb | 2 | 96 | 240 | 0 | 20.2 / 31.2 / 42.7 / 46.1 | 0.15 / 0.17 / 0.17 / 0.20 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pfb | 2 | 4.1 / 4.7 / 4.8 / 4.9 | 1.06 / 1.23 / 1.35 / 1.40 |
| pull-pfb | 2 | 53.1 / 62.1 / 80.2 / 84.9 | 1.09 / 1.32 / 1.37 / 1.43 |
| riptide-ms-pfb | 2 | 8.0 / 11.1 / 11.2 / 11.2 | 1.81 / 2.22 / 2.52 / 2.54 |
| spegel-pfb | 2 | 20.9 / 31.5 / 43.1 / 46.5 | 1.10 / 1.25 / 1.29 / 1.36 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pfb | 2 | 0.96 / 3.00 | 0.13 / 0.21 | 69 / 74 / 77 | 77 | 3.1 | 0/192 (0%) | — |
| pull-pfb | 2 | 0.95 / 1.22 | 0.12 / 0.19 | 115 / 130 / 148 | 148 | 1.6 | 0/192 (0%) | — |
| riptide-ms-pfb | 2 | 1.72 / 7.68 | 0.20 / 0.37 | 75 / 83 / 84 | 84 | 2.9 | 0/192 (0%) | — |
| spegel-pfb | 2 | 0.95 / 1.22 | 0.14 / 0.21 | 87 / 96 / 110 | 110 | 2.2 | 0/192 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pfb | 2 | 628 | 168 | 26808 |
| pull-pfb | 2 | 1889 | 2 | 13708 |
| riptide-ms-pfb | 2 | 475 | 1 | 3781 |
| spegel-pfb | 2 | 1937 | 838 | 21090 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pfb | 2 | 1 / 7 / 77 | 0.2 / 34.1 | nydus-dragonfly 0.05, gvisor_sentry 0.03, kubelet 0.03, bash 0.02, containerd 0.02, ateworker-gviso 0.02 |
| pull-pfb | 2 | 1 / 29 / 64 | 0.1 / 18.4 | atelet 1.12, kworker 0.04, kubelet 0.03, gvisor_sentry 0.03, ateworker-gviso 0.02, bash 0.02 |
| riptide-ms-pfb | 2 | 1 / 27 / 81 | 0.1 / 10.4 | gcfsd-v2 0.62, gvisor_sentry 0.15, riptide 0.14, kubelet 0.04, ateworker-gviso 0.03, bash 0.02 |
| spegel-pfb | 2 | 1 / 46 / 74 | 0.2 / 55.3 | spegel-preload 0.85, kworker 0.04, kubelet 0.03, gvisor_sentry 0.03, ateworker-gviso 0.02, bash 0.02 |
