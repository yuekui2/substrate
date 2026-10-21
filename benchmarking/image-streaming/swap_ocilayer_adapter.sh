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
# swap_ocilayer_adapter.sh — Build, deploy, or swap OCILayerService adapters on GKE nodes.
#
# Usage examples:
#   # 1. Fresh build & deploy of all adapter DaemonSets from source (deletes old DaemonSets first):
#   ./benchmarking/image-streaming/swap_ocilayer_adapter.sh --deploy-all-from-scratch
#
#   # 2. Swap a specific node (or multiple nodes) to nydus-dragonfly:
#   ./benchmarking/image-streaming/swap_ocilayer_adapter.sh --adapter=nydus-dragonfly --nodes="node1,node2"
#
#   # 3. Swap a node to Riptide / SOCI / Spegel / Baseline (disabled):
#   ./benchmarking/image-streaming/swap_ocilayer_adapter.sh --adapter=riptide --nodes="node1"
#   ./benchmarking/image-streaming/swap_ocilayer_adapter.sh --adapter=baseline --nodes="node1"

set -euo pipefail

export PATH="/usr/local/google/home/kuiyue/google-cloud-sdk/bin:${PATH}"
export USE_GKE_GCLOUD_AUTH_PLUGIN=True

ROOT="$(git rev-parse --show-toplevel)"
cd "${ROOT}"
export VERSION="${VERSION:-$(git rev-parse --short=8 HEAD)}"

if [[ -f "${ROOT}/.ate-dev-env.sh" ]]; then
  # shellcheck disable=SC1091
  source "${ROOT}/.ate-dev-env.sh"
fi

ADAPTER=""
NODES_CSV=""
DEPLOY_ALL_FROM_SCRATCH=false
REDEPLOY_ADAPTER=""
SUBSTRATE_VERSION_OVERRIDE=""

for arg in "$@"; do
  case "${arg}" in
    --adapter=*)
      ADAPTER="${arg#*=}"
      ;;
    --nodes=*)
      NODES_CSV="${arg#*=}"
      ;;
    --deploy-all-from-scratch)
      DEPLOY_ALL_FROM_SCRATCH=true
      ;;
    --redeploy-adapter=*)
      REDEPLOY_ADAPTER="${arg#*=}"
      ;;
    --substrate-version=*)
      SUBSTRATE_VERSION_OVERRIDE="${arg#*=}"
      ;;
    -h|--help)
      sed -n '15,28p' "$0"
      exit 0
      ;;
    *)
      echo "Unknown argument: ${arg}" >&2
      exit 1
      ;;
  esac
done

resolve_substrate_version() {
  if [[ -n "${SUBSTRATE_VERSION_OVERRIDE}" ]]; then
    SUBSTRATE_VERSION="${SUBSTRATE_VERSION_OVERRIDE}"
    SUBSTRATE_VERSION_SUFFIX="${SUBSTRATE_VERSION_OVERRIDE}"
    return
  fi
  if [[ "${DEPLOY_ALL_FROM_SCRATCH}" != "true" ]]; then
    local live_ver
    live_ver="$(kubectl -n ate-system get ds -l app=ate-ocilayer-adapter -o jsonpath='{.items[0].metadata.labels.ate\.dev/substrate-version}' 2>/dev/null || true)"
    if [[ -n "${live_ver}" ]]; then
      SUBSTRATE_VERSION="${live_ver}"
      SUBSTRATE_VERSION_SUFFIX="${live_ver}"
      return
    fi
  fi
  local version_flag
  version_flag="$(make -s ldflags | grep 'internal/version\.Version=' | head -n 1 || true)"
  local version_raw="${version_flag#*internal/version.Version=}"
  local derived
  derived="$(go run ./internal/versionlabel/cmd "${version_raw}")"
  SUBSTRATE_VERSION="${derived%% *}"
  SUBSTRATE_VERSION_SUFFIX="${derived##* }"
}

resolve_substrate_version

deploy_adapter_manifest() {
  local provider="$1"
  local manifest="cmd/ocilayer-${provider}/ocilayer-${provider}.yaml"
  if [[ ! -f "${manifest}" ]]; then
    echo "ERROR: manifest not found: ${manifest}" >&2
    exit 1
  fi

  echo "=== Building and applying ${manifest} (version=${SUBSTRATE_VERSION}, suffix=${SUBSTRATE_VERSION_SUFFIX}) ==="
  ./hack/run-tool.sh ko resolve -f "${manifest}" \
    | sed \
        -e "s/: \${SUBSTRATE_VERSION}\$/: \"${SUBSTRATE_VERSION}\"/" \
        -e "s/\${SUBSTRATE_VERSION}/${SUBSTRATE_VERSION}/g" \
        -e "s/\${SUBSTRATE_VERSION_SUFFIX}/${SUBSTRATE_VERSION_SUFFIX}/g" \
        -e "/- disabled/d" \
    | kubectl apply -f -
}

