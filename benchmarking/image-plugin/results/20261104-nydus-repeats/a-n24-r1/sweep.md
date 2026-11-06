### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 96 | 360 | 0 | 1.9 / 2.5 / 2.6 / 2.7 | 0.15 / 0.19 / 0.27 / 0.28 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-o | 2 | 4.4 / 5.4 / 5.8 / 5.9 | 0.54 / 1.25 / 1.32 / 1.39 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 0.95 / 3.43 | 0.14 / 0.22 | 68 / 74 / 76 | 76 | 4.8 | 0/288 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-o | 2 | 694 | 151 | 30470 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-o | 2 | 1 / 9 / 52 | 0.3 / 41.7 | gvisor_sentry 0.05, kubelet 0.03, bash 0.03, ateworker-gviso 0.02, nydusd-amp 0.02, containerd 0.02 |
