# Appendix: raw log lines

Raw lines for every message in the tables of [log-sources.md](log-sources.md).


Captured with `kubectl logs -f --prefix` during a run on 2026-10-07 (`matplotlib__matplotlib-23476`, pull arm then stream arm). Each line is copied verbatim from `ateapi.log`, `atelet.log` or `ateom.log`, with the `[pod/…]` prefix moved to the **Pod** line. Environment-specific values are replaced with placeholders: `<project>`, `<cluster>`, `<bucket>`, `<node>`, `<pod-ip>`, and `<streamer>` for the image streamer name.

### R1
**ateapi · `Handle RPC` · CreateActor** — Pod: `ate-api-server-6f45655c8c-gv26d/ate-api-server`
```json
{"time":"2026-10-07T16:50:41.812259724Z","level":"INFO","msg":"Handle RPC","method":"/ateapi.Control/CreateActor","req":{"actor":{"metadata":{"atespace":"ate-prof-pull","name":"prof-1791391841"},"actor_template":{"atespace":"ate-prof-pull","name":"swebench-matplotlib-matplotlib-23476"}}},"resp":{"metadata":{"atespace":"ate-prof-pull","name":"prof-1791391841","uid":"61a22af9-16b6-43da-8abb-d9daebcf9547","version":1,"create_time":{"seconds":1791391841,"nanos":810124142},"update_time":{"seconds":1791391841,"nanos":810124142}},"actor_template":{"atespace":"ate-prof-pull","name":"swebench-matplotlib-matplotlib-23476"},"status":{"state":4}},"err":null,"elapsed-time":"2.603446ms","principal":{"ID":"system:serviceaccount:ate-system:ate-client","Kind":"jwt","Issuer":"https://container.googleapis.com/v1/projects/<project>/locations/us-west1-c/clusters/<cluster>"},"trace_id":"4d00d87762d7732771c0b5d324386d1d","span_id":"ea54b8ee2a063984","trace_flags":"01"}
```

### R2
**ateapi · `Handle RPC` · ResumeActor** — Pod: `ate-api-server-6f45655c8c-gv26d/ate-api-server`
```json
{"time":"2026-10-07T16:50:47.364793219Z","level":"INFO","msg":"Handle RPC","method":"/ateapi.Control/ResumeActor","req":{"actor":{"atespace":"ate-prof-pull","name":"prof-1791391841"}},"resp":{"actor":{"metadata":{"atespace":"ate-prof-pull","name":"prof-1791391841","uid":"61a22af9-16b6-43da-8abb-d9daebcf9547","version":3,"create_time":{"seconds":1791391841,"nanos":810124142},"update_time":{"seconds":1791391847,"nanos":362345019}},"actor_template":{"atespace":"ate-prof-pull","name":"swebench-matplotlib-matplotlib-23476"},"status":{"state":2,"worker_assignment":{"worker":{"name":"b6332dbb-40db-4dfe-b5da-8bf17293d4fc"},"worker_namespace":"ate-profiling","worker_pool":"profiling-pull","worker_pod":"profiling-pull-775c95d74f-74r88","worker_pod_uid":"b6332dbb-40db-4dfe-b5da-8bf17293d4fc","worker_pod_ip":"<pod-ip>"},"current_actor_template_uid":"88c9fb30-f369-4380-b888-986e7369ef1a"}},"resumed":true},"err":null,"elapsed-time":"465.703717ms","principal":{"ID":"system:serviceaccount:ate-system:ate-client","Kind":"jwt","Issuer":"https://container.googleapis.com/v1/projects/<project>/locations/us-west1-c/clusters/<cluster>"},"trace_id":"59dcd0ba8205abf28e6111c06b5fd94c","span_id":"4f989d91e764e51e","trace_flags":"01"}
```

