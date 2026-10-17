# Log sources for `profile_coldboot.py` and `profile_lifecycle.py`

Companion to [README.md](README.md): which log records each number in the profilers' output comes from, and how each record is tied to the actor being measured. Example lines are in [log-appendix.md](log-appendix.md).

Both scripts read only the logs Substrate already writes, plus the `workload start` / `workload end` lines printed from inside the actor by a [`profiling/workloads`](workloads) script, when the template sets `ACTOR_SCRIPT`. Neither script changes any Substrate code.
They share one scraper, [`logs()`](profile_lifecycle.py#L115-L149):

```bash
kubectl logs -l <selector> -n <namespace> --since=<N>s --tail=-1 [--prefix]
```

| Component | Selector | Namespace | `--prefix`? |
|---|---|---|---|
| ate-api-server (ateapi) | `app=ate-api-server` | `ate-system` | no |
| atelet | `app=atelet` | `ate-system` | no |
| ateom (worker pods) | `ate.dev/worker-pool=<pool>`, or `--worker-selector` if given | `--worker-namespace` (`ate-profiling`) | **yes**, so each record keeps its worker pod |
| actor stdout (workload script) | same as ateom: ateom copies the container's stdout into the worker pod's log | same as ateom | yes |

For each output line, `logs()` does the following:
1. If `--prefix` was used, it removes the `[pod/<pod>/<container>]` prefix and saves the pod name as `_pod`.
2. It drops any line that is not a JSON object.
3. It adds three keys:
   - `_ts`: epoch seconds, parsed from `time` (or `timestamp`) in RFC3339Nano format.
   - `_raw`: the original line, used for substring matching.
   - `_pod`: the worker pod name (empty unless `--prefix` was used).

The message text is read from `msg` (component slog records and workload lines) or `message` (ateom lifecycle envelope), in that order, by [`msg_of`](profile_lifecycle.py#L93-L95).

---

## Log formats

All components write **one JSON object per line**. There are three shapes. Full lines for every scraped message are in the [appendix](log-appendix.md).

**A. Component slog records** (ateapi, atelet, and ateom's own `About to run runsc …`). These come from Go `slog.NewJSONHandler` wrapped in `contextlogging`, which adds the OTel trace fields when the context has a span:

```json
{"time":"…","level":"INFO","msg":"Image pulled into layer cache",
 "digest":"sha256:64bb043b…","layers":10,"took":89797006842,
 "trace_id":"878a2249…","span_id":"ea2554f1…","trace_flags":"00"}
```

**B. ateom lifecycle envelope** (`Actor starting/started`, `restoring/restored`, `checkpointing/checkpointed`). This comes from [`internal/actorlog/logger.go`](../internal/actorlog/logger.go#L76-L101). It uses `message` instead of `msg`, and puts the actor identity in a label group:

```json
{"logging.googleapis.com/labels":{"ate.actor.name":"88c9fb30-…","ate.actor.uid":"cbf80e06-…",
   "ate.atespace":"ate-golden","ate.template.atespace":"ate-prof-pull",
   "ate.template.name":"swebench-matplotlib-matplotlib-23476"},
 "message":"Actor starting","time":"…",
 "trace_id":"878a2249…","span_id":"d3aab064…","trace_flags":"00"}
```

**C. Actor stdout, re-encoded by ateom** (the workload script's `workload start` / `workload end`). The script prints one JSON object per line ([`file-read.sh`](workloads/file-read.sh), [`compile.sh`](workloads/compile.sh)):

```json
{"msg":"workload end","workload":"readfile","trigger":"resume","file":"/testbed/lib/matplotlib/pyplot.py","bytes":…,"elapsed_us":…}
```

ateom's [`WrapContainerLogs`](../internal/actorlog/logger.go#L137-L178) adds the two fields the scripts rely on:
- **`time`**: ateom's clock when it reads the line from the container's stdout. This is the timestamp every TTCE row uses.
- **The actor's label group** (`ate.actor.name`, …): this is how a line is tied to the actor, since there's no trace ID.

On GCE the label group is named `logging.googleapis.com/labels`. Elsewhere it is plain `labels`. [`actor_name_of`](profile_lifecycle.py#L98-L112) reads both.

**Units differ between fields:**

| Field type | Encoding | Converted by |
|---|---|---|
| `slog.Duration` (`took`, `duration`, `elapsed`, FinalizeSuspended fields) | int64 **nanoseconds** | `ns_to_s` |
| ateapi `elapsed-time` | Go duration **string** (`"465.703717ms"`) | `parse_go_duration` |
| `PrepareLayers timing breakdown` `*_ms` | float **milliseconds** | `/ 1000` |
| `Restore timing breakdown` `ate.actor.restore.duration.*` | float **seconds**; older builds wrote bare keys in int64 ns | read as-is, or `ns_to_s` for the old keys |
| `Picked worker` `worker` | proto **text** (`worker_pod:\"…\"  node_name:\"…\"`) | regex |
| workload `elapsed_us` | int **microseconds**, from bash `EPOCHREALTIME` inside the sandbox | `/ 1e6` |

---

## `profile_coldboot.py`

[`profile_coldboot.py`](profile_coldboot.py) runs `swebench-template.sh` with `RECREATE=1` and times it, giving `t0`..`t1`. During the build it follows the ateapi, atelet and worker logs (`kubectl logs -f`, [`LogStream`](profile_lifecycle.py)), because a busy ate-api-server can rotate its log file (kubelet rotates at 10 MiB) and `kubectl logs` only returns the current file. After the build it also scrapes once with `since = (t1 - t0) + 60` s, to cover pods created after the followers started. It merges the two, drops duplicate lines, and keeps records with `t0 - 2 ≤ _ts ≤ t1 + 2`. Within that window it keeps only the golden actor's records ([`golden_records`](profile_coldboot.py)): the ateom `Actor starting` line whose `ate.actor.name` is the template's uid gives the boot's `trace_id` and worker pod. ateapi and atelet lines are kept by that trace ID, and `Picked worker`, which is logged under a different trace, by that pod (the last one at or before `Actor starting`). Without the ateom line it falls back to the window and adds a note. This keeps out another template's golden boot in the same window, even one for the same image.

The subject is the template's **golden actor**: atespace `ate-golden`, name = the template's `metadata.uid`.

### What it reads, and how each record is tied to this build

| Component | Message | Fields used | How it is tied to this build | Raw |
|---|---|---|---|---|
| ateapi | `Added actor template to work queue` | `_ts` | window **+ template name substring** in `_raw` ([`build_segments`](profile_coldboot.py#L79-L112)) | [R8](log-appendix.md#r8) |
| ateapi | `… Booting from ActorTemplate spec` / `… Restoring from snapshot` | message text | **window only** ([`boot_path`](profile_coldboot.py#L212-L238)) | [R5](log-appendix.md#r5) / [R6](log-appendix.md#r6) |
| ateapi | `Picked worker` | `worker` → `worker_pod`, `node_name`; **`_ts` of the first one → `_picked_at`, the TTCE origin** | **window only** | [R4](log-appendix.md#r4) |
| atelet | `Image cache hit` / `Image cache miss` | `ref`, `digest` | window **+ digest** pinned by the template ([`cold_pull`](profile_coldboot.py#L181-L209)). The first outcome seen wins. | [R9](log-appendix.md#r9) / [R10](log-appendix.md#r10) |
| atelet | `Image pulled into layer cache` | `digest`, `layers`, `took` (ns) | window **+ digest** | [R11](log-appendix.md#r11) |
| OCILayerService adapter | `PrepareImage timing breakdown` (Go log text, 1 s timestamps, no trace ID) | `resolve_ms`, `stat_ms`, `prepare_ms`, `view_ms`, `listable_ms`, `wrapper_ms`, `total_ms`, `layers` | window **+ digest** ([`adapter_records`](profile_lifecycle.py), [`atelet_streaming`](profile_lifecycle.py)) | [R12](log-appendix.md#r12) |
| atelet | `Image streamed via OCILayerService` (Debug) | `duration` (ns), `layers` | window **+ digest** | [R14](log-appendix.md#r14) |
| atelet | `… falling back to cache` (5 OCILayerService variants) | `err` | window **+ digest** | [R15](log-appendix.md#r15) |
| OCILayerService adapter | `Cached streaming lease layers missing on disk; re-preparing layers` | — | window **+ digest** | [R16](log-appendix.md#r16) |
| ateom | `Actor starting` → `Actor started` (boot window) | `_ts`, labels | **`ate.actor.name == template uid`** ([`ateom_timeline`](profile_lifecycle.py#L183-L227)) | [R18](log-appendix.md#r18) / [R19](log-appendix.md#r19) |
| ateom | `Actor checkpointing` → `Actor checkpointed` (suspend window) | `_ts`, labels | same | [R22](log-appendix.md#r22) / [R23](log-appendix.md#r23) |
| ateom | `About to run runsc <verb>` | `container`, `_ts` | inside that window **+ same worker pod**, skipping any record labeled with a different actor | [R24](log-appendix.md#r24) |
| ateom | `Readyz reached 200` | `container`, `elapsed` (ns) | same | [R25](log-appendix.md#r25) |
| actor stdout | `workload start` / `workload end`, `trigger: boot` | `_ts`, `trigger`, `elapsed_us` | window **+ `ate.actor.name == template uid`**; first record per (message, trigger) wins ([`workload`](profile_lifecycle.py#L438-L458)) | [R26](log-appendix.md#r26) / [R27](log-appendix.md#r27) |

Some values come from the API instead of logs: `createTime` and `goldenSnapshotStatus.takeGoldenSnapshotAt` from `kubectl-ate get actor-template -o json`, the compressed image size from `crane manifest`, and atelet's `--oci-layer-service-socket` / `--log-level` from the DaemonSet args.

---

## `profile_lifecycle.py`

[`profile_lifecycle.py`](profile_lifecycle.py) runs four steps: create, resume #1, suspend, resume #2. Each step runs `kubectl-ate <verb> actor <name> -a <atespace> --trace`. It takes the trace ID from stderr (`Trace ID: <hex>`, in [`ate()`](profile_lifecycle.py)) and records the call's wall-clock span. For the whole run it follows the ateapi, atelet and worker logs (`kubectl logs -f`, `LogStream`), so a log rotation during a long step loses nothing. After each resume, with `--workload-timeout` > 0, it waits until the followed worker log shows this actor's `workload end` (logged at or after the call's start), or until the timeout, which adds a note to the step. Then, and after create and suspend, it waits `--settle` (3 s), and [`collect`](profile_lifecycle.py) merges the followed records with a per-step scrape (short `--since`), which covers pods created after the followers started.

The subject is the actor the script creates: default name `prof-<unix seconds>`, or `--actor`.

### What it reads, and how each record is tied to this actor

| Component | Message | Fields used | How it is tied to this step/actor | Raw |
|---|---|---|---|---|
| ateapi | `Handle RPC` where `method` ends in `CreateActor` / `ResumeActor` / `SuspendActor` | `elapsed-time` (Go duration string); **`_ts − elapsed-time` → `_rpc_start`, a TTCE origin** | **trace ID** substring in `_raw` ([`for_trace`](profile_lifecycle.py#L152-L156)) | [R1](log-appendix.md#r1) / [R2](log-appendix.md#r2) / [R3](log-appendix.md#r3) |
| ateapi | `Picked worker` | `worker` → `worker_pod`, `node_name` (used for same-node vs. cross-node); **`_ts` of the first one → `_picked_at`, a TTCE origin** | **trace ID** ([`ateapi_facts`](profile_lifecycle.py#L396-L435)) | [R4](log-appendix.md#r4) |
| ateapi | `… Restoring from snapshot` / `… Booting from ActorTemplate spec` | message text | **trace ID** | [R6](log-appendix.md#r6) / [R5](log-appendix.md#r5) |
| ateapi | `FinalizeSuspended store call durations` | `get_actor`, `release_worker`, `refetch_actor`, `release_snapshot`, `update_actor`, `total` (ns) | **trace ID** | [R7](log-appendix.md#r7) |
| atelet | `Image cache hit` / `Image cache miss`, `Image pulled into layer cache` | `ref`, `digest`, `layers`, `took` | **trace ID** ([`atelet_images`](profile_lifecycle.py#L250-L282)) | [R9](log-appendix.md#r9) / [R10](log-appendix.md#r10) / [R11](log-appendix.md#r11) |
| atelet | `Image streamed via OCILayerService`, fallbacks | as in the coldboot table | **trace ID** (`digests=None`, so no digest filter) | [R14](log-appendix.md#r14) / [R15](log-appendix.md#r15) |
| OCILayerService adapter | `PrepareImage timing breakdown`, re-prepare | as in the coldboot table | **the call's span ± 1 s** (no trace ID) | [R13](log-appendix.md#r13) / [R16](log-appendix.md#r16) |
| atelet | `Restore timing breakdown` | `ate.actor.restore.duration.{volume_mount,manifest_fetch,sandbox_assets,download,oci_unpack,ateom_restore,total}` (s) | **trace ID**, last record wins ([`atelet_restore`](profile_lifecycle.py#L230-L247)) | [R17](log-appendix.md#r17) |
| ateom | `Actor starting`/`started` (resume #1 if it boots), `restoring`/`restored` (resume #1/#2), `checkpointing`/`checkpointed` (suspend) | `_ts`, labels | records from the **call's start − 1 s to collection time + 1 s** (after any workload wait and the settle), then **`ate.actor.name == actor`** | [R18](log-appendix.md#r18)–[R23](log-appendix.md#r23) |
| ateom | `About to run runsc <verb>`, `Readyz reached 200` | `container`, `_ts`, `elapsed` | inside that window **+ same worker pod**, skipping records labeled with a different actor | [R24](log-appendix.md#r24) / [R25](log-appendix.md#r25) |
| actor stdout | `workload start` / `workload end`, `trigger: resume` | `_ts`, `trigger`, `elapsed_us` | the same window as ateom, then **`ate.actor.name == actor`**; first record per (message, trigger) wins ([`workload`](profile_lifecycle.py)) | [R26](log-appendix.md#r26) / [R27](log-appendix.md#r27) |

---

## TTCE: how the workload rows are computed

[`workload(records, actor, t_zero, origin)`](profile_lifecycle.py#L438-L458) finds the actor's first `workload start` and `workload end` per trigger, and reports each one's `_ts − t_zero`. The JSON `workload` key holds the same values.

| Script / step | Origin (`t_zero`) | Origin's source record | Rows printed |
|---|---|---|---|
| coldboot: golden actor boot | `_picked_at` | ateapi `Picked worker` (first in window) | `picked worker -> workload start (boot)`, `picked worker -> workload end (boot)`, `workload elapsed in sandbox (boot)` |
| lifecycle: resume #1, resume #2 | `_picked_at` | ateapi `Picked worker` (first under the call's trace) | `picked worker -> workload start (resume)`, `picked worker -> workload end (resume)` |
| lifecycle: resume #1, resume #2 | `_rpc_start` = `Handle RPC._ts − elapsed-time` | ateapi `Handle RPC` (`ResumeActor`) | `rpc start -> workload start (resume)`, `rpc start -> workload end (resume)`, `workload elapsed in sandbox (resume)` |

What each origin covers:
- **`picked worker`** starts after scheduling. It covers the bind and actor-update DB writes, volume attach, the atelet connection, the image fetch or stream, sandbox create/restore, and the workload script reaching its first `mark`.
- **`rpc start`** also includes the actor lookup, lease, and scheduling before `Picked worker`. It excludes client network time, auth and gRPC decoding, which happen before the interceptor's clock starts.
- **`workload elapsed in sandbox`** is the script's own `EPOCHREALTIME` difference around the `cat` (or `compileall`). It shares no clock with the other rows.

**`picked worker` is the common start point.** Cold boots, resume #1 and resume #2 all have it, so TTCE summaries measure every row from `_picked_at`, which is what lets resumes #1 and #2 be pooled and compared with boots. `rpc start` is reported for resumes only: the golden boot has no `Handle RPC` record, because the reconciler calls `ResumeActor` inside ate-api-server.

### Caveats specific to the workload rows
- **Two clocks.** The origin is stamped by the ate-api-server node's clock, the workload lines by the worker node's clock (ateom reads them). Cross-node NTP skew is usually sub-millisecond on GCE, but it enters every TTCE row.
- **`time` is when ateom reads the line, not when bash printed it.** Pipe and goroutine scheduling add a small delay, typically well under a millisecond.
- **Resume detection delay.** After a FULL restore, the script notices the clock jump after its current `sleep 0.05` ends, so `workload start (resume)` can trail `Actor restored` by up to about 50 ms (about 10 ms in earlier runs, whose loop used `read -t 0.01`).
- **The workload wait caps collection.** In `profile_lifecycle.py`, a resume whose `workload end` doesn't appear within `--workload-timeout` gets a note, and the next step starts anyway, so that workload may be suspended mid-run. With `--workload-timeout 0`, a workload line logged after the settle is dropped and its row is missing.
