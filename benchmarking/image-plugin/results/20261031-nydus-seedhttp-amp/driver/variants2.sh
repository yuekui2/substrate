#!/usr/bin/env bash
# Two Nydus seedhttp variants at 24 actors per node, 2 per worker, cold
# bench pool each: fscache cache on tmpfs, then 128 KiB on-demand reads.
set -uo pipefail
cd /usr/local/google/home/kuiyue/src/substrate || exit 1
source .ate-dev-env.sh
B=benchmarking/image-plugin
KO=$(cd hack/tools/ko && go tool -n ko)
for v in tmpfs amp; do
  base=$B/results/20261031-nydus-seedhttp-$v
  mkdir -p "$base/setup"
  export RUN_DIR=$base/setup
  $KO resolve --base-import-paths -f image-plugin/manifests/nydus-dragonfly-seedhttp-$v.yaml 2>"$RUN_DIR/ko.log" |
    DRAGONFLY_PREFETCH=false NYDUS_PREFETCH=false python3 $B/scope_plugin.py nydus >"$RUN_DIR/nydus-dragonfly.applied.yaml" || exit 1
  $B/ev.sh apply "kubectl apply -f $RUN_DIR/nydus-dragonfly.applied.yaml" || exit 1
  $B/ev.sh recreate-bench-pool "KEEP_SEEDS=1 $B/pools.sh recreate nydus"
  $B/ev.sh wait-ready "$B/wait-ready.sh nydus"
  export RUN_DIR=$base/n24
  mkdir -p "$RUN_DIR"
  DENSITY=2 LOADER_ARGS="-actors 72 -turns 5 -reads 1000" $B/ev.sh run-24 "$B/run.sh rep 1 nydus"
  python3 $B/sweep_report.py "$RUN_DIR" >"$RUN_DIR/sweep.md" 2>&1 || true
done
echo VARIANTS2-DONE
