### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 2 | 96 | 360 | 0 | 3.1 / 8.6 / 9.0 / 9.1 | 0.16 / 0.19 / 0.24 / 0.26 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus | 2 | 6.4 / 10.7 / 11.4 / 11.5 | 0.59 / 1.26 / 1.31 / 1.35 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 2 | 0.94 / 3.22 | 0.13 / 0.21 | 70 / 79 / 82 | 82 | 4.4 | 0/288 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 2 | 356 | 100 | 31523 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus | 2 | 1 / 9 / 59 | 0.3 / 31.5 | nydus-dragonfly 0.08, nydusd 0.05, kubelet 0.03, ateworker-gviso 0.03, containerd 0.02, cilium-agent 0.02 |
