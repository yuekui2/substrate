### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 8 | 384 | 2400 | 847 | 155.8 / 286.4 / 336.3 / 370.1 | 0.50 / 79.04 / 167.68 / 288.57 | 0.0 / 67.8 / 173.9 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 8 | 6.88 / 43.70 | 0.79 / 1.25 | 385 / 487 / 524 | 524 | 3.0 | 52/1314 (4%) | 102.77 |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 8 | 1847 | 284 | 36156 |
