### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 96 | 240 | 0 | 1.3 / 1.7 / 1.9 / 1.9 | 0.14 / 0.16 / 0.17 / 0.19 | 0.0 / 0.0 / 0.1 |
| pull-pf | 2 | 96 | 240 | 0 | 45.4 / 79.0 / 85.4 / 89.0 | 0.14 / 0.17 / 0.19 / 0.24 | 0.0 / 0.0 / 0.1 |
| riptide-ms-pf | 2 | 96 | 240 | 0 | 2.2 / 3.4 / 3.7 / 3.8 | 0.15 / 0.18 / 0.22 / 0.25 | 0.0 / 0.1 / 0.1 |
| spegel-pf | 2 | 96 | 240 | 0 | 19.4 / 42.1 / 46.7 / 46.8 | 0.15 / 0.18 / 0.20 / 0.26 | 0.0 / 0.1 / 0.1 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pf | 2 | 3.5 / 4.5 / 4.7 / 4.8 | 0.53 / 1.17 / 1.23 / 1.23 |
| pull-pf | 2 | 46.6 / 79.3 / 85.6 / 89.3 | 0.51 / 1.28 / 1.34 / 1.39 |
| riptide-ms-pf | 2 | 7.6 / 10.1 / 10.3 / 10.3 | 0.85 / 2.00 / 2.03 / 2.09 |
| spegel-pf | 2 | 20.4 / 42.3 / 46.9 / 46.9 | 0.54 / 1.24 / 1.27 / 1.29 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pf | 2 | 0.93 / 2.88 | 0.12 / 0.19 | 67 / 73 / 77 | 77 | 3.1 | 0/192 (0%) | — |
| pull-pf | 2 | 0.39 / 1.19 | 0.13 / 0.21 | 111 / 140 / 157 | 157 | 1.5 | 0/192 (0%) | — |
| riptide-ms-pf | 2 | 1.65 / 6.27 | 0.19 / 0.37 | 73 / 81 / 87 | 87 | 2.8 | 0/192 (0%) | — |
| spegel-pf | 2 | 0.41 / 1.20 | 0.14 / 0.21 | 85 / 104 / 111 | 111 | 2.2 | 0/192 (0%) | — |

### Disk peak per node (node-sampler samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pf | 2 | 491 | 187 | 28871 |
| pull-pf | 2 | 1883 | 2 | 13676 |
| riptide-ms-pf | 2 | 524 | 1 | 4467 |
| spegel-pf | 2 | 1944 | 862 | 19777 |

### CPU per node (node-sampler samples, % of all cores)

| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |
|---|---|---|---|---|
| nydus-pf | 2 | 1 / 7 / 70 | 0.2 / 32.9 | nydus-dragonfly 0.05, kubelet 0.03, gvisor_sentry 0.03, bash 0.02, containerd 0.02, ateworker-gviso 0.02 |
| pull-pf | 2 | 1 / 29 / 48 | 0.1 / 23.1 | atelet 1.23, kworker 0.04, kubelet 0.03, gvisor_sentry 0.02, ateworker-gviso 0.02, bash 0.02 |
| riptide-ms-pf | 2 | 1 / 19 / 83 | 0.1 / 9.3 | gcfsd-v2 0.56, riptide 0.13, gvisor_sentry 0.11, kubelet 0.04, ateworker-gviso 0.03, bash 0.02 |
| spegel-pf | 2 | 1 / 33 / 76 | 0.2 / 79.3 | spegel-preload 0.65, kworker 0.03, kubelet 0.03, bash 0.02, gvisor_sentry 0.02, cilium-agent 0.02 |
