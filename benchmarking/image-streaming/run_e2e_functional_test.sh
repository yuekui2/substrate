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
#
# run_e2e_functional_test.sh — Fresh end-to-end functional test of OCILayerService
# adapters (Nydus+Dragonfly, Riptide, SOCI, Spegel, Baseline) on GKE (kuiyue-gke-dev).
#
# Does not reuse any existing DaemonSets or WorkerPools: tears down existing
# deployments/daemonsets/workerpools, builds fresh images from source, deploys
# fresh control-plane and adapter DaemonSets, and collects full E2E evidence logs.

set -euo pipefail

export PATH="/usr/local/google/home/kuiyue/google-cloud-sdk/bin:${PATH}"
export USE_GKE_GCLOUD_AUTH_PLUGIN=True

ROOT="$(git rev-parse --show-toplevel)"
cd "${ROOT}"
export VERSION="${VERSION:-$(git rev-parse --short=8 HEAD)}"

# shellcheck disable=SC1091
source "${ROOT}/.ate-dev-env.sh"
# shellcheck disable=SC1091
source "${ROOT}/benchmarking/image-streaming/built_images.env"

COS_NODE="${COS_NODE:-gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62}"
UBUNTU_NODE="${UBUNTU_NODE:-gke-substrate-stream-c3-std22-ubuntu--5a335c0d-5h2p}"
PEER_NODE="${PEER_NODE:-gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc}"

EVIDENCE_LOG="${ROOT}/benchmarking/image-streaming/e2e_functional_test_evidence.log"
RESULTS_JSONL="${ROOT}/benchmarking/image-streaming/e2e_functional_test_results.jsonl"
: > "${EVIDENCE_LOG}"
: > "${RESULTS_JSONL}"

log_banner() {
  echo "" | tee -a "${EVIDENCE_LOG}"
  echo "================================================================================" | tee -a "${EVIDENCE_LOG}"
  echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] $*" | tee -a "${EVIDENCE_LOG}"
  echo "================================================================================" | tee -a "${EVIDENCE_LOG}"
}

run_and_log() {
  echo "+ $*" | tee -a "${EVIDENCE_LOG}"
  "$@" 2>&1 | tee -a "${EVIDENCE_LOG}"
}

VERSION_FLAG="$(make -s ldflags | grep 'internal/version\.Version=' | head -n 1 || true)"
VERSION_RAW="${VERSION_FLAG#*internal/version.Version=}"
DERIVED_VER="$(go run ./internal/versionlabel/cmd "${VERSION_RAW}")"
SUBSTRATE_VERSION="${DERIVED_VER%% *}"
SUBSTRATE_VERSION_SUFFIX="${DERIVED_VER##* }"

log_banner "STEP 1: Tear down existing DaemonSets, WorkerPools, and stale host state (Fresh Start)"
echo "Substrate build version: ${SUBSTRATE_VERSION} (suffix: ${SUBSTRATE_VERSION_SUFFIX})" | tee -a "${EVIDENCE_LOG}"

kubectl -n ate-bench-stream delete workerpool --all --ignore-not-found=true 2>&1 | tee -a "${EVIDENCE_LOG}" || true
kubectl delete namespace ate-bench-stream ate-bench-accel --ignore-not-found=true 2>&1 | tee -a "${EVIDENCE_LOG}" || true
for ds in $(kubectl -n ate-system get ds -o name 2>/dev/null | grep -E 'ate-ocilayer-|ate-accelerator-|atelet' || true); do
  kubectl -n ate-system delete "${ds}" --ignore-not-found=true 2>&1 | tee -a "${EVIDENCE_LOG}"
done
kubectl -n ate-system delete deployment atenet-egress atenet-router ate-controller ate-api-server dragonfly-manager dragonfly-scheduler dragonfly-mysql dragonfly-redis --ignore-not-found=true 2>&1 | tee -a "${EVIDENCE_LOG}"
kubectl -n ate-system delete svc atenet-router atenet-egress dragonfly-manager dragonfly-scheduler dragonfly-mysql dragonfly-redis --ignore-not-found=true 2>&1 | tee -a "${EVIDENCE_LOG}"
kubectl -n ate-system delete configmap ate-api-authentication --ignore-not-found=true 2>&1 | tee -a "${EVIDENCE_LOG}"
kubectl -n ate-system delete networkpolicy --all --ignore-not-found=true 2>&1 | tee -a "${EVIDENCE_LOG}"

