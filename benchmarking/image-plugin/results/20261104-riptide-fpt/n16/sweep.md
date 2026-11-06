### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| riptide-ms-pf | 2 | 96 | 240 | 0 | 2.7 / 3.3 / 5.1 / 6.6 | 0.16 / 0.20 / 0.28 / 0.31 | 0.0 / 0.0 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| riptide-ms-pf | 2 | 7.1 / 9.2 / 9.4 / 9.5 | 0.85 / 2.12 / 2.20 / 2.33 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| riptide-ms-pf | 2 | 1.66 / 6.24 | 0.20 / 0.36 | 70 / 82 / 87 | 87 | 2.8 | 0/192 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| riptide-ms-pf | 2 | 522 | 7 | 4333 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| riptide-ms-pf | 2 | 1 / 24 / 81 | 0.2 / 12.0 | gcfsd-v2 0.58, riptide 0.15, gvisor_sentry 0.13, kubelet 0.04, ateworker-gviso 0.03, bash 0.02 |
