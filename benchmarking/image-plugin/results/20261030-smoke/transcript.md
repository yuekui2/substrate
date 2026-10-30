### 002 image-sizes (exit 0, 2026-10-30T01:28:18Z)
```
$ python3 benchmarking/image-plugin/image_sizes.py benchmarking/image-plugin/images-320.jsonl benchmarking/image-plugin/results/20261030-smoke/images
```
Log: [logs/002-image-sizes.log](logs/002-image-sizes.log)

### 001 recreate-pools-3node-hyperdisk (exit 0, 2026-10-30T01:28:18Z)
```
$ benchmarking/image-plugin/pools.sh recreate
```
Log: [logs/001-recreate-pools-3node-hyperdisk.log](logs/001-recreate-pools-3node-hyperdisk.log)

### 003 scale-workers-and-sampler (exit 0, 2026-10-30T02:28:39Z)
```
$ for a in pull riptide nydus spegel; do kubectl -n imgbench patch workerpool bench-$a --type merge -p "{\"spec\":{\"replicas\":48}}"; done; kubectl -n imgbench create configmap node-sampler --from-file=benchmarking/image-plugin/node-sampler.sh --dry-run=client -o yaml | kubectl apply -f -; kubectl apply -f benchmarking/image-plugin/manifests/node-sampler.yaml; benchmarking/image-plugin/wait-ready.sh
```
Log: [logs/003-scale-workers-and-sampler.log](logs/003-scale-workers-and-sampler.log)

### 005 validate-loader-pull-8x2 (exit 0, 2026-10-30T04:26:29Z)
```
$ benchmarking/image-plugin/run.sh rep 0 pull
```
Log: [logs/005-validate-loader-pull-8x2.log](logs/005-validate-loader-pull-8x2.log)

### 006 bake-goldens (exit 0, 2026-10-30T04:38:09Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh bake pull riptide nydus spegel
```
Log: [logs/006-bake-goldens.log](logs/006-bake-goldens.log)

### 007 rep1-pull-recreate-pool (exit 0, 2026-10-30T04:59:37Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate pull
```
Log: [logs/007-rep1-pull-recreate-pool.log](logs/007-rep1-pull-recreate-pool.log)

### 008 rep1-pull-wait-ready (exit 0, 2026-10-30T05:03:02Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh pull
```
Log: [logs/008-rep1-pull-wait-ready.log](logs/008-rep1-pull-wait-ready.log)

### 009 rep1-pull-run (exit 0, 2026-10-30T05:03:30Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 pull
```
Log: [logs/009-rep1-pull-run.log](logs/009-rep1-pull-run.log)

### 010 rep1-riptide-recreate-pool (exit 0, 2026-10-30T05:15:26Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate riptide
```
Log: [logs/010-rep1-riptide-recreate-pool.log](logs/010-rep1-riptide-recreate-pool.log)

### 011 rep1-riptide-wait-ready (exit 0, 2026-10-30T05:18:59Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh riptide
```
Log: [logs/011-rep1-riptide-wait-ready.log](logs/011-rep1-riptide-wait-ready.log)

### 012 rep1-riptide-run (exit 143, 2026-10-30T05:19:27Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 riptide
```
Log: [logs/012-rep1-riptide-run.log](logs/012-rep1-riptide-run.log)

