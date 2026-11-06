### 001 deploy-nydus (exit 0, 2026-11-03T05:44:36Z)
```
$ benchmarking/image-plugin/deploy.sh nydus
```
Log: [logs/001-deploy-nydus.log](logs/001-deploy-nydus.log)

### 002 deploy-nydus-best (exit 0, 2026-11-03T05:45:59Z)
```
$ benchmarking/image-plugin/deploy-nydus-best.sh us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/nydusd@sha256:eeb616c982a51e9d3329d99cd6c61de9d62dbbc0a8c61cec03df34644faea0c5
```
Log: [logs/002-deploy-nydus-best.log](logs/002-deploy-nydus-best.log)

### 003 wait-nydus (exit 0, 2026-11-03T05:46:16Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/003-wait-nydus.log](logs/003-wait-nydus.log)

### 004 seed-preload-b (exit 0, 2026-11-03T05:46:41Z)
```
$ SAVE_DIR=/var/lib/ate/nydus-seed-blobs benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-96-pf-b.jsonl 96
```
Log: [logs/004-seed-preload-b.log](logs/004-seed-preload-b.log)

### 005 seed-preload-c (exit 0, 2026-11-03T05:51:03Z)
```
$ SAVE_DIR=/var/lib/ate/nydus-seed-blobs benchmarking/image-plugin/seed-preload.sh benchmarking/image-plugin/images-96-pf-c.jsonl 96
```
Log: [logs/005-seed-preload-c.log](logs/005-seed-preload-c.log)

### 006 bake-nydus-pfB (exit 1, 2026-11-03T05:55:30Z)
```
$ DENSITY=2 TAG=-pfB LOADER_ARGS='-images /config/images-96-pf-b.jsonl -actors 96 -concurrency 48' benchmarking/image-plugin/run.sh bake nydus
```
Log: [logs/006-bake-nydus-pfB.log](logs/006-bake-nydus-pfB.log)

### 007 bake-nydus-pfb (exit 0, 2026-11-03T06:00:13Z)
```
$ DENSITY=2 TAG=-pfb LOADER_ARGS='-images /config/images-96-pf-b.jsonl -actors 96 -concurrency 48' benchmarking/image-plugin/run.sh bake nydus
```
Log: [logs/007-bake-nydus-pfb.log](logs/007-bake-nydus-pfb.log)

### 008 bake-nydus-pfc (exit 0, 2026-11-03T06:02:55Z)
```
$ DENSITY=2 TAG=-pfc LOADER_ARGS='-images /config/images-96-pf-c.jsonl -actors 96 -concurrency 48' benchmarking/image-plugin/run.sh bake nydus
```
Log: [logs/008-bake-nydus-pfc.log](logs/008-bake-nydus-pfc.log)

### 009 delete-nydus (exit 0, 2026-11-03T06:05:30Z)
```
$ benchmarking/image-plugin/pools.sh delete nydus
```
Log: [logs/009-delete-nydus.log](logs/009-delete-nydus.log)

### 010 create-rest (exit 0, 2026-11-03T06:07:34Z)
```
$ benchmarking/image-plugin/pools.sh create pull riptide-ms spegel
```
Log: [logs/010-create-rest.log](logs/010-create-rest.log)

### 011 deploy-pull-riptide-ms (exit 0, 2026-11-03T06:08:58Z)
```
$ benchmarking/image-plugin/deploy.sh pull riptide-ms
```
Log: [logs/011-deploy-pull-riptide-ms.log](logs/011-deploy-pull-riptide-ms.log)

### 012 deploy-spegel-preload (exit 0, 2026-11-03T06:09:57Z)
```
$ KO=$(cd hack/tools/ko && go tool -n ko) && ${KO} resolve --base-import-paths -f image-plugin/manifests/spegel-preload.yaml | python3 benchmarking/image-plugin/scope_plugin.py spegel | kubectl apply -f -
```
Log: [logs/012-deploy-spegel-preload.log](logs/012-deploy-spegel-preload.log)

### 013 wait-rest (exit 0, 2026-11-03T06:10:09Z)
```
$ benchmarking/image-plugin/wait-ready.sh pull riptide-ms spegel
```
Log: [logs/013-wait-rest.log](logs/013-wait-rest.log)

### 014 wait-spegel-seed (exit 0, 2026-11-03T06:10:26Z)
```
$ kubectl -n ate-system rollout status ds/image-plugin-spegel-seed --timeout=20m
```
Log: [logs/014-wait-spegel-seed.log](logs/014-wait-spegel-seed.log)

### 015 spegel-seed-preload (exit 0, 2026-11-03T06:10:27Z)
```
$ benchmarking/image-plugin/spegel-seed-preload.sh benchmarking/image-plugin/images-320.jsonl 288
```
Log: [logs/015-spegel-seed-preload.log](logs/015-spegel-seed-preload.log)

### 016 bake-rest-pf (exit 0, 2026-11-03T06:20:47Z)
```
$ DENSITY=2 TAG=-pf LOADER_ARGS='-images /config/images-96-pf.jsonl -actors 96 -concurrency 48' benchmarking/image-plugin/run.sh bake pull riptide-ms spegel
```
Log: [logs/016-bake-rest-pf.log](logs/016-bake-rest-pf.log)

### 017 bake-rest-pfb (exit 0, 2026-11-03T06:31:19Z)
```
$ DENSITY=2 TAG=-pfb LOADER_ARGS='-images /config/images-96-pf-b.jsonl -actors 96 -concurrency 48' benchmarking/image-plugin/run.sh bake pull riptide-ms spegel
```
Log: [logs/017-bake-rest-pfb.log](logs/017-bake-rest-pfb.log)

### 018 bake-rest-pfc (exit 0, 2026-11-03T06:37:43Z)
```
$ DENSITY=2 TAG=-pfc LOADER_ARGS='-images /config/images-96-pf-c.jsonl -actors 96 -concurrency 48' benchmarking/image-plugin/run.sh bake pull riptide-ms spegel
```
Log: [logs/018-bake-rest-pfc.log](logs/018-bake-rest-pfc.log)

