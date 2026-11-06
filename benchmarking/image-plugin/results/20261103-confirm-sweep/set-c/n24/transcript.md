### 001 n24-delete-rest (exit 0, 2026-11-03T14:51:36Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh delete pull riptide-ms spegel
```
Log: [logs/001-n24-delete-rest.log](logs/001-n24-delete-rest.log)

### 002 n24-nydus-recreate-pools (exit 0, 2026-11-03T14:53:49Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/002-n24-nydus-recreate-pools.log](logs/002-n24-nydus-recreate-pools.log)

### 003 n24-nydus-wait-ready (exit 0, 2026-11-03T14:56:52Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/003-n24-nydus-wait-ready.log](logs/003-n24-nydus-wait-ready.log)

### 004 n24-nydus-run (exit 0, 2026-11-03T14:57:52Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/004-n24-nydus-run.log](logs/004-n24-nydus-run.log)

### 005 n24-delete-nydus (exit 0, 2026-11-03T15:02:07Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh delete nydus
```
Log: [logs/005-n24-delete-nydus.log](logs/005-n24-delete-nydus.log)

### 006 n24-rest-recreate-pools (exit 0, 2026-11-03T15:04:10Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/pools.sh recreate pull riptide-ms spegel
```
Log: [logs/006-n24-rest-recreate-pools.log](logs/006-n24-rest-recreate-pools.log)

### 007 n24-rest-wait-ready (exit 0, 2026-11-03T15:05:37Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/wait-ready.sh pull riptide-ms spegel
```
Log: [logs/007-n24-rest-wait-ready.log](logs/007-n24-rest-wait-ready.log)

### 008 n24-rest-valkey-flush (exit 0, 2026-11-03T15:06:51Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/valkey-flush.sh
```
Log: [logs/008-n24-rest-valkey-flush.log](logs/008-n24-rest-valkey-flush.log)

### 009 n24-rest-run (exit 0, 2026-11-03T15:07:00Z)
```
$ /usr/local/google/home/kuiyue/src/substrate/benchmarking/image-plugin/run.sh rep 1 pull riptide-ms spegel
```
Log: [logs/009-n24-rest-run.log](logs/009-n24-rest-run.log)

