### 001 gen-lists (exit 123, 2026-10-31T03:27:59Z)
```
$ benchmarking/image-plugin/warm-lists.sh gen benchmarking/image-plugin/images-320.jsonl /tmp/warm-lists2
```
Log: [logs/001-gen-lists.log](logs/001-gen-lists.log)

### 002 apply (exit 0, 2026-10-31T03:33:21Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261031-nydus-warmup-bg/nydus-dragonfly.applied.yaml
```
Log: [logs/002-apply.log](logs/002-apply.log)

### 003 recreate-pools-64 (exit 0, 2026-10-31T03:33:24Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/003-recreate-pools-64.log](logs/003-recreate-pools-64.log)

### 004 wait-ready-64 (exit 0, 2026-10-31T03:36:39Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/004-wait-ready-64.log](logs/004-wait-ready-64.log)

### 005 push-lists-64 (exit 0, 2026-10-31T03:37:55Z)
```
$ benchmarking/image-plugin/warm-lists.sh push /tmp/warm-lists2
```
Log: [logs/005-push-lists-64.log](logs/005-push-lists-64.log)

### 006 run-64 (exit 0, 2026-10-31T03:37:59Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/006-run-64.log](logs/006-run-64.log)

### 007 recreate-pools-n5 (exit 0, 2026-10-31T03:41:44Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/007-recreate-pools-n5.log](logs/007-recreate-pools-n5.log)

### 008 wait-ready-n5 (exit 0, 2026-10-31T03:45:00Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/008-wait-ready-n5.log](logs/008-wait-ready-n5.log)

### 009 push-lists-n5 (exit 0, 2026-10-31T03:46:00Z)
```
$ benchmarking/image-plugin/warm-lists.sh push /tmp/warm-lists2
```
Log: [logs/009-push-lists-n5.log](logs/009-push-lists-n5.log)

