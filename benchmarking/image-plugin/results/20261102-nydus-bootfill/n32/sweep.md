### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 96 | 480 | 5 | 2.3 / 6.0 / 6.3 / 6.3 | 0.16 / 0.18 / 0.22 / 0.32 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pf | 2 | 6.6 / 7.8 / 8.2 / 8.5 | 0.49 / 1.26 / 1.30 / 1.35 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 0.94 / 4.49 | 0.15 / 0.23 | 70 / 77 / 83 | 83 | 5.8 | 0/380 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pf | 2 | 332 | 121 | 39716 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pf | 2 | 1 / 12 / 65 | 0.3 / 22.0 | nydus-dragonfly 0.12, nydusd-bootfill 0.05, ateworker-gviso 0.03, kubelet 0.03, containerd 0.02, fluent-bit 0.02 |
