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

"""Points the nydus-dragonfly DaemonSets at a nydusd built by build.sh.

  use-nydusd.py <nydusd image> < manifests > manifests

Adds an init container that copies /nydusd from the image to
/home/kubernetes/bin/nydusd-amp and makes the snapshotter start that binary.
The stock release download still installs nydus-image and the other tools.
"""
import sys

import yaml

image = sys.argv[1]
init = {
    "name": "install-nydusd-amp",
    "image": image,
    "command": ["/bin/sh", "-ec",
                "mkdir -p /home/kubernetes/bin && cp /nydusd /home/kubernetes/bin/nydusd-amp.tmp"
                " && mv /home/kubernetes/bin/nydusd-amp.tmp /home/kubernetes/bin/nydusd-amp"],
    "volumeMounts": [{"name": "kube-home", "mountPath": "/home/kubernetes"}],
}
OLD_BIN = 'readonly nydusd_bin="${KUBE_HOME}/bin/nydusd"'
NEW_BIN = 'readonly nydusd_bin="${KUBE_HOME}/bin/nydusd-amp"'
OLD_CHECK = 'if [[ ! -x "${nydusd_bin}" ]]; then'
NEW_CHECK = 'if [[ ! -x "${KUBE_HOME}/bin/nydus-image" ]]; then'

docs = list(yaml.safe_load_all(sys.stdin))
objs = []
for d in docs:
    if d and d.get("kind") == "List":
        objs.extend(d.get("items", []))
    elif d:
        objs.append(d)
patched = 0
for d in objs:
    if d.get("kind") != "DaemonSet" or not d["metadata"]["name"].startswith("image-plugin-nydus-dragonfly"):
        continue
    spec = d["spec"]["template"]["spec"]
    found = False
    for c in spec.get("initContainers", []) + spec.get("containers", []):
        cmd = c.get("command", []) + c.get("args", [])
        # Containers shared through YAML anchors are already patched by an
        # earlier DaemonSet.
        if any(NEW_BIN in s for s in cmd if isinstance(s, str)):
            found = True
        elif any(OLD_BIN in s for s in cmd if isinstance(s, str)):
            for key in ("command", "args"):
                if key in c:
                    c[key] = [s.replace(OLD_BIN, NEW_BIN).replace(OLD_CHECK, NEW_CHECK) if isinstance(s, str) else s
                              for s in c[key]]
            found = True
    if not found:
        sys.exit(f"{d['metadata']['name']}: no nydusd_bin line to patch")
    spec["initContainers"] = [init] + spec.get("initContainers", [])
    patched += 1
if patched == 0:
    sys.exit("no nydus-dragonfly DaemonSet found")
yaml.safe_dump_all(docs, sys.stdout, sort_keys=False)
print(f"patched {patched} DaemonSets", file=sys.stderr)
