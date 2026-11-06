#!/usr/bin/env bash
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

# Riptide + Memorystore, gcfsd-v2 dev-e49ebb5, with the metadata walk removed
# from the plugin. Otherwise the same as ../../20261104-riptide-fpt: set A
# (-pf templates), 16, 24 and 32 actors per node, 2 per worker.

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"
cd "${ROOT}" || exit 1
# shellcheck source=/dev/null
source .ate-dev-env.sh
B=benchmarking/image-plugin
R="${B}/results/20261104-riptide-nowalk"
export DENSITY=2 VALKEY_EP=10.1.15.239:6379 PYTHONPYCACHEPREFIX="${PYCACHE:?set PYCACHE}"
mkdir -p "${R}/setup"

# Apply the rebuilt plugin without waiting: a rolling update on used nodes
# crash-loops on busy mounts, and node-sweep.sh recreates the pool anyway.
KO="$(cd hack/tools/ko && go tool -n ko)"
"${KO}" resolve --base-import-paths -f image-plugin/manifests/riptide-ms.yaml 2>"${R}/setup/ko.log" |
  python3 "${B}/scope_plugin.py" riptide-ms >"${R}/setup/riptide-ms.applied.yaml"
kubectl apply -f "${R}/setup/riptide-ms.applied.yaml"

RUN_DIR="${R}" TAG=-pf IMAGES=images-96-pf.jsonl ARMS=riptide-ms "${B}/node-sweep.sh" 16 24 32
for n in 16 24 32; do
  d="${R}/n${n}"
  python3 "${B}/sweep_report.py" "${d}" >"${d}/sweep.md"
  for a in "${d}"/rep-1/*-w2; do
    python3 "${B}/bottleneck.py" "${a}" >"${a}/bottleneck.md"
  done
done
echo RIPTIDE-DONE
