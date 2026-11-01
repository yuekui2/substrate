### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 96 | 480 | 0 | 2.5 / 2.8 / 2.9 / 2.9 | 0.16 / 0.19 / 0.25 / 0.26 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-shared | 2 | 4.6 / 6.4 / 6.6 / 6.6 | 0.24 / 0.29 / 0.35 / 0.40 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 0.09 / 2.93 | 0.13 / 0.20 | 66 / 74 / 76 | 76 | 6.3 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-shared | 2 | 244 | 148 | 43244 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-shared | 2 | 1 / 8 / 70 | 0.3 / 20.8 | nydus-dragonfly 0.27, nydusd 0.04, kubelet 0.03, ateworker-gviso 0.03, containerd 0.02, fluent-bit 0.02 |
