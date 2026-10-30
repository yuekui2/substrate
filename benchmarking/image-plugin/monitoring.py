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

"""Reads per-VM disk and network rates from Cloud Monitoring.

  monitoring.py <project> <start RFC3339> <end RFC3339> <out.jsonl> <node>...

Writes one JSON line per metric, VM and 60 s point, with the rate per second.
GKE node names are the VM instance names.
"""

import json
import subprocess
import sys
import urllib.parse
import urllib.request

METRICS = [
    "compute.googleapis.com/instance/disk/read_bytes_count",
    "compute.googleapis.com/instance/disk/write_bytes_count",
    "compute.googleapis.com/instance/disk/read_ops_count",
    "compute.googleapis.com/instance/disk/write_ops_count",
    "compute.googleapis.com/instance/network/received_bytes_count",
    "compute.googleapis.com/instance/network/sent_bytes_count",
]


def main():
    project, start, end, out = sys.argv[1:5]
    nodes = sys.argv[5:]
    tok = subprocess.check_output(
        ["gcloud", "auth", "application-default", "print-access-token"], text=True).strip()
    names = ",".join(f'"{n}"' for n in nodes)
    n_points = 0
    with open(out, "w") as f:
        for metric in METRICS:
            params = {
                "filter": f'metric.type="{metric}" AND metric.labels.instance_name=one_of({names})',
                "interval.startTime": start,
                "interval.endTime": end,
                "aggregation.alignmentPeriod": "60s",
                "aggregation.perSeriesAligner": "ALIGN_RATE",
            }
            page = ""
            while True:
                q = dict(params)
                if page:
                    q["pageToken"] = page
                url = (f"https://monitoring.googleapis.com/v3/projects/{project}/timeSeries?"
                       + urllib.parse.urlencode(q))
                req = urllib.request.Request(url, headers={
                    "Authorization": f"Bearer {tok}", "x-goog-user-project": project})
                with urllib.request.urlopen(req, timeout=60) as resp:
                    body = json.load(resp)
                for ts in body.get("timeSeries", []):
                    labels = {**ts["metric"].get("labels", {}), **ts["resource"].get("labels", {})}
                    for p in ts.get("points", []):
                        v = p["value"].get("doubleValue", p["value"].get("int64Value"))
                        f.write(json.dumps({
                            "metric": metric.rsplit("/", 2)[-2] + "/" + metric.rsplit("/", 1)[-1],
                            "node": labels.get("instance_name"),
                            "device": labels.get("device_name"),
                            "end": p["interval"]["endTime"],
                            "per_s": float(v),
                        }) + "\n")
                        n_points += 1
                page = body.get("nextPageToken", "")
                if not page:
                    break
    print(f"wrote {n_points} points to {out}")


if __name__ == "__main__":
    main()
