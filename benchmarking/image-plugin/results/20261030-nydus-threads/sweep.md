### TTFI including queue wait (s)

| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |
|---|---|---|---|---|---|---|---|
| nydus | 4 | 192 | 2400 | 0 | 138.5 / 364.9 / 499.6 / 565.2 | 32.78 / 254.45 / 341.17 / 483.14 | 26.0 / 180.1 / 410.8 |
| nydus | 8 | 384 | 2400 | 974 | 171.5 / 268.2 / 340.5 / 359.6 | 0.53 / 79.16 / 177.06 / 274.68 | 0.0 / 72.3 / 184.4 |

### Completion (s)

| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |
|---|---|---|---|---|---|---|---|---|
| nydus | 4 | 4.46 / 35.50 | 0.48 / 0.80 | 544 / 682 / 740 | 740 | 3.2 | 866/1920 (45%) | 98.63 |
| nydus | 8 | 5.95 / 42.62 | 0.75 / 1.17 | 380 / 506 / 532 | 532 | 2.7 | 60/1210 (5%) | 77.89 |

### Disk peak per node (5 s samples)

| arm | actors/worker | write MB/s | read MB/s | IOPS |
|---|---|---|---|---|
| nydus | 4 | 1883 | 295 | 18984 |
| nydus | 8 | 1866 | 326 | 32904 |
