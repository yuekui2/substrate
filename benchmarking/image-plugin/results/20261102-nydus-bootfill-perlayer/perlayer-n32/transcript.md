### 001 rep1-recreate (exit 0, 2026-11-02T14:50:09Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/001-rep1-recreate.log](logs/001-rep1-recreate.log)

### 002 rep1-wait-ready (exit 0, 2026-11-02T14:53:31Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/002-rep1-wait-ready.log](logs/002-rep1-wait-ready.log)

### 003 rep1-run (exit 0, 2026-11-02T14:55:13Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/003-rep1-run.log](logs/003-rep1-run.log)

### 004 rep2-recreate (exit 0, 2026-11-02T14:59:18Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/004-rep2-recreate.log](logs/004-rep2-recreate.log)

### 005 rep2-wait-ready (exit 0, 2026-11-02T15:02:44Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/005-rep2-wait-ready.log](logs/005-rep2-wait-ready.log)

### 006 rep2-run (exit 0, 2026-11-02T15:04:16Z)
```
$ benchmarking/image-plugin/run.sh rep 2 nydus
```
Log: [logs/006-rep2-run.log](logs/006-rep2-run.log)

