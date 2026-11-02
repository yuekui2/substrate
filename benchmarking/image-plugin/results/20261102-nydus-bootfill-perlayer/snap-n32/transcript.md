### 001 rep1-recreate (exit 0, 2026-11-02T15:08:30Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/001-rep1-recreate.log](logs/001-rep1-recreate.log)

### 002 rep1-wait-ready (exit 0, 2026-11-02T15:11:43Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/002-rep1-wait-ready.log](logs/002-rep1-wait-ready.log)

### 003 rep1-run (exit 0, 2026-11-02T15:13:21Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/003-rep1-run.log](logs/003-rep1-run.log)