kubectl label nodes --all "ate.dev/substrate-version=${SUBSTRATE_VERSION}" --overwrite 2>&1 | tee -a "${EVIDENCE_LOG}"
kubectl label nodes --all "ate.dev/oci-layer-service=disabled" --overwrite 2>&1 | tee -a "${EVIDENCE_LOG}"
kubectl label nodes --all "ate.dev/image-accelerator-" 2>/dev/null || true

for NODE in "${COS_NODE}" "${UBUNTU_NODE}" "${PEER_NODE}"; do
  POD="bench-helper-${NODE##*-}"
  echo "Cleaning host state on ${NODE} via ${POD}..." | tee -a "${EVIDENCE_LOG}"
  kubectl -n ate-system exec "${POD}" -- nsenter -t 1 -m -u -i -n -- sh -c '
    pkill -x nydusd 2>/dev/null || true
    pkill -x containerd-nydus-grpc 2>/dev/null || true
    pkill -x soci-snapshotter-grpc 2>/dev/null || true
    grep -E "containerd-nydus|nydus-dragonfly|soci-snapshotter-grpc" /proc/mounts 2>/dev/null | awk "{print \$2}" | sort -r | xargs -r umount -l 2>/dev/null || true
    rm -rf /var/lib/containerd-nydus /var/lib/ateom-gvisor/containerd-nydus /var/lib/ateom-gvisor/streaming/* /var/lib/soci-snapshotter-grpc/snapshotter /var/lib/soci-snapshotter-grpc/unpack /run/ate-image-streaming/* /run/containerd-nydus/* /run/soci-snapshotter-grpc/soci-snapshotter-grpc.sock
    rm -f /var/lib/ateom-gvisor/image-cache/manifests/sha256/c8c61a46* /var/lib/ateom-gvisor/image-cache/manifests/sha256/d89906a9* 2>/dev/null || true
    for d in /var/lib/ateom-gvisor/image-cache/layers/sha256/*; do
      if [ -d "$d" ] && [ ! -f "$d/fs/pause" ]; then
        chmod -R u+w "$d" 2>/dev/null || true
        rm -rf "$d"
      fi
    done
    mkdir -p /var/lib/ateom-gvisor/image-cache/layers/sha256 /var/lib/ateom-gvisor/image-cache/manifests/sha256 /run/ate-image-streaming
  ' 2>&1 | tee -a "${EVIDENCE_LOG}"
done

log_banner "STEP 2: Recompile bench-runner and install on helper pods"
mkdir -p "${ROOT}/bin"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${ROOT}/bin/bench-runner" ./benchmarking/image-streaming/bench-runner
for NODE in "${COS_NODE}" "${UBUNTU_NODE}" "${PEER_NODE}"; do
  POD="bench-helper-${NODE##*-}"
  kubectl -n ate-system cp "${ROOT}/bin/bench-runner" "${POD}:/home/kubernetes/bin/bench-runner.new"
  kubectl -n ate-system exec "${POD}" -- nsenter -t 1 -m -u -i -n -- sh -c "chmod +x /home/kubernetes/bin/bench-runner.new && mv /home/kubernetes/bin/bench-runner.new /home/kubernetes/bin/bench-runner"
done
rm -f "${ROOT}/bin/bench-runner"

log_banner "STEP 3: Fresh deploy of ate-system control plane & atelet (${SUBSTRATE_VERSION})"
./hack/install-ate.sh --deploy-ate-system --rollout-timeout=300s 2>&1 | tee -a "${EVIDENCE_LOG}"

log_banner "STEP 4: Build fresh ateom-gvisor worker image and deploy WorkerPools (wp-c3-cos, wp-c3-ubuntu)"
WORKER_IMAGE="$(./hack/run-tool.sh ko build ./cmd/ateom-gvisor --ldflags="-X=github.com/agent-substrate/substrate/internal/version.Version=${SUBSTRATE_VERSION}")"
echo "Built fresh WORKER_IMAGE: ${WORKER_IMAGE}" | tee -a "${EVIDENCE_LOG}"

kubectl label node "${COS_NODE}" bench-node=c3-cos "ate.dev/substrate-version=${SUBSTRATE_VERSION}" --overwrite
kubectl label node "${UBUNTU_NODE}" bench-node=c3-ubuntu "ate.dev/substrate-version=${SUBSTRATE_VERSION}" --overwrite
kubectl label node "${PEER_NODE}" "ate.dev/substrate-version=${SUBSTRATE_VERSION}" --overwrite

kubectl apply -f - <<EOF 2>&1 | tee -a "${EVIDENCE_LOG}"
apiVersion: v1
kind: Namespace
metadata:
  name: ate-bench-stream
---
apiVersion: ate.dev/v1alpha1
kind: WorkerPool
metadata:
  name: wp-c3-cos
  namespace: ate-bench-stream
  labels:
    workload: bench-cos
spec:
  replicas: 1
  workerImage: ${WORKER_IMAGE}
  template:
    nodeSelector:
      bench-node: c3-cos
    resources:
      requests:
        cpu: "1"
        memory: 2Gi
      limits:
        cpu: "1"
        memory: 2Gi
---
apiVersion: ate.dev/v1alpha1
kind: WorkerPool
metadata:
  name: wp-c3-ubuntu
  namespace: ate-bench-stream
  labels:
    workload: bench-ubuntu
spec:
  replicas: 1
  workerImage: ${WORKER_IMAGE}
  template:
    nodeSelector:
      bench-node: c3-ubuntu
    resources:
      requests:
        cpu: "1"
        memory: 2Gi
      limits:
        cpu: "1"
        memory: 2Gi
EOF

echo "Waiting for WorkerPools wp-c3-cos and wp-c3-ubuntu to become Ready..." | tee -a "${EVIDENCE_LOG}"
for _ in $(seq 1 90); do
  R_COS="$(kubectl -n ate-bench-stream get workerpool wp-c3-cos -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)"
  R_UBU="$(kubectl -n ate-bench-stream get workerpool wp-c3-ubuntu -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)"
  if [[ "${R_COS}" == "1" && "${R_UBU}" == "1" ]]; then
    break
  fi
  sleep 2
done
run_and_log kubectl -n ate-bench-stream get workerpools,pods -o wide

WP_UBUNTU_POD="$(kubectl -n ate-bench-stream get pods --field-selector "spec.nodeName=${UBUNTU_NODE}" -o jsonpath='{.items[0].metadata.name}')"
echo "Inspecting volumeMounts on worker pod ${WP_UBUNTU_POD}:" | tee -a "${EVIDENCE_LOG}"
kubectl -n ate-bench-stream get pod "${WP_UBUNTU_POD}" -o jsonpath='{.spec.containers[0].volumeMounts}' | python3 -m json.tool | tee -a "${EVIDENCE_LOG}"

log_banner "STEP 5: Build & deploy all 4 OCILayerService adapter DaemonSets from scratch via swap_ocilayer_adapter.sh"
./benchmarking/image-streaming/swap_ocilayer_adapter.sh --deploy-all-from-scratch 2>&1 | tee -a "${EVIDENCE_LOG}"
run_and_log kubectl -n ate-system get ds,deploy -o wide

WP_UBUNTU_POD="$(kubectl -n ate-bench-stream get pods --field-selector "spec.nodeName=${UBUNTU_NODE}" -o jsonpath='{.items[0].metadata.name}')"
ATE_TOKEN="$(kubectl -n ate-system create token ate-client --audience=api.ate-system.svc --duration=2h)"
ATEAPI_IP="$(kubectl -n ate-system get pods -l app=ate-api-server --field-selector=status.phase=Running -o jsonpath='{.items[0].status.podIP}')"
ATENET_ADDR="$(kubectl -n ate-system get pods -l app=atenet-router --field-selector=status.phase=Running -o jsonpath='{.items[0].status.podIP}'):8080"
ACTOR_NYDUS="e2e-nydus-dragonfly-actor"

wipe_node_cache() {
  local adapter="$1"
  local helper_pod="$2"
  local image_ref="$3"

  kubectl -n ate-system exec "${helper_pod}" -- nsenter -t 1 -m -u -i -n -- sh -c "
    rm -f /var/lib/ateom-gvisor/image-cache/manifests/sha256/c8c61a46* /var/lib/ateom-gvisor/image-cache/manifests/sha256/d89906a9* 2>/dev/null || true
    for d in /var/lib/ateom-gvisor/image-cache/layers/sha256/*; do
      if [ -d \"\$d\" ] && [ ! -f \"\$d/fs/pause\" ]; then
        chmod -R u+w \"\$d\" 2>/dev/null || true
        rm -rf \"\$d\"
      fi
    done
    if [ '${adapter}' != 'baseline' ] && [ -S /run/ate-image-streaming/ocilayer.sock ]; then
      /home/kubernetes/bin/bench-runner \
        --mode=release-image \
        --socket=/run/ate-image-streaming/ocilayer.sock \
        --image='${image_ref}' >/dev/null 2>&1 || true
    fi
    if [ '${adapter}' = 'spegel' ]; then
      rm -rf /var/lib/ateom-gvisor/streaming/spegel-store/blobs/sha256/* /var/lib/ateom-gvisor/streaming/spegel-store/layers/sha256/* 2>/dev/null || true
      mkdir -p /var/lib/ateom-gvisor/streaming/spegel-store/blobs/sha256 /var/lib/ateom-gvisor/streaming/spegel-store/layers/sha256
    fi
    sync
    echo 3 > /proc/sys/vm/drop_caches || true
  "
}

log_banner "STEP 6: E2E Functional Test — Nydus + Dragonfly P2P on Ubuntu Node (${UBUNTU_NODE})"
./benchmarking/image-streaming/swap_ocilayer_adapter.sh --adapter=nydus-dragonfly --nodes="${UBUNTU_NODE},${PEER_NODE}" 2>&1 | tee -a "${EVIDENCE_LOG}"

echo "--- Seeding Nydus RAFS v6 image on Dragonfly peer node ${PEER_NODE} ---" | tee -a "${EVIDENCE_LOG}"
kubectl -n ate-system exec "bench-helper-${PEER_NODE##*-}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/bench-runner \
  --mode=prepare-image \
  --socket=/run/ate-image-streaming/ocilayer.sock \
  --image="${SWEBENCH_NYDUS_IMG}" 2>&1 | tee -a "${EVIDENCE_LOG}"

echo "--- Creating ActorTemplate tmpl-ubuntu-nydus and waiting for golden snapshot ---" | tee -a "${EVIDENCE_LOG}"
kubectl -n ate-system exec "bench-helper-${UBUNTU_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
  --mode=delete-template \
  --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream \
  --template=tmpl-ubuntu-nydus 2>&1 | tee -a "${EVIDENCE_LOG}" || true

kubectl -n ate-system exec "bench-helper-${UBUNTU_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
  --mode=ensure-template \
  --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream \
  --template=tmpl-ubuntu-nydus \
  --pool-label=bench-ubuntu \
  --image="${SWEBENCH_NYDUS_IMG}" \
  --wait-golden=true 2>&1 | tee -a "${EVIDENCE_LOG}"

kubectl -n ate-system exec "bench-helper-${UBUNTU_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
  --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream --actor="${ACTOR_NYDUS}" >/dev/null 2>&1 || true

wipe_node_cache "nydus-dragonfly" "bench-helper-${UBUNTU_NODE##*-}" "${SWEBENCH_NYDUS_IMG}"

echo "--- [nydus-dragonfly] 1. Cold Boot + Command Execution ---" | tee -a "${EVIDENCE_LOG}"
OUT_NYDUS_BOOT="$(kubectl -n ate-system exec "bench-helper-${UBUNTU_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
  --mode=create-boot-run \
  --adapter=nydus-dragonfly \
  --scenario=cold_boot \
  --workload=functional_check \
  --cmd='cat -- "/testbed/lib/matplotlib/pyplot.py" | head -n 5' \
  --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream \
  --template=tmpl-ubuntu-nydus \
  --actor="${ACTOR_NYDUS}")"
echo "${OUT_NYDUS_BOOT}" | tee -a "${RESULTS_JSONL}" | tee -a "${EVIDENCE_LOG}"

echo "--- [nydus-dragonfly] 2. Verifying Nydus EROFS Mounts Inside Worker Pod (${WP_UBUNTU_POD}) ---" | tee -a "${EVIDENCE_LOG}"
kubectl -n ate-system exec "bench-helper-${UBUNTU_NODE##*-}" -- nsenter -t 1 -m -u -i -n -- sh -c '
  ATEOM_PID="$(pgrep -f ateom-gvisor | head -n 1)"
  echo "Inspecting mount namespace of worker pod process ateom-gvisor (host PID=${ATEOM_PID}):"
  echo "=== Symlink target of layer-0/fs inside worker pod (/proc/${ATEOM_PID}/root) ==="
  ls -la "/proc/${ATEOM_PID}/root/var/lib/ateom-gvisor/streaming/nydus-dragonfly/"*/layer-0/
  echo "=== Reading EROFS mount at /var/lib/ateom-gvisor/containerd-nydus/snapshots/*/mnt inside worker pod ==="
  ls -la "/proc/${ATEOM_PID}/root/var/lib/ateom-gvisor/containerd-nydus/snapshots/"*/mnt/
  echo "=== Active EROFS / containerd-nydus mounts in worker pod mountinfo ==="
  grep -E "erofs|containerd-nydus" "/proc/${ATEOM_PID}/mountinfo"
