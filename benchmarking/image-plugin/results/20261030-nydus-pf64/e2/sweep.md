### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus-pf2 | 8 | 384 | 320 | 0 | 14.4 / 24.9 / 28.6 / 29.1 | 0.17 / 0.23 / 0.25 / 0.26 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus-pf2 | 8 | 19.7 / 29.2 / 31.9 / 32.3 | 3.47 / 5.38 / 5.60 / 5.74 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus-pf2 | 8 | 4.46 / 6.85 | 0.50 / 0.65 | 96 / 108 / 116 | 116 | 2.8 | 0/256 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus-pf2 | 8 | 276 | 40 | 18493 |
