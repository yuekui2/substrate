### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 96 | 240 | 0 | 1.7 / 2.6 / 2.8 / 2.9 | 0.14 / 0.17 / 0.19 / 0.26 | 0.0 / 0.0 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pf | 2 | 4.5 / 5.2 / 5.4 / 5.5 | 0.52 / 1.22 / 1.28 / 1.41 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 0.94 / 3.20 | 0.13 / 0.20 | 68 / 76 / 78 | 78 | 3.1 | 0/192 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pf | 2 | 565 | 175 | 32946 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pf | 2 | 1 / 6 / 76 | 0.2 / 36.9 | nydus-dragonfly 0.05, gvisor_sentry 0.03, kubelet 0.03, bash 0.02, containerd 0.02, nydusd-amp 0.02 |
