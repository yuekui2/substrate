# image-provider-plugin

An image provider plugin prepares container image layers on an
[Agent Substrate](https://github.com/agent-substrate/substrate) node in place
of atelet's image cache, for example by loading them lazily from a remote store.
This repository holds:

- `api/imageprovider/v1alpha1`: the `ImageProviderPlugin` gRPC API that atelet
  calls over a Unix socket.
- `client`: the Go client atelet uses, and `client/mock`, a configurable fake for
  tests.
- `plugin`: a library for writing plugins on top of a containerd remote
  snapshotter.

## The API

A plugin runs as a DaemonSet, one pod per node, and serves these RPCs on
`/run/ate/image-provider.sock`:

- `CanProvide` asks whether the plugin serves an image, and returns the plugin's
  state. A draining plugin reports `PLUGIN_STATE_DRAINING`.
- `PrepareImage` prepares an image's layers and takes the lease that the caller
  names in `lease_id`. It returns the image digest, its config, and one host
  directory per layer, bottom first. Each directory is an overlayfs lowerdir as
  it is, with whiteouts in overlayfs format. Retrying it with the same lease
  takes no second lease. `auth_refresh` says whether the caller must keep
  refreshing the lease's credentials.
- `ReleaseImage` releases a lease, named by its ID, image digest and form.
  Releasing a lease the plugin does not hold succeeds.
- `ReconcileImages` replaces the plugin's leases with the leases that workloads
  still hold, after atelet or the plugin restarts.
- `RefreshAuth` replaces a held lease's registry credentials. It returns
  `NOT_FOUND` for a lease the plugin does not hold, and never takes one.

A plugin that declines an image returns `FAILED_PRECONDITION` or
`UNIMPLEMENTED`, and atelet pulls the image into its image cache instead.

## Writing a plugin

A plugin binary wraps its provider's containerd snapshotter in a
`plugin.Driver` and serves it:

```go
d, err := plugin.NewDriver("myprovider", "/run/myprovider/snapshotter.sock", "",
	plugin.WithAuthRefresh(imageprovider.AuthRefresh_AUTH_REFRESH_NOT_NEEDED))
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

The driver resolves each image, prepares its layers in the snapshotter, and
returns the snapshotter's mounts unchanged. It tracks the leases on each image
by lease ID, and keeps an image mounted for an idle time after its last lease
goes. Options cover what differs between providers: `WithLayerMounter`
replaces `StandardMountLayers`, `WithProbeHook` adds a health check, and
`WithWarmHitHook` and `WithWorkDirTeardownHook` run when an image is reused or
removed.

The driver keeps a `lease.json` per image under its work dir,
`/var/lib/ate/image-plugin-cache/<provider>` by default. After a restart,
`ReconcileImages` takes over the images that workloads still lease and removes
the rest.

Every driver sets `WithAuthRefresh`. A provider that reads the whole image
during `PrepareImage` sets `AUTH_REFRESH_NOT_NEEDED`. A provider that keeps
reading the registry sets `WithAuthRefreshHook` to apply fresh credentials,
and takes its mode from `CredentialRenewalOptions(--credential-renewal)`:
`client` reports `AUTH_REFRESH_REQUIRED` and applies the credentials that
`RefreshAuth` brings, and `self` adds `WithSelfRenewal`, which calls the hook
with no credentials every minute so the provider renews them with its
keychain. Use `self` only with node-level credentials.

On SIGTERM, `ServeUnixSocket` drains: the plugin declines new images, atelet
suspends the actors that hold leases, and when the last lease goes or the
drain times out, the plugin writes `/run/ate/drain-complete`. A provider whose
mounts are FUSE-backed keeps serving them until that file exists.

## Development

```sh
go test -race ./...     # unit tests
hack/verify-all.sh      # formatting, lint, license headers, generated code
go generate ./...       # regenerate the API code after editing the proto
```
