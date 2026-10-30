### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| pull | 10 | 480 | 75 | 0 | 34.6 / 46.2 / 46.2 / 46.2 | 0.20 / 0.26 / 0.28 / 0.32 | 0.0 / 0.0 / 0.0 |
| riptide | 10 | 480 | 75 | 0 | 2.1 / 2.5 / 2.6 / 2.6 | 0.21 / 0.29 / 0.33 / 0.33 | 0.0 / 0.0 / 0.0 |
| spegel | 10 | 480 | 75 | 0 | 18.3 / 24.7 / 25.6 / 25.8 | 0.20 / 0.26 / 0.27 / 0.27 | 0.0 / 0.0 / 0.0 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| pull | 10 | 6.37 / 7.35 | 0.64 / 0.79 | 119 / 139 / 141 | 141 | 0.5 | 0/60 (0%) | — |
| riptide | 10 | 9.00 / 13.42 | 0.80 / 0.90 | 116 / 124 / 128 | 128 | 0.6 | 0/60 (0%) | — |
| spegel | 10 | 6.19 / 6.90 | 0.73 / 0.88 | 111 / 118 / 118 | 118 | 0.6 | 0/60 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| pull | 10 | 1156 | 2 | 6426 |
| riptide | 10 | 956 | 0 | 5760 |
| spegel | 10 | 1578 | 0 | 10338 |
