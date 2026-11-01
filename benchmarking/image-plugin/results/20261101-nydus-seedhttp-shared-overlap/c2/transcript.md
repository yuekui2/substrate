### 001 recreate (exit 0, 2026-11-01T19:00:42Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/001-recreate.log](logs/001-recreate.log)

### 002 wait-ready (exit 0, 2026-11-01T19:03:48Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/002-wait-ready.log](logs/002-wait-ready.log)

### 003 run (exit 0, 2026-11-01T19:05:22Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/003-run.log](logs/003-run.log)

