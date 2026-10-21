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

ROOT="/usr/local/google/home/kuiyue/src/substrate"
SCRATCH="/usr/local/google/home/kuiyue/.gemini/jetski/brain/ab4030cb-4037-4cb4-8630-c51f387dd899/scratch"
cd "${ROOT}"
source "${SCRATCH}/built_images.env"

ATE_TOKEN=$(kubectl -n ate-system create token ate-client --audience=api.ate-system.svc --duration=2h)
ATEAPI_IP=$(kubectl -n ate-system get pods -l app=ate-api-server -o jsonpath='{.items[0].status.podIP}')
ATENET_IP=$(kubectl -n ate-system get svc atenet-router -o jsonpath='{.spec.clusterIP}')
COS_NODE="gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62"
UBUNTU_NODE="gke-substrate-stream-c3-std22-ubuntu--5a335c0d-5h2p"
PEER_NODE="gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc"

RESULTS_JSONL="${SCRATCH}/benchmark_results.jsonl"
EVIDENCE_LOG="${SCRATCH}/benchmark_evidence.log"
: > "${RESULTS_JSONL}"
: > "${EVIDENCE_LOG}"

echo "=== [1] Rendering and applying all 4 OCILayerService Adapter DaemonSets (version 19129425) ==="

render_ds() {
  local src="$1"
  local img_placeholder="$2"
  local img_actual="$3"
  local out="$4"
  sed \
    -e "s|\${SUBSTRATE_VERSION_SUFFIX}|19129425|g" \
    -e "s|\${SUBSTRATE_VERSION}|19129425|g" \
    -e "s|${img_placeholder}|${img_actual}|g" \
    -e "/- disabled/d" \
    -e "s|/var/lib/containerd-nydus|/var/lib/ateom-gvisor/containerd-nydus|g" \
    "${src}" > "${out}"
  kubectl apply -f "${out}"
}

render_ds "cmd/ocilayer-riptide/ocilayer-riptide.yaml" \
  "ko://github.com/agent-substrate/substrate/cmd/ocilayer-riptide" \
  "${RIPTIDE_IMG}" \
  "${SCRATCH}/ds-riptide.yaml"

render_ds "cmd/ocilayer-soci/ocilayer-soci.yaml" \
  "ko://github.com/agent-substrate/substrate/cmd/ocilayer-soci" \
  "${SOCI_IMG}" \
  "${SCRATCH}/ds-soci.yaml"

render_ds "cmd/ocilayer-nydus-dragonfly/ocilayer-nydus-dragonfly.yaml" \
  "ko://github.com/agent-substrate/substrate/cmd/ocilayer-nydus-dragonfly" \
  "${NYDUS_DF_IMG}" \
  "${SCRATCH}/ds-nydus-dragonfly.yaml"

render_ds "cmd/ocilayer-spegel/ocilayer-spegel.yaml" \
  "ko://github.com/agent-substrate/substrate/cmd/ocilayer-spegel" \
  "${SPEGEL_IMG}" \
  "${SCRATCH}/ds-spegel.yaml"

echo "=== [2] Labeling c3-standard-22 nodes and creating 1 CPU / 2 GiB WorkerPools ==="
kubectl label node "${COS_NODE}" bench-node=c3-cos ate.dev/substrate-version=19129425 --overwrite
kubectl label node "${UBUNTU_NODE}" bench-node=c3-ubuntu ate.dev/substrate-version=19129425 --overwrite
kubectl label node "${PEER_NODE}" ate.dev/substrate-version=19129425 --overwrite

WORKER_IMAGE=$(kubectl get workerpool -A -o jsonpath='{.items[0].spec.workerImage}')
echo "Using workerImage: ${WORKER_IMAGE}"

kubectl apply -f - <<EOF
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

echo "Waiting for WorkerPools wp-c3-cos and wp-c3-ubuntu to have 1 Ready replica..."
for i in $(seq 1 90); do
  R_COS=$(kubectl -n ate-bench-stream get workerpool wp-c3-cos -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)
  R_UBU=$(kubectl -n ate-bench-stream get workerpool wp-c3-ubuntu -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)
  if [[ "${R_COS}" == "1" && "${R_UBU}" == "1" ]]; then
    echo "Both WorkerPools Ready (1 CPU, 2 GiB on c3-standard-22)!"
    break
  fi
  sleep 2
done
kubectl -n ate-bench-stream get pods -o wide

