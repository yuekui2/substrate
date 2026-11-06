### 001 n16-delete-rest (exit 0, 2026-11-03T12:35:36Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh delete pull riptide-ms spegel
```
Log: [logs/001-n16-delete-rest.log](logs/001-n16-delete-rest.log)

### 002 n16-nydus-recreate-pools (exit 0, 2026-11-03T12:35:39Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/002-n16-nydus-recreate-pools.log](logs/002-n16-nydus-recreate-pools.log)

### 003 n16-nydus-wait-ready (exit 0, 2026-11-03T12:38:51Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/003-n16-nydus-wait-ready.log](logs/003-n16-nydus-wait-ready.log)

### 004 n16-nydus-run (exit 0, 2026-11-03T12:39:49Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/004-n16-nydus-run.log](logs/004-n16-nydus-run.log)

### 005 n16-delete-nydus (exit 0, 2026-11-03T12:43:35Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh delete nydus
```
Log: [logs/005-n16-delete-nydus.log](logs/005-n16-delete-nydus.log)

### 006 n16-rest-recreate-pools (exit 0, 2026-11-03T12:45:44Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate pull riptide-ms spegel
```
Log: [logs/006-n16-rest-recreate-pools.log](logs/006-n16-rest-recreate-pools.log)

### 007 n16-rest-wait-ready (exit 0, 2026-11-03T12:47:05Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh pull riptide-ms spegel
```
Log: [logs/007-n16-rest-wait-ready.log](logs/007-n16-rest-wait-ready.log)

### 008 n16-rest-valkey-flush (exit 0, 2026-11-03T12:48:01Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/valkey-flush.sh
```
Log: [logs/008-n16-rest-valkey-flush.log](logs/008-n16-rest-valkey-flush.log)

### 009 n16-rest-run (exit 0, 2026-11-03T12:48:12Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 pull riptide-ms spegel
```
Log: [logs/009-n16-rest-run.log](logs/009-n16-rest-run.log)

