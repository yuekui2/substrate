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

# Nov 4 Nydus CPU runs. Original zstd images (no prefetch table), set A,
# 2 actors per worker, workers balanced across nodes by wait-ready.sh.
#
#   cpu.sh setup              deploy Nydus with pprof, bake TAG=-o
#   cpu.sh point <name> <n>   recreate the pool and run one cold rep of
#                             3*n actors into <name>/, profiling the plugin
#
# Env for point: NYDUS_PLUGIN_ARGS, TAG (default -o), IMAGES (default
# images-96.jsonl), WORKER_CPU_LIMIT=none to drop the workers' CPU limit.

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"
cd "${ROOT}" || exit 1
# shellcheck source=/dev/null
source .ate-dev-env.sh
B=benchmarking/image-plugin
R="${R_DIR:-${B}/results/20261104-nydus-cpu}"
NYDUSD=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/nydusd@sha256:eeb616c982a51e9d3329d99cd6c61de9d62dbbc0a8c61cec03df34644faea0c5
export KEEP_SEEDS=1 DENSITY=2 PYTHONPYCACHEPREFIX="${PYCACHE:?set PYCACHE}"
TAG="${TAG:--o}"
IMAGES="${IMAGES:-images-96.jsonl}"

deploy_plugin() {
  RUN_DIR="$1" NYDUS_PLUGIN_ARGS="--pprof-addr=127.0.0.1:6060 ${NYDUS_PLUGIN_ARGS:-}" \
    "${B}/ev.sh" deploy-nydus "${B}/deploy-nydus-best.sh ${NYDUSD}"
}

case "$1" in
  setup)
    export RUN_DIR="${R}/setup"
    mkdir -p "${RUN_DIR}"
    "${B}/ev.sh" deploy "${B}/deploy.sh nydus"
    kubectl -n imgbench rollout restart ds/node-sampler
    deploy_plugin "${RUN_DIR}"
    "${B}/ev.sh" wait-ready "${B}/wait-ready.sh nydus"
    TAG="${TAG}" LOADER_ARGS="-images /config/${IMAGES} -actors 96 -concurrency 48" \
      "${B}/ev.sh" bake "${B}/run.sh bake nydus"
    ;;
  bake)
    export RUN_DIR="${R}/setup"
    TAG="${TAG}" LOADER_ARGS="-images /config/${IMAGES} -actors 96 -concurrency 48" \
      "${B}/ev.sh" "bake${TAG}" "${B}/run.sh bake nydus"
    ;;
  point)
    name="$2" n="$3"
    export RUN_DIR="${R}/${name}"
    mkdir -p "${RUN_DIR}"
    deploy_plugin "${RUN_DIR}"
    if [[ "${WORKER_CPU_LIMIT:-}" == none ]]; then
      kubectl -n imgbench patch workerpool bench-nydus --type json \
        -p '[{"op":"remove","path":"/spec/template/resources/limits/cpu"}]' || true
    else
      kubectl -n imgbench patch workerpool bench-nydus --type merge \
        -p '{"spec":{"template":{"resources":{"limits":{"cpu":"1"}}}}}'
    fi
    "${B}/ev.sh" recreate-pools "${B}/pools.sh recreate nydus"
    "${B}/ev.sh" wait-ready "${B}/wait-ready.sh nydus"
    kubectl -n imgbench get deploy bench-nydus -o jsonpath='{.spec.template.spec.containers[0].resources}' >"${RUN_DIR}/worker-resources.json"
    "${B}/profile-plugin.sh" nydus "${RUN_DIR}/pprof" 30 >"${RUN_DIR}/profile.log" 2>&1 &
    prof=$!
    TAG="${TAG}" LOADER_ARGS="-actors $((3 * n)) -turns 5 -reads 1000 -images /config/${IMAGES}" \
      "${B}/ev.sh" run "${B}/run.sh rep 1 nydus"
    wait "${prof}"
    python3 "${B}/sweep_report.py" "${RUN_DIR}" >"${RUN_DIR}/sweep.md"
    for a in "${RUN_DIR}"/rep-1/*-w2; do
      python3 "${B}/bottleneck.py" "${a}" >"${a}/bottleneck.md"
    done
    ;;
  *) echo "usage: cpu.sh setup | point <name> <n>" >&2; exit 2 ;;
esac
