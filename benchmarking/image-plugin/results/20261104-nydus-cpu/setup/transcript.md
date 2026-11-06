### 001 deploy (exit 0, 2026-11-04T02:47:52Z)
```
$ benchmarking/image-plugin/deploy.sh nydus
```
Log: [logs/001-deploy.log](logs/001-deploy.log)

### 002 deploy-nydus (exit 0, 2026-11-04T02:49:27Z)
```
$ benchmarking/image-plugin/deploy-nydus-best.sh us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/nydusd@sha256:eeb616c982a51e9d3329d99cd6c61de9d62dbbc0a8c61cec03df34644faea0c5
```
Log: [logs/002-deploy-nydus.log](logs/002-deploy-nydus.log)

### 003 wait-ready (exit 1, 2026-11-04T02:49:59Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/003-wait-ready.log](logs/003-wait-ready.log)

### 004 bake (exit 0, 2026-11-04T02:51:05Z)
```
$ benchmarking/image-plugin/run.sh bake nydus
```
Log: [logs/004-bake.log](logs/004-bake.log)

### 005 preload-none (exit 1, 2026-11-04T03:35:40Z)
```
$ benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-96-none.jsonl 96
```
Log: [logs/005-preload-none.log](logs/005-preload-none.log)

### 006 preload-none (exit 1, 2026-11-04T03:44:07Z)
```
$ benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-96-none.jsonl 96
```
Log: [logs/006-preload-none.log](logs/006-preload-none.log)

### 007 preload-none (exit 0, 2026-11-04T03:48:07Z)
```
$ benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-96-none.jsonl 96
```
Log: [logs/007-preload-none.log](logs/007-preload-none.log)

### 008 bake-none (exit 0, 2026-11-04T04:02:08Z)
```
$ benchmarking/image-plugin/run.sh bake nydus
```
Log: [logs/008-bake-none.log](logs/008-bake-none.log)

