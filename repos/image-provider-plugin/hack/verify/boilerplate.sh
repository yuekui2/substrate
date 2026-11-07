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

# Checks that every source file starts with the license header in
# hack/boilerplate. Shell scripts may put a shebang line and a blank line
# before it.

set -o errexit -o nounset -o pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT}"

failed=0
while IFS= read -r f; do
  case "${f}" in
    *.go | *.proto) template=hack/boilerplate/boilerplate.go.txt ;;
    *.sh | *.yaml | *.yml) template=hack/boilerplate/boilerplate.sh.txt ;;
    *) continue ;;
  esac
  [[ -f "${f}" ]] || continue
  lines="$(wc -l < "${template}")"
  head="$(head -n "$((lines + 2))" "${f}")"
  if [[ "${head}" == '#!'* ]]; then
    head="$(tail -n +3 <<< "${head}")"
  fi
  head="$(head -n "${lines}" <<< "${head}")"
  year="$(sed -nE '1s/.*Copyright ([0-9]{4}) Google LLC$/\1/p' <<< "${head}")"
  want="$(sed "s/YEAR/${year:-YEAR}/" "${template}")"
  if [[ -z "${year}" || "${head}" != "${want}" ]]; then
    echo "missing or wrong license header: ${f}" >&2
    failed=1
  fi
done < <(git ls-files --cached --others --exclude-standard -- .)

exit "${failed}"
