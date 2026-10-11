# Riptide v2 `OCILayerService` Adapter — Live GKE Cluster E2E Test Report

## Test Environment
* **GCP Project / Zone / Cluster:** `kuiyue-gke-dev` / `us-central1-a` / `substrate-stream-test` (`v1.36.4-gke.1495000`, `COS_CONTAINERD`)
* **Test Image:** `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest`
  * Digest: `sha256:256545b7d3cda8db8d578e2bd70d4396d3f2d16c1bcd3fa549f8350f7b1daf9d`
  * 3 OCI Layers (`chainID`s):
    * Layer 0 (`34884abbe92863fce933ed7c39c0e045631af0ed86d5cc0dfbdf9fdca426ce3c`): `bin dev etc home lib ...`
    * Layer 1 (`1ab2c1381e7569b513169c0bf4195dcfc3d4834d315b1e9cd16848d141b45f2b`): `largefile`
    * Layer 2 (`23daa0df5f572dea1d9838e93f3011d7a50160750d2af5db2d05e293cb8115a5`): `app-a etc`
* **Nodes Under Test:**
  1. **Case 1 (GKE Image Streaming Disabled):** Node `gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz` in node pool `no-stream-pool` (`gcfsConfig: {}`)
  2. **Case 2 (GKE Image Streaming Enabled):** Node `gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft` in node pool `e2-std32-ssd-pool` (`gcfsConfig: {enabled: true}`)

---

## Why Riptide v2 (`--enable-v2`) Requires an Isolated `gcfsd-v2` + `containerd-gcfs-grpc` Pair in Both Modes (`cmd/ocilayer-riptide/ocilayer-riptide.yaml`)

In Riptide v2 (`--enable-v2`), `containerd-gcfs-grpc` delegates `CreateView` and `DeleteView` over `/run/gcfsd/grpc.sock` using the **unscoped layer `chainID` hex** (`views/<chainID>/fs`) as the view key. Sharing the host's `gcfsd-v2` causes `CreateView` collisions (`already exists: view "<chainID>" already exists`) for any layer already mounted by host `containerd`, and causes host `containerd` GC (`DeleteView(<chainID>)`) to unmount `views/<chainID>/fs` out from under Substrate.

`riptide-provider` therefore enters a child mount namespace (`unshare -m --propagation unchanged`), marks `/run` as `rslave`, bind-mounts `/run/ate-image-streaming/gcfsd` onto `/run/gcfsd` (satisfying `gcfsd-v2`'s hardcoded `/run/gcfsd/keychain.sock` path without touching the host's `/run/gcfsd`), and mounts FUSE at `/var/lib/substrate/gcfs/mnt` (`Bidirectional` propagation). This works identically on both **Case 1 (streaming disabled, `no-stream-pool`)** and **Case 2 (streaming enabled, `e2-std32-ssd-pool`)** with zero interference.

---

## Summary of Results

| Metric / Verification Check | Case 1: Streaming Disabled (`no-stream-pool`) | Case 2: Streaming Enabled (`e2-std32-ssd-pool`) |
| :--- | :--- | :--- |
| **Host `gcfsd.service` / `gcfs-snapshotter.service`** | `inactive` / `inactive` | `active` / `active` |
| **Host `containerd` Default Snapshotter** | `overlayfs` | `gcfs` (`/run/containerd-gcfs-grpc/containerd-gcfs-grpc.sock`) |
| **Substrate Riptide FUSE Mount** | `fuse.gcfsd` on `/var/lib/substrate/gcfs/mnt` | `fuse.gcfsd` on `/var/lib/substrate/gcfs/mnt` (alongside host `/run/gcfsd/mnt`) |
| **Cold `CanProvide` Latency** | `24.905 ms` (`true`) | `28.907 ms` (`true`) |
| **Cold `PrepareLayers` Latency (3 layers)** | `543.235 ms` (`resolve=268.9ms`, `prepare=242.0ms`, `view=24.2ms`) | `612.162 ms` (`resolve=281.1ms`, `prepare=304.2ms`, `view=12.9ms`) |
| **Warm `CanProvide` Latency** | `1.685 ms` (`true`) | `2.295 ms` (`true`) |
| **Warm `PrepareLayers` Latency (3 layers)** | **`0.491 ms`** | **`0.737 ms`** |
| **Host `containerd` Pull + Synchronous GC (`crictl rmi` + `ctr images rm --sync`)** | **0 interference** (6/6 Substrate snapshots & 3/3 FUSE views intact) | **0 interference** (host `gcfs` snapshots `313 -> 314 -> 313`; Substrate `6 -> 6` intact) |
| **`ReleaseLayers (-bypass-ttl=true)` Teardown** | `6.199 ms` (`0` active views remaining) | `12.424 ms` (`0` active views remaining) |

