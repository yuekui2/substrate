### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-ob | 2 | 96 | 360 | 0 | 1.8 / 2.4 / 2.5 / 2.6 | 0.16 / 0.18 / 0.25 / 0.32 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-ob | 2 | 4.7 / 5.2 / 5.3 / 5.3 | 1.10 / 1.23 / 1.33 / 1.49 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-ob | 2 | 0.98 / 3.19 | 0.13 / 0.21 | 68 / 74 / 77 | 77 | 4.7 | 0/288 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-ob | 2 | 523 | 141 | 26742 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-ob | 2 | 1 / 10 / 52 | 0.2 / 37.8 | gvisor_sentry 0.04, kubelet 0.03, bash 0.03, nydusd-amp 0.02, ateworker-gviso 0.02, containerd 0.02 |
