# Spegel seed-preload node sweep

Spegel with 2 dedicated seed nodes (`bench-spegel-seed`, c3-standard-8) as
the only blob owners (`spegel-preload` with `--spegel-owner-dns`). Before
the sweep, `spegel-seed-preload.sh images-320.jsonl 96` stored the 199
unique layer blobs of the first 96 images in the seeds. Same config as the
Nydus seedhttp sweep: cold bench pool per point, one image per actor, 8
actors per 1-vCPU worker, 5 turns, `-reads 1000`. First turn, seconds,
p50/p95/p100. TTFI max per node is in placement order. Per-point tables are
in `n*/sweep.md`.

| Actors per node | TTFI | E2E | Placement (actors per node) | TTFI max per node |
|---|---|---|---|---|
| 16 | 20.6/52.0/56.0 | 23.2/54.0/56.2 | 5/20/23 | 35.1/30.3/56.0 |
| 24 | 27.2/59.9/63.2 | 31.7/60.9/63.5 | 22/24/26 | 55.0/40.4/63.2 |
| 32 | 35.9/65.8/72.9 | 40.7/66.3/74.8 | 29/32/35 | 72.9/67.0/65.8 |

No point meets TTFI p100 < 3 s. No errors. Bench nodes read every blob from
a seed: `upstream_bytes` is 0 and `peer_misses` is 0 on all of them
(`node-snapshots.jsonl`). Each bench node still downloads 5 to 20 GB of
compressed layers and unpacks them before the first actor on an image starts.
oci_unpack, which covers the blob download and the unpack, is the whole
TTFI (p50 19.8, 26.5 and 35.1 s). worker_restore stays under 0.7 s.

The plugin logs (`n24/plugin-logs`, `n32/plugin-logs`) show the largest
layers (1.2 to 1.5 GB) finishing 40 to 70 s after the first request. A node
receives about 250 MB/s in total. The seeds together serve about 800 MB/s at
32 actors per node, below what their NICs allow.

Setup is in `setup/`. The first preload run stored 119 of 199 blobs because
the node-sampler pod on one seed restarted. `preload2.log` is the rerun that
stored all 199. It finished at 21:59:50 UTC, 5 s after the n16 run step
began and before the loader pod started at 22:00:04.