' 2>&1 | tee -a "${EVIDENCE_LOG}"

echo "--- [nydus-dragonfly] 3. Suspend + Warm Resume + Cold Resume ---" | tee -a "${EVIDENCE_LOG}"
kubectl -n ate-system exec "bench-helper-${UBUNTU_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
  --mode=suspend --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream --actor="${ACTOR_NYDUS}" 2>&1 | tee -a "${EVIDENCE_LOG}"

OUT_NYDUS_WARM="$(kubectl -n ate-system exec "bench-helper-${UBUNTU_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
  --mode=resume-and-run \
  --adapter=nydus-dragonfly \
  --scenario=warm_resume \
  --workload=functional_check \
  --cmd='cat -- "/testbed/lib/matplotlib/pyplot.py" | head -n 5' \
  --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream \
  --template=tmpl-ubuntu-nydus \
  --actor="${ACTOR_NYDUS}")"
echo "${OUT_NYDUS_WARM}" | tee -a "${RESULTS_JSONL}" | tee -a "${EVIDENCE_LOG}"

kubectl -n ate-system exec "bench-helper-${UBUNTU_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
  --mode=suspend --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream --actor="${ACTOR_NYDUS}" 2>&1 | tee -a "${EVIDENCE_LOG}"

wipe_node_cache "nydus-dragonfly" "bench-helper-${UBUNTU_NODE##*-}" "${SWEBENCH_NYDUS_IMG}"

