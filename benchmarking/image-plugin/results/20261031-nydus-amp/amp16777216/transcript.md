### 001 amp16777216-apply (exit 0, 2026-10-31T05:39:21Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261031-nydus-amp/amp16777216/nydus-dragonfly.applied.yaml
```
Log: [logs/001-amp16777216-apply.log](logs/001-amp16777216-apply.log)

### 002 amp16777216-restart-scheduler (exit 0, 2026-10-31T05:39:25Z)
```
$ kubectl -n ate-system rollout restart deploy/dragonfly-scheduler
```
Log: [logs/002-amp16777216-restart-scheduler.log](logs/002-amp16777216-restart-scheduler.log)

### 003 amp16777216-scheduler-ready (exit 0, 2026-10-31T05:39:26Z)
```
$ kubectl -n ate-system rollout status deploy/dragonfly-scheduler --timeout=5m
```
Log: [logs/003-amp16777216-scheduler-ready.log](logs/003-amp16777216-scheduler-ready.log)

### 004 amp16777216-recreate-pools (exit 0, 2026-10-31T05:39:29Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/004-amp16777216-recreate-pools.log](logs/004-amp16777216-recreate-pools.log)

### 005 amp16777216-wait-ready (exit 0, 2026-10-31T05:42:35Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/005-amp16777216-wait-ready.log](logs/005-amp16777216-wait-ready.log)

### 006 amp16777216-check-env (exit 0, 2026-10-31T05:44:11Z)
```
$ kubectl -n ate-system get ds image-plugin-nydus-dragonfly -o jsonpath='{.spec.template.spec.containers[?(@.name=="nydus-snapshotter")].env}'
```
Log: [logs/006-amp16777216-check-env.log](logs/006-amp16777216-check-env.log)

### 007 amp16777216-seed-preload (exit 0, 2026-10-31T05:44:12Z)
```
$ benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-64.jsonl 64
```
Log: [logs/007-amp16777216-seed-preload.log](logs/007-amp16777216-seed-preload.log)

### 008 amp16777216-run-64 (exit 0, 2026-10-31T05:46:13Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/008-amp16777216-run-64.log](logs/008-amp16777216-run-64.log)

