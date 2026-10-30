### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 8 | 384 | 2400 | 0 | 65.1 / 124.4 / 133.9 / 142.1 | 0.27 / 0.70 / 44.23 / 95.00 | 0.0 / 13.9 / 75.8 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 8 | 5.16 / 32.34 | 0.62 / 0.85 | 182 / 208 / 232 | 232 | 10.3 | 31/1920 (2%) | 60.29 |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 8 | 473 | 100 | 36980 |
