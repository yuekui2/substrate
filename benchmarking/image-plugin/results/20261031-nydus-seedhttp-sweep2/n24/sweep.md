### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 8 | 384 | 360 | 0 | 4.1 / 6.7 / 7.1 / 7.1 | 0.17 / 0.24 / 0.26 / 0.28 | 0.0 / 0.0 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus | 8 | 9.0 / 13.2 / 13.3 / 13.3 | 2.14 / 5.44 / 5.64 / 5.78 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 8 | 3.72 / 7.01 | 0.50 / 0.67 | 84 / 99 / 104 | 104 | 3.5 | 0/288 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 8 | 440 | 74 | 28409 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus | 8 | 1 / 8 / 51 | 0.3 / 24.5 | nydus-dragonfly 0.07, gvisor_sentry 0.05, nydusd 0.03, kubelet 0.03, ateworker-gviso 0.02, containerd 0.02 |
