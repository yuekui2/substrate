### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 2 | 96 | 480 | 0 | 6.7 / 10.2 / 10.5 / 10.7 | 0.16 / 0.19 / 0.21 / 0.32 | 0.0 / 0.0 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus | 2 | 8.9 / 12.6 / 13.1 / 13.2 | 0.49 / 1.27 / 1.33 / 1.41 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 2 | 0.94 / 3.38 | 0.14 / 0.22 | 74 / 82 / 84 | 84 | 5.7 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 2 | 333 | 115 | 33517 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus | 2 | 1 / 12 / 57 | 0.3 / 28.3 | nydus-dragonfly 0.11, nydusd 0.06, ateworker-gviso 0.03, kubelet 0.03, containerd 0.02, fluent-bit 0.02 |
