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

# Runs gcfsd and containerd-gcfs-grpc with the flags GKE uses, in the
# private mount namespace start-riptide.sh creates. /run/gcfsd is bound to
# /run/ate/gcfsd here, so the host's own gcfsd, if any, keeps its sockets.
# Exits when either process exits.
#
# On SIGTERM it waits for the plugin to finish draining before it stops
# them, so the images in use stay readable until then.
#
# Environment:
#   GCFSD_BIN, GCFS_SNAPSHOTTER_BIN  Set by start-riptide.sh.
#   KUBE_HOME                        GKE node home, for the metrics credentials.
#   ENABLE_GCFS_V2_API               "true" passes --enable-v2 to the snapshotter.
#   ENABLE_GCFS_LAYER_CACHING        "true" keeps fetched layers on disk.
#   ENABLE_USERNAMESPACE_SUPPORT     "true" with the v2 API enables ID mapping.
#   GCFSD_CACHE_SIZE_MIB             gcfsd memory cache size, split between
#                                    content and large files.

set -o errexit -o nounset -o pipefail

readonly gcfs_root="/var/lib/ate/image-plugin-cache/riptide-gcfs"
readonly gcfsd_mnt_dir="${gcfs_root}/mnt"
readonly gcfs_socket_path="/run/gcfsd/grpc.sock"
readonly keychain_socket_path="/run/gcfsd/keychain.sock"
readonly downloader_socket_path="/run/gcfsd/downloader.sock"
readonly layer_cache_dir="${gcfs_root}/snapshotter/layers"
readonly images_in_use_db_path="${gcfs_root}/gcfsd/images_in_use.db"
readonly sentinel_file_path="${gcfs_root}/gcfsd_sentinel.txt"
readonly snapshotter_socket="/run/ate/gcfs-snapshotter.sock"
readonly drain_complete_file="/run/ate/drain-complete"

mount --make-rslave /run
mount --bind /run/ate/gcfsd /run/gcfsd
mount --make-private /run/gcfsd

gcfsd_flags=(
  --mount_point="${gcfsd_mnt_dir}"
  --control_server_socket_path="${gcfs_socket_path}"
  --downloader_uds_path="${downloader_socket_path}"
  --sentinel_file_path="${sentinel_file_path}"
  --containerd_uds_path=""
  --event_reporter_type=none
  --port=11254
  --images_in_use_db_path="${images_in_use_db_path}"
  --enable_pull_secret_keychain
  --client_name=GKE
  --enable_single_flighting=true
  --read_ahead_max_blocks=10
  --read_ahead_cache_size_ratio=0.1
)
if [[ -n "${GCFSD_CACHE_SIZE_MIB:-}" ]]; then
  each_cache_size=$((GCFSD_CACHE_SIZE_MIB / 2))
  gcfsd_flags+=(
    --max_content_cache_size_mb="${each_cache_size}"
    --max_large_files_cache_size_mb="${each_cache_size}"
  )
fi
if [[ "${ENABLE_GCFS_LAYER_CACHING:-false}" == "true" ]]; then
  gcfsd_flags+=(--layer_cache_dir="${layer_cache_dir}")
fi
if [[ "${ENABLE_GCFS_V2_API:-false}" == "true" && "${ENABLE_USERNAMESPACE_SUPPORT:-false}" == "true" ]]; then
  gcfsd_flags+=(--enable_id_mapping)
fi

snapshotter_flags=(
  --log-level=info
  --config="${gcfs_root}/config.toml"
  --address="${snapshotter_socket}"
  --root="${gcfs_root}"
  --gcfs-socket-path="${gcfs_socket_path}"
  --enable-kubernetes-event-reporting=false
  --disable-duplicate-layer-support=false
  --enable-metric-exporter=false
)
if [[ "${ENABLE_GCFS_V2_API:-false}" == "true" ]]; then
  snapshotter_flags+=(--enable-v2)
fi

mkdir -p "${gcfsd_mnt_dir}" "${layer_cache_dir}" "$(dirname "${images_in_use_db_path}")"

pids=()
cleanup() {
  if [[ "${#pids[@]}" -gt 0 ]]; then
    kill -TERM "${pids[@]}" 2> /dev/null || true
    wait "${pids[@]}" 2> /dev/null || true
  fi
  umount -l "${gcfsd_mnt_dir}" 2> /dev/null || true
  rm -f "${snapshotter_socket}" "${gcfs_socket_path}" "${keychain_socket_path}" "${downloader_socket_path}"
}
on_term() {
  echo "Waiting for the plugin to drain"
  while [[ ! -f "${drain_complete_file}" ]]; do
    sleep 0.5
  done
  exit 0
}
trap cleanup EXIT
trap on_term TERM INT

gcfsd_env=(GOGC=10)
if [[ -f "${KUBE_HOME}/gcfsd_application_default_credentials.json" ]]; then
  gcfsd_env+=(GCFSD_METRICS_ADC_CONFIG_PATH="${KUBE_HOME}/gcfsd_application_default_credentials.json")
fi
env "${gcfsd_env[@]}" "${GCFSD_BIN}" "${gcfsd_flags[@]}" &
pids+=("$!")

for _ in $(seq 1 150); do
  [[ -S "${gcfs_socket_path}" && -S "${keychain_socket_path}" ]] && break
  sleep 0.1
done

touch "${gcfs_root}/config.toml"
snapshotter_env=(HOME=/root)
if [[ -f "${KUBE_HOME}/gcfs_snaphotter_application_default_credentials.json" ]]; then
  snapshotter_env+=(GCFS_SNAPSHOTTER_METRICS_ADC_CONFIG_PATH="${KUBE_HOME}/gcfs_snaphotter_application_default_credentials.json")
fi
env "${snapshotter_env[@]}" "${GCFS_SNAPSHOTTER_BIN}" "${snapshotter_flags[@]}" &
pids+=("$!")

wait -n "${pids[@]}"
