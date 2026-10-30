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

# Sweeps actors per worker. Bakes every density's templates in parallel
# (untimed), then for each density recreates all arms' pools so caches start
# cold and runs one rep of all arms at once. Every step is logged with ev.sh.
# Arms run in parallel on their own pools, so they share Artifact Registry
# and the control plane.
#
#   RUN_DIR=... LOADER_ARGS=... [ARMS="nydus spegel"] sweep.sh <density...>

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
: "${RUN_DIR:?set RUN_DIR}"
EV="${HERE}/ev.sh"
export RUN_DIR
ARMS="${ARMS:-pull riptide nydus spegel}"
densities=("$@")
if [[ ${#densities[@]} -eq 0 ]]; then
  densities=(1 2 4 8)
fi

pids=()
for d in "${densities[@]}"; do
  DENSITY="${d}" LOADER_ARGS="-actors 320 -concurrency 48" "${EV}" "bake-d${d}" "${HERE}/run.sh bake ${ARMS}" &
  pids+=($!)
done
for pid in "${pids[@]}"; do
  wait "${pid}"
done

for d in "${densities[@]}"; do
  "${EV}" "d${d}-recreate-pools" "${HERE}/pools.sh recreate ${ARMS}"
  "${EV}" "d${d}-wait-ready" "${HERE}/wait-ready.sh ${ARMS}"
  DENSITY="${d}" "${EV}" "d${d}-run" "${HERE}/run.sh rep 1 ${ARMS}"
done
echo SWEEP-DONE
