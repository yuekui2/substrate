### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 96 | 480 | 0 | 2.2 / 3.4 / 3.9 / 4.1 | 0.15 / 0.18 / 0.24 / 0.29 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-o | 2 | 5.4 / 6.4 / 6.6 / 6.6 | 0.49 / 1.26 / 1.31 / 1.49 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 0.94 / 3.70 | 0.14 / 0.21 | 69 / 75 / 77 | 77 | 6.3 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-o | 2 | 991 | 177 | 29427 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-o | 2 | 2 / 12 / 59 | 0.2 / 41.8 | gvisor_sentry 0.06, nydusd-amp 0.04, kubelet 0.03, bash 0.03, ateworker-gviso 0.03, containerd 0.02 |
