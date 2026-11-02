### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 96 | 480 | 0 | 4.4 / 5.7 / 6.4 / 6.7 | 0.16 / 0.19 / 0.26 / 0.30 | 0.0 / 0.1 / 0.1 |
| nydus-pf | 2 | 96 | 480 | 0 | 4.3 / 5.6 / 5.8 / 6.0 | 0.16 / 0.20 / 0.25 / 0.27 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pf | 2 | 6.8 / 8.2 / 8.6 / 8.6 | 0.50 / 1.27 / 1.30 / 1.43 |
| nydus-pf | 2 | 6.9 / 8.7 / 9.0 / 9.0 | 0.51 / 1.30 / 1.36 / 1.41 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 0.93 / 3.20 | 0.14 / 0.21 | 71 / 77 / 80 | 80 | 6.0 | 0/384 (0%) | — |
| nydus-pf | 2 | 0.96 / 3.57 | 0.14 / 0.21 | 71 / 76 / 78 | 78 | 6.1 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pf | 2 | 335 | 93 | 24368 |
| nydus-pf | 2 | 266 | 93 | 27626 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pf | 2 | 1 / 11 / 60 | 0.3 / 28.8 | nydus-dragonfly 0.09, nydusd 0.06, ateworker-gviso 0.03, kubelet 0.03, containerd 0.02, fluent-bit 0.02 |
| nydus-pf | 2 | 1 / 11 / 55 | 0.3 / 29.9 | nydus-dragonfly 0.09, nydusd 0.06, kubelet 0.03, ateworker-gviso 0.03, gvisor_sentry 0.02, containerd 0.02 |
