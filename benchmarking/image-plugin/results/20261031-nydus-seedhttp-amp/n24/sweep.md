### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 2 | 96 | 360 | 0 | 3.2 / 5.9 / 6.5 / 6.5 | 0.16 / 0.19 / 0.23 / 0.28 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus | 2 | 5.8 / 7.7 / 8.0 / 8.1 | 0.55 / 1.25 / 1.31 / 1.41 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 2 | 0.95 / 2.94 | 0.14 / 0.21 | 69 / 77 / 81 | 81 | 4.4 | 0/288 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 2 | 372 | 83 | 24663 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus | 2 | 1 / 10 / 49 | 0.3 / 20.2 | nydus-dragonfly 0.08, nydusd-amp 0.03, kubelet 0.03, ateworker-gviso 0.02, containerd 0.02, cilium-agent 0.02 |
