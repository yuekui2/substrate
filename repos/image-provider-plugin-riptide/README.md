# image-provider-plugin-riptide

An [image provider plugin](https://github.com/ai-on-gke/image-provider-plugin)
for [Agent Substrate](https://github.com/agent-substrate/substrate) that streams
container images with Riptide, the image streaming in GKE. Actors start before
their images are fully pulled: gcfsd fetches file contents from the registry as
they are read.

## How it runs

`manifests/` deploys one DaemonSet, `image-provider-riptide`, in `ate-system`.
Each pod has two containers:

- `riptide-provider` runs its own gcfsd and `containerd-gcfs-grpc` snapshotter
  in the node's namespaces. They are separate from the host's, so they work
  whether or not GKE image streaming is on for the node.
- `image-provider-riptide` serves the plugin API on
  `/run/ate/image-provider.sock`. It mounts each image through the snapshotter
  and pushes the image's registry credentials to gcfsd's keychain.

By default the DaemonSet runs the gcfsd, gcfsd-v2 and `containerd-gcfs-grpc`
binaries that GKE preloads under `/home/kubernetes/bin`. Custom builds can
replace them; see [Custom binaries](#custom-binaries).

The DaemonSet keeps its state under
`/var/lib/ate/image-plugin-cache/riptide-gcfs`, apart from the host's
containerd, and its sockets under `/run/ate`. A restart keeps the layer cache
and clears the snapshot and view state. On SIGTERM the plugin drains its leases
(see [Draining the plugin](https://github.com/agent-substrate/substrate/blob/main/docs/guides/image-provider.md#draining-the-plugin))
and gcfsd keeps serving until it has finished.

## Prerequisites

- GKE nodes. The default binaries come with every GKE node image.
- Agent Substrate installed, with atelet mounting the node's `/run/ate`, as
  `manifests/ate-install/atelet.yaml` in Substrate does. See
  [Image Provider Plugin](https://github.com/agent-substrate/substrate/blob/main/docs/guides/image-provider.md).
- `kustomize`, [`ko`](https://ko.build) and `kubectl`, and a registry that the
  nodes can pull from for the plugin image.

## Deploy

```sh
export KO_DOCKER_REPO=REGION-docker.pkg.dev/PROJECT/REPO
kustomize build manifests | ko resolve -f - | kubectl apply -f -
```

atelet starts using the plugin once its socket appears. No atelet restart is
needed.

To uninstall:

```sh
kustomize build manifests | kubectl delete -f -
```

## Settings

Set these env vars on the `riptide-provider` container, for example with a
kustomize patch:

| Variable | Default | Meaning |
|---|---|---|
| `RIPTIDE_BIN_DIR` | `${KUBE_HOME}/bin` | Node directory with the gcfsd, gcfsd-v2 and `containerd-gcfs-grpc` binaries. |
| `CLEAR_CACHE_ON_START` | `false` | `true` clears the layer cache at every start. The cache is also cleared when the binaries change. |
| `ENABLE_GCFS_V2_API` | `true` | Runs gcfsd-v2 and the snapshotter's v2 API. `false` runs gcfsd. |
| `ENABLE_GCFS_LAYER_CACHING` | `true` | Keeps fetched layers on the node's disk. |
| `ENABLE_USERNAMESPACE_SUPPORT` | `false` | With the v2 API, enables ID mapping in gcfsd. |
| `GCFSD_CACHE_SIZE_MIB` | gcfsd's default | gcfsd memory cache size, split between file contents and large files. |

## Custom binaries

Set `RIPTIDE_BIN_DIR` to a node directory that holds the binaries. It must be
on an exec mount; `/var` is noexec on Container-Optimized OS.

`cmd/riptide-bin-image` installs custom builds for you. It packs the binaries
into an image, pushes it, and writes a kustomize patch. The patch adds an init
container that copies the binaries to `/home/kubernetes/bin/ate-riptide` and
sets `RIPTIDE_BIN_DIR` to it:

```sh
go run ./cmd/riptide-bin-image \
  --gcfsd=path/to/gcfsd-v2 \
  --snapshotter=path/to/containerd-gcfs-grpc \
  --repo=REGION-docker.pkg.dev/PROJECT/REPO/riptide-bins:TAG \
  --patch-out=overlay/riptide-bins.yaml
```

Pass `--v1` with a gcfsd build for `ENABLE_GCFS_V2_API=false`. Then deploy an
overlay that applies the patch, `overlay/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- ../manifests
patches:
- path: riptide-bins.yaml
```

```sh
kustomize build overlay | ko resolve -f - | kubectl apply -f -
```

## Verify

Check that the pods are ready and the plugin passed its startup check:

```sh
kubectl -n ate-system get pods -l ate.dev/image-provider=riptide
kubectl -n ate-system logs -l ate.dev/image-provider=riptide -c image-provider-riptide
```

The plugin logs "Provider passed its health check" at startup and "Pushed
credentials to gcfsd" for each image it prepares. atelet records each request
on the `ate.imageprovider.requests` and `ate.imageprovider.duration` metrics
with `ate.imageprovider.provider=riptide`. An `ate.imageprovider.outcome` of
`fallback` means atelet pulled the image into its image cache instead.

## Development

```sh
hack/test.sh        # go test -race ./...
hack/verify-all.sh  # gofmt, golangci-lint, shellcheck, license headers,
                    # go mod tidy, generated code, kustomize build
go generate ./...   # regenerates internal/proto/riptidekeychain
```
