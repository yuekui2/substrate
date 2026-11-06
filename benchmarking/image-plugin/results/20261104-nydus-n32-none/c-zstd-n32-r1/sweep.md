### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-oc | 2 | 96 | 480 | 0 | 2.2 / 3.3 / 3.5 / 3.6 | 0.15 / 0.19 / 0.26 / 0.28 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-oc | 2 | 5.2 / 6.4 / 6.6 / 6.7 | 0.51 / 1.27 / 1.35 / 1.44 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-oc | 2 | 0.95 / 3.80 | 0.14 / 0.21 | 69 / 76 / 80 | 80 | 6.0 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-oc | 2 | 718 | 184 | 28779 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-oc | 2 | 1 / 12 / 64 | 0.2 / 42.4 | gvisor_sentry 0.06, nydusd-amp 0.04, bash 0.03, kubelet 0.03, ateworker-gviso 0.03, containerd 0.02 |