wait_for_adapter_on_node() {
  local provider="$1"
  local node="$2"
  echo "Waiting for adapter pod (provider=${provider}) on node ${node} to become Ready..."
  for _ in $(seq 1 180); do
    local ready
    ready="$(kubectl -n ate-system get pods \
      -l "app=ate-ocilayer-adapter,ate.dev/oci-layer-provider=${provider}" \
      --field-selector "spec.nodeName=${node}" \
      -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || true)"
    if [[ "${ready}" == "True" ]]; then
      local pod_name
      pod_name="$(kubectl -n ate-system get pods \
        -l "app=ate-ocilayer-adapter,ate.dev/oci-layer-provider=${provider}" \
        --field-selector "spec.nodeName=${node}" \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
      local sock_ok=true
      if [[ "${provider}" == "soci" ]]; then
        kubectl -n ate-system exec "${pod_name}" -c soci-snapshotter -- test -S /run/soci-snapshotter-grpc/soci-snapshotter-grpc.sock 2>/dev/null || sock_ok=false
      elif [[ "${provider}" == "nydus-dragonfly" ]]; then
        kubectl -n ate-system exec "${pod_name}" -c nydus-snapshotter -- test -S /run/containerd-nydus/containerd-nydus-grpc.sock 2>/dev/null || sock_ok=false
      fi
      if [[ "${sock_ok}" == "true" ]]; then
        echo "Adapter ${provider} (${pod_name}) is Ready on ${node}!"
        return 0
      fi
    fi
    sleep 2
  done
  echo "ERROR: Timed out waiting for adapter ${provider} on ${node}" >&2
  kubectl -n ate-system get pods -l app=ate-ocilayer-adapter -o wide >&2
  return 1
}

if [[ -n "${REDEPLOY_ADAPTER}" ]]; then
  kubectl -n ate-system delete ds -l "app=ate-ocilayer-adapter,ate.dev/oci-layer-provider=${REDEPLOY_ADAPTER}" --ignore-not-found=true
  deploy_adapter_manifest "${REDEPLOY_ADAPTER}"
fi

if [[ "${DEPLOY_ALL_FROM_SCRATCH}" == "true" ]]; then
  echo "=== Deleting any existing OCILayerService / ImageAccelerator DaemonSets and Dragonfly components ==="
  for ds in $(kubectl -n ate-system get ds -o name 2>/dev/null | grep -E 'ate-ocilayer-|ate-accelerator-' || true); do
    kubectl -n ate-system delete "${ds}" --ignore-not-found=true
  done
  kubectl -n ate-system delete deployment dragonfly-manager dragonfly-scheduler dragonfly-mysql dragonfly-redis --ignore-not-found=true
  kubectl -n ate-system delete svc dragonfly-manager dragonfly-scheduler dragonfly-mysql dragonfly-redis --ignore-not-found=true
  kubectl -n ate-system delete networkpolicy --all --ignore-not-found=true

  echo "Waiting for old adapter pods to terminate..."
  for _ in $(seq 1 60); do
    remaining="$(kubectl -n ate-system get pods -l app=ate-ocilayer-adapter -o name 2>/dev/null || true)"
    remaining_accel="$(kubectl -n ate-system get pods -l app=ate-image-accelerator -o name 2>/dev/null || true)"
    if [[ -z "${remaining}" && -z "${remaining_accel}" ]]; then
      break
    fi
    sleep 2
  done

  for p in riptide soci nydus-dragonfly spegel; do
    deploy_adapter_manifest "${p}"
  done

  echo "Waiting for Dragonfly control-plane deployments to become Available..."
  kubectl -n ate-system rollout status deployment/dragonfly-mysql --timeout=180s
  kubectl -n ate-system rollout status deployment/dragonfly-redis --timeout=180s
  kubectl -n ate-system rollout status deployment/dragonfly-manager --timeout=180s
  kubectl -n ate-system rollout status deployment/dragonfly-scheduler --timeout=180s
  echo "=== All 4 OCILayerService adapter DaemonSets deployed cleanly ==="
fi

if [[ -n "${ADAPTER}" ]]; then
  if [[ -z "${NODES_CSV}" ]]; then
    echo "ERROR: --nodes=<node1,node2> is required when --adapter=${ADAPTER} is specified" >&2
    exit 1
  fi
  IFS=',' read -r -a NODE_ARRAY <<< "${NODES_CSV}"

  for node in "${NODE_ARRAY[@]}"; do
    kubectl label node "${node}" "ate.dev/substrate-version=${SUBSTRATE_VERSION}" --overwrite
    kubectl label node "${node}" "ate.dev/image-accelerator-" 2>/dev/null || true
    if [[ "${ADAPTER}" == "baseline" || "${ADAPTER}" == "disabled" ]]; then
      echo "Swapping node ${node} to baseline (ate.dev/oci-layer-service=disabled)..."
      kubectl label node "${node}" "ate.dev/oci-layer-service=disabled" --overwrite
      while [[ -n "$(kubectl -n ate-system get pods -l app=ate-ocilayer-adapter --field-selector "spec.nodeName=${node}" -o name 2>/dev/null)" ]]; do
        sleep 1
      done
      echo "Node ${node} has no active OCILayerService adapter pods."
    else
      echo "Swapping node ${node} to ate.dev/oci-layer-service=${ADAPTER}..."
      kubectl label node "${node}" "ate.dev/oci-layer-service=${ADAPTER}" --overwrite
    fi
  done

  if [[ "${ADAPTER}" != "baseline" && "${ADAPTER}" != "disabled" ]]; then
    for node in "${NODE_ARRAY[@]}"; do
      wait_for_adapter_on_node "${ADAPTER}" "${node}"
    done
  fi
fi
