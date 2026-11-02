### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 96 | 480 | 0 | 2.3 / 5.2 / 5.8 / 5.9 | 0.16 / 0.21 / 0.28 / 0.33 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pf | 2 | 6.4 / 7.2 / 8.2 / 8.5 | 0.51 / 1.28 / 1.37 / 1.44 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 0.96 / 4.33 | 0.14 / 0.22 | 70 / 76 / 79 | 79 | 6.1 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pf | 2 | 297 | 131 | 26379 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pf | 2 | 1 / 12 / 66 | 0.3 / 18.8 | nydus-dragonfly 0.10, nydusd-bootfill 0.04, ateworker-gviso 0.03, kubelet 0.03, containerd 0.02, fluent-bit 0.02 |
