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

# Deploys the benchmark: the loader's RBAC and image list, and per arm its
# atespace, WorkerPool and image plugin (none for pull).
#
#   deploy.sh [arm...]
#
# Needs KO_DOCKER_REPO (from .ate-dev-env.sh) and kubectl-ate in bin/.

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${HERE}/../.." && pwd)"
ALL_ARMS=(pull riptide nydus spegel)
arms=("$@")
if [[ ${#arms[@]} -eq 0 ]]; then
  arms=("${ALL_ARMS[@]}")
fi

cd "${ROOT}"
KO="$(cd hack/tools/ko && go tool -n ko)"
ATE="${ROOT}/bin/kubectl-ate"
OUT="$(mktemp -d)"
trap 'rm -rf "${OUT}"' EXIT

kubectl apply -f "${HERE}/manifests/base.yaml"
image_lists=()
for f in "${HERE}"/images-*.jsonl; do
  image_lists+=(--from-file="${f}")
done
kubectl -n imgbench create configmap imgbench-images "${image_lists[@]}" \
  --dry-run=client -o yaml | kubectl apply --server-side --force-conflicts -f -
kubectl -n imgbench create configmap node-sampler --from-file="${HERE}/node-sampler.sh" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f "${HERE}/manifests/node-sampler.yaml"

manifest_of() {
  case "$1" in
    nydus) echo image-plugin/manifests/nydus-dragonfly.yaml ;;
    *) echo "image-plugin/manifests/$1.yaml" ;;
  esac
}

for arm in "${arms[@]}"; do
  "${ATE}" get atespace "imgbench-${arm}" >/dev/null 2>&1 || "${ATE}" create atespace "imgbench-${arm}"
  # shellcheck disable=SC2016 # envsubst takes the variable names literally
  ARM="${arm}" envsubst '${ARM}' <"${HERE}/manifests/workerpool.yaml.tmpl" >"${OUT}/wp-${arm}.yaml"
  "${KO}" resolve --base-import-paths -f "${OUT}/wp-${arm}.yaml" | kubectl apply -f -
  if [[ "${arm}" != pull ]]; then
    "${KO}" resolve --base-import-paths -f "$(manifest_of "${arm}")" >"${OUT}/${arm}.raw.yaml"
    python3 "${HERE}/scope_plugin.py" "${arm}" <"${OUT}/${arm}.raw.yaml" | kubectl apply -f -
  fi
done

for arm in "${arms[@]}"; do
  if [[ "${arm}" != pull ]]; then
    ds="image-plugin-${arm}"
    [[ "${arm}" == nydus ]] && ds=image-plugin-nydus-dragonfly
    kubectl -n ate-system rollout status "ds/${ds}" --timeout=30m
  fi
  kubectl -n imgbench wait --for=condition=Available "deploy/bench-${arm}" --timeout=30m 2>/dev/null ||
    kubectl -n imgbench get pods -l "workload=bench-${arm}" -o wide | tail -3
done
kubectl -n ate-system get pods -o wide -l app=ate-image-plugin
kubectl -n imgbench get workerpools
