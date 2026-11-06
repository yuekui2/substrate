### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 96 | 360 | 0 | 1.9 / 2.7 / 2.9 / 2.9 | 0.15 / 0.18 / 0.26 / 0.27 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-o | 2 | 4.3 / 5.5 / 5.7 / 5.7 | 0.55 / 1.23 / 1.28 / 1.42 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 0.95 / 3.21 | 0.14 / 0.21 | 67 / 74 / 77 | 77 | 4.7 | 0/288 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-o | 2 | 645 | 164 | 26796 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-o | 2 | 1 / 8 / 51 | 0.2 / 36.6 | gvisor_sentry 0.05, kubelet 0.03, bash 0.03, ateworker-gviso 0.02, nydusd-amp 0.02, containerd 0.02 |
