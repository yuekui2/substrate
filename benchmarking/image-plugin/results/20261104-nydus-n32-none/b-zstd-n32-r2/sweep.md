### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-ob | 2 | 96 | 480 | 0 | 2.4 / 3.4 / 3.5 / 3.6 | 0.15 / 0.19 / 0.27 / 0.32 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-ob | 2 | 6.3 / 6.9 / 7.0 / 7.0 | 1.09 / 1.29 / 1.39 / 1.43 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-ob | 2 | 1.00 / 3.99 | 0.14 / 0.21 | 70 / 76 / 78 | 78 | 6.1 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-ob | 2 | 717 | 165 | 29816 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-ob | 2 | 1 / 10 / 66 | 0.2 / 40.3 | gvisor_sentry 0.06, nydusd-amp 0.03, kubelet 0.03, bash 0.03, ateworker-gviso 0.02, containerd 0.02 |
