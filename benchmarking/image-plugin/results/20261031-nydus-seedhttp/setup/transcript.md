### 001 apply (exit 0, 2026-10-31T14:53:12Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261031-nydus-seedhttp/setup/nydus-dragonfly.applied.yaml
```
Log: [logs/001-apply.log](logs/001-apply.log)

### 002 restart-blob-seed (exit 0, 2026-10-31T14:53:16Z)
```
$ kubectl -n ate-system delete pods -l app=nydus-blob-seed --wait=false
```
Log: [logs/002-restart-blob-seed.log](logs/002-restart-blob-seed.log)

### 003 blob-seed-ready (exit 0, 2026-10-31T14:53:17Z)
```
$ kubectl -n ate-system rollout status ds/nydus-blob-seed --timeout=5m
```
Log: [logs/003-blob-seed-ready.log](logs/003-blob-seed-ready.log)

### 004 blob-count-image-plugin-nydus-dragonfly-seed-bwgm8 (exit 0, 2026-10-31T14:53:18Z)
```
$ kubectl -n ate-system exec image-plugin-nydus-dragonfly-seed-bwgm8 -c nydus-snapshotter -- nsenter -t 1 -m -- sh -c 'ls /var/lib/ate/nydus-seed-blobs | grep -vc tmp; du -sh /var/lib/ate/nydus-seed-blobs'
```
Log: [logs/004-blob-count-image-plugin-nydus-dragonfly-seed-bwgm8.log](logs/004-blob-count-image-plugin-nydus-dragonfly-seed-bwgm8.log)

### 005 blob-count-image-plugin-nydus-dragonfly-seed-p729z (exit 1, 2026-10-31T14:53:19Z)
```
$ kubectl -n ate-system exec image-plugin-nydus-dragonfly-seed-p729z -c nydus-snapshotter -- nsenter -t 1 -m -- sh -c 'ls /var/lib/ate/nydus-seed-blobs | grep -vc tmp; du -sh /var/lib/ate/nydus-seed-blobs'
```
Log: [logs/005-blob-count-image-plugin-nydus-dragonfly-seed-p729z.log](logs/005-blob-count-image-plugin-nydus-dragonfly-seed-p729z.log)

### 006 recreate-bench-pool (exit 0, 2026-10-31T14:53:20Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/006-recreate-bench-pool.log](logs/006-recreate-bench-pool.log)

### 007 wait-ready (exit 0, 2026-10-31T14:56:26Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/007-wait-ready.log](logs/007-wait-ready.log)

### 008 probe (exit 0, 2026-10-31T14:57:20Z)
```
$ /usr/local/google/home/kuiyue/.gemini/jetski/brain/43283b1e-d79e-4611-9b15-474096a7e806/scratch/seedprobe.sh
```
Log: [logs/008-probe.log](logs/008-probe.log)

