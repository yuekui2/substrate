### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pf2 | 8 | 384 | 320 | 0 | 18.9 / 59.5 / 66.6 / 67.7 | 0.22 / 0.32 / 0.38 / 0.39 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pf2 | 8 | 30.1 / 66.6 / 69.7 / 72.0 | 3.96 / 7.95 / 8.30 / 8.34 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pf2 | 8 | 4.80 / 14.63 | 0.62 / 0.78 | 115 / 140 / 140 | 140 | 2.3 | 0/256 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pf2 | 8 | 1616 | 36 | 19024 |
