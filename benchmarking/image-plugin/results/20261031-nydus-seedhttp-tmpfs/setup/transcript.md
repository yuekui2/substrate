### 001 apply (exit 0, 2026-11-01T01:18:34Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261031-nydus-seedhttp-tmpfs/setup/nydus-dragonfly.applied.yaml
```
Log: [logs/001-apply.log](logs/001-apply.log)

### 002 recreate-bench-pool (exit 0, 2026-11-01T01:18:38Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/002-recreate-bench-pool.log](logs/002-recreate-bench-pool.log)

### 003 wait-ready (exit 0, 2026-11-01T01:22:02Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/003-wait-ready.log](logs/003-wait-ready.log)

