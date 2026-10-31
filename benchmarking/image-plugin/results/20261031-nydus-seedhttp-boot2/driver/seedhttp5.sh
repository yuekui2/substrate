#!/usr/bin/env bash
# seedhttp with manifests and bootstraps from the seeds, after the manifest
# permission fix. Seeds keep their data; the bench pool is recreated cold.
set -uo pipefail
cd /usr/local/google/home/kuiyue/src/substrate || exit 1
source .ate-dev-env.sh
B=benchmarking/image-plugin
base=$B/results/20261031-nydus-seedhttp-boot2
mkdir -p "$base"
export RUN_DIR=$base/setup
mkdir -p "$RUN_DIR"
$B/ev.sh recreate-bench-pool "KEEP_SEEDS=1 $B/pools.sh recreate nydus"
$B/ev.sh wait-ready "$B/wait-ready.sh nydus"
ref=$(head -1 $B/images-64.jsonl | jq -r .nydus)
repo=${ref%@*}
repo=${repo#us-central1-docker.pkg.dev/}
ip=$(kubectl -n ate-system get svc nydus-blob-seed -o jsonpath='{.spec.clusterIP}')
pod=$(kubectl -n ate-system get pods -l app=ate-image-plugin,ate.dev/dragonfly-seed=false -o name | head -1)
out=$(kubectl -n ate-system exec "${pod#pod/}" -c nydus-snapshotter -- nsenter -t 1 -m -n -- curl -s -o /dev/null -w '%{http_code}' "http://$ip:4010/v2/$repo/manifests/${ref#*@}")
echo "manifest check: $out" | tee "$RUN_DIR/manifest-check.txt"
[[ $out == 200 ]] || exit 1
for n in 1 64; do
  export RUN_DIR=$base/n$n
  mkdir -p "$RUN_DIR"
  DENSITY=8 LOADER_ARGS="-images /config/images-64.jsonl -actors $n -turns 5 -reads 1000" $B/ev.sh "run-$n" "$B/run.sh rep 1 nydus"
done
mkdir -p "$base/logs"
for p in $(kubectl -n ate-system get pods -l app=ate-image-plugin,ate.dev/image-plugin=nydus-dragonfly -o name); do
  kubectl -n ate-system logs "$p" -c image-plugin-dragonfly 2>&1 | gzip >"$base/logs/${p##*/}.plugin.log.gz"
done
echo SEEDHTTP5-DONE
