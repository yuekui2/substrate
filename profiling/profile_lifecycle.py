#!/usr/bin/env python3
"""Profile one actor lifecycle: create -> boot -> suspend -> resume.

Drives kubectl-ate with --trace so every step has a trace ID, then joins the
logs of ate-api-server, atelet and the ateom worker pods on that ID. Only reads
telemetry Substrate already emits; nothing here requires a code change.

Phase sources (all verified against the tree):
  ate-api-server  "Handle RPC"                  elapsed-time, Go duration string
                  "Picked worker"               worker assignment
                  "Actor has ... snapshot"      boot path taken
                  "FinalizeSuspended store ..." slog.Duration, int64 ns
  atelet          "Image cache hit|miss"        per image reference
                  "Image pulled into layer cache" took, int64 ns: the registry pull
                  "Image streamed via OCILayerService" duration, int64 ns (Debug)
                  "... falling back to cache"   streaming declined or failed
  ocilayer adapter "PrepareImage timing breakdown" *_ms, float64 ms (Info, Go log text, no trace ID)
                  "Restore timing breakdown"    ate.actor.restore.duration.*, float64 s
  ateom worker    "Actor starting|started"      lifecycle envelope, RFC3339Nano
                  "Actor restoring|restored"
                  "Actor checkpointing|checkpointed"
                  "About to run runsc ..."      per-container step markers
                  "Readyz reached 200"          elapsed, int64 ns
atelet's checkpoint path records metrics only, so suspend granularity comes from
the ateom markers and the ateapi tail breakdown.
"""

import argparse
import contextlib
import json
import os
import re
import subprocess
import sys
import tempfile
import time
from datetime import datetime

ATE_SYSTEM = "ate-system"
# The OCILayerService adapter atelet streams through. Its pod has a second
# container (gcfsd), so `kubectl logs -l` needs the adapter's named.
ADAPTER_SELECTOR = "app=ate-ocilayer-adapter"
ADAPTER_CONTAINER = "ocilayer-riptide"
RESTORE_PREFIX = "ate.actor.restore.duration."
# The phase names atelet records, used to read the older bare-key spelling.
RESTORE_PHASES = (
    "volume_mount", "manifest_fetch", "sandbox_assets",
    "download", "oci_unpack", "ateom_restore", "total",
)
# Lifecycle pairs bounding an ateom window, in the order the markers appear.
ATEOM_WINDOWS = {
    "boot": ("Actor starting", "Actor started"),
    "suspend": ("Actor checkpointing", "Actor checkpointed"),
    "restore": ("Actor restoring", "Actor restored"),
}


def run(cmd, check=True):
    """Run a shell command and capture it.

    check=False is the norm here: a log scrape against a selector that matches
    no pods is an empty result, not a failure worth ending the run over.
    """
    p = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    if check and p.returncode != 0:
        raise SystemExit(f"command failed: {cmd}\n{p.stderr.strip()}")
    return p


def parse_ts(value):
    """RFC3339 with any fractional precision -> epoch seconds."""
    if not value:
        return None
    s = value.replace("Z", "+00:00")
    # fromisoformat caps the fraction at 6 digits; RFC3339Nano writes up to 9.
    s = re.sub(r"\.(\d{6})\d+", r".\1", s)
    try:
        return datetime.fromisoformat(s).timestamp()
    except ValueError:
        return None


def parse_go_duration(value):
    """'1.5s', '310ms', '644.935µs' -> float seconds."""
    if value is None:
        return None
    units = {"ns": 1e-9, "us": 1e-6, "µs": 1e-6, "ms": 1e-3, "s": 1.0, "m": 60.0, "h": 3600.0}
    total, matched = 0.0, False
    for num, unit in re.findall(r"([\d.]+)(ns|us|µs|ms|h|m|s)", str(value)):
        total += float(num) * units[unit]
        matched = True
    return total if matched else None


def ns_to_s(value):
    """slog.Duration serializes to int64 nanoseconds in JSON."""
    return float(value) / 1e9 if isinstance(value, (int, float)) else None


def msg_of(rec):
    """ateom lifecycle logs use 'message'; component slog logs use 'msg'."""
    return rec.get("msg") or rec.get("message") or ""


