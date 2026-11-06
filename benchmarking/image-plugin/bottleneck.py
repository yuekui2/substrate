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

"""Shows where the first-turn TTFI tail of one arm run goes.

  bottleneck.py <arm-dir> > bottleneck.md

<arm-dir> is one run.sh result directory, such as n32/rep-1/nydus-pfA-w2.
Reads turns.jsonl, atelet-*.log, worker-phases.log, diskstats.jsonl (the
node-sampler log), nodes.txt and seed-access-*.jsonl, plain or
gzipped. The window runs from the first actor's create to the last first
turn's TTFI. Rates are per node-sampler sample.
"""

import glob
import json
import os
import sys
from collections import defaultdict

from sweep_report import SECTOR, fmt, load, pct, present, ts


def epoch(s):
    return ts(s).timestamp()


def phases(paths, prefix):
    """Maps actor name to (end epoch, {phase: seconds}) of its first restore,
    the one for turn 0."""
    out = {}
    for p in paths:
        for r in load(p):
            if r.get("msg") != "Restore timing breakdown":
                continue
            ph = {k[len(prefix):]: v for k, v in r.items() if k.startswith(prefix)}
            end = epoch(r["time"])
            name = r["ate.actor.name"]
            if ph and (name not in out or end < out[name][0]):
                out[name] = (end, ph)
    return out


def node_series(rows):
    """Per node, the rates between consecutive samples of each kind."""
    by = defaultdict(lambda: defaultdict(list))
    for r in rows:
        if "kind" in r and "ts" in r:
            by[r["node"]][r["kind"]].append((epoch(r["ts"]), r))
    series = defaultdict(list)
    for node, kinds in by.items():
        for kind, samples in kinds.items():
            samples.sort(key=lambda x: x[0])
            for (t0, a), (t1, b) in zip(samples, samples[1:]):
                dt = t1 - t0
                if dt <= 0:
                    continue
                v = rates(kind, a, b, dt)
                if v:
                    series[node].append((t1, v))
    return series


def rates(kind, a, b, dt):
    if kind == "diskstats" and a.get("disks") and b.get("disks"):
        da = {d["dev"]: d for d in a["disks"]}
        d1 = max(b["disks"], key=lambda x: x["write_sectors"] + x["read_sectors"])
        d0 = da.get(d1["dev"])
        if not d0:
            return None
        ios = (d1["reads"] - d0["reads"]) + (d1["writes"] - d0["writes"])
        ms = (d1["read_ms"] - d0["read_ms"]) + (d1["write_ms"] - d0["write_ms"])
        return {
            "rd_mb": (d1["read_sectors"] - d0["read_sectors"]) * SECTOR / dt / 1e6,
            "wr_mb": (d1["write_sectors"] - d0["write_sectors"]) * SECTOR / dt / 1e6,
            "iops": ios / dt,
            "await_ms": ms / ios if ios else 0.0,
            "queue": (d1["weighted_ms"] - d0["weighted_ms"]) / 1000 / dt,
        }
    if kind == "cpustat":
        total = sum(b["cpu"].values()) - sum(a["cpu"].values())
        if total <= 0:
            return None
        idle = (b["cpu"]["idle"] - a["cpu"]["idle"]) + (b["cpu"]["iowait"] - a["cpu"]["iowait"])
        return {"cpu_busy": 100 * (total - idle) / total}
    if kind == "hoststat":
        v = {}
        rx = sum(x[0] for x in b["net"].values()) - sum(x[0] for x in a["net"].values())
        v["rx_mb"] = rx / dt / 1e6
        if a.get("tcp") and b.get("tcp"):
            v["retrans"] = (b["tcp"][0] - a["tcp"][0]) / dt
        for r in ("cpu", "io", "memory"):
            v[f"psi_{r}"] = 100 * (b["psi"][r][0] - a["psi"][r][0]) / 1e6 / dt
        cores = [100 * (y[0] - x[0]) / (y[1] - x[1]) for x, y in zip(a["cpus"], b["cpus"]) if y[1] > x[1]]
        v["top_core"] = max(cores) if cores else 0.0
        top, name = 0.0, ""
        for k, t in b["threads"].items():
            # Ticks are USER_HZ (100 per second), so ticks per second is % of a core.
            d = (t - a["threads"].get(k, t)) / dt
            if d > top:
                top, name = d, k
        v["top_thread"], v["top_thread_name"] = top, name
        v["dirty_mb"] = b["mem"].get("Dirty", 0) / 1024
        v["wb_mb"] = b["mem"].get("Writeback", 0) / 1024
        return v
    return None


