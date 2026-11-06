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

# Repeats and the load sweep for the no-walk setup (original images, 1-CPU
# workers): 16 and 24 per node, two more reps at 32, one more rep of the
# uncompressed set and of the walk baseline at 32.

set -uo pipefail

C="$(dirname "${BASH_SOURCE[0]}")/cpu.sh"
NOWALK=--warm-metadata=false
NYDUS_PLUGIN_ARGS="${NOWALK}" "${C}" point e5-nowalk-n16 16
NYDUS_PLUGIN_ARGS="${NOWALK}" "${C}" point e5-nowalk-n24 24
NYDUS_PLUGIN_ARGS="${NOWALK}" "${C}" point e2-nowalk-n32-r2 32
NYDUS_PLUGIN_ARGS="${NOWALK}" TAG=-none IMAGES=images-96-none.jsonl "${C}" point e4-none-n32-r2 32
NYDUS_PLUGIN_ARGS="${NOWALK}" "${C}" point e2-nowalk-n32-r3 32
NYDUS_PLUGIN_ARGS= "${C}" point e1-base-n32-r2 32
echo QUEUE-DONE
