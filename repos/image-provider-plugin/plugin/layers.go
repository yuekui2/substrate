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
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/ai-on-gke/image-provider-plugin/client"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// ChainInfo holds the OCI ChainID of a layer and of its parent.
type ChainInfo struct {
	ChainID       string
	ParentChainID string
}

// ComputeChainIDs computes the OCI ChainIDs for an ordered slice of layer
// DiffIDs.
func ComputeChainIDs(diffIDs []string) []ChainInfo {
	res := make([]ChainInfo, len(diffIDs))
	parent := ""
	for i, diff := range diffIDs {
		current := diff
		if parent != "" {
			h := sha256.Sum256([]byte(parent + " " + diff))
			current = "sha256:" + hex.EncodeToString(h[:])
		}
		res[i] = ChainInfo{
			ChainID:       current,
			ParentChainID: parent,
		}
		parent = current
	}
	return res
}

// SanitizePathKey turns an OCI image reference into a fixed-length directory
// name. Layer paths under it go to overlay as lowerdir+ values, which the
// kernel caps at 256 bytes, so the name must not grow with the reference.
func SanitizePathKey(ref string) string {
	sum := sha256.Sum256([]byte(ref))
	return hex.EncodeToString(sum[:16])
}

// CanonicalImageRef returns <repo>@<digest> for a tag reference when digest
// is set, and imageRef otherwise.
func CanonicalImageRef(imageRef, digest string) string {
	if digest == "" || strings.Contains(imageRef, "@") {
		return imageRef
	}
	if ref, err := name.ParseReference(imageRef); err == nil {
		return ref.Context().Name() + "@" + digest
	}
	return imageRef
}

// RemoteOptions returns go-containerregistry remote options that
// authenticate with authConfig if it has credentials, then with keychain, and
// then with the default and Google keychains.
func RemoteOptions(ctx context.Context, keychain authn.Keychain, authConfig *client.AuthConfig) []remote.Option {
	opts := []remote.Option{remote.WithContext(ctx)}
	if auth := authenticator(authConfig); auth != nil {
		return append(opts, remote.WithAuth(auth))
	}
	if keychain != nil {
		return append(opts, remote.WithAuthFromKeychain(keychain))
	}
	return append(opts, remote.WithAuthFromKeychain(authn.NewMultiKeychain(authn.DefaultKeychain, google.Keychain)))
}

func authenticator(c *client.AuthConfig) authn.Authenticator {
	switch {
	case c == nil:
		return nil
	case c.Username != "" || c.Password != "":
		return &authn.Basic{Username: c.Username, Password: c.Password}
	case c.Auth != "":
		return authn.FromConfig(authn.AuthConfig{Auth: c.Auth})
	case c.RegistryToken != "":
		return &authn.Bearer{Token: c.RegistryToken}
	case c.IdentityToken != "":
		return &authn.Bearer{Token: c.IdentityToken}
	}
	return nil
}
