### 001 e1b-apply (exit 0, 2026-10-30T16:52:02Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261030-nydus-seeds2-dfnopf/nydus-dragonfly.applied.yaml
```
Log: [logs/001-e1b-apply.log](logs/001-e1b-apply.log)

### 002 e1b-recreate-pools (exit 0, 2026-10-30T16:52:08Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/002-e1b-recreate-pools.log](logs/002-e1b-recreate-pools.log)

### 003 e1b-wait-ready (exit 0, 2026-10-30T16:59:57Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/003-e1b-wait-ready.log](logs/003-e1b-wait-ready.log)

### 004 e1b-run (exit 0, 2026-10-30T17:02:37Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/004-e1b-run.log](logs/004-e1b-run.log)

