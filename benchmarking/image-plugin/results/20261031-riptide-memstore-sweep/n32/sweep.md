### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| riptide-ms | 8 | 384 | 480 | 0 | 5.5 / 15.2 / 15.3 / 15.5 | 0.20 / 0.27 / 0.30 / 0.38 | 0.0 / 0.0 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| riptide-ms | 8 | 17.7 / 24.4 / 28.7 / 28.8 | 2.84 / 9.18 / 9.87 / 10.45 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| riptide-ms | 8 | 4.84 / 14.13 | 0.65 / 0.91 | 95 / 128 / 131 | 131 | 3.7 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| riptide-ms | 8 | 603 | 1 | 3344 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| riptide-ms | 8 | 1 / 57 / 80 | 0.1 / 4.6 | gcfsd-v2 0.96, riptide 0.23, gvisor_sentry 0.20, ateworker-gviso 0.05, kubelet 0.04, fluent-bit 0.02 |
