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

# Build erofs.ko and cachefiles.ko with on-demand (fscache) support for one
# GKE Ubuntu kernel, for the dragonfly-nydus benchmark arm.
#
#   profiling/nydus-kmods.sh <kernel> <out-dir>    e.g. 6.8.0-1061-gke
#
# GKE's Ubuntu kernels build both modules without CONFIG_CACHEFILES_ONDEMAND
# and CONFIG_EROFS_FS_ONDEMAND, so nydusd's "bind ondemand" on /dev/cachefiles
# fails with EINVAL and kernel EROFS can't mount from fscache. This rebuilds
# the two modules from Ubuntu's source for exactly that kernel, with both
# options on, in a throwaway ubuntu:24.04 pod on the system pool (any node
# works: the build uses the kernel's headers package, not the running kernel).
# Writes <out-dir>/{erofs.ko,cachefiles.ko,kernel}; debug info is stripped so
# both fit in one ConfigMap.
set -euo pipefail

K="${1:?kernel release, e.g. 6.8.0-1061-gke}"
OUTDIR="${2:?output directory}"
SYSTEM_POOL="${SYSTEM_POOL:-substrate-system-e2}"
POD="nydus-kmods-build-$$"
NS=ate-system

cleanup() { kubectl -n "$NS" delete pod "$POD" --ignore-not-found --wait=false >/dev/null; }
trap cleanup EXIT

kubectl -n "$NS" run "$POD" --image=ubuntu:24.04 --restart=Never \
  --overrides="{\"spec\":{\"nodeSelector\":{\"cloud.google.com/gke-nodepool\":\"$SYSTEM_POOL\"}}}" \
  --command -- sleep 7200
kubectl -n "$NS" wait pod "$POD" --for=condition=Ready --timeout=5m

kubectl -n "$NS" exec -i "$POD" -- bash -s "$K" <<'BUILD'
set -euo pipefail
K="$1"
export DEBIAN_FRONTEND=noninteractive
sed -i 's/^Types: deb$/Types: deb deb-src/' /etc/apt/sources.list.d/ubuntu.sources
apt-get update -q
apt-get install -y -q build-essential flex bison bc libelf-dev libssl-dev dpkg-dev kmod \
  "linux-headers-$K" >/dev/null
V=$(dpkg-query -W -f='${Version}' "linux-headers-$K")
echo "kernel $K, Ubuntu package version $V"

mkdir -p /build && cd /build
# The archive keeps only current versions; Launchpad keeps every one.
if ! apt-get source -q "linux-gke=$V" >/dev/null 2>&1; then
  apt-get install -y -q ubuntu-dev-tools >/dev/null
  pull-lp-source linux-gke "$V" >/dev/null
fi
SRC=$(ls -d /build/linux-gke-*/ | head -1)

make -s -C "/lib/modules/$K/build" M="$SRC/fs/cachefiles" modules \
  CONFIG_CACHEFILES=m CONFIG_CACHEFILES_ONDEMAND=y KCFLAGS=-DCONFIG_CACHEFILES_ONDEMAND=1
make -s -C "/lib/modules/$K/build" M="$SRC/fs/erofs" modules \
  CONFIG_EROFS_FS=m CONFIG_EROFS_FS_ONDEMAND=y KCFLAGS=-DCONFIG_EROFS_FS_ONDEMAND=1

mkdir -p /out
cp "$SRC/fs/cachefiles/cachefiles.ko" "$SRC/fs/erofs/erofs.ko" /out/
strip --strip-debug /out/*.ko
echo "$K" > /out/kernel
# The on-demand code must be in: these functions exist only with the options on.
nm /out/cachefiles.ko | grep ' cachefiles_ondemand_daemon_read$' >/dev/null
nm /out/erofs.ko | grep ' erofs_fscache_register_fs$' >/dev/null
for m in /out/*.ko; do
  [[ "$(modinfo -F vermagic "$m" | cut -d' ' -f1)" == "$K" ]] || { echo "bad vermagic: $m"; exit 1; }
done
ls -l /out
BUILD

mkdir -p "$OUTDIR"
for f in erofs.ko cachefiles.ko kernel; do
  kubectl -n "$NS" cp "$POD:/out/$f" "$OUTDIR/$f" >/dev/null
done
size=$(cat "$OUTDIR"/*.ko | wc -c)
(( size < 900000 )) || { echo "modules are $size bytes; too large for one ConfigMap" >&2; exit 1; }
sha256sum "$OUTDIR"/*.ko
