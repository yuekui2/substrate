### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 96 | 360 | 0 | 2.0 / 3.2 / 4.9 / 5.3 | 0.14 / 0.17 / 0.22 / 0.27 | 0.0 / 0.1 / 0.1 |
| pull-pf | 2 | 96 | 360 | 0 | 64.8 / 102.0 / 116.7 / 118.0 | 0.14 / 0.18 / 0.23 / 0.28 | 0.0 / 0.1 / 0.1 |
| riptide-ms-pf | 2 | 96 | 360 | 0 | 3.4 / 4.6 / 4.8 / 5.2 | 0.16 / 0.19 / 0.24 / 0.28 | 0.0 / 0.1 / 0.2 |
| spegel-pf | 2 | 96 | 360 | 0 | 26.8 / 50.8 / 66.4 / 67.8 | 0.15 / 0.18 / 0.22 / 0.26 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pf | 2 | 4.7 / 6.7 / 7.0 / 7.0 | 0.53 / 1.25 / 1.28 / 1.33 |
| pull-pf | 2 | 65.5 / 102.3 / 116.9 / 118.0 | 0.55 / 1.37 / 1.44 / 1.68 |
| riptide-ms-pf | 2 | 9.9 / 12.1 / 12.1 / 12.1 | 0.93 / 2.23 / 2.36 / 2.37 |
| spegel-pf | 2 | 27.1 / 51.0 / 66.6 / 67.9 | 0.53 / 1.29 / 1.38 / 1.42 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 0.95 / 3.17 | 0.13 / 0.20 | 69 / 74 / 75 | 75 | 4.8 | 0/288 (0%) | — |
| pull-pf | 2 | 0.41 / 1.29 | 0.13 / 0.21 | 129 / 165 / 184 | 184 | 2.0 | 0/288 (0%) | — |
| riptide-ms-pf | 2 | 1.74 / 8.07 | 0.20 / 0.47 | 76 / 85 / 86 | 86 | 4.2 | 0/288 (0%) | — |
| spegel-pf | 2 | 0.40 / 1.26 | 0.14 / 0.22 | 91 / 116 / 129 | 129 | 2.8 | 0/288 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pf | 2 | 639 | 166 | 36114 |
| pull-pf | 2 | 1906 | 2 | 14264 |
| riptide-ms-pf | 2 | 563 | 1 | 3710 |
| spegel-pf | 2 | 1896 | 673 | 22854 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pf | 2 | 1 / 7 / 80 | 0.2 / 39.3 | nydus-dragonfly 0.05, gvisor_sentry 0.03, kubelet 0.03, bash 0.03, nydusd-amp 0.02, containerd 0.02 |
| pull-pf | 2 | 2 / 32 / 64 | 0.3 / 20.2 | atelet 1.84, kworker 0.06, gvisor_sentry 0.04, kubelet 0.03, ateworker-gviso 0.03, bash 0.02 |
| riptide-ms-pf | 2 | 1 / 54 / 83 | 0.1 / 7.9 | gcfsd-v2 0.90, riptide 0.25, gvisor_sentry 0.21, ateworker-gviso 0.04, kubelet 0.04, fluent-bit 0.02 |
| spegel-pf | 2 | 1 / 53 / 89 | 0.2 / 51.0 | spegel-preload 1.13, kworker 0.06, kubelet 0.03, gvisor_sentry 0.03, ateworker-gviso 0.02, bash 0.02 |
