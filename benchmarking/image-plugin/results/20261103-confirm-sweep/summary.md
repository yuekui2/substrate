# Image plugin benchmark: summary (Nov 3, 2026)

Target: cold-start TTFI p95 < 3 s at 32 actors per node. This replaces the
earlier p100 target, because single outliers dominated p100.

The data is from the Nov 3 confirmation sweep in this directory. Setup and
per-run tables are in `README.md`, `set-*/n*/sweep.md` and
`set-*/n*/rep-1/*/bottleneck.md`.

## 1. Summary

- Nydus is the fastest arm at every load and on every image set, for both
  TTFI and the first task's end-to-end time.
- Nydus meets p95 < 3 s at 16 actors per node on all 3 sets (1.7 to 1.9 s).
  At 32 per node it is 4.5 to 5.3 s, about 1.5 to 2.3 s short.
- Riptide + Memorystore is second.
  - Its TTFI p95 is 1.1 to 3.2 times Nydus's.
  - Its first task's p95 end to end is 1.8 to 2.4 times Nydus's, because
    the app's first file reads are slower.
  - It uses 1.9 to 2.3 times the CPU per cold start, but does almost no
    disk reads.
- Spegel with seed preload and pull have 13 to 42 times Nydus's TTFI p95,
  because each node downloads and unpacks 1.6 to 2.4 GB per image.
  Once the image is on the node, their later turns match Nydus.
- At 32 per node, one phase decides the Nydus p95: `wakeup_probe`, the
  restored guest's first file reads through nydusd.
  - Replaying each run with every actor's wake-up at that run's median gives
    p95 2.35 to 2.49 s, under the target.
  - The same replay for the mount, snapshot download or manifest fetch
    lowers p95 by at most 0.3 s.
- Wake-ups slow down while the node is short of CPU:
  - CPU pressure peaks at 50 to 70 %.
  - Busy CPU plateaus at about 80 % of 22 cores.
  - The Nydus plugin process (`nydus-dragonfly`) uses about 1 core-second
    per cold start, 40 % of all node CPU in the window.

## 2. Setup

- 3 bench nodes of c3-standard-22, gVisor workers of 1 vCPU, 2 actors per
  worker. 32 actors per node is 1.45 actors per vCPU.
- Every point starts on fresh nodes, with one distinct image per actor and
  5 turns of 1,000 file reads each.
- There are 3 disjoint sets of 96 SWE-bench images (A, B, C), one per rep.
  Set A is the wrap-up's image set. Each set is one run per point.
- Arms:
  - nydus: seed HTTP, one fscache domain, bootstrap fill, prefetch off
  - riptide-ms: Riptide with Memorystore for Valkey (7.2), flushed before
    each point
  - spegel: Spegel with 2 seed nodes, preloaded
  - pull: no plugin
- Nydus ran alone at each point. The other three ran together afterwards
  on their own pools.
- No turn failed. img-077 in set A has no template on any arm, so set A at
  32 per node has 95 working actors on every arm.

## 3. Latency

All values are first-turn seconds, sets A / B / C, then the mean of the 3
sets.

### TTFI p95

| Arm | 16 per node | 24 per node | 32 per node |
|---|---|---|---|
| nydus | 1.7 / 1.7 / 1.9 (**1.8**) | 3.2 / 5.2 / 5.2 (**4.5**) | 4.5 / 5.3 / 4.8 (**4.9**) |
| riptide-ms | 3.4 / 3.4 / 4.0 (**3.6**) | 4.6 / 5.6 / 6.3 (**5.5**) | 7.6 / 6.9 / 15.3 (**9.9**) |
| spegel | 42.1 / 31.2 / 49.4 (**40.9**) | 50.8 / 41.4 / 65.6 (**52.6**) | 77.0 / 47.8 / 61.3 (**62.0**) |
| pull | 79.0 / 61.9 / 86.6 (**75.8**) | 102.0 / 86.9 / 99.8 (**96.2**) | 138.2 / 114.0 / 125.9 (**126.0**) |