def actor_name_of(rec):
    """Actor name from a lifecycle envelope.

    The label group has two spellings and the choice is environmental, not
    historical: actorlog.LabelsKey writes logging.googleapis.com/labels on GCE,
    because that is the key Cloud Logging promotes into LogEntry.labels, and
    plain "labels" everywhere else. Both are read so this works against a kind
    cluster as well as GKE.

    The key inside is ateattr.ActorNameKey, which is the only spelling emitted.
    """
    labels = rec.get("labels") or rec.get("logging.googleapis.com/labels") or {}
    if not isinstance(labels, dict):
        return ""
    return labels.get("ate.actor.name") or ""


def parse_go_log(line):
    """One line from Go's default logger ("2026/10/19 17:01:56 INFO msg k=v ...")
    as a dict, or None. The adapter logs through slog's default handler, which
    writes this shape: a UTC timestamp to the second, the level, the unquoted
    message up to the first key=, then key=value attributes."""
    m = re.match(r"(\d{4})/(\d{2})/(\d{2}) (\d{2}:\d{2}:\d{2}(?:\.\d+)?) ([A-Z]+) (.*)", line)
    if not m:
        return None
    rest = m.group(6)
    attrs = re.search(r'\s[^\s="]+=', rest)
    msg = rest[:attrs.start()] if attrs else rest
    rec = {}
    for k, v in re.findall(r'([^\s=]+)=("(?:[^"\\]|\\.)*"|\S*)', rest[len(msg):]):
        if v.startswith('"'):
            try:
                v = json.loads(v)
            except json.JSONDecodeError:
                v = v[1:-1]
        else:
            for conv in (int, float):
                try:
                    v = conv(v)
                    break
                except ValueError:
                    pass
        rec[k] = v
    rec.update(time=f"{m.group(1)}-{m.group(2)}-{m.group(3)}T{m.group(4)}Z",
               level=m.group(5), msg=msg.strip())
    return rec


def parse_log_lines(text, prefix=False):
    """Parse `kubectl logs` output into records.

    Each record gains three synthetic keys the rest of the script reads:
    _pod (empty unless prefix, which is how a worker record is tied back to the
    pod that wrote it), _ts (epoch seconds) and _raw (the line as received, for
    substring matching a trace ID whatever key it was written under).

    Lines are JSON objects, or Go default-logger text from the OCILayerService
    adapter (parse_go_log). Anything else is dropped: kubectl's own prefix
    lines and other text are noise here, not errors.
    """
    records = []
    for line in text.splitlines():
        pod = ""
        if prefix:
            m = re.match(r"\[pod/([^/]+)/[^]]+\]\s*(.*)", line)
            if not m:
                continue
            pod, line = m.group(1), m.group(2)
        line = line.strip()
        if not line.startswith("{"):
            rec = parse_go_log(line)
            if rec is None:
                continue
        else:
            try:
                rec = json.loads(line)
            except json.JSONDecodeError:
                continue
        rec["_pod"] = pod
        rec["_ts"] = parse_ts(rec.get("time") or rec.get("timestamp"))
        rec["_raw"] = line
        records.append(rec)
    return records


def logs(selector, namespace, since, prefix=False, container=""):
    """Scrape the log records of every pod matching a label selector."""
    cmd = f"kubectl logs -l {selector} -n {namespace} --since={since}s --tail=-1"
    if container:
        cmd += f" -c {container}"
    if prefix:
        cmd += " --prefix"
    return parse_log_lines(run(cmd, check=False).stdout, prefix)


class LogStream:
    """Follow every pod matching a selector into a temp file while in use.

    `kubectl logs` returns only a pod's current log file, and kubelet rotates
    it at 10 MiB (containerLogMaxSize), so a busy component such as
    ate-api-server can rotate away a line between when it is written and a
    scrape at the end of a long step. Following the logs as they are written
    keeps those lines. `kubectl logs -f -l` only follows pods that exist when
    it starts, so merge the result with a scrape to cover pods created later.

    The follower appends to a named file that is read through a separate
    handle, so snapshot() can be called while it is still following.
    """

    def __init__(self, selector, namespace, prefix=False, container=""):
        self.selector, self.namespace, self.prefix = selector, namespace, prefix
        self.container = container
        self._path = self._proc = None

    def __enter__(self):
        fd, self._path = tempfile.mkstemp(prefix="logstream-", suffix=".log")
        os.close(fd)
        cmd = ["kubectl", "logs", "-f", "-l", self.selector, "-n", self.namespace,
               "--since=1s", "--tail=-1", "--max-log-requests=50"]
        if self.container:
            cmd += ["-c", self.container]
        if self.prefix:
            cmd.append("--prefix")
        with open(self._path, "a") as out:
            self._proc = subprocess.Popen(cmd, stdout=out, stderr=subprocess.DEVNULL, text=True)
        return self

    def __exit__(self, *exc):
        self.stop()
        os.unlink(self._path)

    def stop(self):
        """Stop following; safe to call more than once."""
        if self._proc and self._proc.poll() is None:
            self._proc.terminate()
            try:
                self._proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self._proc.kill()
                self._proc.wait()

    def snapshot(self):
        """Everything captured so far, without stopping the follower."""
        with open(self._path) as f:
            return parse_log_lines(f.read(), self.prefix)

    def records(self):
        """Stop following and return everything captured."""
        self.stop()
        return self.snapshot()


