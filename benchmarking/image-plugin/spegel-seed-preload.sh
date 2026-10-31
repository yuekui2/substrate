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

# Preloads the OCI layer blobs of the first N images into the Spegel seeds
# (image-plugin-spegel-seed in image-plugin/manifests/spegel-preload.yaml), so
# bench nodes read them from a seed instead of the registry.
#
#   spegel-seed-preload.sh <images.jsonl> <n>
#
# Every seed gets a peer request for every blob on its own :5000, with
# X-Spegel-Fetch-Upstream. The seed that owns the blob fetches it from the
# registry and stores it. The other seeds answer 404. curl runs on each
# seed's host through its node-sampler pod. Needs crane and jq.

set -euo pipefail

images="${1:?usage: spegel-seed-preload.sh <images.jsonl> <n>}"
n="${2:?usage: spegel-seed-preload.sh <images.jsonl> <n>}"
PARALLEL="${PARALLEL:-16}"
REG=us-central1-docker.pkg.dev

work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT
head -n "${n}" "${images}" | jq -r .oci | while read -r ref; do
  repo="${ref%@*}"
  repo="${repo#"${REG}"/}"
  crane manifest "${ref}" | jq -r --arg r "${repo}" '.layers[] | "\($r) \(.digest)"'
done | sort -u -k2,2 >"${work}/blobs"
total="$(wc -l <"${work}/blobs")"
echo "${total} unique layer blobs in the first ${n} images"

mapfile -t seeds < <(kubectl get nodes -l ate.dev/spegel-seed=true -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}')
[[ ${#seeds[@]} -gt 0 ]] || { echo "no seed nodes" >&2; exit 1; }

# fetch reads "<repo> <digest>" lines on stdin on the seed's host, GETs each
# blob from the local seed, PARALLEL at a time, and prints "<status> <digest>".
read -r -d '' fetch <<EOF || true
xargs -P ${PARALLEL} -L 1 sh -c '
  code="\$(curl -s -o /dev/null -w "%{http_code}" \
    -H "X-Spegel-External-Request: true" -H "X-Spegel-Fetch-Upstream: true" \
    "http://127.0.0.1:5000/v2/\$0/blobs/\$1?ns=${REG}")" || code="curl-exit-\$?"
  echo "\${code} \$1"'
EOF

start="$(date +%s)"
pids=()
for i in "${!seeds[@]}"; do
  pod="$(kubectl -n imgbench get pods -l app=node-sampler --field-selector "spec.nodeName=${seeds[$i]}" -o name)"
  [[ -n "${pod}" ]] || { echo "no node-sampler pod on ${seeds[$i]}" >&2; exit 1; }
  kubectl -n imgbench exec -i "${pod}" -- nsenter -t 1 -m -n -- bash -c "${fetch}" \
    <"${work}/blobs" >"${work}/out-${i}" &
  pids+=($!)
done
rc=0
for pid in "${pids[@]}"; do
  wait "${pid}" || rc=1
done
for i in "${!seeds[@]}"; do
  echo "${seeds[$i]}: $(awk '{print $1}' "${work}/out-${i}" | sort | uniq -c | tr '\n' ' ')"
done
stored="$(cat "${work}"/out-* | awk '$1 == 200 {print $2}' | sort -u | wc -l)"
echo "${stored} of ${total} blobs stored in $(($(date +%s) - start)) s"
if [[ "${stored}" -ne "${total}" ]]; then
  rc=1
fi
exit "${rc}"
