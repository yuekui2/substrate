### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 96 | 240 | 0 | 1.6 / 1.9 / 2.1 / 2.1 | 0.15 / 0.18 / 0.25 / 0.26 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-o | 2 | 4.4 / 5.6 / 6.3 / 6.5 | 0.53 / 1.21 / 1.29 / 1.38 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 0.93 / 3.24 | 0.12 / 0.21 | 68 / 76 / 77 | 77 | 3.1 | 0/192 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-o | 2 | 473 | 146 | 20007 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-o | 2 | 1 / 6 / 42 | 0.2 / 35.0 | kubelet 0.03, gvisor_sentry 0.03, bash 0.02, containerd 0.02, ateworker-gviso 0.02, containerd-shim 0.01 |
