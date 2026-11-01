# Nydus seedhttp with 128 KiB on-demand reads, and with the cache on tmpfs

24 actors per node, 2 per worker, cold bench pool, first turn, seconds.
Baseline is the trace run (`../20261031-nydus-fstrace`): per-node TTFI max
4.4/6.7/7.0 at 19/26/27 actors, TTFI 3.3/6.5/7.0.

128 KiB reads (`nydus-dragonfly-seedhttp-amp.yaml`, NYDUS_USER_IO_BATCH_SIZE
=131072): TTFI 3.2/6.0/6.5, E2E 5.8/7.8/8.1, per-node max 4.7/4.8/6.5 at
21/22/29 actors. No change from the 1 MiB default.

fscache cache on tmpfs (`../20261031-nydus-seedhttp-tmpfs`,
`nydus-dragonfly-seedhttp-tmpfs.yaml`): all 360 turns failed. The Nydus
mount failed, and atelet then tried to unpack the Nydus layer as a tar
("archive/tar: invalid tar header"). cachefiles does not work on tmpfs.