### TTFI p50 and p100

| Arm | Load | p50 | p100 |
|---|---|---|---|
| nydus | 16 | 1.3 / 1.4 / 1.4 | 1.9 / 2.1 / 2.4 |
| nydus | 24 | 2.0 / 2.2 / 2.0 | 5.3 / 6.6 / 6.0 |
| nydus | 32 | 2.2 / 2.2 / 2.3 | 4.9 / 6.2 / 5.6 |
| riptide-ms | 16 | 2.2 / 2.7 / 3.1 | 3.8 / 3.5 / 4.1 |
| riptide-ms | 24 | 3.4 / 3.8 / 4.5 | 5.2 / 5.7 / 6.5 |
| riptide-ms | 32 | 4.2 / 5.2 / 6.4 | 13.7 / 8.2 / 15.5 |
| spegel | 32 | 34.2 / 32.4 / 33.4 | 82.2 / 61.9 / 72.1 |
| pull | 32 | 75.3 / 86.5 / 71.8 | 149.4 / 141.1 / 137.7 |

### First task end to end: resume request to work done

| Arm | 16 per node p95 | 24 per node p95 | 32 per node p95 | 32 per node p50 |
|---|---|---|---|---|
| nydus | 4.5 / 4.7 / 4.6 (**4.6**) | 6.7 / 7.8 / 7.1 (**7.2**) | 7.1 / 7.6 / 7.2 (**7.3**) | 6.2 / 6.7 / 6.2 |
| riptide-ms | 10.1 / 11.1 / 9.6 (**10.3**) | 12.1 / 13.1 / 13.4 (**12.9**) | 15.4 / 15.1 / 22.2 (**17.6**) | 12.6 / 13.2 / 17.7 |
| spegel | 42.3 / 31.5 / 49.8 (**41.2**) | 51.0 / 41.7 / 65.7 (**52.8**) | 77.2 / 48.1 / 61.3 (**62.2**) | 34.8 / 33.9 / 34.1 |
| pull | 79.3 / 62.1 / 86.8 (**76.1**) | 102.3 / 87.2 / 99.9 (**96.5**) | 138.5 / 114.4 / 125.9 (**126.3**) | 76.7 / 87.5 / 72.8 |

The first task's work, 1,000 file reads after TTFI, has a p50 of:
- nydus: 1.8 to 4.1 s
- riptide-ms: 4.5 to 8.6 s
- spegel and pull: 0.4 to 1.2 s, because their images are already local

### Later turns (resume from the actor's own snapshot)

All arms are about equal. TTFI p95 is 0.16 to 0.24 s. End-to-end p95 is
1.2 to 1.5 s, except riptide-ms at 2.0 to 2.7 s.

## 4. CPU and disk per arm

The window runs from the first actor's create to the last first-turn TTFI,
plus 1.5 s on each side. Values are per node, averaged over 9 node-runs (3
nodes × 3 sets). Per cold start divides by the actors on the node. CPU
counts all host processes. A process that exits inside the window is
missed.

| Arm, load | Window | Busy cores, mean / peak % | Max CPU pressure | CPU core-s per cold start | Disk write per cold start | Disk read per cold start | Peak write MB/s | Peak IOPS | Network rx per cold start |
|---|---|---|---|---|---|---|---|---|---|
| nydus, 16 | 4 s | 8.0 / 77 % | 47 % | **2.0** | 64 MB | 19 MB | 628 | 29.6k | 13 MB |
| nydus, 32 | 8 s | 10.5 / 81 % | 70 % | **2.5** | 69 MB | 23 MB | 761 | 36.6k | 14 MB |
| riptide-ms, 16 | 6 s | 10.4 / 82 % | 39 % | **3.8** | 73 MB | 0 | 524 | 4.5k | 19 MB |
| riptide-ms, 32 | 14 s | 12.9 / 85 % | 47 % | **5.8** | 61 MB | 0 | 677 | 4.5k | 20 MB |
| spegel, 16 | 50 s | 6.1 / 81 % | 30 % | 19.2 | 2,359 MB | 1 MB | 1,951 | 21.1k | 616 MB |
| spegel, 32 | 74 s | 8.3 / 86 % | 48 % | 19.1 | 2,136 MB | 4 MB | 1,938 | 27.0k | 553 MB |
| pull, 16 | 92 s | 4.2 / 64 % | 31 % | 24.4 | 1,876 MB | 1 MB | 1,889 | 14.1k | 647 MB |
| pull, 32 | 145 s | 5.4 / 74 % | 45 % | 24.6 | 1,608 MB | 1 MB | 1,919 | 22.0k | 578 MB |

