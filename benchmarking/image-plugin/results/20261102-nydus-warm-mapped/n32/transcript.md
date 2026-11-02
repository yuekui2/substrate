### 001 rep1-recreate (exit 0, 2026-11-02T21:09:16Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/001-rep1-recreate.log](logs/001-rep1-recreate.log)

### 002 rep1-wait-ready (exit 0, 2026-11-02T21:12:44Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/002-rep1-wait-ready.log](logs/002-rep1-wait-ready.log)

### 003 rep1-push-lists (exit 0, 2026-11-02T21:13:41Z)
```
$ benchmarking/image-plugin/warm-lists.sh push /usr/local/google/home/kuiyue/.gemini/jetski/brain/d307c843-d9dd-4a26-8249-2733d9f2ff6a/scratch/warm-lists-mapped
```
Log: [logs/003-rep1-push-lists.log](logs/003-rep1-push-lists.log)

### 004 rep1-run (exit 0, 2026-11-02T21:13:46Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/004-rep1-run.log](logs/004-rep1-run.log)

### 005 rep2-recreate (exit 0, 2026-11-02T21:17:37Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/005-rep2-recreate.log](logs/005-rep2-recreate.log)

### 006 rep2-wait-ready (exit 0, 2026-11-02T21:20:55Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/006-rep2-wait-ready.log](logs/006-rep2-wait-ready.log)

### 007 rep2-push-lists (exit 0, 2026-11-02T21:21:50Z)
```
$ benchmarking/image-plugin/warm-lists.sh push /usr/local/google/home/kuiyue/.gemini/jetski/brain/d307c843-d9dd-4a26-8249-2733d9f2ff6a/scratch/warm-lists-mapped
```
Log: [logs/007-rep2-push-lists.log](logs/007-rep2-push-lists.log)

### 008 rep2-run (exit 0, 2026-11-02T21:21:54Z)
```
$ benchmarking/image-plugin/run.sh rep 2 nydus
```
Log: [logs/008-rep2-run.log](logs/008-rep2-run.log)

