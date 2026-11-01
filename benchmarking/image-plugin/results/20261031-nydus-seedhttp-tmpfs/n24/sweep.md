### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 2 | 96 | 360 | 360 | — / — / — / — | — / — / — / — | — / — / — |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus | 2 | — / — / — / — | — / — / — / — |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 2 | — / — | — / — | 62 / 68 / 71 | 71 | 0.0 | 0/0 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 2 | 100 | 0 | 894 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus | 2 | 1 / 2 / 10 | 0.1 / 0.8 | kubelet 0.03, containerd 0.02, cilium-agent 0.01, containerd-shim 0.01, atelet 0.01, nydus-dragonfly 0.01 |
