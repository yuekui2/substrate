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

"""Adapts a resolved image plugin manifest for one benchmark arm.

  scope_plugin.py <arm> < resolved.yaml > arm.yaml

- Scopes every DaemonSet to the arm's pool (nodeSelector bench-arm=<arm>).
  With DRAGONFLY_SEEDS=dedicated (the default, as in pools.sh), the Nydus
  seed DaemonSets go to the seed pool, bench-arm=nydus-seed.
  image-plugin-spegel-seed goes to bench-arm=spegel-seed.
- Clears the plugin's cache on every start, so each rep starts cold.
- Riptide: runs gcfsd v1.37-2 and snapshotter v1.37-1 from the riptide-bins
  image, installed by an init container, as in the teammate's run.
- riptide-ms: the same, with the Memorystore gcfsd-v2 build dev-e49ebb5
  and MEMORYSTORE_ADDRESS from VALKEY_EP.
- Nydus: sets DRAGONFLY_PREFETCH, NYDUS_PREFETCH and NYDUS_USER_IO_BATCH_SIZE
  from the environment when they are set, and appends NYDUS_PLUGIN_ARGS to
  the image plugin's args.
"""

import os
import sys

import yaml

RIPTIDE_BINS = (
    "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/riptide-bins"
    "@sha256:e6648c4d79f258df2ad6aea531428ebfc0d7fec5010b7adca48307f3479343f5"
)
# gcfsd-v2 dev-e49ebb5 from gs://kuiyue-gke-dev-riptide/gcfsd-v2-ms-fpt and
# the snapshotter from RIPTIDE_BINS, packed by
# image-plugin/cmd/riptide-bin-image.
RIPTIDE_MS_BINS = (
    "us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/riptide-bins-ms"
    "@sha256:3ff5e4c2db00244db28d437a3e4e334f1acab10d2f1016e9851daecd1edc49d8"
)
BINS = {"riptide": RIPTIDE_BINS, "riptide-ms": RIPTIDE_MS_BINS}
RIPTIDE_BIN_DIR = "/home/kubernetes/bin/ate-riptide"

INSTALL = """dst=/host-bin/ate-riptide
mkdir -p "${dst}"
for f in /riptide/*; do
  n="$(basename "${f}")"
  cp "${f}" "${dst}/.${n}.tmp"
  chmod 0755 "${dst}/.${n}.tmp"
  mv -f "${dst}/.${n}.tmp" "${dst}/${n}"
done
"""


def set_env(container, name, value):
    env = container.setdefault("env", [])
    for e in env:
        if e["name"] == name:
            e["value"] = value
            return
    env.append({"name": name, "value": value})


def patch(doc, arm):
    if doc.get("kind") != "DaemonSet":
        return
    spec = doc["spec"]["template"]["spec"]
    pool = arm
    if (doc["metadata"]["name"] in ("image-plugin-nydus-dragonfly-seed", "nydus-blob-seed")
            and os.environ.get("DRAGONFLY_SEEDS", "dedicated") == "dedicated"):
        pool = "nydus-seed"
    if doc["metadata"]["name"] == "image-plugin-spegel-seed":
        pool = "spegel-seed"
    spec.setdefault("nodeSelector", {})["bench-arm"] = pool
    for c in spec["containers"]:
        for name, container in (
            ("DRAGONFLY_PREFETCH", "dfdaemon"),
            ("NYDUS_PREFETCH", "nydus-snapshotter"),
            ("NYDUS_USER_IO_BATCH_SIZE", "nydus-snapshotter"),
        ):
            if c["name"] == container and name in os.environ:
                set_env(c, name, os.environ[name])
        if c["name"] == "riptide-provider":
            set_env(c, "CLEAR_CACHE_ON_START", "true")
            set_env(c, "RIPTIDE_BIN_DIR", RIPTIDE_BIN_DIR)
            if arm == "riptide-ms":
                set_env(c, "MEMORYSTORE_ADDRESS", os.environ["VALKEY_EP"])
        if c["name"] == "image-plugin-spegel":
            c["args"] = [a for a in c["args"] if not a.startswith("--clear-cache-on-start")]
            c["args"].append("--clear-cache-on-start=true")
        if c["name"] == "image-plugin-dragonfly" and os.environ.get("NYDUS_PLUGIN_ARGS"):
            c["args"] += os.environ["NYDUS_PLUGIN_ARGS"].split()
    if arm in BINS:
        spec.setdefault("initContainers", []).append({
            "name": "install-riptide-bins",
            "image": BINS[arm],
            "command": ["/bin/sh", "-ec", INSTALL],
            "volumeMounts": [{"name": "kube-home-bin", "mountPath": "/host-bin"}],
        })
        spec["volumes"].append({
            "name": "kube-home-bin",
            "hostPath": {"path": "/home/kubernetes/bin", "type": "Directory"},
        })


def main():
    arm = sys.argv[1]
    docs = [d for d in yaml.safe_load_all(sys.stdin) if d]
    for d in docs:
        for item in d.get("items", []) if d.get("kind") == "List" else [d]:
            patch(item, arm)
    yaml.safe_dump_all(docs, sys.stdout, sort_keys=False)


if __name__ == "__main__":
    main()
