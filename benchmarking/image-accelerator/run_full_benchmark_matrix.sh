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

set -euo pipefail

export PATH="/usr/local/google/home/kuiyue/google-cloud-sdk/bin:${PATH}"
export USE_GKE_GCLOUD_AUTH_PLUGIN=True

ROOT="$(git rev-parse --show-toplevel)"
BENCH_DIR="${ROOT}/benchmarking/image-accelerator"
RESULTS_DIR="${BENCH_DIR}/results"
mkdir -p "${RESULTS_DIR}"
cd "${ROOT}"
source "${BENCH_DIR}/built_images.env"

ATE_TOKEN=$(kubectl -n ate-system create token ate-client --audience=api.ate-system.svc --duration=2h)
ATEAPI_IP=$(kubectl -n ate-system get pods -l app=ate-api-server --field-selector=status.phase=Running -o jsonpath='{.items[0].status.podIP}')
ATENET_IP=$(kubectl -n ate-system get svc atenet-router -o jsonpath='{.spec.clusterIP}')
COS_NODE="${COS_NODE:-gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62}"
UBUNTU_NODE="${UBUNTU_NODE:-gke-substrate-stream-c3-std22-ubuntu--5a335c0d-1261}"
PEER_NODE="${PEER_NODE:-gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc}"
NO_STREAM_COS_NODE="${NO_STREAM_COS_NODE:-gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz}"

COS_HELPER="bench-helper-${COS_NODE##*-}"
UBUNTU_HELPER="bench-helper-${UBUNTU_NODE##*-}"
PEER_HELPER="bench-helper-${PEER_NODE##*-}"
NO_STREAM_HELPER="bench-helper-${NO_STREAM_COS_NODE##*-}"

RESULTS_JSONL="${RESULTS_DIR}/benchmark_results.jsonl"
EVIDENCE_LOG="${RESULTS_DIR}/benchmark_evidence.log"
START_SUITE="${1:-${START_SUITE:-all}}"
if [[ "${START_SUITE}" == "all" ]]; then
  : > "${RESULTS_JSONL}"
  : > "${EVIDENCE_LOG}"
fi

log() {
  echo "$@" | tee -a "${EVIDENCE_LOG}"
}

wait_adapter_on_node() {
  local provider="$1"
  local node="$2"
  log "Waiting for adapter pod (provider=${provider}) on ${node} to become Ready..."
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
        log "Adapter ${provider} Ready on ${node}: $(echo "${pod_lines}" | awk '{print $1, $2, $3}')"
        return 0
      fi
    fi
    sleep 1
  done
  log "ERROR: Timed out waiting for adapter ${provider} on ${node}"
  kubectl -n ate-system get pods -l app=ate-image-accelerator -o wide | tee -a "${EVIDENCE_LOG}"
  return 1
}

