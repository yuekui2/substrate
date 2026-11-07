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

# Starts the DaemonSet's own gcfsd and containerd-gcfs-grpc on the node.
#
# The container runs this script from the ConfigMap. It copies the scripts
# to the node and runs itself again in the node's namespaces, so the
# binaries run against the node's libraries. There it stops a previous
# instance, resets the state that does not survive a restart, and runs
# run-gcfs.sh in a private mount namespace.
#
# Environment:
#   KUBE_HOME             GKE node home. Its bin/ holds the preloaded binaries.
#   RIPTIDE_BIN_DIR       Node directory with gcfsd, gcfsd-v2 and
#                         containerd-gcfs-grpc. Defaults to ${KUBE_HOME}/bin.
#   ENABLE_GCFS_V2_API    "true" runs gcfsd-v2 instead of gcfsd.
#   CLEAR_CACHE_ON_START  "true" clears the layer cache at start.
# run-gcfs.sh reads the other settings.

set -o errexit -o nounset -o pipefail

readonly host_script_dir="/run/ate/riptide-scripts"
readonly gcfs_root="/var/lib/ate/image-plugin-cache/riptide-gcfs"
readonly gcfs_run_dir="/run/ate/gcfsd"
readonly gcfsd_mnt_dir="${gcfs_root}/mnt"
readonly plugin_work_dir="/var/lib/ate/image-plugin-cache/riptide"
readonly snapshotter_socket="/run/ate/gcfs-snapshotter.sock"
readonly drain_complete_file="/run/ate/drain-complete"

if [[ "${1:-}" != "--on-node" ]]; then
  script_dir="$(dirname "${BASH_SOURCE[0]}")"
  mkdir -p "${host_script_dir}"
  cp "${script_dir}/start-riptide.sh" "${script_dir}/run-gcfs.sh" "${host_script_dir}/"
  exec nsenter -t 1 -m -u -i -n -- /bin/bash "${host_script_dir}/start-riptide.sh" --on-node
fi

gcfsd_name="gcfsd"
if [[ "${ENABLE_GCFS_V2_API:-false}" == "true" ]]; then
  gcfsd_name="gcfsd-v2"
fi
bin_dir="${RIPTIDE_BIN_DIR:-${KUBE_HOME}/bin}"
export GCFSD_BIN="${bin_dir}/${gcfsd_name}"
export GCFS_SNAPSHOTTER_BIN="${bin_dir}/containerd-gcfs-grpc"
for bin in "${GCFSD_BIN}" "${GCFS_SNAPSHOTTER_BIN}"; do
  if [[ ! -x "${bin}" ]]; then
    echo "Riptide binary ${bin} is missing or not executable" >&2
    exit 1
  fi
done
echo "Using gcfsd ${GCFSD_BIN} and snapshotter ${GCFS_SNAPSHOTTER_BIN}"

# Stop the processes of a previous instance, found by their root dir in
# the command line. The kernel truncates process names to 15 characters.
for name in "${gcfsd_name}" containerd-gcfs-grpc; do
  for pid in $(pgrep -x "${name:0:15}" || true); do
    if tr '\0' ' ' < "/proc/${pid}/cmdline" 2> /dev/null | grep -q "${gcfs_root}"; then
      kill -9 "${pid}" 2> /dev/null || true
    fi
  done
done

umount -l "${gcfsd_mnt_dir}" 2> /dev/null || true

# Keep the layer cache (snapshotter/layers) across restarts so the node
# stays warm. Clear the view state: gcfsd restores its views from
# images_in_use.db, but nothing keeps them in step with the snapshotter's
# metadata, and a restored view makes the snapshotter fall back to a local
# snapshot ("view already exists"). The plugin's per-image directories
# point into the views, so clear them too. Clear the layer cache as well
# when the binaries changed or CLEAR_CACHE_ON_START is "true".
cache_version="$(cat "${GCFSD_BIN}" "${GCFS_SNAPSHOTTER_BIN}" | sha256sum | cut -c1-16)"
clear_reason=""
if [[ "${CLEAR_CACHE_ON_START:-false}" == "true" ]]; then
  clear_reason="CLEAR_CACHE_ON_START=true"
elif [[ "$(cat "${gcfs_root}/version" 2> /dev/null)" != "${cache_version}" ]]; then
  clear_reason="binaries changed, cache version ${cache_version}"
fi
if [[ -n "${clear_reason}" ]]; then
  echo "Clearing the layer cache: ${clear_reason}"
  rm -rf "${gcfs_root}/snapshotter/layers"
fi
rm -rf "${gcfs_root}/gcfsd" "${gcfs_root}/gcfsd_sentinel.txt" \
  "${gcfs_root}/snapshotter/metadata.db" "${gcfs_root}/snapshotter/snapshots" \
  "${plugin_work_dir:?}"/*
mkdir -p "${gcfs_run_dir}" "${gcfs_root}" "${gcfsd_mnt_dir}" /run/gcfsd "${plugin_work_dir}"
echo "${cache_version}" > "${gcfs_root}/version"
rm -f "${gcfs_run_dir}/grpc.sock" "${gcfs_run_dir}/keychain.sock" "${gcfs_run_dir}/downloader.sock" \
  "${snapshotter_socket}" "${drain_complete_file}"

# Share the FUSE mounts under gcfs_root with the host and the plugin
# container.
if ! mountpoint -q "${gcfs_root}"; then
  mount --bind "${gcfs_root}" "${gcfs_root}"
fi
mount --make-rshared "${gcfs_root}"

exec unshare -m --propagation unchanged -- /bin/bash "${host_script_dir}/run-gcfs.sh"
