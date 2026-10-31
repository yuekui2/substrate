### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 8 | 384 | 480 | 0 | 5.5 / 9.2 / 9.7 / 9.9 | 0.17 / 0.24 / 0.26 / 0.34 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus | 8 | 10.1 / 15.7 / 15.8 / 16.3 | 1.90 / 5.43 / 5.59 / 5.86 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 8 | 3.09 / 6.89 | 0.51 / 0.67 | 82 / 100 / 103 | 103 | 4.6 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 8 | 114 | 39 | 13438 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus | 8 | 1 / 4 / 22 | 0.3 / 17.5 | kubelet 0.02, nydus-dragonfly 0.02, containerd 0.02, cilium-agent 0.01, containerd-shim 0.01, gvisor_sentry 0.01 |
