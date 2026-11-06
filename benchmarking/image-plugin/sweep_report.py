#!/usr/bin/env python3
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Compares a density sweep: one row per arm and actors-per-worker.

  sweep_report.py <run-dir> > sweep.md

Reads <run-dir>/rep-*/<arm>-w<n>/turns.jsonl and diskstats.jsonl (the
node-sampler log, with disk and CPU samples), plain or gzipped. TTFI
includes the time an actor waited for a slot. Session time is an actor's create to the end of its last turn.
"""

import glob
import gzip
import json
import os
import re
import sys
from collections import defaultdict
from datetime import datetime

ARMS = ["pull", "riptide", "nydus", "spegel"]
SECTOR = 512


def pct(xs, p):
    if not xs:
        return float("nan")
    xs = sorted(xs)
    k = (len(xs) - 1) * p / 100.0
    lo, hi = int(k), min(int(k) + 1, len(xs) - 1)
    return xs[lo] + (xs[hi] - xs[lo]) * (k - lo)


def ts(s):
    # Go writes nanoseconds; datetime takes at most microseconds.
    s = re.sub(r"(\.\d{6})\d+", r"\1", s).replace("Z", "+00:00")
    return datetime.fromisoformat(s)


def fmt(x, d=2):
    return "—" if x != x else f"{x:.{d}f}"


def load(path):
    """Reads JSON lines from path, or from path.gz as committed."""
    if not os.path.exists(path) and os.path.exists(path + ".gz"):
        path += ".gz"
    opener = gzip.open if path.endswith(".gz") else open
    with opener(path, "rt") as f:
        return [json.loads(l) for l in f if l.strip()]


def present(path):
    return os.path.exists(path) or os.path.exists(path + ".gz")


def disk_rates(path):
    """Peak and p50 of per-node write MB/s, read MB/s and IOPS per node-sampler sample."""
    by_node = defaultdict(list)
    for r in load(path):
        if r.get("kind") != "diskstats" or not r.get("disks"):
            continue
        d = max(r["disks"], key=lambda x: x["write_sectors"] + x["read_sectors"])
        by_node[r["node"]].append((ts(r["ts"]), d))
    wmb, rmb, iops = [], [], []
    for samples in by_node.values():
        samples.sort(key=lambda x: x[0])
        for (t0, a), (t1, b) in zip(samples, samples[1:]):
            dt = (t1 - t0).total_seconds()
            if dt <= 0:
                continue
            wmb.append((b["write_sectors"] - a["write_sectors"]) * SECTOR / dt / 1e6)
            rmb.append((b["read_sectors"] - a["read_sectors"]) * SECTOR / dt / 1e6)
            iops.append(((b["writes"] - a["writes"]) + (b["reads"] - a["reads"])) / dt)
    return wmb, rmb, iops


def cpu_rates(path):
    """Per-node CPU busy and iowait percent per node-sampler sample, and the mean
    cores per node of each process name over the run."""
    by_node = defaultdict(list)
    for r in load(path):
        if r.get("kind") == "cpustat":
            by_node[r["node"]].append((ts(r["ts"]), r))
    busy, iowait = [], []
    proc_ticks = defaultdict(float)
    span = 0.0
    for samples in by_node.values():
        samples.sort(key=lambda x: x[0])
        for (t0, a), (t1, b) in zip(samples, samples[1:]):
            dt = (t1 - t0).total_seconds()
            total = sum(b["cpu"].values()) - sum(a["cpu"].values())
            if dt <= 0 or total <= 0:
                continue
            idle = b["cpu"]["idle"] - a["cpu"]["idle"]
            io = b["cpu"]["iowait"] - a["cpu"]["iowait"]
            busy.append(100 * (total - idle - io) / total)
            iowait.append(100 * io / total)
            for k, v in b["procs"].items():
                # A name's total drops when one of its processes exits.
                proc_ticks[k] += max(0, v - a["procs"].get(k, v))
            span += dt
    # Ticks are USER_HZ (100 per second).
    cores = {k: v / 100 / span for k, v in proc_ticks.items()} if span else {}
    return busy, iowait, cores


def main():
    run = sys.argv[1]
    rows = []
    for d in sorted(glob.glob(os.path.join(run, "rep-*", "*-w*"))):
        m = re.match(r"(.+)-w(\d+)$", os.path.basename(d))
        tj = os.path.join(d, "turns.jsonl")
        if not m or not present(tj):
            continue
        arm, dens = m.group(1), int(m.group(2))
        turns = load(tj)
        ok = [t for t in turns if not t.get("error")]
        first = [t["ttfi_s"] for t in ok if t["turn"] == 0]
        later = [t["ttfi_s"] for t in ok if t["turn"] > 0]
        allt = [t["ttfi_s"] for t in ok]
        queue = [t["queue_s"] for t in ok]
        work = [t["work_call_s"] for t in ok]
        susp = [t["suspend_s"] for t in ok]
        # End to end: from the resume request to the end of the work call.
        e2e0 = [t["ttfi_s"] + t["work_call_s"] for t in ok if t["turn"] == 0]
        e2en = [t["ttfi_s"] + t["work_call_s"] for t in ok if t["turn"] > 0]
        resumes = [t for t in ok if t["turn"] > 0]
        moved = [t for t in resumes if t.get("moved")]
        # Session: create (turn 0 start) to the end of the last turn.
        sess = defaultdict(lambda: [None, None])
        for t in turns:
            if not t.get("start"):
                continue
            s = ts(t["start"])
            e = s.timestamp() + t.get("turn_s", 0)
            a = sess[t["actor"]]
            a[0] = s.timestamp() if a[0] is None else min(a[0], s.timestamp())
            a[1] = e if a[1] is None else max(a[1], e)
        sessions = [b - a for a, b in sess.values() if a is not None]
        wall = (max(b for _, b in sess.values()) - min(a for a, _ in sess.values())) if sess else float("nan")
        wmb, rmb, iops = disk_rates(os.path.join(d, "diskstats.jsonl")) if present(os.path.join(d, "diskstats.jsonl")) else ([], [], [])
        busy, iowait, cores = cpu_rates(os.path.join(d, "diskstats.jsonl")) if present(os.path.join(d, "diskstats.jsonl")) else ([], [], {})
        rows.append({
            "arm": arm, "w": dens, "slots": 48 * dens, "n": len(turns),
            "failed": len(turns) - len(ok),
            "t0": (pct(first, 50), pct(first, 95), pct(first, 99), max(first, default=float("nan"))),
            "tn": (pct(later, 50), pct(later, 95), pct(later, 99), max(later, default=float("nan"))),
            "all_p100": max(allt, default=float("nan")),
            "e2e0": (pct(e2e0, 50), pct(e2e0, 95), pct(e2e0, 99), max(e2e0, default=float("nan"))),
            "e2en": (pct(e2en, 50), pct(e2en, 95), pct(e2en, 99), max(e2en, default=float("nan"))),
            "queue": (pct(queue, 50), pct(queue, 95), max(queue, default=float("nan"))),
            "work": (pct(work, 50), pct(work, 95)),
            "suspend": (pct(susp, 50), pct(susp, 95)),
            "moved": (len(moved), len(resumes)),
            "moved_ttfi": pct([t["ttfi_s"] for t in moved], 50),
            "session": (pct(sessions, 50), pct(sessions, 95), max(sessions, default=float("nan"))),
            "wall": wall,
            "turns_per_s": len(ok) / wall if wall == wall and wall > 0 else float("nan"),
            "disk": (max(wmb, default=float("nan")), max(rmb, default=float("nan")), max(iops, default=float("nan"))),
            "cpu": (pct(busy, 50), pct(busy, 95), max(busy, default=float("nan")), pct(iowait, 50), max(iowait, default=float("nan"))),
            "cores": cores,
        })
    rows.sort(key=lambda r: (ARMS.index(r["arm"]) if r["arm"] in ARMS else 9, r["w"]))

    print("### TTFI including queue wait (s)\n")
    print("| arm | actors/worker | slots | turns | failed | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 | queue p50 / p95 / max |")
    print("|---|---|---|---|---|---|---|---|")
    for r in rows:
        print(f"| {r['arm']} | {r['w']} | {r['slots']} | {r['n']} | {r['failed']} | "
              + " / ".join(fmt(x, 1) for x in r["t0"]) + " | "
              + " / ".join(fmt(x) for x in r["tn"]) + " | "
              + " / ".join(fmt(x, 1) for x in r["queue"]) + " |")

    print("\n### End to end per turn: resume request to work done (s)\n")
    print("| arm | actors/worker | first turn p50 / p95 / p99 / p100 | later turns p50 / p95 / p99 / p100 |")
    print("|---|---|---|---|")
    for r in rows:
        print(f"| {r['arm']} | {r['w']} | " + " / ".join(fmt(x, 1) for x in r["e2e0"]) + " | "
              + " / ".join(fmt(x) for x in r["e2en"]) + " |")

    print("\n### Completion (s)\n")
    print("| arm | actors/worker | work p50 / p95 | suspend p50 / p95 | session p50 / p95 / max | wall | turns/s | moved resumes | moved TTFI p50 |")
    print("|---|---|---|---|---|---|---|---|---|")
    for r in rows:
        mv, rs = r["moved"]
        print(f"| {r['arm']} | {r['w']} | " + " / ".join(fmt(x) for x in r["work"]) + " | "
              + " / ".join(fmt(x) for x in r["suspend"]) + " | "
              + " / ".join(fmt(x, 0) for x in r["session"]) + f" | {fmt(r['wall'], 0)} | {fmt(r['turns_per_s'], 1)} | "
              + f"{mv}/{rs} ({100 * mv / rs if rs else 0:.0f}%) | {fmt(r['moved_ttfi'])} |")

    print("\n### Disk peak per node (node-sampler samples)\n")
    print("| arm | actors/worker | write MB/s | read MB/s | IOPS |")
    print("|---|---|---|---|---|")
    for r in rows:
        w, rd, io = r["disk"]
        print(f"| {r['arm']} | {r['w']} | {fmt(w, 0)} | {fmt(rd, 0)} | {fmt(io, 0)} |")

    print("\n### CPU per node (node-sampler samples, % of all cores)\n")
    print("| arm | actors/worker | busy p50 / p95 / max | iowait p50 / max | top processes, mean cores per node |")
    print("|---|---|---|---|---|")
    for r in rows:
        top = sorted(r["cores"].items(), key=lambda x: -x[1])[:6]
        print(f"| {r['arm']} | {r['w']} | " + " / ".join(fmt(x, 0) for x in r["cpu"][:3]) + " | "
              + " / ".join(fmt(x, 1) for x in r["cpu"][3:]) + " | "
              + ", ".join(f"{k} {v:.2f}" for k, v in top) + " |")


if __name__ == "__main__":
    main()
