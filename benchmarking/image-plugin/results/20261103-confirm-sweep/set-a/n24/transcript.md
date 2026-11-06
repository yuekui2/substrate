### 001 n24-delete-rest (exit 0, 2026-11-03T12:53:01Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh delete pull riptide-ms spegel
```
Log: [logs/001-n24-delete-rest.log](logs/001-n24-delete-rest.log)

### 002 n24-nydus-recreate-pools (exit 0, 2026-11-03T12:55:20Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/002-n24-nydus-recreate-pools.log](logs/002-n24-nydus-recreate-pools.log)

### 003 n24-nydus-wait-ready (exit 0, 2026-11-03T12:56:35Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/003-n24-nydus-wait-ready.log](logs/003-n24-nydus-wait-ready.log)

### 004 n24-nydus-run (exit 0, 2026-11-03T12:57:44Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/004-n24-nydus-run.log](logs/004-n24-nydus-run.log)

### 005 n24-delete-nydus (exit 0, 2026-11-03T13:02:57Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh delete nydus
```
Log: [logs/005-n24-delete-nydus.log](logs/005-n24-delete-nydus.log)

### 006 n24-rest-recreate-pools (exit 0, 2026-11-03T13:04:56Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate pull riptide-ms spegel
```
Log: [logs/006-n24-rest-recreate-pools.log](logs/006-n24-rest-recreate-pools.log)

### 007 n24-rest-wait-ready (exit 0, 2026-11-03T13:06:17Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh pull riptide-ms spegel
```
Log: [logs/007-n24-rest-wait-ready.log](logs/007-n24-rest-wait-ready.log)

### 008 n24-rest-valkey-flush (exit 0, 2026-11-03T13:07:09Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/valkey-flush.sh
```
Log: [logs/008-n24-rest-valkey-flush.log](logs/008-n24-rest-valkey-flush.log)

### 009 n24-rest-run (exit 0, 2026-11-03T13:07:17Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 pull riptide-ms spegel
```
Log: [logs/009-n24-rest-run.log](logs/009-n24-rest-run.log)

