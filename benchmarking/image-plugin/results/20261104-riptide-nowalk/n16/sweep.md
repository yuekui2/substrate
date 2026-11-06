### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| riptide-ms-pf | 2 | 96 | 240 | 0 | 2.3 / 3.0 / 3.3 / 3.5 | 0.16 / 0.18 / 0.21 / 0.27 | 0.0 / 0.0 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| riptide-ms-pf | 2 | 6.2 / 6.7 / 6.9 / 7.0 | 0.85 / 1.98 / 2.11 / 2.15 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| riptide-ms-pf | 2 | 1.65 / 4.05 | 0.19 / 0.27 | 72 / 80 / 83 | 83 | 2.9 | 0/192 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| riptide-ms-pf | 2 | 611 | 4 | 4343 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| riptide-ms-pf | 2 | 1 / 15 / 63 | 0.1 / 21.4 | gcfsd-v2 0.32, gvisor_sentry 0.09, kubelet 0.04, ateworker-gviso 0.02, bash 0.02, containerd 0.02 |
