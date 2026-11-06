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

# Reruns the Nydus bakes of setup-nydus.sh, whose first try used an upper
# case TAG that Kubernetes names reject, then runs setup-rest.sh.

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${HERE}/../../../../.." && pwd)"
cd "${ROOT}" || exit 1
# shellcheck source=/dev/null
source .ate-dev-env.sh
B=benchmarking/image-plugin
export RUN_DIR="${B}/results/20261103-confirm-sweep/setup"
for s in b c; do
  "${B}/ev.sh" "bake-nydus-pf${s}" "DENSITY=2 TAG=-pf${s} LOADER_ARGS='-images /config/images-96-pf-${s}.jsonl -actors 96 -concurrency 48' ${B}/run.sh bake nydus" || exit 1
done
"${HERE}/setup-rest.sh"
