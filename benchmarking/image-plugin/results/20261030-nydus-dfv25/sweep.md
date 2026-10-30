### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 8 | 384 | 2400 | 10 | 61.0 / 127.1 / 138.1 / 147.0 | 0.33 / 0.98 / 69.75 / 113.68 | 0.0 / 15.1 / 73.9 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 8 | 5.79 / 33.93 | 0.67 / 0.95 | 181 / 213 / 298 | 298 | 8.0 | 37/1912 (2%) | 73.24 |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 8 | 1680 | 541 | 36419 |
