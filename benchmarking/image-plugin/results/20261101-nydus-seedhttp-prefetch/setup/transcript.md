### 001 apply (exit 0, 2026-11-01T05:34:05Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261101-nydus-seedhttp-prefetch/setup/nydus-dragonfly.applied.yaml
```
Log: [logs/001-apply.log](logs/001-apply.log)

### 002 recreate-bench-pool (exit 0, 2026-11-01T05:34:10Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/002-recreate-bench-pool.log](logs/002-recreate-bench-pool.log)

### 003 wait-ready (exit 0, 2026-11-01T05:37:19Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/003-wait-ready.log](logs/003-wait-ready.log)

### 004 bake (exit 0, 2026-11-01T05:38:23Z)
```
$ benchmarking/image-plugin/run.sh bake nydus
```
Log: [logs/004-bake.log](logs/004-bake.log)

### 005 seed-preload (exit 0, 2026-11-01T05:41:20Z)
```
$ SAVE_DIR=/var/lib/ate/nydus-seed-blobs benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-96-pf.jsonl 96
```
Log: [logs/005-seed-preload.log](logs/005-seed-preload.log)

### 006 bake2 (exit 0, 2026-11-01T06:02:35Z)
```
$ benchmarking/image-plugin/run.sh bake nydus
```
Log: [logs/006-bake2.log](logs/006-bake2.log)