OUT_NYDUS_COLD="$(kubectl -n ate-system exec "bench-helper-${UBUNTU_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
  --mode=resume-and-run \
  --adapter=nydus-dragonfly \
  --scenario=cold_resume \
  --workload=functional_check \
  --cmd='cat -- "/testbed/lib/matplotlib/pyplot.py" | head -n 5' \
  --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream \
  --template=tmpl-ubuntu-nydus \
  --actor="${ACTOR_NYDUS}")"
echo "${OUT_NYDUS_COLD}" | tee -a "${RESULTS_JSONL}" | tee -a "${EVIDENCE_LOG}"

kubectl -n ate-system exec "bench-helper-${UBUNTU_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
  --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream --actor="${ACTOR_NYDUS}" 2>&1 | tee -a "${EVIDENCE_LOG}"

NYDUS_POD="$(kubectl -n ate-system get pods -l app=ate-ocilayer-adapter,ate.dev/oci-layer-provider=nydus-dragonfly --field-selector "spec.nodeName=${UBUNTU_NODE}" -o jsonpath='{.items[0].metadata.name}')"
echo "--- [nydus-dragonfly] 4. Inspecting Adapter, Snapshotter/nydusd, and dfdaemon Logs (${NYDUS_POD}) ---" | tee -a "${EVIDENCE_LOG}"
echo ">>> Container: ocilayer-nydus-dragonfly <<<" | tee -a "${EVIDENCE_LOG}"
kubectl -n ate-system logs "${NYDUS_POD}" -c ocilayer-nydus-dragonfly --tail=40 2>&1 | tee -a "${EVIDENCE_LOG}"
echo ">>> Container: nydus-snapshotter (checking for any 403 token errors) <<<" | tee -a "${EVIDENCE_LOG}"
kubectl -n ate-system logs "${NYDUS_POD}" -c nydus-snapshotter --tail=50 2>&1 | tee -a "${EVIDENCE_LOG}"
if kubectl -n ate-system logs "${NYDUS_POD}" -c nydus-snapshotter | grep -q "403"; then
  echo "ERROR: Found 403 in nydus-snapshotter logs!" | tee -a "${EVIDENCE_LOG}"
  exit 1
