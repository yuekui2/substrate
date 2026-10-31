### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 8 | 384 | 15 | 0 | 1.2 / 1.2 / 1.2 / 1.2 | 0.17 / 0.20 / 0.21 / 0.21 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus | 8 | 3.5 / 6.6 / 6.8 / 6.9 | 1.58 / 4.85 / 4.86 / 4.86 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 8 | 1.47 / 5.00 | 0.47 / 0.59 | 75 / 81 / 82 | 82 | 0.2 | 0/12 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 8 | 176 | 10 | 2292 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus | 8 | 1 / 2 / 4 | 0.1 / 3.9 | kubelet 0.03, containerd 0.02, containerd-shim 0.01, cilium-agent 0.01, fluent-bit 0.01, ateworker-gviso 0.01 |
