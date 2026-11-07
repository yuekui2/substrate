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

# build-erofs-modules.sh builds erofs and cachefiles with on-demand fscache
# for the running Ubuntu kernel, which ships them without it. It builds
# them once per node and kernel from the kernel's exact Ubuntu source,
# against the headers in the node image, into ${KUBE_HOME}/bin/modules.
# nydus-snapshotter.sh loads them.

set -euo pipefail

kver="$(uname -r)"
config="/boot/config-${kver}"
out="${KUBE_HOME}/bin/modules/${kver}"
if grep -qx CONFIG_EROFS_FS_ONDEMAND=y "${config}" && grep -qx CONFIG_CACHEFILES_ONDEMAND=y "${config}"; then
  echo "${kver} has on-demand EROFS and cachefiles; nothing to build."
  exit 0
fi
if [[ -f "${out}/erofs.ko" && -f "${out}/cachefiles.ko" ]]; then
  echo "Using existing modules in ${out}."
  exit 0
fi

# /proc/version_signature looks like "Ubuntu 6.8.0-1062.70-gke 6.8.12".
read -r _ sig _ < /proc/version_signature
version="${sig%-*}"
src="linux-${sig##*-}"
launchpad="https://launchpad.net/ubuntu/+archive/primary/+sourcefiles/${src}/${version}"
pool="http://archive.ubuntu.com/ubuntu/pool/main/l/${src}"
echo "Building on-demand erofs and cachefiles for ${kver} from ${src} ${version}."

apt-get update -qq
DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends \
  build-essential ca-certificates curl kmod libelf1t64 patchutils > /dev/null

# The archive pool is fast but drops old diffs. Launchpad keeps every
# version. The .dsc checksums cover both. Some archive servers stall and
# Launchpad sometimes returns 502, so drop a connection slower than 1 MB/s
# for 20s and try again. Each attempt is a new curl, so it resolves the name
# again and may reach a different server. -C - keeps what earlier attempts
# downloaded.
fetch() {
  local name="$1" i base
  shift
  for i in {1..10}; do
    for base in "$@"; do
      curl -fsSL -C - --speed-limit 1000000 --speed-time 20 -o "${name}" "${base}/${name}" && return 0
    done
    echo "Attempt ${i} to download ${name} failed."
    sleep 5
  done
  echo "FATAL: could not download ${name}." >&2
  return 1
}
cd /build
fetch "${src}_${version}.dsc" "${launchpad}"
awk '/^Checksums-Sha256:/ {f=1; next} /^[^ ]/ {f=0} f {print $1 "  " $3}' "${src}_${version}.dsc" > SHA256SUMS
while read -r _ name; do
  fetch "${name}" "${pool}" "${launchpad}"
done < SHA256SUMS
sha256sum -c SHA256SUMS

# Extract only the two module directories, then apply the Ubuntu diff to
# them.
mkdir src
tar -xzf "${src}"_*.orig.tar.gz -C src --strip-components=1 --wildcards '*/fs/erofs/*' '*/fs/cachefiles/*'
zcat "${src}_${version}.diff.gz" | filterdiff -p1 -i 'fs/erofs/*' -i 'fs/cachefiles/*' | patch -d src -p1

# Kbuild reads the rest of the module config from the headers.
sed -i '1i CONFIG_EROFS_FS_ONDEMAND := y\nccflags-y += -DCONFIG_EROFS_FS_ONDEMAND=1' src/fs/erofs/Makefile
sed -i '1i CONFIG_CACHEFILES_ONDEMAND := y\nccflags-y += -DCONFIG_CACHEFILES_ONDEMAND=1' src/fs/cachefiles/Makefile
for m in erofs cachefiles; do
  make -C "/lib/modules/${kver}/build" M="/build/src/fs/${m}" CC="$(uname -m)-linux-gnu-gcc-13" -j"$(nproc)" modules
  vermagic="$(modinfo -F vermagic "src/fs/${m}/${m}.ko")"
  if [[ "${vermagic%% *}" != "${kver}" ]]; then
    echo "FATAL: ${m}.ko vermagic ${vermagic} does not match ${kver}." >&2
    exit 1
  fi
done

rm -rf "${out}.tmp"
mkdir -p "${out}.tmp"
cp src/fs/erofs/erofs.ko src/fs/cachefiles/cachefiles.ko "${out}.tmp/"
rm -rf "${out}"
mv "${out}.tmp" "${out}"
echo "Installed erofs.ko and cachefiles.ko in ${out}."
