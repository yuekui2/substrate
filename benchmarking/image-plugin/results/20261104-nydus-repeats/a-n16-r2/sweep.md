### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 96 | 240 | 0 | 1.4 / 1.6 / 1.7 / 1.7 | 0.15 / 0.18 / 0.21 / 0.30 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-o | 2 | 3.1 / 4.3 / 4.5 / 4.5 | 0.53 / 1.21 / 1.26 / 1.35 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 0.94 / 2.79 | 0.13 / 0.19 | 67 / 73 / 74 | 74 | 3.3 | 0/192 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-o | 2 | 611 | 111 | 20286 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-o | 2 | 1 / 7 / 41 | 0.2 / 31.7 | kubelet 0.03, gvisor_sentry 0.03, bash 0.02, containerd 0.02, ateworker-gviso 0.02, containerd-shim 0.01 |