Largest CPU users, in core-seconds per cold start:

| Arm, load | Top processes |
|---|---|
| nydus, 32 | nydus-dragonfly 0.98, nydusd 0.41, ateworker-gvisor 0.09, objectstore-plugin 0.08 |
| riptide-ms, 32 | gcfsd-v2 3.46, riptide plugin 1.05, ateworker-gvisor 0.08, objectstore-plugin 0.08 |
| spegel, 32 | spegel-preload 15.45, kworker 0.73 |
| pull, 32 | atelet 19.85 (pull and unpack), kworker 0.61 |

Riptide compared with Nydus:
- CPU:
  - Riptide uses 1.9 times the CPU per cold start at 16 per node (3.8
    against 2.0 core-s) and 2.3 times at 32 (5.8 against 2.5). gcfsd-v2
    alone uses 3.5 core-s per cold start at 32 per node.
  - The Riptide window also ends at TTFI. gcfsd keeps serving the app's
    first file reads after that, which this table doesn't count.
- Disk:
  - Riptide reads nothing from disk, and its peak IOPS is 4.5k.
  - Nydus reads 19 to 23 MB per cold start from its fscache files and peaks
    at 30k to 37k IOPS.
  - Both write about 60 to 75 MB per cold start. Nydus fills its cache, and
    Riptide writes its local layer cache.
- Network: Nydus fetches 13 to 14 MB per cold start from its seeds, and
  Riptide 19 to 20 MB from Memorystore and the registry.
- Pressure: Riptide's nodes show less CPU and IO pressure (39 to 47 % and
  20 to 33 %) than Nydus's (47 to 70 % and 46 to 55 %). But Riptide spreads
  more CPU over a window up to twice as long, and its TTFI is still higher.

Spegel and pull write 1.6 to 2.4 GB and receive about 0.6 GB per cold start,
and unpacking costs 19 to 25 core-s. Their cost grows with image size, not
with the number of files an actor touches.

## 5. What p95 < 3 s at 32 per node takes

### Where Nydus stands

| Load | p95, sets A / B / C | Per-node p95 range | Met? |
|---|---|---|---|
| 16 | 1.7 / 1.7 / 1.9 | 1.4 to 2.1 | Yes, 1.1 to 1.3 s to spare |
| 24 | 3.2 / 5.2 / 5.2 | 1.0 to 6.2 | No |
| 32 | 4.5 / 5.3 / 4.8 | 3.2 to 6.0 | No, 1.5 to 2.3 s short |

At 24 per node, placement is the problem, not load:
- Placement was 20/21/31, 8/31/33 and 11/27/34 actors per node.
- Nodes with 31 to 34 actors had p95 4.1 to 6.2 s.
- Nodes with 27 or fewer had p95 1.0 to 3.2 s.
- An even 24 per node would likely be close to the target.

At 32 per node, worker capacity forces 32/32/32, so the gap there is
per-node load.

### Which phase decides p95 at 32 per node

Mean phase times of the actors at or above p95, against all actors:

