# Image plugin benchmark: wrap-up (Nov 2, 2026)

Goal: cold-start TTFI (time to first input) p100 < 3 s at 32 actors per node.

Setup:
- 3 × c3-standard-22 bench nodes, gVisor workers
- 96 real SWE-bench images, one image per actor
- All numbers are first-turn p50 / p95 / p99 / p100 in seconds

Per-run tables are in each run's `sweep.md` under this directory.

## 1. Recommendations per plugin flavor

### Nydus (`image-plugin/manifests/nydus-dragonfly*.yaml`)

| Flavor / option | Recommendation | Evidence |
|---|---|---|
| Seed HTTP (`nydus-dragonfly-seedhttp.yaml`) | Use. Fastest arm at every load. | p100 3.0 s at 16 per node, 5.5–5.9 s at 32 |
| One fscache domain (`domain_id`, now the manifest default) | Use | 32 per node: 10.7 → 7.2 s |
| nydusd with `nydusd-amp/bootstrap-fill.patch` | Use (benchmark-only build) | 32 per node: 6.9 → 5.5–5.9 s, 0 failed turns |
| nydusd prefetch (`NYDUS_PREFETCH`) | Off | 91 s p100 with it on, because it fetches whole blobs |
| Actors per worker | 2 | E2E 13.2 vs 16.3 s at 8 per worker |
| Snapshotter mounts (default) | Use for distinct images | p50 2.3 s |
| `--layered-mount` (parent chain once per node, or per layer) | Use only when images share base layers | Shared-base: 2.0–2.5 s p100. Distinct real images: p50 worse (3.6–4.4 s vs 2.3), tail unchanged |
| `--warm-list-dir` warm-up (`*-warm.yaml`) | Don't use | Whole-file reads compete with restore: 6.1–7.0 vs 5.9 s |
| Dragonfly P2P (`nydus-dragonfly.yaml`), Dragonfly seed peers (`*-seeds*.yaml`) | Don't use | 25–50 s p100 at 20–40 per node, failed turns at 160 per node |
| tmpfs cache (`*-tmpfs.yaml`) | Don't use | All turns failed |
| EROFS read-ahead 2 MB, larger fscache reads (`*-amp.yaml`) | No measurable gain; leave default | 10.8 vs 10.7 s |

### Riptide (`riptide.yaml`, `riptide-ms.yaml`)

| Flavor | Recommendation | Evidence |
|---|---|---|
| Riptide (gcfsd) | Fallback when images can't be converted to Nydus. Keep to 20 or fewer actors per node. | p100 2.6–6.3 s at 5–20 per node, 9.7 s at 40, 37.6 s at 80 |
| Riptide + Memorystore | Fallback at up to 24 per node. Its value is the shared cache, not speed. | p100 5.1 s at 16, 5.9 s at 24, 15.5 s at 32 (uneven placement). E2E 17–29 s, slower than Nydus, because the app's file reads are slower. Costs a Memorystore instance |

### Spegel (`spegel.yaml`, `spegel-preload.yaml`)

| Flavor | Recommendation | Evidence |
|---|---|---|
| Spegel P2P | Not for cold starts of large, distinct images | p50 37–62 s at 20–40 per node |
| Spegel with seed preload | Only for pre-warmed nodes or small images | 56–73 s p100 at 16–32 per node. Each node still downloads 5–20 GB of layers and unpacks them before start. The registry isn't the bottleneck |

Pull (no plugin) stays as the baseline: 109–189 s p100 at 20–40 per node.

## 2. Best numbers (cold, 2 actors per worker)

| Load | Real images (96 distinct SWE-bench) | Shared-base images (synthetic) |
|---|---|---|
| 16 per node | Nydus seed HTTP + one domain: 2.0 / 2.8 / 2.9 / 3.0 (`20261101-nydus-seedhttp-distinct-domain/n16`) | Nydus + one domain: 1.6 / 1.8 / 1.8 / 1.8 (`20261101-nydus-seedhttp-shared-domain/n16`) |
| 32 per node | Nydus + one domain + overlap + bootstrap fill: 2.3 / 5.0 / 5.3 / 5.5 traced (`20261102-nydus-bootfill2`), 2.3 / 5.2 / 5.8 / 5.9 untraced (`20261102-nydus-bootfill-perlayer/snap-n32`) | Nydus + layered mount: 1.8 / 1.9 / 1.9 / 2.0 (`20261101-nydus-seedhttp-shared-layered/n32`) |

Against the goal:
- Met at 32 per node only for shared-base images.
- Real images just miss it at 16 per node (3.0 s).
- Real images are about 2.5–3 s short at 32 per node.

The best other plugins at 32 per node:
- Riptide + Memorystore: 15.5 s
- Spegel with seed preload: 72.9 s

## 3. Key optimizations and their impact

In the order we found them. All p100 numbers are on real images.

1. Seed HTTP for Nydus (Oct 31).
   - Two seed nodes serve blobs, bootstraps, manifests and image resolution over plain HTTP (nginx `nydus-blob-seed`), instead of Dragonfly P2P.
   - Removes Dragonfly scheduling and its peer lookups.
   - p100 at about 16–20 per node: 24.8 → 3.7 s.
2. 2 actors per worker.
   - Gives each actor 500 millicores instead of 125.
   - Mostly shortens E2E: 16.3 → 13.2 s at 32 per node.
3. One fscache domain (`"domain_id": "ate-shared"`, commit 20e98f30).
   - All images share one cachefiles domain, so shared blobs and bootstraps are cached once per node.
   - 32 per node: 10.7 → 7.2 s. 16 per node: 4.2 → 3.0 s.
