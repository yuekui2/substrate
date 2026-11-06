### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-oc | 2 | 96 | 240 | 0 | 1.4 / 1.7 / 1.8 / 1.8 | 0.15 / 0.17 / 0.18 / 0.19 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-oc | 2 | 3.1 / 4.5 / 4.6 / 4.7 | 0.46 / 1.16 / 1.21 / 1.26 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-oc | 2 | 0.85 / 2.82 | 0.12 / 0.21 | 66 / 73 / 75 | 75 | 3.2 | 0/192 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-oc | 2 | 411 | 125 | 19273 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-oc | 2 | 1 / 6 / 36 | 0.2 / 32.6 | kubelet 0.03, bash 0.02, gvisor_sentry 0.02, containerd 0.02, ateworker-gviso 0.02, containerd-shim 0.01 |
