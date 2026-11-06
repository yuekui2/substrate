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

# zstd against uncompressed blobs at 32 actors per node on sets B and C, 2
# reps each. Converts B and C uncompressed in the background while the zstd
# points run, then preloads the seeds, bakes and runs the uncompressed
# points. Plugin without the metadata walk, current ateapi.

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${HERE}/../../../../.." && pwd)"
cd "${ROOT}" || exit 1
# shellcheck source=/dev/null
source .ate-dev-env.sh
export PYTHONPYCACHEPREFIX="${PYCACHE:?set PYCACHE}"
export R_DIR="${HERE}/.."
C="${ROOT}/benchmarking/image-plugin/results/20261104-nydus-cpu/driver/cpu.sh"
B=benchmarking/image-plugin
W="${WORK_ROOT:?set WORK_ROOT}"
mkdir -p "${R_DIR}/setup"

convert() {
  PREFETCH=0 COMPRESSOR=none TAG=none PARALLEL=8 WORK_DIR="${W}/rebuild-$1" \
    TARGET_REPO=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/swebench-nydus-none \
    "${B}/rebuild-nydus.sh" "${B}/images-96-$1.jsonl" 96 "${B}/images-96-none-$1.jsonl"
}
echo "convert $(date -u +%T)"
{ convert b && convert c; } >"${R_DIR}/setup/convert.log" 2>&1 &
conv=$!

for rep in 1 2; do
  for s in b c; do
    echo "zstd ${s} r${rep} $(date -u +%T)"
    TAG="-o${s}" IMAGES="images-96-${s}.jsonl" "${C}" point "${s}-zstd-n32-r${rep}" 32
  done
done

wait "${conv}" || { echo "conversion failed, see setup/convert.log"; exit 1; }
wc -l "${B}"/images-96-none-[bc].jsonl

lists=()
for f in "${B}"/images-*.jsonl; do
  lists+=(--from-file="${f}")
done
kubectl -n imgbench create configmap imgbench-images "${lists[@]}" --dry-run=client -o yaml |
  kubectl apply --server-side --force-conflicts -f -

for s in b c; do
  echo "preload ${s} $(date -u +%T)"
  RUN_DIR="${R_DIR}/setup" DIRECT=1 SAVE_DIR=/var/lib/ate/nydus-seed-blobs \
    "${B}/ev.sh" "preload-none-${s}" "${B}/seed-preload.sh ${B}/images-96-none-${s}.jsonl 96"
  echo "bake ${s} $(date -u +%T)"
  TAG="-none${s}" IMAGES="images-96-none-${s}.jsonl" "${C}" bake
done

for rep in 1 2; do
  for s in b c; do
    echo "none ${s} r${rep} $(date -u +%T)"
    TAG="-none${s}" IMAGES="images-96-none-${s}.jsonl" "${C}" point "${s}-none-n32-r${rep}" 32
  done
done
echo "ALL-DONE $(date -u +%T)"