---

## Key Commands Used

```bash
# 1. Inspect node pool GCFS configuration
gcloud container node-pools describe no-stream-pool \
  --cluster=substrate-stream-test --zone=us-central1-a --project=kuiyue-gke-dev \
  --format="yaml(name,config.imageType,config.gcfsConfig)"

gcloud container node-pools describe e2-std32-ssd-pool \
  --cluster=substrate-stream-test --zone=us-central1-a --project=kuiyue-gke-dev \
  --format="yaml(name,config.imageType,config.gcfsConfig)"

# 2. Label target node(s) and deploy the Riptide OCILayerService Adapter DaemonSet
kubectl label node <NODE_NAME> ate.dev/oci-layer-service=riptide --overwrite
kubectl apply -f cmd/ocilayer-riptide/ocilayer-riptide.yaml

# 3. Probe Cold & Warm PrepareLayers (retaining lease in 10m idle TTL for GC test)
/home/kubernetes/bin/e2e-stream-probe \
  -socket=/run/ate-image-streaming/ocilayer.sock \
  -image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest \
  -bypass-ttl=false

# 4. Host containerd Non-Interference Test (on the node in host namespaces)
crictl pull --creds "oauth2accesstoken:${TOKEN}" \
  us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest
crictl rmi us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest
ctr -n k8s.io images rm --sync \
  us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest

# 5. Immediate ReleaseLayers teardown (bypassing idle TTL)
/home/kubernetes/bin/e2e-stream-probe \
  -socket=/run/ate-image-streaming/ocilayer.sock \
  -image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest \
  -bypass-ttl=true
```

---

## Case 1 Evidence: Cluster / Node Pool with GKE Image Streaming DISABLED

### 1.1 Node Pool Config & Host Systemd Status (`gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz`)
```yaml
config:
  gcfsConfig: {}
  imageType: COS_CONTAINERD
name: no-stream-pool
```
```text
gcfsd.service: inactive
gcfs-snapshotter.service: inactive
containerd.service: active
      snapshotter = 'overlayfs'

# Running processes on node p1tz:
root 77600 77554 /home/kubernetes/bin/containerd-gcfs-grpc --log-level=info --config=/var/lib/substrate/gcfs/config.toml --address=/run/ate-image-streaming/gcfs-snapshotter.sock --root=/var/lib/substrate/gcfs --gcfs-socket-path=/run/gcfsd/grpc.sock --enable-kubernetes-event-reporting=false --disable-duplicate-layer-support=false --enable-metric-exporter=false --enable-v2
root 77601 77554 /home/kubernetes/bin/ocilayer-riptide --listen-socket=/run/ate-image-streaming/ocilayer.sock --snapshotter-socket=/run/ate-image-streaming/gcfs-snapshotter.sock --keychain-socket=/run/ate-image-streaming/gcfsd/keychain.sock --work-dir=/var/lib/ateom-gvisor/streaming/riptide --idle-ttl=10m
root 77667 77600 /home/kubernetes/bin/gcfsd-v2 --mount_point=/var/lib/substrate/gcfs/mnt --control_server_socket_path=/run/gcfsd/grpc.sock --downloader_uds_path=/run/gcfsd/downloader.sock --sentinel_file_path=/var/lib/substrate/gcfs/gcfsd_sentinel.txt --containerd_uds_path= --event_reporter_type=none --port=11254 --layer_cache_dir=/var/lib/substrate/gcfs/snapshotter/layers --images_in_use_db_path=/var/lib/substrate/gcfs/gcfsd/images_in_use.db --enable_pull_secret_keychain --client_name=GKE --enable_single_flighting=true --read_ahead_max_blocks=10 --read_ahead_cache_size_ratio=0.1
root 77703 77667 /home/kubernetes/bin/gcfsd-v2 --downloader --downloader_uds_path=/run/gcfsd/downloader.sock --service_account=
```

