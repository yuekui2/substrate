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

# Bakes the uncompressed set, then runs E3 (no worker CPU limit) and E4
# (uncompressed images), both without the metadata walk.

set -uo pipefail

C="$(dirname "${BASH_SOURCE[0]}")/cpu.sh"
B="$(dirname "${BASH_SOURCE[0]}")/../../.."
lists=()
for f in "${B}"/images-*.jsonl; do
  lists+=(--from-file="${f}")
done
kubectl -n imgbench create configmap imgbench-images "${lists[@]}" --dry-run=client -o yaml |
  kubectl apply --server-side --force-conflicts -f -
export NYDUS_PLUGIN_ARGS=--warm-metadata=false
TAG=-none IMAGES=images-96-none.jsonl "${C}" bake
WORKER_CPU_LIMIT=none "${C}" point e3-nolimit-n32 32
TAG=-none IMAGES=images-96-none.jsonl "${C}" point e4-none-n32 32
echo QUEUE-DONE
