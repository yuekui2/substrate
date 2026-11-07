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

# Lints every shell script in the repository. Uses shellcheck from PATH if
# present, and the pinned image otherwise.

set -o errexit -o nounset -o pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT}"

SHELLCHECK_IMAGE="docker.io/koalaman/shellcheck:v0.9.0@sha256:f35e8987b02760d4e76fc99a68ad5c42cc10bb32f3dd2143a3cf92f1e5446a45"

scripts=()
while IFS= read -r f; do
  [[ -f "${f}" ]] && scripts+=("${f}")
done < <(git ls-files --cached --others --exclude-standard -- '*.sh')

if [[ "${#scripts[@]}" -eq 0 ]]; then
  exit 0
fi

if command -v shellcheck > /dev/null; then
  shellcheck "${scripts[@]}"
else
  docker run --rm -v "${ROOT}:${ROOT}:ro" -w "${ROOT}" "${SHELLCHECK_IMAGE}" "${scripts[@]}"
fi
