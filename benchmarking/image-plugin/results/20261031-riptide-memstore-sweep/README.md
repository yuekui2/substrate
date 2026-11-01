# Riptide with a Memorystore for Valkey cache, node sweep

Arm riptide-ms (image-plugin/manifests/riptide-ms.yaml): gcfsd user build
dev-c98764b from gs://kuiyue-gke-dev-riptide/gcfsd-v2-ms with
--use_memorystore --redis_cluster_address=10.1.0.6:6379
--memorystore_opt_stage=1 and its default experimental flags (cross-node
singleflight, coalesced reads, sibling prefetch, hexists guard). Valkey 9.1,
1 shard, 1 primary + 2 replicas, us-central1-a, flushed before each point.
Cold bench pool per point, one image per actor, 8 actors per worker, first
turn, seconds, p50/p95/p100. 0 failed actors. img-077 has no template on
any arm (Python 2 image).

| Actors per node | TTFI | E2E | Placement | Per-node TTFI max |
|---|---|---|---|---|
| 16 | 2.9/5.1/5.1 | 10.1/17.0/17.0 | 11/12/25 | 3.0/3.1/5.1 |
| 24 | 4.2/5.4/5.9 | 11.9/18.6/18.8 | 22/24/26 | 4.5/5.4/5.9 |
| 32 | 5.9/15.3/15.5 | 17.7/24.5/28.8 | 21/29/46 | 5.0/6.1/15.5 |

Delete the instance when done:
`gcloud memorystore instances delete riptide-valkey --project=kuiyue-gke-dev --location=us-central1`
