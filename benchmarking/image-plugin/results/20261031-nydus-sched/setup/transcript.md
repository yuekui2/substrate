### 001 apply (exit 0, 2026-10-31T05:24:18Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261031-nydus-sched/setup/nydus-dragonfly.applied.yaml
```
Log: [logs/001-apply.log](logs/001-apply.log)

### 002 restart-scheduler (exit 0, 2026-10-31T05:24:21Z)
```
$ kubectl -n ate-system rollout restart deploy/dragonfly-scheduler
```
Log: [logs/002-restart-scheduler.log](logs/002-restart-scheduler.log)

### 003 scheduler-ready (exit 0, 2026-10-31T05:24:22Z)
```
$ kubectl -n ate-system rollout status deploy/dragonfly-scheduler --timeout=5m
```
Log: [logs/003-scheduler-ready.log](logs/003-scheduler-ready.log)

### 004 recreate-pools (exit 0, 2026-10-31T05:24:25Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/004-recreate-pools.log](logs/004-recreate-pools.log)

### 005 wait-ready (exit 0, 2026-10-31T05:28:00Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/005-wait-ready.log](logs/005-wait-ready.log)

### 006 seed-preload (exit 0, 2026-10-31T05:28:54Z)
```
$ benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-64.jsonl 64
```
Log: [logs/006-seed-preload.log](logs/006-seed-preload.log)

