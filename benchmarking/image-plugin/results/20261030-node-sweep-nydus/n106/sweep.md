### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 8 | 384 | 1590 | 5 | 43.4 / 87.6 / 94.5 / 97.1 | 0.23 / 0.33 / 0.39 / 1.15 | 0.0 / 0.1 / 0.2 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus | 8 | 56.5 / 91.8 / 97.3 / 100.6 | 2.38 / 6.60 / 6.91 / 7.16 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 8 | 5.00 / 17.88 | 0.58 / 0.76 | 139 / 167 / 244 | 244 | 6.5 | 0/1268 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 8 | 401 | 95 | 41439 |
