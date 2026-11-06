### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-ob | 2 | 96 | 240 | 0 | 1.3 / 1.6 / 1.8 / 1.8 | 0.15 / 0.17 / 0.20 / 0.28 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-ob | 2 | 3.9 / 4.4 / 4.5 / 4.5 | 1.09 / 1.21 / 1.32 / 1.35 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-ob | 2 | 0.96 / 2.89 | 0.13 / 0.21 | 67 / 73 / 76 | 76 | 3.1 | 0/192 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-ob | 2 | 479 | 132 | 22044 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-ob | 2 | 1 / 7 / 38 | 0.3 / 33.7 | kubelet 0.03, gvisor_sentry 0.03, bash 0.02, containerd 0.02, ateworker-gviso 0.02, cilium-agent 0.01 |
