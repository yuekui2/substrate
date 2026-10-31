### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| spegel | 8 | 384 | 360 | 0 | 26.8 / 57.2 / 61.3 / 63.2 | 0.18 / 0.25 / 0.27 / 0.46 | 0.0 / 0.0 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| spegel | 8 | 31.7 / 58.8 / 62.5 / 63.5 | 2.10 / 5.53 / 5.73 / 5.90 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| spegel | 8 | 2.00 / 5.75 | 0.58 / 0.71 | 112 / 127 / 133 | 133 | 2.7 | 0/288 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| spegel | 8 | 1873 | 2 | 17824 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| spegel | 8 | 1 / 43 / 77 | 0.3 / 44.6 | spegel-preload 0.94, kworker 0.05, gvisor_sentry 0.03, kubelet 0.03, ateworker-gviso 0.02, cilium-agent 0.02 |
