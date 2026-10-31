#!/usr/bin/env bash
# Seeds variant with patched nydusd read amplification, 64 actors on old images.
set -uo pipefail
cd /usr/local/google/home/kuiyue/src/substrate || exit 1
source .ate-dev-env.sh
while pgrep -f scratch/sched.sh >/dev/null; do sleep 30; done
B=benchmarking/image-plugin
KO=$(cd hack/tools/ko && go tool -n ko)
base=$B/results/20261031-nydus-amp
mkdir -p "$base"
for amp in 16777216 4194304; do
  export RUN_DIR=$base/amp$amp
  mkdir -p "$RUN_DIR"
  $KO resolve --base-import-paths -f image-plugin/manifests/nydus-dragonfly-seeds-amp.yaml 2>"$RUN_DIR/ko.log" |
    DRAGONFLY_PREFETCH=false NYDUS_PREFETCH=false NYDUS_USER_IO_BATCH_SIZE=$amp python3 $B/scope_plugin.py nydus >"$RUN_DIR/nydus-dragonfly.applied.yaml" || exit 1
  $B/ev.sh "amp$amp-apply" "kubectl apply -f $RUN_DIR/nydus-dragonfly.applied.yaml"
  $B/ev.sh "amp$amp-restart-scheduler" "kubectl -n ate-system rollout restart deploy/dragonfly-scheduler"
  $B/ev.sh "amp$amp-scheduler-ready" "kubectl -n ate-system rollout status deploy/dragonfly-scheduler --timeout=5m"
  $B/ev.sh "amp$amp-recreate-pools" "$B/pools.sh recreate nydus"
  $B/ev.sh "amp$amp-wait-ready" "$B/wait-ready.sh nydus"
  $B/ev.sh "amp$amp-check-env" "kubectl -n ate-system get ds image-plugin-nydus-dragonfly -o jsonpath='{.spec.template.spec.containers[?(@.name==\"nydus-snapshotter\")].env}'"
  $B/ev.sh "amp$amp-seed-preload" "$B/seed-preload.sh $B/images-64.jsonl 64"
  DENSITY=8 LOADER_ARGS="-images /config/images-64.jsonl -actors 64 -turns 5 -reads 1000" $B/ev.sh "amp$amp-run-64" "$B/run.sh rep 1 nydus"
  for p in $(kubectl -n ate-system get pods -l app=ate-image-plugin,ate.dev/image-plugin=nydus-dragonfly -o name | grep -v seed); do
    kubectl -n ate-system logs "$p" -c nydus-snapshotter >"$RUN_DIR/${p##*/}.snapshotter.log" 2>&1
  done
done
echo AMP-DONE
