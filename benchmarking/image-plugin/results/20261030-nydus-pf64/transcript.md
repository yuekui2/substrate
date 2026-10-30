### 001 bake-pf (exit 0, 2026-10-30T17:20:53Z)
```
$ benchmarking/image-plugin/run.sh bake nydus
```
Log: [logs/001-bake-pf.log](logs/001-bake-pf.log)

### 002 e3-apply (exit 0, 2026-10-30T17:24:01Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261030-nydus-pf64/e3.applied.yaml
```
Log: [logs/002-e3-apply.log](logs/002-e3-apply.log)

### 003 e3-recreate-pools (exit 0, 2026-10-30T17:24:04Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/003-e3-recreate-pools.log](logs/003-e3-recreate-pools.log)

### 004 e3-wait-ready (exit 0, 2026-10-30T17:27:29Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/004-e3-wait-ready.log](logs/004-e3-wait-ready.log)

### 005 e2old-apply (exit 0, 2026-10-30T17:29:44Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261030-nydus-pf64/e2old.applied.yaml
```
Log: [logs/005-e2old-apply.log](logs/005-e2old-apply.log)

### 006 e2old-recreate-pools (exit 0, 2026-10-30T17:29:47Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/006-e2old-recreate-pools.log](logs/006-e2old-recreate-pools.log)

### 007 e2old-wait-ready (exit 0, 2026-10-30T17:33:11Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/007-e2old-wait-ready.log](logs/007-e2old-wait-ready.log)

### 008 e2-apply (exit 0, 2026-10-30T17:37:30Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261030-nydus-pf64/e2.applied.yaml
```
Log: [logs/008-e2-apply.log](logs/008-e2-apply.log)

### 009 e2-recreate-pools (exit 0, 2026-10-30T17:37:33Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/009-e2-recreate-pools.log](logs/009-e2-recreate-pools.log)

### 010 e2-wait-ready (exit 0, 2026-10-30T17:41:08Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/010-e2-wait-ready.log](logs/010-e2-wait-ready.log)

### 011 bake-pf (exit 0, 2026-10-30T19:20:47Z)
```
$ benchmarking/image-plugin/run.sh bake nydus
```
Log: [logs/011-bake-pf.log](logs/011-bake-pf.log)

### 012 e3-apply (exit 0, 2026-10-30T19:21:36Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261030-nydus-pf64/e3.applied.yaml
```
Log: [logs/012-e3-apply.log](logs/012-e3-apply.log)

### 013 e3-recreate-pools (exit 0, 2026-10-30T19:21:39Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/013-e3-recreate-pools.log](logs/013-e3-recreate-pools.log)

### 014 bake-pf (exit 0, 2026-10-30T19:26:21Z)
```
$ benchmarking/image-plugin/run.sh bake nydus
```
Log: [logs/014-bake-pf.log](logs/014-bake-pf.log)

### 015 e3-apply (exit 0, 2026-10-30T19:29:13Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261030-nydus-pf64/e3.applied.yaml
```
Log: [logs/015-e3-apply.log](logs/015-e3-apply.log)

### 016 e3-recreate-pools (exit 0, 2026-10-30T19:29:16Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/016-e3-recreate-pools.log](logs/016-e3-recreate-pools.log)

### 017 e3-wait-ready (exit 0, 2026-10-30T19:32:38Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/017-e3-wait-ready.log](logs/017-e3-wait-ready.log)

### 018 e2-apply (exit 0, 2026-10-30T19:38:18Z)
```
$ kubectl apply -f benchmarking/image-plugin/results/20261030-nydus-pf64/e2.applied.yaml
```
Log: [logs/018-e2-apply.log](logs/018-e2-apply.log)

### 019 e2-recreate-pools (exit 0, 2026-10-30T19:38:21Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/019-e2-recreate-pools.log](logs/019-e2-recreate-pools.log)

### 020 e2-wait-ready (exit 0, 2026-10-30T19:41:52Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/020-e2-wait-ready.log](logs/020-e2-wait-ready.log)

