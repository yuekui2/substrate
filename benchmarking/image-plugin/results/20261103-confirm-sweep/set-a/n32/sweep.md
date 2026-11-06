### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 96 | 480 | 0 | 2.2 / 4.5 / 4.8 / 4.9 | 0.15 / 0.18 / 0.24 / 0.28 | 0.0 / 0.1 / 0.1 |
| pull-pf | 2 | 96 | 480 | 0 | 75.3 / 138.2 / 147.6 / 149.4 | 0.15 / 0.21 / 0.27 / 0.31 | 0.0 / 0.1 / 0.1 |
| riptide-ms-pf | 2 | 96 | 480 | 0 | 4.2 / 7.6 / 8.1 / 13.7 | 0.16 / 0.20 / 0.27 / 0.34 | 0.0 / 0.2 / 0.2 |
| spegel-pf | 2 | 96 | 480 | 0 | 34.2 / 77.0 / 82.2 / 82.2 | 0.16 / 0.24 / 0.28 / 0.37 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pf | 2 | 6.2 / 7.1 / 7.3 / 7.7 | 0.48 / 1.27 / 1.34 / 1.47 |
| pull-pf | 2 | 76.7 / 138.5 / 148.0 / 149.7 | 0.55 / 1.48 / 1.58 / 1.66 |
| riptide-ms-pf | 2 | 12.6 / 15.4 / 15.6 / 16.4 | 0.81 / 2.26 / 2.48 / 2.61 |
| spegel-pf | 2 | 34.8 / 77.2 / 82.4 / 82.4 | 0.52 / 1.37 / 1.56 / 1.66 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 0.95 / 4.42 | 0.13 / 0.20 | 70 / 76 / 81 | 81 | 5.9 | 0/384 (0%) | — |
| pull-pf | 2 | 0.38 / 1.34 | 0.14 / 0.23 | 141 / 203 / 218 | 218 | 2.2 | 0/384 (0%) | — |
| riptide-ms-pf | 2 | 1.73 / 10.19 | 0.20 / 0.50 | 78 / 89 / 91 | 91 | 5.3 | 0/384 (0%) | — |
| spegel-pf | 2 | 0.38 / 1.32 | 0.15 / 0.24 | 98 / 138 / 147 | 147 | 3.3 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pf | 2 | 702 | 187 | 35784 |
| pull-pf | 2 | 1919 | 30 | 21965 |
| riptide-ms-pf | 2 | 677 | 1 | 4482 |
| spegel-pf | 2 | 1905 | 794 | 26919 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pf | 2 | 1 / 13 / 81 | 0.2 / 40.3 | nydus-dragonfly 0.10, gvisor_sentry 0.06, nydusd-amp 0.04, kubelet 0.03, ateworker-gviso 0.03, bash 0.03 |
| pull-pf | 2 | 2 / 37 / 67 | 0.2 / 23.6 | atelet 2.10, kworker 0.07, gvisor_sentry 0.04, kubelet 0.04, ateworker-gviso 0.03, bash 0.02 |
| riptide-ms-pf | 2 | 1 / 76 / 84 | 0.1 / 4.6 | gcfsd-v2 1.18, riptide 0.28, gvisor_sentry 0.24, ateworker-gviso 0.06, kubelet 0.04, fluent-bit 0.03 |
| spegel-pf | 2 | 1 / 54 / 82 | 0.2 / 60.0 | spegel-preload 1.23, kworker 0.06, gvisor_sentry 0.03, kubelet 0.03, ateworker-gviso 0.02, bash 0.02 |
