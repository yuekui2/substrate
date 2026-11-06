### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 96 | 240 | 0 | 1.4 / 2.1 / 2.2 / 2.3 | 0.15 / 0.18 / 0.18 / 0.18 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-o | 2 | 3.5 / 4.6 / 4.7 / 4.8 | 0.51 / 1.20 / 1.26 / 1.29 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-o | 2 | 0.93 / 2.82 | 0.13 / 0.20 | 67 / 74 / 75 | 75 | 3.2 | 0/192 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-o | 2 | 753 | 111 | 22387 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-o | 2 | 1 / 6 / 39 | 0.2 / 30.9 | kubelet 0.03, gvisor_sentry 0.02, bash 0.02, containerd 0.02, ateworker-gviso 0.02, cilium-agent 0.01 |
