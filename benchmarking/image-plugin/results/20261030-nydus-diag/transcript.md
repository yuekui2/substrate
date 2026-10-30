### 001 diag-apply (exit 0, 2026-10-30T19:47:08Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261030-nydus-diag/nydus-dragonfly.applied.yaml
```
Log: [logs/001-diag-apply.log](logs/001-diag-apply.log)

### 002 diag-recreate-pools (exit 0, 2026-10-30T19:47:11Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/002-diag-recreate-pools.log](logs/002-diag-recreate-pools.log)

### 003 diag-wait-ready (exit 0, 2026-10-30T19:50:57Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/003-diag-wait-ready.log](logs/003-diag-wait-ready.log)

### 004 diag-run (exit 0, 2026-10-30T19:51:56Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/004-diag-run.log](logs/004-diag-run.log)

