### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| pull | 10 | 480 | 1200 | 0 | 134.5 / 232.8 / 263.4 / 266.0 | 0.35 / 0.53 / 0.64 / 0.81 | 0.0 / 0.1 / 0.1 |
| riptide | 10 | 480 | 1200 | 0 | 25.0 / 37.3 / 37.4 / 37.6 | 0.35 / 0.46 / 0.53 / 0.60 | 0.0 / 0.2 / 0.2 |
| spegel | 10 | 480 | 1200 | 0 | 105.9 / 159.9 / 173.3 / 177.1 | 0.33 / 0.57 / 0.74 / 1.23 | 0.0 / 0.1 / 0.1 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| pull | 10 | 3.71 / 10.97 | 0.86 / 1.17 | 237 / 315 / 349 | 349 | 3.4 | 0/960 (0%) | — |
| riptide | 10 | 11.59 / 26.41 | 1.06 / 1.41 | 134 / 188 / 193 | 193 | 6.2 | 0/960 (0%) | — |
| spegel | 10 | 5.75 / 11.70 | 0.94 / 1.32 | 210 / 237 / 258 | 258 | 4.7 | 0/960 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| pull | 10 | 1527 | 76 | 16917 |
| riptide | 10 | 1408 | 0 | 7900 |
| spegel | 10 | 1649 | 209 | 21046 |
