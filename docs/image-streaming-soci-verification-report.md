# AWS SOCI (`--min-layer-size=0`) Live Cluster Verification Report

**Cluster:** `substrate-stream-test` (`us-central1-a`, project `kuiyue-gke-dev`)  
**Nodes:** `gke-substrate-stream-substrate-node-p-01c8a8ea-sae4`, `gke-substrate-stream-substrate-node-p-01c8a8ea-65vz`  
**Date:** September 30, 2026  
**Verified Commit:** `f19c7522` / `fcfd8b5f` (`prototype-remote-snapshotter-v3`)  
**`atelet` Image:** `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/atelet-89dbecdd4e8d5cd4d125a2de341f399c@sha256:c23abaa73932813efb1862732a89f696dea32a984708ba180fad1c230692587a`  
**Workload Image:** `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e` (9 layers, 1.19 GB compressed / ~3.5 GB unpacked, indexed with `soci create --min-layer-size=0`)

---

## 1. Summary of Live Verification Results (`--image-streamer=soci`)

| Phase / Scenario | `PrepareLayers` Breakdown (`soci`) | `oci_unpack` | `ateom_restore` | `AteomHerder` RPC Total | CLI Wall-Clock |
| :--- | :--- | ---: | ---: | ---: | ---: |
| **1. Cold `create actor-template`** (`sandbox-1gb-soci`, `Run` + `Checkpoint`) | `resolve`: `437.3 ms`<br>`stat`: `21.5 ms`<br>`prepare`: `3,388.3 ms`<br>`view`: `62.5 ms`<br>`listable`: `3.8 ms`<br>`wrapper`: `1.2 ms`<br>**`total`: `3,914.8 ms`** | `3,916.0 ms` | — | **`4,880.8 ms`** (`Run`)<br>**`461.5 ms`** (`Checkpoint`) | `42,000 ms` (full golden snapshot lifecycle) |
| **2. Cold-Node `resume actor`** (`bench-soci-1`, empty `soci-snapshotter` & cache) | `resolve`: **`0 ms`** *(cached)*<br>`stat`: `13.9 ms`<br>`prepare`: `3,471.7 ms`<br>`view`: `39.5 ms`<br>`listable`: `3.6 ms`<br>`wrapper`: `1.4 ms`<br>**`total`: `3,530.3 ms`** | **`3,532.3 ms`** | `149.0 ms` | **`3,759.1 ms`** | **`4,728 ms`** |
| **3. Warm-Lease `resume actor`** (`bench-soci-2`, `refCount: 1 → 2`) | In-memory lease hit (`3.65 ms`) | **`5.2 ms`** | `138.2 ms` | **`277.1 ms`** | **`1,303 ms`** |
| **4. Full Drain (`2 → 0`) + Re-Attach (`0 → 1`)** (`bench-soci-1`, `Stat(chainID)` hit) | `resolve`: **`0 ms`** *(cached)*<br>`stat`: **`10.6 ms`** *(9/9 hit)*<br>`prepare`: **`0 ms`**<br>`view`: **`40.4 ms`**<br>`listable`: `1.6 ms`<br>`wrapper`: `0.9 ms`<br>**`total`: `53.6 ms`** | **`55.1 ms`** | `156.4 ms` | **`235.2 ms`** | **`1,327 ms`** |

---

## 2. Key Latest-Version (`fcfd8b5f`) Behaviors Confirmed with AWS SOCI

1. **Startup Probe & Socket Validation:**
   ```json
   {"time":"2026-09-30T05:00:34.482390304Z","level":"INFO","msg":"Image streaming enabled","mode":"soci","provider":"soci","socket":"/run/soci-snapshotter-grpc/soci-snapshotter-grpc.sock","reachable":true}
   ```
