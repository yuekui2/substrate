### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 96 | 480 | 0 | 2.3 / 2.4 / 3.0 / 3.0 | 0.15 / 0.18 / 0.22 / 0.33 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-shared | 2 | 3.9 / 4.9 / 5.0 / 5.0 | 0.23 / 0.28 / 0.35 / 0.44 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 0.09 / 2.21 | 0.13 / 0.22 | 66 / 71 / 73 | 73 | 6.6 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-shared | 2 | 249 | 166 | 42531 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-shared | 2 | 1 / 8 / 63 | 0.3 / 23.8 | nydus-dragonfly 0.26, nydusd 0.05, kubelet 0.03, ateworker-gviso 0.03, containerd 0.02, fluent-bit 0.02 |
