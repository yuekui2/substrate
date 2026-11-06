### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-none | 2 | 96 | 480 | 0 | 2.2 / 3.2 / 3.4 / 3.4 | 0.16 / 0.19 / 0.26 / 0.30 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-none | 2 | 4.7 / 5.6 / 5.9 / 5.9 | 0.53 / 1.28 / 1.39 / 1.56 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-none | 2 | 0.94 / 3.14 | 0.14 / 0.21 | 68 / 74 / 76 | 76 | 6.3 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-none | 2 | 761 | 356 | 40749 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-none | 2 | 2 / 12 / 61 | 0.2 / 43.4 | gvisor_sentry 0.06, ateworker-gviso 0.03, kubelet 0.03, bash 0.03, nydusd-amp 0.02, containerd 0.02 |
