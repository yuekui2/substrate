### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 8 | 384 | 240 | 0 | 2.2 / 3.3 / 3.7 / 3.7 | 0.17 / 0.22 / 0.24 / 0.25 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus | 8 | 6.8 / 9.6 / 9.8 / 9.8 | 2.02 / 5.17 / 5.26 / 5.27 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 8 | 3.13 / 6.33 | 0.47 / 0.65 | 82 / 94 / 95 | 95 | 2.5 | 0/192 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 8 | 152 | 57 | 17607 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus | 8 | 1 / 4 / 30 | 0.2 / 19.4 | kubelet 0.03, nydus-dragonfly 0.03, containerd 0.02, gvisor_sentry 0.01, cilium-agent 0.01, containerd-shim 0.01 |
