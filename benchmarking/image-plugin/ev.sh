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

# Runs a command and records it, its output and its exit code as evidence.
#
#   ev.sh <step-name> <command...>
#
# Appends to ${RUN_DIR}/transcript.md and writes the output to
# ${RUN_DIR}/logs/<NNN>-<step-name>.log.

set -uo pipefail

: "${RUN_DIR:?set RUN_DIR to the results directory of this run}"
mkdir -p "${RUN_DIR}/logs"
step="$1"
shift
# mkdir is atomic, so parallel callers get distinct numbers.
n=1
while ! mkdir "${RUN_DIR}/logs/.seq-${n}" 2>/dev/null; do
  n=$((n + 1))
done
num="$(printf '%03d' "${n}")"
log="${RUN_DIR}/logs/${num}-${step}.log"
start="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
{
  echo "# ${step}"
  echo "# start: ${start}"
  echo "\$ $*"
  echo
} >"${log}"
bash -c "$*" >>"${log}" 2>&1
rc=$?
echo "# exit: ${rc}  end: $(date -u +%Y-%m-%dT%H:%M:%SZ)" >>"${log}"
{
  echo "### ${num} ${step} (exit ${rc}, ${start})"
  echo '```'
  echo "\$ $*"
  echo '```'
  echo "Log: [logs/${num}-${step}.log](logs/${num}-${step}.log)"
  echo
} >>"${RUN_DIR}/transcript.md"
tail -20 "${log}"
exit "${rc}"
