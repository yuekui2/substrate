### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 96 | 480 | 0 | 2.3 / 3.4 / 3.8 / 4.0 | 0.16 / 0.18 / 0.22 / 0.26 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-o | 2 | 5.3 / 6.1 / 6.3 / 6.5 | 0.49 / 1.26 / 1.32 / 1.44 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 0.95 / 3.67 | 0.13 / 0.21 | 69 / 76 / 79 | 79 | 6.1 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-o | 2 | 697 | 172 | 30189 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-o | 2 | 2 / 12 / 55 | 0.2 / 42.3 | gvisor_sentry 0.06, nydusd-amp 0.04, kubelet 0.03, ateworker-gviso 0.03, bash 0.03, containerd 0.02 |
