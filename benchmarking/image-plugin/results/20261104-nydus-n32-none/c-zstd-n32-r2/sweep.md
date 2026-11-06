### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-oc | 2 | 96 | 480 | 0 | 2.3 / 3.3 / 3.5 / 3.9 | 0.15 / 0.19 / 0.27 / 0.33 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-oc | 2 | 5.3 / 6.2 / 6.5 / 6.5 | 0.52 / 1.27 / 1.33 / 1.36 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-oc | 2 | 0.95 / 3.56 | 0.14 / 0.21 | 68 / 74 / 78 | 78 | 6.2 | 0/384 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-oc | 2 | 770 | 168 | 29307 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-oc | 2 | 1 / 10 / 60 | 0.2 / 38.2 | gvisor_sentry 0.06, kubelet 0.03, nydusd-amp 0.03, bash 0.03, ateworker-gviso 0.03, containerd 0.02 |