2. **Persisted `diffIDs` / `layerDigests` in `lease.json` & `"streamed": true` in `rootfs-overlay.json`:**
   - `/var/lib/ateom-gvisor/streaming/soci/.../lease.json` persisted all 9 `diffIDs`, `layerDigests`, `snapshotKeys`, and `layerDirs`.
   - `/var/lib/ateom-gvisor/actors/a6935d44-4f42-42db-af76-661ba4cd454f/bundles/sandbox/rootfs-overlay.json` recorded `"streamed": true` and all 9 `/var/lib/ateom-gvisor/streaming/soci/.../layer-{0..8}` paths.
3. **`refCount: 2 → 0 → 1` Fast Re-Attachment (`53.6 ms` total `PrepareLayers`):**
   - After pausing both actors (`refCount` dropped to `0` and views were removed) and resuming `bench-soci-1`, `PrepareLayers` skipped the registry network round-trip (`resolve_ms: 0`), hit `Stat(chainID)` on all 9 committed layers in `soci-snapshotter-grpc` (`stat_ms: 10.589`, `prepare_ms: 0`), re-created the 9 views (`view_ms: 40.375`), and completed the entire `AteomHerder/Restore` RPC in **`235.2 ms`**.

---

## 3. On-Node Filesystem Evidence

### `lease.json` (`/var/lib/ateom-gvisor/streaming/soci/.../lease.json`)
```json
{
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e",
  "digest": "sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e",
  "config": {
    "Entrypoint": [
      "/ko-app/sandbox"
    ],
    "Env": [
      "PATH=/builder/google-cloud-sdk/bin/:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/ko-app",
      "KO_DATA_PATH=/var/run/ko"
    ],
    "Labels": {
      "org.opencontainers.image.version": "22.04"
    }
  },
  "diffIDs": [
    "sha256:ea16cace89338c84eb6bcb91a7efdfcae6838fff359efe951858227436486c34",
    "sha256:fd14681b3f78357c6ffbcef2d304d2323c29797bde46a32f3037701f7a431fd6",
    "sha256:17e3008119c76df2e3a50da6fe465488993955150901315b1f09d9fded346507",
    "sha256:8512b779ac15768e46d1c70d0c7d0031eae14a47cd0c3ce7588d843419cee4d1",
    "sha256:f5f3409de2bcb398e84055464ed1a8b701f7e964f019e9c5d679002e6a0bea93",
    "sha256:fc77ab78e128550fbf89806ef258df94a58ff8a9cdcb5c66241c7815bf630de0",
    "sha256:584c997258045503a652a2773234cd3f9c48e320ebbeb037b7a703d8a684d2b8",
    "sha256:ffe56a1c5f3878e9b5f803842adb9e2ce81584b6bd027e8599582aefe14a975b",
    "sha256:b80be4baa9e9d46a68c44421143a8241fc7f7e03178118805f002b931332caa1"
  ],
  "layerDigests": [
    "sha256:84ad5f3a28e133b7e3d79a5974fa2411b373762f03734f30c1c2b75fb05bcb40",
    "sha256:a17129cc092793e477505b92a06cd8d44c74989d116656cb6624a2e5b87aa089",
    "sha256:e9a96d24d607549aeede6055c93293c34678b2225a6f20c8923a103a55171e4a",
    "sha256:2edc65f4c1b13f4dbd79d1298ea6a3070f86022e6726d99a73830a6a25005ac9",
    "sha256:eff5bf5846f61928f32134940b127704dec6e7b6fb2919eae7d6043f96198f4b",
    "sha256:eee00ea291453c8f52c7bd4d604a2dee3ae6e1ca990c77342a0989d51524ca2f",
    "sha256:6cca528c578ed71cc7845b45d5d2b4d13be0167c2873e44fbca2b847980bbda1",
    "sha256:4a2796d8ea5ef7ae1a64ba20ba56759987f49f2ea2091ddce6bade81c0ad1fa6",
    "sha256:4778a54a0081597a421cef5da837a421e979562e2a0cf1dd9d360ce727d4eea9"
  ],
  "snapshotKeys": [
    "us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e-18da00a3b7f7ef90-l0-view",
    "us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e-18da00a3b7f7ef90-l1-view",
    "us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e-18da00a3b7f7ef90-l2-view",
    "us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e-18da00a3b7f7ef90-l3-view",
    "us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e-18da00a3b7f7ef90-l4-view",
    "us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e-18da00a3b7f7ef90-l5-view",
    "us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e-18da00a3b7f7ef90-l6-view",
    "us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e-18da00a3b7f7ef90-l7-view",
    "us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e-18da00a3b7f7ef90-l8-view"
  ],
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
  ]
}
```