def merge_records(*sources):
    """Union record lists, dropping lines seen in more than one source."""
    seen, out = set(), []
    for records in sources:
        for r in records:
            key = (r["_pod"], r["_raw"])
            if key not in seen:
                seen.add(key)
                out.append(r)
    return out


def for_trace(records, trace_id):
    """Match on the raw line: the trace ID is spelled trace_id in component
    slog records and ate.dev/trace-id in the ateapi RPC log."""
    hits = [r for r in records if trace_id and trace_id in r["_raw"]]
    return sorted(hits, key=lambda r: r["_ts"] or 0.0)


def ate(args, sub, retries=0):
    """Run a kubectl-ate subcommand, returning (wall_seconds, trace_id, stdout).

    A worker record can outlive its pod, and ateapi fails the call rather than
    picking again, so retry that one error: the next assignment may be live.
    Each attempt is timed on its own, so the returned latency is the attempt
    that did the work, not the dead ones before it.
    """
    cmd = f"{args.kubectl_ate} {sub} -a {args.atespace} --trace"
    for attempt in range(retries + 1):
        t0, t_wall = time.perf_counter(), time.time()
        p = run(cmd, check=False)
        wall = time.perf_counter() - t0
        m = re.search(r"Trace ID:\s*([0-9a-f]+)", p.stderr)
        if p.returncode == 0:
            return wall, (m.group(1) if m else ""), (t_wall, time.time())
        if attempt < retries and "worker pod not found" in p.stderr:
            print(f"  retrying ({attempt + 1}/{retries}): assigned worker pod is gone")
            time.sleep(2)
            continue
        raise SystemExit(f"command failed: {cmd}\n{p.stderr.strip()}")



def ateom_timeline(records, window, actor):
    """Inter-marker deltas inside an ateom lifecycle window.

    The worker's lifecycle logs name the actor but carry no trace ID, so the
    window is selected by actor and the step markers inside it by time and pod.
    A pod multiplexes actors, so restricting to the pod that logged the window
    is what keeps a neighbour's runsc step out of this breakdown.

    Reports every marker the worker emitted rather than a fixed phase list, so
    the breakdown stays as granular as the logs allow.
    """
    start_msg, end_msg = ATEOM_WINDOWS[window]
    start = next((r for r in records
                  if msg_of(r) == start_msg and actor_name_of(r) == actor), None)
    end = next((r for r in reversed(records)
                if msg_of(r) == end_msg and actor_name_of(r) == actor), None)
    if not start or not end or start["_ts"] is None or end["_ts"] is None:
        return {}, {}

    pod = start.get("_pod", "")
    marks = []
    for r in records:
        if r["_ts"] is None or not (start["_ts"] <= r["_ts"] <= end["_ts"]):
            continue
        if r.get("_pod") != pod:
            continue
        m = msg_of(r)
        named = actor_name_of(r)
        if named and named != actor:
            continue
        if m in (start_msg, end_msg) or m.startswith("About to run runsc ") or m == "Readyz reached 200":
            label = m.replace("About to run ", "")
            if r.get("container"):
                label = f"{label}[{r['container']}]"
            marks.append((label, r["_ts"], r))

    phases = {}
    for (label, ts, _), (_, next_ts, _) in zip(marks, marks[1:]):
        # Markers repeat per container, so number them to keep each delta.
        phases[f"{len(phases) + 1:02d}. {label}"] = next_ts - ts
    for _, _, r in marks:
        if msg_of(r) == "Readyz reached 200":
            phases[f"readyz[{r.get('container', '')}] (reported)"] = ns_to_s(r.get("elapsed"))

    return phases, {"worker_pod": pod, "ateom_total": end["_ts"] - start["_ts"]}


