### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| riptide-ms | 8 | 384 | 240 | 0 | 2.9 / 5.1 / 5.1 / 5.1 | 0.18 / 0.25 / 0.32 / 0.38 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| riptide-ms | 8 | 10.1 / 17.0 / 17.0 / 17.0 | 2.82 / 8.40 / 8.68 / 8.86 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| riptide-ms | 8 | 5.12 / 11.15 | 0.62 / 0.78 | 86 / 115 / 116 | 116 | 2.1 | 0/192 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| riptide-ms | 8 | 456 | 19 | 3120 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| riptide-ms | 8 | 2 / 20 / 79 | 0.2 / 3.8 | gcfsd-v2 0.63, riptide 0.15, gvisor_sentry 0.11, kubelet 0.04, ateworker-gviso 0.03, containerd 0.02 |
