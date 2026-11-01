### 001 recreate-bench-pool (exit 0, 2026-10-31T22:53:37Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/001-recreate-bench-pool.log](logs/001-recreate-bench-pool.log)

### 002 wait-ready (exit 0, 2026-10-31T22:56:48Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/002-wait-ready.log](logs/002-wait-ready.log)

### 003 tracer-ready (exit 0, 2026-10-31T22:57:42Z)
```
$ kubectl -n imgbench rollout status ds/fstrace --timeout=5m
```
Log: [logs/003-tracer-ready.log](logs/003-tracer-ready.log)

