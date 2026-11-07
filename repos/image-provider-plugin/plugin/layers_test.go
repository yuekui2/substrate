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
	"strings"
	"testing"
)

func TestSanitizePathKey(t *testing.T) {
	long := "us-central1-docker.pkg.dev/project/repo/very-long-image-name-for-a-benchmark-suite/eval.x86_64.some-project_1776_some-project-14141@sha256:" + strings.Repeat("a", 64)
	for _, tc := range []struct {
		name string
		ref  string
	}{
		{"short", "busybox@sha256:" + strings.Repeat("b", 64)},
		{"long", long},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizePathKey(tc.ref)
			if len(got) != 32 {
				t.Errorf("SanitizePathKey(%q) = %q, want 32 characters", tc.ref, got)
			}
			if strings.ContainsAny(got, "/:@") {
				t.Errorf("SanitizePathKey(%q) = %q, has path separators", tc.ref, got)
			}
			if got == SanitizePathKey(tc.ref+"x") {
				t.Errorf("SanitizePathKey gave %q for two different refs", got)
			}
		})
	}
}

func TestComputeChainIDs(t *testing.T) {
	a := "sha256:" + strings.Repeat("a", 64)
	b := "sha256:" + strings.Repeat("b", 64)
	// sha256("<a> <b>"), the OCI ChainID of [a, b].
	ab := "sha256:ccd722928bd92476ba1745586fed6e45a102504185ad88cd89e01ff116fd146c"
	got := ComputeChainIDs([]string{a, b})
	want := []ChainInfo{{ChainID: a}, {ChainID: ab, ParentChainID: a}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("ComputeChainIDs = %+v, want %+v", got, want)
	}
}

func TestCanonicalImageRef(t *testing.T) {
	digest := "sha256:" + strings.Repeat("c", 64)
	for _, tc := range []struct {
		ref, digest, want string
	}{
		{"example.com/app:v1", digest, "example.com/app@" + digest},
		{"example.com/app@" + digest, "sha256:other", "example.com/app@" + digest},
		{"example.com/app:v1", "", "example.com/app:v1"},
		{"busybox", digest, "index.docker.io/library/busybox@" + digest},
	} {
		if got := CanonicalImageRef(tc.ref, tc.digest); got != tc.want {
			t.Errorf("CanonicalImageRef(%q, %q) = %q, want %q", tc.ref, tc.digest, got, tc.want)
		}
	}
}
