### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-ob | 2 | 96 | 360 | 0 | 1.9 / 2.4 / 2.6 / 2.6 | 0.15 / 0.18 / 0.20 / 0.24 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-ob | 2 | 5.1 / 5.7 / 5.8 / 5.8 | 1.07 / 1.23 / 1.29 / 1.37 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-ob | 2 | 0.95 / 3.43 | 0.14 / 0.21 | 68 / 76 / 78 | 78 | 4.6 | 0/288 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-ob | 2 | 699 | 134 | 28182 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-ob | 2 | 1 / 9 / 49 | 0.2 / 41.4 | gvisor_sentry 0.05, kubelet 0.03, bash 0.03, ateworker-gviso 0.03, nydusd-amp 0.02, containerd 0.02 |
