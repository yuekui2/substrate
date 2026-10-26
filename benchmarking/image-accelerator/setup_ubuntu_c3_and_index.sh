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
source "${BENCH_DIR}/built_images.env"

COS_NODE="${COS_NODE:-gke-substrate-stream-c3-std22-cos-poo-1abf5470-wj62}"
UBUNTU_NODE="${UBUNTU_NODE:-gke-substrate-stream-c3-std22-ubuntu--5a335c0d-1261}"
PEER_NODE="${PEER_NODE:-gke-substrate-stream-erofs-ubuntu-poo-ea32e702-fkxc}"
NO_STREAM_COS_NODE="${NO_STREAM_COS_NODE:-gke-substrate-stream-t-no-stream-pool-7360e0e0-p1tz}"

TMP_BIN_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_BIN_DIR}"' EXIT

echo "=== [1] Compiling bench-runner and e2e-stream-probe ==="
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${TMP_BIN_DIR}/bench-runner" ./benchmarking/image-accelerator/bench-runner
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${TMP_BIN_DIR}/e2e-stream-probe" ./benchmarking/image-accelerator/e2e-stream-probe

echo "=== [2] Deploying fresh bench-helper pods on target nodes ==="
kubectl label namespace ate-system ate.dev/api-access=true --overwrite
for NODE in "${COS_NODE}" "${UBUNTU_NODE}" "${PEER_NODE}" "${NO_STREAM_COS_NODE}"; do
  POD="bench-helper-${NODE##*-}"
  kubectl -n ate-system delete pod "${POD}" --ignore-not-found --wait=true
  kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: ${POD}
  namespace: ate-system
  labels:
    app: bench-helper
spec:
  nodeName: ${NODE}
  hostPID: true
  serviceAccountName: ate-client
  tolerations:
  - operator: Exists
  containers:
  - name: helper
    image: debian:13-slim@sha256:d7e12182ce18b85b93007c1dedf31f2d29e01ccf3182cc4017c709b6259bc132
    command: ["/bin/sh", "-c", "sleep 86400"]
    securityContext:
      privileged: true
    volumeMounts:
    - name: host-bin
      mountPath: /home/kubernetes/bin
    - name: run-ate
      mountPath: /run/ate
    - name: podidentity
      mountPath: /run/podidentity.podcert.ate.dev
  volumes:
  - name: host-bin
    hostPath:
      path: /home/kubernetes/bin
      type: DirectoryOrCreate
  - name: run-ate
    hostPath:
      path: /run/ate
      type: DirectoryOrCreate
  - name: podidentity
    projected:
      sources:
      - podCertificate:
          signerName: podidentity.podcert.ate.dev/identity
          keyType: ECDSAP256
          credentialBundlePath: credential-bundle.pem
EOF
done

for NODE in "${COS_NODE}" "${UBUNTU_NODE}" "${PEER_NODE}" "${NO_STREAM_COS_NODE}"; do
  POD="bench-helper-${NODE##*-}"
  kubectl -n ate-system wait --for=condition=Ready "pod/${POD}" --timeout=120s
  kubectl -n ate-system cp "${TMP_BIN_DIR}/bench-runner" "${POD}:/home/kubernetes/bin/bench-runner.new"
  kubectl -n ate-system cp "${TMP_BIN_DIR}/e2e-stream-probe" "${POD}:/home/kubernetes/bin/e2e-stream-probe.new"
  kubectl -n ate-system exec "${POD}" -- nsenter -t 1 -m -u -i -n -- sh -c "
    chmod +x /home/kubernetes/bin/bench-runner.new /home/kubernetes/bin/e2e-stream-probe.new
    mv /home/kubernetes/bin/bench-runner.new /home/kubernetes/bin/bench-runner
    mv /home/kubernetes/bin/e2e-stream-probe.new /home/kubernetes/bin/e2e-stream-probe
  "
done

echo "=== [3] Ensuring soci v0.15.0 on ${PEER_NODE}, ${COS_NODE}, and ${UBUNTU_NODE} ==="
for NODE in "${PEER_NODE}" "${COS_NODE}" "${UBUNTU_NODE}"; do
  POD="bench-helper-${NODE##*-}"
  kubectl -n ate-system exec "${POD}" -- nsenter -t 1 -m -u -i -n -- sh -c '
    if [ ! -x /home/kubernetes/bin/soci ] || [ ! -x /home/kubernetes/bin/soci-snapshotter-grpc ]; then
      curl -fsSL "https://github.com/awslabs/soci-snapshotter/releases/download/v0.15.0/soci-snapshotter-0.15.0-linux-amd64.tar.gz" \
        | tar -xz -C /home/kubernetes/bin soci-snapshotter-grpc soci
      chmod +x /home/kubernetes/bin/soci-snapshotter-grpc /home/kubernetes/bin/soci
    fi
    ls -lh /home/kubernetes/bin/soci /home/kubernetes/bin/soci-snapshotter-grpc
  '
done
