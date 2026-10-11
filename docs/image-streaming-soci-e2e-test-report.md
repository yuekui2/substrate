# AWS SOCI (`ate-ocilayer-soci`) Live GKE Cluster E2E Test Report

## 1. Test Environment

- **GCP Project / Zone:** `kuiyue-gke-dev` / `us-central1-a`
- **GKE Cluster:** `substrate-stream-test` (`v1.36.4-gke.1495000`, `COS_CONTAINERD`)
- **Adapter Under Test:** `ate-ocilayer-soci` DaemonSet ([`cmd/ocilayer-soci/ocilayer-soci.yaml`](../cmd/ocilayer-soci/ocilayer-soci.yaml), [`cmd/ocilayer-soci`](../cmd/ocilayer-soci) + `soci-snapshotter-grpc` `v0.15.0`, commit `7716bd67e813f8e80873948e0a02d0a7c5370854`)
- **Nodes Tested (switched live via `ate.dev/oci-layer-service=soci`):**
  1. **Node 1 (`no-stream-pool`):** `gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz` (GKE Image Streaming disabled, host `gcfsd.service` inactive)
  2. **Node 2 (`e2-std32-ssd-pool`):** `gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft` (GKE Image Streaming enabled, host `gcfsd.service` active)
- **Test Images in Artifact Registry (`us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo`):**
  1. **SOCI-indexed image (9 layers, ~1.19 GB compressed / ~3.5 GB unpacked):**
     `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e`
     - OCI Referrers SOCI v1 Index Digest: `sha256:b1fd22e44bb07ee1623c42e8321414ffd460221e97ac7a5d6a5dc75df951bb8a` (`application/vnd.amazon.soci.index.v1+json`, built with `AWS SOCI CLI v0.2`, 9 zTOCs)
     - Compressed layer sizes:
       - Layer 0 (`sha256:84ad5f3a...`): `30,664,193` B (~30.6 MB)
       - Layer 1 (`sha256:a17129cc...`): `287` B
       - Layer 2 (`sha256:e9a96d24...`): `272` B
       - Layer 3 (`sha256:2edc65f4...`): `348,951,060` B (~348.9 MB)
       - Layer 4 (`sha256:eff5bf58...`): `488` B
       - Layer 5 (`sha256:eee00ea2...`): `167` B
       - Layer 6 (`sha256:6cca528c...`): `807,417,796` B (~807.4 MB)
       - Layer 7 (`sha256:4a2796d8...`): `125` B
       - Layer 8 (`sha256:4778a54a...`): `6,020,751` B (~6.02 MB)
  2. **Unindexed image (no SOCI index in GAR):**
     `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest`

---

## 2. Live Provider Hot-Swap (`riptide` -> `soci`) & DaemonSet Startup

Relabeling both worker nodes from `ate.dev/oci-layer-service=riptide` to `ate.dev/oci-layer-service=soci` automatically terminated `ate-ocilayer-riptide` and scheduled `ate-ocilayer-soci`:

```text
$ kubectl label node gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft ate.dev/oci-layer-service=soci --overwrite
node/gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz labeled
node/gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft labeled

$ kubectl get pods -n ate-system -l ate.dev/oci-layer-provider=soci -o wide
NAME                      READY   STATUS    RESTARTS   AGE   IP            NODE                                                  NOMINATED NODE   READINESS GATES
ate-ocilayer-soci-2t2z2   2/2     Running   0          26s   10.1.15.221   gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft   <none>           <none>
ate-ocilayer-soci-kbg2c   2/2     Running   0          27s   10.1.15.225   gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz   <none>           <none>
```

Startup logs of `soci-snapshotter` and `ocilayer-soci`:

```text
{"level":"info","msg":"starting soci-snapshotter-grpc","revision":"7716bd67e813f8e80873948e0a02d0a7c5370854","time":"2026-10-10T17:27:27.396399957Z","version":"v0.15.0"}
{"dropPolicy":"newest","emitMetricPeriod":10000000000,"fetchPeriod":500000000,"level":"info","maxQueueSize":300,"msg":"constructing background fetcher","silencePeriod":30000000000,"time":"2026-10-10T17:27:27.896570347Z"}
{"address":"/run/soci-snapshotter-grpc/soci-snapshotter-grpc.sock","level":"info","msg":"soci-snapshotter-grpc successfully started","time":"2026-10-10T17:27:27.901225565Z"}
---
2026/10/10 17:29:28 INFO OCILayerService adapter listening provider=soci listenSocket=/run/ate-image-streaming/ocilayer.sock snapshotterSocket=/run/soci-snapshotter-grpc/soci-snapshotter-grpc.sock
```

---

## 3. Node 1 (`no-stream-pool`, `p1tz`): Cold & Warm FUSE Streaming of All 9 Layers

