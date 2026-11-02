### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 96 | 480 | 0 | 2.2 / 2.4 / 2.4 / 2.5 | 0.15 / 0.19 / 0.26 / 0.29 | 0.0 / 0.0 / 0.1 |
| nydus-shared | 2 | 96 | 480 | 0 | 1.8 / 1.9 / 1.9 / 2.0 | 0.15 / 0.18 / 0.19 / 0.21 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-shared | 2 | 3.2 / 4.0 / 4.1 / 4.1 | 0.24 / 0.29 / 0.35 / 0.42 |
| nydus-shared | 2 | 2.7 / 3.3 / 3.4 / 3.4 | 0.23 / 0.28 / 0.30 / 0.34 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-shared | 2 | 0.09 / 1.28 | 0.13 / 0.27 | 65 / 70 / 72 | 72 | 6.6 | 0/384 (0%) | — |
| nydus-shared | 2 | 0.09 / 1.17 | 0.12 / 0.25 | 64 / 69 / 73 | 73 | 6.5 | 0/384 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-shared | 2 | 266 | 48 | 6745 |
| nydus-shared | 2 | 224 | 38 | 5298 |

### CPU per node (5 s samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-shared | 2 | 1 / 8 / 58 | 0.3 / 9.2 | nydus-dragonfly 0.22, kubelet 0.03, ateworker-gviso 0.03, containerd 0.02, fluent-bit 0.02, cilium-agent 0.02 |
| nydus-shared | 2 | 1 / 7 / 59 | 0.3 / 7.7 | nydus-dragonfly 0.24, kubelet 0.03, ateworker-gviso 0.03, containerd 0.02, fluent-bit 0.02, cilium-agent 0.02 |
