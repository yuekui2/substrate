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

# Riptide + Memorystore with gcfsd-v2 dev-e49ebb5 (gcfsd-v2-ms-fpt), set A,
# 16, 24 and 32 actors per node, 2 per worker, workers balanced by
# wait-ready.sh. Uses the -pf templates: Riptide reads the OCI refs, which
# are the same in images-96-pf.jsonl and images-96.jsonl.

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"
cd "${ROOT}" || exit 1
# shellcheck source=/dev/null
source .ate-dev-env.sh
B=benchmarking/image-plugin
R="${B}/results/20261104-riptide-fpt"
export DENSITY=2 VALKEY_EP=10.1.15.239:6379 PYTHONPYCACHEPREFIX="${PYCACHE:?set PYCACHE}"
mkdir -p "${R}/setup"
[[ "${SKIP_DEPLOY:-0}" == 1 ]] || RUN_DIR="${R}/setup" "${B}/ev.sh" deploy "${B}/deploy.sh riptide-ms"
RUN_DIR="${R}" TAG=-pf IMAGES=images-96-pf.jsonl ARMS=riptide-ms "${B}/node-sweep.sh" 16 24 32
for n in 16 24 32; do
  d="${R}/n${n}"
  python3 "${B}/sweep_report.py" "${d}" >"${d}/sweep.md"
  for a in "${d}"/rep-1/*-w2; do
    python3 "${B}/bottleneck.py" "${a}" >"${a}/bottleneck.md"
  done
done
echo RIPTIDE-DONE
