### 001 n32-delete-rest (exit 0, 2026-11-03T14:13:22Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh delete pull riptide-ms spegel
```
Log: [logs/001-n32-delete-rest.log](logs/001-n32-delete-rest.log)

### 002 n32-nydus-recreate-pools (exit 0, 2026-11-03T14:15:41Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/002-n32-nydus-recreate-pools.log](logs/002-n32-nydus-recreate-pools.log)

### 003 n32-nydus-wait-ready (exit 0, 2026-11-03T14:16:57Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/003-n32-nydus-wait-ready.log](logs/003-n32-nydus-wait-ready.log)

### 004 n32-nydus-run (exit 0, 2026-11-03T14:18:03Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/004-n32-nydus-run.log](logs/004-n32-nydus-run.log)

### 005 n32-delete-nydus (exit 0, 2026-11-03T14:21:57Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh delete nydus
```
Log: [logs/005-n32-delete-nydus.log](logs/005-n32-delete-nydus.log)

### 006 n32-rest-recreate-pools (exit 0, 2026-11-03T14:24:01Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate pull riptide-ms spegel
```
Log: [logs/006-n32-rest-recreate-pools.log](logs/006-n32-rest-recreate-pools.log)

### 007 n32-rest-wait-ready (exit 0, 2026-11-03T14:25:43Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh pull riptide-ms spegel
```
Log: [logs/007-n32-rest-wait-ready.log](logs/007-n32-rest-wait-ready.log)

### 008 n32-rest-valkey-flush (exit 0, 2026-11-03T14:26:43Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/valkey-flush.sh
```
Log: [logs/008-n32-rest-valkey-flush.log](logs/008-n32-rest-valkey-flush.log)

### 009 n32-rest-run (exit 0, 2026-11-03T14:26:52Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 pull riptide-ms spegel
```
Log: [logs/009-n32-rest-run.log](logs/009-n32-rest-run.log)

