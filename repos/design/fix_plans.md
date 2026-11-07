# Fix plans for the design doc issues

Fix plans for the issues in section 11 of each design doc. Approved, with the decisions below. "Fold into" names the commit each fix amends.

## Summary

| # | Block | Issue | Recommended fix | Fold into | Size |
|---|---|---|---|---|---|
| T1 | atelet | No RPC timeouts | Add consts with `context.WithTimeout` | C2, C4 | S |
| T2 | atelet | `fallback` recorded with no plugin | New `client.ErrUnavailable`. Record nothing when no plugin is present. Time only the plugin part. | A2, C2, S5 | S |
| T3 | atelet | Spec removed after a failed release | Add a `providerLeasesDirty` flag that triggers a reconcile | C4 | S |
| T4 | atelet | No e2e | Fake plugin fixture + suite | new C8 | L |
| L1 | library | Warm hit skips the auth check | Re-authorize per (cred hash, ref) with a TTL | A7, A9 | M |
| L2 | library | Tag cache never expires | The same cache as L1, 1m TTL. Key leases by canonical ref. | A7, A9 | M |
| L3 | library | `time.Sleep` in `server_test` | `drainStarted` channel + `WaitForReady` | A10, A11 | S |
| N1 | Nydus | Bad seed digest fails every retry | Verify into a temp file before unpacking | B2 | S |
| N2 | Nydus | EROFS mounts leak after a restart | Glob `erofs-mnt-*` in the teardown hook | B1 | S |
| N3 | Nydus | Seed port 4010 has no auth | Bearer token from a Secret | B2, B5 | M |
| N4 | Nydus | Hardcoded passwords | Generate a Secret with `hack/create-secrets.sh` | B3, B7 | M |
| N5 | Nydus | Restart deletes `lease.json` (new) | Keep it. `CLEAR_CACHE_ON_START=true` wipes everything. | B4 | XS |
| R1 | Riptide | Restart deletes `lease.json` | Keep it. `CLEAR_CACHE_ON_START=true` (default false) wipes everything. | D3 | XS |
| R2 | Riptide | gcfsd port 11254 may collide | Add `GCFSD_PORT`, check on start, fail if taken | D3, D5 | S |
| R3 | Riptide | `RemoveCreds` never called | Call it from the work dir teardown hook, with a refcount | D2 | S |

## atelet

### T1. Plugin RPC timeouts
**Cause.** `ReleaseImage` (30s) and `State` (2s) already have timeouts. Three RPCs have none:
- `CanProvide` and `PrepareImage` use the Run/Restore ctx ([oci.go](../../cmd/atelet/oci.go)).
- `ReconcileImages` uses the monitor ctx and holds `providerLeaseMu.Lock()`. `prepareOCIBundles` takes `RLock`, so a hung reconcile blocks every bundle. It also stalls the monitor loop, so drain is never seen.

**Fix.** Follow the existing const + `context.WithTimeout` pattern:
- `CanProvide`: `imageProviderStateTimeout` (2s).
- `PrepareImage`: new `providerPrepareTimeout` = 1m. The cache fallback keeps the parent ctx and its 10m pull timeout.
- `ReconcileImages`: new `providerReconcileTimeout` = 30s. The write lock stays held across the RPC.

Use consts, not flags. A prepare that times out can leave an orphaned lease in the plugin. T3 cleans it up.

**Tests.** Assert `ctx.Deadline()` is set in the mock funcs in `TestImageSourceEnsure` and `TestReconcileProviderLeases`. Both fail today and need no sleeps.

### T2. Metric when no plugin is present
**Cause.** `GRPCClient.CanProvide` returns `(false, nil)` when the socket is missing. `ensure` treats that as a decline and records `outcome=fallback, provider="image-provider"`. The duration also includes the cache pull, but the registry brief says "time the plugin takes".

**Fix.**
- **Library (A2):** add an `ErrUnavailable` sentinel. `CanProvide` returns it when the socket is missing.
- **atelet (C2):** on `ErrUnavailable`, go straight to the cache and record nothing. The counter then counts only requests that reach a plugin. A stale socket or an RPC error is still recorded as `fallback`.
- **Duration:** take `elapsed` right after `s.prepare` returns.
- **Registry (S5):** reword the `fallback` and `error` briefs to match.

Alternatives:
- Check the socket in atelet. No library change, but every test row needs a real listener.
- A new `unavailable` outcome. It keeps the noise we have today.

**Tests.** Add a "plugin unavailable" row with no outcome recorded and 0 prepares. Add the missing `error` outcome row.