wait_adapter_on_node() {
  local provider="$1"
  local node="$2"
  echo "Waiting for adapter pod (provider=${provider}) on ${node} to become Ready..."
  for i in $(seq 1 90); do
    local ready
    ready=$(kubectl -n ate-system get pods -l "app=ate-ocilayer-adapter,ate.dev/oci-layer-provider=${provider}" --field-selector "spec.nodeName=${node}" -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || true)
    if [[ "${ready}" == "True" ]]; then
      echo "Adapter ${provider} Ready on ${node}!"
      return 0
    fi
    sleep 1
  done
  echo "ERROR: Timed out waiting for adapter ${provider} on ${node}"
  kubectl -n ate-system get pods -l app=ate-ocilayer-adapter -o wide
  return 1
}

echo "=== [3] Pre-initializing ActorTemplates (1 CPU / 2 GiB) and completing one-time golden snapshots ==="
kubectl label node "${COS_NODE}" ate.dev/oci-layer-service=riptide --overwrite
kubectl label node "${UBUNTU_NODE}" "${PEER_NODE}" ate.dev/oci-layer-service=nydus-dragonfly --overwrite
wait_adapter_on_node "riptide" "${COS_NODE}"
wait_adapter_on_node "nydus-dragonfly" "${UBUNTU_NODE}"
wait_adapter_on_node "nydus-dragonfly" "${PEER_NODE}"

kubectl -n ate-system exec "bench-helper-${PEER_NODE##*-}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
  -socket /run/ate-image-streaming/ocilayer.sock \
  -image "${SWEBENCH_NYDUS_IMG}" \
  -bypass-ttl=false

kubectl -n ate-system exec "bench-helper-${COS_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
  --mode=ensure-template \
  --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:80" --atespace=ate-bench-stream \
  --template=tmpl-cos-oci \
  --pool-label=bench-cos \
  --image="${SWEBENCH_IMG}" \
  --wait-golden=true

kubectl -n ate-system exec "bench-helper-${UBUNTU_NODE##*-}" -- /home/kubernetes/bin/bench-runner \
  --mode=ensure-template \
  --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:80" --atespace=ate-bench-stream \
  --template=tmpl-ubuntu-nydus \
  --pool-label=bench-ubuntu \
  --image="${SWEBENCH_NYDUS_IMG}" \
  --wait-golden=true

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
    mkdir -p /var/lib/ateom-gvisor/image-cache/layers/sha256 /var/lib/ateom-gvisor/image-cache/manifests/sha256
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

  echo "--- [${adapter} | ${scenario} | ${workload}] Running (${run_mode}) on ${node_name} ---" | tee -a "${EVIDENCE_LOG}"
  local out_json
  out_json=$(kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
    --mode="${run_mode}" \
    --adapter="${adapter}" \
    --scenario="${scenario}" \
    --workload="${workload}" \
    --cmd="${cmd_str}" \
    --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:80" --atespace=ate-bench-stream \
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
    ad_pod=$(kubectl -n ate-system get pods -l "app=ate-ocilayer-adapter,ate.dev/oci-layer-provider=${adapter}" --field-selector "spec.nodeName=${node_name}" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    if [[ -n "${ad_pod}" ]]; then
      echo "Adapter (${ad_pod}) recent logs:" >> "${EVIDENCE_LOG}"
      kubectl -n ate-system logs "${ad_pod}" -c "ocilayer-${adapter}" --tail=6 >> "${EVIDENCE_LOG}" 2>&1 || \
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

  echo "================================================================================" | tee -a "${EVIDENCE_LOG}"
  echo "STARTING SUITE: Adapter=${adapter} Node=${node_name} Image=${image_ref}" | tee -a "${EVIDENCE_LOG}"
  echo "================================================================================" | tee -a "${EVIDENCE_LOG}"

  for wl in small medium large; do
    local cmd_str=""
    case "${wl}" in
      small)  cmd_str="${CMD_SMALL}" ;;
      medium) cmd_str="${CMD_MEDIUM}" ;;
      large)  cmd_str="${CMD_LARGE}" ;;
    esac

    local actor_name="bench-${adapter}-${wl}"
    kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
      --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:80" --atespace=ate-bench-stream --actor="${actor_name}" >/dev/null 2>&1 || true

    wipe_node_cache "${adapter}" "${helper_pod}" "${image_ref}"
    run_one_measurement "${adapter}" "cold_boot" "${wl}" "${cmd_str}" "create-boot-run" "${helper_pod}" "${tmpl_name}" "${actor_name}" "${node_name}"

    kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
      --mode=suspend --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:80" --atespace=ate-bench-stream --actor="${actor_name}" --clean-pycache=true >> "${EVIDENCE_LOG}"

    run_one_measurement "${adapter}" "warm_resume" "${wl}" "${cmd_str}" "resume-and-run" "${helper_pod}" "${tmpl_name}" "${actor_name}" "${node_name}"

    kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
      --mode=suspend --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:80" --atespace=ate-bench-stream --actor="${actor_name}" --clean-pycache=true >> "${EVIDENCE_LOG}"

    wipe_node_cache "${adapter}" "${helper_pod}" "${image_ref}"
    run_one_measurement "${adapter}" "cold_resume" "${wl}" "${cmd_str}" "resume-and-run" "${helper_pod}" "${tmpl_name}" "${actor_name}" "${node_name}"

    kubectl -n ate-system exec "${helper_pod}" -- /home/kubernetes/bin/bench-runner \
      --mode=delete --token="${ATE_TOKEN}" --ateapi="${ATEAPI_IP}:443" --atenet="${ATENET_IP}:80" --atespace=ate-bench-stream --actor="${actor_name}" >> "${EVIDENCE_LOG}"
  done
}

