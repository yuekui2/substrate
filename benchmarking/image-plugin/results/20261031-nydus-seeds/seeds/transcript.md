### 001 seeds-apply (exit 0, 2026-10-31T04:12:49Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261031-nydus-seeds/seeds/nydus-dragonfly.applied.yaml
```
Log: [logs/001-seeds-apply.log](logs/001-seeds-apply.log)

### 002 seeds-restart-scheduler (exit 0, 2026-10-31T04:12:52Z)
```
$ kubectl -n ate-system rollout restart deploy/dragonfly-scheduler
```
Log: [logs/002-seeds-restart-scheduler.log](logs/002-seeds-restart-scheduler.log)

### 003 seeds-scheduler-ready (exit 0, 2026-10-31T04:12:53Z)
```
$ kubectl -n ate-system rollout status deploy/dragonfly-scheduler --timeout=5m
```
Log: [logs/003-seeds-scheduler-ready.log](logs/003-seeds-scheduler-ready.log)

### 004 seeds-recreate-pools (exit 0, 2026-10-31T04:12:55Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/004-seeds-recreate-pools.log](logs/004-seeds-recreate-pools.log)

### 005 seeds-wait-ready (exit 0, 2026-10-31T04:16:06Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/005-seeds-wait-ready.log](logs/005-seeds-wait-ready.log)

### 006 seeds-seed-preload (exit 0, 2026-10-31T04:17:40Z)
```
$ benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-64.jsonl 64
```
Log: [logs/006-seeds-seed-preload.log](logs/006-seeds-seed-preload.log)

### 007 seeds-run-64 (exit 0, 2026-10-31T04:19:48Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/007-seeds-run-64.log](logs/007-seeds-run-64.log)