### 3.1 Cold `PrepareLayers` Probe Output

```json
{
  "provider": "soci",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e",
  "imageDigest": "sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e",
  "canProvide": true,
  "layerDirs": [
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-0",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-1",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-2",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-3",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-4",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-5",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-6",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-7",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-8"
  ],
  "canProvideMs": 3.552,
  "prepareMs": 3874.117,
  "releaseMs": 0.477,
  "verifiedLayers": 9
}
```

### 3.2 Warm `PrepareLayers` Probe Output (Lease Hit)

```json
{
  "provider": "soci",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e",
  "imageDigest": "sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e",
  "canProvide": true,
  "layerDirs": [
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-0",
    "..."
  ],
  "canProvideMs": 1.694,
  "prepareMs": 0.557,
  "releaseMs": 0.268,
  "verifiedLayers": 9
}
```

### 3.3 `ocilayer-soci` and `soci-snapshotter-grpc` Logs

```text
2026/10/10 17:30:10 INFO Synchronized registry credentials to SOCI docker config path=/run/soci-snapshotter-grpc/docker/config.json image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e server=us-central1-docker.pkg.dev username=_token
2026/10/10 17:30:14 INFO PrepareLayers timing breakdown provider=soci image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e digest=sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e layers=9 resolve_ms=368.158 stat_ms=19.641 prepare_ms=3434.107 view_ms=45.998 listable_ms=3.256 wrapper_ms=1.339 total_ms=3873.046
```

`soci-snapshotter-grpc` logs showing remote FUSE preparation of all 9 layers (`snapshots/2/fs` through `snapshots/18/fs`):

```text
{"digest":"sha256:647fb25342b8eed44a0ec26e220b5192484ce3b7d5cc29c4d8dbcc54fefa45d0","key":"...-l0-prep","level":"info","mountpoint":"/var/lib/soci-snapshotter-grpc/snapshotter/snapshots/2/fs","msg":"fetching artifact from remote","parent":"","time":"2026-10-10T17:30:11.173263655Z"}
{"key":"...-l0-prep","level":"info","msg":"remote snapshot successfully prepared.","parent":"","remote-snapshot-prepared":"true","time":"2026-10-10T17:30:12.050371026Z"}
{"key":"...-l1-prep","level":"info","msg":"preparing filesystem mount at mountpoint=/var/lib/soci-snapshotter-grpc/snapshotter/snapshots/4/fs","parent":"sha256:ea16cace89338c84eb6bcb91a7efdfcae6838fff359efe951858227436486c34","time":"2026-10-10T17:30:12.077457576Z"}
{"key":"...-l1-prep","level":"info","msg":"remote snapshot successfully prepared.","parent":"sha256:ea16cace89338c84eb6bcb91a7efdfcae6838fff359efe951858227436486c34","remote-snapshot-prepared":"true","time":"2026-10-10T17:30:12.276661441Z"}
{"key":"...-l2-prep","level":"info","msg":"remote snapshot successfully prepared.","parent":"sha256:5dda86b5f741fd620228596b7d11aafada6529e6011f7f54e957e2b5636c18e9","remote-snapshot-prepared":"true","time":"2026-10-10T17:30:12.295148790Z"}
{"key":"...-l3-prep","level":"info","msg":"remote snapshot successfully prepared.","parent":"sha256:3cb8e82e2e0c0fe5e61a20fb35e1a621188305e44850f9c58dbe4e0ac4b449c2","remote-snapshot-prepared":"true","time":"2026-10-10T17:30:14.242803628Z"}
{"key":"...-l4-prep","level":"info","msg":"remote snapshot successfully prepared.","parent":"sha256:4af17c5627211dde358de00cf665cc8e5b5c1d253c97f56057a6ec9ea3d349eb","remote-snapshot-prepared":"true","time":"2026-10-10T17:30:14.256117591Z"}
{"key":"...-l5-prep","level":"info","msg":"remote snapshot successfully prepared.","parent":"sha256:6b4d0fa0650dcf70101a9df4e0a73f20ccf7343b3942e72eda85884b8ea84f13","remote-snapshot-prepared":"true","time":"2026-10-10T17:30:14.285963258Z"}
{"key":"...-l6-prep","level":"info","msg":"remote snapshot successfully prepared.","parent":"sha256:35e76a8344cfcd7992490dc23d7deacf38c2219b7cb5da4192109e03ddec4ef1","remote-snapshot-prepared":"true","time":"2026-10-10T17:30:14.316279487Z"}
{"key":"...-l7-prep","level":"info","msg":"remote snapshot successfully prepared.","parent":"sha256:f86362802e5cd5d78264a3406407f8677f5c1007920e3825faf1f9b55f6b5792","remote-snapshot-prepared":"true","time":"2026-10-10T17:30:14.329214103Z"}
{"key":"...-l8-prep","level":"info","msg":"remote snapshot successfully prepared.","parent":"sha256:7734848956b45c32d89c894d965cc31f272af4822e8554e292b20e7c67ca62ac","remote-snapshot-prepared":"true","time":"2026-10-10T17:30:14.338816644Z"}
```

