### 001 apply (exit 0, 2026-11-02T14:50:01Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261102-nydus-bootfill-perlayer/perlayer-n32/setup/nydus-dragonfly.applied.yaml
```
Log: [logs/001-apply.log](logs/001-apply.log)

### 002 seed-nginx (exit 0, 2026-11-02T14:50:06Z)
```
$ kubectl -n ate-system rollout restart ds/nydus-blob-seed && kubectl -n ate-system rollout status ds/nydus-blob-seed --timeout=5m
```
Log: [logs/002-seed-nginx.log](logs/002-seed-nginx.log)

