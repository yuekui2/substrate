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

# Builds and installs the per-image file lists the Nydus plugin reads after
# mounting (--warm-list-dir).
#
#   warm-lists.sh gen <images.jsonl> <out-dir>   one <digest hex>.txt per Nydus image
#   warm-lists.sh push <out-dir>                 copy them to every bench nydus node
#
# gen runs prefetch_list.py on each image's bootstrap, so the list is the same
# one rebuild-nydus.sh bakes into rebuilt images. Needs crane, jq and python3.

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=rebuild-nydus.sh
source "${HERE}/rebuild-nydus.sh"
NODE_DIR=/var/lib/ate/nydus-warm-lists
PARALLEL="${PARALLEL:-16}"

list_one() {
  local line="$1" out="$2" ref dir
  ref="$(jq -r .nydus <<<"${line}")"
  dir="$(mktemp -d)"
  fetch_bootstrap "${ref}" "${dir}"
  "${NYDUS}/nydus-image" check -v "${dir}/image/image.boot" 2>/dev/null |
    python3 "${HERE}/prefetch_list.py" --mark-background >"${out}/${ref##*@sha256:}.txt"
  rm -rf "${dir}"
}

case "${1:-}" in
  gen)
    images="$2" out="$3"
    mkdir -p "${out}"
    install_nydus
    tr '\n' '\0' <"${images}" |
      xargs -0 -P "${PARALLEL}" -I{} "${BASH_SOURCE[0]}" list-one {} "${out}"
    echo "$(find "${out}" -name '*.txt' | wc -l) lists in ${out}"
    ;;
  list-one)
    list_one "$2" "$3"
    ;;
  push)
    out="$2"
    for p in $(kubectl -n ate-system get pods -l app=ate-image-plugin,ate.dev/image-plugin=nydus-dragonfly \
      -o name | grep -v seed); do
      tar -C "${out}" -cz . | kubectl -n ate-system exec -i "${p}" -c nydus-snapshotter -- \
        nsenter -t 1 -m -- bash -c "mkdir -p ${NODE_DIR} && tar -xz -C ${NODE_DIR}"
      echo "pushed to ${p}"
    done
    ;;
  *)
    echo "usage: warm-lists.sh gen <images.jsonl> <out-dir> | push <out-dir>" >&2
    exit 2
    ;;
esac
