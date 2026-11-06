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

# Setup, phase 1, on the existing bench-nydus pool: deploys the best Nydus
# arm and the new sampler, preloads sets B and C on the Nydus seeds, and
# bakes the Nydus templates of sets B and C. Set A reuses imgbench-nydus-pf-w2.

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../.." && pwd)"
cd "${ROOT}" || exit 1
# shellcheck source=/dev/null
source .ate-dev-env.sh
B=benchmarking/image-plugin
export RUN_DIR="${B}/results/20261103-confirm-sweep/setup"
export KEEP_SEEDS=1 SPEGEL_SEEDS=dedicated
NYDUSD=us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/nydusd@sha256:eeb616c982a51e9d3329d99cd6c61de9d62dbbc0a8c61cec03df34644faea0c5
EV="${B}/ev.sh"

"${EV}" deploy-nydus "${B}/deploy.sh nydus" || exit 1
"${EV}" deploy-nydus-best "${B}/deploy-nydus-best.sh ${NYDUSD}" || exit 1
"${EV}" wait-nydus "${B}/wait-ready.sh nydus" || exit 1
for s in b c; do
  "${EV}" "seed-preload-${s}" "SAVE_DIR=/var/lib/ate/nydus-seed-blobs ${B}/seed-preload.sh ${B}/images-96-pf-${s}.jsonl 96" || exit 1
done
for s in b c; do
  f="images-96-pf-${s}.jsonl"
  "${EV}" "bake-nydus-pf${s}" "DENSITY=2 TAG=-pf${s} LOADER_ARGS='-images /config/${f} -actors 96 -concurrency 48' ${B}/run.sh bake nydus" || exit 1
done
echo SETUP-NYDUS-DONE
