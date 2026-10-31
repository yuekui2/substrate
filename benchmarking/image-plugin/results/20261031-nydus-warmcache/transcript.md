### 001 apply (exit 0, 2026-10-31T01:44:01Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261031-nydus-warmcache/nydus-dragonfly.applied.yaml
```
Log: [logs/001-apply.log](logs/001-apply.log)

### 002 restart-scheduler (exit 0, 2026-10-31T01:44:04Z)
```
$ kubectl -n ate-system rollout restart deploy/dragonfly-scheduler
```
Log: [logs/002-restart-scheduler.log](logs/002-restart-scheduler.log)

### 003 recreate-pools (exit 0, 2026-10-31T01:44:05Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/003-recreate-pools.log](logs/003-recreate-pools.log)

### 004 wait-ready (exit 0, 2026-10-31T01:47:35Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/004-wait-ready.log](logs/004-wait-ready.log)

### 005 rep1-cold (exit 0, 2026-10-31T01:49:29Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/005-rep1-cold.log](logs/005-rep1-cold.log)

### 006 restart-plugin (exit 0, 2026-10-31T01:53:10Z)
```
$ kubectl -n ate-system delete pods -l app=ate-image-plugin,ate.dev/image-plugin=nydus-dragonfly --grace-period=0 --force
```
Log: [logs/006-restart-plugin.log](logs/006-restart-plugin.log)

### 007 wait-ready-2 (exit 0, 2026-10-31T01:53:11Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/007-wait-ready-2.log](logs/007-wait-ready-2.log)

### 008 rep2-dfdaemon-warm (exit 0, 2026-10-31T01:53:27Z)
```
$ benchmarking/image-plugin/run.sh rep 2 nydus
```
Log: [logs/008-rep2-dfdaemon-warm.log](logs/008-rep2-dfdaemon-warm.log)

### 009 rep3-all-warm (exit 0, 2026-10-31T01:57:01Z)
```
$ benchmarking/image-plugin/run.sh rep 3 nydus
```
Log: [logs/009-rep3-all-warm.log](logs/009-rep3-all-warm.log)

