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
#
# DENSITY=<n> runs n actors per 1-vCPU / 2 GiB worker: templates live in
# atespace imgbench-<arm>-w<n>, each actor gets 1000/n millicores and 1984/n
# MiB, and results go to <arm>-w<n>/. Unset, it uses imgbench-<arm> (10 per
# worker, baked by an earlier run).
#
# TAG=<t> appends t to the atespace, Job and results names, so templates
# baked from another image list (LOADER_ARGS="-images ...") stay separate.

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

cd "${ROOT}" || exit 1
KO="$(cd hack/tools/ko && go tool -n ko)"
ATE="${ROOT}/bin/kubectl-ate"
loader_mode=run
[[ "${mode}" == bake ]] && loader_mode=bake

suffix=""
density_args=""
if [[ -n "${DENSITY:-}" ]]; then
  # A 1 vCPU / 2 GiB worker reports 1000m and 1984 MiB of actor capacity.
  suffix="-w${DENSITY}"
  density_args="-cpu-millicores $((1000 / DENSITY)) -memory-bytes $((1984 * 1024 * 1024 / DENSITY))"
fi

extra=""
# shellcheck disable=SC2086 # word splitting is intended
for a in -rep "${rep}" ${density_args} ${LOADER_ARGS:-}; do
  extra+=", \"${a}\""
done

run_arm() {
  local arm="$1" job dir pod start atespace
  atespace="imgbench-${arm}${TAG:-}${suffix}"
  job="imgbench-${mode}-${rep}-${arm}${TAG:-}${suffix}"
  dir="${RUN_DIR}/${mode}-${rep}/${arm}${TAG:-}${suffix}"
  mkdir -p "${dir}"
  "${ATE}" get atespace "${atespace}" >/dev/null 2>&1 || "${ATE}" create atespace "${atespace}" 2>/dev/null
  kubectl -n imgbench delete job "${job}" --ignore-not-found --wait=true >/dev/null
  # shellcheck disable=SC2016 # envsubst takes the variable names literally
  JOB="${job}" ARM="${arm}" ATESPACE="${atespace}" MODE="${loader_mode}" EXTRA_ARGS="${extra}" \
    envsubst '${JOB} ${ARM} ${ATESPACE} ${MODE} ${EXTRA_ARGS}' <"${HERE}/manifests/loader-job.yaml.tmpl" >"${dir}/job.yaml"
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
  local nodes
  nodes="$(kubectl get nodes -l "bench-arm=${arm}" -o jsonpath='{.items[*].metadata.name}')"
  for n in ${nodes}; do
    p="$(kubectl -n ate-system get pods -l app=atelet --field-selector "spec.nodeName=${n}" -o jsonpath='{.items[0].metadata.name}')"
    kubectl -n ate-system logs "${p}" -c atelet --since-time="${start}" 2>/dev/null |
      grep -E 'Restore timing breakdown|falling back|ImagePlugin|Suspended actor on draining' >"${dir}/atelet-${n}.log" || true
  done
  # Seed nodes, if the arm has any.
  nodes+=" $(kubectl get nodes -l "bench-arm=${arm}-seed" -o jsonpath='{.items[*].metadata.name}')"
  # ateworker's restore and checkpoint phase timings for this rep.
  for p in $(kubectl -n imgbench get pods -o name | grep "^pod/bench-${arm}-"); do
    kubectl -n imgbench logs "${p}" --since-time="${start}" 2>/dev/null |
      grep -E 'Restore timing breakdown|Checkpoint timing breakdown' >>"${dir}/worker-phases.log" || true
  done
  # nginx access log of the Nydus seeds (deploy-nydus-best.sh), one JSON
  # line per request.
  if [[ "${arm}" == nydus ]]; then
    for p in $(kubectl -n ate-system get pods -l app=nydus-blob-seed -o name); do
      kubectl -n ate-system logs "${p}" --since-time="${start}" 2>/dev/null |
        grep '^{"msec"' >"${dir}/seed-access-${p#pod/}.jsonl" || true
    done
  fi
  # Node name and internal IP, to map seed requests to nodes.
  kubectl get nodes -l "bench-arm=${arm}" \
    -o jsonpath='{range .items[*]}{.metadata.name} {.status.addresses[?(@.type=="InternalIP")].address}{"\n"}{end}' >"${dir}/nodes.txt"
  collect_nodes "${arm}" "${dir}" "${start}" "${nodes}"
  kubectl -n imgbench delete job "${job}" --wait=false >/dev/null
}

# collect_nodes saves, per bench node of the arm: the 5 s disk counters from
# node-sampler, a snapshot of the image cache sizes and Spegel /stats, and
# Cloud Monitoring disk and network rates for the run window.
collect_nodes() {
  local arm="$1" dir="$2" start="$3" nodes="$4" n p end
  end="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  : >"${dir}/node-snapshots.jsonl"
  : >"${dir}/diskstats.jsonl"
  for n in ${nodes}; do
    p="$(kubectl -n imgbench get pods -l app=node-sampler --field-selector "spec.nodeName=${n}" -o jsonpath='{.items[0].metadata.name}')"
    kubectl -n imgbench logs "${p}" --since-time="${start}" >>"${dir}/diskstats.jsonl" 2>/dev/null || true
    kubectl -n imgbench exec "${p}" -- /bin/bash /scripts/node-sampler.sh snapshot >>"${dir}/node-snapshots.jsonl" 2>/dev/null || true
  done
  # shellcheck disable=SC2086 # nodes is a space-separated list
  python3 "${HERE}/monitoring.py" "${PROJECT:-kuiyue-gke-dev}" "${start}" "${end}" "${dir}/monitoring.jsonl" ${nodes} || true
}

pids=()
for arm in "${arms[@]}"; do
  run_arm "${arm}" &
  pids+=($!)
done
for pid in "${pids[@]}"; do
  wait "${pid}"
done
