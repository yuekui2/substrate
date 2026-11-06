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

# The sweep: for each set (A, B, C) and 16, 24 and 32 actors per node, Nydus
# alone on a fresh pool, then pull, riptide-ms and spegel together.
#
#   sweep.sh [set...]    sets a, b, c (default all)

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"
cd "${ROOT}" || exit 1
# shellcheck source=/dev/null
source .ate-dev-env.sh
B=benchmarking/image-plugin
R="${B}/results/20261103-confirm-sweep"
export KEEP_SEEDS=1 SPEGEL_SEEDS=dedicated DENSITY=2 VALKEY_EP=10.1.15.239:6379
sets=("$@")
[[ ${#sets[@]} -eq 0 ]] && sets=(a b c)
for s in "${sets[@]}"; do
  if [[ "${s}" == a ]]; then
    tag=-pf images=images-96-pf.jsonl
  else
    tag="-pf${s}" images="images-96-pf-${s}.jsonl"
  fi
  RUN_DIR="${R}/set-${s}" TAG="${tag}" IMAGES="${images}" SOLO=nydus \
    ARMS="nydus pull riptide-ms spegel" "${B}/node-sweep.sh" 16 24 32
  for n in 16 24 32; do
    d="${R}/set-${s}/n${n}"
    python3 "${B}/sweep_report.py" "${d}" >"${d}/sweep.md"
    for a in "${d}"/rep-1/*-w2; do
      python3 "${B}/bottleneck.py" "${a}" >"${a}/bottleneck.md"
    done
  done
done
echo SWEEP-DONE
