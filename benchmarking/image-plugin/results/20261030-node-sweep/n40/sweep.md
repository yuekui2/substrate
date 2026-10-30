### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| pull | 10 | 480 | 600 | 0 | 100.4 / 167.9 / 188.4 / 188.9 | 0.24 / 0.38 / 0.46 / 0.57 | 0.0 / 0.1 / 0.1 |
| riptide | 10 | 480 | 600 | 0 | 7.3 / 9.3 / 9.5 / 9.7 | 0.28 / 0.40 / 0.44 / 0.50 | 0.0 / 0.1 / 0.1 |
| spegel | 10 | 480 | 600 | 0 | 61.7 / 104.5 / 116.6 / 120.9 | 0.25 / 0.43 / 0.54 / 0.60 | 0.0 / 0.1 / 0.1 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| pull | 10 | 2.65 / 8.77 | 0.73 / 0.93 | 196 / 240 / 267 | 267 | 2.2 | 0/480 (0%) | — |
| riptide | 10 | 10.00 / 21.42 | 0.96 / 1.27 | 102 / 154 / 158 | 158 | 3.8 | 0/480 (0%) | — |
| spegel | 10 | 3.26 / 9.41 | 0.84 / 1.08 | 147 / 181 / 193 | 193 | 3.1 | 0/480 (0%) | — |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| pull | 10 | 1306 | 36 | 14378 |
| riptide | 10 | 1311 | 0 | 8809 |
| spegel | 10 | 1740 | 62 | 16995 |
