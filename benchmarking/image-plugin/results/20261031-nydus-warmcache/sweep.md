### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 8 | 384 | 320 | 0 | 14.7 / 29.9 / 32.1 / 34.0 | 0.17 / 0.22 / 0.25 / 0.29 | 0.0 / 0.0 / 0.1 |
| nydus | 8 | 384 | 320 | 0 | 4.8 / 19.0 / 21.0 / 21.6 | 0.17 / 0.23 / 0.28 / 0.36 | 0.0 / 0.0 / 0.1 |
| nydus | 8 | 384 | 320 | 0 | 1.5 / 6.2 / 7.3 / 7.4 | 0.17 / 0.24 / 0.27 / 0.31 | 0.0 / 0.0 / 0.0 |

### End to end per turn: resume request to work done (s)

| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |
|---|---|---|---|
| nydus | 8 | 20.7 / 33.7 / 36.0 / 37.8 | 3.51 / 5.24 / 5.35 / 5.47 |
| nydus | 8 | 11.0 / 21.5 / 24.1 / 25.1 | 3.43 / 5.26 / 5.43 / 5.56 |
| nydus | 8 | 6.5 / 12.6 / 13.4 / 13.6 | 3.51 / 5.36 / 5.50 / 5.61 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 8 | 4.50 / 6.50 | 0.50 / 0.66 | 97 / 113 / 117 | 117 | 2.7 | 0/256 (0%) | — |
| nydus | 8 | 4.30 / 6.74 | 0.50 / 0.66 | 90 / 103 / 109 | 109 | 2.9 | 0/256 (0%) | — |
| nydus | 8 | 4.20 / 6.62 | 0.52 / 0.67 | 86 / 99 / 100 | 100 | 3.2 | 0/256 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 8 | 125 | 53 | 25982 |
| nydus | 8 | 237 | 108 | 30706 |
| nydus | 8 | 167 | 53 | 16083 |