def atelet_restore(records):
    """Phases from atelet's restore record, normalized to seconds.

    Newer atelet writes ate.actor.restore.duration.<phase> as float seconds
    because the matching instrument declares unit s; older builds write the bare
    phase name as the int64 nanoseconds slog.Duration produces. Read both so the
    profiler does not silently report nothing against a lagging control plane.
    """
    rec = next((r for r in reversed(records) if msg_of(r) == "Restore timing breakdown"), None)
    if not rec:
        return {}
    phases = {}
    for k, v in rec.items():
        if k.startswith(RESTORE_PREFIX):
            phases[k[len(RESTORE_PREFIX):]] = float(v)
        elif k in RESTORE_PHASES and isinstance(v, (int, float)):
            phases.setdefault(k, ns_to_s(v))
    return phases


def atelet_images(records):
    """Per-image cache outcome and, on a miss, how long the pull took.

    The layer cache is node-local, so the same image is a miss the first time
    an actor lands on a node and a hit forever after; the outcome is what makes
    the pull duration readable. Both the cold-boot and the restore path go
    through here, which is the only place either one times the registry fetch.

    Returns (table, notes). A miss with no pull under it is called out rather
    than left to be read as a free one: atelet dedupes concurrent requests for
    a digest through a singleflight group, so a request that loses the race
    logs its miss, blocks on the winner's fetch, and never logs a pull. The
    step still paid for the wait, and the wait is not in this table.
    """
    out, pulls, misses = {}, [], 0
    for r in records:
        m = msg_of(r)
        if m in ("Image cache hit", "Image cache miss"):
            out[short_ref(r.get("ref", ""))] = m.rsplit(" ", 1)[1].upper()
            misses += m == "Image cache miss"
        elif m == "Image pulled into layer cache":
            pulls.append(r)
    for r in pulls:
        out[f"pull {r.get('digest', '')[7:19]} ({r.get('layers')} layers)"] = ns_to_s(r.get("took"))
    notes = []
    if misses > len(pulls):
        n = misses - len(pulls)
        notes.append(
            f"{n} cache miss{'es' if n > 1 else ''} with no pull logged: deduped into "
            "another actor's in-flight fetch (atelet singleflight). This step waited "
            "on that fetch; the wait is inside ateapi_rpc_total, not in the pull rows."
        )
    return out, notes


# Image streaming messages (cmd/atelet/oci.go and the adapter's
# internal/ocilayer/adapter/driver.go).
STREAM_BREAKDOWN = "PrepareImage timing breakdown"
STREAM_BREAKDOWN_FIELDS = ("resolve_ms", "stat_ms", "prepare_ms", "view_ms",
                           "listable_ms", "wrapper_ms", "total_ms")
STREAMED = "Image streamed via OCILayerService"
STREAM_REPREPARE = "Cached streaming lease layers missing on disk; re-preparing layers"
STREAM_FALLBACKS = (
    "Image not streamable via OCILayerService; falling back to cache",
    "OCILayerService layer preparation failed; falling back to cache",
    "OCILayerService declined image; falling back to cache",
    "Error evaluating OCILayerService eligibility; falling back to cache",
    "Failed to resolve image config for streamed image; falling back to cache",
)


def adapter_records(records, begin, until, digests=None):
    """Adapter records logged in [begin, until], limited to digests if given.

    The adapter's lines carry no trace ID (atelet's gRPC client propagates
    none), so time and image are all there is to select them by.
    """
    return [r for r in records
            if r["_ts"] is not None and begin <= r["_ts"] <= until
            and (digests is None or image_digest(r) in digests)]


def atelet_flags():
    """--oci-layer-service-socket and --log-level from the atelet DaemonSet args.

    image_streamer holds the socket, or "none" when unset (streaming off).
    Call this outside every timed span: it is script overhead, not part of any step.
    """
    out = run(f"kubectl -n {ATE_SYSTEM} get ds -l app=atelet -o jsonpath='{{..args}}'",
              check=False).stdout

    def flag(name, default):
        # More than one value means more than one atelet DaemonSet is live.
        vals = sorted(set(re.findall(rf"--{name}=([^\",\s\]]+)", out)))
        return ",".join(vals) or default

    return {"image_streamer": flag("oci-layer-service-socket", "none"),
            "log_level": flag("log-level", "info")}


def image_digest(rec):
    """The digest pinned in a record's image or ref, else its digest field."""
    m = re.search(r"@(sha256:[0-9a-f]+)", rec.get("image") or rec.get("ref") or "")
    return m.group(1) if m else rec.get("digest", "")


