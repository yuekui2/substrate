### 001 recreate (exit 0, 2026-11-02T05:51:56Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/001-recreate.log](logs/001-recreate.log)

### 003 recreate (exit 0, 2026-11-02T05:55:30Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/003-recreate.log](logs/003-recreate.log)

### 005 recreate (exit 0, 2026-11-02T06:16:02Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/005-recreate.log](logs/005-recreate.log)

### 006 wait-ready (exit 0, 2026-11-02T06:19:10Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/006-wait-ready.log](logs/006-wait-ready.log)

### 007 tracer-ready (exit 0, 2026-11-02T06:33:03Z)
```
$ kubectl -n imgbench rollout status ds/readset --timeout=5m
```
Log: [logs/007-tracer-ready.log](logs/007-tracer-ready.log)

### 008 run (exit 0, 2026-11-02T06:33:23Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/008-run.log](logs/008-run.log)

