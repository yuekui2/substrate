### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pfc | 2 | 96 | 360 | 0 | 2.0 / 5.2 / 5.4 / 6.0 | 0.14 / 0.17 / 0.19 / 0.24 | 0.0 / 0.1 / 0.1 |
| pull-pfc | 2 | 96 | 360 | 0 | 62.6 / 99.8 / 115.9 / 118.2 | 0.15 / 0.18 / 0.21 / 0.25 | 0.0 / 0.1 / 0.1 |
| riptide-ms-pfc | 2 | 96 | 360 | 0 | 4.5 / 6.3 / 6.4 / 6.5 | 0.15 / 0.19 / 0.28 / 0.33 | 0.0 / 0.2 / 0.2 |
| spegel-pfc | 2 | 96 | 360 | 0 | 31.9 / 65.6 / 72.6 / 74.5 | 0.15 / 0.22 / 0.26 / 0.28 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pfc | 2 | 5.2 / 7.1 / 7.6 / 8.0 | 0.54 / 1.24 / 1.28 / 1.35 |
| pull-pfc | 2 | 63.3 / 99.9 / 116.2 / 118.6 | 0.53 / 1.38 / 1.48 / 1.54 |
| riptide-ms-pfc | 2 | 10.6 / 13.4 / 13.5 / 13.7 | 0.94 / 2.28 / 2.44 / 2.54 |
| spegel-pfc | 2 | 33.2 / 65.7 / 72.9 / 74.6 | 0.58 / 1.36 / 1.44 / 1.53 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pfc | 2 | 0.94 / 3.54 | 0.13 / 0.19 | 69 / 77 / 79 | 79 | 4.5 | 0/288 (0%) | — |
| pull-pfc | 2 | 0.39 / 1.26 | 0.14 / 0.21 | 127 / 164 / 182 | 182 | 2.0 | 0/288 (0%) | — |
| riptide-ms-pfc | 2 | 1.73 / 8.37 | 0.20 / 0.47 | 75 / 87 / 88 | 88 | 4.1 | 0/288 (0%) | — |
| spegel-pfc | 2 | 0.41 / 1.29 | 0.14 / 0.22 | 98 / 129 / 141 | 141 | 2.6 | 0/288 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pfc | 2 | 544 | 175 | 37544 |
| pull-pfc | 2 | 1893 | 2 | 15690 |
| riptide-ms-pfc | 2 | 472 | 148 | 3714 |
| spegel-pfc | 2 | 1915 | 868 | 22945 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pfc | 2 | 1 / 8 / 80 | 0.2 / 34.9 | nydus-dragonfly 0.06, gvisor_sentry 0.05, kubelet 0.03, bash 0.03, nydusd-amp 0.02, ateworker-gviso 0.02 |
| pull-pfc | 2 | 2 / 32 / 73 | 0.2 / 16.4 | atelet 1.83, kworker 0.06, gvisor_sentry 0.04, kubelet 0.04, ateworker-gviso 0.03, bash 0.02 |
| riptide-ms-pfc | 2 | 1 / 53 / 83 | 0.1 / 7.3 | gcfsd-v2 0.91, riptide 0.21, gvisor_sentry 0.18, ateworker-gviso 0.04, kubelet 0.04, fluent-bit 0.02 |
| spegel-pfc | 2 | 1 / 44 / 85 | 0.2 / 75.6 | spegel-preload 0.94, kworker 0.05, kubelet 0.03, gvisor_sentry 0.02, bash 0.02, ateworker-gviso 0.02 |
