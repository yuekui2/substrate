### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| pull | 10 | 480 | 150 | 0 | 43.8 / 78.4 / 86.0 / 86.7 | 0.19 / 0.26 / 0.28 / 0.31 | 0.0 / 0.0 / 0.0 |
| riptide | 10 | 480 | 150 | 0 | 2.6 / 3.7 / 4.0 / 4.0 | 0.22 / 0.31 / 0.33 / 0.34 | 0.0 / 0.0 / 0.0 |
| spegel | 10 | 480 | 150 | 0 | 30.7 / 52.0 / 54.3 / 54.6 | 0.20 / 0.27 / 0.29 / 0.30 | 0.0 / 0.0 / 0.1 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| pull | 10 | 2.52 / 7.49 | 0.65 / 0.82 | 135 / 151 / 164 | 164 | 0.9 | 0/120 (0%) | — |
| riptide | 10 | 7.07 / 14.86 | 0.84 / 1.00 | 94 / 128 / 133 | 133 | 1.1 | 0/120 (0%) | — |
| spegel | 10 | 2.49 / 7.42 | 0.71 / 0.89 | 118 / 131 / 132 | 132 | 1.1 | 0/120 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| pull | 10 | 1379 | 31 | 8481 |
| riptide | 10 | 1258 | 0 | 6705 |
| spegel | 10 | 1422 | 31 | 10873 |
