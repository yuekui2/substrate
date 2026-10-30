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

package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	"go.ate.dev/substrate/api/ateapi/v1"
)

func TestTemplateJSON(t *testing.T) {
	img := image{InstanceID: "a", OCI: "reg/oci@sha256:1", Nydus: "reg/nydus@sha256:2", Python: "/py"}
	for _, tc := range []struct {
		arm       string
		wantImage string
	}{
		{arm: "pull", wantImage: img.OCI},
		{arm: "nydus", wantImage: img.Nydus},
	} {
		t.Run(tc.arm, func(t *testing.T) {
			js, err := templateJSON(opts{arm: tc.arm, atespace: "ns", pool: "bench", probeTimeout: 180, cpuMilli: 100, memBytes: 1}, 7, img)
			if err != nil {
				t.Fatal(err)
			}
			var tpl ateapi.ActorTemplate
			if err := protojson.Unmarshal([]byte(js), &tpl); err != nil {
				t.Fatalf("protojson: %v", err)
			}
			c := tpl.GetContainers()[0]
			if c.GetImage() != tc.wantImage {
				t.Errorf("image = %q, want %q", c.GetImage(), tc.wantImage)
			}
			boot := c.GetCommand()[2]
			if len(boot) > 4096 {
				t.Errorf("command[2] is %d characters, the API allows 4096", len(boot))
			}
			// The command decodes back to guest.py.
			b64 := strings.TrimSuffix(strings.TrimPrefix(boot, "import base64,zlib;exec(zlib.decompress(base64.b64decode('"), "')))")
			raw, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				t.Fatalf("base64: %v", err)
			}
			zr, err := zlib.NewReader(bytes.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(zr)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, guestPy) {
				t.Error("decoded guest differs from guest.py")
			}
		})
	}
}

func TestReadImages(t *testing.T) {
	imgs, err := readImages("../images-320.jsonl", 320)
	if err != nil {
		t.Fatal(err)
	}
	for _, img := range imgs {
		if img.OCI == "" || img.Nydus == "" {
			b, _ := json.Marshal(img)
			t.Fatalf("image without oci or nydus ref: %s", b)
		}
	}
}