### T3. Spec removed after a failed release
**Cause.**
- `release` only logs the failure, and `releaseProviderLeases` removes `rootfs-overlay.json` anyway.
- Reconcile runs only when the socket appears ([imageprovider_reconcile.go:244](../../cmd/atelet/imageprovider_reconcile.go#L244)). I checked this. So nothing retries in steady state.
- The plugin's idle TTL never starts for the leaked lease, so the image stays leased forever.
- A drain waits out the full 30m, because no spec points at that lease.

**Fix.** Add `providerLeasesDirty atomic.Bool` on `Atelet`.
- Set it when `ReleaseImage` fails, and when `PrepareImage` fails with anything other than `ErrNotSupported` (this covers T1's orphan).
- `monitorImageProvider` reconciles when the socket appears **or** the flag is set, and clears the flag on success.
- Reconcile sends the full recorded set, so the plugin drops the leaked lease within about 2s. Removing the spec stays correct.

Alternative: keep the spec and retry on Terminate. That only partly works, because Run's `ResetActorDirsKeepingVolumes` deletes specs anyway.

**Tests.** Split out a `reconcileIfNeeded(available, now, dirty)` step and test it with table rows. This is also the first coverage for the monitor's edge detection.

### T4. e2e for gVisor and microVM
**Approach.**
- Add a ko-built fake plugin under `internal/e2e/fixtures/imageprovider`, following the `probe` and `testserver` precedent. It runs as a DaemonSet on `/run/ate/image-provider.sock`.
- Build it on the library's `plugin.Driver`. Its mounter pulls with go-containerregistry. This exercises the real lease, persist, reconcile and drain code.
- `CanProvide` accepts only refs passed in a flag, so parallel suites still fall back.
- The fake writes its lease set to a JSON file. The test reads it with `kubectl exec`.
- Runtime comes from `E2E_SANDBOX_CLASS`, so the same suite runs in both lanes.

**Scenarios.**
- `suites/imageprovider` (non-disruptive):
  - prepare via the plugin: lease `<uid>/<ctr>` appears;
  - release on terminate;
  - suspend, then resume.
- `internal/e2e/imageprovider` (disruptive, its own step):
  - reconcile after an atelet restart;
  - drain: the actor ends SUSPENDED and the plugin exits with zero leases.
- Fallback with no socket is already covered by every existing suite.

**Fold into.** A new commit C8 after C6. **Cost.** atelet vendors `plugin` and the containerd snapshots API. Adds CI time to both lanes.

**Alternative.** A raw gRPC fake built on the vendored proto. Fewer dependencies, but it reimplements drain and reconcile.

## Library

### L1. Warm hit skips the auth check
**Cause.** On a warm hit, `PrepareImage` grants the lease with no registry call. The warm-hit hook runs after the grant, so it pushes an unauthorized caller's creds to gcfsd or nydusd. The cold path has the same hole through the `resolvedMeta[ref]` cache.

**Fix.** Always re-authorize. Add no new option.
- `credKey(auth)`: a sha256 of the creds, or `"keychain"` for nil. Store only the hash.
- Replace `resolvedMeta` with a cache keyed by `credKey + ref` that holds `{meta, expires}`. Sweep expired entries on insert.
- Resolve before the warm-hit branch, with the lock dropped. Use the existing `imageResolver`, so no new export.
- Grant the lease and run the hook only after the resolve succeeds.
- Fail closed: map registry 401/403 to `PermissionDenied`.

**Limit.** Nil creds fall back to the node keychain, the same as kubelet. Document it.

**Alternatives.**
- Key warm entries by credential. That loses sharing, and rotating tokens keep turning warm hits cold.
- An opt-in option. That's a new knob and isn't fail-closed.

**Tests.** A new table in `driver_test.go` with a resolver that accepts only user "a". Rows:
- "b" on a warm hit: rejected, and the hook is not called.
- "a" again within the TTL: no resolver call.
- "b" cold after release: rejected.

**Risk.** Once the TTL lapses, warm hits need the registry.

### L2. Tag cache never expires
**Fix.** Use the same cache as L1 with `DefaultResolveTTL = 1m`. A burst of starts costs one resolve per credential, and a moved tag shows within a minute.

**Second bug found here.** If a tag moves while the old digest is still leased, the cold path replaces `d.leases[ref]` and tears down the old views under the old holders.
- Fix: key `leases`, `inflight` and `imageWorkDir` by `CanonicalImageRef(ref, digest)`.
- Reconcile rebuilds the key from `ActiveLease.ImageRef` + `ImageDigest`.
- Bump `workDirLayoutVersion`.

Nydus and Riptide already use `env.CanonicalRef`, so neither changes.

**Tests.** Synctest rows:
- The resolver returns D1, then D2. A prepare within the TTL gets D1, and after it gets D2.
- A lease still held on D1 releases cleanly.

### L3. `waitFor` sleeps in server_test
**Fix.**
- Drain test: add `drainStarted chan struct{}` to `Driver`. `StartDrain` closes it, next to the existing `drained` channel. The test blocks on it.
- Socket test: call `CanProvide` with `grpc.WaitForReady(true)`, bounded by `t.Context()`.
- Delete `waitFor`.

synctest doesn't fit here, because these goroutines block on real unix-socket I/O. A regression now hangs until `go test -timeout` instead of failing at 10s. Say that in the commit message.

## Nydus

### N1. A bad seed digest fails every retry
**Cause.** `digestReader` reports the mismatch only at EOF. By then `commitLayer` has already unpacked the layer, so the code can't fall back.

**Fix.** `seedLayer` copies the blob to a temp file while hashing and checks the digest. It then returns a reader over the file, which is deleted on Close. A mismatch becomes an ordinary seed miss, so the existing fallback to dfdaemon runs. Only bootstraps come through here, and they are small. Drop `digestReader`. Log the miss only, with no metric, since the plugin has no metrics yet.

**Tests.** Change the "seed serves a corrupt layer" row in `TestBlobSeedReads` to expect a fallback with correct content. It fails today.

### N2. EROFS mounts leak after a restart
**Cause.** `mountedDirs` is in memory only. After a restart, the teardown hook unmounts nothing, and `RemoveAll` fails on the read-only mount.

**Fix.** Mount paths are already deterministic (`<workDir>/erofs-mnt-<i>`). `unmountWorkDir` globs them and unmounts each in reverse with `MNT_DETACH`. It repeats until `EINVAL` to handle stacked mounts, and treats `EINVAL` and `ENOENT` as done. Delete `mountedDirs`. No library change.

**Tests.** Use a fresh driver over a work dir that has `erofs-mnt-0`, then call `ReconcileImages(nil)`. Assert the fake `Unmount` saw the dir.

### N3. Seed port 4010 has no auth
**Why not NetworkPolicy.** The seed and node pods are both hostNetwork. Most CNIs, including GKE Dataplane V2, don't enforce policy there, and `podSelector` can't match node-IP traffic.

**Fix.** A shared bearer token in Secret `nydus-blob-seed-token`.
- nginx checks `Authorization` through an envsubst template and returns 503 on a mismatch, so clients fall back.
- The plugin gets a new `--blob-seed-token-file` flag.
- nydusd gets the header in its seed mirror `headers`, and its config is chmod 600.

**Limit.** It's plain HTTP, so someone on the node network can sniff the token.

**Alternatives.** mTLS would be stronger, but it's unclear whether nydusd mirrors support client certs; that needs checking. A host firewall would change host-wide state, which `AGENTS.md` forbids.

**Out of scope.** The dfdaemon upload ports have the same exposure.

**Tests.** The fake seed rejects requests without the header. Add a "wrong token" row that expects a fallback.

### N4. Hardcoded Dragonfly passwords
**Where.** In `dragonfly.yaml`:
- Redis: `--requirepass` (L61) and `redis.password` (L98).
- MySQL: L90, `MYSQL_ROOT_PASSWORD`, and `--password=dragonfly` (L201).
- The JWT key (L83).

**Fix.**
- `hack/create-secrets.sh` creates `dragonfly-credentials` only if it's missing, using `openssl rand`. That works on macOS.
- Pods read it through non-optional `secretKeyRef`, so a missing Secret fails loudly.
- An init container renders the Redis config and `manager.yaml` into an emptyDir, so the passwords never sit in args. The node pods are hostPID and could read process args.
- The MySQL probe uses `MYSQL_PWD`.

**Also.** `dragonfly-controlplane` allows ingress from any pod. Narrow it in the same change.

**Tests.** `hack/verify/manifests.sh` fails if the `kustomize build` output contains the literals. It fails today.

## Riptide

### R1. A restart deletes `lease.json`
**Cause.** `start-riptide.sh:97` runs `rm -rf "${plugin_work_dir:?}"/*` on every start. I checked this. The library needs only `version` and `*/lease.json`. It already handles layout changes, orphan sweeps, and missing layers (re-prepare).

**Fix.** By default, drop the plugin work dir from the `rm`. Keep wiping the gcfs view state (`gcfsd`, `metadata.db`, `snapshots`). Rewrite the comment to say the library manages its work dir.

**Clean restart for benchmarks.** Reuse the existing `CLEAR_CACHE_ON_START` env, which defaults to `false`, as the switch:
- `false`: keep the layer cache and `lease.json`. This is the prod default.
- `true`: also wipe `"${plugin_work_dir:?}"/*`, on top of the layer cache it clears today. Every start is then fully cold.
- The benchmark harness (`scope_plugin.py`) already sets `CLEAR_CACHE_ON_START=true` for Riptide, so it needs no change.
- No new knob. A separate flag for "fresh leases, warm layers" has no user.
- Document in the README env table that `true` drops leases atelet still holds. Use it only when no actors run on the node, as between benchmark reps.

**Tests.** The script can't be unit tested, so verify by hand on GKE:
1. With the default, take a lease, then run `crictl stop` on the plugin container. Check that `lease.json` survives and the logs say "Reconciled image leases".
2. With `CLEAR_CACHE_ON_START=true`, do the same. Check that the work dir and the layer cache are empty.

Add case 1 to the M4 e2e as the restart case.

### N5. Nydus also deletes `lease.json` on every start
This is a new finding with the same bug as R1. [nydus-snapshotter.sh:138](../../repos/image-provider-plugin-nydus/manifests/scripts/nydus-snapshotter.sh#L138) runs `rm -rf "${nydus_root:?}"/* /var/lib/ate/image-plugin-cache/nydus/*` on every snapshotter start. Unmounting EROFS and clearing snapshots is required, because they die with nydusd. Wiping the plugin work dir is not.

**Fix (B4).** Use the same switch name as Riptide:
- Default: unmount EROFS and clear `snapshots` and `mnt`. Keep `lease.json`. The library re-prepares leases whose layers are gone. With N2's glob unmount, stale mounts are cleaned too.
- `CLEAR_CACHE_ON_START=true`: also clear the blob cache (`${nydus_root}/cache`) and the plugin work dir. This is today's behavior.
- Keeping the blob cache by default makes a normal restart warm, which is a behavior change. Check that nydusd reuses that cache after a restart. If it doesn't, keep wiping it by default and gate only the work dir.
- `scope_plugin.py` then sets `CLEAR_CACHE_ON_START=true` on the Nydus snapshotter container too.

**Tests.** The same two manual cases as R1, on a Nydus node.

### R2. gcfsd port 11254
**Facts.** The host gcfsd's port isn't stated anywhere in the repo or the prototype. Dropping `hostNetwork` doesn't help, because `nsenter -n` puts gcfsd in the host netns anyway.

**Fix.**
- Add a `GCFSD_PORT` env with a non-11254 default, after checking it on a node.
- Before the kills in `start-riptide.sh`, look for a listener on that port in `/proc/net/tcp{,6}` whose owner isn't our gcfsd. If there is one, exit 1 with a clear message.
- Document `GCFSD_PORT` in the README (D5).

**Verify on a GKE node with image streaming on:**
```
ps -o pid,args -C gcfsd
ss -ltnp | grep -E 'gcfsd|1125'
/home/kubernetes/bin/gcfsd --help 2>&1 | grep -iE 'port|socket'
```
If gcfsd accepts a unix socket for this listener, use that instead.

### R3. `RemoveCreds` never called
**Fix.** In D2:
- Register `plugin.WithWorkDirTeardownHook(d.removeCreds)`. Teardown runs after the idle TTL, on eviction, drain and sweep.
- Keep a `mounted` map from workDir to canonicalRef. Call `RemoveCreds` only when the last work dir for that canonicalRef goes.
- Serialize `UpdateCreds` and `RemoveCreds` under one `keychainMu`.
- Treat errors, including `Unimplemented`, as warnings.

Remove creds at teardown, not when the last lease goes. gcfsd fetches lazily, so it needs them while the views exist.

**Limit.** After a plugin-only restart, restored leases aren't in the map, so their creds stay until gcfsd restarts. Document it.

**Tests.** Table rows against `fakeKeychain.removes`:
- one image: one call;
- two refs with the same canonicalRef: a call only after both are torn down;
- an unknown work dir: no call;
- an error: `pushed` is still cleared.

## Decisions

1. **T4:** add the fake plugin as a new commit C8, built on `plugin.Driver`.
2. **L1/L2:** fold the canonical-ref lease keying and the layout version bump into A7/A9.
3. **N3:** use a bearer token now. mTLS is a follow-up once nydusd support is checked.
4. **R2:** add `GCFSD_PORT` and the fail-on-conflict check, and keep 11254 as the default until a GKE node check picks another port.
5. **N5:** keep wiping the Nydus blob cache by default until nydusd's reuse of it after a restart is checked. `CLEAR_CACHE_ON_START=true` also wipes the plugin work dir.
6. **R1/N5:** reuse `CLEAR_CACHE_ON_START` (default `false`) as the clean-restart switch for benchmarks.
7. **Folding:** each fix amends its commit with fixup and autosquash.

## Suggested order

1. Small and local: R1, N2, N1, L3, T1.
2. Plugin auth and lifecycle: L1+L2 together, then R3, then T2+T3.
3. Manifests and secrets: N4, N3, R2 (after the node check).
4. e2e: T4.

Each step runs `go test -race -count=3 -shuffle=on` and `hack/verify-all.sh` in every touched module, then updates section 11 of each design doc and the plan.