def atelet_streaming(records, digests=None):
    """Per-image streaming path and the adapter's PrepareImage breakdown.

    records are atelet's plus the adapter's (adapter_records), in time order.
    Returns (table, notes, streamed digests). digests limits the scan to those
    images; None keeps every record, which suits records already selected.

    The path is read from which lines appear:
      breakdown, prepare_ms > 0   cold: snapshots prepared through the adapter
      breakdown, prepare_ms == 0  stat hit: committed layers already on the node
      breakdown, resolve_ms == 0  resolve cached: image metadata reused, no registry call
      "Image streamed ..." only   lease reuse: the adapter's fast paths log no breakdown

    "unattributed" is total_ms minus the named phases: work inside the
    adapter's PrepareImage that has no field of its own.
    """
    out, notes = {}, []
    broken_down, streamed, fell_back = set(), set(), set()
    for r in records:
        m, d = msg_of(r), image_digest(r)
        if digests is not None and d not in digests:
            continue
        ref = short_ref(r.get("image", "")) or d[7:19]
        if m == STREAM_BREAKDOWN:
            broken_down.add(d)
            path = "cold" if (r.get("prepare_ms") or 0) > 0 else "stat hit (layers already on node)"
            if r.get("resolve_ms") == 0:
                path += ", resolve cached"
            out[f"{ref} path"] = f"STREAMED {path}, {r.get('layers')} layers"
            for k in STREAM_BREAKDOWN_FIELDS:
                if isinstance(r.get(k), (int, float)):
                    out[f"  prepare_layers.{k[:-3]}"] = r[k] / 1000.0
            parts = [r.get(k) for k in STREAM_BREAKDOWN_FIELDS]
            if all(isinstance(v, (int, float)) for v in parts):
                out["  prepare_layers.unattributed"] = (parts[-1] - sum(parts[:-1])) / 1000.0
        elif m == STREAMED:
            streamed.add(d)
            if d not in broken_down:
                out[f"{ref} path"] = f"STREAMED lease reuse, {r.get('layers')} layers"
            out[f"stream {ref} (atelet total)"] = ns_to_s(r.get("duration"))
        elif m in STREAM_FALLBACKS:
            fell_back.add(d)
            reason = m.split(";")[0]
            out[f"{ref} path"] = f"FALLBACK to pull: {reason}" + (f" ({r['err']})" if r.get("err") else "")
        elif m == STREAM_REPREPARE:
            notes.append(f"{ref}: {m}")

    if broken_down - streamed - fell_back:
        notes.append(
            f"the adapter logged a breakdown but atelet logged no '{STREAMED}': "
            "atelet may not be running at debug level."
        )
    return out, notes, streamed | (broken_down - fell_back)


def log_level_note(flags):
    """A warning when streaming is on but atelet drops its Debug lines, else None."""
    if flags.get("image_streamer") == "none" or "debug" in flags.get("log_level", ""):
        return None
    return (
        f"atelet runs at --log-level={flags.get('log_level')}: '{STREAMED}' and "
        "'OCILayerService declined image' are Debug, so lease reuse and declines are "
        "invisible and only the adapter's Info breakdown shows. Set --log-level=debug."
    )


def short_ref(ref):
    """Last path element of an image reference, minus the digest."""
    return ref.rsplit("/", 1)[-1].split("@")[0] or ref


def ateapi_facts(records, method):
    """Server-side total, plus the placement and path facts ateapi logs.

    Returns (facts, notes). Numbers go in facts and are printed as a table;
    notes are messages worth reading verbatim, chiefly which snapshot the
    resume chose, because that is what decides whether the step is a boot from
    spec or a restore and therefore which other tables can exist at all.

    Keys prefixed with _ are working values rather than timings: the caller
    compares _node across steps to classify a resume as same-node or a
    migration, and table() hides them.
    """
    facts, notes = {}, []
    for r in records:
        m = msg_of(r)
        if m == "Handle RPC" and r.get("method", "").endswith(method):
            facts["ateapi_rpc_total"] = parse_go_duration(r.get("elapsed-time"))
            # Logged after the handler returns, so its start is ts - elapsed.
            if facts["ateapi_rpc_total"] is not None and r["_ts"] is not None:
                facts["_rpc_start"] = r["_ts"] - facts["ateapi_rpc_total"]
        elif m == "Picked worker":
            if r["_ts"] is not None:
                facts.setdefault("_picked_at", r["_ts"])  # first attempt, as in profile_coldboot
            w = str(r.get("worker", ""))
            pod = re.search(r'worker_pod:"([^"]+)"', w)
            node = re.search(r'node_name:"([^"]+)"', w)
            if pod:
                facts["_worker_pod"] = pod.group(1)
            if node:
                facts["_node"] = node.group(1)
            notes.append(f"worker {pod.group(1) if pod else '?'} on {node.group(1) if node else '?'}")
        elif "Restoring from snapshot" in m or "Booting from ActorTemplate spec" in m:
            # The clause, not an "Actor has " prefix: the unrelated "Actor has
            # no assigned worker pod during delete" warning shares that prefix.
            notes.append(m)
        elif m == "FinalizeSuspended store call durations":
            for k in ("get_actor", "release_worker", "refetch_actor", "release_snapshot", "update_actor", "total"):
                if k in r:
                    facts[f"finalize_suspended.{k}"] = ns_to_s(r[k])
    return facts, notes


