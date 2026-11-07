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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ai-on-gke/image-provider-plugin/client"
	snapshotsapi "github.com/containerd/containerd/api/services/snapshots/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StandardMountLayers prepares an image the way containerd's CRI plugin does
// with a remote snapshotter, and takes one read-only view per layer. For
// each layer missing from the snapshotter it calls Prepare with the CRI
// labels, and expects AlreadyExists, which means the snapshotter committed
// the layer remotely. A plain success means the snapshotter would need the
// layer downloaded, so the image is declined with client.ErrNotSupported.
// Providers whose snapshotter streams every layer, such as Riptide, use it
// as is.
func StandardMountLayers(ctx context.Context, env *MountEnv) (*MountedLayers, error) {
	imgKey := SanitizePathKey(env.ImageRef)
	runID := fmt.Sprintf("%x", time.Now().UnixNano())
	nsCtx := env.Driver.WithNamespaceContext(ctx)

	for i, c := range env.ChainInfos {
		prepKey := fmt.Sprintf("%s-%s-l%d-prep", imgKey, runID, i)
		labels := StandardCRILayerLabels(env.CanonicalRef, env.ImageDigest, env.LayerDigests, i, c.ChainID)
		if err := prepareRemoteLayer(nsCtx, env, c, prepKey, labels); err != nil {
			return nil, fmt.Errorf("layer %d (%s): %w", i, c.ChainID, err)
		}
	}

	res := &MountedLayers{
		MountDirs:    make([]string, len(env.ChainInfos)),
		SnapshotKeys: make([]string, len(env.ChainInfos)),
	}
	errs := make([]error, len(env.ChainInfos))
	var wg sync.WaitGroup
	for i, c := range env.ChainInfos {
		viewKey := fmt.Sprintf("%s-%s-l%d-view", imgKey, runID, i)
		res.SnapshotKeys[i] = viewKey
		wg.Go(func() {
			mounts, err := ViewLayerMounts(nsCtx, env.Client, env.Snapshotter, viewKey, c.ChainID)
			if err != nil {
				errs[i] = fmt.Errorf("layer %d (%s): %w", i, c.ChainID, err)
				return
			}
			if res.MountDirs[i] = ExtractMountDir(mounts); res.MountDirs[i] == "" {
				errs[i] = fmt.Errorf("layer %d (%s): no mount directory in view", i, c.ChainID)
			}
		})
	}
	wg.Wait()
	return res, errors.Join(errs...)
}

// prepareRemoteLayer makes sure the snapshotter has committed layer c,
// asking it to prepare the layer remotely if needed.
func prepareRemoteLayer(nsCtx context.Context, env *MountEnv, c ChainInfo, prepKey string, labels map[string]string) error {
	lk := env.Driver.AcquireChainLock(c.ChainID)
	lk.Lock()
	defer lk.Unlock()

	committed, err := HasSnapshot(nsCtx, env.Client, env.Snapshotter, c.ChainID)
	if err != nil || committed {
		return err
	}
	_, err = env.Client.Prepare(nsCtx, &snapshotsapi.PrepareSnapshotRequest{
		Snapshotter: env.Snapshotter,
		Key:         prepKey,
		Parent:      c.ParentChainID,
		Labels:      labels,
	})
	switch status.Code(err) {
	case codes.AlreadyExists:
		found, err := HasSnapshot(nsCtx, env.Client, env.Snapshotter, c.ChainID)
		if err != nil {
			return err
		}
		if !found {
			return errors.New("prepare returned AlreadyExists but the chain ID snapshot does not exist")
		}
		return nil
	case codes.OK:
		RemoveSnapshot(nsCtx, env.Client, env.Snapshotter, prepKey)
		return fmt.Errorf("declined by snapshotter: %w", client.ErrNotSupported)
	default:
		RemoveSnapshot(nsCtx, env.Client, env.Snapshotter, prepKey)
		return fmt.Errorf("preparing snapshot: %w", err)
	}
}
