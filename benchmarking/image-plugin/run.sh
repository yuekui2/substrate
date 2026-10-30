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

# Runs one loader Job per arm in parallel and copies its results out.
#
#   run.sh bake [arm...]               bake goldens (untimed)
#   run.sh rep <n> [arm...]            one timed rep
#
# Results go to ${RUN_DIR}/<mode>-<rep>/<arm>/. Extra loader flags can be
# passed in LOADER_ARGS, e.g. LOADER_ARGS="-turns 2 -actors 8".

set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${HERE}/../.." && pwd)"
: "${RUN_DIR:?set RUN_DIR}"
ALL_ARMS=(pull riptide nydus spegel)

mode="${1:?usage: run.sh bake|rep <n> [arm...]}"
shift
rep=0
if [[ "${mode}" == rep ]]; then
  rep="${1:?rep number}"
  shift
fi
arms=("$@")
if [[ ${#arms[@]} -eq 0 ]]; then
  arms=("${ALL_ARMS[@]}")
fi

cd "${ROOT}"
KO="$(cd hack/tools/ko && go tool -n ko)"
loader_mode=run
[[ "${mode}" == bake ]] && loader_mode=bake

extra=""
for a in -rep "${rep}" ${LOADER_ARGS:-}; do
  extra+=", \"${a}\""
done

run_arm() {
  local arm="$1" job dir pod start
  job="imgbench-${mode}-${rep}-${arm}"
  dir="${RUN_DIR}/${mode}-${rep}/${arm}"
  mkdir -p "${dir}"
  kubectl -n imgbench delete job "${job}" --ignore-not-found --wait=true >/dev/null
  JOB="${job}" ARM="${arm}" MODE="${loader_mode}" EXTRA_ARGS="${extra}" \
    envsubst '${JOB} ${ARM} ${MODE} ${EXTRA_ARGS}' <"${HERE}/manifests/loader-job.yaml.tmpl" >"${dir}/job.yaml"
  "${KO}" resolve --base-import-paths -f "${dir}/job.yaml" 2>/dev/null >"${dir}/job.resolved.yaml"
  start="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  kubectl apply -f "${dir}/job.resolved.yaml"
  kubectl -n imgbench wait --for=condition=Ready pod -l "job-name=${job}" --timeout=10m
  pod="$(kubectl -n imgbench get pods -l "job-name=${job}" -o jsonpath='{.items[0].metadata.name}')"
  echo "${arm}: ${job} pod ${pod} started ${start}"
  # The loader serves /results on :8080. Poll done, copying the partial
  # JSONL each time so a crash still leaves data.
  local port=$((18080 + RANDOM % 1000)) pf
  while :; do
    kubectl -n imgbench port-forward "pod/${pod}" "${port}:8080" >/dev/null 2>&1 &
    pf=$!
    sleep 3
    curl -sf "http://127.0.0.1:${port}/out.jsonl" -o "${dir}/turns.jsonl.part" && mv "${dir}/turns.jsonl.part" "${dir}/turns.jsonl"
    if curl -sf "http://127.0.0.1:${port}/done" -o "${dir}/done"; then
      kill "${pf}" 2>/dev/null
      break
    fi
    kill "${pf}" 2>/dev/null
    if ! kubectl -n imgbench get pod "${pod}" >/dev/null 2>&1; then
      echo "${arm}: pod ${pod} is gone"
      break
    fi
    sleep 60
  done
  kubectl -n imgbench logs "${pod}" >"${dir}/loader.log" 2>&1
  echo "${arm}: done (exit $(cat "${dir}/done" 2>/dev/null)), $(wc -l <"${dir}/turns.jsonl" 2>/dev/null || echo 0) lines, $(grep -c '"error"' "${dir}/turns.jsonl" 2>/dev/null || echo 0) errors"
  # atelet's restore timing breakdown for this rep, from the arm's nodes.
  for n in $(kubectl get nodes -l "bench-arm=${arm}" -o jsonpath='{.items[*].metadata.name}'); do
    p="$(kubectl -n ate-system get pods -l app=atelet --field-selector "spec.nodeName=${n}" -o jsonpath='{.items[0].metadata.name}')"
    kubectl -n ate-system logs "${p}" -c atelet --since-time="${start}" 2>/dev/null |
      grep -E 'Restore timing breakdown|falling back|ImagePlugin|Suspended actor on draining' >"${dir}/atelet-${n}.log" || true
  done
  kubectl -n imgbench delete job "${job}" --wait=false >/dev/null
}

pids=()
for arm in "${arms[@]}"; do
  run_arm "${arm}" &
  pids+=($!)
done
for pid in "${pids[@]}"; do
  wait "${pid}"
done
