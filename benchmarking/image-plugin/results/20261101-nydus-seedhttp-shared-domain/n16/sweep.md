### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 96 | 240 | 0 | 1.6 / 1.8 / 1.8 / 1.8 | 0.15 / 0.18 / 0.21 / 0.25 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-shared | 2 | 2.3 / 2.5 / 2.5 / 2.5 | 0.23 / 0.28 / 0.31 / 0.41 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 0.08 / 0.70 | 0.11 / 0.21 | 64 / 69 / 72 | 72 | 3.3 | 0/192 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-shared | 2 | 232 | 89 | 23373 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-shared | 2 | 1 / 4 / 34 | 0.3 / 27.8 | nydus-dragonfly 0.11, kubelet 0.03, nydusd 0.02, containerd 0.02, ateworker-gviso 0.02, cilium-agent 0.01 |
