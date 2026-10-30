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
# recreate the bench pools so node caches start cold, wait for workers and
# plugins, and run all arms in parallel. Every step is logged with ev.sh.
#
#   RUN_DIR=... bench.sh [reps]

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
: "${RUN_DIR:?set RUN_DIR}"
REPS="${1:-3}"
EV="${HERE}/ev.sh"
export RUN_DIR

"${EV}" bake-goldens "${HERE}/run.sh bake"
for rep in $(seq 1 "${REPS}"); do
  "${EV}" "rep${rep}-recreate-pools" "${HERE}/pools.sh recreate"
  "${EV}" "rep${rep}-wait-ready" "${HERE}/wait-ready.sh"
  "${EV}" "rep${rep}-run" "${HERE}/run.sh rep ${rep}"
done
echo BENCH-DONE
