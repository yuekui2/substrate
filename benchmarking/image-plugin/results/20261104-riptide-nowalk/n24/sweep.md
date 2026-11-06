### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| riptide-ms-pf | 2 | 96 | 360 | 0 | 3.5 / 4.4 / 4.9 / 5.0 | 0.17 / 0.19 / 0.22 / 0.30 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| riptide-ms-pf | 2 | 7.2 / 8.4 / 8.4 / 8.4 | 0.91 / 2.19 / 2.28 / 2.30 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| riptide-ms-pf | 2 | 1.69 / 4.71 | 0.20 / 0.31 | 73 / 82 / 84 | 84 | 4.3 | 0/288 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| riptide-ms-pf | 2 | 395 | 1 | 3782 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| riptide-ms-pf | 2 | 1 / 24 / 76 | 0.1 / 19.2 | gcfsd-v2 0.49, gvisor_sentry 0.15, ateworker-gviso 0.04, kubelet 0.04, bash 0.02, fluent-bit 0.02 |
