# fscache on-demand read traces, Nydus seedhttp at 24 per node

bpftrace on cachefiles_ondemand_read (nydusd picks up a request) and
cachefiles_ondemand_cread (nydusd replies), 2 actors per worker. The first
trace (`../20261031-nydus-fstrace`) counted pickups only; its reply match
failed.

| Node | Actors | Reads | Peak reads/s | TTFI max |
|---|---|---|---|---|
| 2mdc (trace 1) | 19 | 42,873 | 11,934 | 4.4 |
| 6x65 (trace 1) | 26 | 64,551 | 14,826 | 6.7 |
| 9c2c (trace 1) | 27 | 66,513 | 15,282 | 7.0 |

Each actor causes about 2,400 reads. In trace 2, 93-117 nydusd threads
pick up reads each busy second and replies match pickups per second, so
nydusd does not serialize reads. During the burst, node disk writes peak
at 352 MB/s and 25k IOPS, and iowait at 28.6%.
