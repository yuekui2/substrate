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

# Preloads the Nydus blobs of the first N images into the Dragonfly seed
# peers, so bench nodes read them from a seed instead of the registry.
#
#   seed-preload.sh <images.jsonl> <n>
#
# Each blob is fetched whole through one seed's dfdaemon proxy, round robin
# over the seeds. The task ID is the blob digest, the same task nydusd's
# range reads use. Bench nodes find the seed through the scheduler only with
# schedulingPolicy: always (nydus-dragonfly-seeds*.yaml). Needs crane and jq.
#
# With SAVE_DIR set, every seed fetches every blob and also saves it as
# SAVE_DIR/<hex digest> on its node, for nydus-blob-seed to serve
# (nydus-dragonfly-seedhttp.yaml).

set -euo pipefail

images="${1:?usage: seed-preload.sh <images.jsonl> <n>}"
n="${2:?usage: seed-preload.sh <images.jsonl> <n>}"
PARALLEL="${PARALLEL:-16}"
SAVE_DIR="${SAVE_DIR:-}"
REG=us-central1-docker.pkg.dev

blobs="$(mktemp)"
trap 'rm -f "${blobs}"' EXIT
head -n "${n}" "${images}" | jq -r .nydus | while read -r ref; do
  repo="${ref%@*}"
  repo="${repo#"${REG}"/}"
  crane manifest "${ref}" | jq -r --arg r "${repo}" '.layers[] | "\($r) \(.digest)"'
done | sort -u -k2,2 >"${blobs}"
echo "$(wc -l <"${blobs}") unique blobs in the first ${n} images"

mapfile -t seeds < <(kubectl -n ate-system get pods -l app=ate-image-plugin,ate.dev/dragonfly-seed=true -o name)
[[ ${#seeds[@]} -gt 0 ]] || { echo "no seed pods" >&2; exit 1; }

# fetch reads "<repo> <digest>" lines on stdin on the seed's host and GETs
# each blob through the local proxy, PARALLEL at a time.
read -r -d '' fetch <<EOF || true
tok="\$(curl -s -H Metadata-Flavor:Google http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token | sed 's/.*"access_token":"\([^"]*\)".*/\1/')"
export AUTH="Authorization: Basic \$(printf 'oauth2accesstoken:%s' "\${tok}" | base64 -w0)"
export SAVE_DIR="${SAVE_DIR}"
if [ -n "\${SAVE_DIR}" ]; then
  mkdir -p "\${SAVE_DIR}"
fi
xargs -P ${PARALLEL} -L 1 sh -c '
  out=/dev/null
  if [ -n "\${SAVE_DIR}" ]; then
    out="\${SAVE_DIR}/.\${1#sha256:}.tmp"
  fi
  if curl -sf -o "\${out}" -H "\${AUTH}" -H "X-Dragonfly-Registry: https://${REG}" "http://127.0.0.1:4001/v2/\$0/blobs/\$1"; then
    if [ -n "\${SAVE_DIR}" ]; then
      mv "\${out}" "\${SAVE_DIR}/\${1#sha256:}"
    fi
  else
    echo "failed \$1"
  fi'
EOF

start="$(date +%s)"
pids=()
for i in "${!seeds[@]}"; do
  # Round robin over the seeds, or every blob to every seed with SAVE_DIR.
  j="${i}" k="${#seeds[@]}"
  if [[ -n "${SAVE_DIR}" ]]; then
    j=0 k=1
  fi
  awk -v i="${j}" -v k="${k}" 'NR % k == i' "${blobs}" |
    kubectl -n ate-system exec -i "${seeds[$i]}" -c nydus-snapshotter -- nsenter -t 1 -m -n -- bash -c "${fetch}" &
  pids+=($!)
done
rc=0
for pid in "${pids[@]}"; do
  wait "${pid}" || rc=1
done
echo "preloaded in $(($(date +%s) - start)) s"
exit "${rc}"
