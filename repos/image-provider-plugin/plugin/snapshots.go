// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	snapshotsapi "github.com/containerd/containerd/api/services/snapshots/v1"
	"github.com/containerd/containerd/api/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StandardCRILayerLabels returns the labels the containerd CRI plugin sets
// when it asks a remote snapshotter to prepare the layer at layerIndex.
func StandardCRILayerLabels(canonicalRef, manifestDigest string, layerDigests []string, layerIndex int, chainID string) map[string]string {
	labels := map[string]string{
		"containerd.io/snapshot/remote":              "true",
		"containerd.io/snapshot.ref":                 chainID,
		"containerd.io/snapshot/cri.image-ref":       canonicalRef,
		"containerd.io/snapshot/cri.manifest-digest": manifestDigest,
	}
	if layerIndex >= 0 && layerIndex < len(layerDigests) {
		labels["containerd.io/snapshot/cri.layer-digest"] = layerDigests[layerIndex]
		labels["containerd.io/snapshot/cri.image-layers"] = strings.Join(layerDigests[layerIndex:], ",")
	}
	return labels
}

// HasSnapshot reports whether key exists in the snapshotter. Pass a context
// from Driver.WithNamespaceContext.
func HasSnapshot(ctx context.Context, client snapshotsapi.SnapshotsClient, snapshotter, key string) (bool, error) {
	_, err := client.Stat(ctx, &snapshotsapi.StatSnapshotRequest{
		Snapshotter: snapshotter,
		Key:         key,
	})
	switch status.Code(err) {
	case codes.OK:
		return true, nil
	case codes.NotFound:
		return false, nil
	default:
		return false, fmt.Errorf("stat snapshot %q: %w", key, err)
	}
}

// RemoveSnapshot removes key from the snapshotter. It runs on cleanup paths,
// so it logs failures instead of returning them. A missing key is not a
// failure. Pass a context from Driver.WithNamespaceContext.
func RemoveSnapshot(ctx context.Context, client snapshotsapi.SnapshotsClient, snapshotter, key string) {
	_, err := client.Remove(ctx, &snapshotsapi.RemoveSnapshotRequest{
		Snapshotter: snapshotter,
		Key:         key,
	})
	if err != nil && status.Code(err) != codes.NotFound {
		slog.WarnContext(ctx, "Failed to remove snapshot",
			slog.String("snapshotter", snapshotter),
			slog.String("key", key),
			slog.Any("err", err))
	}
}

// ViewLayerMounts creates the read-only view viewKey of the committed
// snapshot parentChainID and returns its mounts. It removes the view if the
// snapshotter returns no mounts.
func ViewLayerMounts(ctx context.Context, client snapshotsapi.SnapshotsClient, snapshotter, viewKey, parentChainID string) ([]*types.Mount, error) {
	resp, err := client.View(ctx, &snapshotsapi.ViewSnapshotRequest{
		Snapshotter: snapshotter,
		Key:         viewKey,
		Parent:      parentChainID,
	})
	if err != nil {
		RemoveSnapshot(ctx, client, snapshotter, viewKey)
		return nil, fmt.Errorf("creating view: %w", err)
	}
	if len(resp.GetMounts()) == 0 {
		RemoveSnapshot(ctx, client, snapshotter, viewKey)
		return nil, fmt.Errorf("no mounts returned for snapshot %s", parentChainID)
	}
	return resp.GetMounts(), nil
}

// ExtractMountDir returns the read-only directory of a view: the bind
// source, or the first overlay lowerdir. It returns "" for no mounts.
func ExtractMountDir(mounts []*types.Mount) string {
	if len(mounts) == 0 {
		return ""
	}
	for _, m := range mounts {
		if m == nil {
			continue
		}
		if m.Source != "" && m.Source != "overlay" && m.Source != "none" {
			return m.Source
		}
		for _, opt := range m.Options {
			if lower, ok := strings.CutPrefix(opt, "lowerdir="); ok {
				if first, _, _ := strings.Cut(lower, ":"); first != "" {
					return first
				}
			}
		}
	}
	return mounts[0].GetSource()
}
