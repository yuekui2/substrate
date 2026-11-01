### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 96 | 480 | 0 | 2.6 / 2.8 / 2.8 / 2.8 | 0.15 / 0.17 / 0.18 / 0.23 | 0.0 / 0.0 / 0.1 |
| nydus-shared | 2 | 96 | 480 | 0 | 1.6 / 1.8 / 1.9 / 1.9 | 0.15 / 0.17 / 0.25 / 0.27 | 0.0 / 0.1 / 0.1 |
| nydus-shared | 2 | 96 | 480 | 0 | 0.6 / 1.6 / 1.7 / 1.7 | 0.15 / 0.17 / 0.20 / 0.25 | 0.0 / 0.0 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-shared | 2 | 3.9 / 5.1 / 5.3 / 5.4 | 0.23 / 0.27 / 0.29 / 0.31 |
| nydus-shared | 2 | 2.5 / 2.7 / 2.9 / 3.0 | 0.23 / 0.28 / 0.35 / 0.37 |
| nydus-shared | 2 | 0.8 / 2.2 / 2.2 / 2.2 | 0.23 / 0.28 / 0.29 / 0.35 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 0.09 / 2.09 | 0.12 / 0.25 | 66 / 72 / 73 | 73 | 6.6 | 0/384 (0%) | — |
| nydus-shared | 2 | 0.09 / 0.85 | 0.12 / 0.21 | 63 / 70 / 73 | 73 | 6.5 | 0/384 (0%) | — |
| nydus-shared | 2 | 0.09 / 0.50 | 0.13 / 0.28 | 62 / 69 / 70 | 70 | 6.8 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-shared | 2 | 262 | 123 | 40911 |
| nydus-shared | 2 | 188 | 107 | 33441 |
| nydus-shared | 2 | 162 | 108 | 21524 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-shared | 2 | 1 / 7 / 62 | 0.3 / 25.4 | nydus-dragonfly 0.26, nydusd 0.05, kubelet 0.03, ateworker-gviso 0.03, containerd 0.02, fluent-bit 0.02 |
| nydus-shared | 2 | 1 / 7 / 43 | 0.3 / 32.1 | nydus-dragonfly 0.16, kubelet 0.03, nydusd 0.03, ateworker-gviso 0.03, containerd 0.02, fluent-bit 0.02 |
| nydus-shared | 2 | 1 / 6 / 37 | 0.3 / 25.5 | nydus-dragonfly 0.08, kubelet 0.03, ateworker-gviso 0.03, containerd 0.02, fluent-bit 0.02, containerd-shim 0.02 |