kubectl label node "${COS_NODE}" ate.dev/oci-layer-service=riptide --overwrite
wait_adapter_on_node "riptide" "${COS_NODE}"
run_adapter_suite "riptide" "${COS_NODE}" "tmpl-cos-oci" "${SWEBENCH_IMG}"

kubectl label node "${COS_NODE}" ate.dev/oci-layer-service=soci --overwrite
wait_adapter_on_node "soci" "${COS_NODE}"
run_adapter_suite "soci" "${COS_NODE}" "tmpl-cos-oci" "${SWEBENCH_IMG}"

kubectl label node "${COS_NODE}" "${PEER_NODE}" ate.dev/oci-layer-service=spegel --overwrite
wait_adapter_on_node "spegel" "${PEER_NODE}"
wait_adapter_on_node "spegel" "${COS_NODE}"
echo "Seeding ${SWEBENCH_IMG} on Spegel peer node ${PEER_NODE}..."
kubectl -n ate-system exec "bench-helper-${PEER_NODE##*-}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
  -socket /run/ate-image-streaming/ocilayer.sock \
  -image "${SWEBENCH_IMG}" \
  -bypass-ttl=false
run_adapter_suite "spegel" "${COS_NODE}" "tmpl-cos-oci" "${SWEBENCH_IMG}"

kubectl label node "${COS_NODE}" ate.dev/oci-layer-service=disabled --overwrite
while [[ -n "$(kubectl -n ate-system get pods -l app=ate-ocilayer-adapter --field-selector spec.nodeName=${COS_NODE} -o name 2>/dev/null)" ]]; do
  sleep 1
done
kubectl -n ate-system exec "bench-helper-${COS_NODE##*-}" -- nsenter -t 1 -m -u -i -n -- rm -f /run/ate-image-streaming/ocilayer.sock
run_adapter_suite "baseline" "${COS_NODE}" "tmpl-cos-oci" "${SWEBENCH_IMG}"

kubectl label node "${UBUNTU_NODE}" "${PEER_NODE}" ate.dev/oci-layer-service=nydus-dragonfly --overwrite
wait_adapter_on_node "nydus-dragonfly" "${PEER_NODE}"
wait_adapter_on_node "nydus-dragonfly" "${UBUNTU_NODE}"
echo "Seeding ${SWEBENCH_NYDUS_IMG} on Dragonfly peer node ${PEER_NODE}..."
kubectl -n ate-system exec "bench-helper-${PEER_NODE##*-}" -- nsenter -t 1 -m -u -i -n -- /home/kubernetes/bin/e2e-stream-probe \
  -socket /run/ate-image-streaming/ocilayer.sock \
  -image "${SWEBENCH_NYDUS_IMG}" \
  -bypass-ttl=false
run_adapter_suite "nydus-dragonfly" "${UBUNTU_NODE}" "tmpl-ubuntu-nydus" "${SWEBENCH_NYDUS_IMG}"

echo "=== ALL 45 BENCHMARK RUNS COMPLETED SUCCESSFULLY! ==="
cat "${RESULTS_JSONL}"
