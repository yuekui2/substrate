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

# Sweeps actors per node with cold caches. For each n, recreates the arms'
# pools and runs one rep of 3*n actors, each on a distinct image, in
# <RUN_DIR>/n<n>. Uses the templates already baked at DENSITY (default 8).
#
#   RUN_DIR=... [ARMS="pull riptide spegel"] [DENSITY=8] [TURNS=5] node-sweep.sh <n...>

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
: "${RUN_DIR:?set RUN_DIR}"
EV="${HERE}/ev.sh"
ARMS="${ARMS:-pull riptide nydus spegel}"
DENSITY="${DENSITY:-8}"
TURNS="${TURNS:-5}"
NODES=3
base="${RUN_DIR}"
export DENSITY

for n in "$@"; do
  export RUN_DIR="${base}/n${n}"
  mkdir -p "${RUN_DIR}"
  "${EV}" "n${n}-recreate-pools" "${HERE}/pools.sh recreate ${ARMS}"
  "${EV}" "n${n}-wait-ready" "${HERE}/wait-ready.sh ${ARMS}"
  LOADER_ARGS="-actors $((NODES * n)) -turns ${TURNS} -reads 1000" \
    "${EV}" "n${n}-run" "${HERE}/run.sh rep 1 ${ARMS}"
done
echo NODE-SWEEP-DONE
