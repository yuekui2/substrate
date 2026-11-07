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

# nydus-snapshotter.sh runs containerd-nydus-grpc on the host, with one
# shared nydusd in EROFS over fscache mode. nydusd reads blobs from the blob
# seeds first, if the nydus-blob-seed Service exists, then through the
# node's dfdaemon. It fails without on-demand fscache, since there is no
# FUSE fallback.
#
# Environment:
#   KUBE_HOME     the host directory to install binaries and modules in.
#   REGISTRY_URL  the registry nydusd reads blobs from through dfdaemon.

set -euo pipefail

# Run the rest in the host's namespaces, so nydusd mounts EROFS where
# containerd and the plugin see it.
if [[ "${NYDUS_IN_HOST_NS:-}" != 1 ]]; then
  NYDUS_IN_HOST_NS=1 exec nsenter -t 1 -m -u -i -n -- /bin/bash -s < "${BASH_SOURCE[0]}"
fi

: "${KUBE_HOME:?}" "${REGISTRY_URL:?}"

readonly nydus_version=2.3.0
readonly snapshotter_version=0.15.0
readonly nydus_root=/var/lib/ate/image-plugin-cache/nydus-snapshotter
readonly nydus_run=/run/containerd-nydus
readonly nydusd_bin="${KUBE_HOME}/bin/nydusd"
readonly snapshotter_bin="${KUBE_HOME}/bin/containerd-nydus-grpc"
readonly df_addr=http://127.0.0.1:4001
readonly df_health=http://127.0.0.1:4003/healthy

case "$(uname -m)" in
  x86_64)
    arch=amd64
    nydus_sha256=c6bed2e26151ba07236df995553f568b9f12ce4a294df7e9373a30c05381a1a1
    snapshotter_sha256=141df7de067500cdf0878783547c2f114dc27f10971febb5db99487a567f405b
    ;;
  aarch64)
    arch=arm64
    nydus_sha256=bcaa95bae842d0b14433fbcc0dfefa64c61de808ea8121b5beafcf2504bfbbe4
    snapshotter_sha256=63803c22241973fd4dc0fb92f036342e170ac668e06704b661f51b6e17630bcf
    ;;
  *)
    echo "FATAL: unsupported architecture $(uname -m)." >&2
    exit 1
    ;;
esac

rm -f "${nydus_run}/containerd-nydus-grpc.sock"

# build-erofs-modules.sh leaves on-demand builds here unless the kernel
# already has on-demand fscache. Stock cachefiles also exports
# cachefiles_ondemand_* tracepoints, so match whole function names that
# only on-demand builds contain.
has_ondemand() {
  grep -qw cachefiles_ondemand_init_object /proc/kallsyms && grep -qw erofs_fscache_register_fs /proc/kallsyms
}
mod_dir="${KUBE_HOME}/bin/modules/$(uname -r)"
if [[ -f "${mod_dir}/erofs.ko" ]] && ! has_ondemand; then
  mapfile -t deps < <(modinfo -F depends "${mod_dir}/erofs.ko" "${mod_dir}/cachefiles.ko" | tr ',' '\n' | sed '/^$/d' | sort -u)
  if [[ "${#deps[@]}" -gt 0 ]]; then
    modprobe -a "${deps[@]}"
  fi
  for m in cachefiles erofs; do
    if [[ -d "/sys/module/${m}" ]]; then
      rmmod "${m}"
    fi
  done
  insmod "${mod_dir}/erofs.ko"
  insmod "${mod_dir}/cachefiles.ko"
else
  modprobe erofs
  modprobe cachefiles
fi
if ! has_ondemand; then
  echo "FATAL: erofs or cachefiles lacks on-demand fscache support." >&2
  exit 1
fi
if ! grep -qw erofs /proc/filesystems; then
  echo "FATAL: erofs is not in /proc/filesystems." >&2
  exit 1
fi
if [[ ! -e /dev/cachefiles ]]; then
  echo "FATAL: /dev/cachefiles does not exist." >&2
  exit 1
fi

