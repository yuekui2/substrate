### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-nonec | 2 | 96 | 480 | 0 | 2.2 / 2.8 / 2.9 / 3.1 | 0.15 / 0.19 / 0.26 / 0.28 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-nonec | 2 | 4.5 / 5.7 / 5.9 / 6.0 | 0.54 / 1.27 / 1.28 / 1.33 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-nonec | 2 | 0.95 / 3.17 | 0.13 / 0.21 | 69 / 75 / 77 | 77 | 6.2 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-nonec | 2 | 611 | 240 | 37865 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-nonec | 2 | 2 / 12 / 64 | 0.2 / 43.3 | gvisor_sentry 0.06, kubelet 0.03, bash 0.03, ateworker-gviso 0.03, nydusd-amp 0.02, containerd 0.02 |
