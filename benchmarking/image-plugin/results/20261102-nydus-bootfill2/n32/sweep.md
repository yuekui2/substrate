### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 96 | 480 | 0 | 2.3 / 5.0 / 5.3 / 5.5 | 0.16 / 0.19 / 0.24 / 0.31 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pf | 2 | 6.3 / 7.2 / 7.3 / 7.5 | 0.50 / 1.28 / 1.34 / 1.44 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 0.94 / 4.37 | 0.14 / 0.21 | 70 / 76 / 77 | 77 | 6.2 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pf | 2 | 317 | 121 | 29409 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pf | 2 | 1 / 11 / 55 | 0.3 / 24.7 | nydus-dragonfly 0.10, nydusd-bootfill 0.04, kubelet 0.03, ateworker-gviso 0.03, containerd 0.02, fluent-bit 0.02 |