### `rootfs-overlay.json` (`/var/lib/ateom-gvisor/actors/a6935d44-4f42-42db-af76-661ba4cd454f/bundles/sandbox/rootfs-overlay.json`)
```json
{
  "version": 1,
  "imageDigest": "sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e",
  "imageRef": "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e",
  "streamed": true,
  "layers": [
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-0",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-1",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-2",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-3",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-4",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-5",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-6",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-7",
    "/var/lib/ateom-gvisor/streaming/soci/us-central1-docker.pkg.dev_kuiyue-gke-dev_kuiyue-gke-dev-repo_ate-images_sandbox-04176181c57f0bf23e61506b0ddcf1fd_sha256_fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e/layer-8"
  ]
}
```

---

## 4. Structured `atelet` Logs (`pod/atelet-v612b1bb0bb-n9q2h` on `gke-substrate-stream-substrate-node-p-01c8a8ea-65vz`)

```json
{"time":"2026-09-30T05:00:34.482390304Z","level":"INFO","msg":"Image streaming enabled","mode":"soci","provider":"soci","socket":"/run/soci-snapshotter-grpc/soci-snapshotter-grpc.sock","reachable":true}
{"time":"2026-09-30T05:01:01.479124274Z","level":"INFO","msg":"Image cache hit","ref":"registry.k8s.io/pause:3.10.2@sha256:f548e0e8e3dc1896ca956272154dde3314e8cc4fde0a57577ee9fa1c63f5baf4","digest":"sha256:f548e0e8e3dc1896ca956272154dde3314e8cc4fde0a57577ee9fa1c63f5baf4","trace_id":"8edc8db06da81271f6f2d2d3e37e6935","span_id":"d8e79270512968ac","trace_flags":"00"}
{"time":"2026-09-30T05:01:05.39496063Z","level":"INFO","msg":"PrepareLayers timing breakdown","streamer":"soci","image":"us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e","digest":"sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e","layers":9,"resolve_ms":437.266,"stat_ms":21.482,"prepare_ms":3388.329,"view_ms":62.51,"listable_ms":3.841,"wrapper_ms":1.162,"total_ms":3914.842,"trace_id":"8edc8db06da81271f6f2d2d3e37e6935","span_id":"c5cbd0b31da8456b","trace_flags":"00"}
{"time":"2026-09-30T05:01:05.3950427Z","level":"DEBUG","msg":"Image streamed","image":"us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e","streamer":"soci","layers":9,"duration":3916043786,"trace_id":"8edc8db06da81271f6f2d2d3e37e6935","span_id":"c5cbd0b31da8456b","trace_flags":"00"}
{"time":"2026-09-30T05:01:06.33624317Z","level":"INFO","msg":"Handle RPC","method":"/atelet.AteomHerder/Run","req":{"target_ateom_uid":"ad0d31e7-3253-4432-b5b0-cab208794649","atespace":"ate-golden","actor_name":"ba067555-1f81-4929-a2ab-d75433f4f708","actor_uid":"d116a160-25e7-40ee-bb50-fcce02d8e60d","actor_template_atespace":"ate-demo-sandbox","actor_template_name":"sandbox-1gb-soci"},"resp":{},"err":null,"elapsed-time":"4.880849623s","trace_id":"8edc8db06da81271f6f2d2d3e37e6935","span_id":"e714394a7b9b19ba","trace_flags":"00"}
{"time":"2026-09-30T05:01:26.808295178Z","level":"INFO","msg":"Handle RPC","method":"/atelet.AteomHerder/Checkpoint","req":{"target_ateom_uid":"ad0d31e7-3253-4432-b5b0-cab208794649","atespace":"ate-golden","actor_name":"ba067555-1f81-4929-a2ab-d75433f4f708","actor_uid":"d116a160-25e7-40ee-bb50-fcce02d8e60d","actor_template_atespace":"ate-demo-sandbox","actor_template_name":"sandbox-1gb-soci"},"resp":{},"err":null,"elapsed-time":"461.452534ms","trace_id":"89bc240e8227bf89fd4d4b84f13dc346","span_id":"fcb7ec51b2a1612d","trace_flags":"00"}
{"time":"2026-09-30T05:01:44.998179821Z","level":"INFO","msg":"Image cache hit","ref":"registry.k8s.io/pause:3.10.2@sha256:f548e0e8e3dc1896ca956272154dde3314e8cc4fde0a57577ee9fa1c63f5baf4","digest":"sha256:f548e0e8e3dc1896ca956272154dde3314e8cc4fde0a57577ee9fa1c63f5baf4","trace_id":"f9a5f8cb0ee8a44d318454ad0f39978d","span_id":"d678f1fb0069f7a5","trace_flags":"00"}
{"time":"2026-09-30T05:01:48.529623485Z","level":"INFO","msg":"PrepareLayers timing breakdown","streamer":"soci","image":"us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e","digest":"sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e","layers":9,"resolve_ms":0,"stat_ms":13.885,"prepare_ms":3471.721,"view_ms":39.547,"listable_ms":3.588,"wrapper_ms":1.388,"total_ms":3530.285,"trace_id":"f9a5f8cb0ee8a44d318454ad0f39978d","span_id":"ce0f023798a8f94a","trace_flags":"00"}
{"time":"2026-09-30T05:01:48.529690054Z","level":"DEBUG","msg":"Image streamed","image":"us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e","streamer":"soci","layers":9,"duration":3531459095,"trace_id":"f9a5f8cb0ee8a44d318454ad0f39978d","span_id":"ce0f023798a8f94a","trace_flags":"00"}
{"time":"2026-09-30T05:01:48.679185228Z","level":"INFO","msg":"Restore timing breakdown","ate.atespace":"ate-demo-sandbox","ate.actor.name":"bench-soci-1","ate.actor.uid":"f0ae1083-953c-4495-918e-fbb64e6b9ebd","ate.template.atespace":"ate-demo-sandbox","ate.template.name":"sandbox-1gb-soci","ate.snapshot.scope":"full","ate.snapshot.kind":"golden","ate.sandbox.class":"gvisor","ate.actor.restore.duration.volume_mount":5.38e-7,"ate.actor.restore.duration.manifest_fetch":0.054478964,"ate.actor.restore.duration.sandbox_assets":0.000076876,"ate.actor.restore.duration.download":0.133046016,"ate.actor.restore.duration.oci_unpack":3.532259863,"ate.actor.restore.duration.ateom_restore":0.148990047,"ate.actor.restore.duration.total":3.759066387,"trace_id":"f9a5f8cb0ee8a44d318454ad0f39978d","span_id":"14c5c30ba55349e3","trace_flags":"00"}
{"time":"2026-09-30T05:01:48.679311075Z","level":"INFO","msg":"Handle RPC","method":"/atelet.AteomHerder/Restore","req":{"target_ateom_uid":"ccd6de17-bdf3-41a6-a27d-6adc30421f8d","atespace":"ate-demo-sandbox","actor_name":"bench-soci-1","actor_uid":"f0ae1083-953c-4495-918e-fbb64e6b9ebd","actor_template_atespace":"ate-demo-sandbox","actor_template_name":"sandbox-1gb-soci"},"resp":{},"err":null,"elapsed-time":"3.75943719s","trace_id":"f9a5f8cb0ee8a44d318454ad0f39978d","span_id":"14c5c30ba55349e3","trace_flags":"00"}
{"time":"2026-09-30T05:01:50.770658865Z","level":"INFO","msg":"Image cache hit","ref":"registry.k8s.io/pause:3.10.2@sha256:f548e0e8e3dc1896ca956272154dde3314e8cc4fde0a57577ee9fa1c63f5baf4","digest":"sha256:f548e0e8e3dc1896ca956272154dde3314e8cc4fde0a57577ee9fa1c63f5baf4","trace_id":"a33f90158b9c7b4f3ceef4fa0dd591aa","span_id":"b428b4050a141fa1","trace_flags":"00"}
{"time":"2026-09-30T05:01:50.774026067Z","level":"DEBUG","msg":"Image streamed","image":"us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e","streamer":"soci","layers":9,"duration":3651586,"trace_id":"a33f90158b9c7b4f3ceef4fa0dd591aa","span_id":"ea50c8d2e5aa9ec7","trace_flags":"00"}
{"time":"2026-09-30T05:01:50.987410865Z","level":"INFO","msg":"Restore timing breakdown","ate.atespace":"ate-demo-sandbox","ate.actor.name":"bench-soci-2","ate.actor.uid":"a6935d44-4f42-42db-af76-661ba4cd454f","ate.template.atespace":"ate-demo-sandbox","ate.template.name":"sandbox-1gb-soci","ate.snapshot.scope":"full","ate.snapshot.kind":"golden","ate.sandbox.class":"gvisor","ate.actor.restore.duration.volume_mount":4.64e-7,"ate.actor.restore.duration.manifest_fetch":0.034568578,"ate.actor.restore.duration.sandbox_assets":0.000172939,"ate.actor.restore.duration.download":0.079830663,"ate.actor.restore.duration.oci_unpack":0.005239838,"ate.actor.restore.duration.ateom_restore":0.138208351,"ate.actor.restore.duration.total":0.277086495,"trace_id":"a33f90158b9c7b4f3ceef4fa0dd591aa","span_id":"7c31587d25ab3acc","trace_flags":"00"}
{"time":"2026-09-30T05:01:50.987563332Z","level":"INFO","msg":"Handle RPC","method":"/atelet.AteomHerder/Restore","req":{"target_ateom_uid":"ad0d31e7-3253-4432-b5b0-cab208794649","atespace":"ate-demo-sandbox","actor_name":"bench-soci-2","actor_uid":"a6935d44-4f42-42db-af76-661ba4cd454f","actor_template_atespace":"ate-demo-sandbox","actor_template_name":"sandbox-1gb-soci"},"resp":{},"err":null,"elapsed-time":"277.323267ms","trace_id":"a33f90158b9c7b4f3ceef4fa0dd591aa","span_id":"7c31587d25ab3acc","trace_flags":"00"}
{"time":"2026-09-30T05:01:54.878860694Z","level":"INFO","msg":"Handle RPC","method":"/atelet.AteomHerder/Checkpoint","req":{"target_ateom_uid":"ccd6de17-bdf3-41a6-a27d-6adc30421f8d","atespace":"ate-demo-sandbox","actor_name":"bench-soci-1","actor_uid":"f0ae1083-953c-4495-918e-fbb64e6b9ebd","actor_template_atespace":"ate-demo-sandbox","actor_template_name":"sandbox-1gb-soci"},"resp":{},"err":null,"elapsed-time":"105.743484ms","trace_id":"ef9528e483ff5e96110dfe7a2e74069c","span_id":"38caebcf2da72618","trace_flags":"01"}
{"time":"2026-09-30T05:01:56.001082799Z","level":"INFO","msg":"Handle RPC","method":"/atelet.AteomHerder/Checkpoint","req":{"target_ateom_uid":"ad0d31e7-3253-4432-b5b0-cab208794649","atespace":"ate-demo-sandbox","actor_name":"bench-soci-2","actor_uid":"a6935d44-4f42-42db-af76-661ba4cd454f","actor_template_atespace":"ate-demo-sandbox","actor_template_name":"sandbox-1gb-soci"},"resp":{},"err":null,"elapsed-time":"124.36601ms","trace_id":"7fde768ce6c14388076f17c085da605a","span_id":"a6f7f2d24e4058bb","trace_flags":"01"}
{"time":"2026-09-30T05:01:59.123914608Z","level":"INFO","msg":"Image cache hit","ref":"registry.k8s.io/pause:3.10.2@sha256:f548e0e8e3dc1896ca956272154dde3314e8cc4fde0a57577ee9fa1c63f5baf4","digest":"sha256:f548e0e8e3dc1896ca956272154dde3314e8cc4fde0a57577ee9fa1c63f5baf4","trace_id":"1bbd560aaac970509613dc9b008ce880","span_id":"aaf9cbfaeb9e4bf4","trace_flags":"00"}
{"time":"2026-09-30T05:01:59.177964809Z","level":"INFO","msg":"PrepareLayers timing breakdown","streamer":"soci","image":"us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e","digest":"sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e","layers":9,"resolve_ms":0,"stat_ms":10.589,"prepare_ms":0,"view_ms":40.375,"listable_ms":1.551,"wrapper_ms":0.946,"total_ms":53.604,"trace_id":"1bbd560aaac970509613dc9b008ce880","span_id":"3a1b783e97481485","trace_flags":"00"}
{"time":"2026-09-30T05:01:59.178112029Z","level":"DEBUG","msg":"Image streamed","image":"us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:fa2f27bfa131cd35d2b81074ffd2e4ff6fe2bc78b92f0f9dfc494318f70e819e","streamer":"soci","layers":9,"duration":54296900,"trace_id":"1bbd560aaac970509613dc9b008ce880","span_id":"3a1b783e97481485","trace_flags":"00"}
{"time":"2026-09-30T05:01:59.33526386Z","level":"INFO","msg":"Restore timing breakdown","ate.atespace":"ate-demo-sandbox","ate.actor.name":"bench-soci-1","ate.actor.uid":"f0ae1083-953c-4495-918e-fbb64e6b9ebd","ate.template.atespace":"ate-demo-sandbox","ate.template.name":"sandbox-1gb-soci","ate.snapshot.scope":"full","ate.snapshot.kind":"local","ate.sandbox.class":"gvisor","ate.actor.restore.duration.volume_mount":6.44e-7,"ate.actor.restore.duration.manifest_fetch":0.00005867,"ate.actor.restore.duration.sandbox_assets":0.000074629,"ate.actor.restore.duration.download":0.001310669,"ate.actor.restore.duration.oci_unpack":0.055122594,"ate.actor.restore.duration.ateom_restore":0.15642491,"ate.actor.restore.duration.total":0.235194849,"trace_id":"1bbd560aaac970509613dc9b008ce880","span_id":"d5ed7c9c1064cde0","trace_flags":"00"}
{"time":"2026-09-30T05:01:59.335410899Z","level":"INFO","msg":"Handle RPC","method":"/atelet.AteomHerder/Restore","req":{"target_ateom_uid":"ad0d31e7-3253-4432-b5b0-cab208794649","atespace":"ate-demo-sandbox","actor_name":"bench-soci-1","actor_uid":"f0ae1083-953c-4495-918e-fbb64e6b9ebd","actor_template_atespace":"ate-demo-sandbox","actor_template_name":"sandbox-1gb-soci"},"resp":{},"err":null,"elapsed-time":"235.493526ms","trace_id":"1bbd560aaac970509613dc9b008ce880","span_id":"d5ed7c9c1064cde0","trace_flags":"00"}
```
