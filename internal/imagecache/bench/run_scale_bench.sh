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

set -o errexit -o nounset -o pipefail

ROOT="$(git rev-parse --show-toplevel)"
BENCH_DIR="${ROOT}/internal/imagecache/bench"
BUILD_DIR="${BENCH_DIR}/.bin"
POD="${BENCH_POD:-soci-snapshotter-zbr27}"
NAMESPACE="${BENCH_NAMESPACE:-ate-system}"
REPO="${BENCH_REPO:-us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/bench-scale}"
TOKEN="$(gcloud auth print-access-token)"

mkdir -p "${BUILD_DIR}"
trap 'rm -rf "${BUILD_DIR}"' EXIT

echo "=== Building gen-scale-images and validate-image-cache binaries ==="
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${BUILD_DIR}/gen-scale-images" "${ROOT}/internal/imagecache/bench"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${BUILD_DIR}/validate-opt" "${ROOT}/tools/validate-image-cache"

echo "=== Uploading binaries to ${POD} (${NAMESPACE}) ==="
kubectl exec -n "${NAMESPACE}" "${POD}" -- mkdir -p /var/lib/containerd/bench-scale-bin /var/lib/containerd/bench-gen
kubectl exec -i -n "${NAMESPACE}" "${POD}" -- sh -c 'cat > /var/lib/containerd/bench-scale-bin/node_sampler.sh && chmod +x /var/lib/containerd/bench-scale-bin/node_sampler.sh' < "${BENCH_DIR}/node_sampler.sh"
kubectl exec -i -n "${NAMESPACE}" "${POD}" -- sh -c 'cat > /var/lib/containerd/bench-scale-bin/validate-opt && chmod +x /var/lib/containerd/bench-scale-bin/validate-opt' < "${BUILD_DIR}/validate-opt"
kubectl exec -i -n "${NAMESPACE}" "${POD}" -- sh -c 'cat > /var/lib/containerd/bench-scale-bin/gen-scale-images && chmod +x /var/lib/containerd/bench-scale-bin/gen-scale-images' < "${BUILD_DIR}/gen-scale-images"
if [[ -n "${BASELINE_BIN:-}" && -f "${BASELINE_BIN}" ]]; then
  kubectl exec -i -n "${NAMESPACE}" "${POD}" -- sh -c 'cat > /var/lib/containerd/bench-scale-bin/validate-baseline && chmod +x /var/lib/containerd/bench-scale-bin/validate-baseline' < "${BASELINE_BIN}"
fi

run_tag_bench() {
  local tag="$1"
  echo ""
  echo "================================================================================"
  echo "=== GENERATING & PUSHING IMAGE: ${REPO}:${tag} ==="
  echo "================================================================================"
  kubectl exec -n "${NAMESPACE}" "${POD}" -- /var/lib/containerd/bench-scale-bin/gen-scale-images \
    -repo "${REPO}" \
    -work-dir /var/lib/containerd/bench-gen \
    -only "${tag}" \
    -access-token "${TOKEN}"

  kubectl exec -n "${NAMESPACE}" "${POD}" -- sh -c "
    set -e
    echo '${REPO}:${tag}' > /var/lib/containerd/bench-scale-bin/ref-${tag}.txt

    if [ -x /var/lib/containerd/bench-scale-bin/validate-baseline ]; then
      rm -rf /var/lib/containerd/bench-scale-cache
      mkdir -p /var/lib/containerd/bench-scale-cache
      sync; echo 3 > /proc/sys/vm/drop_caches 2>/dev/null || true
      /var/lib/containerd/bench-scale-bin/node_sampler.sh '[${tag}] 1. Baseline (main)' \
        /var/lib/containerd/bench-scale-bin/validate-baseline \
        -refs-file /var/lib/containerd/bench-scale-bin/ref-${tag}.txt \
        -cache-dir /var/lib/containerd/bench-scale-cache \
        -parallel 1 \
        -out /var/lib/containerd/bench-scale-bin/out-${tag}-base.csv
      cat /var/lib/containerd/bench-scale-bin/out-${tag}-base.csv
    fi

    rm -rf /var/lib/containerd/bench-scale-cache
    mkdir -p /var/lib/containerd/bench-scale-cache
    sync; echo 3 > /proc/sys/vm/drop_caches 2>/dev/null || true
    ATE_DISABLE_IGZIP=1 /var/lib/containerd/bench-scale-bin/node_sampler.sh '[${tag}] 2. 3-Part Opt (klauspost/gzip)' \
      /var/lib/containerd/bench-scale-bin/validate-opt \
      -refs-file /var/lib/containerd/bench-scale-bin/ref-${tag}.txt \
      -cache-dir /var/lib/containerd/bench-scale-cache \
      -parallel 1 \
      -out /var/lib/containerd/bench-scale-bin/out-${tag}-opt-go.csv
    cat /var/lib/containerd/bench-scale-bin/out-${tag}-opt-go.csv

    rm -rf /var/lib/containerd/bench-scale-cache
    mkdir -p /var/lib/containerd/bench-scale-cache
    sync; echo 3 > /proc/sys/vm/drop_caches 2>/dev/null || true
    /var/lib/containerd/bench-scale-bin/node_sampler.sh '[${tag}] 3. 3-Part Opt (igzip SIMD)' \
      /var/lib/containerd/bench-scale-bin/validate-opt \
      -refs-file /var/lib/containerd/bench-scale-bin/ref-${tag}.txt \
      -cache-dir /var/lib/containerd/bench-scale-cache \
      -parallel 1 \
      -out /var/lib/containerd/bench-scale-bin/out-${tag}-opt-igzip.csv
    cat /var/lib/containerd/bench-scale-bin/out-${tag}-opt-igzip.csv

    rm -rf /var/lib/containerd/bench-scale-cache /var/lib/containerd/bench-gen/*
  "
}

if [[ "$#" -eq 0 ]]; then
  set -- 16kb 1mb 1gb 10gb 50gb
fi

for t in "$@"; do
  run_tag_bench "${t}"
done
