### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-nonec | 2 | 96 | 480 | 0 | 2.3 / 3.1 / 3.4 / 3.5 | 0.15 / 0.19 / 0.26 / 0.34 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-nonec | 2 | 4.8 / 6.0 / 6.3 / 6.3 | 0.51 / 1.26 / 1.37 / 1.52 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-nonec | 2 | 0.95 / 3.22 | 0.14 / 0.21 | 68 / 74 / 77 | 77 | 6.2 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-nonec | 2 | 967 | 319 | 39708 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-nonec | 2 | 2 / 12 / 62 | 0.2 / 44.3 | gvisor_sentry 0.06, kubelet 0.03, bash 0.03, ateworker-gviso 0.03, nydusd-amp 0.02, containerd 0.02 |
