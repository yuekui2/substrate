# Confirmation sweep: 16, 24 and 32 actors per node, 3 image sets (Nov 3, 2026)

Each arm uses the recommended flavor from `../wrap-up.md`, and pull is the
baseline. Every point starts on a cold pool: 3 × c3-standard-22, 2 actors per
worker, 5 turns, `-reads 1000`, one distinct image per actor.

There are 3 disjoint image sets of 96 from `images-320.jsonl`, one per rep:

| Set | Lines | Image list | TAG |
|---|---|---|---|
| A | 1–96 | `images-96-pf.jsonl` | `-pf` |
| B | 97–192 | `images-96-pf-b.jsonl` | `-pfb` |
| C | 193–288 | `images-96-pf-c.jsonl` | `-pfc` |

Nydus ran alone at every point. Pull, riptide-ms and spegel then ran together
on fresh pools (`node-sweep.sh` with `SOLO=nydus`). That keeps Nydus clear of
the others' load and keeps the cluster under the C3 quota.

Arms:
- nydus: seed HTTP, one fscache domain and bootstrap fill (`deploy-nydus-best.sh`)
- riptide-ms: Riptide with Memorystore for Valkey, flushed before each point
- spegel: Spegel with seed preload
- pull: no plugin

In set A, img-077 (psf__requests-1724) failed to bake on every arm, as it did
in earlier runs. Every arm was baked and run on the same images. No turn
failed on any arm.

Each point's tables are in `set-*/n*/sweep.md`. The per-arm breakdowns are in
`set-*/n*/rep-1/*/bottleneck.md`. Setup, the smoke test and the drivers are
in `setup/`, `smoke/` and `driver/`.

## First-turn TTFI p100 (s), sets A / B / C

| Arm | 16 per node | 24 per node | 32 per node |
|---|---|---|---|
| nydus | 1.9 / 2.1 / 2.4 | 5.3 / 6.6 / 6.0 | 4.9 / 6.2 / 5.6 |
| riptide-ms | 3.8 / 3.5 / 4.1 | 5.2 / 5.7 / 6.5 | 13.7 / 8.2 / 15.5 |
| spegel | 46.8 / 46.1 / 51.0 | 67.8 / 55.6 / 74.5 | 82.2 / 61.9 / 72.1 |
| pull | 89.0 / 84.6 / 95.9 | 118.0 / 115.1 / 118.2 | 149.4 / 141.1 / 137.7 |

TTFI p50 (s), sets A / B / C:

| Arm | 16 per node | 24 per node | 32 per node |
|---|---|---|---|
| nydus | 1.3 / 1.4 / 1.4 | 2.0 / 2.2 / 2.0 | 2.2 / 2.2 / 2.3 |
| riptide-ms | 2.2 / 2.7 / 3.1 | 3.4 / 3.8 / 4.5 | 4.2 / 5.2 / 6.4 |
| spegel | 19.4 / 20.2 / 25.1 | 26.8 / 25.8 / 31.9 | 34.2 / 32.4 / 33.4 |
| pull | 45.4 / 51.8 / 53.6 | 64.8 / 59.1 / 62.6 | 75.3 / 86.5 / 71.8 |

## First-turn E2E p100 (s): resume request to work done, sets A / B / C

| Arm | 16 per node | 24 per node | 32 per node |
|---|---|---|---|
| nydus | 4.8 / 4.9 / 4.9 | 7.0 / 9.2 / 8.0 | 7.7 / 8.5 / 8.1 |
| riptide-ms | 10.3 / 11.2 / 9.7 | 12.1 / 13.4 / 13.7 | 16.4 / 16.3 / 22.4 |
| spegel | 46.9 / 46.5 / 51.1 | 67.9 / 56.0 / 74.6 | 82.4 / 62.3 / 72.6 |
| pull | 89.3 / 84.9 / 96.2 | 118.0 / 115.4 / 118.6 | 149.7 / 141.5 / 138.1 |

Against the wrap-up:
- The order holds on every set: nydus, riptide-ms, spegel, pull.
- Nydus meets TTFI p100 < 3 s at 16 per node on all 3 sets: 1.9 to 2.4 s.
  The wrap-up had 3.0 s.
- Nydus at 32 per node is 4.9 to 6.2 s. The wrap-up had 5.5 to 5.9 s on set
  A. Sets differ by up to 1.3 s, more than the 0.4 s between reps of one set.
- riptide-ms has a TTFI tail close to nydus up to 24 per node, but its
  first-turn work is 6 to 9 s, against 2 to 4 s for nydus. So its first-turn
  E2E is 1.5 to 2.5 times nydus's.
- Spegel and pull are 10 to 70 times slower on the first turn. Their later
  turns match nydus.

## Where the Nydus tail goes

Placement at 24 per node was uneven on every set. One node got 31 to 34
actors and set the p100, while another got 8 to 21:

| Set | Actors per node | TTFI max per node (s) |
|---|---|---|
| A | 20 / 31 / 21 | 2.5 / 5.3 / 2.6 |
| B | 33 / 31 / 8 | 6.6 / 5.2 / 1.0 |
| C | 34 / 27 / 11 | 6.0 / 3.6 / 1.4 |

At 32 per node, worker capacity (16 workers × 2 actors) forces 32 / 32 / 32.
That is why 24 per node can end up slower than 32.

In the slowest actors of each 32 per node run:
- `wakeup_probe` is 2.8 to 4.1 s. That is the restored guest's first file
  reads through nydusd.
- `oci_unpack`, the Nydus mount, is 1.1 to 1.3 s.
- `manifest_fetch` plus `download` is 0.5 to 0.9 s.

Per node, across the cold-start window of a fully loaded node (1 s samples):
- CPU busy peaks at 79 to 81 % of 22 cores on every loaded node, with the
  busiest core at 90 to 100 %.
- CPU pressure (PSI some) peaks at 50 to 70 %.
- The plugin process `nydus-dragonfly` uses 4.0 to 5.8 cores, averaged over
  the window. nydusd uses 1.7 to 2.1 cores. Every other process stays under
  0.7 cores.
- IO pressure peaks at 43 to 55 %. Writes reach 450 to 700 MB/s and IOPS 30k
  to 36k, but the mean await stays at 0.7 to 5.5 ms.
- On the Nydus seeds, request time is 0 ms at p50, 110 to 200 ms at p99 and
  at most 0.6 s. A 32 per node run fetches about 1 GB in 3,300 to 4,000
  requests.

The nodes run out of CPU while actors wake up, and the Nydus plugin process
is the largest single user. Seed fetches and disk latency are smaller costs.
Busy CPU plateaus at about 80 % on every loaded node. That may be a cap, such
as worker CPU limits, rather than plain saturation. It should be checked.

Next steps for p100 < 3 s at 32 per node:
1. Profile `nydus-dragonfly` during a cold start (pprof), and find what uses
   4 to 6 cores per node.
2. Check what holds busy CPU at about 80 %: cgroup limits on workers and
   system pods.
3. Spread actors evenly across nodes. At 24 per node, the overloaded node
   alone costs 2 to 4 s.
4. Look at the 1.1 to 1.3 s Nydus mount, which is the second-largest
   phase.
