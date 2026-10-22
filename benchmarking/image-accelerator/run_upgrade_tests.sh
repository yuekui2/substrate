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

# Live E2E verification of decoupled RollingUpdate workflows from a fresh state
# across all 4 ImageAccelerator adapters (riptide, soci, nydus-dragonfly, spegel):
#
# 1. Scenario A: Substrate rolling upgrade (`rollout restart ds/atelet`) while
#    an actor with streamed FUSE layers is active.
# 2. Scenario B.1: Riptide (`ds/ate-accelerator-riptide`) RollingUpdate with
#    SIGTERM lame-duck self-draining and `/run/ate/drain-complete` preStop sync.
# 3. Scenario B.2: AWS SOCI (`ds/ate-accelerator-soci`) RollingUpdate with
#    SIGTERM lame-duck self-draining and `/run/ate/drain-complete` preStop sync.
# 4. Scenario B.3: Nydus + Dragonfly (`ds/ate-accelerator-nydus-dragonfly`)
#    RollingUpdate with SIGTERM lame-duck self-draining and preStop sync.
# 5. Scenario B.4: Spegel P2P (`ds/ate-accelerator-spegel`) stateless
#    RollingUpdate while an actor using unpacked P2P layers is active.

set -euo pipefail

export PATH="/usr/local/google/home/kuiyue/google-cloud-sdk/bin:${PATH}"
export USE_GKE_GCLOUD_AUTH_PLUGIN=True

ROOT="$(git rev-parse --show-toplevel)"
BENCH_DIR="${ROOT}/benchmarking/image-accelerator"
RESULTS_DIR="${BENCH_DIR}/results"
mkdir -p "${RESULTS_DIR}"
cd "${ROOT}"
source "${BENCH_DIR}/built_images.env"

UPGRADE_LOG="${RESULTS_DIR}/upgrade_verification.log"
START_FROM="${START_FROM:-all}"
if [[ "${START_FROM}" == "all" ]]; then
  : > "${UPGRADE_LOG}"
fi

COS_NODE="${COS_NODE:-gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62}"
UBUNTU_NODE="${UBUNTU_NODE:-gke-substrate-stream-c3-std22-ubuntu--5a335c0d-5h2p}"
PEER_NODE="${PEER_NODE:-gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc}"
COS_HELPER="bench-helper-${COS_NODE##*-}"
UBUNTU_HELPER="bench-helper-${UBUNTU_NODE##*-}"
PEER_HELPER="bench-helper-${PEER_NODE##*-}"

log() {
  echo "$@" | tee -a "${UPGRADE_LOG}"
}

run_cmd() {
  log "$ $*"
  "$@" 2>&1 | tee -a "${UPGRADE_LOG}"
}

