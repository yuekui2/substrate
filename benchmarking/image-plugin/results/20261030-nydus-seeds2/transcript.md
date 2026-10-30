### 001 e1-apply (exit 0, 2026-10-30T16:09:45Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261030-nydus-seeds2/nydus-dragonfly.applied.yaml
```
Log: [logs/001-e1-apply.log](logs/001-e1-apply.log)

### 002 e1-recreate-pools (exit 0, 2026-10-30T16:09:49Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/002-e1-recreate-pools.log](logs/002-e1-recreate-pools.log)

### 003 e1-wait-ready (exit 0, 2026-10-30T16:21:18Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/003-e1-wait-ready.log](logs/003-e1-wait-ready.log)

### 004 e1-run (exit 0, 2026-10-30T16:22:14Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/004-e1-run.log](logs/004-e1-run.log)

