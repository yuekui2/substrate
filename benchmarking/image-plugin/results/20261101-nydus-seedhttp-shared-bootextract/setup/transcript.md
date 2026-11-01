### 001 apply (exit 0, 2026-11-01T21:10:15Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261101-nydus-seedhttp-shared-bootextract/setup/nydus-dragonfly.applied.yaml
```
Log: [logs/001-apply.log](logs/001-apply.log)

### 002 seed-nginx (exit 0, 2026-11-01T21:10:19Z)
```
$ kubectl -n ate-system rollout restart ds/nydus-blob-seed && kubectl -n ate-system rollout status ds/nydus-blob-seed --timeout=5m
```
Log: [logs/002-seed-nginx.log](logs/002-seed-nginx.log)

### 003 seed-preload (exit 0, 2026-11-01T21:10:27Z)
```
$ SAVE_DIR=/var/lib/ate/nydus-seed-blobs benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-shared-96.jsonl 96
```
Log: [logs/003-seed-preload.log](logs/003-seed-preload.log)

