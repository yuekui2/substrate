# Nydus seedhttp with the bootstrap from the seeds

The plugin reads bootstrap layers from the seeds. Manifest reads fell back
to the registry: the seeds returned 503 because seed-preload.sh copied the
manifests with mode 700 and nginx could not read them (fixed in
seed-preload.sh; see `../20261031-nydus-seedhttp-boot2`). Old images, cold
bench pool, 8 actors per worker, first turn, seconds, p50/p95/p100.

| Run | TTFI | E2E | seedhttp (`../20261031-nydus-seedhttp`) |
|---|---|---|---|
| 1 actor | 1.9 | 3.6 | TTFI 2.1, E2E 3.8 |
| 64 actors | 3.0/4.1/4.4 | 8.7/10.3/10.6 | TTFI 3.4/5.0/5.4, E2E 9.1/11.4/11.9 |