4. atelet starts image prep before the snapshot manifest fetch (commit 6f86db2f).
   - The image mount overlaps the manifest fetch.
   - 32 per node: 7.2 → 6.9 s.
5. Bootstrap fill in nydusd (`nydusd-amp/bootstrap-fill.patch`, commit 2f258cbd).
   - Cause: stock nydusd v2.3.0 passes an async block to `spawn_blocking`, so the bootstrap copy into the cache file never runs. Every metadata page is then fetched on demand: 99% of wake-up reads.
   - Fix: the patch runs the copy at explicit offsets, so a retry after a partial write (ENOBUFS) starts again at 0.
   - The first version copied by file position and corrupted one bootstrap, which failed 5 turns.
   - Bootstrap reads 236K → 1K. 32 per node: 6.9 → 5.5–5.9 s.
6. Layered mount (`--layered-mount`, commits e9fc6dbe, 6d723fa9).
   - The shared parent layers are mounted once per node.
   - Shared-base images: 3.4 → 2.0 s. No gain on distinct images.

Ruled out with data:
- nydusd prefetch
- access-ordered images
- EROFS read-ahead
- local SSD
- serving the extracted bootstrap
- tmpfs cache
- per-layer mounts on distinct images
- warm lists, both the heuristic and the memory-mapped files (`20261102-nydus-warm-mapped`)
- the gVisor 64 KB read-ahead limit. It doesn't apply here: gVisor reads image files through host file descriptors, which skip its page cache. A live sandbox held 2,543 host file descriptors to image files.

## 4. Setup

Prerequisites:
- A GKE cluster with Agent Substrate installed (`go run ./cmd/ate-setup deploy ate-system`)
- `.ate-dev-env.sh` with `KO_DOCKER_REPO`
- docker, crane, jq, and python3 with PyYAML
- `gcloud auth login` and `gcloud auth application-default login` (an `invalid_rapt` error means these are needed)
- A `bench-system` node pool for the loader (`manifests/loader-job.yaml.tmpl` selects it by name). The Nov 2 runs used 3 × e2-standard-16, untainted, with GKE workload metadata.

```sh
source .ate-dev-env.sh
go build -o bin/kubectl-ate ./cmd/kubectl-ate
B=benchmarking/image-plugin

# 1. Node pools: 3 x c3-standard-22 per arm; seed pool modes are in the pools.sh header.
$B/pools.sh create nydus
$B/deploy.sh nydus

# 2. Put the 96 images' Nydus blobs and manifests on the seeds for nginx to serve.
SAVE_DIR=/var/lib/ate/nydus-seed-blobs $B/seed-preload.sh $B/images-96-pf.jsonl 96

# 3. Build nydusd with the bootstrap fill patch; prints <repo>@sha256:...
PATCHES=bootstrap-fill.patch $B/nydusd-amp/build.sh <repo>/nydusd

# 4. Deploy the best Nydus setup.
export RUN_DIR=$B/results/<run>
$B/deploy-nydus-best.sh <repo>/nydusd@sha256:...

# 5. One cold rep: fresh bench nodes, keep the seeds.
KEEP_SEEDS=1 $B/pools.sh recreate nydus
$B/wait-ready.sh nydus
DENSITY=2 TAG=-pf LOADER_ARGS="-images /config/images-96-pf.jsonl -actors 96 -turns 5 -reads 1000" \
  $B/run.sh rep 1 nydus
python3 $B/sweep_report.py "$RUN_DIR" >"$RUN_DIR/sweep.md"
```

Other arms:
- Riptide + Memorystore: needs the Memorystore for Valkey instance `riptide-valkey` (us-central1). Run `VALKEY_EP=<ip>:6379 $B/valkey-flush.sh` before each point so it starts cold.
- Spegel with seed preload: run `$B/spegel-seed-preload.sh $B/images-320.jsonl 96` after deploying `spegel-preload.yaml`.

Tips:
- Run one driver at a time on a pool.
- A pool recreate takes 3–14 minutes.
- Launch long drivers with `setsid nohup ... &` and read their output files, rather than holding a terminal.

Teardown (costs money while idle):
- `$B/pools.sh delete <arm>` per arm. `bench-nydus-seed` goes too unless `KEEP_SEEDS=1`.
- Delete node pools by name. `bench-system` hosts the loader, and `substrate-node-pool` hosts the cluster's system pods.
- `gcloud memorystore instances delete riptide-valkey --project=kuiyue-gke-dev --location=us-central1`

## 5. What to explore next

1. Test real production images. Production is expected to be many images sharing one large base, which is closer to the shared-base set. Layered mount reached 2.0 s p100 at 32 per node there. Confirming this on real images matters most.
2. Prefetch only the ranges the guest touches. A restored guest maps the same ~8 files (the Python binary, libc, libm, 5 modules) but touches only 2–4 MB of each. Recording page ranges at checkpoint time and prefetching just those during the snapshot download could cut the remaining wake-up fetches (~1.2 MB per actor). Whole-file warm-up was too heavy.
3. Fix bootstrap fill properly. Send the `spawn_blocking` fix upstream to nydusd. Find why the first write sometimes returns ENOBUFS (0–2 per node per run; retries succeed).
4. Retest how much nydusd fetches per cache miss (`NYDUS_USER_IO_BATCH_SIZE`) on the current setup. It was only tested in the old, slower setups.
5. Placement and scheduling. The Oct 31 runs showed uneven placement, for example 12/42/42. The slowest node sets p100, so even placement, or placement that knows which images a node has cached, would cut the tail.
6. Statistics. Most rows are single runs, and repeats differ by about 0.4 s at p100. Use 3 or more reps before deciding on changes under 0.5 s.
