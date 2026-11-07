#!/usr/bin/env bash

# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Regenerates the Go code for the protos in this repository with buf.gen.yaml.
# go generate ./... runs it.

set -o errexit -o nounset -o pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

protoc_gen_go="$(hack/run-tool.sh --print-bin-path protoc-gen-go)"
protoc_gen_go_grpc="$(hack/run-tool.sh --print-bin-path protoc-gen-go-grpc)"

# buf.gen.yaml names the plugins. Put the pinned builds first on PATH.
PATH="$(dirname "${protoc_gen_go}"):$(dirname "${protoc_gen_go_grpc}"):${PATH}" \
  hack/run-tool.sh buf generate
