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

ROOT="$(git rev-parse --show-toplevel)"
BENCH_DIR="${ROOT}/benchmarking/image-accelerator"
cd "${ROOT}"

if [[ -f "${ROOT}/.ate-dev-env.sh" ]]; then
  # shellcheck disable=SC1091
  source "${ROOT}/.ate-dev-env.sh"
fi

echo "=== [1] Building and publishing atelet, objectstore-plugin, ateworker-gvisor, and all 4 ImageAccelerator adapter images with ko ==="
RIPTIDE_IMG="$("${ROOT}/hack/run-tool.sh" ko build ./oci-image-accelerator/cmd/riptide)"
SOCI_IMG="$("${ROOT}/hack/run-tool.sh" ko build ./oci-image-accelerator/cmd/soci)"
NYDUS_DF_IMG="$("${ROOT}/hack/run-tool.sh" ko build ./oci-image-accelerator/cmd/nydus-dragonfly)"
SPEGEL_IMG="$("${ROOT}/hack/run-tool.sh" ko build ./oci-image-accelerator/cmd/spegel)"
ATELET_IMG="$("${ROOT}/hack/run-tool.sh" ko build ./cmd/atelet)"
OBJECTSTORE_PLUGIN_IMG="$("${ROOT}/hack/run-tool.sh" ko build ./cmd/objectstore-plugin)"
WORKER_IMG="$("${ROOT}/hack/run-tool.sh" ko build ./cmd/ateworker-gvisor)"

echo "=== [2] Deleting existing ds/atelet and freshly deploying manifests/ate-install/atelet.yaml ==="
kubectl -n ate-system delete ds atelet --ignore-not-found --wait=true
sed -e "s|ko://go.ate.dev/substrate/cmd/atelet|${ATELET_IMG}|g" \
    -e "s|ko://go.ate.dev/substrate/cmd/objectstore-plugin|${OBJECTSTORE_PLUGIN_IMG}|g" \
    "${ROOT}/manifests/ate-install/atelet.yaml" | kubectl apply -f -
kubectl -n ate-system rollout status ds/atelet --timeout=180s
kubectl get pods -n ate-system -l app=atelet -o wide

SWEBENCH_IMG="${SWEBENCH_IMG:-us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:c8c61a46b939c1b801e08930a2a7cc850b6f8790672634dd26cb8a38ba2965d4}"
SWEBENCH_TAG_IMG="${SWEBENCH_TAG_IMG:-us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:swebench-matplotlib}"
SWEBENCH_NYDUS_IMG="${SWEBENCH_NYDUS_IMG:-us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@sha256:d89906a9ec2c2edd335a39f38b415dfc86ae62d73a9cfa8a2b69a0feda53a7f7}"
SWEBENCH_NYDUS_TAG_IMG="${SWEBENCH_NYDUS_TAG_IMG:-us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:swebench-nydus-v6}"

cat > "${BENCH_DIR}/built_images.env" <<EOF
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

export ATELET_IMG="${ATELET_IMG}"
export OBJECTSTORE_PLUGIN_IMG="${OBJECTSTORE_PLUGIN_IMG}"
export WORKER_IMG="${WORKER_IMG}"
export RIPTIDE_IMG="${RIPTIDE_IMG}"
export SOCI_IMG="${SOCI_IMG}"
export NYDUS_DF_IMG="${NYDUS_DF_IMG}"
export SPEGEL_IMG="${SPEGEL_IMG}"
export SWEBENCH_IMG="${SWEBENCH_IMG}"
export SWEBENCH_TAG_IMG="${SWEBENCH_TAG_IMG}"
export SWEBENCH_NYDUS_IMG="${SWEBENCH_NYDUS_IMG}"
export SWEBENCH_NYDUS_TAG_IMG="${SWEBENCH_NYDUS_TAG_IMG}"
EOF

echo "=== [3] Running setup_ubuntu_c3_and_index.sh ==="
bash "${BENCH_DIR}/setup_ubuntu_c3_and_index.sh"

