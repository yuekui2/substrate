### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-ob | 2 | 96 | 240 | 0 | 1.4 / 1.7 / 1.7 / 1.8 | 0.15 / 0.18 / 0.22 / 0.28 | 0.0 / 0.0 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-ob | 2 | 4.1 / 4.5 / 4.8 / 4.9 | 1.09 / 1.26 / 1.34 / 1.40 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-ob | 2 | 0.96 / 2.93 | 0.13 / 0.20 | 67 / 74 / 75 | 75 | 3.2 | 0/192 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-ob | 2 | 419 | 140 | 21031 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-ob | 2 | 1 / 6 / 38 | 0.2 / 34.3 | kubelet 0.03, gvisor_sentry 0.03, bash 0.02, containerd 0.02, ateworker-gviso 0.02, containerd-shim 0.01 |