### 1.2 `ate-ocilayer-riptide` Startup Logs (`ate-ocilayer-riptide-jglnd`)
```text
[riptide-provider]:
No active host gcfsd.service detected (Mode 2: GKE Image Streaming disabled); starting isolated Substrate Riptide stack
time="2026-10-09T23:14:21.529502988Z" level=info msg="Emitting Riptide connection metric: scheme=\"google-c2p\", status=\"OK\""
time="2026-10-09T23:14:21.529519798Z" level=info msg="Successfully created a Container File System client with addr=\"google-c2p:///us-central1-containerfilesystem-directpath.googleapis.com\", layer_cache_dir=\"/var/lib/substrate/gcfs/snapshotter/layers\""
time="2026-10-09T23:14:21.530082532Z" level=info msg="Starting keychain server on /run/gcfsd/keychain.sock"
time="2026-10-09T23:14:21.531772617Z" level=info msg="Starting control plane server on /run/gcfsd/grpc.sock" module=controlserver
time="2026-10-09T23:14:21.532018879Z" level=info msg="GCFSD: mounting with mount_point=\"/var/lib/substrate/gcfs/mnt\", maxReadAhead=1048576 (<=0 for default), asyncRead=true"
time="2026-10-09T23:14:21.534987625Z" level=info msg="GCFSD: serving"
time="2026-10-09T23:14:21.630516107Z" level=info msg="Starting GCFS snapshotter dev-e4a08049b38d157e0af92126a4bd0436d914c74e"

[ocilayer-riptide]:
2026/10/09 23:14:20 INFO OCILayerService adapter listening provider=riptide listenSocket=/run/ate-image-streaming/ocilayer.sock snapshotterSocket=/run/ate-image-streaming/gcfs-snapshotter.sock
2026/10/09 23:14:29 INFO Pushed credentials to Riptide gcfsd keychain socket=/run/ate-image-streaming/gcfsd/keychain.sock image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a@sha256:256545b7d3cda8db8d578e2bd70d4396d3f2d16c1bcd3fa549f8350f7b1daf9d server=us-central1-docker.pkg.dev username=_token
2026/10/09 23:14:29 INFO PrepareLayers timing breakdown provider=riptide image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest digest=sha256:256545b7d3cda8db8d578e2bd70d4396d3f2d16c1bcd3fa549f8350f7b1daf9d layers=3 resolve_ms=268.916 stat_ms=4.302 prepare_ms=242.007 view_ms=24.167 listable_ms=0.734 wrapper_ms=0.472 total_ms=542.495
```

### 1.3 Cold & Warm `PrepareLayers` Outputs
```json
// Cold PrepareLayers (-bypass-ttl=false)
{
  "provider": "riptide",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest",
  "imageDigest": "sha256:256545b7d3cda8db8d578e2bd70d4396d3f2d16c1bcd3fa549f8350f7b1daf9d",
  "canProvide": true,
  "layerDirs": [
    "/var/lib/ateom-gvisor/streaming/riptide/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_image-a_latest/layer-0",
    "/var/lib/ateom-gvisor/streaming/riptide/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_image-a_latest/layer-1",
    "/var/lib/ateom-gvisor/streaming/riptide/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_image-a_latest/layer-2"
  ],
  "canProvideMs": 24.905,
  "prepareMs": 543.235,
  "releaseMs": 0.422,
  "verifiedLayers": 3
}

// Warm PrepareLayers (-bypass-ttl=false)
{
  "provider": "riptide",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest",
  "imageDigest": "sha256:256545b7d3cda8db8d578e2bd70d4396d3f2d16c1bcd3fa549f8350f7b1daf9d",
  "canProvide": true,
  "layerDirs": [
    "/var/lib/ateom-gvisor/streaming/riptide/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_image-a_latest/layer-0",
    "/var/lib/ateom-gvisor/streaming/riptide/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_image-a_latest/layer-1",
    "/var/lib/ateom-gvisor/streaming/riptide/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_image-a_latest/layer-2"
  ],
  "canProvideMs": 1.685,
  "prepareMs": 0.491,
  "releaseMs": 0.351,
  "verifiedLayers": 3
}
```

