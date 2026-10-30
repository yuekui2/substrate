### 001 deploy-fixed-plugins (exit 0, 2026-10-30T11:47:52Z)
```
$ seed=$(kubectl get nodes -l bench-arm=nydus -o jsonpath="{.items[0].metadata.name}"); kubectl label node $seed ate.dev/dragonfly-seed=true --overwrite; kubectl -n ate-system delete ds image-plugin-nydus-dragonfly --wait=true; benchmarking/image-plugin/deploy.sh nydus spegel && benchmarking/image-plugin/wait-ready.sh nydus spegel; kubectl -n ate-system get pods -o wide -l "ate.dev/image-plugin in (nydus-dragonfly,spegel)"
```
Log: [logs/001-deploy-fixed-plugins.log](logs/001-deploy-fixed-plugins.log)

### 002 bake-d8 (exit 0, 2026-10-30T12:11:39Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh bake
```
Log: [logs/002-bake-d8.log](logs/002-bake-d8.log)

### 003 bake-d4 (exit 0, 2026-10-30T12:11:39Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh bake
```
Log: [logs/003-bake-d4.log](logs/003-bake-d4.log)

### 004 d4-recreate-pools (exit 0, 2026-10-30T12:14:17Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate
```
Log: [logs/004-d4-recreate-pools.log](logs/004-d4-recreate-pools.log)

### 005 d4-wait-ready (exit 0, 2026-10-30T12:18:13Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh
```
Log: [logs/005-d4-wait-ready.log](logs/005-d4-wait-ready.log)

### 006 d4-run (exit 0, 2026-10-30T12:19:23Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1
```
Log: [logs/006-d4-run.log](logs/006-d4-run.log)

### 007 d8-recreate-pools (exit 0, 2026-10-30T12:34:30Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate
```
Log: [logs/007-d8-recreate-pools.log](logs/007-d8-recreate-pools.log)

### 008 d8-wait-ready (exit 0, 2026-10-30T12:38:46Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh
```
Log: [logs/008-d8-wait-ready.log](logs/008-d8-wait-ready.log)

### 009 d8-run (exit 0, 2026-10-30T12:39:56Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1
```
Log: [logs/009-d8-run.log](logs/009-d8-run.log)

