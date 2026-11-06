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

# Repeats at 16 and 24 actors per node on sets A, B and C, 2 reps each,
# with the metadata walk removed from the plugin and ateapi feeding binds to
# its worker cache. Original zstd images, 1-CPU workers.

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
C="${HERE}/../../20261104-nydus-cpu/driver/cpu.sh"
B="${HERE}/../../.."
export R_DIR="${HERE}/.."

lists=()
for f in "${B}"/images-*.jsonl; do
  lists+=(--from-file="${f}")
done
kubectl -n imgbench create configmap imgbench-images "${lists[@]}" --dry-run=client -o yaml |
  kubectl apply --server-side --force-conflicts -f -

# The set C preload must finish before its bake reads the seeds.
until grep -q '^# exit:' "${R_DIR}"/setup/logs/*-preload-c.log 2>/dev/null; do
  sleep 20
done

TAG=-ob IMAGES=images-96-b.jsonl "${C}" bake
TAG=-oc IMAGES=images-96-c.jsonl "${C}" bake

for rep in 1 2; do
  for n in 16 24; do
    TAG=-o IMAGES=images-96.jsonl "${C}" point "a-n${n}-r${rep}" "${n}"
    TAG=-ob IMAGES=images-96-b.jsonl "${C}" point "b-n${n}-r${rep}" "${n}"
    TAG=-oc IMAGES=images-96-c.jsonl "${C}" point "c-n${n}-r${rep}" "${n}"
  done
done
echo QUEUE-DONE
