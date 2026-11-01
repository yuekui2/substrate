### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| riptide-ms | 8 | 384 | 360 | 0 | 4.1 / 5.4 / 5.6 / 5.9 | 0.18 / 0.25 / 0.29 / 0.36 | 0.0 / 0.0 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| riptide-ms | 8 | 11.8 / 18.5 / 18.8 / 18.8 | 3.00 / 8.51 / 8.66 / 8.86 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| riptide-ms | 8 | 5.92 / 13.45 | 0.64 / 0.81 | 90 / 117 / 120 | 120 | 3.0 | 0/288 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| riptide-ms | 8 | 447 | 3 | 3149 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| riptide-ms | 8 | 5 / 58 / 79 | 0.2 / 5.7 | gcfsd-v2 0.99, riptide 0.25, gvisor_sentry 0.20, ateworker-gviso 0.05, kubelet 0.04, fluent-bit 0.02 |
