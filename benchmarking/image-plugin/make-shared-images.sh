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

# Builds N images that share one base image and differ by one small layer,
# converts each to Nydus, and writes OUT with the schema of images-320.jsonl.
#
#   make-shared-images.sh <base instance_id> <n> <out.jsonl>
#
# The base comes from IMAGES. Image i appends FILES generated .py files of
# FILE_KB KiB each, unique per image, under /testbed/0shared. The guest walks
# /testbed in sorted order, so it reads these right after the top-level .py
# files. The OCI images go to OCI_REPO/<id>, the Nydus ones (RAFS v6, zstd,
# with a prefetch table, as in rebuild-nydus.sh) to NYDUS_REPO/<id>.
#
# Every image has the same paths, so one prefetch list serves all of them.
# It comes from the bootstrap of image 0 converted without one, pushed to
# NYDUS_REPO/<id>:plain. At the end the script checks that all Nydus images
# share the blobs of the base layers.
#
# Needs crane, jq and python3, and push access through the docker config.
# WORK_DIR keeps per-image logs. Use a disk-backed WORK_DIR: each running
# conversion needs several times the base image size.

set -euo pipefail
shopt -s inherit_errexit

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGES="${IMAGES:-${HERE}/images-320.jsonl}"
REPO=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo
OCI_REPO="${OCI_REPO:-${REPO}/swebench-shared}"
NYDUS_REPO="${NYDUS_REPO:-${REPO}/swebench-shared-nydus}"
WORK_DIR="${WORK_DIR:-${TMPDIR:-/tmp}/make-shared-images}"
PARALLEL="${PARALLEL:-8}"
FILES="${FILES:-256}"
FILE_KB="${FILE_KB:-16}"
NYDUS_VERSION=v2.3.0
NYDUS_SHA256=c6bed2e26151ba07236df995553f568b9f12ce4a294df7e9373a30c05381a1a1
NYDUS="${WORK_DIR}/nydus-static"
PREFETCH="${WORK_DIR}/prefetch.txt"
BOOTSTRAP_ANNOTATION=containerd.io/snapshot/nydus-bootstrap

install_nydus() {
  [[ -x "${NYDUS}/nydusify" ]] && return
  mkdir -p "${WORK_DIR}"
  local tgz="${WORK_DIR}/nydus.tgz"
  curl -fsSL -o "${tgz}" \
    "https://github.com/dragonflyoss/nydus/releases/download/${NYDUS_VERSION}/nydus-static-${NYDUS_VERSION}-linux-amd64.tgz"
  echo "${NYDUS_SHA256}  ${tgz}" | sha256sum -c --quiet
  tar xzf "${tgz}" -C "${WORK_DIR}"
}

image_id() {
  printf '%s-s%03d' "$1" "$2"
}

# Writes the unique files of image $1 under directory $2.
gen_files() {
  python3 - "$1" "$2" "${FILES}" "${FILE_KB}" <<'EOF'
import hashlib, os, sys
seed, root, files, kb = sys.argv[1], sys.argv[2], int(sys.argv[3]), int(sys.argv[4])
d = os.path.join(root, "testbed", "0shared")
os.makedirs(d)
for j in range(files):
    lines, size, k = [], 0, 0
    while size < kb * 1024:
        h = hashlib.sha256(f"{seed}/{j}/{k}".encode()).hexdigest()
        line = f"X_{k} = '{h}'\n"
        lines.append(line)
        size += len(line)
        k += 1
    with open(os.path.join(d, f"m{j:04d}.py"), "w") as f:
        f.writelines(lines)
EOF
}

# Pushes OCI image $2 (id) on top of base $1 and prints its reference by
# digest.
build_oci() {
  local base="$1" id="$2"
  local dir="${WORK_DIR}/${id}"
  rm -rf "${dir}"
  mkdir -p "${dir}/root"
  gen_files "${id}" "${dir}/root"
  tar -C "${dir}/root" --owner=0 --group=0 --numeric-owner --sort=name \
    --mode=a+rX,u+w,go-w --mtime=@0 -cf "${dir}/layer.tar" testbed/0shared
  crane append -b "${base}" -f "${dir}/layer.tar" -t "${OCI_REPO}/${id}:latest" >"${dir}/append.log" 2>&1
  rm -rf "${dir}/root" "${dir}/layer.tar"
  echo "${OCI_REPO}/${id}@$(crane digest "${OCI_REPO}/${id}:latest")"
}

# Converts OCI image $1 to Nydus image $2 and logs to $3. Extra arguments go
# to nydusify; stdin is passed through for --prefetch-patterns.
convert() {
  local src="$1" dst="$2" dir="$3"
  shift 3
  "${NYDUS}/nydusify" convert --source "${src}" --target "${dst}" \
    --fs-version 6 --compressor zstd \
    --nydus-image "${NYDUS}/nydus-image" --work-dir "${dir}/tmp" "$@" >"${dir}/convert.log" 2>&1
  rm -rf "${dir}/tmp"
}

