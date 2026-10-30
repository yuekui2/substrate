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

# Runs the whole benchmark: bake goldens once (untimed), then for each rep
# run the arms one at a time, rotating their order. Before each arm runs,
# its pool is recreated so node caches start cold. Every step is logged with
# ev.sh. LOADER_ARGS passes through to the loader (see run.sh).
#
#   RUN_DIR=... bench.sh [reps] [arm...]

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
: "${RUN_DIR:?set RUN_DIR}"
REPS="${1:-5}"
shift || true
arms=("$@")
if [[ ${#arms[@]} -eq 0 ]]; then
  arms=(pull riptide nydus spegel)
fi
EV="${HERE}/ev.sh"
export RUN_DIR

"${EV}" bake-goldens "${HERE}/run.sh bake ${arms[*]}"
for rep in $(seq 1 "${REPS}"); do
  n=${#arms[@]}
  for k in $(seq 0 $((n - 1))); do
    arm="${arms[$(((k + rep - 1) % n))]}"
    "${EV}" "rep${rep}-${arm}-recreate-pool" "${HERE}/pools.sh recreate ${arm}"
    "${EV}" "rep${rep}-${arm}-wait-ready" "${HERE}/wait-ready.sh ${arm}"
    "${EV}" "rep${rep}-${arm}-run" "${HERE}/run.sh rep ${rep} ${arm}"
  done
done
echo BENCH-DONE
