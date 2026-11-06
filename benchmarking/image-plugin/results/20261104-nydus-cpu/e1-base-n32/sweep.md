### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 96 | 480 | 0 | 2.3 / 5.0 / 5.8 / 9.3 | 0.16 / 0.19 / 0.28 / 0.31 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-o | 2 | 6.5 / 7.1 / 7.6 / 10.6 | 0.50 / 1.24 / 1.30 / 1.38 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 0.94 / 4.37 | 0.14 / 0.20 | 69 / 77 / 79 | 79 | 6.1 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-o | 2 | 774 | 167 | 34961 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-o | 2 | 1 / 10 / 81 | 0.2 / 41.6 | nydus-dragonfly 0.09, gvisor_sentry 0.05, nydusd-amp 0.04, kubelet 0.03, bash 0.03, ateworker-gviso 0.03 |
