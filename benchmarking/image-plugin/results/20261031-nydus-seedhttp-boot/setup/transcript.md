### 001 apply (exit 0, 2026-10-31T16:10:10Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261031-nydus-seedhttp-boot/setup/nydus-dragonfly.applied.yaml
```
Log: [logs/001-apply.log](logs/001-apply.log)

### 002 restart-blob-seed (exit 0, 2026-10-31T16:10:14Z)
```
$ kubectl -n ate-system delete pods -l app=nydus-blob-seed --wait=false
```
Log: [logs/002-restart-blob-seed.log](logs/002-restart-blob-seed.log)

### 003 blob-seed-ready (exit 0, 2026-10-31T16:10:15Z)
```
$ kubectl -n ate-system rollout status ds/nydus-blob-seed --timeout=5m
```
Log: [logs/003-blob-seed-ready.log](logs/003-blob-seed-ready.log)

### 004 blob-count-image-plugin-nydus-dragonfly-seed-9f7nx (exit 0, 2026-10-31T16:10:17Z)
```
$ kubectl -n ate-system exec image-plugin-nydus-dragonfly-seed-9f7nx -c nydus-snapshotter -- nsenter -t 1 -m -- sh -c 'ls /var/lib/ate/nydus-seed-blobs | grep -vc tmp; du -sh /var/lib/ate/nydus-seed-blobs'
```
Log: [logs/004-blob-count-image-plugin-nydus-dragonfly-seed-9f7nx.log](logs/004-blob-count-image-plugin-nydus-dragonfly-seed-9f7nx.log)

### 005 blob-count-image-plugin-nydus-dragonfly-seed-fn97l (exit 0, 2026-10-31T16:10:18Z)
```
$ kubectl -n ate-system exec image-plugin-nydus-dragonfly-seed-fn97l -c nydus-snapshotter -- nsenter -t 1 -m -- sh -c 'ls /var/lib/ate/nydus-seed-blobs | grep -vc tmp; du -sh /var/lib/ate/nydus-seed-blobs'
```
Log: [logs/005-blob-count-image-plugin-nydus-dragonfly-seed-fn97l.log](logs/005-blob-count-image-plugin-nydus-dragonfly-seed-fn97l.log)

### 006 recreate-bench-pool (exit 0, 2026-10-31T16:10:19Z)
```
$ KEEP_SEEDS=1 benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/006-recreate-bench-pool.log](logs/006-recreate-bench-pool.log)

### 008 blob-seed-ready (exit 0, 2026-10-31T18:02:35Z)
```
$ kubectl -n ate-system rollout status ds/nydus-blob-seed --timeout=5m
```
Log: [logs/008-blob-seed-ready.log](logs/008-blob-seed-ready.log)

### 009 wait-ready (exit 0, 2026-10-31T18:02:36Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/009-wait-ready.log](logs/009-wait-ready.log)

### 010 seed-preload (exit 0, 2026-10-31T18:02:42Z)
```
$ SAVE_DIR=/var/lib/ate/nydus-seed-blobs benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-64.jsonl 64
```
Log: [logs/010-seed-preload.log](logs/010-seed-preload.log)

### 011 check-manifest (exit 0, 2026-10-31T18:05:50Z)
```
$ kubectl -n ate-system exec image-plugin-nydus-dragonfly-g85cs -c nydus-snapshotter -- nsenter -t 1 -m -n -- curl -s -o /dev/null -w '%{http_code} %{size_download} %{time_total}\n' http://34.118.228.23:4010/v2/kuiyue-gke-dev/kuiyue-gke-dev-repo/swebench-nydus/sweb.eval.x86_64.sphinx-doc_1776_sphinx-9591/manifests/sha256:fe55f16a9153ad44268fdfe5c2113ef83e8aff757fb44f316c353b4e1381de14
```
Log: [logs/011-check-manifest.log](logs/011-check-manifest.log)

