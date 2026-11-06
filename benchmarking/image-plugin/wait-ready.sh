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

# Waits until the arms' workers, image plugin, node sampler and atelet are
# ready, for example after pools.sh recreate.
#
# Worker pods land on whichever bench nodes are Ready first, for example
# 21/6/21, and actors stay where their first turn ran. So once all NODES
# nodes are Ready, this moves idle workers until each node runs the same
# number, give or take one. It cordons the nodes at the target and deletes
# the extra workers, so their replacements land on the other nodes.
#
#   wait-ready.sh [arm...]

set -euo pipefail

NODES="${NODES:-3}"

arms=("$@")
if [[ ${#arms[@]} -eq 0 ]]; then
  arms=(pull riptide nydus spegel)
fi

# Prints "<count> <node>" per bench node of arm $1, for its Running workers.
worker_counts() {
  {
    kubectl get nodes -l "bench-arm=$1" -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
    kubectl -n imgbench get pods -l "ate.dev/worker-pool=bench-$1" \
      --field-selector=status.phase=Running -o jsonpath='{range .items[*]}{.spec.nodeName}{"\n"}{end}'
  } | sort | uniq -c | awk '{print $1 - 1, $2}'
}

balance_workers() {
  local arm="$1" total target count node spread
  trap 'kubectl uncordon -l "bench-arm=${arm}" >/dev/null' EXIT
  for attempt in 1 2 3; do
    spread="$(worker_counts "${arm}" | sort -n)"
    total="$(awk '{s += $1} END {print s}' <<<"${spread}")"
    target=$(((total + NODES - 1) / NODES))
    if (($(tail -1 <<<"${spread}" | cut -d' ' -f1) - $(head -1 <<<"${spread}" | cut -d' ' -f1) <= 1)); then
      echo "bench-${arm} workers per node: $(cut -d' ' -f1 <<<"${spread}" | tr '\n' ' ')"
      return 0
    fi
    echo "bench-${arm} workers per node: $(cut -d' ' -f1 <<<"${spread}" | tr '\n' ' '); moving the extra ones (attempt ${attempt})"
    while read -r count node; do
      if ((count >= target)); then
        kubectl cordon "${node}" >/dev/null
      fi
    done <<<"${spread}"
    while read -r count node; do
      ((count > target)) || continue
      kubectl -n imgbench get pods -l "ate.dev/worker-pool=bench-${arm}" --field-selector "spec.nodeName=${node}" -o name |
        awk -v n=$((count - target)) 'NR <= n' | xargs kubectl -n imgbench delete --timeout=10m
    done <<<"${spread}"
    kubectl -n imgbench rollout status "deploy/bench-${arm}" --timeout=20m
    kubectl uncordon -l "bench-arm=${arm}" >/dev/null
  done
  echo "bench-${arm} workers are still uneven: $(cut -d' ' -f1 <<<"${spread}" | tr '\n' ' ')" >&2
  return 1
}

for arm in "${arms[@]}"; do
  kubectl wait --for=condition=Ready node -l "bench-arm=${arm}" --timeout=40m
  [[ "$(kubectl get nodes -l "bench-arm=${arm}" --no-headers | wc -l)" -eq "${NODES}" ]] || {
    echo "bench-arm=${arm} does not have ${NODES} nodes" >&2
    exit 1
  }
  kubectl -n imgbench rollout status "deploy/bench-${arm}" --timeout=40m
  balance_workers "${arm}"
  if [[ "${arm}" != pull ]]; then
    ds="image-plugin-${arm}"
    [[ "${arm}" == nydus ]] && ds=image-plugin-nydus-dragonfly
    kubectl -n ate-system rollout status "ds/${ds}" --timeout=40m
    if [[ "${arm}" == nydus ]]; then
      kubectl -n ate-system rollout status ds/image-plugin-nydus-dragonfly-seed --timeout=40m
    fi
  fi
done
kubectl -n ate-system rollout status ds/atelet --timeout=20m
kubectl -n imgbench rollout status ds/node-sampler --timeout=10m
kubectl -n imgbench get workerpools
kubectl -n ate-system get pods -o wide -l app=ate-image-plugin
