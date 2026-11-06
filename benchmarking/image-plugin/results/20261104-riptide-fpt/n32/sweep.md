### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| riptide-ms-pf | 2 | 96 | 480 | 0 | 4.6 / 10.3 / 10.4 / 10.5 | 0.17 / 0.20 / 0.25 / 0.36 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| riptide-ms-pf | 2 | 15.2 / 16.2 / 16.8 / 17.6 | 0.83 / 2.24 / 2.57 / 2.68 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| riptide-ms-pf | 2 | 1.73 / 11.06 | 0.20 / 0.55 | 79 / 91 / 94 | 94 | 5.1 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| riptide-ms-pf | 2 | 526 | 3 | 4574 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| riptide-ms-pf | 2 | 1 / 78 / 84 | 0.1 / 7.3 | gcfsd-v2 1.21, riptide 0.28, gvisor_sentry 0.26, ateworker-gviso 0.06, kubelet 0.04, fluent-bit 0.03 |
