### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 96 | 240 | 0 | 2.6 / 4.0 / 4.3 / 4.3 | 0.16 / 0.19 / 0.25 / 0.25 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-shared | 2 | 3.5 / 4.8 / 5.1 / 5.1 | 0.23 / 0.29 / 0.32 / 0.40 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 0.09 / 0.93 | 0.11 / 0.20 | 66 / 71 / 73 | 73 | 3.3 | 0/192 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-shared | 2 | 331 | 86 | 28577 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-shared | 2 | 1 / 4 / 37 | 0.3 / 32.4 | nydus-dragonfly 0.12, nydusd 0.03, kubelet 0.03, ateworker-gviso 0.02, containerd 0.02, cilium-agent 0.02 |
