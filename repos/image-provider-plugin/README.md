# image-provider-plugin

An image provider plugin prepares container image layers on a
[Agent Substrate](https://github.com/agent-substrate/substrate) node in place
of atelet's image cache, for example by loading them lazily from a remote store.
This repository holds:

- `api/imageprovider/v1alpha1`: the `ImageProviderPlugin` gRPC API that atelet
  calls over a Unix socket.
- `client`: the Go client atelet uses, and `client/mock`, a configurable fake for
  tests.
- `plugin`: a library for writing plugins on top of a containerd remote
  snapshotter.

See [docs/overview.md](docs/overview.md) for how atelet and a plugin work
together.

## The API

atelet calls four RPCs on the plugin's socket, `/run/ate/image-provider.sock` by
default:

- `CanProvide` asks whether the plugin serves an image, and returns the plugin's
  state. A draining plugin reports `PLUGIN_STATE_DRAINING`.
- `PrepareImage` prepares an image's layers and takes a lease on them. It returns
  the image digest, its config, and one host directory per layer, bottom first,
  each with the layer's tree under `fs/`.
- `ReleaseImage` releases one lease.
- `ReconcileImages` replaces the plugin's leases with the images that workloads
  still use, after atelet or the plugin restarts.

A plugin that declines an image returns `FAILED_PRECONDITION` or
`UNIMPLEMENTED`, and atelet pulls the image into its image cache instead.

## Writing a plugin

A plugin binary wraps its provider's containerd snapshotter in a
`plugin.Driver` and serves it:

```go
d, err := plugin.NewDriver("myprovider", "/run/myprovider/snapshotter.sock", "")
if err != nil {
	log.Fatal(err)
}
defer d.Close()
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
defer stop()
if err := plugin.ServeUnixSocket(ctx, plugin.DefaultListenSocket, plugin.DefaultDrainCompleteFile, d); err != nil {
	log.Fatal(err)
}
```

The driver resolves each image, prepares its layers in the snapshotter, links
them under its work dir, `/var/lib/ate/image-plugin-cache/<provider>` by
default, and counts leases per image. The work dir must be under
`/var/lib/ate`, which the worker pods mount. Options cover what differs between
providers: `WithLayerMounter` replaces `StandardMountLayers`, `WithProbeHook`
adds a health check, and `WithWarmHitHook` and `WithWorkDirTeardownHook` run
when an image is reused or removed.

## Development

```sh
go test -race ./...     # unit tests
hack/verify-all.sh      # formatting, lint, license headers, generated code
go generate ./...       # regenerate the API code after editing the proto
```
