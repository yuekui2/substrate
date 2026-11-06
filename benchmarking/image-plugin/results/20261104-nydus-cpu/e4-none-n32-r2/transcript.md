### 001 deploy-nydus (exit 0, 2026-11-04T05:03:15Z)
```
$ benchmarking/image-plugin/deploy-nydus-best.sh us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/nydusd@sha256:eeb616c982a51e9d3329d99cd6c61de9d62dbbc0a8c61cec03df34644faea0c5
```
Log: [logs/001-deploy-nydus.log](logs/001-deploy-nydus.log)

### 002 recreate-pools (exit 0, 2026-11-04T05:03:30Z)
```
$ benchmarking/image-plugin/pools.sh recreate nydus
```
Log: [logs/002-recreate-pools.log](logs/002-recreate-pools.log)

### 003 wait-ready (exit 0, 2026-11-04T05:06:41Z)
```
$ benchmarking/image-plugin/wait-ready.sh nydus
```
Log: [logs/003-wait-ready.log](logs/003-wait-ready.log)

### 004 run (exit 0, 2026-11-04T05:07:38Z)
```
$ benchmarking/image-plugin/run.sh rep 1 nydus
```
Log: [logs/004-run.log](logs/004-run.log)

