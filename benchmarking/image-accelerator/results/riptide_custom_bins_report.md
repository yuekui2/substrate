# Live test: custom Riptide binaries

**Result: PASS.** The DaemonSet ran the GCS builds of `gcfsd` (v1.37-2) and `containerd-gcfs-grpc` (v1.37-1). Running them only took the `riptide-bin-image` tool plus one `kubectl patch`. A real PrepareImage on the custom stack mounted and verified 10 layers in 1.54s.

Raw log of every command, its output and exit code: [riptide_custom_bins_evidence.log](riptide_custom_bins_evidence.log)

## Setup

| Item | Value |
|---|---|
| Cluster | `substrate-stream-test` (us-central1-a, project `kuiyue-gke-dev`) |
| New node pool | `riptide-custom-pool`: COS, c3-standard-4, 1.36.4-gke.2046003, image streaming on, label `riptide-custom-test=true` |
| Node | `gke-substrate-stream-riptide-custom-p-4a125b4f-bjs1` |
| DaemonSet | `ate-system/ate-accelerator-riptide`, created fresh (none existed before), `nodeSelector` limits it to the new pool |
| Adapter image | built with ko from the current tree: `.../riptide-6083…@sha256:e2f09f41…` |
| Custom binaries | `gs://gke-release-staging/gcfsd/v1.37-2/gcfsd`, `gs://gke-release-staging/gcfs-snapshotter/v1.37-1/containerd-gcfs-grpc` |
| Binary image | `us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/riptide-bins@sha256:e6648c4d79f258df2ad6aea531428ebfc0d7fec5010b7adca48307f3479343f5` |

The node runs host `gcfsd.service` (Mode 1). That's the harder case, because the isolated stack has to coexist with GKE's own Riptide.

## Steps

| # | Step | Outcome |
|---|---|---|
| 1 | Create fresh node pool | Created |
| 2 | Download binaries from GCS | Both are amd64 ELF. sha256: gcfsd `41f9b517…`, snapshotter `094a195e…` |
| 3 | ko build of the adapter | Pushed `sha256:e2f09f41…` |
| 4b | `--help` flag check | Both binaries accept every flag `riptide.yaml` passes, including the v2-only `--enable_id_mapping` and `--enable-v2` |
| 5 | `riptide-bin-image --patch-out` | Pushed `riptide-bins@sha256:e6648c4d…` and wrote `riptide-custom-bins.yaml` (printed in the log) |
| 5b | Inspect pushed image | `linux/amd64`. `/riptide/gcfsd-v2` and `/riptide/containerd-gcfs-grpc`, mode 0755, owned by root, mtime 1970 |
| 5c | Rebuild with same inputs | Same digest `e6648c4d…`, so builds are reproducible |
| 7 | Host baseline | `/var` is `noexec`, `/home/kubernetes/bin` is exec, no `ate-riptide/` dir yet |
| 8 | Base DaemonSet (control run) | 2/2 Ready. `Using gcfsd /home/kubernetes/bin/gcfsd-v2 …`, GCFSD **v1.48.0**, snapshotter `dev-e4a08049…` |
| 9 | `kubectl patch … --patch-file riptide-custom-bins.yaml` | Init container exited 0. 2/2 Ready. `Using gcfsd /home/kubernetes/bin/ate-riptide/gcfsd-v2 …`, GCFSD **v1.37-2**, snapshotter `dev-0445520e…` |
| 10 | Host after patch | See below |
| 11c | PrepareImage / ReleaseImage | **PASS**: `canProvide=true`, 10 layers verified, prepare 1539.8 ms, release 28.4 ms |
| 12 | Negative test: missing dir | See below |
| 12b | Restore env | 2/2 Ready again on the custom binaries |

## Key evidence

**Step 10, host after the patch:**

```
local:  41f9b517…  bins/gcfsd                 094a195e…  bins/containerd-gcfs-grpc
host:   41f9b517…  ate-riptide/gcfsd-v2       094a195e…  ate-riptide/containerd-gcfs-grpc

pid=17063 exe=/home/kubernetes/bin/ate-riptide/gcfsd-v2              (isolated stack)
pid=17120 exe=/home/kubernetes/bin/ate-riptide/containerd-gcfs-grpc  (isolated stack)
pid=2370  exe=/home/kubernetes/bin/gcfsd-v2                          (host gcfsd.service, unchanged PID)
pid=2400  exe=/home/kubernetes/bin/containerd-gcfs-grpc              (host, unchanged)
preloaded sha256 before and after: b8ff21bb… / ba2fc10c… / 89d2250d…  (identical)
```

**Step 12, negative test:**

```
RIPTIDE_BIN_DIR=/home/kubernetes/bin/does-not-exist
Riptide binary /home/kubernetes/bin/does-not-exist/gcfsd-v2 is missing or not executable
riptide-provider exitCode=1, CrashLoopBackOff, pod 0/2 Ready
```

## Notes

- **First PrepareImage was declined.** The Riptide backend returned `FAILED_PRECONDITION: Image import is in progress` for this image. The adapter then cached the decline for its 10-minute `DefaultDeclineTTL`. Once that expired, the retry passed (step 11c). This comes from the backend. The custom binaries handled it correctly.
- **First negative-test run captured the wrong pod.** The old pod was still finishing its lame-duck drain, so the run logged the old pod. Step 12 (redo) waits for the new pod and is the valid result. Both runs are in the log.
- **Benchmark probe.** `e2e-stream-probe` exits 0 when `canProvide=false`. Step 11c checks for `canProvide: true` explicitly.

## Resources left running

- Node pool `riptide-custom-pool` (1 × c3-standard-4)
- DaemonSet `ate-system/ate-accelerator-riptide`, pod `ate-system/riptide-test-helper`
- AR images `riptide-bins:gcfsd-v1.37-2_snap-v1.37-1` and `:rebuild-check`

Cleanup:

```bash
kubectl -n ate-system delete pod riptide-test-helper
kubectl -n ate-system delete ds ate-accelerator-riptide
gcloud container node-pools delete riptide-custom-pool --cluster substrate-stream-test --zone us-central1-a --project kuiyue-gke-dev
```
