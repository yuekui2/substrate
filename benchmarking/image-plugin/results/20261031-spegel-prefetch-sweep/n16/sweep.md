### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| spegel | 8 | 384 | 240 | 0 | 20.6 / 47.6 / 55.4 / 56.0 | 0.18 / 0.25 / 0.27 / 0.28 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| spegel | 8 | 23.0 / 49.3 / 55.6 / 56.2 | 2.04 / 5.47 / 5.63 / 5.81 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| spegel | 8 | 1.90 / 5.57 | 0.58 / 0.70 | 103 / 119 / 128 | 128 | 1.9 | 0/192 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| spegel | 8 | 107 | 0 | 483 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| spegel | 8 | 1 / 5 / 7 | 0.1 / 0.5 | kubelet 0.03, gvisor_sentry 0.03, ateworker-gviso 0.02, cilium-agent 0.02, containerd 0.01, containerd-shim 0.01 |