def workload(records, actor, t_zero, origin):
    """Time from t_zero to the actor's first "workload start" and "workload end".

    The profiling/workloads scripts print those records from inside the
    container, tagged with the trigger (boot or resume); ateom stamps them with
    the worker's clock. origin names t_zero in the row labels. The in-sandbox
    elapsed_us, and the end record's bytes and rc when present, are reported
    alongside; bytes is how two arms are checked for doing the same work.
    """
    out = {}
    for r in records:
        m = msg_of(r)
        if m not in ("workload start", "workload end") or actor_name_of(r) != actor or r["_ts"] is None:
            continue
        trigger = r.get("trigger", "?")
        key = f"{origin} -> {m} ({trigger})"
        if key in out:
            continue
        out[key] = r["_ts"] - t_zero if t_zero is not None else None
        if m == "workload end" and isinstance(r.get("elapsed_us"), (int, float)):
            out[f"workload elapsed in sandbox ({trigger})"] = r["elapsed_us"] / 1e6
        if m == "workload end":
            # Integers, so table() prints them as they are rather than as ms.
            for field in ("bytes", "rc"):
                if isinstance(r.get(field), int):
                    out[f"workload {field} ({trigger})"] = r[field]
    return out


def wait_for_workload_end(read_records, actor, after, timeout, poll=1.0,
                          clock=time.time, sleep=time.sleep):
    """Block until the actor logs "workload end" at or after `after`.

    read_records returns the worker records captured so far. Returns the
    record, or None once `timeout` seconds pass without one: a workload that
    hung, crashed or never started would otherwise stall the run.
    """
    deadline = clock() + timeout
    while True:
        for r in read_records():
            if (msg_of(r) == "workload end" and actor_name_of(r) == actor
                    and r["_ts"] is not None and r["_ts"] >= after):
                return r
        if clock() >= deadline:
            return None
        sleep(poll)


def collect(args, streams, step, method, window, trace_id, since, actor, span):
    """Join one lifecycle step's records from all four components.

    Each component's followed log (streams: api, atelet, worker, adapter) is
    merged with a per-step scrape, which covers pods the followers could not see.

    ateapi and atelet are selected by trace ID. ateom is not: its lifecycle
    envelope names the actor but carries no trace ID, so its records are
    selected from the call's start, padded by a second for clock skew between
    the client and the nodes, up to now, which includes any workload wait.
    The adapter has no trace ID either; its records are those logged during
    the call, which assumes no other actor prepares an image meanwhile.
    """
    api_s, let_s, wrk_s, ada_s = streams
    api = for_trace(merge_records(api_s.snapshot(), logs("app=ate-api-server", ATE_SYSTEM, since)), trace_id)
    let = for_trace(merge_records(let_s.snapshot(), logs("app=atelet", ATE_SYSTEM, since)), trace_id)
    begin, until = span[0], time.time()
    wrk = [r for r in merge_records(wrk_s.snapshot(),
                                    logs(args.worker_selector, args.worker_namespace, since, prefix=True))
           if r["_ts"] is not None and begin - 1 <= r["_ts"] <= until + 1]
    wrk.sort(key=lambda r: r["_ts"])
    ada = adapter_records(merge_records(ada_s.snapshot(), logs(ADAPTER_SELECTOR, ATE_SYSTEM, since, prefix=True,
                                                               container=ADAPTER_CONTAINER)),
                          begin - 1, span[1] + 1)

    facts, notes = ateapi_facts(api, method)
    images, image_notes = atelet_images(let)
    streaming, stream_notes, _ = atelet_streaming(sorted(let + ada, key=lambda r: r["_ts"] or 0.0))
    phases, meta = ateom_timeline(wrk, window, actor)
    return {
        "step": step,
        "trace_id": trace_id,
        "notes": notes + image_notes + stream_notes,
        "ateapi": facts,
        "images": images,
        "streaming": streaming,
        "atelet_restore": atelet_restore(let),
        "ateom": phases,
        "ateom_meta": meta,
        "workload": {**workload(wrk, actor, facts.get("_picked_at"), "picked worker"),
                     **workload(wrk, actor, facts.get("_rpc_start"), "rpc start")},
    }


