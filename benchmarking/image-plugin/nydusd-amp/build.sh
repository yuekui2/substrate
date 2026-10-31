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

# Builds nydusd v2.3.0 with user-io-batch-size.patch and pushes an image
# holding it at /nydusd. Prints the pushed reference with its digest.
#
#   build.sh <repo>
#
# The patch lets NYDUS_USER_IO_BATCH_SIZE set how far nydusd extends each
# on-demand fscache read. Stock nydusd fixes it at 1 MiB in fscache mode.
# Needs docker, git and crane.

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$1"
WORK_DIR="${WORK_DIR:-${TMPDIR:-/tmp}/nydusd-amp}"
NYDUS_VERSION=v2.3.0
MUSLRUST=clux/muslrust:1.72.1@sha256:8d5f54894aa50a47d1b0e675aef4475ca3a05ed6d9a8bc9307fbeda6df27bc5f
BUSYBOX=busybox:1.36@sha256:73aaf090f3d85aa34ee199857f03fa3a95c8ede2ffd4cc2cdb5b94e566b11662

mkdir -p "${WORK_DIR}/cargo-registry"
if [[ ! -d "${WORK_DIR}/nydus" ]]; then
  git clone -q --depth 1 --branch "${NYDUS_VERSION}" https://github.com/dragonflyoss/nydus.git "${WORK_DIR}/nydus"
  git -C "${WORK_DIR}/nydus" apply "${HERE}/user-io-batch-size.patch"
fi

docker run --rm -v "${WORK_DIR}/nydus:/nydus-rs" -v "${WORK_DIR}/cargo-registry:/root/.cargo/registry" \
  -w /nydus-rs "${MUSLRUST}" bash -ec '
    rustup target add x86_64-unknown-linux-musl
    cargo build --release --target x86_64-unknown-linux-musl --features=virtiofs --bin nydusd'

ctx="${WORK_DIR}/image"
rm -rf "${ctx}"
mkdir -p "${ctx}"
cp "${WORK_DIR}/nydus/target/x86_64-unknown-linux-musl/release/nydusd" "${ctx}/nydusd"
printf 'FROM %s\nCOPY nydusd /nydusd\n' "${BUSYBOX}" >"${ctx}/Dockerfile"
tag="${REPO}:${NYDUS_VERSION}-amp"
docker build -q -t "${tag}" "${ctx}" >/dev/null
docker push -q "${tag}" >/dev/null
echo "${REPO}@$(crane digest "${tag}")"
