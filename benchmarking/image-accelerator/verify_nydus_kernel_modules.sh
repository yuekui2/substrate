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

# Deploys ds/ate-accelerator-nydus-dragonfly onto a fresh GKE Ubuntu NODE and
# PEER_NODE, and records how build-erofs-modules builds and loads on-demand
# erofs and cachefiles. Every command and its output goes to
# results/nydus_kernel_modules.log.
#
# Usage: NODE=<fresh ubuntu node> [PEER_NODE=<seed node>] ./verify_nydus_kernel_modules.sh

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
BENCH_DIR="${ROOT}/benchmarking/image-accelerator"
cd "${ROOT}"
source "${BENCH_DIR}/built_images.env"

NODE="${NODE:?set NODE to a fresh GKE Ubuntu node}"
PEER_NODE="${PEER_NODE:-gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc}"
HELPER="bench-helper-${NODE##*-}"
PEER_HELPER="bench-helper-${PEER_NODE##*-}"
LOG="${BENCH_DIR}/results/nydus_kernel_modules.log"
SEL="app=ate-image-accelerator,ate.dev/image-accelerator=nydus-dragonfly"
: > "${LOG}"

section() { printf '\n=== %s ===\n' "$*" | tee -a "${LOG}"; }
run() {
  printf '\n$ %s\n' "$*" | tee -a "${LOG}"
  "$@" 2>&1 | tee -a "${LOG}"
}
on_host() { run kubectl -n ate-system exec "$1" -- nsenter -t 1 -m -u -i -n -- sh -c "$2"; }
pod_on() { kubectl -n ate-system get pods -l "${SEL}" --field-selector "spec.nodeName=$1,status.phase!=Failed" -o name | head -1; }

section "[1] Fresh node"
run gcloud container node-pools describe "$(kubectl get node "${NODE}" -o jsonpath='{.metadata.labels.cloud\.google\.com/gke-nodepool}')" \
  --cluster=substrate-stream-test --zone=us-central1-a --project=kuiyue-gke-dev \
  --format='yaml(name,config.machineType,config.imageType,config.shieldedInstanceConfig,version)'
run kubectl get node "${NODE}" -o wide

kubectl -n ate-system delete pod "${HELPER}" --ignore-not-found --wait=true
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: ${HELPER}
  namespace: ate-system
  labels:
    app: bench-helper
spec:
  nodeName: ${NODE}
  hostPID: true
  tolerations:
  - operator: Exists
  containers:
  - name: helper
    image: debian:13-slim@sha256:d7e12182ce18b85b93007c1dedf31f2d29e01ccf3182cc4017c709b6259bc132
    command: ["/bin/sh", "-c", "sleep 86400"]
    securityContext:
      privileged: true
EOF
kubectl -n ate-system wait --for=condition=Ready "pod/${HELPER}" --timeout=120s

on_host "${HELPER}" '
  uname -r; cat /proc/version_signature; mokutil --sb-state
  grep -E "^(# )?CONFIG_(EROFS_FS|EROFS_FS_ONDEMAND|CACHEFILES|CACHEFILES_ONDEMAND|FSCACHE|NETFS_SUPPORT)[= ]" /boot/config-$(uname -r)
  ls -la /lib/modules/$(uname -r)/build
  ls /home/kubernetes/bin/modules 2>&1 || true
  lsmod | grep -E "^(erofs|cachefiles|netfs) " || echo "erofs, cachefiles, netfs not loaded"
  echo "cachefiles_ondemand symbols: $(grep -c cachefiles_ondemand /proc/kallsyms || true)"'

section "[2] Deploy ds/ate-accelerator-nydus-dragonfly onto ${NODE} and ${PEER_NODE}"
for h in "${HELPER}" "${PEER_HELPER}"; do
  kubectl -n ate-system exec "${h}" -- nsenter -t 1 -m -u -i -n -- touch /run/ate/drain-complete
