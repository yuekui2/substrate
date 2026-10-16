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
PEER_NODE="gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc"

echo "=== [1] Recompiling updated bench-runner and copying to bench-helper pods ==="
mkdir -p "${ROOT}/cmd/bench-runner-tmp" && cp "${SCRATCH}/bench_runner.go" "${ROOT}/cmd/bench-runner-tmp/main.go" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${SCRATCH}/bench-runner" ./cmd/bench-runner-tmp && rm -rf "${ROOT}/cmd/bench-runner-tmp"
for NODE in "${COS_NODE}" "${UBUNTU_NODE}" "${PEER_NODE}"; do
  POD="bench-helper-${NODE##*-}"
  kubectl -n ate-system cp "${SCRATCH}/bench-runner" "${POD}:/home/kubernetes/bin/bench-runner.new"
  kubectl -n ate-system exec "${POD}" -- nsenter -t 1 -m -u -i -n -- sh -c "chmod +x /home/kubernetes/bin/bench-runner.new && mv /home/kubernetes/bin/bench-runner.new /home/kubernetes/bin/bench-runner"
done

echo "=== [2] Installing soci v0.15.0 on ${PEER_NODE}, ${COS_NODE}, and ${UBUNTU_NODE}, and pushing AWS SOCI index ==="
TOKEN=$(gcloud auth print-access-token)
NYDUS_TARGET="us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd:swebench-nydus-v6"

for NODE in "${PEER_NODE}" "${COS_NODE}" "${UBUNTU_NODE}"; do
  POD="bench-helper-${NODE##*-}"
  kubectl -n ate-system exec "${POD}" -- nsenter -t 1 -m -u -i -n -- sh -c '
    if [ ! -x /home/kubernetes/bin/soci ]; then
      curl -fsSL "https://github.com/awslabs/soci-snapshotter/releases/download/v0.15.0/soci-snapshotter-0.15.0-linux-amd64.tar.gz" \
        | tar -xz -C /home/kubernetes/bin soci-snapshotter-grpc soci
      chmod +x /home/kubernetes/bin/soci-snapshotter-grpc /home/kubernetes/bin/soci
    fi
    ls -lh /home/kubernetes/bin/soci /home/kubernetes/bin/soci-snapshotter-grpc
  '
done

kubectl -n ate-system exec "bench-helper-${PEER_NODE##*-}" -- nsenter -t 1 -m -u -i -n -- sh -c "
  export PATH=/home/kubernetes/bin:\${PATH}
  echo '--- Building and pushing AWS SOCI index for ${SWEBENCH_IMG} ---'
  soci --namespace k8s.io create --min-layer-size 0 '${SWEBENCH_IMG}'
  soci --namespace k8s.io push --user 'oauth2accesstoken:${TOKEN}' '${SWEBENCH_IMG}'
"

NYDUS_DIGEST=$(/usr/local/google/home/kuiyue/go/bin/crane digest "${NYDUS_TARGET}")
grep -v "SWEBENCH_NYDUS" "${SCRATCH}/built_images.env" > "${SCRATCH}/built_images.env.tmp" || true
mv "${SCRATCH}/built_images.env.tmp" "${SCRATCH}/built_images.env"
echo "export SWEBENCH_NYDUS_IMG=\"us-central1-docker.pkg.dev/kuiyue-gke-dev/kuiyue-gke-dev-repo/ate-images/sandbox-04176181c57f0bf23e61506b0ddcf1fd@${NYDUS_DIGEST}\"" >> "${SCRATCH}/built_images.env"
echo "export SWEBENCH_NYDUS_TAG_IMG=\"${NYDUS_TARGET}\"" >> "${SCRATCH}/built_images.env"
cat "${SCRATCH}/built_images.env"
