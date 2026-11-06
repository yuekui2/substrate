### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| riptide-ms-pf | 2 | 96 | 360 | 0 | 3.3 / 5.5 / 6.0 / 6.0 | 0.17 / 0.20 / 0.26 / 0.31 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| riptide-ms-pf | 2 | 10.3 / 13.2 / 13.3 / 13.5 | 0.97 / 2.36 / 2.48 / 2.53 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| riptide-ms-pf | 2 | 1.69 / 8.61 | 0.21 / 0.45 | 75 / 85 / 90 | 90 | 4.0 | 0/288 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| riptide-ms-pf | 2 | 772 | 6 | 4880 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| riptide-ms-pf | 2 | 1 / 52 / 83 | 0.1 / 9.3 | gcfsd-v2 0.92, riptide 0.24, gvisor_sentry 0.20, ateworker-gviso 0.04, kubelet 0.04, fluent-bit 0.02 |
