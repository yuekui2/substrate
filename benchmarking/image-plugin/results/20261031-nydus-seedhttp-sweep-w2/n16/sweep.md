### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 2 | 96 | 240 | 0 | 2.1 / 3.5 / 4.1 / 4.2 | 0.16 / 0.18 / 0.19 / 0.20 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus | 2 | 4.6 / 5.7 / 6.0 / 6.0 | 0.50 / 1.20 / 1.25 / 1.25 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 2 | 0.94 / 2.93 | 0.13 / 0.20 | 69 / 75 / 75 | 75 | 3.2 | 0/192 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 2 | 1 | 0 | 89 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus | 2 | 1 / 1 / 2 | 0.1 / 0.4 | kubelet 0.02, containerd 0.02, cilium-agent 0.01, containerd-shim 0.01, fluent-bit 0.01, prometheus 0.00 |
