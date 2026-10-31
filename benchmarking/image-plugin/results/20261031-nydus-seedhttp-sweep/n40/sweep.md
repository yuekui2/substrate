### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 8 | 384 | 600 | 0 | 7.9 / 13.5 / 14.2 / 14.8 | 0.18 / 0.25 / 0.29 / 0.34 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus | 8 | 13.2 / 18.9 / 20.3 / 20.8 | 2.03 / 5.60 / 5.92 / 6.10 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 8 | 3.56 / 7.50 | 0.52 / 0.68 | 86 / 107 / 110 | 110 | 5.4 | 0/480 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 8 | 53 | 59 | 5553 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus | 8 | 1 / 4 / 5 | 0.3 / 8.1 | kubelet 0.02, containerd 0.02, cilium-agent 0.01, containerd-shim 0.01, gvisor_sentry 0.01, fluent-bit 0.01 |
