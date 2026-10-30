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

# Creates, deletes or recreates the per-arm bench node pools, in parallel.
#
#   pools.sh create|delete|recreate [arm...]
#
# Each arm gets a pool bench-<arm> of 3 x c3-standard-22 on Ubuntu with
# 600 GB Hyperdisk Balanced boot disks at the per-disk maximum IOPS and
# throughput, labeled bench-arm=<arm> and tainted for gVisor workers.
# recreate gives every rep fresh nodes, so node caches start empty.

set -euo pipefail

PROJECT="${PROJECT:-kuiyue-gke-dev}"
ZONE="${ZONE:-us-central1-a}"
CLUSTER="${CLUSTER:-substrate-stream-test}"
NODES="${NODES:-3}"
MACHINE="${MACHINE:-c3-standard-22}"
DISK_TYPE="${DISK_TYPE:-hyperdisk-balanced}"
DISK_SIZE="${DISK_SIZE:-600}"
DISK_IOPS="${DISK_IOPS:-160000}"
DISK_THROUGHPUT="${DISK_THROUGHPUT:-2400}"
ALL_ARMS=(pull riptide nydus spegel)

action="${1:?usage: pools.sh create|delete|recreate [arm...]}"
shift
arms=("$@")
if [[ ${#arms[@]} -eq 0 ]]; then
  arms=("${ALL_ARMS[@]}")
fi

sysctls_file="$(mktemp)"
trap 'rm -f "${sysctls_file}"' EXIT
cat >"${sysctls_file}" <<'EOF'
linuxConfig:
  sysctl:
    net.ipv4.neigh.default.gc_thresh1: "4096"
    net.ipv4.neigh.default.gc_thresh2: "16384"
    net.ipv4.neigh.default.gc_thresh3: "32768"
    fs.inotify.max_user_instances: "65536"
    fs.inotify.max_user_watches: "1048576"
EOF

# GKE rejects a node pool operation while some others run on the cluster.
# Retry those until the other operation finishes.
gke() {
  local out
  for _ in $(seq 1 120); do
    if out="$(gcloud container "$@" 2>&1)"; then
      return 0
    fi
    if [[ "${out}" != *"incompatible operation"* ]]; then
      echo "${out}" >&2
      return 1
    fi
    sleep 20
  done
  echo "${out}" >&2
  return 1
}

exists() {
  gcloud container node-pools describe "bench-$1" --project "${PROJECT}" --zone "${ZONE}" --cluster "${CLUSTER}" >/dev/null 2>&1
}

create_one() {
  local arm="$1"
  exists "${arm}" && return 0
  gke node-pools create "bench-${arm}" \
    --project "${PROJECT}" --zone "${ZONE}" --cluster "${CLUSTER}" \
    --machine-type "${MACHINE}" --num-nodes "${NODES}" \
    --image-type UBUNTU_CONTAINERD --disk-type "${DISK_TYPE}" --disk-size "${DISK_SIZE}" \
    --boot-disk-provisioned-iops "${DISK_IOPS}" --boot-disk-provisioned-throughput "${DISK_THROUGHPUT}" \
    --no-shielded-secure-boot --shielded-integrity-monitoring --no-enable-image-streaming \
    --workload-metadata GKE_METADATA \
    --no-enable-autoupgrade --no-enable-autorepair \
    --node-labels "bench-arm=${arm}" \
    --node-taints "ate.dev/sandboxClass=gvisor:NoSchedule" \
    --system-config-from-file "${sysctls_file}" \
    --quiet
  echo "created bench-${arm}"
}

delete_one() {
  local arm="$1"
  exists "${arm}" || return 0
  gke node-pools delete "bench-${arm}" --project "${PROJECT}" --zone "${ZONE}" --cluster "${CLUSTER}" --quiet
  echo "deleted bench-${arm}"
}

pids=()
for arm in "${arms[@]}"; do
  case "${action}" in
    create) create_one "${arm}" & ;;
    delete) delete_one "${arm}" & ;;
    recreate) { delete_one "${arm}" && create_one "${arm}"; } & ;;
    *) echo "unknown action ${action}" >&2; exit 1 ;;
  esac
  pids+=($!)
done
rc=0
for pid in "${pids[@]}"; do
  wait "${pid}" || rc=1
done
kubectl get nodes -L bench-arm -l bench-arm
exit "${rc}"
