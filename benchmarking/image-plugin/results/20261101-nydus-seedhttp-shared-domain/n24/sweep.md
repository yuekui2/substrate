### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 96 | 360 | 0 | 2.2 / 2.2 / 2.2 / 2.2 | 0.15 / 0.18 / 0.24 / 0.29 | 0.0 / 0.0 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-shared | 2 | 3.1 / 3.8 / 3.9 / 3.9 | 0.23 / 0.28 / 0.36 / 0.40 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 0.09 / 1.36 | 0.12 / 0.21 | 65 / 70 / 70 | 70 | 5.1 | 0/288 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-shared | 2 | 439 | 128 | 35917 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-shared | 2 | 1 / 6 / 54 | 0.3 / 27.7 | nydus-dragonfly 0.19, nydusd 0.04, kubelet 0.03, ateworker-gviso 0.02, containerd 0.02, cilium-agent 0.02 |
