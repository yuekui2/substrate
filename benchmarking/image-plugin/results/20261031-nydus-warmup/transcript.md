### 001 apply (exit 0, 2026-10-31T02:21:46Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261031-nydus-warmup/nydus-dragonfly.applied.yaml
```
Log: [logs/001-apply.log](logs/001-apply.log)

### 002 recreate-pools (exit 0, 2026-10-31T02:21:50Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/002-recreate-pools.log](logs/002-recreate-pools.log)

### 003 wait-ready (exit 0, 2026-10-31T02:25:13Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/003-wait-ready.log](logs/003-wait-ready.log)

### 004 push-lists (exit 0, 2026-10-31T02:26:13Z)
```
$ benchmarking/image-plugin/warm-lists.sh push /tmp/warm-lists
```
Log: [logs/004-push-lists.log](logs/004-push-lists.log)

### 005 run-64 (exit 0, 2026-10-31T02:26:18Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/005-run-64.log](logs/005-run-64.log)

### 006 recreate-pools-n5 (exit 0, 2026-10-31T02:31:07Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/006-recreate-pools-n5.log](logs/006-recreate-pools-n5.log)

### 007 wait-ready-n5 (exit 0, 2026-10-31T02:34:17Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/007-wait-ready-n5.log](logs/007-wait-ready-n5.log)

### 008 push-lists-n5 (exit 0, 2026-10-31T02:35:14Z)
```
$ benchmarking/image-plugin/warm-lists.sh push /tmp/warm-lists
```
Log: [logs/008-push-lists-n5.log](logs/008-push-lists-n5.log)

