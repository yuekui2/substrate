#!/usr/bin/env bash
# Nydus seedhttp at 24 actors per node, 2 per worker, with a bpftrace tracer
# on every bench node timing fscache on-demand reads in nydusd.
set -uo pipefail
cd /usr/local/google/home/kuiyue/src/substrate || exit 1
source .ate-dev-env.sh
S=/usr/local/google/home/kuiyue/.gemini/jetski/brain/43283b1e-d79e-4611-9b15-474096a7e806/scratch
B=benchmarking/image-plugin
base=$B/results/20261031-nydus-fstrace
mkdir -p "$base/trace"
export RUN_DIR=$base/setup
mkdir -p "$RUN_DIR"
$B/ev.sh recreate-bench-pool "KEEP_SEEDS=1 $B/pools.sh recreate nydus"
$B/ev.sh wait-ready "$B/wait-ready.sh nydus"
kubectl -n imgbench create configmap fstrace --from-file="$S/fstrace.bt" --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f "$S/fstrace-ds.yaml"
$B/ev.sh tracer-ready "kubectl -n imgbench rollout status ds/fstrace --timeout=5m"
for _ in $(seq 1 60); do
  ready=$(for p in $(kubectl -n imgbench get pods -l app=fstrace -o name); do kubectl -n imgbench logs "$p" 2>/dev/null | grep -c '^Attaching'; done | paste -sd+ | bc)
  [[ ${ready:-0} -ge 3 ]] && break
  sleep 5
done
echo "tracers attached: ${ready:-0}"
export RUN_DIR=$base/n24
mkdir -p "$RUN_DIR"
DENSITY=2 LOADER_ARGS="-actors 72 -turns 5 -reads 1000" $B/ev.sh run-24 "$B/run.sh rep 1 nydus"
for p in $(kubectl -n imgbench get pods -l app=fstrace -o name); do
  n=$(kubectl -n imgbench get "$p" -o jsonpath='{.spec.nodeName}')
  kubectl -n imgbench logs "$p" >"$base/trace/${n}.log" 2>&1
done
kubectl -n imgbench delete ds fstrace --wait=false
echo FSTRACE-DONE
