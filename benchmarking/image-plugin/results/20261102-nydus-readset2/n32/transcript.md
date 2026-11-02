### 001 recreate (exit 0, 2026-11-02T05:21:29Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/001-recreate.log](logs/001-recreate.log)

### 002 wait-ready (exit 0, 2026-11-02T05:24:40Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/002-wait-ready.log](logs/002-wait-ready.log)

### 003 tracer-ready (exit 0, 2026-11-02T05:27:40Z)
```
$ kubectl -n imgbench rollout status ds/readset --timeout=5m
```
Log: [logs/003-tracer-ready.log](logs/003-tracer-ready.log)

### 004 run (exit 0, 2026-11-02T05:28:11Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/004-run.log](logs/004-run.log)

