#!/usr/bin/env bash
# Seeds variant rerun with scheduler instrumentation: 1 actor, then 64.
set -uo pipefail
cd /usr/local/google/home/kuiyue/src/substrate || exit 1
source .ate-dev-env.sh
S=/usr/local/google/home/kuiyue/.gemini/jetski/brain/43283b1e-d79e-4611-9b15-474096a7e806/scratch
B=benchmarking/image-plugin
KO=$(cd hack/tools/ko && go tool -n ko)
base=$B/results/20261031-nydus-sched
mkdir -p "$base"
snap() {
  local p
  p=$(kubectl -n ate-system get pods -l app=dragonfly-scheduler -o name | head -1 | cut -d/ -f2)
  kubectl get --raw "/api/v1/namespaces/ate-system/pods/$p:8000/proxy/metrics" >"$base/sched-metrics-$1.txt"
}
export RUN_DIR=$base/setup
mkdir -p "$RUN_DIR"
$KO resolve --base-import-paths -f image-plugin/manifests/nydus-dragonfly-seeds.yaml 2>"$RUN_DIR/ko.log" |
  DRAGONFLY_PREFETCH=false NYDUS_PREFETCH=false python3 $B/scope_plugin.py nydus >"$RUN_DIR/nydus-dragonfly.applied.yaml" || exit 1
$B/ev.sh apply "kubectl apply -f $RUN_DIR/nydus-dragonfly.applied.yaml"
$B/ev.sh restart-scheduler "kubectl -n ate-system rollout restart deploy/dragonfly-scheduler"
$B/ev.sh scheduler-ready "kubectl -n ate-system rollout status deploy/dragonfly-scheduler --timeout=5m"
$B/ev.sh recreate-pools "$B/pools.sh recreate nydus"
$B/ev.sh wait-ready "$B/wait-ready.sh nydus"
$B/ev.sh seed-preload "$B/seed-preload.sh $B/images-64.jsonl 64"
snap 0-before
setsid "$S/sched-sample.sh" "$base/sched-samples.log" &
sampler=$!
for n in 1 64; do
  export RUN_DIR=$base/n$n
  mkdir -p "$RUN_DIR"
  echo "$(date +%s) start n$n" >>"$base/sched-samples.log"
  DENSITY=8 LOADER_ARGS="-images /config/images-64.jsonl -actors $n -turns 5 -reads 1000" $B/ev.sh "run-$n" "$B/run.sh rep 1 nydus"
  echo "$(date +%s) end n$n" >>"$base/sched-samples.log"
  snap "after-n$n"
done
kill -- -"$sampler" 2>/dev/null || kill "$sampler"
mkdir -p "$base/logs"
for p in $(kubectl -n ate-system get pods -l app=ate-image-plugin,ate.dev/image-plugin=nydus-dragonfly -o name); do
  for c in $(kubectl -n ate-system get "$p" -o jsonpath='{.spec.containers[*].name}'); do
    kubectl -n ate-system logs "$p" -c "$c" >"$base/logs/${p##*/}.$c.log" 2>&1
  done
done
kubectl -n ate-system logs deploy/dragonfly-scheduler >"$base/logs/scheduler.log" 2>&1
echo SCHED-DONE
