### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pfc | 2 | 96 | 480 | 0 | 2.3 / 4.8 / 5.4 / 5.6 | 0.14 / 0.17 / 0.25 / 0.28 | 0.0 / 0.1 / 0.1 |
| pull-pfc | 2 | 96 | 480 | 0 | 71.8 / 125.9 / 135.6 / 137.7 | 0.15 / 0.22 / 0.27 / 0.31 | 0.0 / 0.1 / 0.1 |
| riptide-ms-pfc | 2 | 96 | 480 | 0 | 6.4 / 15.3 / 15.5 / 15.5 | 0.16 / 0.21 / 0.26 / 0.33 | 0.0 / 0.2 / 0.2 |
| spegel-pfc | 2 | 96 | 480 | 0 | 33.4 / 61.3 / 71.9 / 72.1 | 0.15 / 0.21 / 0.33 / 0.48 | 0.0 / 0.2 / 0.2 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pfc | 2 | 6.2 / 7.2 / 7.6 / 8.1 | 0.52 / 1.25 / 1.31 / 1.37 |
| pull-pfc | 2 | 72.8 / 125.9 / 135.7 / 138.1 | 0.59 / 1.46 / 1.54 / 1.61 |
| riptide-ms-pfc | 2 | 17.7 / 22.2 / 22.3 / 22.4 | 1.01 / 2.66 / 2.80 / 2.98 |
| spegel-pfc | 2 | 34.1 / 61.3 / 72.3 / 72.6 | 0.57 / 1.33 / 1.41 / 1.79 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pfc | 2 | 0.95 / 4.29 | 0.13 / 0.21 | 69 / 74 / 80 | 80 | 6.0 | 0/384 (0%) | — |
| pull-pfc | 2 | 0.40 / 1.31 | 0.14 / 0.22 | 139 / 191 / 204 | 204 | 2.4 | 0/384 (0%) | — |
| riptide-ms-pfc | 2 | 1.77 / 10.92 | 0.22 / 0.59 | 83 / 95 / 96 | 96 | 5.0 | 0/384 (0%) | — |
| spegel-pfc | 2 | 0.43 / 1.36 | 0.15 / 0.25 | 98 / 124 / 135 | 135 | 3.6 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pfc | 2 | 761 | 208 | 36619 |
| pull-pfc | 2 | 1904 | 50 | 17735 |
| riptide-ms-pfc | 2 | 591 | 0 | 4422 |
| spegel-pfc | 2 | 1938 | 875 | 27048 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pfc | 2 | 2 / 12 / 81 | 0.2 / 39.2 | nydus-dragonfly 0.10, gvisor_sentry 0.06, nydusd-amp 0.04, kubelet 0.03, bash 0.03, ateworker-gviso 0.03 |
| pull-pfc | 2 | 2 / 37 / 74 | 0.2 / 28.3 | atelet 1.96, kworker 0.06, gvisor_sentry 0.04, kubelet 0.04, ateworker-gviso 0.03, bash 0.02 |
| riptide-ms-pfc | 2 | 1 / 76 / 85 | 0.1 / 18.1 | gcfsd-v2 1.18, riptide 0.27, gvisor_sentry 0.26, ateworker-gviso 0.06, kubelet 0.04, fluent-bit 0.03 |
| spegel-pfc | 2 | 1 / 46 / 82 | 0.2 / 57.7 | spegel-preload 0.98, kworker 0.05, gvisor_sentry 0.03, kubelet 0.03, ateworker-gviso 0.02, bash 0.02 |
