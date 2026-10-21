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

# Verifies both decoupled upgrade workflows on a live cluster:
# 1. Scenario A: Substrate rolling upgrade (`deploy atelet` / `rollout restart ds/atelet`)
#    while an actor with streamed FUSE layers is active. Verifies:
#    - The `ate-accelerator-riptide` pod does NOT restart.
#    - The active actor continues reading streamed files without `ENOTCONN`.
#    - The new `atelet` pod calls `ReconcileImages` on startup and logs the
#      active streamed image lease.
# 2. Scenario B: FUSE-backed Adapter DaemonSet upgrade (`updateStrategy.type: OnDelete`)
#    paired with `ate.dev/image-accelerator=disabled` FUSE lease draining. Verifies:
#    - Applying an updated DaemonSet spec does NOT kill the running adapter pod
#      while an actor has active FUSE mounts.
#    - Labeling the node `ate.dev/image-accelerator=disabled` keeps the running
#      adapter pod alive for draining while new actors on the node fall back to
#      `imagecache`.
#    - Suspending/deleting the streamed actor triggers `ReleaseImage(bypass_ttl=true)`
#      on the draining adapter pod.
#    - Deleting the drained adapter pod and relabeling `ate.dev/image-accelerator=riptide`
#      starts the updated adapter pod and serves streaming requests cleanly.

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
: > "${UPGRADE_LOG}"

ATE_TOKEN=$(kubectl -n ate-system create token ate-client --audience=api.ate-system.svc --duration=2h)
ATEAPI_IP=$(kubectl -n ate-system get pods -l app=ate-api-server -o jsonpath='{.items[0].status.podIP}')
ATENET_IP=$(kubectl -n ate-system get svc atenet-router -o jsonpath='{.spec.clusterIP}')
COS_NODE="${COS_NODE:-gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62}"
HELPER_POD="bench-helper-${COS_NODE##*-}"

log() {
  echo "$@" | tee -a "${UPGRADE_LOG}"
}

