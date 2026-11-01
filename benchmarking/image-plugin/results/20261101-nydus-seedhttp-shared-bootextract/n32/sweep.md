### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 96 | 480 | 0 | 2.7 / 2.8 / 3.5 / 3.5 | 0.16 / 0.19 / 0.27 / 0.29 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-shared | 2 | 4.9 / 6.9 / 7.0 / 7.1 | 0.24 / 0.30 / 0.40 / 0.45 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 0.09 / 3.33 | 0.12 / 0.22 | 67 / 74 / 77 | 77 | 6.2 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-shared | 2 | 275 | 147 | 44543 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-shared | 2 | 1 / 9 / 70 | 0.3 / 20.7 | nydus-dragonfly 0.26, nydusd 0.05, kubelet 0.03, ateworker-gviso 0.03, containerd 0.02, fluent-bit 0.02 |
