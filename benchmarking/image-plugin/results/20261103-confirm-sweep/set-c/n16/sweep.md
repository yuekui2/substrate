### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pfc | 2 | 96 | 240 | 0 | 1.4 / 1.9 / 2.2 / 2.4 | 0.14 / 0.17 / 0.23 / 0.26 | 0.0 / 0.0 / 0.0 |
| pull-pfc | 2 | 96 | 240 | 0 | 53.6 / 86.6 / 92.3 / 95.9 | 0.14 / 0.17 / 0.21 / 0.22 | 0.0 / 0.1 / 0.2 |
| riptide-ms-pfc | 2 | 96 | 240 | 0 | 3.1 / 4.0 / 4.1 / 4.1 | 0.15 / 0.18 / 0.21 / 0.26 | 0.0 / 0.1 / 0.1 |
| spegel-pfc | 2 | 96 | 240 | 0 | 25.1 / 49.4 / 51.0 / 51.0 | 0.15 / 0.18 / 0.21 / 0.26 | 0.0 / 0.1 / 0.2 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pfc | 2 | 3.3 / 4.6 / 4.9 / 4.9 | 0.45 / 1.19 / 1.24 / 1.31 |
| pull-pfc | 2 | 54.6 / 86.8 / 92.7 / 96.2 | 0.46 / 1.28 / 1.37 / 1.46 |
| riptide-ms-pfc | 2 | 8.1 / 9.6 / 9.7 / 9.7 | 0.71 / 2.04 / 2.15 / 2.17 |
| spegel-pfc | 2 | 25.7 / 49.8 / 51.1 / 51.1 | 0.47 / 1.25 / 1.32 / 1.38 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pfc | 2 | 0.91 / 2.96 | 0.12 / 0.20 | 66 / 74 / 76 | 76 | 3.1 | 0/192 (0%) | — |
| pull-pfc | 2 | 0.33 / 1.19 | 0.12 / 0.20 | 118 / 150 / 165 | 165 | 1.5 | 0/192 (0%) | — |
| riptide-ms-pfc | 2 | 1.56 / 5.92 | 0.19 / 0.33 | 72 / 81 / 83 | 83 | 2.9 | 0/192 (0%) | — |
| spegel-pfc | 2 | 0.33 / 1.21 | 0.14 / 0.20 | 90 / 112 / 118 | 118 | 2.0 | 0/192 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pfc | 2 | 540 | 172 | 29571 |
| pull-pfc | 2 | 1889 | 1 | 14127 |
| riptide-ms-pfc | 2 | 377 | 1 | 3320 |
| spegel-pfc | 2 | 1951 | 862 | 21108 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pfc | 2 | 1 / 6 / 70 | 0.2 / 32.9 | nydus-dragonfly 0.05, kubelet 0.03, gvisor_sentry 0.02, bash 0.02, containerd 0.02, ateworker-gviso 0.02 |
| pull-pfc | 2 | 1 / 29 / 45 | 0.2 / 15.5 | atelet 1.43, kworker 0.05, kubelet 0.03, ateworker-gviso 0.02, bash 0.02, gvisor_sentry 0.02 |
| riptide-ms-pfc | 2 | 1 / 25 / 81 | 0.1 / 9.2 | gcfsd-v2 0.64, riptide 0.17, gvisor_sentry 0.10, kubelet 0.04, ateworker-gviso 0.03, bash 0.02 |
| spegel-pfc | 2 | 1 / 41 / 81 | 0.2 / 64.0 | spegel-preload 0.89, kworker 0.04, kubelet 0.03, bash 0.02, gvisor_sentry 0.02, ateworker-gviso 0.02 |
