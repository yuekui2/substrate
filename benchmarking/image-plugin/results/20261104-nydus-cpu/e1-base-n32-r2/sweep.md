### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 96 | 480 | 0 | 2.4 / 4.7 / 5.2 / 5.7 | 0.16 / 0.20 / 0.28 / 0.31 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-o | 2 | 6.1 / 7.2 / 7.5 / 7.6 | 0.52 / 1.29 / 1.35 / 1.41 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 0.96 / 3.86 | 0.15 / 0.21 | 69 / 76 / 78 | 78 | 6.2 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-o | 2 | 964 | 190 | 36439 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-o | 2 | 1 / 12 / 82 | 0.2 / 40.1 | nydus-dragonfly 0.10, gvisor_sentry 0.06, nydusd-amp 0.04, ateworker-gviso 0.03, kubelet 0.03, bash 0.03 |
