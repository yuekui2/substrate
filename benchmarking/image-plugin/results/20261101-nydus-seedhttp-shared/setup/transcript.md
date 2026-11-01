### 001 apply (exit 0, 2026-11-01T07:03:18Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261101-nydus-seedhttp-shared/setup/nydus-dragonfly.applied.yaml
```
Log: [logs/001-apply.log](logs/001-apply.log)

### 002 seed-preload (exit 0, 2026-11-01T07:03:21Z)
```
$ SAVE_DIR=/var/lib/ate/nydus-seed-blobs benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-shared-96.jsonl 96
```
Log: [logs/002-seed-preload.log](logs/002-seed-preload.log)

### 003 bake (exit 0, 2026-11-01T07:06:21Z)
```
$ benchmarking/image-plugin/run.sh bake nydus
```
Log: [logs/003-bake.log](logs/003-bake.log)

