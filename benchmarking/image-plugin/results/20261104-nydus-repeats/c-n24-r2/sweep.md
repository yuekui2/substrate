### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-oc | 2 | 96 | 360 | 0 | 1.8 / 2.3 / 2.4 / 2.5 | 0.15 / 0.19 / 0.24 / 0.32 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-oc | 2 | 4.0 / 5.2 / 5.2 / 5.2 | 0.55 / 1.24 / 1.29 / 1.30 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-oc | 2 | 0.95 / 3.15 | 0.14 / 0.21 | 67 / 74 / 78 | 78 | 4.6 | 0/288 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-oc | 2 | 601 | 148 | 25540 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-oc | 2 | 1 / 9 / 48 | 0.2 / 39.2 | gvisor_sentry 0.05, kubelet 0.03, bash 0.03, ateworker-gviso 0.02, nydusd-amp 0.02, containerd 0.02 |
