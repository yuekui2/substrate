#!/usr/bin/env python3
"""Summarizes one bench point: first-turn TTFI/E2E, per-node placement and
TTFI, and the first restore breakdown per actor from the atelet logs.

  pointstat.py <dir with turns.jsonl and atelet-*.log>
"""
import collections
import glob
import json
import sys


def pct(v, q):
    v = sorted(v)
    return v[min(len(v) - 1, int(q * len(v)))] if v else float("nan")


def trio(v):
    return f"{pct(v, .5):.1f}/{pct(v, .95):.1f}/{max(v):.1f}"


d = sys.argv[1]
turns = [json.loads(l) for l in open(d + "/turns.jsonl") if l.strip()]
f = [t for t in turns if t["turn"] == 0 and not t.get("error")]
print(f"first turn n={len(f)} errors={sum(1 for t in turns if t.get('error'))}")
print("TTFI", trio([t["ttfi_s"] for t in f]), " E2E", trio([t["ttfi_s"] + t["work_call_s"] for t in f]))
by = collections.defaultdict(list)
for t in f:
    by[t["node"][-4:]].append(t["ttfi_s"])
for n, v in sorted(by.items(), key=lambda x: len(x[1])):
    print(f"  node {n}: {len(v)} actors TTFI p50 {pct(v, .5):.1f} max {max(v):.1f}")
first = {}
for fn in glob.glob(d + "/atelet-*.log"):
    for l in open(fn):
        if "Restore timing breakdown" not in l:
            continue
        r = json.loads(l[l.index("{"):])
        a = r.get("ate.actor.name")
        if a not in first or r["time"] < first[a]["time"]:
            first[a] = r
rows = list(first.values())
keys = sorted({k for r in rows for k in r if k.startswith("ate.actor.restore.duration")})
for k in keys:
    v = [r.get(k, 0) for r in rows]
    print(f"  {k.split('.')[-1]:15s} p50 {pct(v, .5):.2f} p95 {pct(v, .95):.2f} max {max(v):.2f}")
