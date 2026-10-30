### 001 apply-nydus-backend-timeout60 (exit 0, 2026-10-30T14:27:31Z)
```
$ KO=$(cd hack/tools/ko && go tool -n ko); $KO resolve --base-import-paths -f image-plugin/manifests/nydus-dragonfly.yaml 2>/dev/null | python3 benchmarking/image-plugin/scope_plugin.py nydus | kubectl apply -f - | grep -i daemonset
```
Log: [logs/001-apply-nydus-backend-timeout60.log](logs/001-apply-nydus-backend-timeout60.log)

### 003 apply-nydus-backend-timeout60-retry (exit 0, 2026-10-30T14:28:03Z)
```
$ python3 benchmarking/image-plugin/scope_plugin.py nydus < /tmp/nydus-resolved.yaml | kubectl apply -f -; grep -c "\"timeout\": 60" /tmp/nydus-resolved.yaml
```
Log: [logs/003-apply-nydus-backend-timeout60-retry.log](logs/003-apply-nydus-backend-timeout60-retry.log)

### 002 bake-d8 (exit 0, 2026-10-30T14:27:44Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh bake nydus
```
Log: [logs/002-bake-d8.log](logs/002-bake-d8.log)

### 005 d8-recreate-pools (exit 0, 2026-10-30T14:28:28Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/005-d8-recreate-pools.log](logs/005-d8-recreate-pools.log)

### 006 d8-wait-ready (exit 0, 2026-10-30T14:31:46Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/006-d8-wait-ready.log](logs/006-d8-wait-ready.log)

### 007 d8-run (exit 0, 2026-10-30T14:32:40Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/007-d8-run.log](logs/007-d8-run.log)

