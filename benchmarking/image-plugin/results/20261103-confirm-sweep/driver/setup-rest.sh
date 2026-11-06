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

# Setup, phase 2: swaps the Nydus bench pool for the pull, riptide-ms and
# spegel pools (C3 quota), deploys those arms, preloads the Spegel seeds
# with the 288 images of sets A to C, and bakes sets A to C on all three.

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"
cd "${ROOT}" || exit 1
# shellcheck source=/dev/null
source .ate-dev-env.sh
B=benchmarking/image-plugin
export RUN_DIR="${B}/results/20261103-confirm-sweep/setup"
export KEEP_SEEDS=1 SPEGEL_SEEDS=dedicated VALKEY_EP=10.1.15.239:6379
EV="${B}/ev.sh"
ARMS="pull riptide-ms spegel"

"${EV}" delete-nydus "${B}/pools.sh delete nydus" || exit 1
"${EV}" create-rest "${B}/pools.sh create ${ARMS}" || exit 1
"${EV}" deploy-pull-riptide-ms "${B}/deploy.sh pull riptide-ms" || exit 1
"${EV}" deploy-spegel-preload "KO=\$(cd hack/tools/ko && go tool -n ko) && \${KO} resolve --base-import-paths -f image-plugin/manifests/spegel-preload.yaml | python3 ${B}/scope_plugin.py spegel | kubectl apply -f -" || exit 1
"${EV}" wait-rest "${B}/wait-ready.sh ${ARMS}" || exit 1
"${EV}" wait-spegel-seed "kubectl -n ate-system rollout status ds/image-plugin-spegel-seed --timeout=20m" || exit 1
"${EV}" spegel-seed-preload "${B}/spegel-seed-preload.sh ${B}/images-320.jsonl 288" || exit 1
for s in "" b c; do
  f=images-96-pf.jsonl
  [[ -n "${s}" ]] && f="images-96-pf-${s}.jsonl"
  "${EV}" "bake-rest-pf${s}" "DENSITY=2 TAG=-pf${s} LOADER_ARGS='-images /config/${f} -actors 96 -concurrency 48' ${B}/run.sh bake ${ARMS}" || exit 1
done
echo SETUP-REST-DONE
