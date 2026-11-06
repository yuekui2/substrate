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

# Deploys the fastest Nydus arm: seed HTTP blobs, one fscache domain,
# snapshotter mounts, nydusd prefetch off, the seed access log on
# (seed-access-log.py), and a nydusd that fills bootstrap
# caches (nydusd-amp/bootstrap-fill.patch).
#
#   deploy-nydus-best.sh <nydusd image>
#
# Build the image with:
#   PATCHES=bootstrap-fill.patch nydusd-amp/build.sh <repo>
# Writes the applied manifest to ${RUN_DIR}/nydus-dragonfly.applied.yaml.
# Needs KO_DOCKER_REPO (from .ate-dev-env.sh), python3 with PyYAML.

set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${HERE}/../.." && pwd)"
NYDUSD_IMAGE="$1"
RUN_DIR="${RUN_DIR:?set RUN_DIR}"
mkdir -p "${RUN_DIR}"
KO="$(cd "${ROOT}/hack/tools/ko" && go tool -n ko)"

out="${RUN_DIR}/nydus-dragonfly.applied.yaml"
(cd "${ROOT}" && "${KO}" resolve --base-import-paths -f image-plugin/manifests/nydus-dragonfly-seedhttp.yaml) 2>"${RUN_DIR}/ko.log" |
  DRAGONFLY_PREFETCH=false NYDUS_PREFETCH=false python3 "${HERE}/scope_plugin.py" nydus |
  python3 "${HERE}/nydusd-amp/use-nydusd.py" "${NYDUSD_IMAGE}" |
  python3 "${HERE}/seed-access-log.py" >"${out}"
kubectl apply -f "${out}"
# nginx reads its config only at start.
kubectl -n ate-system rollout restart ds/nydus-blob-seed
kubectl -n ate-system rollout status ds/nydus-blob-seed --timeout=10m
