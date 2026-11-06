### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| riptide-ms-pf | 2 | 96 | 480 | 0 | 4.2 / 5.7 / 5.8 / 6.0 | 0.17 / 0.23 / 0.30 / 0.35 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| riptide-ms-pf | 2 | 8.0 / 9.9 / 10.0 / 10.2 | 0.86 / 2.45 / 2.64 / 2.79 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| riptide-ms-pf | 2 | 1.70 / 4.76 | 0.21 / 0.32 | 74 / 82 / 86 | 86 | 5.6 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| riptide-ms-pf | 2 | 716 | 1 | 4560 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| riptide-ms-pf | 2 | 1 / 33 / 79 | 0.1 / 11.8 | gcfsd-v2 0.61, gvisor_sentry 0.20, ateworker-gviso 0.05, kubelet 0.04, fluent-bit 0.03, bash 0.02 |