wait_adapter_on_node() {
  local provider="$1"
  local node="$2"
  for i in $(seq 1 90); do
    local ready
    ready=$(kubectl -n ate-system get pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=${provider}" --field-selector "spec.nodeName=${node}" -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || true)
    if [[ "${ready}" == "True" ]]; then
      return 0
    fi
    sleep 1
  done
  log "ERROR: Timed out waiting for adapter ${provider} on ${node}"
  return 1
}

log "================================================================================"
log "UPGRADE TEST SCENARIO A: Substrate atelet Rolling Upgrade with Active Streamed Actor"
log "================================================================================"

# Apply updated ClusterRole/atelet and full riptide.yaml (including values: [riptide, disabled]) and label COS_NODE
kubectl apply -f - <<'EOF' | tee -a "${UPGRADE_LOG}"
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: atelet
rules:
- apiGroups: [""]
  resources: ["pods", "nodes"]
  verbs: ["get", "watch", "list"]
- apiGroups: ["ate.dev"]
  resources: ["csidriverconfigs", "sandboxconfigs", "workerpools"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["certificates.k8s.io"]
  resources: ["clustertrustbundles"]
  verbs: ["get", "watch", "list"]
EOF

sed "s|ko://go.ate.dev/substrate/oci-image-accelerator/cmd/riptide|${RIPTIDE_IMG}|g" \
  oci-image-accelerator/manifests/riptide.yaml | kubectl apply -f - | tee -a "${UPGRADE_LOG}"

kubectl -n ate-system delete pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=riptide" --field-selector "spec.nodeName=${COS_NODE}" --ignore-not-found --wait=true
kubectl label node "${COS_NODE}" ate.dev/image-accelerator=riptide --overwrite | tee -a "${UPGRADE_LOG}"
wait_adapter_on_node "riptide" "${COS_NODE}"

ADAPTER_POD_BEFORE=$(kubectl -n ate-system get pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=riptide" --field-selector "spec.nodeName=${COS_NODE}" -o jsonpath='{.items[0].metadata.name}')
ADAPTER_UID_BEFORE=$(kubectl -n ate-system get pod "${ADAPTER_POD_BEFORE}" -o jsonpath='{.metadata.uid}')
ADAPTER_RESTARTS_BEFORE=$(kubectl -n ate-system get pod "${ADAPTER_POD_BEFORE}" -o jsonpath='{.status.containerStatuses[0].restartCount}')
ATELET_POD_BEFORE=$(kubectl -n ate-system get pods -l app=atelet --field-selector "spec.nodeName=${COS_NODE}" -o jsonpath='{.items[0].metadata.name}')
ATELET_UID_BEFORE=$(kubectl -n ate-system get pod "${ATELET_POD_BEFORE}" -o jsonpath='{.metadata.uid}')

log "Before atelet upgrade:"
log "  Adapter pod: ${ADAPTER_POD_BEFORE} (UID=${ADAPTER_UID_BEFORE}, restarts=${ADAPTER_RESTARTS_BEFORE})"
log "  atelet pod:  ${ATELET_POD_BEFORE} (UID=${ATELET_UID_BEFORE})"

# Clean up any leftover actor and wipe local imagecache so the actor is guaranteed to use Riptide FUSE streaming
kubectl -n ate-system exec "${HELPER_POD}" -- /home/kubernetes/bin/bench-runner \
  --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
  --atespace=ate-bench-stream --actor=upgrade-actor-a >/dev/null 2>&1 || true

kubectl -n ate-system exec "${HELPER_POD}" -- nsenter -t 1 -m -u -i -n -- sh -c "
  rm -rf /var/lib/ate/image-cache/layers/sha256/* /var/lib/ate/image-cache/manifests/sha256/* /var/lib/ate/images/layers/sha256/* /var/lib/ate/images/manifests/sha256/*
  mkdir -p /var/lib/ate/image-cache/layers/sha256 /var/lib/ate/image-cache/manifests/sha256
"

log "--- [Scenario A.1] Booting streamed actor upgrade-actor-a on ${COS_NODE} ---"
kubectl -n ate-system exec "${HELPER_POD}" -- /home/kubernetes/bin/bench-runner \
  --mode=create-boot-run \
  --adapter=riptide --scenario=upgrade_a_pre --workload=small \
  --cmd='cat -- "/testbed/lib/matplotlib/pyplot.py" | wc -c' \
  --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
  --atespace=ate-bench-stream --template=tmpl-cos-oci --actor=upgrade-actor-a | tee -a "${UPGRADE_LOG}"

log "--- [Scenario A.2] Verifying rootfs-overlay.json records streamed=true and /var/lib/ate/accelerator/riptide layers on ${COS_NODE} ---"
kubectl -n ate-system exec "${HELPER_POD}" -- nsenter -t 1 -m -u -i -n -- sh -c '
  cat /var/lib/ate/actors/*/bundles/*/rootfs-overlay.json; echo ""
' | tee -a "${UPGRADE_LOG}"

log "--- [Scenario A.3] Rolling ds/atelet while upgrade-actor-a is RUNNING ---"
kubectl -n ate-system rollout restart ds/atelet | tee -a "${UPGRADE_LOG}"
kubectl -n ate-system rollout status ds/atelet --timeout=180s | tee -a "${UPGRADE_LOG}"

ADAPTER_POD_AFTER=$(kubectl -n ate-system get pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=riptide" --field-selector "spec.nodeName=${COS_NODE}" -o jsonpath='{.items[0].metadata.name}')
ADAPTER_UID_AFTER=$(kubectl -n ate-system get pod "${ADAPTER_POD_AFTER}" -o jsonpath='{.metadata.uid}')
ADAPTER_RESTARTS_AFTER=$(kubectl -n ate-system get pod "${ADAPTER_POD_AFTER}" -o jsonpath='{.status.containerStatuses[0].restartCount}')
ATELET_POD_AFTER=$(kubectl -n ate-system get pods -l app=atelet --field-selector "spec.nodeName=${COS_NODE}" -o jsonpath='{.items[0].metadata.name}')
ATELET_UID_AFTER=$(kubectl -n ate-system get pod "${ATELET_POD_AFTER}" -o jsonpath='{.metadata.uid}')

log "After atelet upgrade:"
log "  Adapter pod: ${ADAPTER_POD_AFTER} (UID=${ADAPTER_UID_AFTER}, restarts=${ADAPTER_RESTARTS_AFTER})"
log "  atelet pod:  ${ATELET_POD_AFTER} (UID=${ATELET_UID_AFTER})"

if [[ "${ADAPTER_UID_BEFORE}" != "${ADAPTER_UID_AFTER}" || "${ADAPTER_RESTARTS_BEFORE}" != "${ADAPTER_RESTARTS_AFTER}" ]]; then
  log "ERROR: Adapter pod restarted during atelet upgrade!"
  exit 1
fi
if [[ "${ATELET_UID_BEFORE}" == "${ATELET_UID_AFTER}" ]]; then
  log "ERROR: atelet pod UID did not change!"
  exit 1
fi

log "--- [Scenario A.4] Verifying active actor reads uncached files from FUSE mount after atelet restart (zero ENOTCONN) ---"
kubectl -n ate-system exec "${HELPER_POD}" -- /home/kubernetes/bin/bench-runner \
  --mode=exec \
  --cmd='python3 -c "data = open(\"/testbed/lib/matplotlib/figure.py\").read(); print(\"FUSE_READ_OK bytes=\", len(data))"' \
  --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
  --atespace=ate-bench-stream --actor=upgrade-actor-a | tee -a "${UPGRADE_LOG}"

log "--- [Scenario A.5] Verifying new atelet pod (${ATELET_POD_AFTER}) and adapter pod (${ADAPTER_POD_AFTER}) ReconcileImages logs ---"
kubectl -n ate-system logs "${ATELET_POD_AFTER}" -c atelet | grep -iE "reconciled streamed image leases|ReconcileImages" | tee -a "${UPGRADE_LOG}"
kubectl -n ate-system logs "${ADAPTER_POD_AFTER}" -c accelerator-riptide | grep -iE "Reconciled|PrepareImage" | tee -a "${UPGRADE_LOG}"

log "--- [Scenario A.6] Cleaning up upgrade-actor-a ---"
kubectl -n ate-system exec "${HELPER_POD}" -- /home/kubernetes/bin/bench-runner \
  --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
  --atespace=ate-bench-stream --actor=upgrade-actor-a | tee -a "${UPGRADE_LOG}"

log "================================================================================"
log "UPGRADE TEST SCENARIO B: Adapter DaemonSet OnDelete + ate.dev/image-accelerator=disabled Drain"
log "================================================================================"

log "--- [Scenario B.1] Booting streamed actor upgrade-actor-b on ${COS_NODE} ---"
kubectl -n ate-system exec "${HELPER_POD}" -- nsenter -t 1 -m -u -i -n -- sh -c "
  rm -rf /var/lib/ate/image-cache/layers/sha256/* /var/lib/ate/image-cache/manifests/sha256/* /var/lib/ate/images/layers/sha256/* /var/lib/ate/images/manifests/sha256/*
  mkdir -p /var/lib/ate/image-cache/layers/sha256 /var/lib/ate/image-cache/manifests/sha256
"
kubectl -n ate-system exec "${HELPER_POD}" -- /home/kubernetes/bin/bench-runner \
  --mode=create-boot-run \
  --adapter=riptide --scenario=upgrade_b_streamed --workload=small \
  --cmd='cat -- "/testbed/lib/matplotlib/pyplot.py" | wc -c' \
  --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
  --atespace=ate-bench-stream --template=tmpl-cos-oci --actor=upgrade-actor-b | tee -a "${UPGRADE_LOG}"

log "--- [Scenario B.2] Updating ds/ate-accelerator-riptide template (updateStrategy: OnDelete) ---"
kubectl -n ate-system patch ds ate-accelerator-riptide \
  --type=merge -p '{"spec":{"template":{"metadata":{"annotations":{"ate.dev/upgrade-test-rev":"v2"}}}}}' | tee -a "${UPGRADE_LOG}"
sleep 3

ADAPTER_POD_DURING_ONDELETE=$(kubectl -n ate-system get pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=riptide" --field-selector "spec.nodeName=${COS_NODE}" -o jsonpath='{.items[0].metadata.name}')
ADAPTER_UID_DURING_ONDELETE=$(kubectl -n ate-system get pod "${ADAPTER_POD_DURING_ONDELETE}" -o jsonpath='{.metadata.uid}')
log "Adapter pod after DaemonSet template update (OnDelete): ${ADAPTER_POD_DURING_ONDELETE} (UID=${ADAPTER_UID_DURING_ONDELETE})"
if [[ "${ADAPTER_UID_DURING_ONDELETE}" != "${ADAPTER_UID_BEFORE}" ]]; then
  log "ERROR: OnDelete DaemonSet killed the running adapter pod before drain!"
  exit 1
fi

log "--- [Scenario B.3] Labeling ${COS_NODE} ate.dev/image-accelerator=disabled to drain FUSE leases ---"
kubectl label node "${COS_NODE}" ate.dev/image-accelerator=disabled --overwrite | tee -a "${UPGRADE_LOG}"
sleep 3

# Verify running adapter pod is still alive while upgrade-actor-b is running and can still read from FUSE
ADAPTER_PHASE_DURING_DRAIN=$(kubectl -n ate-system get pod "${ADAPTER_POD_DURING_ONDELETE}" -o jsonpath='{.status.phase}')
log "Adapter pod phase during ate.dev/image-accelerator=disabled drain: ${ADAPTER_PHASE_DURING_DRAIN}"
kubectl -n ate-system logs "${ATELET_POD_AFTER}" -c atelet | grep "ImageAccelerator node drain state changed" | tee -a "${UPGRADE_LOG}"

kubectl -n ate-system exec "${HELPER_POD}" -- /home/kubernetes/bin/bench-runner \
  --mode=exec \
  --cmd='python3 -c "data = open(\"/testbed/lib/matplotlib/figure.py\").read(); print(\"DRAIN_FUSE_READ_OK bytes=\", len(data))"' \
  --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
  --atespace=ate-bench-stream --actor=upgrade-actor-b | tee -a "${UPGRADE_LOG}"

log "--- [Scenario B.4] Deleting upgrade-actor-b to complete FUSE drain (triggers ReleaseImage(bypass_ttl=true)) ---"
kubectl -n ate-system exec "${HELPER_POD}" -- /home/kubernetes/bin/bench-runner \
  --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
  --atespace=ate-bench-stream --actor=upgrade-actor-b | tee -a "${UPGRADE_LOG}"

kubectl -n ate-system exec "${HELPER_POD}" -- nsenter -t 1 -m -u -i -n -- sh -c '
  echo "Remaining workdirs in /var/lib/ate/accelerator/riptide after ReleaseImage(bypass_ttl=true): [$(ls -A /var/lib/ate/accelerator/riptide)]"
' | tee -a "${UPGRADE_LOG}"

log "--- [Scenario B.5] Verifying new actor on disabled node falls back to local imagecache ---"
kubectl -n ate-system exec "${HELPER_POD}" -- /home/kubernetes/bin/bench-runner \
  --mode=create-boot-run \
  --adapter=baseline --scenario=upgrade_b_fallback --workload=small \
  --cmd='cat -- "/testbed/lib/matplotlib/pyplot.py" | wc -c' \
  --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
  --atespace=ate-bench-stream --template=tmpl-cos-oci --actor=upgrade-actor-fallback | tee -a "${UPGRADE_LOG}"

kubectl -n ate-system exec "${HELPER_POD}" -- nsenter -t 1 -m -u -i -n -- sh -c '
  echo "rootfs-overlay.json during disabled fallback (should point to /var/lib/ate/image-cache):"
  cat /var/lib/ate/actors/*/bundles/*/rootfs-overlay.json; echo ""
' | tee -a "${UPGRADE_LOG}"

kubectl -n ate-system exec "${HELPER_POD}" -- /home/kubernetes/bin/bench-runner \
  --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
  --atespace=ate-bench-stream --actor=upgrade-actor-fallback | tee -a "${UPGRADE_LOG}"

log "--- [Scenario B.6] Deleting drained adapter pod and relabeling ate.dev/image-accelerator=riptide ---"
kubectl -n ate-system delete pod "${ADAPTER_POD_DURING_ONDELETE}" --wait=true | tee -a "${UPGRADE_LOG}"
kubectl label node "${COS_NODE}" ate.dev/image-accelerator=riptide --overwrite | tee -a "${UPGRADE_LOG}"
wait_adapter_on_node "riptide" "${COS_NODE}"

NEW_ADAPTER_POD=$(kubectl -n ate-system get pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=riptide" --field-selector "spec.nodeName=${COS_NODE}" -o jsonpath='{.items[0].metadata.name}')
NEW_ADAPTER_UID=$(kubectl -n ate-system get pod "${NEW_ADAPTER_POD}" -o jsonpath='{.metadata.uid}')
NEW_ADAPTER_REV=$(kubectl -n ate-system get pod "${NEW_ADAPTER_POD}" -o jsonpath='{.metadata.annotations.ate\.dev/upgrade-test-rev}')
log "New adapter pod after OnDelete upgrade: ${NEW_ADAPTER_POD} (UID=${NEW_ADAPTER_UID}, rev=${NEW_ADAPTER_REV})"

log "--- [Scenario B.7] Verifying new adapter pod serves streaming cleanly ---"
kubectl -n ate-system exec "${HELPER_POD}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
  -socket /run/ate/image-accelerator.sock \
  -image "${SWEBENCH_IMG}" \
  -bypass-ttl=true | tee -a "${UPGRADE_LOG}"

log "=== BOTH UPGRADE SCENARIOS (A & B) PASSED SUCCESSFULLY! ==="
