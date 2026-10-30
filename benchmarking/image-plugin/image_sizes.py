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

"""Measures the benchmark images from their registry manifests.

  image_sizes.py <images.jsonl> <out-dir> [n]

Reads the first n images (default all), fetches the OCI and Nydus manifests
from Artifact Registry with the gcloud access token, and writes:

  <out-dir>/image-layers.jsonl  one line per image and flavor with its layers
  <out-dir>/image-sizes.json    totals, unique bytes, and how many images
                                share each layer

Manifests carry compressed sizes only. Uncompressed sizes come from the pull
arm's node snapshots after a rep.
"""

import collections
import json
import os
import subprocess
import sys
import urllib.request

ACCEPT = ",".join([
    "application/vnd.oci.image.index.v1+json",
    "application/vnd.docker.distribution.manifest.list.v2+json",
    "application/vnd.oci.image.manifest.v1+json",
    "application/vnd.docker.distribution.manifest.v2+json",
])


def token():
    return subprocess.check_output(["gcloud", "auth", "print-access-token"], text=True).strip()


def fetch(ref, tok):
    host, rest = ref.split("/", 1)
    repo, digest = rest.split("@", 1)
    req = urllib.request.Request(
        f"https://{host}/v2/{repo}/manifests/{digest}",
        headers={"Authorization": f"Bearer {tok}", "Accept": ACCEPT},
    )
    with urllib.request.urlopen(req, timeout=60) as resp:
        m = json.load(resp)
    if "manifests" in m:
        for d in m["manifests"]:
            p = d.get("platform", {})
            if p.get("os") == "linux" and p.get("architecture") == "amd64":
                return fetch(f"{host}/{repo}@{d['digest']}", tok)
        raise ValueError(f"{ref}: no linux/amd64 manifest")
    return m


def summarize(rows):
    total = 0
    layer_size = {}
    shared_by = collections.Counter()
    for r in rows:
        total += r["compressed_bytes"]
        for digest, size in r["layers"]:
            layer_size[digest] = size
            shared_by[digest] += 1
    unique = sum(layer_size.values())
    hist = collections.Counter(shared_by.values())
    shared_bytes = sum(layer_size[d] for d, n in shared_by.items() if n > 1)
    return {
        "images": len(rows),
        "layers_total": sum(len(r["layers"]) for r in rows),
        "layers_unique": len(layer_size),
        "compressed_bytes_total": total,
        "compressed_bytes_unique": unique,
        "dedup_ratio": round(total / unique, 3) if unique else None,
        "bytes_in_shared_layers": shared_bytes,
        "per_image_compressed_bytes": percentiles([r["compressed_bytes"] for r in rows]),
        "layers_shared_by_n_images": {str(k): hist[k] for k in sorted(hist)},
        "top_shared_layers": [
            {"digest": d, "bytes": layer_size[d], "images": n}
            for d, n in shared_by.most_common(10)
        ],
    }


def percentiles(xs):
    xs = sorted(xs)
    if not xs:
        return {}

    def p(q):
        return xs[min(len(xs) - 1, int(q * len(xs)))]

    return {"min": xs[0], "p50": p(0.5), "p95": p(0.95), "max": xs[-1], "mean": sum(xs) // len(xs)}


def main():
    src, out = sys.argv[1], sys.argv[2]
    n = int(sys.argv[3]) if len(sys.argv) > 3 else None
    images = [json.loads(line) for line in open(src) if line.strip()][:n]
    os.makedirs(out, exist_ok=True)
    tok = token()
    rows = {"oci": [], "nydus": []}
    with open(os.path.join(out, "image-layers.jsonl"), "w") as f:
        for img in images:
            for flavor in ("oci", "nydus"):
                ref = img.get(flavor)
                if not ref:
                    continue
                m = fetch(ref, tok)
                layers = [(l["digest"], l["size"]) for l in m.get("layers", [])]
                row = {
                    "instance_id": img["instance_id"],
                    "flavor": flavor,
                    "ref": ref,
                    "layers": layers,
                    "compressed_bytes": sum(s for _, s in layers),
                }
                rows[flavor].append(row)
                f.write(json.dumps(row) + "\n")
    summary = {flavor: summarize(r) for flavor, r in rows.items() if r}
    with open(os.path.join(out, "image-sizes.json"), "w") as f:
        json.dump(summary, f, indent=2)
    print(json.dumps({k: {kk: v[kk] for kk in (
        "images", "layers_unique", "compressed_bytes_total", "compressed_bytes_unique", "dedup_ratio")}
        for k, v in summary.items()}, indent=2))


if __name__ == "__main__":
    main()
