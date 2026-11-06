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

# Smoke test: one Nydus run at 16 actors per node on set A, alone, to check
# the new sampler, the seed access log and the nydusd blob metrics.

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"
cd "${ROOT}" || exit 1
# shellcheck source=/dev/null
source .ate-dev-env.sh
B=benchmarking/image-plugin
export RUN_DIR="${B}/results/20261103-confirm-sweep/smoke"
export KEEP_SEEDS=1 SPEGEL_SEEDS=dedicated DENSITY=2 TAG=-pf IMAGES=images-96-pf.jsonl
mkdir -p "${RUN_DIR}"
"${B}/ev.sh" delete-rest "${B}/pools.sh delete pull riptide-ms spegel" || exit 1
ARMS=nydus "${B}/node-sweep.sh" 16
python3 "${B}/bottleneck.py" "${RUN_DIR}/n16/rep-1/nydus-pf-w2" >"${RUN_DIR}/n16/bottleneck.md"
python3 "${B}/sweep_report.py" "${RUN_DIR}/n16" >"${RUN_DIR}/n16/sweep.md"
echo SMOKE-DONE
