#!/usr/bin/env bash

# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Installs Agent Substrate.
#
# This is a translation shim: cmd/ate-setup carries out every action, and this
# script only maps the flags and environment variables the installer has always
# accepted onto its commands, so existing command lines and CI jobs keep
# working unchanged. It deliberately holds no install logic of its own.
#
# New work belongs in cmd/ate-setup. See cmd/ate-setup/commands.md for the
# equivalent invocation of each flag below, and cmd/ate-setup/cli-diff.md for
# the behaviors that differ after the move.
#
# The environment variables the installer reads -- BUCKET_NAME, KO_DOCKER_REPO,
# PROJECT_ID, CLUSTER_NAME, CLUSTER_LOCATION, KUBECTL_CONTEXT, NO_DEV_ENV, the
# ATE_API_POSTGRES_* set, and the rest -- are read by ate-setup directly, as is
# .ate-dev-env.sh. They need no translation and are not repeated here.

set -o errexit -o nounset -o pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "${ROOT}"

SETUP_IMAGE_ACCELERATOR="${SETUP_IMAGE_ACCELERATOR:-none}"

setup_image_accelerator() {
  local adapter="${SETUP_IMAGE_ACCELERATOR:-none}"
  case "${adapter}" in
    ""|none|false)
      return
      ;;
    disabled)
      echo "=== [setup_image_accelerator (disabled)] ==="
      kubectl label nodes --all "ate.dev/image-accelerator=disabled" --overwrite
      return
      ;;
  esac

  echo "=== [setup_image_accelerator (${adapter})] ==="

  if [[ -z "${KO_DOCKER_REPO:-}" && -z "${NO_DEV_ENV:-}" && -f "${ROOT}/.ate-dev-env.sh" ]]; then
    # shellcheck disable=SC1091
    source "${ROOT}/.ate-dev-env.sh"
  fi

  # Resolve the accelerator manifest in priority order:
  # 1. Explicit manifest file or URL override (ATE_IMAGE_ACCELERATOR_MANIFEST)
  # 2. Local accelerator repository checkout (ATE_IMAGE_ACCELERATOR_PATH/manifests/${adapter}.yaml)
  # 3. In-tree manifest (oci-image-accelerator/manifests/${adapter}.yaml)
  local manifest="${ATE_IMAGE_ACCELERATOR_MANIFEST:-}"
  if [[ -z "${manifest}" ]]; then
    if [[ -n "${ATE_IMAGE_ACCELERATOR_PATH:-}" ]]; then
      local base_path="${ATE_IMAGE_ACCELERATOR_PATH}"
      manifest="${base_path}/manifests/${adapter}.yaml"
    else
      manifest="${ROOT}/oci-image-accelerator/manifests/${adapter}.yaml"
    fi
  fi

  if [[ "${manifest}" != http://* && "${manifest}" != https://* && ! -f "${manifest}" ]]; then
    echo "Error: manifest not found for image accelerator \"${adapter}\": ${manifest}" >&2
    exit 1
  fi

  kubectl apply -f manifests/ate-install/ate-system-namespace.yaml \
    && kubectl wait --for=jsonpath='{.status.phase}'=Active namespace/ate-system --timeout=60s

  local rendered=""
  if [[ "${manifest}" == http://* || "${manifest}" == https://* ]]; then
    rendered="$(curl -fsSL "${manifest}")"
  elif grep -q "ko://" "${manifest}"; then
    rendered="$("${ROOT}/hack/run-tool.sh" ko resolve -f "${manifest}")"
  else
    rendered="$(cat "${manifest}")"
  fi
  echo "${rendered}" | kubectl apply -f -

  kubectl label nodes --all "ate.dev/image-accelerator=${adapter}" --overwrite
  local deploy=""
  for deploy in $(kubectl -n ate-system get deployment -l "ate.dev/image-accelerator=${adapter}" -o name 2>/dev/null); do
    kubectl rollout status "${deploy}" -n ate-system --timeout=120s || true
  done
  local ds_name="ate-accelerator-${adapter}"
  kubectl rollout status "daemonset/${ds_name}" -n ate-system --timeout=120s || true
}

forward_args=()
run_accelerator=false
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --setup-image-accelerator=*)
      SETUP_IMAGE_ACCELERATOR="${1#*=}"
      run_accelerator=true
      ;;
    --setup-image-accelerator)
      if [[ "$#" -gt 1 && "$2" != -* ]]; then
        shift
        SETUP_IMAGE_ACCELERATOR="$1"
      else
        SETUP_IMAGE_ACCELERATOR="riptide"
      fi
      run_accelerator=true
      ;;
    *)
      forward_args+=("$1")
      ;;
  esac
  shift
done

if [[ "${#forward_args[@]}" -gt 0 || ( "${run_accelerator}" == "false" && "${SETUP_IMAGE_ACCELERATOR}" == "none" ) ]]; then
  go run ./cmd/ate-setup ${forward_args[@]+"${forward_args[@]}"}
fi

if [[ "${run_accelerator}" == "true" || "${SETUP_IMAGE_ACCELERATOR}" != "none" ]]; then
  setup_image_accelerator
fi
