#!/usr/bin/env bash
# Node sweep (1, 40 and 64 actors per node, one image per actor, cold bench pool
# per point) for the best Nydus variant: seedhttp with image resolve,
# manifests, bootstraps and blobs from the seeds. 8 actors per worker, as in
# the earlier sweep. Seeds keep their data across points.
set -uo pipefail
cd /usr/local/google/home/kuiyue/src/substrate || exit 1
source .ate-dev-env.sh
B=benchmarking/image-plugin
base=$B/results/20261031-nydus-seedhttp-sweep
mkdir -p "$base"
export RUN_DIR=$base/setup
mkdir -p "$RUN_DIR"
$B/ev.sh seed-preload "SAVE_DIR=/var/lib/ate/nydus-seed-blobs $B/seed-preload.sh $B/images-320.jsonl 192" || exit 1
RUN_DIR=$base KEEP_SEEDS=1 ARMS=nydus DENSITY=8 $B/node-sweep.sh 1 40 64
python3 $B/sweep_report.py "$base" >"$base/sweep.md" 2>&1 || true
echo SWEEP-DONE
