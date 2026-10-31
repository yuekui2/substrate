### 001 seeds-warm-apply (exit 0, 2026-10-31T04:23:38Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261031-nydus-seeds/seeds-warm/nydus-dragonfly.applied.yaml
```
Log: [logs/001-seeds-warm-apply.log](logs/001-seeds-warm-apply.log)

### 002 seeds-warm-restart-scheduler (exit 0, 2026-10-31T04:23:41Z)
```
$ kubectl -n ate-system rollout restart deploy/dragonfly-scheduler
```
Log: [logs/002-seeds-warm-restart-scheduler.log](logs/002-seeds-warm-restart-scheduler.log)

### 003 seeds-warm-scheduler-ready (exit 0, 2026-10-31T04:23:42Z)
```
$ kubectl -n ate-system rollout status deploy/dragonfly-scheduler --timeout=5m
```
Log: [logs/003-seeds-warm-scheduler-ready.log](logs/003-seeds-warm-scheduler-ready.log)

### 004 seeds-warm-recreate-pools (exit 0, 2026-10-31T04:23:44Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/004-seeds-warm-recreate-pools.log](logs/004-seeds-warm-recreate-pools.log)

### 005 seeds-warm-wait-ready (exit 0, 2026-10-31T04:26:49Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/005-seeds-warm-wait-ready.log](logs/005-seeds-warm-wait-ready.log)

### 006 seeds-warm-seed-preload (exit 0, 2026-10-31T04:27:52Z)
```
$ benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-64.jsonl 64
```
Log: [logs/006-seeds-warm-seed-preload.log](logs/006-seeds-warm-seed-preload.log)

### 007 seeds-warm-push-lists (exit 0, 2026-10-31T04:30:00Z)
```
$ benchmarking/image-plugin/warm-lists.sh push /tmp/warm-lists2
```
Log: [logs/007-seeds-warm-push-lists.log](logs/007-seeds-warm-push-lists.log)

### 008 seeds-warm-run-64 (exit 0, 2026-10-31T04:30:05Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/008-seeds-warm-run-64.log](logs/008-seeds-warm-run-64.log)

