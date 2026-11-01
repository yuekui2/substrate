### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 96 | 480 | 0 | 6.9 / 10.2 / 10.5 / 10.8 | 0.16 / 0.19 / 0.24 / 0.27 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pf | 2 | 9.3 / 12.1 / 12.5 / 13.0 | 0.49 / 1.29 / 1.35 / 1.45 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 0.95 / 3.66 | 0.14 / 0.21 | 73 / 80 / 84 | 84 | 5.7 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pf | 2 | 352 | 138 | 36611 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pf | 2 | 1 / 13 / 61 | 0.3 / 28.3 | nydus-dragonfly 0.11, nydusd 0.07, ateworker-gviso 0.03, kubelet 0.03, gvisor_sentry 0.02, containerd 0.02 |