clean_all_adapters() {
  log "--- Cleaning all existing adapter DaemonSets, pods, sockets, and node labels ---"
  for hp in "${COS_HELPER}" "${UBUNTU_HELPER}" "${PEER_HELPER}" "${NO_STREAM_HELPER}"; do
    kubectl -n ate-system exec "${hp}" -- nsenter -t 1 -m -u -i -n -- touch /run/ate/drain-complete 2>/dev/null || true
  done
  kubectl label nodes --all ate.dev/image-accelerator- bench-accel-target- --overwrite >/dev/null 2>&1 || true
  kubectl -n ate-system delete ds ate-accelerator-riptide ate-accelerator-soci ate-accelerator-nydus-dragonfly ate-accelerator-spegel --ignore-not-found --wait=false >/dev/null 2>&1 || true
  kubectl -n ate-system delete pods -l app=ate-image-accelerator --grace-period=0 --force --ignore-not-found >/dev/null 2>&1 || true
  for _ in $(seq 1 60); do
    if [[ -z "$(kubectl -n ate-system get pods -l app=ate-image-accelerator -o name 2>/dev/null)" ]]; then
      break
    fi
    sleep 1
  done
  for hp in "${COS_HELPER}" "${UBUNTU_HELPER}" "${PEER_HELPER}" "${NO_STREAM_HELPER}"; do
    kubectl -n ate-system exec "${hp}" -- nsenter -t 1 -m -u -i -n -- sh -c '
      for pid in $(pgrep -x containerd-gcfs || true) $(pgrep -x gcfsd-v2 || true); do
        if tr "\0" " " < "/proc/${pid}/cmdline" 2>/dev/null | grep -q "/var/lib/ate/accelerator"; then
          kill -9 "${pid}" 2>/dev/null || true
        fi
      done
      pkill -x soci-snapshotter-grpc 2>/dev/null || true
      pkill -x containerd-nydus-grpc 2>/dev/null || true
      pkill -x nydusd 2>/dev/null || true
      rm -f \
        /run/ate/image-accelerator.sock \
        /run/ate/drain-complete \
        /run/ate/gcfs-snapshotter.sock \
        /run/soci-snapshotter-grpc/soci-snapshotter-grpc.sock \
        /run/soci-snapshotter-grpc/docker/config.json \
        /run/containerd-nydus/containerd-nydus-grpc.sock \
        /run/containerd-nydus/docker/config.json 2>/dev/null || true
      grep "/var/lib/ate/accelerator" /proc/mounts | awk "{print \$2}" | sort -r | xargs -r umount -lf 2>/dev/null || true
      chmod -R u+w /var/lib/ate/accelerator-image-cache 2>/dev/null || true
      rm -rf /var/lib/ate/accelerator-image-cache/* 2>/dev/null || true
    ' 2>/dev/null || true
  done
}

deploy_fresh_adapter() {
  local provider="$1"
  local manifest="$2"
  local img_placeholder="$3"
  local img_actual="$4"
  shift 4
  local target_nodes=("$@")

  clean_all_adapters

  log "--- Freshly deploying ds/ate-accelerator-${provider} from ${manifest} onto: ${target_nodes[*]} ---"
  kubectl label node "${target_nodes[@]}" "bench-accel-target=${provider}" --overwrite | tee -a "${EVIDENCE_LOG}"
  sed -e "s|${img_placeholder}|${img_actual}|g" \
      -e "s|      hostNetwork: true|      hostNetwork: true\n      nodeSelector:\n        bench-accel-target: ${provider}|g" \
      "${manifest}" | kubectl apply -f - | tee -a "${EVIDENCE_LOG}"

  if [[ "${provider}" == "nydus-dragonfly" ]]; then
    kubectl -n ate-system rollout status deployment/dragonfly-redis deployment/dragonfly-mysql deployment/dragonfly-manager deployment/dragonfly-scheduler --timeout=180s | tee -a "${EVIDENCE_LOG}"
  fi

  for n in "${target_nodes[@]}"; do
    wait_adapter_on_node "${provider}" "${n}"
  done
  kubectl -n ate-system get ds "ate-accelerator-${provider}" -o wide | tee -a "${EVIDENCE_LOG}"
  kubectl -n ate-system get pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=${provider}" -o wide | tee -a "${EVIDENCE_LOG}"
}

if [[ "${START_SUITE}" == "all" ]]; then
  log "=== [1] Deleting existing DaemonSets and freshly deploying ds/atelet ==="
  clean_all_adapters
  kubectl -n ate-system delete ds atelet --ignore-not-found --wait=true | tee -a "${EVIDENCE_LOG}"
  sed -e "s|ko://go.ate.dev/substrate/cmd/atelet|${ATELET_IMG}|g" \
      -e "s|ko://go.ate.dev/substrate/cmd/objectstore-plugin|${OBJECTSTORE_PLUGIN_IMG}|g" \
      "${ROOT}/manifests/ate-install/atelet.yaml" | kubectl apply -f - | tee -a "${EVIDENCE_LOG}"
  kubectl -n ate-system rollout status deploy/ate-api-server deploy/ate-controller deploy/atenet-egress deploy/atenet-router ds/atelet --timeout=180s | tee -a "${EVIDENCE_LOG}"
  kubectl -n ate-system get pods -l app=atelet -o wide | tee -a "${EVIDENCE_LOG}"

  log "=== [2] Deleting existing WorkerPools and freshly creating 1 CPU / 2 GiB WorkerPools on c3-standard-22 nodes ==="
  kubectl label node "${COS_NODE}" bench-node=c3-cos --overwrite | tee -a "${EVIDENCE_LOG}"
  kubectl label node "${UBUNTU_NODE}" bench-node=c3-ubuntu --overwrite | tee -a "${EVIDENCE_LOG}"

  for ad in riptide soci spegel baseline nydus-dragonfly; do
    for wl in small medium large; do
      kubectl -n ate-system exec "${COS_HELPER}" -- /home/kubernetes/bin/bench-runner \
        --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" \
        --atespace=ate-bench-stream --actor="bench-${ad}-${wl}" >/dev/null 2>&1 || true
    done
  done

  kubectl -n ate-bench-stream delete workerpool wp-c3-cos wp-c3-ubuntu --ignore-not-found --wait=true | tee -a "${EVIDENCE_LOG}"
  log "Using workerImage: ${WORKER_IMG}"

  kubectl apply -f - <<EOF | tee -a "${EVIDENCE_LOG}"
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
  workerImage: ${WORKER_IMG}
  sandboxClasses:
  - name: gvisor
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

  log "Waiting for fresh WorkerPools wp-c3-cos and wp-c3-ubuntu to have 1 Ready replica..."
  for _ in $(seq 1 90); do
    R_COS=$(kubectl -n ate-bench-stream get workerpool wp-c3-cos -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)
    R_UBU=$(kubectl -n ate-bench-stream get workerpool wp-c3-ubuntu -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)
    if [[ "${R_COS}" == "1" && "${R_UBU}" == "1" ]]; then
      log "Both fresh WorkerPools Ready (1 CPU, 2 GiB on c3-standard-22)!"
      break
    fi
    sleep 2
  done
  kubectl -n ate-bench-stream get workerpool,pods -o wide | tee -a "${EVIDENCE_LOG}"

  log "=== [3] Verifying Riptide on both COS modes (GKE Image Streaming enabled vs disabled) ==="
  deploy_fresh_adapter "riptide" \
    "oci-image-accelerator/manifests/riptide.yaml" \
    "ko://go.ate.dev/substrate/oci-image-accelerator/cmd/riptide" \
    "${RIPTIDE_IMG}" \
    "${COS_NODE}" "${NO_STREAM_COS_NODE}"

  log "--- Probing Riptide on ${COS_NODE} (host gcfsd enabled) ---"
  kubectl -n ate-system exec "${COS_HELPER}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
    -socket /run/ate/image-accelerator.sock \
    -image "${SWEBENCH_IMG}" \
    -bypass-ttl=true | tee -a "${EVIDENCE_LOG}"

  log "--- Probing Riptide on ${NO_STREAM_COS_NODE} (host gcfsd disabled) ---"
  kubectl -n ate-system exec "${NO_STREAM_HELPER}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
    -socket /run/ate/image-accelerator.sock \
    -image "${SWEBENCH_IMG}" \
    -bypass-ttl=true | tee -a "${EVIDENCE_LOG}"

  log "=== [4] Pre-initializing ActorTemplates (1 CPU / 2 GiB) and completing one-time golden snapshots ==="
  kubectl -n ate-system exec "${COS_HELPER}" -- /home/kubernetes/bin/bench-runner \
    --mode=ensure-template \
    --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" --atespace=ate-bench-stream \
    --template=tmpl-cos-oci \
    --pool-label=bench-cos \
    --image="${SWEBENCH_IMG}" \
    --wait-golden=true | tee -a "${EVIDENCE_LOG}"

  deploy_fresh_adapter "nydus-dragonfly" \
    "oci-image-accelerator/manifests/nydus-dragonfly.yaml" \
    "ko://go.ate.dev/substrate/oci-image-accelerator/cmd/nydus-dragonfly" \
    "${NYDUS_DF_IMG}" \
    "${UBUNTU_NODE}" "${PEER_NODE}"

  kubectl -n ate-system exec "${PEER_HELPER}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
    -socket /run/ate/image-accelerator.sock \
    -image "${SWEBENCH_NYDUS_IMG}" \
    -bypass-ttl=false | tee -a "${EVIDENCE_LOG}"

  kubectl -n ate-system exec "${UBUNTU_HELPER}" -- /home/kubernetes/bin/bench-runner \
    --mode=ensure-template \
    --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" --atespace=ate-bench-stream \
    --template=tmpl-ubuntu-nydus \
    --pool-label=bench-ubuntu \
    --image="${SWEBENCH_NYDUS_IMG}" \
    --wait-golden=true | tee -a "${EVIDENCE_LOG}"
fi

wipe_node_cache() {
  local adapter="$1"
  local helper_pod="$2"
  local image_ref="$3"

  kubectl -n ate-system exec "${helper_pod}" -- nsenter -t 1 -m -u -i -n -- sh -c "
    chmod -R u+w /var/lib/ate/image-cache/layers/sha256 /var/lib/ate/images/layers/sha256 2>/dev/null || true
    rm -rf /var/lib/ate/image-cache/layers/sha256/* /var/lib/ate/image-cache/manifests/sha256/* /var/lib/ate/images/layers/sha256/* /var/lib/ate/images/manifests/sha256/*
    mkdir -p /var/lib/ate/image-cache/layers/sha256 /var/lib/ate/image-cache/manifests/sha256
    if [ '${adapter}' != 'baseline' ] && [ -S /run/ate/image-accelerator.sock ]; then
      /home/kubernetes/bin/bench-runner \
        --mode=release-image \
        --socket=/run/ate/image-accelerator.sock \
        --image='${image_ref}' >/dev/null 2>&1 || true
    fi
    if [ '${adapter}' = 'spegel' ]; then
      chmod -R u+w /var/lib/ate/accelerator-image-cache/spegel-store/layers/sha256 /var/lib/ate/accelerator/spegel-store/layers/sha256 2>/dev/null || true
      rm -rf /var/lib/ate/accelerator-image-cache/spegel-store/blobs/sha256/* /var/lib/ate/accelerator-image-cache/spegel-store/layers/sha256/* /var/lib/ate/accelerator/spegel-store/blobs/sha256/* /var/lib/ate/accelerator/spegel-store/layers/sha256/* 2>/dev/null || true
      mkdir -p /var/lib/ate/accelerator-image-cache/spegel-store/blobs/sha256 /var/lib/ate/accelerator-image-cache/spegel-store/layers/sha256
    fi
    sync
    echo 3 > /proc/sys/vm/drop_caches || true
  "
}

CMD_SMALL='cat -- "/testbed/lib/matplotlib/pyplot.py"'
CMD_MEDIUM='/opt/miniconda3/envs/testbed/bin/python -m compileall -f -q "/testbed/lib/matplotlib"'
CMD_LARGE='tar -cf - --one-file-system / | wc -c'

run_one_measurement() {
  local adapter="$1"
  local scenario="$2"
  local workload="$3"
  local cmd_str="$4"
  local run_mode="$5"
  local helper_pod="$6"
  local tmpl_name="$7"
  local actor_name="$8"
  local node_name="$9"

  log "--- [${adapter} | ${scenario} | ${workload}] Running (${run_mode}) on ${node_name} ---"
  local out_json
  out_json=$(kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
    --mode="${run_mode}" \
    --adapter="${adapter}" \
    --scenario="${scenario}" \
    --workload="${workload}" \
    --cmd="${cmd_str}" \
    --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" --atespace=ate-bench-stream \
    --template="${tmpl_name}" \
    --actor="${actor_name}")

  local picked_line
  picked_line=$(kubectl -n ate-system logs -l app=ate-api-server --timestamps=true --since=180s 2>/dev/null | grep "Picked worker" | sort | tail -n 1 || true)
  local picked_ts
  picked_ts=$(echo "${picked_line}" | awk '{print $1}')
  local picked_ns=0
  if [[ -n "${picked_ts}" ]]; then
    picked_ns=$(date -d "${picked_ts}" +%s%N 2>/dev/null || echo 0)
  fi

  local enriched_json
  enriched_json=$(python3 -c '
import json, sys
d = json.loads(sys.argv[1])
picked_ns = int(sys.argv[2])
picked_ts = sys.argv[3]
if picked_ns > 0 and abs(picked_ns - d["resume_rpc_start_ns"]) < 30_000_000_000:
    d["picked_worker_ts"] = picked_ts
    d["picked_worker_ns"] = picked_ns
    d["ttce_ms"] = round((d["cmd_t0_ns"] - picked_ns) / 1e6, 2)
    d["e2e_ms"] = round((d["cmd_t1_ns"] - picked_ns) / 1e6, 2)
else:
    d["picked_worker_ts"] = "rpc_start_fallback"
    d["picked_worker_ns"] = d["resume_rpc_start_ns"]
    d["ttce_ms"] = round(d["client_ttce_ms"], 2)
    d["e2e_ms"] = round(d["client_e2e_ms"], 2)
d["workload_run_ms"] = round(d["workload_run_ms"], 2)
print(json.dumps(d))
' "${out_json}" "${picked_ns}" "${picked_ts}")

  echo "${enriched_json}" | tee -a "${RESULTS_JSONL}" | tee -a "${EVIDENCE_LOG}"
  echo "ateapi Picked worker log: ${picked_line}" >> "${EVIDENCE_LOG}"

  if [[ "${adapter}" != "baseline" ]]; then
    local ad_pod
    ad_pod=$(kubectl -n ate-system get pods -l "app=ate-image-accelerator,ate.dev/image-accelerator=${adapter}" --field-selector "spec.nodeName=${node_name}" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    local container_name="accelerator-${adapter}"
    if [[ "${adapter}" == "nydus-dragonfly" ]]; then
      container_name="accelerator-dragonfly"
    fi
    if [[ -n "${ad_pod}" ]]; then
      echo "Adapter (${ad_pod}) recent logs:" >> "${EVIDENCE_LOG}"
      kubectl -n ate-system logs "${ad_pod}" -c "${container_name}" --tail=6 >> "${EVIDENCE_LOG}" 2>&1 || \
      kubectl -n ate-system logs "${ad_pod}" --tail=6 >> "${EVIDENCE_LOG}" 2>&1 || true
    fi
  else
    local atelet_pod
    atelet_pod=$(kubectl -n ate-system get pods -l app=atelet --field-selector "spec.nodeName=${node_name}" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    if [[ -n "${atelet_pod}" ]]; then
      echo "atelet (${atelet_pod}) recent imagecache logs:" >> "${EVIDENCE_LOG}"
      kubectl -n ate-system logs "${atelet_pod}" --tail=15 | grep -E "Image cache|EnsureImage|PrepareImage" >> "${EVIDENCE_LOG}" 2>&1 || true
    fi
  fi
}

run_adapter_suite() {
  local adapter="$1"
  local node_name="$2"
  local helper_pod="bench-helper-${node_name##*-}"
  local tmpl_name="$3"
  local image_ref="$4"

  log "================================================================================"
  log "STARTING SUITE: Adapter=${adapter} Node=${node_name} Image=${image_ref}"
  log "================================================================================"

  for wl in small medium large; do
    local cmd_str=""
    case "${wl}" in
      small)  cmd_str="${CMD_SMALL}" ;;
      medium) cmd_str="${CMD_MEDIUM}" ;;
      large)  cmd_str="${CMD_LARGE}" ;;
    esac

    local actor_name="bench-${adapter}-${wl}"
    kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
      --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" --atespace=ate-bench-stream --actor="${actor_name}" >/dev/null 2>&1 || true

    wipe_node_cache "${adapter}" "${helper_pod}" "${image_ref}"
    run_one_measurement "${adapter}" "cold_boot" "${wl}" "${cmd_str}" "create-boot-run" "${helper_pod}" "${tmpl_name}" "${actor_name}" "${node_name}"

    kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
      --mode=suspend --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" --atespace=ate-bench-stream --actor="${actor_name}" --clean-pycache=true >> "${EVIDENCE_LOG}"

    run_one_measurement "${adapter}" "warm_resume" "${wl}" "${cmd_str}" "resume-and-run" "${helper_pod}" "${tmpl_name}" "${actor_name}" "${node_name}"

    kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
      --mode=suspend --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" --atespace=ate-bench-stream --actor="${actor_name}" --clean-pycache=true >> "${EVIDENCE_LOG}"

    # Evict node-local snapshot while preserving durable GCS snapshot so the next resume is a true cold_resume
    kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
      --mode=revert --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" --atespace=ate-bench-stream --actor="${actor_name}" >> "${EVIDENCE_LOG}"

    wipe_node_cache "${adapter}" "${helper_pod}" "${image_ref}"
    run_one_measurement "${adapter}" "cold_resume" "${wl}" "${cmd_str}" "resume-and-run" "${helper_pod}" "${tmpl_name}" "${actor_name}" "${node_name}"

    kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
      --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:443" --atespace=ate-bench-stream --actor="${actor_name}" >> "${EVIDENCE_LOG}"
  done
}

should_run_suite() {
  local s="$1"
  case "${START_SUITE}" in
    all|riptide) return 0 ;;
    soci) [[ "${s}" != "riptide" ]] ;;
    spegel) [[ "${s}" == "spegel" || "${s}" == "baseline" || "${s}" == "nydus-dragonfly" ]] ;;
    baseline) [[ "${s}" == "baseline" || "${s}" == "nydus-dragonfly" ]] ;;
    nydus-dragonfly) [[ "${s}" == "nydus-dragonfly" ]] ;;
    *) return 1 ;;
  esac
}

if should_run_suite "riptide"; then
  # Suite 1: Riptide on COS_NODE
  deploy_fresh_adapter "riptide" \
    "oci-image-accelerator/manifests/riptide.yaml" \
    "ko://go.ate.dev/substrate/oci-image-accelerator/cmd/riptide" \
    "${RIPTIDE_IMG}" \
    "${COS_NODE}"
  log "Pre-probing Riptide on ${COS_NODE} (-bypass-ttl=true) to verify fresh mount + eviction..."
  kubectl -n ate-system exec "${COS_HELPER}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
    -socket /run/ate/image-accelerator.sock \
    -image "${SWEBENCH_IMG}" \
    -bypass-ttl=true | tee -a "${EVIDENCE_LOG}"
  run_adapter_suite "riptide" "${COS_NODE}" "tmpl-cos-oci" "${SWEBENCH_IMG}"
fi

if should_run_suite "soci"; then
  # Suite 2: AWS SOCI on COS_NODE
  deploy_fresh_adapter "soci" \
    "oci-image-accelerator/manifests/soci.yaml" \
    "ko://go.ate.dev/substrate/oci-image-accelerator/cmd/soci" \
    "${SOCI_IMG}" \
    "${COS_NODE}"
  log "Pre-probing SOCI on ${COS_NODE} (-bypass-ttl=true) to verify fresh mount + eviction..."
  kubectl -n ate-system exec "${COS_HELPER}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
    -socket /run/ate/image-accelerator.sock \
    -image "${SWEBENCH_IMG}" \
    -bypass-ttl=true | tee -a "${EVIDENCE_LOG}"
  run_adapter_suite "soci" "${COS_NODE}" "tmpl-cos-oci" "${SWEBENCH_IMG}"
fi

if should_run_suite "spegel"; then
  # Suite 3: Spegel P2P on COS_NODE (seeded from PEER_NODE)
  deploy_fresh_adapter "spegel" \
    "oci-image-accelerator/manifests/spegel.yaml" \
    "ko://go.ate.dev/substrate/oci-image-accelerator/cmd/spegel" \
    "${SPEGEL_IMG}" \
    "${COS_NODE}" "${PEER_NODE}"
  log "Seeding ${SWEBENCH_IMG} on Spegel peer node ${PEER_NODE}..."
  kubectl -n ate-system exec "${PEER_HELPER}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
    -socket /run/ate/image-accelerator.sock \
    -image "${SWEBENCH_IMG}" \
    -bypass-ttl=false | tee -a "${EVIDENCE_LOG}"
  run_adapter_suite "spegel" "${COS_NODE}" "tmpl-cos-oci" "${SWEBENCH_IMG}"
fi

if should_run_suite "baseline"; then
  # Suite 4: Baseline (local image-cache, no adapter DaemonSet) on COS_NODE
  clean_all_adapters
  kubectl label node "${COS_NODE}" ate.dev/image-accelerator=disabled --overwrite | tee -a "${EVIDENCE_LOG}"
  run_adapter_suite "baseline" "${COS_NODE}" "tmpl-cos-oci" "${SWEBENCH_IMG}"
  kubectl label node "${COS_NODE}" ate.dev/image-accelerator- --overwrite | tee -a "${EVIDENCE_LOG}"
fi

if should_run_suite "nydus-dragonfly"; then
  # Suite 5: Nydus + Dragonfly P2P on UBUNTU_NODE (seeded from PEER_NODE)
  deploy_fresh_adapter "nydus-dragonfly" \
    "oci-image-accelerator/manifests/nydus-dragonfly.yaml" \
    "ko://go.ate.dev/substrate/oci-image-accelerator/cmd/nydus-dragonfly" \
    "${NYDUS_DF_IMG}" \
    "${UBUNTU_NODE}" "${PEER_NODE}"
  log "Seeding ${SWEBENCH_NYDUS_IMG} on Dragonfly peer node ${PEER_NODE}..."
  kubectl -n ate-system exec "${PEER_HELPER}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
    -socket /run/ate/image-accelerator.sock \
    -image "${SWEBENCH_NYDUS_IMG}" \
    -bypass-ttl=false | tee -a "${EVIDENCE_LOG}"
  log "Pre-probing Nydus+Dragonfly on ${UBUNTU_NODE} (-bypass-ttl=true) to verify fresh mount + eviction..."
  kubectl -n ate-system exec "${UBUNTU_HELPER}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
    -socket /run/ate/image-accelerator.sock \
    -image "${SWEBENCH_NYDUS_IMG}" \
    -bypass-ttl=true | tee -a "${EVIDENCE_LOG}"
  run_adapter_suite "nydus-dragonfly" "${UBUNTU_NODE}" "tmpl-ubuntu-nydus" "${SWEBENCH_NYDUS_IMG}"
fi

log "=== ALL 45 BENCHMARK RUNS COMPLETED SUCCESSFULLY! ==="
python3 - "${RESULTS_JSONL}" <<'PYEOF' | tee -a "${EVIDENCE_LOG}"
import json, sys

rows = [json.loads(line) for line in open(sys.argv[1]) if line.strip()]
for wl in ["small", "medium", "large"]:
    print(f"\n### Workload: {wl}")
    print("| Adapter | Scenario | ResumeActor RPC (ms) | Client TTCE (ms) | Workload Run (ms) | Client E2E (ms) |")
    print("| :--- | :--- | ---: | ---: | ---: | ---: |")
    for r in rows:
        if r["workload"] == wl:
            print(f"| `{r['adapter']}` | `{r['scenario']}` | {r['resume_rpc_dur_ms']:.2f} | {r['client_ttce_ms']:.2f} | {r['workload_run_ms']:.2f} | **{r['client_e2e_ms']:.2f}** |")
PYEOF

