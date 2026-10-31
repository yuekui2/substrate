#!/usr/bin/env bash
# seedhttp plus manifests and bootstrap layers from the seeds. Keeps the seed
# nodes, recreates the bench pool cold, preloads blobs and manifests.
set -uo pipefail
cd /usr/local/google/home/kuiyue/src/substrate || exit 1
source .ate-dev-env.sh
B=benchmarking/image-plugin
KO=$(cd hack/tools/ko && go tool -n ko)
base=$B/results/20261031-nydus-seedhttp-boot
mkdir -p "$base"
export RUN_DIR=$base/setup
mkdir -p "$RUN_DIR"
$KO resolve --base-import-paths -f image-plugin/manifests/nydus-dragonfly-seedhttp.yaml 2>"$RUN_DIR/ko.log" |
  DRAGONFLY_PREFETCH=false NYDUS_PREFETCH=false python3 $B/scope_plugin.py nydus >"$RUN_DIR/nydus-dragonfly.applied.yaml" || exit 1
$B/ev.sh apply "kubectl apply -f $RUN_DIR/nydus-dragonfly.applied.yaml"
$B/ev.sh restart-blob-seed "kubectl -n ate-system delete pods -l app=nydus-blob-seed --wait=false"
$B/ev.sh blob-seed-ready "kubectl -n ate-system rollout status ds/nydus-blob-seed --timeout=5m" || exit 1
$B/ev.sh recreate-bench-pool "KEEP_SEEDS=1 $B/pools.sh recreate nydus"
$B/ev.sh wait-ready "$B/wait-ready.sh nydus"
$B/ev.sh seed-preload "SAVE_DIR=/var/lib/ate/nydus-seed-blobs $B/seed-preload.sh $B/images-64.jsonl 64" || exit 1
ref=$(head -1 $B/images-64.jsonl | jq -r .nydus)
repo=${ref%@*}
repo=${repo#us-central1-docker.pkg.dev/}
ip=$(kubectl -n ate-system get svc nydus-blob-seed -o jsonpath='{.spec.clusterIP}')
pod=$(kubectl -n ate-system get pods -l app=ate-image-plugin,ate.dev/dragonfly-seed=false -o name | head -1)
$B/ev.sh check-manifest "kubectl -n ate-system exec ${pod#pod/} -c nydus-snapshotter -- nsenter -t 1 -m -n -- curl -s -o /dev/null -w '%{http_code} %{size_download} %{time_total}\n' http://$ip:4010/v2/$repo/manifests/${ref#*@}"
for n in 1 64; do
  export RUN_DIR=$base/n$n
  mkdir -p "$RUN_DIR"
  DENSITY=8 LOADER_ARGS="-images /config/images-64.jsonl -actors $n -turns 5 -reads 1000" $B/ev.sh "run-$n" "$B/run.sh rep 1 nydus"
done
mkdir -p "$base/logs"
for p in $(kubectl -n ate-system get pods -l app=ate-image-plugin,ate.dev/image-plugin=nydus-dragonfly -o name); do
  kubectl -n ate-system logs "$p" -c nydus-snapshotter 2>&1 | gzip >"$base/logs/${p##*/}.snapshotter.log.gz"
  kubectl -n ate-system logs "$p" -c image-plugin-dragonfly 2>&1 | gzip >"$base/logs/${p##*/}.plugin.log.gz"
done
echo SEEDHTTP3-DONE
