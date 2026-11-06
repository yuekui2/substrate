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
# With the riptide-ms arm, also empties its Valkey cluster before each run.
#
#   RUN_DIR=... [ARMS="pull riptide spegel"] [DENSITY=8] [TURNS=5] [VALKEY_EP=...]
#     [IMAGES=images-96-pf-b.jsonl] [SOLO=nydus] node-sweep.sh <n...>
#
# IMAGES picks the image list from the imgbench-images ConfigMap. Pair it
# with a TAG that names the atespace its templates were baked in.
#
# SOLO runs that arm alone first, on its own fresh pool, with the other
# arms' pools deleted. Then it deletes the SOLO pool and runs the other arms
# together on fresh pools. This keeps the SOLO arm clear of the others'
# load and keeps at most the larger group's pools up, for the C3 quota.

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
: "${RUN_DIR:?set RUN_DIR}"
EV="${HERE}/ev.sh"
ARMS="${ARMS:-pull riptide nydus spegel}"
DENSITY="${DENSITY:-8}"
TURNS="${TURNS:-5}"
SOLO="${SOLO:-}"
NODES=3
base="${RUN_DIR}"
export DENSITY

# run_group <n> <label> <arms> recreates the arms' pools and runs one rep.
run_group() {
  local n="$1" label="$2" arms="$3"
  "${EV}" "n${n}-${label}recreate-pools" "${HERE}/pools.sh recreate ${arms}"
  "${EV}" "n${n}-${label}wait-ready" "${HERE}/wait-ready.sh ${arms}"
  if [[ " ${arms} " == *" riptide-ms "* ]]; then
    "${EV}" "n${n}-${label}valkey-flush" "${HERE}/valkey-flush.sh" || exit 1
  fi
  LOADER_ARGS="-actors $((NODES * n)) -turns ${TURNS} -reads 1000${IMAGES:+ -images /config/${IMAGES}}" \
    "${EV}" "n${n}-${label}run" "${HERE}/run.sh rep 1 ${arms}"
}

rest=""
for a in ${ARMS}; do
  [[ "${a}" == "${SOLO}" ]] || rest+="${rest:+ }${a}"
done

for n in "$@"; do
  export RUN_DIR="${base}/n${n}"
  mkdir -p "${RUN_DIR}"
  if [[ -z "${SOLO}" ]]; then
    run_group "${n}" "" "${ARMS}"
    continue
  fi
  [[ -n "${rest}" ]] && "${EV}" "n${n}-delete-rest" "${HERE}/pools.sh delete ${rest}"
  run_group "${n}" "${SOLO}-" "${SOLO}"
  if [[ -n "${rest}" ]]; then
    "${EV}" "n${n}-delete-${SOLO}" "${HERE}/pools.sh delete ${SOLO}"
    run_group "${n}" "rest-" "${rest}"
  fi
done
echo NODE-SWEEP-DONE