### 1.4 Host `containerd` Non-Interference & `ReleaseLayers` Verification
```text
Substrate Riptide snapshots BEFORE host containerd pull & GC:
6
Filesystem     Type       1K-blocks  Used Available Use% Mounted on
gcfsd          fuse.gcfsd      1200   600       600  50% /var/lib/substrate/gcfs/mnt
Pulling us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest via host crictl...
Image is up to date for sha256:e1de99b9300a9cd0f42feea5d35b1a47722bec0a3bab2e5084705fd892a558c7
Removing us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest from host containerd with synchronous GC (ctr -n k8s.io images rm --sync)...
us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest
Substrate Riptide snapshots AFTER host containerd synchronous GC:
6
Verifying all streamed layer views in /var/lib/ateom-gvisor/streaming/riptide are still mounted and readable:
  /var/lib/substrate/gcfs/mnt/views/34884abbe92863fce933ed7c39c0e045631af0ed86d5cc0dfbdf9fdca426ce3c/fs -> bin dev etc home lib 
  /var/lib/substrate/gcfs/mnt/views/1ab2c1381e7569b513169c0bf4195dcfc3d4834d315b1e9cd16848d141b45f2b/fs -> largefile 
  /var/lib/substrate/gcfs/mnt/views/23daa0df5f572dea1d9838e93f3011d7a50160750d2af5db2d05e293cb8115a5/fs -> app-a etc 

--- ReleaseLayers (-bypass-ttl=true) ---
releaseMs: 6.199 ms
Remaining view directories in /var/lib/ateom-gvisor/streaming/riptide after ReleaseLayers (-bypass-ttl=true):
0
Remaining snapshots in /var/lib/substrate/gcfs/snapshotter/snapshots (only committed parent chainIDs remain, active views removed):
3
```

---

## Case 2 Evidence: Cluster / Node Pool with GKE Image Streaming ENABLED

