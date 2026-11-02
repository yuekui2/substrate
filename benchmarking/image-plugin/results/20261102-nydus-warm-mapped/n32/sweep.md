### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 96 | 480 | 0 | 2.5 / 5.5 / 5.8 / 6.1 | 0.17 / 0.21 / 0.29 / 0.33 | 0.0 / 0.1 / 0.1 |
| nydus-pf | 2 | 96 | 480 | 0 | 2.5 / 5.3 / 5.8 / 7.0 | 0.17 / 0.19 / 0.24 / 0.28 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pf | 2 | 6.4 / 7.6 / 7.7 / 7.7 | 0.52 / 1.29 / 1.36 / 1.41 |
| nydus-pf | 2 | 6.3 / 7.4 / 8.3 / 8.3 | 0.51 / 1.28 / 1.34 / 1.37 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 0.97 / 4.40 | 0.15 / 0.22 | 70 / 77 / 80 | 80 | 6.0 | 0/384 (0%) | — |
| nydus-pf | 2 | 0.95 / 4.33 | 0.14 / 0.22 | 70 / 77 / 78 | 78 | 6.1 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pf | 2 | 409 | 227 | 29869 |
| nydus-pf | 2 | 531 | 163 | 24981 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pf | 2 | 1 / 11 / 66 | 0.3 / 27.7 | nydus-dragonfly 0.11, nydusd-bootfill 0.05, ateworker-gviso 0.03, kubelet 0.03, containerd 0.02, gvisor_sentry 0.02 |
| nydus-pf | 2 | 1 / 11 / 53 | 0.3 / 23.9 | nydus-dragonfly 0.10, nydusd-bootfill 0.04, kubelet 0.03, ateworker-gviso 0.03, containerd 0.02, fluent-bit 0.02 |