wait_adapter_on_node() {
  local provider="$1"
  local node="$2"
  for _ in $(seq 1 120); do
    local pod_lines
    pod_lines=$(kubectl -n ate-system get pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=${provider}" --field-selector "spec.nodeName=${node}" --no-headers 2>/dev/null | grep -v "Terminating" || true)
    local count
    count=$(echo "${pod_lines}" | sed '/^$/d' | wc -l)
    if [[ "${count}" -eq 1 ]]; then
      local ready_col
      ready_col=$(echo "${pod_lines}" | awk '{print $2}')
      local ready_cur="${ready_col%/*}"
      local ready_tot="${ready_col#*/}"
      if [[ -n "${ready_cur}" && "${ready_cur}" == "${ready_tot}" ]]; then
        return 0
      fi
    fi
    sleep 1
  done
  log "ERROR: Timed out waiting for adapter ${provider} on ${node}"
  kubectl -n ate-system get pods -l app=ate-image-accelerator -o wide | tee -a "${UPGRADE_LOG}"
  return 1
}

clean_all_adapters() {
  log "--- Cleaning all existing adapter DaemonSets and node labels ---"
  for hp in "${COS_HELPER}" "${UBUNTU_HELPER}" "${PEER_HELPER}"; do
    kubectl -n ate-system exec "${hp}" -- nsenter -t 1 -m -u -i -n -- touch /run/ate/drain-complete 2>/dev/null || true
  done
  run_cmd kubectl label nodes --all ate.dev/image-accelerator- --overwrite
  run_cmd kubectl -n ate-system delete ds ate-accelerator-riptide ate-accelerator-soci ate-accelerator-nydus-dragonfly ate-accelerator-spegel --ignore-not-found --wait=true
  for _ in $(seq 1 60); do
    if [[ -z "$(kubectl -n ate-system get pods -l app=ate-image-accelerator -o name 2>/dev/null)" ]]; then
      break
    fi
    sleep 1
  done
  for hp in "${COS_HELPER}" "${UBUNTU_HELPER}" "${PEER_HELPER}"; do
    kubectl -n ate-system exec "${hp}" -- nsenter -t 1 -m -u -i -n -- rm -f \
      /run/ate/image-accelerator.sock \
      /run/ate/drain-complete \
      /run/ate/gcfs-snapshotter.sock \
      /run/soci-snapshotter-grpc/soci-snapshotter-grpc.sock \
      /run/containerd-nydus/containerd-nydus-grpc.sock 2>/dev/null || true
  done
}

wipe_local_imagecache() {
  local helper_pod="$1"
  kubectl -n ate-system exec "${helper_pod}" -- nsenter -t 1 -m -u -i -n -- sh -c "
    rm -rf /var/lib/ate/image-cache/layers/sha256/* /var/lib/ate/image-cache/manifests/sha256/* /var/lib/ate/images/layers/sha256/* /var/lib/ate/images/manifests/sha256/*
    mkdir -p /var/lib/ate/image-cache/layers/sha256 /var/lib/ate/image-cache/manifests/sha256
  "
}

ATE_TOKEN=$(kubectl -n ate-system create token ate-client --audience=api.ate-system.svc --duration=2h)
ATEAPI_IP=$(kubectl -n ate-system get pods -l app=ate-api-server --field-selector=status.phase=Running -o jsonpath='{.items[0].status.podIP}')
ATENET_IP=$(kubectl -n ate-system get svc atenet-router -o jsonpath='{.spec.clusterIP}')

if [[ "${START_FROM}" == "all" ]]; then
  log "================================================================================"
  log "PRE-FLIGHT: Verifying ate-system Control Plane, WorkerPools, and ActorTemplates"
  log "================================================================================"

  run_cmd kubectl -n ate-system rollout status deploy/ate-api-server deploy/ate-controller deploy/atenet-egress deploy/atenet-router ds/atelet --timeout=180s

  run_cmd kubectl label node "${COS_NODE}" bench-node=c3-cos --overwrite
  run_cmd kubectl label node "${UBUNTU_NODE}" bench-node=c3-ubuntu --overwrite

  for act in up-riptide-stream up-riptide-fallback up-soci-stream up-soci-fallback up-nydus-dragonfly-stream up-nydus-dragonfly-fallback up-spegel-actor; do
    kubectl -n ate-system exec "${COS_HELPER}" -- /home/kubernetes/bin/bench-runner \
      --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
      --atespace=ate-bench-stream --actor="${act}" >/dev/null 2>&1 || true
  done

  clean_all_adapters

  kubectl -n ate-bench-stream delete workerpool wp-c3-cos wp-c3-ubuntu --ignore-not-found --wait=true >/dev/null 2>&1 || true
  kubectl apply -f - <<EOF | tee -a "${UPGRADE_LOG}"
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
  workerImage: ${WORKER_IMG}
  sandboxClasses:
  - name: gvisor
  template:
    nodeSelector:
      bench-node: c3-cos
    resources:
      requests:
        cpu: "4"
        memory: 8Gi
      limits:
        cpu: "4"
        memory: 8Gi
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
  workerImage: ${WORKER_IMG}
  sandboxClasses:
  - name: gvisor
  template:
    nodeSelector:
      bench-node: c3-ubuntu
    resources:
      requests:
        cpu: "4"
        memory: 8Gi
      limits:
        cpu: "4"
        memory: 8Gi
EOF

  log "Waiting for WorkerPools wp-c3-cos and wp-c3-ubuntu to become Ready..."
  for _ in $(seq 1 90); do
    R_COS=$(kubectl -n ate-bench-stream get workerpool wp-c3-cos -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)
    R_UBU=$(kubectl -n ate-bench-stream get workerpool wp-c3-ubuntu -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)
    if [[ "${R_COS}" == "1" && "${R_UBU}" == "1" ]]; then
      break
    fi
    sleep 2
  done
  run_cmd kubectl -n ate-bench-stream get workerpool,pods -o wide

  run_cmd kubectl -n ate-system exec "${COS_HELPER}" -- /home/kubernetes/bin/bench-runner \
    --mode=ensure-template \
    --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" --atespace=ate-bench-stream \
    --template=tmpl-cos-oci \
    --pool-label=bench-cos \
    --image="${SWEBENCH_IMG}" \
    --wait-golden=true

  run_cmd kubectl -n ate-system exec "${UBUNTU_HELPER}" -- /home/kubernetes/bin/bench-runner \
    --mode=ensure-template \
    --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" --atespace=ate-bench-stream \
    --template=tmpl-ubuntu-oci \
    --pool-label=bench-ubuntu \
    --image="${SWEBENCH_IMG}" \
    --wait-golden=true

  if ! kubectl -n ate-system exec "${UBUNTU_HELPER}" -- /home/kubernetes/bin/bench-runner \
    --mode=ensure-template \
    --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" --atespace=ate-bench-stream \
    --template=tmpl-ubuntu-nydus \
    --pool-label=bench-ubuntu \
    --image="${SWEBENCH_NYDUS_IMG}" \
    --wait-golden=true >> "${UPGRADE_LOG}" 2>&1; then
    sed "s|ko://go.ate.dev/substrate/oci-image-accelerator/cmd/nydus-dragonfly|${NYDUS_DF_IMG}|g" \
      oci-image-accelerator/manifests/nydus-dragonfly.yaml | kubectl apply -f - >/dev/null
    kubectl -n ate-system rollout status deployment/dragonfly-redis deployment/dragonfly-mysql deployment/dragonfly-manager deployment/dragonfly-scheduler --timeout=180s >/dev/null
    kubectl label node "${UBUNTU_NODE}" "${PEER_NODE}" ate.dev/image-accelerator=nydus-dragonfly --overwrite >/dev/null
    wait_adapter_on_node "nydus-dragonfly" "${PEER_NODE}"
    wait_adapter_on_node "nydus-dragonfly" "${UBUNTU_NODE}"
    run_cmd kubectl -n ate-system exec "${UBUNTU_HELPER}" -- /home/kubernetes/bin/bench-runner \
      --mode=ensure-template \
      --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" --atespace=ate-bench-stream \
      --template=tmpl-ubuntu-nydus \
      --pool-label=bench-ubuntu \
      --image="${SWEBENCH_NYDUS_IMG}" \
      --wait-golden=true
  fi
fi

run_fuse_lameduck_rolling_update_test() {
  local provider="$1"
  local manifest="$2"
  local img_placeholder="$3"
  local img_actual="$4"
  local node="$5"
  local helper_pod="$6"
  local tmpl="$7"
  local fallback_tmpl="$8"
  local test_image="$9"
  local ds_name="ate-accelerator-${provider}"
  local container_name="accelerator-${provider}"
  if [[ "${provider}" == "nydus-dragonfly" ]]; then
    container_name="accelerator-dragonfly"
  fi

  log "================================================================================"
  log "ROLLING UPDATE TEST (${provider}): Fresh Deploy -> Streamed Actor -> SIGTERM Lame-Duck Drain -> Rollover"
  log "================================================================================"

  clean_all_adapters

  log "--- [${provider} Step 1] Freshly applying ${manifest} and labeling ${node} ---"
  log "$ sed 's|${img_placeholder}|${img_actual}|g' ${manifest} | kubectl apply -f -"
  sed "s|${img_placeholder}|${img_actual}|g" "${manifest}" | kubectl apply -f - 2>&1 | tee -a "${UPGRADE_LOG}"

  if [[ "${provider}" == "nydus-dragonfly" ]]; then
    run_cmd kubectl -n ate-system rollout status deployment/dragonfly-redis deployment/dragonfly-mysql deployment/dragonfly-manager deployment/dragonfly-scheduler --timeout=180s
    run_cmd kubectl label node "${PEER_NODE}" "ate.dev/image-accelerator=${provider}" --overwrite
    wait_adapter_on_node "${provider}" "${PEER_NODE}"
    log "Seeding ${test_image} on Dragonfly peer node ${PEER_NODE}..."
    run_cmd kubectl -n ate-system exec "${PEER_HELPER}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
      -socket /run/ate/image-accelerator.sock \
      -image "${test_image}" \
      -bypass-ttl=false
  fi

  run_cmd kubectl label node "${node}" "ate.dev/image-accelerator=${provider}" --overwrite
  wait_adapter_on_node "${provider}" "${node}"

  run_cmd kubectl -n ate-system get ds "${ds_name}" -o wide
  run_cmd kubectl -n ate-system get pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=${provider}" -o wide

  local pod_before
  pod_before=$(kubectl -n ate-system get pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=${provider}" --field-selector "spec.nodeName=${node}" -o jsonpath='{.items[0].metadata.name}')
  local uid_before
  uid_before=$(kubectl -n ate-system get pod "${pod_before}" -o jsonpath='{.metadata.uid}')
  log "Initial ${provider} pod on ${node}: ${pod_before} (UID=${uid_before})"

  local actor_stream="up-${provider}-stream"
  local actor_fallback="up-${provider}-fallback"
  kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
    --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
    --atespace=ate-bench-stream --actor="${actor_stream}" >/dev/null 2>&1 || true
  kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
    --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
    --atespace=ate-bench-stream --actor="${actor_fallback}" >/dev/null 2>&1 || true

  wipe_local_imagecache "${helper_pod}"

  log "--- [${provider} Step 2] Booting streamed actor ${actor_stream} on ${node} ---"
  run_cmd kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
    --mode=create-boot-run \
    --adapter="${provider}" --scenario="upgrade_${provider}_streamed" --workload=small \
    --cmd='cat -- "/testbed/lib/matplotlib/pyplot.py" | wc -c' \
    --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
    --atespace=ate-bench-stream --template="${tmpl}" --actor="${actor_stream}"

  log "--- [${provider} Step 2b] Verifying rootfs-overlay.json for ${actor_stream} on ${node} ---"
  run_cmd kubectl -n ate-system exec "${helper_pod}" -- nsenter -t 1 -m -u -i -n -- sh -c '
    cat /var/lib/ate/actors/*/bundles/*/rootfs-overlay.json; echo ""
  '

  if [[ "${provider}" == "riptide" ]]; then
    log "================================================================================"
    log "SCENARIO A: Rolling ds/atelet on ${node} while ${actor_stream} is RUNNING"
    log "================================================================================"
    local atelet_before
    atelet_before=$(kubectl -n ate-system get pods -l app=atelet --field-selector "spec.nodeName=${node}" -o jsonpath='{.items[0].metadata.name}')
    local atelet_uid_before
    atelet_uid_before=$(kubectl -n ate-system get pod "${atelet_before}" -o jsonpath='{.metadata.uid}')
    log "atelet pod before rollout: ${atelet_before} (UID=${atelet_uid_before})"

    run_cmd kubectl -n ate-system rollout restart ds/atelet
    run_cmd kubectl -n ate-system rollout status ds/atelet --timeout=180s

    local atelet_after
    atelet_after=$(kubectl -n ate-system get pods -l app=atelet --field-selector "spec.nodeName=${node}" -o jsonpath='{.items[0].metadata.name}')
    local atelet_uid_after
    atelet_uid_after=$(kubectl -n ate-system get pod "${atelet_after}" -o jsonpath='{.metadata.uid}')
    local uid_after_atelet
    uid_after_atelet=$(kubectl -n ate-system get pod "${pod_before}" -o jsonpath='{.metadata.uid}')
    log "After ds/atelet rollout: atelet=${atelet_after} (UID=${atelet_uid_after}), adapter=${pod_before} (UID=${uid_after_atelet})"

    run_cmd kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
      --mode=exec \
      --cmd='python3 -c "data = open(\"/testbed/lib/matplotlib/figure.py\").read(); print(\"ATELET_ROLLOUT_FUSE_READ_OK bytes=\", len(data))"' \
      --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
      --atespace=ate-bench-stream --actor="${actor_stream}"

    log "--- Verifying ReconcileImages logs on ${atelet_after} and ${pod_before} ---"
    kubectl -n ate-system logs "${atelet_after}" -c atelet | grep -iE "reconciled streamed image leases|ReconcileImages" | tee -a "${UPGRADE_LOG}" || true
    kubectl -n ate-system logs "${pod_before}" -c "${container_name}" | grep -iE "Reconciled|PrepareImage" | tee -a "${UPGRADE_LOG}" || true
  fi

  log "--- [${provider} Step 3] Triggering RollingUpdate on ds/${ds_name} while ${actor_stream} holds active lease (refCount=1) ---"
  if [[ "${provider}" == "nydus-dragonfly" ]]; then
    # Unlabel PEER_NODE first so maxUnavailable=10% immediately terminates the pod on UBUNTU_NODE
    run_cmd kubectl label node "${PEER_NODE}" ate.dev/image-accelerator- --overwrite
    for _ in $(seq 1 60); do
      if [[ -z "$(kubectl -n ate-system get pods -l app=ate-image-accelerator --field-selector spec.nodeName=${PEER_NODE} -o name 2>/dev/null)" ]]; then
        break
      fi
      sleep 1
    done
  fi

  run_cmd kubectl -n ate-system patch ds "${ds_name}" \
    --type=merge -p '{"spec":{"template":{"metadata":{"annotations":{"ate.dev/upgrade-test-rev":"v2"}}}}}'

  log "Waiting for ${pod_before} to receive SIGTERM and enter lame-duck drain mode..."
  for _ in $(seq 1 30); do
    if kubectl -n ate-system logs "${pod_before}" -c "${container_name}" 2>/dev/null | grep -q "Entering lame-duck drain mode"; then
      break
    fi
    sleep 1
  done

  log "--- [${provider} Step 4a] Pod status and adapter lame-duck logs during active lease drain ---"
  run_cmd kubectl -n ate-system get pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=${provider}" -o wide
  run_cmd kubectl -n ate-system logs "${pod_before}" -c "${container_name}"

  log "--- [${provider} Step 4b] Verifying /run/ate/drain-complete is NOT written while refCount=1 ---"
  run_cmd kubectl -n ate-system exec "${helper_pod}" -- nsenter -t 1 -m -u -i -n -- sh -c '
    if [ -f /run/ate/drain-complete ]; then
      echo "ERROR: /run/ate/drain-complete exists prematurely!"
      exit 1
    else
      echo "CONFIRMED: /run/ate/drain-complete does not exist while refCount=1 (preStop is holding provider container alive)"
    fi
  '

  log "--- [${provider} Step 4c] Probing /run/ate/image-accelerator.sock during lame-duck drain (expect CanProvide=false) ---"
  run_cmd kubectl -n ate-system exec "${helper_pod}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
    -socket /run/ate/image-accelerator.sock \
    -image "${test_image}" \
    -bypass-ttl=true

  log "--- [${provider} Step 4d] Verifying active actor ${actor_stream} reads uncached file from FUSE/EROFS during SIGTERM drain (zero ENOTCONN) ---"
  run_cmd kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
    --mode=exec \
    --cmd='python3 -c "data = open(\"/testbed/lib/matplotlib/axes/_axes.py\").read(); print(\"LAME_DUCK_DRAIN_READ_OK bytes=\", len(data))"' \
    --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
    --atespace=ate-bench-stream --actor="${actor_stream}"

  log "--- [${provider} Step 4e] Booting new actor ${actor_fallback} during lame-duck drain (verifying automatic fallback to local image-cache) ---"
  run_cmd kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
    --mode=create-boot-run \
    --adapter=baseline --scenario="upgrade_${provider}_fallback" --workload=small \
    --cmd='cat -- "/testbed/lib/matplotlib/pyplot.py" | wc -c' \
    --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
    --atespace=ate-bench-stream --template="${fallback_tmpl}" --actor="${actor_fallback}"

  log "--- [${provider} Step 4f] Inspecting rootfs-overlay.json of both actors (${actor_stream} streamed=true vs ${actor_fallback} image-cache) ---"
  run_cmd kubectl -n ate-system exec "${helper_pod}" -- nsenter -t 1 -m -u -i -n -- sh -c '
    for f in /var/lib/ate/actors/*/bundles/*/rootfs-overlay.json; do
      echo "=== ${f} ==="
      cat "${f}"; echo ""
    done
  '

  run_cmd kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
    --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
    --atespace=ate-bench-stream --actor="${actor_fallback}"

  log "--- [${provider} Step 5] Deleting streamed actor ${actor_stream} to drop refCount -> 0 and complete RollingUpdate ---"
  run_cmd kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
    --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
    --atespace=ate-bench-stream --actor="${actor_stream}"

  log "--- [${provider} Step 5b] Waiting for ds/${ds_name} RollingUpdate replacement pod to become Ready ---"
  run_cmd kubectl -n ate-system rollout status "ds/${ds_name}" --timeout=180s
  wait_adapter_on_node "${provider}" "${node}"

  local pod_after
  pod_after=$(kubectl -n ate-system get pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=${provider}" --field-selector "spec.nodeName=${node}" -o jsonpath='{.items[0].metadata.name}')
  local uid_after
  uid_after=$(kubectl -n ate-system get pod "${pod_after}" -o jsonpath='{.metadata.uid}')
  local rev_after
  rev_after=$(kubectl -n ate-system get pod "${pod_after}" -o jsonpath='{.metadata.annotations.ate\.dev/upgrade-test-rev}')
  log "Replacement ${provider} pod Ready: ${pod_after} (UID=${uid_after}, rev=${rev_after})"

  if [[ "${uid_before}" == "${uid_after}" || "${rev_after}" != "v2" ]]; then
    log "ERROR: Replacement pod did not roll to v2!"
    exit 1
  fi

  log "--- [${provider} Step 5c] Probing new v2 ${provider} pod (${pod_after}) to verify clean streaming ---"
  if [[ "${provider}" == "nydus-dragonfly" ]]; then
    run_cmd kubectl label node "${PEER_NODE}" "ate.dev/image-accelerator=${provider}" --overwrite
    wait_adapter_on_node "${provider}" "${PEER_NODE}"
  fi
  run_cmd kubectl -n ate-system exec "${helper_pod}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
    -socket /run/ate/image-accelerator.sock \
    -image "${test_image}" \
    -bypass-ttl=true

  run_cmd kubectl -n ate-system logs "${pod_after}" -c "${container_name}"
}

run_spegel_rolling_update_test() {
  log "================================================================================"
  log "ROLLING UPDATE TEST (spegel): Fresh Deploy -> P2P Unpacked Actor -> RollingUpdate -> Verify"
  log "================================================================================"

  clean_all_adapters

  log "--- [spegel Step 1] Freshly applying oci-image-accelerator/manifests/spegel.yaml and labeling ${PEER_NODE} & ${COS_NODE} ---"
  log "$ sed 's|ko://go.ate.dev/substrate/oci-image-accelerator/cmd/spegel|${SPEGEL_IMG}|g' oci-image-accelerator/manifests/spegel.yaml | kubectl apply -f -"
  sed "s|ko://go.ate.dev/substrate/oci-image-accelerator/cmd/spegel|${SPEGEL_IMG}|g" \
    oci-image-accelerator/manifests/spegel.yaml | kubectl apply -f - 2>&1 | tee -a "${UPGRADE_LOG}"

  run_cmd kubectl label node "${PEER_NODE}" "${COS_NODE}" ate.dev/image-accelerator=spegel --overwrite
  wait_adapter_on_node "spegel" "${PEER_NODE}"
  wait_adapter_on_node "spegel" "${COS_NODE}"

  run_cmd kubectl -n ate-system get ds ate-accelerator-spegel -o wide
  run_cmd kubectl -n ate-system get pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=spegel" -o wide

  log "Seeding ${SWEBENCH_IMG} on Spegel peer node ${PEER_NODE}..."
  run_cmd kubectl -n ate-system exec "${PEER_HELPER}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
    -socket /run/ate/image-accelerator.sock \
    -image "${SWEBENCH_IMG}" \
    -bypass-ttl=false

  local pod_before
  pod_before=$(kubectl -n ate-system get pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=spegel" --field-selector "spec.nodeName=${COS_NODE}" -o jsonpath='{.items[0].metadata.name}')
  local uid_before
  uid_before=$(kubectl -n ate-system get pod "${pod_before}" -o jsonpath='{.metadata.uid}')
  log "Initial spegel pod on ${COS_NODE}: ${pod_before} (UID=${uid_before})"

  local actor_spegel="up-spegel-actor"
  kubectl -n ate-system exec "${COS_HELPER}" -- /home/kubernetes/bin/bench-runner \
    --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
    --atespace=ate-bench-stream --actor="${actor_spegel}" >/dev/null 2>&1 || true

  wipe_local_imagecache "${COS_HELPER}"
  kubectl -n ate-system exec "${COS_HELPER}" -- nsenter -t 1 -m -u -i -n -- sh -c "
    rm -rf /var/lib/ate/accelerator/spegel-store/blobs/sha256/* /var/lib/ate/accelerator/spegel-store/layers/sha256/* 2>/dev/null || true
    mkdir -p /var/lib/ate/accelerator/spegel-store/blobs/sha256 /var/lib/ate/accelerator/spegel-store/layers/sha256
  "

  log "--- [spegel Step 2] Booting actor ${actor_spegel} on ${COS_NODE} via Spegel P2P ---"
  run_cmd kubectl -n ate-system exec "${COS_HELPER}" -- /home/kubernetes/bin/bench-runner \
    --mode=create-boot-run \
    --adapter=spegel --scenario=upgrade_spegel_p2p --workload=small \
    --cmd='cat -- "/testbed/lib/matplotlib/pyplot.py" | wc -c' \
    --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
    --atespace=ate-bench-stream --template=tmpl-cos-oci --actor="${actor_spegel}"

  run_cmd kubectl -n ate-system exec "${COS_HELPER}" -- nsenter -t 1 -m -u -i -n -- sh -c '
    cat /var/lib/ate/actors/*/bundles/*/rootfs-overlay.json; echo ""
  '

  log "--- [spegel Step 3] Triggering RollingUpdate on ds/ate-accelerator-spegel while ${actor_spegel} is RUNNING ---"
  run_cmd kubectl -n ate-system patch ds ate-accelerator-spegel \
    --type=merge -p '{"spec":{"template":{"metadata":{"annotations":{"ate.dev/upgrade-test-rev":"v2"}}}}}'
  run_cmd kubectl -n ate-system rollout status ds/ate-accelerator-spegel --timeout=180s
  wait_adapter_on_node "spegel" "${COS_NODE}"

  local pod_after
  pod_after=$(kubectl -n ate-system get pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=spegel" --field-selector "spec.nodeName=${COS_NODE}" -o jsonpath='{.items[0].metadata.name}')
  local uid_after
  uid_after=$(kubectl -n ate-system get pod "${pod_after}" -o jsonpath='{.metadata.uid}')
  local rev_after
  rev_after=$(kubectl -n ate-system get pod "${pod_after}" -o jsonpath='{.metadata.annotations.ate\.dev/upgrade-test-rev}')
  log "Replacement spegel pod on ${COS_NODE}: ${pod_after} (UID=${uid_after}, rev=${rev_after})"

  log "--- [spegel Step 4] Verifying active actor ${actor_spegel} continues reading unpacked layers after Spegel rollout ---"
  run_cmd kubectl -n ate-system exec "${COS_HELPER}" -- /home/kubernetes/bin/bench-runner \
    --mode=exec \
    --cmd='python3 -c "data = open(\"/testbed/lib/matplotlib/figure.py\").read(); print(\"SPEGEL_ROLLOUT_READ_OK bytes=\", len(data))"' \
    --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
    --atespace=ate-bench-stream --actor="${actor_spegel}"

  run_cmd kubectl -n ate-system exec "${COS_HELPER}" -- /home/kubernetes/bin/bench-runner \
    --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
    --atespace=ate-bench-stream --actor="${actor_spegel}"

  log "--- [spegel Step 5] Probing new v2 spegel pod (${pod_after}) ---"
  run_cmd kubectl -n ate-system exec "${COS_HELPER}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
    -socket /run/ate/image-accelerator.sock \
    -image "${SWEBENCH_IMG}" \
    -bypass-ttl=true

  run_cmd kubectl -n ate-system logs "${pod_after}" -c accelerator-spegel
}

if [[ "${START_FROM}" == "all" ]]; then
  # 1. Riptide (includes Scenario A atelet rolling upgrade + Scenario B.1 Riptide RollingUpdate lame-duck drain)
  run_fuse_lameduck_rolling_update_test \
    "riptide" \
    "oci-image-accelerator/manifests/riptide.yaml" \
    "ko://go.ate.dev/substrate/oci-image-accelerator/cmd/riptide" \
    "${RIPTIDE_IMG}" \
    "${COS_NODE}" \
    "${COS_HELPER}" \
    "tmpl-cos-oci" \
    "tmpl-cos-oci" \
    "${SWEBENCH_IMG}"

  # 2. AWS SOCI (Scenario B.2 SOCI RollingUpdate lame-duck drain)
  run_fuse_lameduck_rolling_update_test \
    "soci" \
    "oci-image-accelerator/manifests/soci.yaml" \
    "ko://go.ate.dev/substrate/oci-image-accelerator/cmd/soci" \
    "${SOCI_IMG}" \
    "${COS_NODE}" \
    "${COS_HELPER}" \
    "tmpl-cos-oci" \
    "tmpl-cos-oci" \
    "${SWEBENCH_IMG}"
fi

# 3. Nydus + Dragonfly (Scenario B.3 Nydus+Dragonfly RollingUpdate lame-duck drain)
run_fuse_lameduck_rolling_update_test \
  "nydus-dragonfly" \
  "oci-image-accelerator/manifests/nydus-dragonfly.yaml" \
  "ko://go.ate.dev/substrate/oci-image-accelerator/cmd/nydus-dragonfly" \
  "${NYDUS_DF_IMG}" \
  "${UBUNTU_NODE}" \
  "${UBUNTU_HELPER}" \
  "tmpl-ubuntu-nydus" \
  "tmpl-ubuntu-oci" \
  "${SWEBENCH_NYDUS_IMG}"

# 4. Spegel P2P (Scenario B.4 Stateless RollingUpdate)
run_spegel_rolling_update_test

log "================================================================================"
log "ALL 4 ADAPTER ROLLING UPDATE TESTS + ATELET UPGRADE TEST PASSED SUCCESSFULLY!"
log "================================================================================"
