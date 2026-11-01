### 001 recreate (exit 0, 2026-11-01T14:55:36Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/001-recreate.log](logs/001-recreate.log)

### 002 wait-ready (exit 0, 2026-11-01T14:58:46Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/002-wait-ready.log](logs/002-wait-ready.log)

### 003 rep1 (exit 0, 2026-11-01T14:59:39Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/003-rep1.log](logs/003-rep1.log)

### 004 rep2 (exit 0, 2026-11-01T15:03:36Z)
```
$ benchmarking/image-plugin/run.sh rep 2 nydus
```
Log: [logs/004-rep2.log](logs/004-rep2.log)

### 005 rep3 (exit 0, 2026-11-01T15:07:22Z)
```
$ benchmarking/image-plugin/run.sh rep 3 nydus
```
Log: [logs/005-rep3.log](logs/005-rep3.log)

