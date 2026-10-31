### 001 apply (exit 0, 2026-10-31T14:24:59Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261031-nydus-seedhttp/setup/nydus-dragonfly.applied.yaml
```
Log: [logs/001-apply.log](logs/001-apply.log)

### 002 restart-scheduler (exit 0, 2026-10-31T14:25:04Z)
```
$ kubectl -n ate-system rollout restart deploy/dragonfly-scheduler
```
Log: [logs/002-restart-scheduler.log](logs/002-restart-scheduler.log)

### 003 scheduler-ready (exit 0, 2026-10-31T14:25:05Z)
```
$ kubectl -n ate-system rollout status deploy/dragonfly-scheduler --timeout=5m
```
Log: [logs/003-scheduler-ready.log](logs/003-scheduler-ready.log)

### 004 recreate-pools (exit 0, 2026-10-31T14:25:07Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/004-recreate-pools.log](logs/004-recreate-pools.log)

### 005 wait-ready (exit 0, 2026-10-31T14:28:23Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/005-wait-ready.log](logs/005-wait-ready.log)

### 006 blob-seed-ready (exit 1, 2026-10-31T14:29:22Z)
```
$ kubectl -n ate-system rollout status ds/nydus-blob-seed --timeout=10m
```
Log: [logs/006-blob-seed-ready.log](logs/006-blob-seed-ready.log)

### 007 seed-preload (exit 0, 2026-10-31T14:39:22Z)
```
$ SAVE_DIR=/var/lib/ate/nydus-seed-blobs benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-64.jsonl 64
```
Log: [logs/007-seed-preload.log](logs/007-seed-preload.log)

### 008 probe (exit 0, 2026-10-31T14:42:18Z)
```
$ /usr/local/google/home/kuiyue/.gemini/jetski/brain/43283b1e-d79e-4611-9b15-474096a7e806/scratch/seedprobe.sh
```
Log: [logs/008-probe.log](logs/008-probe.log)

