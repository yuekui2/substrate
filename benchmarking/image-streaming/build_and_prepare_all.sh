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

ROOT="/usr/local/google/home/kuiyue/src/substrate"
SCRATCH="/usr/local/google/home/kuiyue/.gemini/jetski/brain/ab4030cb-4037-4cb4-8630-c51f387dd899/scratch"

cd "${ROOT}"
source "${SCRATCH}/built_images.env"

COS_NODE="gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62"
UBUNTU_NODE="gke-substrate-stream-c3-std22-ubuntu--5a335c0d-5h2p"

echo "=== [4] Checking atelet pods on ${COS_NODE} and ${UBUNTU_NODE} ==="
kubectl get pods -n ate-system -l app=atelet -o wide

echo "=== [5] Appending SWE-bench matplotlib layer to benchmark-9layer ==="
TAR_PATH="${SCRATCH}/swebench_matplotlib_layer.tar.gz"
ls -lh "${TAR_PATH}"

BASE_IMG="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:benchmark-9layer"
TARGET_IMG="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:swebench-matplotlib"

/usr/local/google/home/kuiyue/go/bin/crane append -b "${BASE_IMG}" -f "${TAR_PATH}" -t "${TARGET_IMG}"
TARGET_DIGEST=$(/usr/local/google/home/kuiyue/go/bin/crane digest "${TARGET_IMG}")
echo "Built and pushed ${TARGET_IMG}@${TARGET_DIGEST}"
grep -v "SWEBENCH_" "${SCRATCH}/built_images.env" > "${SCRATCH}/built_images.env.tmp" || true
mv "${SCRATCH}/built_images.env.tmp" "${SCRATCH}/built_images.env"
echo "export SWEBENCH_IMG=\"us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@${TARGET_DIGEST}\"" >> "${SCRATCH}/built_images.env"
echo "export SWEBENCH_TAG_IMG=\"${TARGET_IMG}\"" >> "${SCRATCH}/built_images.env"

echo "=== [6] Running setup_ubuntu_c3_and_index.sh ==="
bash "${SCRATCH}/setup_ubuntu_c3_and_index.sh"
