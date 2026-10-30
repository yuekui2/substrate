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

# Waits until every arm's workers, image plugin and atelet are ready, for
# example after pools.sh recreate.

set -euo pipefail

for arm in pull riptide nydus spegel; do
  kubectl -n imgbench rollout status "deploy/bench-${arm}" --timeout=40m
  if [[ "${arm}" != pull ]]; then
    ds="image-plugin-${arm}"
    [[ "${arm}" == nydus ]] && ds=image-plugin-nydus-dragonfly
    kubectl -n ate-system rollout status "ds/${ds}" --timeout=40m
  fi
done
kubectl -n ate-system rollout status ds/atelet --timeout=20m
kubectl -n imgbench get workerpools
kubectl -n ate-system get pods -o wide -l app=ate-image-plugin