def seed_requests(d):
    reqs = []
    for p in glob.glob(os.path.join(d, "seed-access-*.jsonl*")):
        for r in load(p.removesuffix(".gz")):
            reqs.append(r)
    return reqs


def main():
    d = sys.argv[1]
    turns = [t for t in load(os.path.join(d, "turns.jsonl")) if t["turn"] == 0 and "ttfi_s" in t]
    if not turns:
        sys.exit("no first turns")
    t0 = min(epoch(t["start"]) for t in turns)
    t1 = max(epoch(t["start"]) + t["ttfi_s"] for t in turns)
    atelet = phases(sorted(glob.glob(os.path.join(d, "atelet-*.log*"))), "ate.actor.restore.duration.")
    worker = phases([os.path.join(d, "worker-phases.log")] if present(os.path.join(d, "worker-phases.log")) else [],
                    "ateworker.actor.restore.duration.")
    ttfi = [t["ttfi_s"] for t in turns]
    print(f"# Bottleneck: {d}\n")
    print(f"{len(turns)} first turns. TTFI p50 / p95 / p99 / p100: "
          f"{fmt(pct(ttfi, 50))} / {fmt(pct(ttfi, 95))} / {fmt(pct(ttfi, 99))} / {fmt(max(ttfi))} s. "
          f"Window {t1 - t0:.1f} s.\n")

    print("## Per node\n")
    print("| node | actors | TTFI p50 | TTFI max |")
    print("|---|---|---|---|")
    by_node = defaultdict(list)
    for t in turns:
        by_node[t["node"]].append(t["ttfi_s"])
    for n, xs in sorted(by_node.items()):
        print(f"| {n[-4:]} | {len(xs)} | {fmt(pct(xs, 50))} | {fmt(max(xs))} |")

    print("\n## Slowest 10 first turns\n")
    print("| actor | node | TTFI | queue | manifest_fetch | download | oci_unpack | worker_restore | net_setup | sandbox_create | sandbox_restore | app_restore | wakeup_probe |")
    print("|---|---|---|---|---|---|---|---|---|---|---|---|---|")
    for t in sorted(turns, key=lambda t: -t["ttfi_s"])[:10]:
        a = atelet.get(t["actor"], (0, {}))[1]
        w = worker.get(t["actor"], (0, {}))[1]
        cells = [fmt(a.get(k, float("nan"))) for k in ("manifest_fetch", "download", "oci_unpack", "worker_restore")]
        cells += [fmt(w.get(k, float("nan"))) for k in ("net_setup", "sandbox_create", "sandbox_restore", "app_restore", "wakeup_probe")]
        print(f"| {t['actor']} | {t['node'][-4:]} | {fmt(t['ttfi_s'])} | {fmt(t['queue_s'])} | " + " | ".join(cells) + " |")

    # Actors in wake-up per node and second: [end - wakeup_probe, end].
    actor_node = {t["actor"]: t["node"] for t in turns}
    waking = defaultdict(lambda: defaultdict(int))
    for actor, (end, w) in worker.items():
        if actor in actor_node and "wakeup_probe" in w:
            for s in range(int(end - w["wakeup_probe"] - t0), int(end - t0) + 1):
                waking[actor_node[actor]][s] += 1

    reqs = seed_requests(d)
    ip_node = {}
    if os.path.exists(os.path.join(d, "nodes.txt")):
        for line in open(os.path.join(d, "nodes.txt")):
            if line.split():
                name, ip = line.split()
                ip_node[ip] = name
    # Seed requests per requesting node and second they started.
    seed_by_s = defaultdict(lambda: defaultdict(list))
    for r in reqs:
        seed_by_s[ip_node.get(r["ra"], "")][int(r["msec"] - r["rt"] - t0)].append(r)

    series = node_series(load(os.path.join(d, "diskstats.jsonl")))
    keys = ["rd_mb", "wr_mb", "iops", "await_ms", "queue", "rx_mb", "retrans", "cpu_busy", "top_core",
            "top_thread", "psi_cpu", "psi_io", "psi_memory", "dirty_mb", "wb_mb"]
    print("\n## Peaks in the window, per node\n")
    print("| node | " + " | ".join(keys) + " | busiest thread |")
    print("|---|" + "---|" * (len(keys) + 1))
    for n in sorted(by_node):
        win = [v for t, v in series.get(n, []) if t0 <= t <= t1 + 1]
        peaks = []
        for k in keys:
            xs = [v[k] for v in win if k in v]
            peaks.append(fmt(max(xs), 1) if xs else "—")
        tt = max((v for v in win if "top_thread" in v), key=lambda v: v["top_thread"], default={})
        print(f"| {n[-4:]} | " + " | ".join(peaks) + f" | {tt.get('top_thread_name', '—')} |")

    print("\n## Top processes in the window, cores per node\n")
    cpu = defaultdict(list)
    for r in load(os.path.join(d, "diskstats.jsonl")):
        if r.get("kind") == "cpustat":
            cpu[r["node"]].append((epoch(r["ts"]), r["procs"]))
    for n in sorted(by_node):
        win = sorted(x for x in cpu.get(n, []) if t0 - 1 <= x[0] <= t1 + 1)
        if len(win) < 2:
            continue
        (ta, a), (tb, b) = win[0], win[-1]
        # Ticks are USER_HZ (100 per second).
        cores = {k: (v - a.get(k, v)) / 100 / (tb - ta) for k, v in b.items()}
        top = sorted(cores.items(), key=lambda x: -x[1])[:8]
        print(f"- {n[-4:]}: " + ", ".join(f"{k} {v:.2f}" for k, v in top))

    for n in sorted(by_node):
        print(f"\n## Timeline, node {n[-4:]}\n")
        print("| s | waking | rd MB/s | wr MB/s | IOPS | await ms | rx MB/s | top core % | top thread % | psi io % | psi cpu % | seed req/s | seed rt p99 ms |")
        print("|---|---|---|---|---|---|---|---|---|---|---|---|---|")
        merged = defaultdict(dict)
        for t, v in series.get(n, []):
            if t0 - 1 <= t <= t1 + 2:
                merged[int(t - t0)].update(v)
        for s in sorted(merged):
            v = merged[s]
            sr = seed_by_s[n].get(s, [])
            rt = [r["rt"] * 1000 for r in sr]
            print(f"| {s} | {waking[n].get(s, 0)} | {fmt(v.get('rd_mb', float('nan')), 0)} | {fmt(v.get('wr_mb', float('nan')), 0)} | "
                  f"{fmt(v.get('iops', float('nan')), 0)} | {fmt(v.get('await_ms', float('nan')))} | {fmt(v.get('rx_mb', float('nan')), 0)} | "
                  f"{fmt(v.get('top_core', float('nan')), 0)} | {fmt(v.get('top_thread', float('nan')), 0)} | "
                  f"{fmt(v.get('psi_io', float('nan')), 1)} | {fmt(v.get('psi_cpu', float('nan')), 1)} | "
                  f"{len(sr)} | {fmt(pct(rt, 99), 1)} |")

    print("\n## Seed requests (nginx)\n")
    if reqs:
        win = [r for r in reqs if t0 <= r["msec"] <= t1 + 1]
        rt = [r["rt"] * 1000 for r in win]
        sizes = [r["b"] for r in win]
        status = defaultdict(int)
        for r in win:
            status[r["st"]] += 1
        print(f"{len(win)} requests in the window, {sum(sizes) / 1e6:.0f} MB. "
              f"Seed time p50 / p99 / max: {fmt(pct(rt, 50), 1)} / {fmt(pct(rt, 99), 1)} / {fmt(max(rt, default=float('nan')), 1)} ms. "
              f"Size p50 / p99: {pct(sizes, 50) / 1024:.0f} / {pct(sizes, 99) / 1024:.0f} KB. "
              f"Status: {dict(status)}.")
    else:
        print("No seed access log.")


if __name__ == "__main__":
    main()
