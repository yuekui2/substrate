#!/usr/bin/env bash
# Seeds serve blobs over HTTP to nydusd (seedhttp variant): probe, 1 actor, 64 actors.
set -uo pipefail
cd /usr/local/google/home/kuiyue/src/substrate || exit 1
source .ate-dev-env.sh
S=/usr/local/google/home/kuiyue/.gemini/jetski/brain/43283b1e-d79e-4611-9b15-474096a7e806/scratch
B=benchmarking/image-plugin
KO=$(cd hack/tools/ko && go tool -n ko)
base=$B/results/20261031-nydus-seedhttp
mkdir -p "$base"
export RUN_DIR=$base/setup
mkdir -p "$RUN_DIR"
$KO resolve --base-import-paths -f image-plugin/manifests/nydus-dragonfly-seedhttp.yaml 2>"$RUN_DIR/ko.log" |
  DRAGONFLY_PREFETCH=false NYDUS_PREFETCH=false python3 $B/scope_plugin.py nydus >"$RUN_DIR/nydus-dragonfly.applied.yaml" || exit 1
$B/ev.sh apply "kubectl apply -f $RUN_DIR/nydus-dragonfly.applied.yaml"
$B/ev.sh restart-scheduler "kubectl -n ate-system rollout restart deploy/dragonfly-scheduler"
$B/ev.sh scheduler-ready "kubectl -n ate-system rollout status deploy/dragonfly-scheduler --timeout=5m"
$B/ev.sh recreate-pools "$B/pools.sh recreate nydus"
$B/ev.sh wait-ready "$B/wait-ready.sh nydus"
$B/ev.sh blob-seed-ready "kubectl -n ate-system rollout status ds/nydus-blob-seed --timeout=10m"
$B/ev.sh seed-preload "SAVE_DIR=/var/lib/ate/nydus-seed-blobs $B/seed-preload.sh $B/images-64.jsonl 64"
$B/ev.sh probe "$S/seedprobe.sh"
for n in 1 64; do
  export RUN_DIR=$base/n$n
  mkdir -p "$RUN_DIR"
  DENSITY=8 LOADER_ARGS="-images /config/images-64.jsonl -actors $n -turns 5 -reads 1000" $B/ev.sh "run-$n" "$B/run.sh rep 1 nydus"
done
mkdir -p "$base/logs"
for p in $(kubectl -n ate-system get pods -l app=ate-image-plugin,ate.dev/image-plugin=nydus-dragonfly -o name); do
  kubectl -n ate-system logs "$p" -c nydus-snapshotter 2>&1 | gzip >"$base/logs/${p##*/}.snapshotter.log.gz"
done
echo SEEDHTTP-DONE
