---
title: "Image Provider Plugin"
linkTitle: "Image Provider Plugin"
weight: 12
docsite: false
description: >
  How atelet prepares container images through a node's image provider plugin.
---

# Image Provider Plugin

By default atelet pulls each container and image volume image into the
node-local image cache and unpacks its layers there. A node can also run an
image provider plugin, a DaemonSet that prepares image layers itself, for
example by loading them lazily from a remote store. The plugin API, a Go client
and a library for writing plugins live in
[ai-on-gke/image-provider-plugin](https://github.com/ai-on-gke/image-provider-plugin).

## Configuration

atelet talks to the plugin over the Unix socket that `--image-provider-socket`
names, `/run/ate/image-provider.sock` by default. The atelet DaemonSet mounts
the node's `/run/ate` for it. An empty flag value turns the plugin off.

atelet uses the plugin only while its socket exists, so you can install and
remove a plugin without restarting atelet.

The plugin returns host directories, one per layer, each with the layer's tree
under `fs/`. The worker pods compose the actor's rootfs from them, and worker
pods see only `/var/lib/ate`, so the plugin must put them there.

## Preparing an image

For each image a Run or Restore needs, atelet:

1. Serves a digest ref straight from the image cache when the cache holds the
   whole image.
2. Otherwise asks the plugin with `CanProvide`, and if it serves the image,
   calls `PrepareImage`. The plugin gets the registry credentials atelet's image
   credential providers resolve for the image.
3. Pulls into the image cache when the plugin declines the image or fails.

atelet records each request on `ate.imageprovider.requests` and
`ate.imageprovider.duration`, by provider and outcome: `success`, `fallback`
when the image cache served the image, or `error` when both failed. See
[Observability](observability.md).

## Leases

The plugin holds a lease for each use of an image it prepared, and keeps the
image's layers while any lease is left. atelet names each lease
`<actor UID>/<container>` for a container image, and
`<actor UID>/<container>/<volume>` for an image volume. The overlay spec of
each container bundle (`rootfs-overlay.json`) records the image ref, digest and
lease ID, for the rootfs and for each image volume.

atelet releases an actor's leases when the actor checkpoints or terminates, and
when a Run or Restore fails. Each time the plugin's socket appears, at atelet
startup or after the plugin restarts, atelet sends `ReconcileImages` with the
leases the overlay specs on the node record. The plugin drops every other lease.

## Draining the plugin

A plugin that is stopping reports `PLUGIN_STATE_DRAINING` and serves no new
images, so new activations use the image cache. Running actors keep their
leases, so while the plugin drains, atelet asks the control plane to suspend
each running actor on the node that holds one. The suspend checkpoints the
actor and releases its leases. The actor resumes later from its snapshot, from
the image cache or from the plugin once it is serving again.