### 2.1 Node Pool Config & Host Systemd Status (`gke-substrate-stream-e2-std32-ssd-poo-cadaf832-bpft`)
```yaml
config:
  gcfsConfig:
    enabled: true
  imageType: COS_CONTAINERD
name: e2-std32-ssd-pool
```
```text
gcfsd.service: active
gcfs-snapshotter.service: active
containerd.service: active
      snapshotter = 'gcfs'
  [proxy_plugins.gcfs]
    address = '/run/containerd-gcfs-grpc/containerd-gcfs-grpc.sock'

# Coexisting host Riptide daemons AND isolated Substrate Riptide daemons on node bpft:
root 22151     1 /home/kubernetes/bin/gcfsd-v2 --mount_point=/run/gcfsd/mnt --max_content_cache_size_mb=187 --max_large_files_cache_size_mb=187 --layer_cache_dir=/var/lib/containerd/io.containerd.snapshotter.v1.gcfs/snapshotter/layers --images_in_use_db_path=/var/lib/containerd/io.containerd.snapshotter.v1.gcfs/gcfsd/images_in_use.db --enable_pull_secret_keychain --client_name=GKE --client_version=v1.36.4-gke.1495000 --enable_single_flighting=true --read_ahead_max_blocks=10 --read_ahead_cache_size_ratio=0.1
root 22214     1 /home/kubernetes/bin/containerd-gcfs-grpc --log-level=info --config=/etc/containerd-gcfs-grpc/config.toml --enable-image-proxy-keychain-client --disable-duplicate-layer-support=false --enable-v2
root 25157 25109 /home/kubernetes/bin/containerd-gcfs-grpc --log-level=info --config=/var/lib/substrate/gcfs/config.toml --address=/run/ate-image-streaming/gcfs-snapshotter.sock --root=/var/lib/substrate/gcfs --gcfs-socket-path=/run/gcfsd/grpc.sock --enable-kubernetes-event-reporting=false --disable-duplicate-layer-support=false --enable-metric-exporter=false --enable-v2
root 25202 25109 /home/kubernetes/bin/ocilayer-riptide --listen-socket=/run/ate-image-streaming/ocilayer.sock --snapshotter-socket=/run/ate-image-streaming/gcfs-snapshotter.sock --keychain-socket=/run/ate-image-streaming/gcfsd/keychain.sock --work-dir=/var/lib/ateom-gvisor/streaming/riptide --idle-ttl=10m
root 25211 25157 /home/kubernetes/bin/gcfsd-v2 --mount_point=/var/lib/substrate/gcfs/mnt --control_server_socket_path=/run/gcfsd/grpc.sock --downloader_uds_path=/run/gcfsd/downloader.sock --sentinel_file_path=/var/lib/substrate/gcfs/gcfsd_sentinel.txt --containerd_uds_path= --event_reporter_type=none --port=11254 --layer_cache_dir=/var/lib/substrate/gcfs/snapshotter/layers --images_in_use_db_path=/var/lib/substrate/gcfs/gcfsd/images_in_use.db --enable_pull_secret_keychain --client_name=GKE --enable_single_flighting=true --read_ahead_max_blocks=10 --read_ahead_cache_size_ratio=0.1
```

### 2.2 `ate-ocilayer-riptide` Startup Logs (`ate-ocilayer-riptide-l42r6`)
```text
[riptide-provider]:
Detected active host gcfsd.service (Mode 1: GKE Image Streaming enabled); starting isolated Substrate Riptide stack
time="2026-10-09T23:36:04Z" level=info msg="Emitting Riptide connection metric: scheme=\"google-c2p\", status=\"OK\""
time="2026-10-09T23:36:04Z" level=info msg="Successfully created a Container File System client with addr=\"google-c2p:///us-central1-containerfilesystem-directpath.googleapis.com\", layer_cache_dir=\"/var/lib/substrate/gcfs/snapshotter/layers\""
time="2026-10-09T23:36:04.743276764Z" level=info msg="GCFSD: mounting with mount_point=\"/var/lib/substrate/gcfs/mnt\", maxReadAhead=1048576 (<=0 for default), asyncRead=true"
time="2026-10-09T23:36:04.745948674Z" level=info msg="GCFSD: serving"
time="2026-10-09T23:36:04.844133505Z" level=info msg="Starting GCFS snapshotter dev-e4a08049b38d157e0af92126a4bd0436d914c74e"

[ocilayer-riptide]:
2026/10/09 23:36:03 INFO OCILayerService adapter listening provider=riptide listenSocket=/run/ate-image-streaming/ocilayer.sock snapshotterSocket=/run/ate-image-streaming/gcfs-snapshotter.sock
2026/10/09 23:36:13 INFO Pushed credentials to Riptide gcfsd keychain socket=/run/ate-image-streaming/gcfsd/keychain.sock image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a@sha256:256545b7d3cda8db8d578e2bd70d4396d3f2d16c1bcd3fa549f8350f7b1daf9d server=us-central1-docker.pkg.dev username=_token
2026/10/09 23:36:14 INFO PrepareLayers timing breakdown provider=riptide image=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest digest=sha256:256545b7d3cda8db8d578e2bd70d4396d3f2d16c1bcd3fa549f8350f7b1daf9d layers=3 resolve_ms=281.129 stat_ms=7.36 prepare_ms=304.182 view_ms=12.861 listable_ms=2.136 wrapper_ms=0.748 total_ms=611.034
```

