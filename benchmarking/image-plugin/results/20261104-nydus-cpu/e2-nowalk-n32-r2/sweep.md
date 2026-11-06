### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 96 | 480 | 0 | 2.2 / 3.6 / 4.2 / 4.3 | 0.15 / 0.18 / 0.23 / 0.25 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-o | 2 | 5.1 / 6.2 / 6.5 / 7.1 | 0.49 / 1.27 / 1.35 / 1.39 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 0.93 / 3.62 | 0.14 / 0.21 | 68 / 77 / 81 | 81 | 6.0 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-o | 2 | 698 | 158 | 30750 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-o | 2 | 1 / 9 / 65 | 0.2 / 39.3 | gvisor_sentry 0.04, bash 0.03, kubelet 0.03, nydusd-amp 0.03, ateworker-gviso 0.02, containerd 0.02 |
