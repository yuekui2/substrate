### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 8 | 384 | 960 | 0 | 9.3 / 20.4 / 21.5 / 26.1 | 0.19 / 0.27 / 0.33 / 0.40 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus | 8 | 16.5 / 24.0 / 27.0 / 32.0 | 2.32 / 5.93 / 6.32 / 6.84 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 8 | 4.71 / 9.99 | 0.55 / 0.70 | 95 / 109 / 117 | 117 | 8.2 | 0/768 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 8 | 56 | 71 | 764 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus | 8 | 1 / 5 / 18 | 0.3 / 3.1 | kubelet 0.03, containerd 0.02, gvisor_sentry 0.01, cilium-agent 0.01, containerd-shim 0.01, ateworker-gviso 0.01 |