else
  echo "VERIFIED: 0 occurrences of HTTP 403 in nydus-snapshotter / nydusd logs!" | tee -a "${EVIDENCE_LOG}"
fi

log_banner "STEP 7: E2E Functional Test — Riptide, SOCI, Spegel, and Baseline on COS Node (${COS_NODE})"
./benchmarking/image-streaming/swap_ocilayer_adapter.sh --adapter=riptide --nodes="${COS_NODE}" 2>&1 | tee -a "${EVIDENCE_LOG}"
kubectl -n ate-system exec "bench-helper-${COS_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
  --mode=ensure-template \
  --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream \
  --template=tmpl-cos-oci \
  --pool-label=bench-cos \
  --image="${SWEBENCH_IMG}" \
  --wait-golden=true 2>&1 | tee -a "${EVIDENCE_LOG}"

for ADAPTER in riptide soci spegel baseline; do
  log_banner "Testing Adapter=${ADAPTER} on COS Node (${COS_NODE})"
  if [[ "${ADAPTER}" == "spegel" ]]; then
    ./benchmarking/image-streaming/swap_ocilayer_adapter.sh --adapter=spegel --nodes="${COS_NODE},${PEER_NODE}" 2>&1 | tee -a "${EVIDENCE_LOG}"
    kubectl -n ate-system exec "bench-helper-${PEER_NODE##*-}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/bench-runner \
      --mode=prepare-image \
      --socket=/run/ate-image-streaming/ocilayer.sock \
      --image="${SWEBENCH_IMG}" 2>&1 | tee -a "${EVIDENCE_LOG}"
  else
    ./benchmarking/image-streaming/swap_ocilayer_adapter.sh --adapter="${ADAPTER}" --nodes="${COS_NODE}" 2>&1 | tee -a "${EVIDENCE_LOG}"
    if [[ "${ADAPTER}" == "baseline" ]]; then
      kubectl -n ate-system exec "bench-helper-${COS_NODE##*-}" -- nsenter -t 1 -m -u -i -n -- rm -f /run/ate-image-streaming/ocilayer.sock
    fi
  fi

  ACTOR_COS="e2e-${ADAPTER}-actor"
  kubectl -n ate-system exec "bench-helper-${COS_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
    --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream --actor="${ACTOR_COS}" >/dev/null 2>&1 || true

  wipe_node_cache "${ADAPTER}" "bench-helper-${COS_NODE##*-}" "${SWEBENCH_IMG}"

  OUT_BOOT="$(kubectl -n ate-system exec "bench-helper-${COS_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
    --mode=create-boot-run \
    --adapter="${ADAPTER}" \
    --scenario=cold_boot \
    --workload=functional_check \
    --cmd='cat -- "/testbed/lib/matplotlib/pyplot.py" | head -n 5' \
    --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream \
    --template=tmpl-cos-oci \
    --actor="${ACTOR_COS}")"
  echo "${OUT_BOOT}" | tee -a "${RESULTS_JSONL}" | tee -a "${EVIDENCE_LOG}"

  kubectl -n ate-system exec "bench-helper-${COS_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
    --mode=suspend --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream --actor="${ACTOR_COS}" 2>&1 | tee -a "${EVIDENCE_LOG}"

  wipe_node_cache "${ADAPTER}" "bench-helper-${COS_NODE##*-}" "${SWEBENCH_IMG}"

  OUT_COLD="$(kubectl -n ate-system exec "bench-helper-${COS_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
    --mode=resume-and-run \
    --adapter="${ADAPTER}" \
    --scenario=cold_resume \
    --workload=functional_check \
    --cmd='cat -- "/testbed/lib/matplotlib/pyplot.py" | head -n 5' \
    --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream \
    --template=tmpl-cos-oci \
    --actor="${ACTOR_COS}")"
  echo "${OUT_COLD}" | tee -a "${RESULTS_JSONL}" | tee -a "${EVIDENCE_LOG}"

  kubectl -n ate-system exec "bench-helper-${COS_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
    --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_ADDR}" --atespace=ate-bench-stream --actor="${ACTOR_COS}" 2>&1 | tee -a "${EVIDENCE_LOG}"

  if [[ "${ADAPTER}" != "baseline" ]]; then
    AD_POD="$(kubectl -n ate-system get pods -l "app=ate-ocilayer-adapter,ate.dev/oci-layer-provider=${ADAPTER}" --field-selector "spec.nodeName=${COS_NODE}" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
    if [[ -n "${AD_POD}" ]]; then
      echo "Adapter (${AD_POD}) logs:" | tee -a "${EVIDENCE_LOG}"
      kubectl -n ate-system logs "${AD_POD}" -c "ocilayer-${ADAPTER}" --tail=15 2>&1 | tee -a "${EVIDENCE_LOG}"
    fi
  fi
done

log_banner "ALL E2E FUNCTIONAL TESTS PASSED FROM FRESH DEPLOYMENT!"
cat "${RESULTS_JSONL}" | tee -a "${EVIDENCE_LOG}"
