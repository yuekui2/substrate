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

# Waits for the arm's loader pod to start, then takes a CPU profile of the
# Nydus image plugin on every bench node of the arm, all at once. Deploy the
# plugin with NYDUS_PLUGIN_ARGS=--pprof-addr=127.0.0.1:6060.
#
#   profile-plugin.sh <arm> <outdir> [seconds]
#
# Writes <outdir>/plugin-<node>.pprof.

set -euo pipefail

arm="${1:?usage: profile-plugin.sh <arm> <outdir> [seconds]}"
out="${2:?usage: profile-plugin.sh <arm> <outdir> [seconds]}"
secs="${3:-30}"
mkdir -p "${out}"

until kubectl -n imgbench get pods -l "job-name" -o jsonpath='{range .items[*]}{.metadata.labels.job-name} {.status.phase}{"\n"}{end}' 2>/dev/null |
  grep -qE "^imgbench-(run|rep)-[0-9]+-${arm}[^ ]* Running"; do
  sleep 1
done
echo "loader running at $(date -u +%H:%M:%S), profiling ${secs}s"

pids=()
port=16060
for node in $(kubectl get nodes -l "bench-arm=${arm}" -o jsonpath='{.items[*].metadata.name}'); do
  pod="$(kubectl -n ate-system get pods -l app=ate-image-plugin --field-selector "spec.nodeName=${node}" -o jsonpath='{.items[0].metadata.name}')"
  port=$((port + 1))
  (
    kubectl -n ate-system port-forward "pod/${pod}" "${port}:6060" >/dev/null 2>&1 &
    pf=$!
    sleep 2
    curl -sf -o "${out}/plugin-${node##*-}.pprof" "http://127.0.0.1:${port}/debug/pprof/profile?seconds=${secs}" ||
      echo "profile of ${node} failed" >&2
    kill "${pf}"
  ) &
  pids+=($!)
done
for pid in "${pids[@]}"; do
  wait "${pid}"
done
ls -l "${out}"/plugin-*.pprof
