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
	long := "us-central1-docker.pkg.dev/project/repo/swebench-nydus/sweb.eval.x86_64.scikit-learn_1776_scikit-learn-14141@sha256:" + strings.Repeat("a", 64)
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
