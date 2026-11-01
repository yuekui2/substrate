### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 96 | 480 | 0 | 45.6 / 83.9 / 90.1 / 91.3 | 0.19 / 0.22 / 0.23 / 0.29 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pf | 2 | 50.9 / 87.2 / 93.5 / 94.4 | 0.61 / 1.66 / 1.70 / 1.73 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 1.04 / 7.59 | 0.16 / 0.22 | 116 / 152 / 163 | 163 | 2.9 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pf | 2 | 1271 | 293 | 33504 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pf | 2 | 3 / 47 / 57 | 2.9 / 25.4 | nydusd 1.74, nydus-dragonfly 0.08, nginx 0.03, kubelet 0.03, ateworker-gviso 0.03, gvisor_sentry 0.02 |
