### 001 update-node-sampler (exit 0, 2026-10-30T15:11:02Z)
```
$ kubectl -n imgbench create configmap node-sampler --from-file=benchmarking/image-plugin/node-sampler.sh --dry-run=client -o yaml | kubectl apply -f - && kubectl -n imgbench rollout restart ds/node-sampler
```
Log: [logs/001-update-node-sampler.log](logs/001-update-node-sampler.log)

### 002 deploy-dragonfly-v2.5 (exit 0, 2026-10-30T15:11:20Z)
```
$ kubectl -n ate-system delete ds image-plugin-nydus-dragonfly image-plugin-nydus-dragonfly-seed --wait=false; kubectl -n ate-system delete pod -l app=ate-image-plugin,ate.dev/image-plugin=nydus-dragonfly --grace-period=0 --force; kubectl -n ate-system delete deploy dragonfly-manager dragonfly-scheduler dragonfly-mysql dragonfly-redis --wait=true; kubectl apply -f benchmarking/image-plugin/results/20261030-nydus-dfv25/nydus-dragonfly.applied.yaml; kubectl -n ate-system rollout status deploy/dragonfly-mysql --timeout=5m; kubectl -n ate-system rollout status deploy/dragonfly-manager --timeout=10m; kubectl -n ate-system rollout status deploy/dragonfly-scheduler --timeout=10m; kubectl -n ate-system get pods -o wide | grep -E 'dragonfly'
```
Log: [logs/002-deploy-dragonfly-v2.5.log](logs/002-deploy-dragonfly-v2.5.log)

### 003 bake-d8 (exit 0, 2026-10-30T15:14:00Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh bake nydus
```
Log: [logs/003-bake-d8.log](logs/003-bake-d8.log)

### 004 d8-recreate-pools (exit 0, 2026-10-30T15:14:58Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/004-d8-recreate-pools.log](logs/004-d8-recreate-pools.log)

### 005 d8-wait-ready (exit 0, 2026-10-30T15:18:40Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/005-d8-wait-ready.log](logs/005-d8-wait-ready.log)

### 006 d8-run (exit 0, 2026-10-30T15:20:18Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/006-d8-run.log](logs/006-d8-run.log)