# Extracts the bootstrap of Nydus image $1 into directory $2.
fetch_bootstrap() {
  local d
  d="$(crane manifest "$1" | jq -er --arg a "${BOOTSTRAP_ANNOTATION}" \
    '.layers[] | select(.annotations[$a] == "true") | .digest')"
  mkdir -p "$2"
  crane blob "${1%@*}@${d}" | tar xz -C "$2" image/image.boot
}

# Writes PREFETCH from image 0 of base $1 converted without a prefetch table.
make_prefetch() {
  local base="$1" id="$2"
  local dir="${WORK_DIR}/${id}" oci plain
  oci="$(build_oci "${base}" "${id}")"
  convert "${oci}" "${NYDUS_REPO}/${id}:plain" "${dir}" </dev/null
  plain="${NYDUS_REPO}/${id}@$(crane digest "${NYDUS_REPO}/${id}:plain")"
  fetch_bootstrap "${plain}" "${dir}/plain"
  "${NYDUS}/nydus-image" check -v "${dir}/plain/image/image.boot" >"${dir}/plain.txt" 2>/dev/null
  grep -q 'metadata is valid' "${dir}/plain.txt"
  python3 "${HERE}/prefetch_list.py" <"${dir}/plain.txt" >"${PREFETCH}.new"
  grep -q '^/testbed/0shared/' "${PREFETCH}.new"
  mv "${PREFETCH}.new" "${PREFETCH}"
}

# Builds and converts image $2 (index) from base $1, and writes its line.
build_one() {
  local base="$1" i="$2" base_id="$3"
  local id dir oci nydus manifest
  id="$(image_id "${base_id}" "${i}")"
  dir="${WORK_DIR}/${id}"
  oci="$(build_oci "${base}" "${id}")"
  convert "${oci}" "${NYDUS_REPO}/${id}:latest" "${dir}" --prefetch-patterns <"${PREFETCH}"
  nydus="${NYDUS_REPO}/${id}@$(crane digest "${NYDUS_REPO}/${id}:latest")"
  fetch_bootstrap "${nydus}" "${dir}/new"
  "${NYDUS}/nydus-image" inspect -R prefetch "${dir}/new/image/image.boot" 2>/dev/null >"${dir}/prefetch-table.json"
  if [[ "$(jq length "${dir}/prefetch-table.json")" -lt 2 ]]; then
    echo "${id}: no prefetch table" >&2
    return 1
  fi
  manifest="$(crane manifest "${oci}")"
  jq -nc --arg id "${id}" --arg oci "${oci}" --arg nydus "${nydus}" \
    --argjson bytes "$(jq '[.layers[].size] | add' <<<"${manifest}")" \
    --argjson layers "$(jq '.layers | length' <<<"${manifest}")" \
    '{instance_id: $id, oci: $oci, nydus: $nydus, compressed_bytes: $bytes, layers: $layers}' >"${dir}/out.json"
  echo "${id} done" >&2
}

# Checks that the Nydus images in OUT $1 list the same blobs for the base
# layers: every blob but the last one and the bootstrap.
check_shared() {
  local lists
  lists="$(jq -r .nydus "$1" | while read -r ref; do
    crane manifest "${ref}" | jq -c --arg a "${BOOTSTRAP_ANNOTATION}" \
      '[.layers[] | select(.annotations[$a] != "true") | .digest] | .[:-1]'
  done | sort | uniq -c)"
  echo "${lists}" >&2
  [[ "$(wc -l <<<"${lists}")" -eq 1 ]]
}

main() {
  if [[ "${1:-}" == build-one ]]; then
    build_one "$2" "$3" "$4"
    return
  fi
  local base_id="$1" n="$2" out="$3" base
  base="$(jq -r --arg id "${base_id}" 'select(.instance_id == $id) | .oci' "${IMAGES}")"
  [[ -n "${base}" ]] || { echo "no ${base_id} in ${IMAGES}" >&2; return 1; }
  install_nydus
  make_prefetch "${base}" "$(image_id "${base_id}" 0)"
  echo "prefetch list: $(wc -l <"${PREFETCH}") files" >&2
  export HERE WORK_DIR NYDUS OCI_REPO NYDUS_REPO FILES FILE_KB PREFETCH BOOTSTRAP_ANNOTATION
  seq 0 $((n - 1)) | xargs -P "${PARALLEL}" -I{} "${BASH_SOURCE[0]}" build-one "${base}" {} "${base_id}" ||
    echo "some images failed, see ${WORK_DIR}/<id>/*.log" >&2
  local failed=0 i id
  : >"${out}"
  for ((i = 0; i < n; i++)); do
    id="$(image_id "${base_id}" "${i}")"
    if [[ -s "${WORK_DIR}/${id}/out.json" ]]; then
      cat "${WORK_DIR}/${id}/out.json" >>"${out}"
    else
      echo "missing ${id}" >&2
      failed=1
    fi
  done
  check_shared "${out}" || { echo "base blobs differ across images" >&2; failed=1; }
  return "${failed}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
