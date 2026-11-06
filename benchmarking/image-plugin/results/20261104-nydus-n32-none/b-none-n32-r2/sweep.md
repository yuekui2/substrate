### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-noneb | 2 | 96 | 480 | 0 | 2.2 / 3.1 / 3.4 / 3.7 | 0.16 / 0.18 / 0.27 / 0.33 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-noneb | 2 | 4.9 / 5.6 / 5.9 / 6.2 | 1.10 / 1.28 / 1.33 / 1.38 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-noneb | 2 | 0.99 / 3.06 | 0.14 / 0.21 | 69 / 76 / 78 | 78 | 6.1 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-noneb | 2 | 841 | 352 | 45213 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-noneb | 2 | 1 / 12 / 65 | 0.2 / 40.5 | gvisor_sentry 0.07, ateworker-gviso 0.03, kubelet 0.03, bash 0.03, nydusd-amp 0.02, containerd 0.02 |
