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

# Rebuilds the Nydus images of the first N entries of IMAGES with a prefetch
# table, which puts the listed files first in each blob, pushes them to
# TARGET_REPO/<name>:pf, and
# writes OUT with the same schema as IMAGES and `nydus` set to the new
# digest. The prefetch list comes from prefetch_list.py run on the existing
# Nydus bootstrap, so the full image is pulled only by nydusify.
#
#   rebuild-nydus.sh IMAGES N OUT
#
# Needs crane, jq and python3, and push access to TARGET_REPO through the
# docker config. WORK_DIR keeps per-image logs and lists.
#
# PREFETCH=0 converts without a prefetch table. COMPRESSOR sets the blob
# compressor: zstd (default), lz4_block or none.

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_REPO="${TARGET_REPO:-us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/swebench-nydus-pf}"
WORK_DIR="${WORK_DIR:-${TMPDIR:-/tmp}/rebuild-nydus}"
PARALLEL="${PARALLEL:-8}"
TAG="${TAG:-pf}"
PREFETCH="${PREFETCH:-1}"
COMPRESSOR="${COMPRESSOR:-zstd}"
# Optional nydus-image chunk size, such as 0x400000.
CHUNK_SIZE="${CHUNK_SIZE:-}"
NYDUS_VERSION=v2.3.0
NYDUS_SHA256=c6bed2e26151ba07236df995553f568b9f12ce4a294df7e9373a30c05381a1a1
NYDUS="${WORK_DIR}/nydus-static"
BOOTSTRAP_ANNOTATION=containerd.io/snapshot/nydus-bootstrap

# Prints the digest of the bootstrap layer of a Nydus image.
bootstrap_digest() {
  crane manifest "$1" | jq -er --arg a "${BOOTSTRAP_ANNOTATION}" \
    '.layers[] | select(.annotations[$a] == "true") | .digest'
}

# Extracts the bootstrap of Nydus image $1 into directory $2.
fetch_bootstrap() {
  mkdir -p "$2"
  crane blob "${1%@*}@$(bootstrap_digest "$1")" | tar xz -C "$2" image/image.boot
}

install_nydus() {
  [[ -x "${NYDUS}/nydusify" ]] && return
  mkdir -p "${WORK_DIR}"
  local tgz="${WORK_DIR}/nydus.tgz"
  curl -fsSL -o "${tgz}" \
    "https://github.com/dragonflyoss/nydus/releases/download/${NYDUS_VERSION}/nydus-static-${NYDUS_VERSION}-linux-amd64.tgz"
  echo "${NYDUS_SHA256}  ${tgz}" | sha256sum -c --quiet
  tar xzf "${tgz}" -C "${WORK_DIR}"
}

# Converts one image and prints its line for OUT.
convert_one() {
  local line="$1"
  local id oci nydus name dir target start secs digest
  id="$(jq -r .instance_id <<<"${line}")"
  oci="$(jq -r .oci <<<"${line}")"
  nydus="$(jq -r .nydus <<<"${line}")"
  name="${oci%@*}"
  name="${name##*/}"
  dir="${WORK_DIR}/${id}"
  target="${TARGET_REPO}/${name}"
  rm -rf "${dir}"
  mkdir -p "${dir}"

  local chunk=()
  if [[ -n "${CHUNK_SIZE}" ]]; then
    chunk=(--chunk-size "${CHUNK_SIZE}")
  fi
  local prefetch=(--prefetch-patterns)
  if [[ "${PREFETCH}" == 1 ]]; then
    fetch_bootstrap "${nydus}" "${dir}/old"
    "${NYDUS}/nydus-image" check -v "${dir}/old/image/image.boot" >"${dir}/old.txt" 2>/dev/null
    grep -q 'metadata is valid' "${dir}/old.txt"
    python3 "${HERE}/prefetch_list.py" <"${dir}/old.txt" >"${dir}/prefetch.txt"
  else
    prefetch=()
    : >"${dir}/prefetch.txt"
  fi
  start="$(date +%s)"
  "${NYDUS}/nydusify" convert \
    --source "${oci}" --target "${target}:${TAG}" "${chunk[@]}" \
    --fs-version 6 --compressor "${COMPRESSOR}" \
    --nydus-image "${NYDUS}/nydus-image" --work-dir "${dir}/tmp" \
    "${prefetch[@]}" <"${dir}/prefetch.txt" >"${dir}/convert.log" 2>&1
  secs=$(($(date +%s) - start))
  rm -rf "${dir}/tmp"

  digest="$(crane digest "${target}:${TAG}")"
  if [[ "${PREFETCH}" == 1 ]]; then
    fetch_bootstrap "${target}@${digest}" "${dir}/new"
    "${NYDUS}/nydus-image" inspect -R prefetch "${dir}/new/image/image.boot" 2>/dev/null >"${dir}/prefetch-table.json"
    if [[ "$(jq length "${dir}/prefetch-table.json")" -lt 2 ]]; then
      echo "${id}: no prefetch table" >&2
      return 1
    fi
  fi
  echo "${id} convert_s=${secs} prefetch_list=$(wc -l <"${dir}/prefetch.txt")" >&2
  jq -c --arg n "${target}@${digest}" '.nydus = $n' <<<"${line}" >"${dir}/out.json"
}

main() {
  if [[ "${1:-}" == convert-one ]]; then
    convert_one "$2"
    return
  fi
  local images="$1" n="$2" out="$3"
  install_nydus
  export HERE TARGET_REPO WORK_DIR NYDUS BOOTSTRAP_ANNOTATION TAG CHUNK_SIZE PREFETCH COMPRESSOR
  head -n "${n}" "${images}" | tr '\n' '\0' |
    xargs -0 -P "${PARALLEL}" -I{} "${BASH_SOURCE[0]}" convert-one {} ||
    echo "some conversions failed, see ${WORK_DIR}/<id>/convert.log" >&2
  local failed=0 id
  : >"${out}"
  while read -r id; do
    if [[ -s "${WORK_DIR}/${id}/out.json" ]]; then
      cat "${WORK_DIR}/${id}/out.json" >>"${out}"
    else
      echo "missing ${id}" >&2
      failed=1
    fi
  done < <(head -n "${n}" "${images}" | jq -r .instance_id)
  return "${failed}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
