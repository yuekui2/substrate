# Riptide + Memorystore, gcfsd-v2 dev-e49ebb5 (fpt), Nov 4 2026

gcfsd-v2 from `gs://kuiyue-gke-dev-riptide/gcfsd-v2-ms-fpt`, packed as
`riptide-bins-ms@sha256:3ff5e4c2…` (`scope_plugin.py`), same flags as
before. Valkey `10.1.15.239:6379`, flushed before each point. Set A
(`-pf` templates; Riptide reads the OCI refs), 2 actors per worker,
workers balanced by `wait-ready.sh`, cold pool per point. The plugin still
ran the metadata walk. Driver: `driver/sweep.sh`. `n16-unbalanced` ran
before a `wait-ready.sh` fix and is not used.

First-turn TTFI p50 / p95 / p99 / p100 (s):

| Per node | dev-e49ebb5 (this run) | dev-c98764b (`20261103-confirm-sweep/set-a`) | Actors per node |
|---|---|---|---|
| 16 | 2.65 / 3.26 / 5.10 / 6.60 | 2.23 / 3.43 / 3.65 / 3.79 | 13 / 20 / 15 |
| 24 | 3.28 / 5.51 / 5.96 / 5.97 | 3.43 / 4.59 / 4.82 / 5.22 | 23 / 25 / 24 |
| 32 | 4.62 / 10.35 / 10.42 / 10.45 | 4.17 / 7.57 / 8.07 / 13.70 | 32 / 32 / 32 |

gcfsd-v2 uses 6.6–8.4 cores per node in the cold-start window, as before.
The fpt build is not faster on these single runs. First-task E2E p95 at 32
per node is 16.2 s, against 6.4 s for Nydus without the walk.
