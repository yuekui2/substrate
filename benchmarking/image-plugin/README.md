# Image plugin benchmark

Runs SWE-bench actors through resume, probe, work, suspend and think turns
on one node pool per arm (`pull`, `riptide`, `nydus`, `spegel`), and records
TTFI including the time an actor waited for a slot.

## Steps

```sh
source .ate-dev-env.sh
go build -o bin/kubectl-ate ./cmd/kubectl-ate
export RUN_DIR=benchmarking/image-plugin/results/<run>
benchmarking/image-plugin/pools.sh create        # 3 x c3-standard-22 per arm, Hyperdisk Balanced
benchmarking/image-plugin/deploy.sh              # loader RBAC, WorkerPools, plugins, node-sampler
python3 benchmarking/image-plugin/image_sizes.py benchmarking/image-plugin/images-320.jsonl "${RUN_DIR}/images"

# One arm at a time, fresh pool per arm and rep:
LOADER_ARGS="-actors 480 -turns 10" benchmarking/image-plugin/bench.sh 5

# Or sweep actors per worker, all arms at once:
LOADER_ARGS="-actors 480 -turns 5 -reads 1000" benchmarking/image-plugin/sweep.sh 1 2 4 8
python3 benchmarking/image-plugin/sweep_report.py "${RUN_DIR}" >"${RUN_DIR}/sweep.md"
```

Every step runs through `ev.sh`, which appends the command to
`${RUN_DIR}/transcript.md` and its output to `${RUN_DIR}/logs/`.

## Per run directory

- `turns.jsonl`: one line per actor turn, from the loader.
- `atelet-<node>.log`: atelet restore timing for the run window.
- `diskstats.jsonl`: `/proc/diskstats` counters every 5 s per node.
- `node-snapshots.jsonl`: image cache sizes and Spegel `/stats` after the run.
- `monitoring.jsonl`: Cloud Monitoring disk and network rates per VM.

Large files are committed gzipped. The report scripts read either form.
