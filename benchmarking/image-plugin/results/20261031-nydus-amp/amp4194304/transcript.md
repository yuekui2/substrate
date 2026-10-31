### 001 amp4194304-apply (exit 0, 2026-10-31T05:49:57Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261031-nydus-amp/amp4194304/nydus-dragonfly.applied.yaml
```
Log: [logs/001-amp4194304-apply.log](logs/001-amp4194304-apply.log)

### 002 amp4194304-restart-scheduler (exit 0, 2026-10-31T05:50:00Z)
```
$ kubectl -n ate-system rollout restart deploy/dragonfly-scheduler
```
Log: [logs/002-amp4194304-restart-scheduler.log](logs/002-amp4194304-restart-scheduler.log)

### 003 amp4194304-scheduler-ready (exit 0, 2026-10-31T05:50:01Z)
```
$ kubectl -n ate-system rollout status deploy/dragonfly-scheduler --timeout=5m
```
Log: [logs/003-amp4194304-scheduler-ready.log](logs/003-amp4194304-scheduler-ready.log)

### 004 amp4194304-recreate-pools (exit 0, 2026-10-31T05:50:03Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/004-amp4194304-recreate-pools.log](logs/004-amp4194304-recreate-pools.log)

### 005 amp4194304-wait-ready (exit 0, 2026-10-31T05:53:07Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/005-amp4194304-wait-ready.log](logs/005-amp4194304-wait-ready.log)

### 006 amp4194304-check-env (exit 0, 2026-10-31T05:54:20Z)
```
$ kubectl -n ate-system get ds image-plugin-nydus-dragonfly -o jsonpath='{.spec.template.spec.containers[?(@.name=="nydus-snapshotter")].env}'
```
Log: [logs/006-amp4194304-check-env.log](logs/006-amp4194304-check-env.log)

### 007 amp4194304-seed-preload (exit 0, 2026-10-31T05:54:20Z)
```
$ benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-64.jsonl 64
```
Log: [logs/007-amp4194304-seed-preload.log](logs/007-amp4194304-seed-preload.log)

### 008 amp4194304-run-64 (exit 0, 2026-10-31T05:56:19Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/008-amp4194304-run-64.log](logs/008-amp4194304-run-64.log)