def table(title, data):
    """Print one block of measurements, or nothing if there are none.

    A float is seconds and renders as milliseconds; anything else prints as it
    is, which is how the image cache reports HIT and MISS alongside durations.
    Underscore-prefixed keys are internal to the join and stay hidden.

    Printing nothing for an empty block is deliberate: a component that logged
    no timings for a step should leave no heading behind suggesting it did.
    """
    rows = [(k, v) for k, v in data.items() if v is not None and not k.startswith("_")]
    if not rows:
        return
    print(f"\n  {title}")
    for k, v in rows:
        print(f"    {k:<44} {v * 1000:>10.1f} ms" if isinstance(v, float) else f"    {k:<44} {v}")


def report(result, placement):
    """Print one step: its trace ID, what happened, then a table per component.

    The tables read outside-in — client, then ate-api-server, then atelet, then
    the worker — so each row's remainder over the one below it is the overhead
    that layer added.
    """
    print("\n" + "=" * 68)
    print(f" {result['step'].upper()}   (trace {result['trace_id'] or 'n/a'})")
    print("=" * 68)
    for n in result["notes"]:
        print(f"  - {n}")
    if result["ateom_meta"].get("worker_pod"):
        print(f"  - worker pod {result['ateom_meta']['worker_pod']}")
    if placement:
        print(f"  - {placement}")
    table("client + ate-api-server", {"client_wall": result["client_wall"], **result["ateapi"]})
    table("atelet image cache (pull from the registry)", result["images"])
    table("atelet image streaming", result["streaming"])
    table("atelet restore phases (concurrent: download || assets+oci_unpack)", result["atelet_restore"])
    table("ateom worker steps", {**result["ateom"], "ateom_total": result["ateom_meta"].get("ateom_total")})
    table("workload (from picked worker and ate-api-server RPC start)", result.get("workload", {}))


