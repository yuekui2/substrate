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

# seed-preload-node.sh runs on a seed node for seed-preload.sh.
#
#   seed-preload-node.sh DIR PARALLEL < blobs
#
# Each line of stdin is "REGISTRY REPO DIGEST". It fetches each blob that
# is not in DIR through the local dfdaemon, PARALLEL at a time, checks its
# digest and saves it as DIR/<hex digest>. It authenticates with the node
# service account's token from the GCE metadata server.

set -euo pipefail

dir="$1"
parallel="$2"

tok="$(curl -fsS -H Metadata-Flavor:Google \
  http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token |
  sed 's/.*"access_token":"\([^"]*\)".*/\1/')"
auth="Authorization: Basic $(printf 'oauth2accesstoken:%s' "${tok}" | base64 -w0)"

mkdir -p "${dir}/manifests"
failed="$(mktemp)"
trap 'rm -f "${failed}"' EXIT

fetch() {
  local registry="$1" repo="$2" digest="$3"
  local hex="${digest#sha256:}"
  local tmp="${dir}/.${hex}.tmp"
  if ! curl -fsSL -o "${tmp}" -H "${auth}" -H "X-Dragonfly-Registry: https://${registry}" \
    "http://127.0.0.1:4001/v2/${repo}/blobs/${digest}"; then
    echo "failed to fetch ${digest}" >&2
    echo "${digest}" >> "${failed}"
    rm -f "${tmp}"
    return
  fi
  if ! echo "${hex}  ${tmp}" | sha256sum -c --quiet -; then
    echo "${digest}" >> "${failed}"
    rm -f "${tmp}"
    return
  fi
  chmod 644 "${tmp}"
  mv "${tmp}" "${dir}/${hex}"
}

while read -r registry repo digest; do
  if [[ -f "${dir}/${digest#sha256:}" ]]; then
    continue
  fi
  fetch "${registry}" "${repo}" "${digest}" &
  if [[ "$(jobs -rp | wc -l)" -ge "${parallel}" ]]; then
    wait -n
  fi
done
wait
if [[ -s "${failed}" ]]; then
  echo "$(wc -l < "${failed}") blobs failed" >&2
  exit 1
fi
