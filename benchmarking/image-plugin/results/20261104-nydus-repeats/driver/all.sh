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

# Tests and deploys the ateapi that feeds binds to its worker cache, checks
# placement at 16 per node twice on the warm pool, then runs queue.sh.

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${HERE}/../../../../.." && pwd)"
cd "${ROOT}" || exit 1
# shellcheck source=/dev/null
source .ate-dev-env.sh
export PYTHONPYCACHEPREFIX="${PYCACHE:?set PYCACHE}"
R="${HERE}/.."
B=benchmarking/image-plugin

echo "tests $(date -u +%T)"
go test -count=1 ./cmd/ateapi/internal/controlapi/... ./cmd/ateapi/internal/store/... \
  ./cmd/ateapi/internal/workercache/ ./cmd/ateapi/internal/scheduling/ >"${R}/setup/ateapi-tests.log" 2>&1 || {
  echo "ateapi tests failed, see setup/ateapi-tests.log"
  exit 1
}

echo "deploy ateapi $(date -u +%T)"
KO="$(cd hack/tools/ko && go tool -n ko)"
img="$("${KO}" build --base-import-paths ./cmd/ateapi 2>>"${R}/setup/ko.log" | tail -1)"
echo "${img}" >"${R}/setup/ateapi-image.txt"
kubectl -n ate-system set image deploy/ate-api-server "ate-api-server=${img}"
kubectl -n ate-system rollout status deploy/ate-api-server --timeout=10m

# Placement on the warm pool: 48 actors, one turn, twice.
for i in 1 2; do
  echo "placement ${i} $(date -u +%T)"
  RUN_DIR="${R}/placement-${i}" DENSITY=2 TAG=-o \
    LOADER_ARGS="-images /config/images-96.jsonl -actors 48 -turns 1 -reads 1000" \
    "${B}/run.sh" rep "${i}" nydus >"${R}/setup/placement-${i}.log" 2>&1
  python3 "${B}/bottleneck.py" "${R}/placement-${i}/rep-${i}/nydus-o-w2" >"${R}/placement-${i}/bottleneck.md"
  grep -A5 '^| node | actors' "${R}/placement-${i}/bottleneck.md"
done

echo "queue $(date -u +%T)"
"${HERE}/queue.sh"