done
printf '\n$ sed <image, nodeSelector> oci-image-accelerator/manifests/nydus-dragonfly.yaml | kubectl apply -f -\n' | tee -a "${LOG}"
sed -e "s|ko://go.ate.dev/substrate/oci-image-accelerator/cmd/nydus-dragonfly|${NYDUS_DF_IMG}|g" \
    -e "s|      hostNetwork: true|      hostNetwork: true\n      nodeSelector:\n        bench-accel-target: nydus-dragonfly|g" \
    oci-image-accelerator/manifests/nydus-dragonfly.yaml | kubectl apply -f - 2>&1 | tee -a "${LOG}"
# Label after the apply so the previous DaemonSet spec never runs on NODE.
run kubectl label node "${NODE}" "${PEER_NODE}" bench-accel-target=nydus-dragonfly --overwrite
run kubectl -n ate-system rollout status ds/ate-accelerator-nydus-dragonfly --timeout=15m
run kubectl -n ate-system get pods -l "${SEL}" -o wide

section "[3] build-erofs-modules on ${NODE}"
POD="$(pod_on "${NODE}")"
run kubectl -n ate-system logs "${POD}" -c build-erofs-modules --timestamps
run kubectl -n ate-system get "${POD}" -o jsonpath='{range .status.initContainerStatuses[*]}{.name}: started={.state.terminated.startedAt} finished={.state.terminated.finishedAt} exit={.state.terminated.exitCode}{"\n"}{end}'
run kubectl -n ate-system logs "${POD}" -c nydus-snapshotter --timestamps --tail=15

section "[4] Kernel state on ${NODE} after start"
on_host "${HELPER}" '
  ls -la /home/kubernetes/bin/modules/$(uname -r)
  modinfo /home/kubernetes/bin/modules/$(uname -r)/*.ko | grep -E "^(filename|vermagic|depends)"
  lsmod | grep -E "^(erofs|cachefiles|netfs) "
  echo "cachefiles_ondemand symbols: $(grep -c cachefiles_ondemand /proc/kallsyms)"
  grep -E " (erofs_fscache_register_fs|cachefiles_ondemand_init_object)" /proc/kallsyms
  grep -w erofs /proc/filesystems; ls -la /dev/cachefiles
  echo "kernel taint: $(cat /proc/sys/kernel/tainted)"'

section "[5] Prepare and release a Nydus image on ${NODE} through Dragonfly"
TMP_BIN_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_BIN_DIR}"' EXIT
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${TMP_BIN_DIR}/e2e-stream-probe" ./benchmarking/image-accelerator/e2e-stream-probe
kubectl -n ate-system exec -i "${HELPER}" -- nsenter -t 1 -m -u -i -n -- \
  sh -c 'cat > /home/kubernetes/bin/e2e-stream-probe && chmod +x /home/kubernetes/bin/e2e-stream-probe' < "${TMP_BIN_DIR}/e2e-stream-probe"
on_host "${PEER_HELPER}" "/home/kubernetes/bin/e2e-stream-probe -socket /run/ate/image-accelerator.sock -image ${SWEBENCH_NYDUS_IMG} -bypass-ttl=false"
on_host "${HELPER}" "/home/kubernetes/bin/e2e-stream-probe -socket /run/ate/image-accelerator.sock -image ${SWEBENCH_NYDUS_IMG} -bypass-ttl=true"

section "[6] Restart the pod on ${NODE}: modules are reused"
kubectl -n ate-system exec "${HELPER}" -- nsenter -t 1 -m -u -i -n -- touch /run/ate/drain-complete
run kubectl -n ate-system delete "${POD}" --wait=true
for _ in $(seq 1 120); do
  POD="$(pod_on "${NODE}")"
  [[ -n "${POD}" ]] && break
  sleep 1
done
run kubectl -n ate-system wait --for=condition=Ready "${POD}" --timeout=5m
run kubectl -n ate-system logs "${POD}" -c build-erofs-modules --timestamps
run kubectl -n ate-system get "${POD}" -o jsonpath='{range .status.initContainerStatuses[*]}{.name}: started={.state.terminated.startedAt} finished={.state.terminated.finishedAt} exit={.state.terminated.exitCode}{"\n"}{end}'

kubectl -n ate-system delete pod "${HELPER}" --wait=false
