#!/bin/sh

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

# Copies the binaries in /riptide to the node directory mounted at
# /host-bin/ate-riptide. riptide-bin-image adds this script to the image,
# and its patch runs it in an init container. Each file is written under a
# temporary name and renamed, so a process still running the old file
# doesn't make the copy fail with "text file busy".

set -eu

dst=/host-bin/ate-riptide
mkdir -p "${dst}"
for f in /riptide/*; do
  n="$(basename "${f}")"
  cp "${f}" "${dst}/.${n}.tmp"
  chmod 0755 "${dst}/.${n}.tmp"
  mv -f "${dst}/.${n}.tmp" "${dst}/${n}"
done
