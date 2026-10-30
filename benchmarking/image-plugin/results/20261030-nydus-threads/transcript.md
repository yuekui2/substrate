### 001 deploy-nydus-threads64 (exit 1, 2026-10-30T13:07:22Z)
```
$ benchmarking/image-plugin/deploy.sh nydus
```
Log: [logs/001-deploy-nydus-threads64.log](logs/001-deploy-nydus-threads64.log)

### 002 bake-d8 (exit 0, 2026-10-30T13:44:20Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh bake nydus
```
Log: [logs/002-bake-d8.log](logs/002-bake-d8.log)

### 003 bake-d4 (exit 0, 2026-10-30T13:44:20Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh bake nydus
```
Log: [logs/003-bake-d4.log](logs/003-bake-d4.log)

### 004 d4-recreate-pools (exit 0, 2026-10-30T13:45:07Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/004-d4-recreate-pools.log](logs/004-d4-recreate-pools.log)

### 005 d4-wait-ready (exit 0, 2026-10-30T13:48:25Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/005-d4-wait-ready.log](logs/005-d4-wait-ready.log)

### 006 apply-nydus-config-version (exit 0, 2026-10-30T13:51:46Z)
```
$ KO=$(cd hack/tools/ko && go tool -n ko); $KO resolve --base-import-paths -f image-plugin/manifests/nydus-dragonfly.yaml 2>/dev/null | python3 benchmarking/image-plugin/scope_plugin.py nydus | kubectl apply -f - | grep -i daemonset; kubectl -n ate-system delete pod -l app=ate-image-plugin,ate.dev/image-plugin=nydus-dragonfly --grace-period=0 --force 2>&1 | tail -3; sleep 90; kubectl -n ate-system get pods -l app=ate-image-plugin,ate.dev/image-plugin=nydus-dragonfly --no-headers; for p in $(kubectl -n ate-system get pods -l app=ate-image-plugin,ate.dev/image-plugin=nydus-dragonfly -o name); do timeout 20 kubectl -n ate-system exec $p -c nydus-snapshotter -- sh -c "ps -eo nlwp,args | grep \"[n]ydusd singleton\" | cut -c1-220"; done
```
Log: [logs/006-apply-nydus-config-version.log](logs/006-apply-nydus-config-version.log)

### 007 d4-run (exit 0, 2026-10-30T13:52:07Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/007-d4-run.log](logs/007-d4-run.log)

### 008 d8-recreate-pools (exit 0, 2026-10-30T14:05:54Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/008-d8-recreate-pools.log](logs/008-d8-recreate-pools.log)

### 009 d8-wait-ready (exit 0, 2026-10-30T14:09:48Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/009-d8-wait-ready.log](logs/009-d8-wait-ready.log)

### 010 d8-run (exit 0, 2026-10-30T14:10:46Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/010-d8-run.log](logs/010-d8-run.log)

