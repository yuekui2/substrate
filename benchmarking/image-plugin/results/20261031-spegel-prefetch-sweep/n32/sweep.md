### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| spegel | 8 | 384 | 480 | 0 | 35.9 / 64.8 / 72.0 / 72.9 | 0.18 / 0.26 / 0.31 / 0.53 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| spegel | 8 | 40.7 / 66.2 / 73.9 / 74.8 | 2.02 / 5.77 / 6.10 / 6.28 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| spegel | 8 | 1.88 / 5.90 | 0.59 / 0.74 | 120 / 134 / 147 | 147 | 3.3 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| spegel | 8 | 1863 | 3 | 17324 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| spegel | 8 | 1 / 51 / 73 | 0.3 / 46.2 | spegel-preload 1.23, kworker 0.06, gvisor_sentry 0.03, kubelet 0.03, ateworker-gviso 0.02, cilium-agent 0.02 |
