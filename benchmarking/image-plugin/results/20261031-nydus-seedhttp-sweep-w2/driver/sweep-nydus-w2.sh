#!/usr/bin/env bash
# Nydus seedhttp node sweep at 16, 24 and 32 actors per node with 2 actors
# per worker (500 millicores each), after the DENSITY=8 n32 rerun.
set -uo pipefail
cd /usr/local/google/home/kuiyue/src/substrate || exit 1
source .ate-dev-env.sh
while pgrep -f scratch/sweep-nydus3-n32.sh >/dev/null; do sleep 30; done
B=benchmarking/image-plugin
base=$B/results/20261031-nydus-seedhttp-sweep-w2
mkdir -p "$base"
RUN_DIR=$base KEEP_SEEDS=1 ARMS=nydus DENSITY=2 $B/node-sweep.sh 16 24 32
for n in 16 24 32; do
  python3 $B/sweep_report.py "$base/n$n" >"$base/n$n/sweep.md" 2>&1 || true
done
echo NYDUS-SWEEP-W2-DONE
