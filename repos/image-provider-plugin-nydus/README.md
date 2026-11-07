# image-provider-plugin-nydus

An [image provider plugin](https://github.com/ai-on-gke/image-provider-plugin)
for [Agent Substrate](https://github.com/ai-on-gke/substrate) that serves
[Nydus](https://nydus.dev) RAFS v6 images lazily. Actors start before their
image is on the node, and nodes share image data over
[Dragonfly](https://d7y.io).

## How it works

Each node runs the `image-provider-nydus` DaemonSet:

- `nydus-snapshotter` runs containerd-nydus-grpc with one shared nydusd in
  EROFS over fscache mode. The kernel reads file data on demand, and nydusd
  fetches the missing chunks.
- `dfdaemon` is the node's Dragonfly peer. Its registry mirror proxy fetches
  each blob range from peers that have it, and from the registry only when no
  peer does.
- `image-provider`, the plugin, listens on `/run/ate/image-provider.sock`.
  For each image, it registers the data blob layers with the snapshotter,
  writes the bootstrap layer, mounts it as EROFS, and returns the mount to
  atelet as one layer directory.

Nodes labeled `ate.dev/dragonfly-seed=true` run `image-provider-nydus-seed`
instead, where dfdaemon is a super seed peer, and `nydus-blob-seed`, an nginx
that serves the blobs and manifests preloaded on the node.

nydusd reads a blob range from these sources in order:

1. The seed HTTP servers, through the `nydus-blob-seed` Service.
2. The node's dfdaemon, which gets the pieces from peers and seeds.
3. The registry.

The plugin reads manifests and bootstrap layers from the seed HTTP servers
first, then through dfdaemon. It checks the digest of everything it reads
from the seeds.

The plugin declines images that are not Nydus RAFS v6, and atelet pulls
them into its image cache instead.

## Prerequisites

- A kernel with EROFS and on-demand fscache (`CONFIG_EROFS_FS_ONDEMAND` and
  `CONFIG_CACHEFILES_ONDEMAND`). On Ubuntu kernels without them, such as GKE
  Ubuntu node images, the `build-erofs-modules` init container builds both
  modules on each node's first start. That needs egress to the Ubuntu archive
  and launchpad.net, and secure boot off so the kernel loads the unsigned
  modules. The node DaemonSet installs binaries and modules under
  `/home/kubernetes`.
- One or more seed nodes, labeled with
  `kubectl label node NODE ate.dev/dragonfly-seed=true`.
- Agent Substrate with atelet's image provider socket, `/run/ate/image-provider.sock`
  by default. See the
  [image provider guide](https://github.com/ai-on-gke/substrate/blob/main/docs/guides/image-provider.md).
- Images converted to Nydus RAFS v6, for example with
  `nydusify convert --fs-version 6`.
- [kustomize](https://kustomize.io) and [ko](https://ko.build), and
  `KO_DOCKER_REPO` set to a registry the nodes can pull from.

## Deploy

Set `REGISTRY_URL` in [manifests/kustomization.yaml](manifests/kustomization.yaml)
to the registry of your Nydus images. Then build the plugin image and apply
everything to `ate-system`:

```sh
kustomize build manifests | ko resolve -f - | kubectl apply -f -
```

## Preload the seeds

Seed nodes serve images from local disk only after you preload them. Write
the image refs, one per line, and run:

```sh
hack/seed-preload.sh < images.txt
```

The script needs crane, jq and kubectl. The seed nodes fetch the blobs with
their GCE service account, so this works only for Artifact Registry and GCR
on GCE. Run it again after you push new images.

## Verify

Check that the pods are ready:

```sh
kubectl -n ate-system get pods -l ate.dev/image-provider=nydus
```

The plugin logs each image it prepares, and each seed miss:

```sh
kubectl -n ate-system logs ds/image-provider-nydus -c image-provider
```

atelet records each request to the plugin on `ate.imageprovider.requests` and
`ate.imageprovider.duration`, by provider (`nydus`) and outcome. `success`
means the plugin served the image, and `fallback` means atelet pulled it into
its image cache instead.

## Uninstall

```sh
kustomize build manifests | kubectl delete -f -
```

atelet stops using the plugin once its socket is gone. Each node keeps the
downloaded binaries and modules under `/home/kubernetes/bin`, the cache under
`/var/lib/ate/image-plugin-cache`, the Dragonfly pieces under
`/var/lib/dragonfly`, and the seed blobs under `/var/lib/ate/nydus-seed-blobs`.

## Development

```sh
hack/test.sh        # unit tests
hack/verify-all.sh  # lint, formatting, headers, go.mod, manifests, shellcheck
```