### 2.3 Cold & Warm `PrepareLayers` Outputs
```json
// Cold PrepareLayers (-bypass-ttl=false)
{
  "provider": "riptide",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest",
  "imageDigest": "sha256:256545b7d3cda8db8d578e2bd70d4396d3f2d16c1bcd3fa549f8350f7b1daf9d",
  "canProvide": true,
  "layerDirs": [
    "/var/lib/ateom-gvisor/streaming/riptide/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_image-a_latest/layer-0",
    "/var/lib/ateom-gvisor/streaming/riptide/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_image-a_latest/layer-1",
    "/var/lib/ateom-gvisor/streaming/riptide/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_image-a_latest/layer-2"
  ],
  "canProvideMs": 28.907,
  "prepareMs": 612.162,
  "releaseMs": 0.554,
  "verifiedLayers": 3
}

// Warm PrepareLayers (-bypass-ttl=false)
{
  "provider": "riptide",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest",
  "imageDigest": "sha256:256545b7d3cda8db8d578e2bd70d4396d3f2d16c1bcd3fa549f8350f7b1daf9d",
  "canProvide": true,
  "layerDirs": [
    "/var/lib/ateom-gvisor/streaming/riptide/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_image-a_latest/layer-0",
    "/var/lib/ateom-gvisor/streaming/riptide/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_image-a_latest/layer-1",
    "/var/lib/ateom-gvisor/streaming/riptide/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_image-a_latest/layer-2"
  ],
  "canProvideMs": 2.295,
  "prepareMs": 0.737,
  "releaseMs": 0.608,
  "verifiedLayers": 3
}
```

### 2.4 Host `containerd` (`gcfs` Snapshotter) Non-Interference & `ReleaseLayers` Verification
```text
Substrate Riptide snapshots (/var/lib/substrate/gcfs) BEFORE host containerd pull & GC:
6
Host containerd gcfs snapshots (/var/lib/containerd/io.containerd.snapshotter.v1.gcfs) BEFORE crictl pull:
313
Filesystem     Type       1K-blocks  Used Available Use% Mounted on
gcfsd          fuse.gcfsd      1200   600       600  50% /var/lib/substrate/gcfs/mnt
gcfsd          fuse.gcfsd      1200   600       600  50% /run/gcfsd/mnt

Pulling us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest via host crictl (uses host containerd + host gcfs snapshotter)...
Image is up to date for sha256:e1de99b9300a9cd0f42feea5d35b1a47722bec0a3bab2e5084705fd892a558c7
Host containerd gcfs snapshots (/var/lib/containerd/io.containerd.snapshotter.v1.gcfs) AFTER crictl pull:
314

Removing all references to us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest from host containerd with synchronous GC (crictl rmi + ctr -n k8s.io images rm --sync)...
Deleted: us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/image-a:latest
Host containerd gcfs snapshots AFTER synchronous GC:
313
Substrate Riptide snapshots (/var/lib/substrate/gcfs) AFTER host containerd synchronous GC (unchanged at 6):
6

Verifying all streamed layer views in /var/lib/ateom-gvisor/streaming/riptide are still mounted and readable:
  /var/lib/substrate/gcfs/mnt/views/34884abbe92863fce933ed7c39c0e045631af0ed86d5cc0dfbdf9fdca426ce3c/fs -> bin dev etc home lib 
  /var/lib/substrate/gcfs/mnt/views/1ab2c1381e7569b513169c0bf4195dcfc3d4834d315b1e9cd16848d141b45f2b/fs -> largefile 
  /var/lib/substrate/gcfs/mnt/views/23daa0df5f572dea1d9838e93f3011d7a50160750d2af5db2d05e293cb8115a5/fs -> app-a etc 

--- ReleaseLayers (-bypass-ttl=true) ---
releaseMs: 12.424 ms
Remaining view directories in /var/lib/ateom-gvisor/streaming/riptide after ReleaseLayers (-bypass-ttl=true):
0
Remaining snapshots in /var/lib/substrate/gcfs/snapshotter/snapshots (only committed parent chainIDs remain, active views removed):
3
```
