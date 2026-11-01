### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 96 | 240 | 0 | 2.0 / 2.8 / 2.9 / 3.0 | 0.16 / 0.19 / 0.19 / 0.27 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pf | 2 | 4.6 / 5.9 / 6.0 / 6.0 | 0.55 / 1.24 / 1.28 / 1.30 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 0.96 / 3.68 | 0.14 / 0.21 | 69 / 75 / 80 | 80 | 3.0 | 0/192 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pf | 2 | 468 | 76 | 27055 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pf | 2 | 1 / 7 / 39 | 0.3 / 21.6 | nydus-dragonfly 0.05, kubelet 0.03, nydusd 0.03, ateworker-gviso 0.02, containerd 0.02, cilium-agent 0.01 |
