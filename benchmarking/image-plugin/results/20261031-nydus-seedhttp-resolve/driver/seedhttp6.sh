#!/usr/bin/env bash
# seedhttp with image resolve, manifests and bootstraps from the seeds.
# Runs after seedhttp5. Seeds keep their data; bench pool recreated cold.
set -uo pipefail
cd /usr/local/google/home/kuiyue/src/substrate || exit 1
source .ate-dev-env.sh
while pgrep -f scratch/seedhttp5.sh >/dev/null; do sleep 20; done
B=benchmarking/image-plugin
KO=$(cd hack/tools/ko && go tool -n ko)
base=$B/results/20261031-nydus-seedhttp-resolve
mkdir -p "$base"
export RUN_DIR=$base/setup
mkdir -p "$RUN_DIR"
$KO resolve --base-import-paths -f image-plugin/manifests/nydus-dragonfly-seedhttp.yaml 2>"$RUN_DIR/ko.log" |
  DRAGONFLY_PREFETCH=false NYDUS_PREFETCH=false python3 $B/scope_plugin.py nydus >"$RUN_DIR/nydus-dragonfly.applied.yaml" || exit 1
$B/ev.sh apply "kubectl apply -f $RUN_DIR/nydus-dragonfly.applied.yaml" || exit 1
$B/ev.sh recreate-bench-pool "KEEP_SEEDS=1 $B/pools.sh recreate nydus"
$B/ev.sh wait-ready "$B/wait-ready.sh nydus"
$B/ev.sh seed-preload "SAVE_DIR=/var/lib/ate/nydus-seed-blobs $B/seed-preload.sh $B/images-64.jsonl 64" || exit 1
ref=$(head -1 $B/images-64.jsonl | jq -r .nydus)
repo=${ref%@*}
repo=${repo#us-central1-docker.pkg.dev/}
cfg=$(crane manifest "$ref" | jq -r .config.digest)
ip=$(kubectl -n ate-system get svc nydus-blob-seed -o jsonpath='{.spec.clusterIP}')
pod=$(kubectl -n ate-system get pods -l app=ate-image-plugin,ate.dev/dragonfly-seed=false -o name | head -1)
for path in "manifests/${ref#*@}" "blobs/$cfg"; do
  out=$(kubectl -n ate-system exec "${pod#pod/}" -c nydus-snapshotter -- nsenter -t 1 -m -n -- curl -s -o /dev/null -w '%{http_code}' "http://$ip:4010/v2/$repo/$path")
  echo "$path: $out" | tee -a "$RUN_DIR/seed-check.txt"
  [[ $out == 200 ]] || exit 1
done
for n in 1 64; do
  export RUN_DIR=$base/n$n
  mkdir -p "$RUN_DIR"
  DENSITY=8 LOADER_ARGS="-images /config/images-64.jsonl -actors $n -turns 5 -reads 1000" $B/ev.sh "run-$n" "$B/run.sh rep 1 nydus"
done
mkdir -p "$base/logs"
for p in $(kubectl -n ate-system get pods -l app=ate-image-plugin,ate.dev/image-plugin=nydus-dragonfly -o name); do
  kubectl -n ate-system logs "$p" -c image-plugin-dragonfly 2>&1 | gzip >"$base/logs/${p##*/}.plugin.log.gz"
done
echo SEEDHTTP6-DONE
