### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 8 | 384 | 320 | 0 | 15.9 / 29.1 / 30.6 / 31.5 | 0.17 / 0.24 / 0.39 / 0.68 | 0.0 / 0.1 / 0.1 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 8 | 4.28 / 6.54 | 0.50 / 0.67 | 96 / 112 / 121 | 121 | 2.7 | 0/256 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 8 | 131 | 53 | 25671 |
