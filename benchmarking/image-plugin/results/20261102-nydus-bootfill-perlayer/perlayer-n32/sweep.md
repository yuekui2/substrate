### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 96 | 480 | 0 | 3.6 / 5.0 / 5.3 / 5.5 | 0.16 / 0.19 / 0.25 / 0.32 | 0.0 / 0.1 / 0.1 |
| nydus-pf | 2 | 96 | 480 | 0 | 3.7 / 5.4 / 5.7 / 5.8 | 0.16 / 0.19 / 0.26 / 0.29 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pf | 2 | 6.2 / 7.8 / 8.0 / 8.0 | 0.49 / 1.25 / 1.31 / 1.41 |
| nydus-pf | 2 | 6.7 / 8.0 / 8.5 / 8.9 | 0.50 / 1.28 / 1.34 / 1.43 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 0.94 / 3.17 | 0.13 / 0.21 | 69 / 77 / 81 | 81 | 5.9 | 0/384 (0%) | — |
| nydus-pf | 2 | 0.94 / 3.70 | 0.14 / 0.21 | 70 / 77 / 79 | 79 | 6.0 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pf | 2 | 258 | 79 | 19471 |
| nydus-pf | 2 | 275 | 88 | 20457 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pf | 2 | 1 / 10 / 41 | 0.3 / 26.6 | nydus-dragonfly 0.09, nydusd-bootfill 0.04, kubelet 0.03, ateworker-gviso 0.03, containerd 0.02, fluent-bit 0.02 |
| nydus-pf | 2 | 1 / 10 / 60 | 0.3 / 27.2 | nydus-dragonfly 0.09, nydusd-bootfill 0.05, kubelet 0.03, ateworker-gviso 0.03, containerd 0.02, fluent-bit 0.02 |
