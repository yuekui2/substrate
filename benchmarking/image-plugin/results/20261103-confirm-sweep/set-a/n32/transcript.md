### 001 n32-delete-rest (exit 0, 2026-11-03T13:12:10Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh delete pull riptide-ms spegel
```
Log: [logs/001-n32-delete-rest.log](logs/001-n32-delete-rest.log)

### 002 n32-nydus-recreate-pools (exit 0, 2026-11-03T13:14:34Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/002-n32-nydus-recreate-pools.log](logs/002-n32-nydus-recreate-pools.log)

### 003 n32-nydus-wait-ready (exit 0, 2026-11-03T13:15:50Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/003-n32-nydus-wait-ready.log](logs/003-n32-nydus-wait-ready.log)

### 004 n32-nydus-run (exit 0, 2026-11-03T13:16:53Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/004-n32-nydus-run.log](logs/004-n32-nydus-run.log)

### 005 n32-delete-nydus (exit 0, 2026-11-03T13:20:43Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh delete nydus
```
Log: [logs/005-n32-delete-nydus.log](logs/005-n32-delete-nydus.log)

### 006 n32-rest-recreate-pools (exit 0, 2026-11-03T13:22:46Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate pull riptide-ms spegel
```
Log: [logs/006-n32-rest-recreate-pools.log](logs/006-n32-rest-recreate-pools.log)

### 007 n32-rest-wait-ready (exit 0, 2026-11-03T13:24:12Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh pull riptide-ms spegel
```
Log: [logs/007-n32-rest-wait-ready.log](logs/007-n32-rest-wait-ready.log)

### 008 n32-rest-valkey-flush (exit 0, 2026-11-03T13:24:54Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/valkey-flush.sh
```
Log: [logs/008-n32-rest-valkey-flush.log](logs/008-n32-rest-valkey-flush.log)

### 009 n32-rest-run (exit 0, 2026-11-03T13:25:00Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 pull riptide-ms spegel
```
Log: [logs/009-n32-rest-run.log](logs/009-n32-rest-run.log)

