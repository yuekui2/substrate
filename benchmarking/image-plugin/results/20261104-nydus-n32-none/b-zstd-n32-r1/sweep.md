### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-ob | 2 | 96 | 480 | 0 | 2.4 / 3.7 / 4.3 / 4.7 | 0.16 / 0.19 / 0.28 / 0.31 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-ob | 2 | 6.3 / 6.9 / 7.3 / 7.6 | 1.11 / 1.27 / 1.39 / 1.48 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-ob | 2 | 0.99 / 4.10 | 0.14 / 0.22 | 70 / 77 / 80 | 80 | 6.0 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-ob | 2 | 708 | 162 | 31073 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-ob | 2 | 1 / 11 / 63 | 0.2 / 42.3 | gvisor_sentry 0.06, nydusd-amp 0.03, kubelet 0.03, bash 0.03, ateworker-gviso 0.03, containerd 0.02 |
