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

# seed-preload.sh fills the blob seeds with the blobs and manifests of
# Nydus images, so nydusd and the plugin read them from a seed node instead
# of the registry.
#
#   hack/seed-preload.sh < images.txt
#
# Each line of stdin is a fully qualified image ref, with a tag or a
# digest. The script reads the manifests with crane, then has every seed
# node fetch every config and layer blob through its local dfdaemon into
# /var/lib/ate/nydus-seed-blobs, and copies the manifests to
# /var/lib/ate/nydus-seed-blobs/manifests. nydus-blob-seed serves both.
#
# Credentials: crane uses your local registry credentials. The seed nodes
# use an access token of the node's service account from the GCE metadata
# server, so they can only fetch from Artifact Registry and GCR on GCE.
#
# Needs crane, jq and kubectl. PARALLEL sets the downloads per seed node
# (default 16).

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly dir=/var/lib/ate/nydus-seed-blobs
parallel="${PARALLEL:-16}"

blobs="$(mktemp)"
manifests="$(mktemp -d)"
trap 'rm -rf "${blobs}" "${manifests}"' EXIT
# nginx runs as an unprivileged user and must read the copies.
chmod 755 "${manifests}"

while read -r ref; do
  [[ -n "${ref}" ]] || continue
  registry="${ref%%/*}"
  if [[ "${registry}" != *.* && "${registry}" != *:* ]]; then
    echo "${ref} is not a fully qualified image ref" >&2
    exit 1
  fi
  name="${ref%@*}"
  if [[ "${name##*/}" == *:* ]]; then
    name="${name%:*}"
  fi
  digest="$(crane digest "${ref}")"
  crane manifest "${name}@${digest}" > "${manifests}/${digest#sha256:}"
  jq -e '.layers' "${manifests}/${digest#sha256:}" > /dev/null || {
    echo "${ref} is not an image manifest" >&2
    exit 1
  }
  jq -r --arg reg "${registry}" --arg repo "${name#*/}" \
    '(.config, .layers[]) | "\($reg) \($repo) \(.digest)"' "${manifests}/${digest#sha256:}"
done | sort -u -k3,3 > "${blobs}"
echo "$(wc -l < "${blobs}") unique blobs"

mapfile -t seeds < <(kubectl -n ate-system get pods -l app=ate-image-provider,ate.dev/dragonfly-seed=true -o name)
if [[ "${#seeds[@]}" -eq 0 ]]; then
  echo "no seed pods" >&2
  exit 1
fi

node_script="$(< "${ROOT}/hack/seed-preload-node.sh")"
pids=()
for seed in "${seeds[@]}"; do
  kubectl -n ate-system exec -i "${seed}" -c nydus-snapshotter -- \
    nsenter -t 1 -m -n -- bash -c "${node_script}" seed-preload-node.sh "${dir}" "${parallel}" < "${blobs}" &
  pids+=("$!")
done
rc=0
for pid in "${pids[@]}"; do
  wait "${pid}" || rc=1
done
for seed in "${seeds[@]}"; do
  tar -C "${manifests}" -cf - . |
    kubectl -n ate-system exec -i "${seed}" -c nydus-snapshotter -- \
      nsenter -t 1 -m -- tar -C "${dir}/manifests" --no-same-owner -xf - || rc=1
done
exit "${rc}"