### 3.4 Host FUSE Mounts (`df -T`), Wrapper Symlinks, and File Verification

```text
Filesystem     Type           1K-blocks     Used Available Use% Mounted on
soci           fuse.rawBridge  98831908 10280936  88534588  11% /var/lib/soci-snapshotter-grpc/snapshotter/snapshots/2/fs
soci           fuse.rawBridge  98831908 10280936  88534588  11% /var/lib/soci-snapshotter-grpc/snapshotter/snapshots/4/fs
soci           fuse.rawBridge  98831908 10280936  88534588  11% /var/lib/soci-snapshotter-grpc/snapshotter/snapshots/6/fs
soci           fuse.rawBridge  98831908 10280936  88534588  11% /var/lib/soci-snapshotter-grpc/snapshotter/snapshots/8/fs
soci           fuse.rawBridge  98831908 10280936  88534588  11% /var/lib/soci-snapshotter-grpc/snapshotter/snapshots/10/fs
soci           fuse.rawBridge  98831908 10280936  88534588  11% /var/lib/soci-snapshotter-grpc/snapshotter/snapshots/12/fs
soci           fuse.rawBridge  98831908 10280936  88534588  11% /var/lib/soci-snapshotter-grpc/snapshotter/snapshots/14/fs
soci           fuse.rawBridge  98831908 10280936  88534588  11% /var/lib/soci-snapshotter-grpc/snapshotter/snapshots/16/fs
soci           fuse.rawBridge  98831908 10280936  88534588  11% /var/lib/soci-snapshotter-grpc/snapshotter/snapshots/18/fs
```

Sample file read over FUSE from `layer-0/fs/etc/os-release` and directory listings across `layer-0/fs`..`layer-8/fs`:

```text
PRETTY_NAME="Ubuntu 22.04.5 LTS"
NAME="Ubuntu"
VERSION_ID="22.04"
VERSION="22.04.5 LTS (Jammy Jellyfish)"
VERSION_CODENAME=jammy

--- layer-0/fs/ ---
lrwxrwxrwx  1 root root  7 Sep  1 20:33 bin -> usr/bin
drwxr-xr-x  2 root root  0 Apr 18  2022 boot
drwxr-xr-x  2 root root  0 Sep  1 20:42 dev
drwxr-xr-x 32 root root  0 Sep  1 20:42 etc
drwxr-xr-x  2 root root  0 Apr 18  2022 home
--- layer-1/fs/ ---
-rw-r--r-- 1 root root 272 Sep 24 18:27 requirements.txt
--- layer-2/fs/ ---
-rw-r--r-- 1 root root 244 Sep 24 18:27 requirements_py2.txt
--- layer-3/fs/ ---
drwxr-xr-x  3 root root 0 Sep 24 18:29 builder
drwxr-xr-x 45 root root 0 Sep 24 18:29 etc
drwx------  4 root root 0 Sep 24 18:30 root
drwxr-xr-x  7 root root 0 Sep 24 18:29 run
drwxr-xr-x 11 root root 0 Sep  1 20:33 usr
--- layer-8/fs/ ---
dr-xr-xr-x 2 root root 0 Jan  1  1970 ko-app
```

---

## 4. Node 2 (`e2-std32-ssd-pool`, `bpft`): Cold & Warm FUSE Streaming

On `gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft` (`ate-ocilayer-soci-mdsk2`), where host `gcfsd.service` is also active:

### 4.1 Cold & Warm Probe Outputs

```json
{
  "provider": "soci",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e",
  "imageDigest": "sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e",
  "canProvide": true,
  "layerDirs": [
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-0",
    "..."
  ],
  "canProvideMs": 5.273,
  "prepareMs": 6537.029,
  "releaseMs": 0.646,
  "verifiedLayers": 9
}
```

Warm lease hit on `bpft`:

```json
{
  "provider": "soci",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e",
  "imageDigest": "sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e",
  "canProvide": true,
  "canProvideMs": 3.019,
  "prepareMs": 0.899,
  "releaseMs": 0.539,
  "verifiedLayers": 9
}
```

### 4.2 `soci-snapshotter-grpc` OCI Referrers Discovery Logs on `bpft`

