### 001 apply (exit 0, 2026-10-31T18:46:20Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261031-nydus-seedhttp-resolve/setup/nydus-dragonfly.applied.yaml
```
Log: [logs/001-apply.log](logs/001-apply.log)

### 002 recreate-bench-pool (exit 0, 2026-10-31T18:46:23Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/002-recreate-bench-pool.log](logs/002-recreate-bench-pool.log)

### 003 wait-ready (exit 0, 2026-10-31T18:49:39Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/003-wait-ready.log](logs/003-wait-ready.log)

### 004 seed-preload (exit 0, 2026-10-31T18:50:37Z)
```
$ SAVE_DIR=/var/lib/ate/nydus-seed-blobs benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-64.jsonl 64
```
Log: [logs/004-seed-preload.log](logs/004-seed-preload.log)

