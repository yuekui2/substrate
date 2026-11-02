### 001 recreate (exit 0, 2026-11-02T00:25:35Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/001-recreate.log](logs/001-recreate.log)

### 002 wait-ready (exit 0, 2026-11-02T00:28:46Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/002-wait-ready.log](logs/002-wait-ready.log)

### 003 run (exit 0, 2026-11-02T00:30:05Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/003-run.log](logs/003-run.log)

### 004 recreate (exit 0, 2026-11-02T00:40:19Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/004-recreate.log](logs/004-recreate.log)

### 005 wait-ready (exit 0, 2026-11-02T00:43:27Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/005-wait-ready.log](logs/005-wait-ready.log)

### 006 run (exit 0, 2026-11-02T00:44:52Z)
```
$ benchmarking/image-plugin/run.sh rep 2 nydus
```
Log: [logs/006-run.log](logs/006-run.log)