### R3
**ateapi · `Handle RPC` · SuspendActor** — Pod: `ate-api-server-6f45655c8c-gv26d/ate-api-server`
```json
{"time":"2026-10-07T16:50:54.886947723Z","level":"INFO","msg":"Handle RPC","method":"/ateapi.Control/SuspendActor","req":{"actor":{"atespace":"ate-prof-pull","name":"prof-1791391841"}},"resp":{"actor":{"metadata":{"atespace":"ate-prof-pull","name":"prof-1791391841","uid":"61a22af9-16b6-43da-8abb-d9daebcf9547","version":5,"create_time":{"seconds":1791391841,"nanos":810124142},"update_time":{"seconds":1791391854,"nanos":885054314}},"actor_template":{"atespace":"ate-prof-pull","name":"swebench-matplotlib-matplotlib-23476"},"status":{"state":4,"external_snapshot":{"snapshot_uri":"gs://<bucket>/ate-prof-pull/atespaces/ate-prof-pull/actors/61a22af9-16b6-43da-8abb-d9daebcf9547/snapshots/100a9dc7-1e78-46b6-b950-9aca5aec2a65","content_scope":1},"current_actor_template_uid":"88c9fb30-f369-4380-b888-986e7369ef1a"}}},"err":null,"elapsed-time":"588.119409ms","principal":{"ID":"system:serviceaccount:ate-system:ate-client","Kind":"jwt","Issuer":"https://container.googleapis.com/v1/projects/<project>/locations/us-west1-c/clusters/<cluster>"},"trace_id":"e11b544e7b1248182f369d5a1c2ee2f7","span_id":"6eb76445201afa0d","trace_flags":"01"}
```

### R4
**ateapi · `Picked worker`** (pull golden build) — Pod: `ate-api-server-6f45655c8c-p5sbx/ate-api-server`
```json
{"time":"2026-10-07T16:48:42.226962468Z","level":"INFO","msg":"Picked worker","worker":"metadata:{name:\"b6332dbb-40db-4dfe-b5da-8bf17293d4fc\"  uid:\"a22efe50-a210-4d52-813f-199b8b22b639\"  version:2  create_time:{seconds:1791390765  nanos:180885109}  update_time:{seconds:1791390766  nanos:160533484}}  worker_namespace:\"ate-profiling\"  worker_pool:\"profiling-pull\"  worker_pod:\"profiling-pull-775c95d74f-74r88\"  worker_pod_uid:\"b6332dbb-40db-4dfe-b5da-8bf17293d4fc\"  node_name:\"<node>\"  ip:\"<pod-ip>\"  sandbox_class:\"gvisor\"  labels:{key:\"workload\"  value:\"profiling-pull\"}  status:{state:WORKER_STATE_ACTIVE  capacity:{resources:{limits:{name:\"cpu\"  quantity:\"1\"}  limits:{name:\"memory\"  quantity:\"2Gi\"}}  actors:1}}","trace_id":"9011e5cc6658766391a285e39e1c5a20","span_id":"78b47e59d429eb59","trace_flags":"00"}
```

### R5
**ateapi · `… Booting from ActorTemplate spec`** (pull golden build) — Pod: `ate-api-server-6f45655c8c-p5sbx/ate-api-server`
```json
{"time":"2026-10-07T16:48:42.235028244Z","level":"INFO","msg":"Actor has no snapshot; ActorTemplate has no golden snapshot; Booting from ActorTemplate spec","trace_id":"878a2249db18aefd8907b6e059412f0a","span_id":"05e2bb03da3d4afc","trace_flags":"00"}
```

