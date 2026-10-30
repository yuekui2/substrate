### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| pull | 10 | 480 | 1590 | 0 | 161.8 / 297.6 / 304.1 / 304.2 | 0.40 / 0.61 / 0.71 / 0.86 | 0.0 / 0.1 / 0.2 |
| riptide | 10 | 480 | 1590 | 0 | 43.1 / 47.1 / 47.2 / 177.8 | 0.37 / 0.52 / 0.64 / 0.77 | 0.0 / 0.1 / 0.1 |
| spegel | 10 | 480 | 1590 | 0 | 92.3 / 169.0 / 207.8 / 213.0 | 0.43 / 0.69 / 0.84 / 1.56 | 0.0 / 0.2 / 0.3 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| pull | 10 | 5.40 / 12.28 | 0.94 / 1.24 | 277 / 365 / 383 | 383 | 4.1 | 0/1272 (0%) | — |
| riptide | 10 | 12.36 / 25.67 | 1.13 / 1.64 | 152 / 211 / 263 | 263 | 6.0 | 0/1272 (0%) | — |
| spegel | 10 | 6.31 / 12.88 | 1.12 / 1.46 | 203 / 245 / 290 | 290 | 5.5 | 0/1272 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| pull | 10 | 1570 | 160 | 22594 |
| riptide | 10 | 1378 | 11 | 9305 |
| spegel | 10 | 1799 | 396 | 26801 |
