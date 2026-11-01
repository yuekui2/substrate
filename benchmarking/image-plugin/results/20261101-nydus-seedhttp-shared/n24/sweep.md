### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 96 | 360 | 0 | 3.9 / 5.7 / 6.0 / 6.3 | 0.15 / 0.18 / 0.19 / 0.27 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-shared | 2 | 4.9 / 6.6 / 6.9 / 7.1 | 0.24 / 0.28 / 0.29 / 0.42 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 0.09 / 0.97 | 0.12 / 0.20 | 66 / 72 / 74 | 74 | 4.9 | 0/288 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-shared | 2 | 433 | 113 | 32880 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-shared | 2 | 1 / 5 / 50 | 0.3 / 31.6 | nydus-dragonfly 0.19, nydusd 0.05, kubelet 0.03, ateworker-gviso 0.02, containerd 0.02, cilium-agent 0.02 |
