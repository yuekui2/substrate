#!/usr/bin/env bash
# seedhttp rerun after the nginx config fix. Keeps the seeds (blobs on disk),
# recreates the bench pool so bench caches start cold.
set -uo pipefail
cd /usr/local/google/home/kuiyue/src/substrate || exit 1
source .ate-dev-env.sh
S=/usr/local/google/home/kuiyue/.gemini/jetski/brain/43283b1e-d79e-4611-9b15-474096a7e806/scratch
B=benchmarking/image-plugin
KO=$(cd hack/tools/ko && go tool -n ko)
old=$B/results/20261031-nydus-seedhttp
[[ -d $old ]] && mv "$old" "$B/results/20261031-nydus-seedhttp-nginxdown"
base=$old
mkdir -p "$base"
export RUN_DIR=$base/setup
mkdir -p "$RUN_DIR"
$KO resolve --base-import-paths -f image-plugin/manifests/nydus-dragonfly-seedhttp.yaml 2>"$RUN_DIR/ko.log" |
  DRAGONFLY_PREFETCH=false NYDUS_PREFETCH=false python3 $B/scope_plugin.py nydus >"$RUN_DIR/nydus-dragonfly.applied.yaml" || exit 1
$B/ev.sh apply "kubectl apply -f $RUN_DIR/nydus-dragonfly.applied.yaml"
$B/ev.sh restart-blob-seed "kubectl -n ate-system delete pods -l app=nydus-blob-seed --wait=false"
$B/ev.sh blob-seed-ready "kubectl -n ate-system rollout status ds/nydus-blob-seed --timeout=5m" || exit 1
for p in $(kubectl -n ate-system get pods -l app=ate-image-plugin,ate.dev/dragonfly-seed=true -o name); do
  $B/ev.sh "blob-count-${p##*/}" "kubectl -n ate-system exec ${p##*/} -c nydus-snapshotter -- nsenter -t 1 -m -- sh -c 'ls /var/lib/ate/nydus-seed-blobs | grep -vc tmp; du -sh /var/lib/ate/nydus-seed-blobs'"
done
$B/ev.sh recreate-bench-pool "KEEP_SEEDS=1 $B/pools.sh recreate nydus"
$B/ev.sh wait-ready "$B/wait-ready.sh nydus"
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
echo SEEDHTTP2-DONE