### R6
**ateapi · `… Restoring from snapshot`** (lifecycle resume #1) — Pod: `ate-api-server-6f45655c8c-gv26d/ate-api-server`
```json
{"time":"2026-10-07T16:50:46.907420997Z","level":"INFO","msg":"Actor has durable snapshot; Restoring from snapshot","trace_id":"59dcd0ba8205abf28e6111c06b5fd94c","span_id":"63c4efe61763922c","trace_flags":"01"}
```

### R7
**ateapi · `FinalizeSuspended store call durations`** (pull golden actor) — Pod: `ate-api-server-6f45655c8c-gv26d/ate-api-server`
```json
{"time":"2026-10-07T16:50:32.93987529Z","level":"INFO","msg":"FinalizeSuspended store call durations","actor":{"type":"*ateapipb.Actor","atespace":"ate-golden","name":"88c9fb30-f369-4380-b888-986e7369ef1a"},"total":5958943,"get_actor":309404,"release_worker":4329502,"refetch_actor":239399,"release_snapshot":6783,"update_actor":1050827,"trace_id":"68e813631520385129253779d35f3d57","span_id":"07479d845773dd52","trace_flags":"00"}
```

### R8
**ateapi · `Added actor template to work queue`** — Pod: `ate-api-server-6f45655c8c-p5sbx/ate-api-server`
```json
{"time":"2026-10-07T16:48:42.211499786Z","level":"INFO","msg":"Added actor template to work queue","ActorTemplate":"ate-prof-pull/swebench-matplotlib-matplotlib-23476"}
```

### R9
**atelet · `Image cache hit`** (pause image) — Pod: `atelet-vf1438f5987-dw5w8/atelet`
```json
{"time":"2026-10-07T16:48:42.267726832Z","level":"INFO","msg":"Image cache hit","ref":"registry.k8s.io/pause:3.10.2@sha256:f548e0e8e3dc1896ca956272154dde3314e8cc4fde0a57577ee9fa1c63f5baf4","digest":"sha256:f548e0e8e3dc1896ca956272154dde3314e8cc4fde0a57577ee9fa1c63f5baf4","trace_id":"878a2249db18aefd8907b6e059412f0a","span_id":"1c661c145e952c65","trace_flags":"00"}
```

### R10
**atelet · `Image cache miss`** — Pod: `atelet-vf1438f5987-dw5w8/atelet`
```json
{"time":"2026-10-07T16:48:42.267512945Z","level":"INFO","msg":"Image cache miss","ref":"us-west1-docker.pkg.dev/<project>/swebench-mirror/swebench-verified@sha256:64bb043b17a25dbeeba766c82d1d5f66ca446ca32da12c88418ded9e0d79cfdc","digest":"sha256:64bb043b17a25dbeeba766c82d1d5f66ca446ca32da12c88418ded9e0d79cfdc","trace_id":"878a2249db18aefd8907b6e059412f0a","span_id":"ea2554f19efecfb6","trace_flags":"00"}
```

### R11
**atelet · `Image pulled into layer cache`** — Pod: `atelet-vf1438f5987-dw5w8/atelet`
```json
{"time":"2026-10-07T16:50:12.064597276Z","level":"INFO","msg":"Image pulled into layer cache","digest":"sha256:64bb043b17a25dbeeba766c82d1d5f66ca446ca32da12c88418ded9e0d79cfdc","layers":10,"took":89797006842,"trace_id":"878a2249db18aefd8907b6e059412f0a","span_id":"ea2554f19efecfb6","trace_flags":"00"}
```

### R12
**atelet · `PrepareLayers timing breakdown`** (stream golden build, cold) — Pod: `atelet-vf1438f5987-frmw9/atelet`
```json
{"time":"2026-10-07T16:52:24.614465269Z","level":"INFO","msg":"PrepareLayers timing breakdown","streamer":"<streamer>","image":"us-west1-docker.pkg.dev/<project>/swebench-mirror/swebench-verified@sha256:64bb043b17a25dbeeba766c82d1d5f66ca446ca32da12c88418ded9e0d79cfdc","digest":"sha256:64bb043b17a25dbeeba766c82d1d5f66ca446ca32da12c88418ded9e0d79cfdc","layers":10,"resolve_ms":373.928,"stat_ms":15.985,"prepare_ms":1894.955,"view_ms":35.744,"listable_ms":5.992,"wrapper_ms":2.477,"total_ms":2331.164,"trace_id":"3badba5a63d6709254bb24c383c595a6","span_id":"eb7cc6a06d326151","trace_flags":"00"}
```

### R13
**atelet · `PrepareLayers timing breakdown`** (stream lifecycle resume, stat hit) — Pod: `atelet-vf1438f5987-frmw9/atelet`
```json
{"time":"2026-10-07T16:52:59.13913479Z","level":"INFO","msg":"PrepareLayers timing breakdown","streamer":"<streamer>","image":"us-west1-docker.pkg.dev/<project>/swebench-mirror/swebench-verified@sha256:64bb043b17a25dbeeba766c82d1d5f66ca446ca32da12c88418ded9e0d79cfdc","digest":"sha256:64bb043b17a25dbeeba766c82d1d5f66ca446ca32da12c88418ded9e0d79cfdc","layers":10,"resolve_ms":0,"stat_ms":6.659,"prepare_ms":0,"view_ms":25.066,"listable_ms":5.321,"wrapper_ms":2.026,"total_ms":40.059,"trace_id":"298e7eb1ce8dadcde4f6e863ad72e4b7","span_id":"4aea7de7f95200f9","trace_flags":"01"}
```

### R14
**atelet · `Image streamed`** (Debug level) — Pod: `atelet-vf1438f5987-frmw9/atelet`
```json
{"time":"2026-10-07T16:52:24.614553329Z","level":"DEBUG","msg":"Image streamed","image":"us-west1-docker.pkg.dev/<project>/swebench-mirror/swebench-verified@sha256:64bb043b17a25dbeeba766c82d1d5f66ca446ca32da12c88418ded9e0d79cfdc","streamer":"<streamer>","layers":10,"duration":2332981208,"trace_id":"3badba5a63d6709254bb24c383c595a6","span_id":"eb7cc6a06d326151","trace_flags":"00"}
```

### R15
**atelet · `… falling back to cache`** — **Not observed.** Logged only when streaming is unavailable, the image is ineligible, or preparing/resolving it fails. Streaming succeeded in this run. The five messages the script matches ([`STREAM_FALLBACKS`](profile_lifecycle.py#L292-L298)):
- `Image not streamable; falling back to cache`
- `Image streaming layer preparation failed; falling back to cache`
- `Image streaming unavailable; falling back to cache`
- `Error evaluating image streaming eligibility; falling back to cache`
- `Failed to resolve image config for streamed image; falling back to cache`

### R16
**atelet · `Cached streaming lease layers missing on disk; re-preparing layers`** — **Not observed.** Logged only when a cached streaming lease exists but its layer directories are gone from disk. The node's lease and layers were intact in this run.

### R17
**atelet · `Restore timing breakdown`** (pull lifecycle resume #1) — Pod: `atelet-vf1438f5987-dw5w8/atelet`
```json
{"time":"2026-10-07T16:50:47.36055291Z","level":"INFO","msg":"Restore timing breakdown","ate.atespace":"ate-prof-pull","ate.actor.name":"prof-1791391841","ate.actor.uid":"61a22af9-16b6-43da-8abb-d9daebcf9547","ate.template.atespace":"ate-prof-pull","ate.template.name":"swebench-matplotlib-matplotlib-23476","ate.snapshot.scope":"full","ate.snapshot.kind":"golden","ate.sandbox.class":"gvisor","ate.actor.restore.duration.volume_mount":0.00000115,"ate.actor.restore.duration.manifest_fetch":0.060613243,"ate.actor.restore.duration.sandbox_assets":0.000090007,"ate.actor.restore.duration.download":0.062245902,"ate.actor.restore.duration.oci_unpack":0.001223442,"ate.actor.restore.duration.ateom_restore":0.324004254,"ate.actor.restore.duration.total":0.45178428,"trace_id":"59dcd0ba8205abf28e6111c06b5fd94c","span_id":"900b971165b69316","trace_flags":"01"}
```

### R18
**ateom · `Actor starting`** (pull golden actor) — Pod: `profiling-pull-775c95d74f-74r88/ateom`
```json
{"logging.googleapis.com/labels":{"ate.actor.name":"88c9fb30-f369-4380-b888-986e7369ef1a","ate.actor.uid":"cbf80e06-e986-416a-b226-1a2d970504a4","ate.atespace":"ate-golden","ate.template.atespace":"ate-prof-pull","ate.template.name":"swebench-matplotlib-matplotlib-23476"},"message":"Actor starting","span_id":"d3aab0648c128bcc","time":"2026-10-07T16:50:12.068130331Z","trace_flags":"00","trace_id":"878a2249db18aefd8907b6e059412f0a"}
```

### R19
**ateom · `Actor started`** (pull golden actor) — Pod: `profiling-pull-775c95d74f-74r88/ateom`
```json
{"logging.googleapis.com/labels":{"ate.actor.name":"88c9fb30-f369-4380-b888-986e7369ef1a","ate.actor.uid":"cbf80e06-e986-416a-b226-1a2d970504a4","ate.atespace":"ate-golden","ate.template.atespace":"ate-prof-pull","ate.template.name":"swebench-matplotlib-matplotlib-23476"},"message":"Actor started","span_id":"d3aab0648c128bcc","time":"2026-10-07T16:50:12.330372182Z","trace_flags":"00","trace_id":"878a2249db18aefd8907b6e059412f0a"}
```

### R20
**ateom · `Actor restoring`** (pull lifecycle resume #1) — Pod: `profiling-pull-775c95d74f-74r88/ateom`
```json
{"logging.googleapis.com/labels":{"ate.actor.name":"prof-1791391841","ate.actor.uid":"61a22af9-16b6-43da-8abb-d9daebcf9547","ate.atespace":"ate-prof-pull","ate.template.atespace":"ate-prof-pull","ate.template.name":"swebench-matplotlib-matplotlib-23476"},"message":"Actor restoring","span_id":"72ca55fea1a190f0","time":"2026-10-07T16:50:47.037394106Z","trace_flags":"01","trace_id":"59dcd0ba8205abf28e6111c06b5fd94c"}
```

### R21
**ateom · `Actor restored`** (pull lifecycle resume #1) — Pod: `profiling-pull-775c95d74f-74r88/ateom`
```json
{"logging.googleapis.com/labels":{"ate.actor.name":"prof-1791391841","ate.actor.uid":"61a22af9-16b6-43da-8abb-d9daebcf9547","ate.atespace":"ate-prof-pull","ate.template.atespace":"ate-prof-pull","ate.template.name":"swebench-matplotlib-matplotlib-23476"},"message":"Actor restored","span_id":"72ca55fea1a190f0","time":"2026-10-07T16:50:47.35949409Z","trace_flags":"01","trace_id":"59dcd0ba8205abf28e6111c06b5fd94c"}
```

### R22
**ateom · `Actor checkpointing`** (pull golden actor) — Pod: `profiling-pull-775c95d74f-74r88/ateom`
```json
{"logging.googleapis.com/labels":{"ate.actor.name":"88c9fb30-f369-4380-b888-986e7369ef1a","ate.actor.uid":"cbf80e06-e986-416a-b226-1a2d970504a4","ate.atespace":"ate-golden","ate.template.atespace":"ate-prof-pull","ate.template.name":"swebench-matplotlib-matplotlib-23476"},"message":"Actor checkpointing","span_id":"15c0333b714c860c","time":"2026-10-07T16:50:32.354315573Z","trace_flags":"00","trace_id":"792c33b648d8938b99f6c866c70dfc83"}
```

### R23
**ateom · `Actor checkpointed`** (pull golden actor) — Pod: `profiling-pull-775c95d74f-74r88/ateom`
```json
{"logging.googleapis.com/labels":{"ate.actor.name":"88c9fb30-f369-4380-b888-986e7369ef1a","ate.actor.uid":"cbf80e06-e986-416a-b226-1a2d970504a4","ate.atespace":"ate-golden","ate.template.atespace":"ate-prof-pull","ate.template.name":"swebench-matplotlib-matplotlib-23476"},"message":"Actor checkpointed","span_id":"15c0333b714c860c","time":"2026-10-07T16:50:32.567305441Z","trace_flags":"00","trace_id":"792c33b648d8938b99f6c866c70dfc83"}
```

### R24
**ateom · `About to run runsc <verb>`** (pull golden boot) — Pod: `profiling-pull-775c95d74f-74r88/ateom`
```json
{"time":"2026-10-07T16:50:12.085718313Z","level":"INFO","msg":"About to run runsc create","container":"_pause","trace_id":"878a2249db18aefd8907b6e059412f0a","span_id":"d3aab0648c128bcc","trace_flags":"00"}
```

### R25
**ateom · `Readyz reached 200`** — **Not observed.** Logged by [`internal/readyz/readyz.go:127`](../internal/readyz/readyz.go#L127) only when a container defines a readyz probe. The SWE-bench template defines none.

### R26
**actor stdout · `workload start`** (stream golden actor, `trigger: boot`, captured 2026-10-09 from `file-read-n5` rep 4) — Pod: `profiling-stream-588f9cd8f4-zrfxr/ateom`
```json
{"file":"/testbed/lib/matplotlib/pyplot.py","logging.googleapis.com/labels":{"ate.actor.container.name":"testbed","ate.actor.name":"597a6cdc-ea0e-4676-9aad-8863ba2afa60","ate.actor.uid":"ff8d4ae3-8943-4f27-b37e-3bb9fb3093b5","ate.atespace":"ate-golden","ate.template.atespace":"ate-prof-stream","ate.template.name":"swebench-matplotlib-matplotlib-23476"},"msg":"workload start","time":"2026-10-09T19:00:38.113563068Z","trigger":"boot","workload":"readfile"}
```

### R27
**actor stdout · `workload end`** (stream lifecycle resume, `trigger: resume`, captured 2026-10-09 from `file-read-n5` rep 4) — Pod: `profiling-stream-588f9cd8f4-hvq9t/ateom`
```json
{"bytes":112924,"elapsed_us":20779,"file":"/testbed/lib/matplotlib/pyplot.py","logging.googleapis.com/labels":{"ate.actor.container.name":"testbed","ate.actor.name":"prof-1791572463","ate.actor.uid":"6f7837cc-015e-454f-be18-d7a83b14ae17","ate.atespace":"ate-prof-stream","ate.template.atespace":"ate-prof-stream","ate.template.name":"swebench-matplotlib-matplotlib-23476"},"msg":"workload end","time":"2026-10-09T19:01:09.25751085Z","trigger":"resume","workload":"readfile"}
```

---

## Findings from the live cluster samples

1. **Golden-actor builds carry trace IDs, but `Picked worker` is in a different trace.** In the pull golden build, ateapi `Booting from ActorTemplate spec` ([R5](#r5)), atelet `Image cache miss`/`pulled` ([R10](#r10), [R11](#r11)) and ateom `Actor starting`/`runsc` ([R18](#r18), [R24](#r24)) all share `trace_id 878a2249…`. The `Picked worker` line 8 ms earlier ([R4](#r4)) has `9011e5cc…`, and that trace ID appears nowhere else. The stream golden build shows the same split (`Picked worker` `0a1a7c64…` vs. atelet `PrepareLayers` `3badba5a…`).
   - So `profile_coldboot.py` could attribute by trace ID for ateapi boot-path and atelet lines:
     1. Read `trace_id` from the ateom `Actor starting` envelope where `ate.actor.name == uid`.
     2. Run `for_trace` over the ateapi and atelet records.
   - `Picked worker` would still need another key on golden builds, e.g. the `worker_pod` that ateom's records came from. This matters more now, since `Picked worker` is the coldboot TTCE origin.
   - For lifecycle resumes, `Picked worker` does share the call's trace ID: the resume #1 `Picked worker` at 16:50:46.903 has `59dcd0ba…`, the same as [R2](#r2), [R6](#r6), [R17](#r17) and [R20](#r20). So the lifecycle `_picked_at` origin is tied to the right call.
2. **The checkpoint is a separate trace.** Golden `Actor checkpointing`/`checkpointed` use `792c33b6…`, not the boot trace. The FinalizeSuspended line for that golden actor ([R7](#r7)) has yet another, `68e81363…`.
3. **`Added actor template to work queue` has no trace ID** ([R8](#r8)). It carries only `ActorTemplate: "<atespace>/<name>"`, which is why `build_segments` matches it by substring.
4. **`trace_flags` tells the two sources apart.** Lifecycle calls made with `kubectl-ate --trace` are sampled (`"01"`); reconciler/golden work is `"00"`.
5. **The ateom envelope now has `trace_id`.** The `profile_lifecycle.py` docstrings say the lifecycle envelope "carries no trace ID", which is out of date for this build. Both scripts still select the window by actor name, which remains correct. Workload lines (shape C) do **not** carry a trace ID, so actor name is the only way to match them.
6. **ateapi logs rotate within hours.** `Handle RPC` logs full `req`/`resp` payloads, and `ListActors` is polled about once a second during template builds. That fills the log quickly. This is why both scripts scrape right after each operation, and why a long or concurrent run should stream logs to files (as this run did with `kubectl logs -f`), or read from Cloud Logging.