```text
{"digest":"sha256:b1fd22e44bb07ee1623c42e8321414ffd460221e97ac7a5d6a5dc75df951bb8a","key":"...-l0-prep","level":"info","mountpoint":"/var/lib/soci-snapshotter-grpc/snapshotter/snapshots/1/fs","msg":"fetching SOCI artifacts using index descriptor","parent":"","time":"2026-10-10T17:35:06.002276328Z"}
{"digest":"sha256:b1fd22e44bb07ee1623c42e8321414ffd460221e97ac7a5d6a5dc75df951bb8a","key":"...-l0-prep","level":"info","mountpoint":"/var/lib/soci-snapshotter-grpc/snapshotter/snapshots/1/fs","msg":"fetching SOCI index from remote registry","parent":"","time":"2026-10-10T17:35:06.002338408Z"}
2026/10/10 17:35:11 INFO PrepareLayers timing breakdown provider=soci image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e digest=sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e layers=9 resolve_ms=263.529 stat_ms=12.866 prepare_ms=6158.684 view_ms=90.264 listable_ms=7.742 wrapper_ms=2.001 total_ms=6535.938
```

---

## 5. Small-Layer Local Materialization, Unindexed Image Decline (`ErrNotSupported`), & `declineTTL` Cache

Testing against `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest` (which has no SOCI index in Artifact Registry; Layer 0 is `< 10 MiB` and Layer 1 is `52.4 MB`):

1. **First call (`PrepareLayers`):**
   - On Layer 0 (`sha256:55afa1ecc2...`, `< 10 MiB`), `soci-snapshotter-grpc` returns `codes.OK` with a local writable snapshot directory (`/var/lib/soci-snapshotter-grpc/snapshotter/snapshots/20/fs`), and `ocilayer-soci`'s `materializeSmallLayer` downloads, unpacks, and commits Layer 0 locally via `Snapshots.Commit`.
   - On Layer 1 (`sha256:ae32ebc759...`, `52,444,975` B compressed), `soci-snapshotter-grpc` finds no SOCI index (`no valid SOCI index found; deferring to container runtime`) and returns `codes.OK`. `ocilayer-soci`'s `--max-local-layer-size` guard (`10 MiB`) detects that Layer 1 exceeds the small-layer threshold, rolls back the committed Layer 0 snapshot (`cleanupOnErr`), and immediately returns `FailedPrecondition` (`ocilayer.ErrNotSupported`) without downloading the 52.4 MB unindexed blob:

```text
2026/10/10 17:40:21 INFO Synchronized registry credentials to SOCI docker config path=/run/soci-snapshotter-grpc/docker/config.json image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a@sha256:256545b7d3cda8db8d578e2bd70d4396d3f2d16c1bcd3fa549f8350f7b1daf9d server=us-central1-docker.pkg.dev username=_token
2026/10/10 17:40:22 INFO Materialized small SOCI layer locally and committed snapshot layerIndex=0 layerDigest=sha256:55afa1ecc21d2bb5e5045f32dafee56272ffd89860bac26f6c32123439af26a4 chainID=sha256:34884abbe92863fce933ed7c39c0e045631af0ed86d5cc0dfbdf9fdca426ce3c targetDir=/var/lib/soci-snapshotter-grpc/snapshotter/snapshots/20/fs

PrepareLayers error: image not supported by OCILayerService: layer 1 (sha256:1ab2c1381e7569b513169c0bf4195dcfc3d4834d315b1e9cd16848d141b45f2b): fetching soci small layer 1 (sha256:ae32ebc75941d3d769b37b60880950e1d289670ef921b0fc13599e4e646d532c): layer 1 (sha256:ae32ebc75941d3d769b37b60880950e1d289670ef921b0fc13599e4e646d532c) compressed size 52444975 exceeds max local layer size 10485760 (image not indexed by soci): image not supported by OCILayerService
```

2. **Second call (`CanProvide` negative cache hit):** Because `ocilayer-soci` records declined images for `--decline-ttl=10m`, subsequent `CanProvide` calls return `false` in `1.979 ms` so `atelet` routes directly to `internal/imagecache`:

```json
{
  "provider": "soci",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest",
  "canProvide": false,
  "canProvideMs": 1.979,
  "prepareMs": 0,
  "releaseMs": 0,
  "verifiedLayers": 0
}
```

---

## 6. `ReleaseLayers (-bypass-ttl=true)` Immediate Teardown Verification

Running `e2e-stream-probe -bypass-ttl=true` removes all 9 `Snapshots.View` mounts and deletes the wrapper directory under `/var/lib/ateom-gvisor/streaming/soci`:

```json
{
  "provider": "soci",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e",
  "imageDigest": "sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e",
  "canProvide": true,
  "canProvideMs": 1.882,
  "prepareMs": 0.579,
  "releaseMs": 35.424,
  "verifiedLayers": 9
}
```

Directory verification after teardown:

```text
$ ls -la /var/lib/ateom-gvisor/streaming/soci/
total 8
drwxr-xr-x 2 root root 4096 Oct 10 17:32 .
drwxr-xr-x 4 root root 4096 Oct 10 17:30 ..
```
