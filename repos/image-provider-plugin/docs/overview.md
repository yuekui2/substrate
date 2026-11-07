# Overview

This document describes how atelet and an image provider plugin work together
on a node. For atelet's side, see the Agent Substrate guide on image provider
plugins.

## Deployment

A plugin runs as a DaemonSet outside the Substrate install, with one pod per
node it serves. It listens on `/run/ate/image-provider.sock`, which atelet
mounts from the node. atelet uses the plugin while the socket exists, so a
plugin can be installed, upgraded or removed without restarting atelet. A
`nodeSelector` on the DaemonSet limits the plugin to some nodes.

atelet and the plugin upgrade independently. The API changes only by adding
fields, so either side can run a newer version than the other.

## Image preparation

For each image a workload needs, atelet calls `CanProvide` and, if the plugin
serves the image, `PrepareImage`. atelet passes the registry credentials it
resolved for the image. The plugin returns one host directory per layer, bottom
first, with the layer's tree under `fs/`. The worker pod stacks them as
read-only overlayfs lower directories, as it stacks layers from atelet's image
cache. The plugin may decline an image, and then atelet pulls it into its image
cache.

`plugin.Driver` coalesces concurrent preparations of an image, reuses a mounted
image, and remembers declined images for a while, so a declined image does not
cost a round trip to the registry on every start.

## Leases

Each `PrepareImage` takes the lease that the caller names in `lease_id`, and
each `ReleaseImage` returns one. atelet names a lease after the actor, the
container and, for an image volume, the volume, so a retried `PrepareImage`
returns the leased image without taking a second lease. A lease ID reused for
another image fails with `ALREADY_EXISTS`. `ReleaseImage` names the lease's
image digest and form, and fails with `INVALID_ARGUMENT` if they don't match
the lease. Releasing a lease the plugin does not hold succeeds.

atelet releases an actor's leases when the actor checkpoints or terminates.
When the last lease of an image goes, the driver keeps the image mounted for an
idle time, then unmounts it.

The driver writes a `lease.json` beside each image's layer directories. It
keeps lease IDs only in memory. When atelet or the plugin restarts, atelet
sends `ReconcileImages` with the leases its actors still hold. The driver
takes over their images from their `lease.json`, unmounts the rest, and
removes directories that belong to no image.

## Draining

When the plugin pod receives SIGTERM, `ServeUnixSocket` drains the driver:

1. `CanProvide` returns false with `PLUGIN_STATE_DRAINING`, and `PrepareImage`
   declines every image, so atelet uses its image cache for new and resuming
   actors.
2. Idle images are unmounted at once.
3. atelet asks the control plane to suspend each running actor on the node that
   holds a lease. Each suspend checkpoints the actor and releases its leases.
4. When the last lease is released, or the drain timeout passes, the plugin
   stops serving and writes `/run/ate/drain-complete`.

A FUSE-backed provider must not exit while workloads read its mounts. Its
container can run a `preStop` hook that waits for the drain-complete file, with
a termination grace period longer than the drain timeout.

## On-disk layout

```
/var/lib/ate/image-plugin-cache/<provider>/   the driver's work dir
├── version                                   layout version
└── <image key>/                              one per image ref
    ├── lease.json                            state for ReconcileImages
    ├── layer-0/
    │   ├── fs -> <snapshotter mount>         the layer's tree
    │   └── finalized
    └── layer-1/
        └── ...
```

A plugin that finds a different layout version in its work dir removes the
image directories in it at startup.
