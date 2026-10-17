#!/bin/bash
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

# Reads every regular file on the rootfs at boot, then again after each
# restore, logging start/end. cat reads each path in full whatever the
# filesystem says about hardlinks, so a pulled and a streamed rootfs of the
# same image read the same bytes. It streams into wc rather than /dev/null so
# no tool can skip the read because nothing consumes it.
set -u
R=${READ_ROOT:-/}

mark() { echo "{\"msg\":\"workload $1\",\"workload\":\"readall\",\"trigger\":\"$2\",\"root\":\"$R\"$3}"; }

work() {
  mark start "$1" ""
  t0=${EPOCHREALTIME/./}
  # -xdev stays off /proc, /sys, /dev and tmpfs mounts.
  bytes=$(set -o pipefail; find "$R" -xdev -type f -print0 2>/dev/null | xargs -0 cat -- 2>/dev/null | wc -c)
  rc=$?
  t1=${EPOCHREALTIME/./}
  mark end "$1" ",\"bytes\":$bytes,\"rc\":$rc,\"elapsed_us\":$((t1 - t0))"
}

work boot
# A restore shows up as a wall-clock jump much larger than one ~50 ms pass.
# The wait is an external sleep rather than bash's read -t: a checkpoint taken
# during read -t could leave bash blocked in it after every later restore.
last=${EPOCHREALTIME/./}
while :; do
  sleep 0.05
  now=${EPOCHREALTIME/./}
  if (( now - last > 1000000 )); then work resume; now=${EPOCHREALTIME/./}; fi
  last=$now
done
