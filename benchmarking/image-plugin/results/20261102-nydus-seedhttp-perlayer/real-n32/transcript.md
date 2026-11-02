### 001 real-n32-recreate (exit 0, 2026-11-02T03:35:23Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/001-real-n32-recreate.log](logs/001-real-n32-recreate.log)

### 002 real-n32-wait-ready (exit 0, 2026-11-02T03:38:28Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/002-real-n32-wait-ready.log](logs/002-real-n32-wait-ready.log)

### 003 real-n32-run (exit 0, 2026-11-02T03:40:34Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/003-real-n32-run.log](logs/003-real-n32-run.log)

### 004 rep2-recreate (exit 0, 2026-11-02T04:05:47Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/004-rep2-recreate.log](logs/004-rep2-recreate.log)

### 005 rep2-wait-ready (exit 0, 2026-11-02T04:11:45Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/005-rep2-wait-ready.log](logs/005-rep2-wait-ready.log)

### 006 rep2-run (exit 0, 2026-11-02T04:13:10Z)
```
$ benchmarking/image-plugin/run.sh rep 2 nydus
```
Log: [logs/006-rep2-run.log](logs/006-rep2-run.log)

