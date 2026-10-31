#!/usr/bin/env bash
# Nydus seedhttp node sweep at 16, 24 and 32 actors per node (cold bench pool
# per point, one image per actor, 8 actors per worker). Seeds already hold
# the first 192 images.
set -uo pipefail
cd /usr/local/google/home/kuiyue/src/substrate || exit 1
source .ate-dev-env.sh
B=benchmarking/image-plugin
base=$B/results/20261031-nydus-seedhttp-sweep2
mkdir -p "$base"
RUN_DIR=$base KEEP_SEEDS=1 ARMS=nydus DENSITY=8 $B/node-sweep.sh 32
for n in 16 24 32; do
  python3 $B/sweep_report.py "$base/n$n" >"$base/n$n/sweep.md" 2>&1 || true
done
echo NYDUS-SWEEP2-N32-DONE
