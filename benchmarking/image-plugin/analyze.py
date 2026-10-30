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

"""Summarizes a benchmark run directory into Markdown tables.

  analyze.py <run-dir> > summary.md

Reads <run-dir>/rep-<n>/<arm>/turns.jsonl and atelet-*.log. Turn 0 is the
first restore from the golden snapshot; turns 1+ are resumes from the
actor's own snapshot after a suspend.
"""

import glob
import gzip
import json
import os
import re
import sys
from collections import defaultdict

ARMS = ["pull", "riptide", "nydus", "spegel"]
PHASES = ["manifest_fetch", "download", "oci_unpack", "worker_restore", "total"]


def pct(xs, p):
    if not xs:
        return float("nan")
    xs = sorted(xs)
    k = (len(xs) - 1) * p / 100.0
    lo, hi = int(k), min(int(k) + 1, len(xs) - 1)
    return xs[lo] + (xs[hi] - xs[lo]) * (k - lo)


def fmt(x):
    return "—" if x != x else ("%.2f" % x if x < 10 else "%.1f" % x)


def open_text(path):
    """Opens a results file, gzipped (as committed) or plain."""
    if path.endswith(".gz"):
        return gzip.open(path, "rt")
    return open(path)


def load(run_dir):
    turns = defaultdict(list)
    for path in glob.glob(os.path.join(run_dir, "rep-*", "*", "turns.jsonl*")):
        arm = os.path.basename(os.path.dirname(path))
        rep = os.path.basename(os.path.dirname(os.path.dirname(path)))
        if rep == "rep-0":
            continue
        with open_text(path) as f:
            for line in f:
                line = line.strip()
                if line:
                    turns[arm].append(json.loads(line))
    return turns


def phases(run_dir, arm):
    out = defaultdict(list)
    pat = re.compile(r'"ate\.actor\.restore\.duration\.([a-z_]+)":([0-9.eE+-]+)')
    for path in glob.glob(os.path.join(run_dir, "rep-[1-9]*", arm, "atelet-*.log*")):
        with open_text(path) as f:
            for line in f:
                if "Restore timing breakdown" not in line:
                    continue
                for k, v in pat.findall(line):
                    out[k].append(float(v))
    return out


def table(rows, header):
    print("| " + " | ".join(header) + " |")
    print("|" + "---|" * len(header))
    for r in rows:
        print("| " + " | ".join(str(c) for c in r) + " |")
    print()


def main():
    run_dir = sys.argv[1]
    turns = load(run_dir)
    for title, sel in (("First restore (turn 0, from golden)", lambda t: t["turn"] == 0),
                       ("Resumes (turns 1+, from the actor's own snapshot)", lambda t: t["turn"] > 0),
                       ("All turns", lambda t: True)):
        print("### " + title + "\n")
        rows = []
        for arm in ARMS:
            ts = [t for t in turns.get(arm, []) if sel(t)]
            ok = [t for t in ts if not t.get("error")]
            failed = len(ts) - len(ok)
            def col(k, src=ts):
                return [t[k] for t in src if t.get(k)]
            rows.append([
                arm, len(ts), "%d (%.1f%%)" % (failed, 100.0 * failed / len(ts)) if ts else "—",
                fmt(pct(col("ttfi_s", ok), 50)), fmt(pct(col("ttfi_s", ok), 95)), fmt(max(col("ttfi_s", ok) or [float("nan")])),
                fmt(pct(col("read_s", ok), 50)), fmt(pct(col("read_s", ok), 95)),
                fmt(pct(col("write_s", ok), 50)), fmt(pct(col("write_s", ok), 95)),
                fmt(pct(col("task_s", ok), 50)), fmt(pct(col("task_s", ok), 95)),
                fmt(pct(col("suspend_s", ok), 50)), fmt(pct(col("suspend_s", ok), 95)),
                fmt(pct(col("queue_s", ts), 50)), fmt(pct(col("queue_s", ts), 95)),
                sum(t.get("no_room_retries", 0) for t in ts),
            ])
        table(rows, ["arm", "n", "failed", "ttfi p50", "ttfi p95", "ttfi max", "read p50", "read p95",
                     "write p50", "write p95", "task p50", "task p95", "suspend p50", "suspend p95",
                     "queue p50", "queue p95", "no-room retries"])

    print("### Failures by step\n")
    rows = []
    for arm in ARMS:
        by = defaultdict(int)
        for t in turns.get(arm, []):
            if t.get("error"):
                by[t.get("error_step", "?")] += 1
        rows.append([arm] + [by.get(s, 0) for s in ("create", "resume", "probe", "work", "suspend")])
    table(rows, ["arm", "create", "resume", "probe", "work", "suspend"])

    print("### Rep wall time (s, first turn start to last turn end)\n")
    rows = []
    for arm in ARMS:
        per = defaultdict(list)
        for t in turns.get(arm, []):
            per[t["rep"]].append(t)
        walls = []
        for rep, ts in sorted(per.items()):
            from datetime import datetime
            starts = [datetime.fromisoformat(t["start"].replace("Z", "+00:00")).timestamp() for t in ts if t.get("start")]
            ends = [s + t["turn_s"] for s, t in zip(starts, [t for t in ts if t.get("start")])]
            if starts:
                walls.append("%d: %.0f" % (rep, max(ends) - min(starts)))
        rows.append([arm, ", ".join(walls)])
    table(rows, ["arm", "rep: wall s"])

    print("### Restore phases from atelet (s, p50 / p95, all restores)\n")
    rows = []
    for arm in ARMS:
        ph = phases(run_dir, arm)
        rows.append([arm, len(ph.get("total", []))] +
                    ["%s / %s" % (fmt(pct(ph.get(p, []), 50)), fmt(pct(ph.get(p, []), 95))) for p in PHASES])
    table(rows, ["arm", "n"] + PHASES)


if __name__ == "__main__":
    main()
