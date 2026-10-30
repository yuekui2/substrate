### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| pull | 10 | 480 | 300 | 0 | 67.2 / 85.9 / 102.6 / 109.3 | 0.20 / 0.25 / 0.27 / 0.30 | 0.0 / 0.0 / 0.0 |
| riptide | 10 | 480 | 300 | 0 | 4.2 / 6.0 / 6.3 / 6.3 | 0.23 / 0.34 / 0.42 / 0.50 | 0.0 / 0.0 / 0.1 |
| spegel | 10 | 480 | 300 | 0 | 36.7 / 62.6 / 70.9 / 73.0 | 0.20 / 0.26 / 0.29 / 0.35 | 0.0 / 0.0 / 0.1 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| pull | 10 | 4.65 / 7.94 | 0.65 / 0.83 | 157 / 176 / 185 | 185 | 1.6 | 0/240 (0%) | — |
| riptide | 10 | 9.03 / 18.94 | 0.89 / 1.11 | 106 / 139 / 142 | 142 | 2.1 | 0/240 (0%) | — |
| spegel | 10 | 4.54 / 7.90 | 0.74 / 0.89 | 128 / 142 / 144 | 144 | 2.1 | 0/240 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| pull | 10 | 1529 | 10 | 9647 |
| riptide | 10 | 1312 | 10 | 8302 |
| spegel | 10 | 1625 | 7 | 14327 |