def main():
    """Drive one actor through its lifecycle, reporting each step as it goes.

    create (born suspended, no worker) -> resume (boot from spec, or a restore
    if the template has a golden snapshot) -> suspend (checkpoint to object
    storage, release the worker) -> resume (restore what suspend just wrote).

    Whether the last step lands on the same node is the scheduler's choice, so
    it is classified after the fact rather than forced.
    """
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--atespace", required=True)
    p.add_argument("--template", required=True, help="<atespace>/<name> or <name>")
    p.add_argument("--worker-namespace", required=True, help="namespace of the WorkerPool pods")
    p.add_argument("--worker-selector", default="", help="defaults to ate.dev/worker-pool=<--worker-pool>")
    p.add_argument("--worker-pool", default="", help="WorkerPool name")
    p.add_argument("--actor", default="", help="actor name; defaults to a timestamped one")
    p.add_argument("--kubectl-ate", default="kubectl-ate")
    p.add_argument("--settle", type=float, default=3.0,
                   help="seconds to wait before scraping each step's logs (after the workload wait, if any)")
    p.add_argument("--workload-timeout", type=float, default=0.0,
                   help="after each resume, wait up to this many seconds for the actor's "
                        "'workload end' line before settling; 0 disables the wait")
    p.add_argument("--step-gap", type=float, default=0.0,
                   help="seconds to idle before the suspend and before resume #2, outside every "
                        "measured span, so a cold stream's background layer download can finish")
    p.add_argument("--retries", type=int, default=5, help="retries when the assigned worker pod is gone")
    p.add_argument("--json", default="", help="write the full result to this path")
    p.add_argument("--keep", action="store_true", help="do not delete the actor afterwards")
    args = p.parse_args()

    if not args.worker_selector:
        if not args.worker_pool:
            p.error("one of --worker-selector or --worker-pool is required")
        args.worker_selector = f"ate.dev/worker-pool={args.worker_pool}"

    # Read before the first step so it never lands in a measured span.
    flags = atelet_flags()
    print(f"atelet --oci-layer-service-socket={flags['image_streamer']} --log-level={flags['log_level']}")
    if log_level_note(flags):
        print(f"WARNING: {log_level_note(flags)}")

    actor = args.actor or f"prof-{int(time.time())}"
    results, t_origin = [], time.time()

    def since():
        """Seconds of history to ask kubectl for, widening as the run goes on.

        Always reaches back to the start of the run, so a step whose records
        landed late is still in the window, with a floor for the first step.
        """
        return max(int(time.time() - t_origin) + 5, 10)

    def settle_after_resume(streams, span):
        """Wait for the resume's workload to finish, then settle.

        Returns a note when the wait timed out, else None.
        """
        note = None
        if args.workload_timeout > 0:
            if wait_for_workload_end(streams[2].snapshot, actor, span[0] - 1, args.workload_timeout) is None:
                note = f"workload end not seen within {args.workload_timeout:.0f} s of the resume"
                print(f"WARNING: {note}")
        time.sleep(args.settle)
        return note

    try:
        with contextlib.ExitStack() as stack:
            # Followed for the whole lifecycle: a busy ate-api-server can rotate
            # its log file during a long workload wait.
            streams = tuple(stack.enter_context(s) for s in (
                LogStream("app=ate-api-server", ATE_SYSTEM),
                LogStream("app=atelet", ATE_SYSTEM),
                LogStream(args.worker_selector, args.worker_namespace, prefix=True),
                LogStream(ADAPTER_SELECTOR, ATE_SYSTEM, prefix=True, container=ADAPTER_CONTAINER)))

            # 1. create: metadata only, the actor is born suspended and holds no worker.
            wall, trace, span = ate(args, f"create actor {actor} --template {args.template}")
            time.sleep(args.settle)
            r = collect(args, streams, "create actor record", "CreateActor", "boot", trace, since(), actor, span)
            r["client_wall"] = wall
            results.append((r, ""))

            # 2. first resume: cold boot from the template spec, or a golden restore
            #    if the template has a golden snapshot. ateapi logs which it chose.
            wall, trace, span = ate(args, f"resume actor {actor}", retries=args.retries)
            wait_note = settle_after_resume(streams, span)
            boot = collect(args, streams, "cold boot (resume #1)", "ResumeActor", "boot", trace, since(), actor, span)
            if not boot["ateom"]:  # a golden restore lands in the restore window
                boot.update(collect(args, streams, "initial activation (resume #1)", "ResumeActor", "restore",
                                    trace, since(), actor, span))
            if wait_note:
                boot["notes"].append(wait_note)
            boot["client_wall"] = wall
            node_before = boot["ateapi"].get("_node", "")
            results.append((boot, f"node {node_before or 'unknown'}"))

            # 3. live suspend: checkpoint to object storage and release the worker.
            #    --step-gap idles first, outside the measured span: a cold stream
            #    keeps downloading layers in the background after the resume returns.
            time.sleep(args.step_gap)
            wall, trace, span = ate(args, f"suspend actor {actor}", retries=args.retries)
            time.sleep(args.settle)
            r = collect(args, streams, "live suspend", "SuspendActor", "suspend", trace, since(), actor, span)
            r["client_wall"] = wall
            results.append((r, ""))

            # 4. second resume: restore from the snapshot just written. Whether it
            #    lands on the same node is the pool's choice, so classify it rather
            #    than force it. --step-gap idles first, as before the suspend.
            time.sleep(args.step_gap)
            wall, trace, span = ate(args, f"resume actor {actor}", retries=args.retries)
            wait_note = settle_after_resume(streams, span)
            r = collect(args, streams, "resume from snapshot", "ResumeActor", "restore", trace, since(), actor, span)
            if wait_note:
                r["notes"].append(wait_note)
            r["client_wall"] = wall
            node_after = r["ateapi"].get("_node", "")
            kind = "same-node warm resume" if node_after and node_after == node_before else "cross-node migration"
            results.append((r, f"{kind}: {node_before or '?'} -> {node_after or '?'}"))
    finally:
        # A step that fails still measured the ones before it, and still leaves
        # an actor behind, so report and clean up on the way out either way.
        for r, placement in results:
            r["placement"] = placement
            report(r, placement)

        if args.json and results:
            with open(args.json, "w") as f:
                json.dump({"actor": actor, "atelet_flags": flags,
                           "steps": [r for r, _ in results]}, f, indent=2)
            print(f"\nwrote {args.json}")

        if not args.keep:
            # Deletion requires the actor to be suspended; a run that died
            # mid-resume leaves it running.
            run(f"{args.kubectl_ate} suspend actor {actor} -a {args.atespace}", check=False)
            run(f"{args.kubectl_ate} delete actor {actor} -a {args.atespace}", check=False)
            print(f"cleaned up actor {actor}")


if __name__ == "__main__":
    sys.exit(main())
