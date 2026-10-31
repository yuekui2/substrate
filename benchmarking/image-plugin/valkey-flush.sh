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

# Empties the Memorystore for Valkey cluster of the riptide-ms arm, so a run
# starts cold. Runs valkey-cli in a pod on the system pool, because the PSC
# endpoint is only reachable from the VPC. Fails unless every node then
# reports DBSIZE 0.
#
#   VALKEY_EP=10.x.x.x:6379 valkey-flush.sh

set -euo pipefail

: "${VALKEY_EP:?set VALKEY_EP to the discovery endpoint}"
# valkey/valkey:9.0-alpine
IMAGE=docker.io/valkey/valkey@sha256:2d04268dfa26ab31cfa3fdc538866e76438b2d00e41db41b8cb86d780ac5a416

cli() {
  kubectl -n imgbench run "valkey-cli-$(date +%s%N)" --rm -i --quiet --restart=Never \
    --image="${IMAGE}" \
    --overrides='{"spec":{"nodeSelector":{"cloud.google.com/gke-nodepool":"bench-system"}}}' \
    --command -- valkey-cli "$@"
}

cli --cluster call "${VALKEY_EP}" FLUSHALL SYNC --cluster-only-masters
# Replicas apply the flush asynchronously.
for _ in $(seq 1 10); do
  out="$(cli --cluster call "${VALKEY_EP}" DBSIZE)"
  echo "${out}"
  if ! grep -Eq ': [1-9][0-9]*$' <<<"${out}" && grep -Eq ': 0$' <<<"${out}"; then
    exit 0
  fi
  sleep 2
done
echo "Valkey still has keys after FLUSHALL" >&2
exit 1