| Phase | p95+ actors, sets A / B / C | All actors |
|---|---|---|
| manifest_fetch + download | 0.6 / 0.6 / 0.4 | 0.5 to 0.7 |
| oci_unpack (Nydus mount) | 1.2 / 1.1 / 1.0 | 0.8 to 0.9 |
| wakeup_probe (guest's first file reads) | **2.6 / 3.9 / 3.3** | 0.7 to 0.9 |

Replay of p95 with one phase at its run's median for every actor:

| Phase at median | p95, sets A / B / C |
|---|---|
| None (measured) | 4.54 / 5.32 / 4.82 |
| wakeup_probe | **2.35 / 2.43 / 2.49** |
| oci_unpack | 4.27 / 5.17 / 4.64 |
| download | 4.49 / 5.32 / 4.81 |
| manifest_fetch | 4.48 / 5.32 / 4.82 |

So p95 < 3 s at 32 per node needs the wake-up tail cut by about 2 s: from
2.6 to 3.9 s down to about 1 s. The other phases add up to 0.3 s together.
They don't move p95 enough alone.

### Why wake-ups are slow under load

- The guest's first reads miss the node cache, and nydusd fetches them from
  the seeds. That is about 14 MB per cold start. Seed-side p99 is 110 to
  200 ms, which is small next to a 2.6 to 3.9 s wake-up.
- These reads run while the node is short of CPU:
  - CPU pressure peaks at 50 to 70 % and busy CPU at about 80 %.
  - The busiest core reaches 90 to 100 %.
- nydus-dragonfly (about 1 core-s per cold start, 4 to 6 cores per node at
  peak) and nydusd (0.4 core-s) compete with the guests.
- Disk is not the limit. IO pressure reaches 45 to 55 %, but the mean await
  is 0.7 to 5.5 ms.

### Options, by expected effect on p95

1. Make wake-up reads hit the local cache.
   - The prefetch table already lists the files a cold guest reads. Warm only
     those ranges while the image mounts, and not whole blobs: prefetch of
     whole blobs gave 91 s p100 before (wrap-up).
   - This targets the 2 s gap directly, and also shortens the first task's
     work (p50 3 to 4 s at 32 per node).
   - The meeting notes say Nydus with prefetch matches Riptide. In this
     sweep Nydus with prefetch off is already ahead of Riptide at p95 on
     every point.
2. Cut the Nydus plugin's CPU.
   - nydus-dragonfly uses 40 % of all node CPU in the cold-start window.
     Profile it (pprof) during a cold start.
   - Halving it would free 2 to 3 cores per node at peak, lowering the CPU
     pressure the wake-ups wait on.
   - Also find why busy CPU stops at about 80 %, for example worker CPU
     limits.
3. Use shared base layers.
   - On the synthetic shared-base images, layered mount reached
     1.8 / 1.9 / 1.9 / 2.0 s (p50 / p95 / p99 / p100) at 32 per node
     (`../20261101-nydus-seedhttp-shared-layered`).
   - Production images are expected to share a large base, so the target may
     already hold for them. That needs a run on real production images.
4. Spread actors evenly across nodes. That fixes 24 per node, and any load
   below full capacity.
5. Shorten the Nydus mount (about 1.1 s in p95+ actors). That is worth
   about 0.2 to 0.3 s of p95.

### Density

- The meeting set the test density at 1.5 to 2 sandboxes per CPU.
- On c3-standard-22 that is 33 to 44 actors per node. This sweep's 32 per
  node is 1.45.
- p95 rose from 1.8 s at 16 per node to 4.9 s at 32 (set means). With the
  current setup, 33 to 44 per node would be further from 3 s.
- Options 1 and 2 above are needed before that range is worth testing.

## 6. Caveats

- Each set is one run per point. Sets differ by up to 1.3 s at p95, and
  repeats of one set by about 0.4 s. Read differences under 0.5 s as noise.
- One riptide-ms run (set C, 32 per node) had p95 15.3 s against 6.9 to
  7.6 s on the other sets. It may be uneven cache state and needs a repeat.
- The replays assume the other phases stay as measured. A faster wake-up
  would also lower CPU pressure, so the real gain could be larger.
- CPU per process comes from `/proc` and misses processes that exit inside
  the window, such as unpack helpers.
- The Memorystore instance ran Valkey 7.2. The Oct 31 Riptide run used 9.1.