# install_release URL SHA256 MEMBER... extracts MEMBERs of the tarball at URL into
# ${KUBE_HOME}/bin after checking its digest.
install_release() {
  local url="$1" sum="$2" tgz
  shift 2
  tgz="$(mktemp)"
  curl -fsSL -o "${tgz}" "${url}"
  echo "${sum}  ${tgz}" | sha256sum -c -
  tar -xzf "${tgz}" -C "${KUBE_HOME}/bin" --strip-components=1 "$@"
  rm -f "${tgz}"
}
mkdir -p "${KUBE_HOME}/bin" "${nydus_run}/docker"
if [[ ! -x "${nydusd_bin}" ]]; then
  install_release "https://github.com/dragonflyoss/nydus/releases/download/v${nydus_version}/nydus-static-v${nydus_version}-linux-${arch}.tgz" \
    "${nydus_sha256}" nydus-static/nydusd nydus-static/nydus-image nydus-static/nydusify nydus-static/nydusctl
fi
if [[ ! -x "${snapshotter_bin}" ]]; then
  install_release "https://github.com/containerd/nydus-snapshotter/releases/download/v${snapshotter_version}/nydus-snapshotter-v${snapshotter_version}-linux-${arch}.tar.gz" \
    "${snapshotter_sha256}" bin/containerd-nydus-grpc bin/nydus-overlayfs
fi

echo "Waiting for dfdaemon at ${df_health}."
until curl -fsS "${df_health}" > /dev/null 2>&1; do
  sleep 1
done

# The plugin writes registry credentials here for nydusd.
export DOCKER_CONFIG="${nydus_run}/docker"
if [[ ! -f "${DOCKER_CONFIG}/config.json" ]]; then
  echo '{"auths":{}}' > "${DOCKER_CONFIG}/config.json"
fi

# EROFS mounts and snapshots die with the old nydusd, so start clean.
awk '$3 == "erofs" {print $2}' /proc/mounts | while read -r mnt; do
  umount -l "${mnt}" || true
done
rm -rf "${nydus_root:?}"/* /var/lib/ate/image-plugin-cache/nydus/* 2> /dev/null || true
mkdir -p "${nydus_root}/snapshots" "${nydus_root}/cache/metadata" "${nydus_root}/mnt"

mirrors=""
# Kubernetes sets NYDUS_BLOB_SEED_SERVICE_{HOST,PORT} from the Service. The
# host's resolver cannot resolve cluster names.
if [[ -n "${NYDUS_BLOB_SEED_SERVICE_HOST:-}" ]]; then
  # shellcheck disable=SC2153
  mirrors="{\"host\": \"http://${NYDUS_BLOB_SEED_SERVICE_HOST}:${NYDUS_BLOB_SEED_SERVICE_PORT}\", \"health_check_interval\": 5, \"failure_limit\": 5},"
fi
cat > "${nydus_root}/nydusd-config.json" << EOF
{
  "type": "bootstrap",
  "domain_id": "ate-shared",
  "config": {
    "backend_type": "registry",
    "backend_config": {
      "scheme": "https",
      "mirrors": [
        ${mirrors}
        {
          "host": "${df_addr}",
          "ping_url": "${df_health}",
          "headers": {"X-Dragonfly-Registry": "${REGISTRY_URL}"},
          "health_check_interval": 5,
          "failure_limit": 5
        }
      ],
      "timeout": 60,
      "connect_timeout": 10,
      "retry_limit": 5
    },
    "cache_type": "fscache",
    "cache_config": {"work_dir": "${nydus_root}/cache"},
    "prefetch_config": {"enable": false},
    "metadata_path": "${nydus_root}/cache/metadata"
  }
}
EOF

# The fscache worker threads of the shared nydusd bound how many cache
# misses it fetches at once.
cat > "${nydus_root}/config.toml" << EOF
version = 1
[daemon]
threads_number = 64
[metrics]
address = ":9110"
EOF

exec "${snapshotter_bin}" \
  --config="${nydus_root}/config.toml" \
  --log-level=info \
  --log-to-stdout \
  --fs-driver=fscache \
  --daemon-mode=shared \
  --nydusd="${nydusd_bin}" \
  --nydusd-config="${nydus_root}/nydusd-config.json" \
  --address="${nydus_run}/containerd-nydus-grpc.sock" \
  --root="${nydus_root}"
