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

# Runs in the node-sampler DaemonSet on every bench node.
#
#   node-sampler.sh sample     print /proc/diskstats counters of whole disks
#                              and CPU counters (see cpustat) as JSON lines
#                              every SAMPLE_INTERVAL seconds
#   node-sampler.sh snapshot   print one JSON line with the size of each
#                              image cache under /var/lib/ate and the Spegel
#                              /stats, if a Spegel plugin listens on :5000
#
# /proc/diskstats is not namespaced, so the pod sees the host's disks.
# Rates are computed offline from the counter deltas.

set -euo pipefail

NODE="${NODE_NAME:-unknown}"
INTERVAL="${SAMPLE_INTERVAL:-5}"
ATE=/host/var/lib/ate

sample() {
  local ncpu
  ncpu="$(grep -c '^cpu[0-9]' /proc/stat)"
  while :; do
    local ts disks="" sep=""
    ts="$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)"
    # Fields: major minor name reads reads_merged sectors_read read_ms
    # writes writes_merged sectors_written write_ms in_flight io_ms weighted_ms
    while read -r _ _ name rd _ rsec rms wr _ wsec wms inflight ioms wms2 _; do
      [[ "${name}" =~ ^(nvme[0-9]+n[0-9]+|sd[a-z]+)$ ]] || continue
      disks+="${sep}{\"dev\":\"${name}\",\"reads\":${rd},\"read_sectors\":${rsec},\"read_ms\":${rms},\"writes\":${wr},\"write_sectors\":${wsec},\"write_ms\":${wms},\"in_flight\":${inflight},\"io_ms\":${ioms},\"weighted_ms\":${wms2}}"
      sep=","
    done </proc/diskstats
    echo "{\"kind\":\"diskstats\",\"ts\":\"${ts}\",\"node\":\"${NODE}\",\"disks\":[${disks}]}"
    cpustat "${ts}" "${ncpu}"
    sleep "${INTERVAL}"
  done
}

# cpustat prints the node's CPU counters from /proc/stat and the CPU ticks
# (utime + stime) of live host processes summed by command name. Counters
# are in clock ticks (USER_HZ, 100 per second). Needs hostPID.
cpustat() {
  local user nice system idle iowait irq softirq steal procs
  read -r _ user nice system idle iowait irq softirq steal _ </proc/stat
  # comm is in parentheses and may hold spaces. utime and stime are the
  # 12th and 13th fields after it.
  procs="$(cat /proc/[0-9]*/stat 2>/dev/null | awk '
    match($0, /\(.*\)/) {
      comm = substr($0, RSTART + 1, RLENGTH - 2)
      gsub(/["\\]/, "", comm)
      # Kernel threads like kworker/3:1-events count as kworker.
      sub(/\/.*/, "", comm)
      split(substr($0, RSTART + RLENGTH + 1), f, " ")
      t[comm] += f[12] + f[13]
    }
    END {
      sep = ""
      for (k in t) if (t[k] > 0) { printf "%s\"%s\":%d", sep, k, t[k]; sep = "," }
    }')"
  echo "{\"kind\":\"cpustat\",\"ts\":\"$1\",\"node\":\"${NODE}\",\"ncpu\":$2,\"cpu\":{\"user\":${user},\"nice\":${nice},\"system\":${system},\"idle\":${idle},\"iowait\":${iowait},\"irq\":${irq},\"softirq\":${softirq},\"steal\":${steal}},\"procs\":{${procs}}}"
}

# du_bytes prints the apparent size of a directory in bytes, or 0 if absent.
# -x stays on the boot disk: walking a FUSE layer mount would fetch it.
du_bytes() {
  if [[ -d "$1" ]]; then
    du -sbx "$1" 2>/dev/null | cut -f1
  else
    echo 0
  fi
}

spegel_stats() {
  local resp
  if ! exec 3<>/dev/tcp/127.0.0.1/5000 2>/dev/null; then
    echo null
    return
  fi
  printf 'GET /stats HTTP/1.0\r\nHost: localhost\r\n\r\n' >&3
  resp="$(timeout 10 cat <&3 || true)"
  exec 3<&-
  # Drop the HTTP headers.
  resp="${resp#*$'\r\n\r\n'}"
  if [[ "${resp}" == \{* ]]; then
    echo "${resp}"
  else
    echo null
  fi
}

snapshot() {
  local ts dirs="" sep="" d
  ts="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  for d in image-cache image-plugin-cache actors; do
    dirs+="${sep}\"${d}\":$(du_bytes "${ATE}/${d}")"
    sep=","
  done
  if [[ -d "${ATE}/image-plugin-cache" ]]; then
    for d in "${ATE}"/image-plugin-cache/*/; do
      [[ -d "${d}" ]] || continue
      dirs+=",\"image-plugin-cache/$(basename "${d}")\":$(du_bytes "${d}")"
    done
  fi
  local fs
  fs="$(df -B1 --output=size,used,avail "${ATE}" 2>/dev/null | tail -1 | awk '{printf "{\"size\":%s,\"used\":%s,\"avail\":%s}", $1, $2, $3}')"
  echo "{\"kind\":\"snapshot\",\"ts\":\"${ts}\",\"node\":\"${NODE}\",\"bytes\":{${dirs}},\"rootfs\":${fs:-null},\"spegel\":$(spegel_stats),\"dragonfly\":$(dragonfly_metrics),\"nydusd\":$(nydusd_metrics)}"
}

# prom_lines <port> <path> <regex> prints the Prometheus lines matching regex
# from 127.0.0.1:<port><path> as a JSON array, or null if nothing listens.
prom_lines() {
  local resp
  if ! exec 4<>"/dev/tcp/127.0.0.1/$1" 2>/dev/null; then
    echo null
    return
  fi
  printf 'GET %s HTTP/1.0\r\nHost: localhost\r\n\r\n' "$2" >&4
  resp="$(timeout 10 cat <&4 || true)"
  exec 4<&-
  local lines
  lines="$(printf '%s\n' "${resp}" | grep -E "$3" |
    sed 's/\\/\\\\/g; s/"/\\"/g; s/^/"/; s/$/"/' | paste -sd,)"
  echo "[${lines}]"
}

# dragonfly_metrics prints the Dragonfly client's traffic and task counters.
dragonfly_metrics() {
  prom_lines 4002 /metrics '^dragonfly_client_[a-z_]*(traffic|task_total|back_to_source|upload|download)[a-z_]*[{ ]'
}

# nydusd_metrics prints nydusd's counters and latency histograms, which
# nydus-snapshotter exports on port 9110.
nydusd_metrics() {
  prom_lines 9110 /v1/metrics '^nydusd_'
}

case "${1:-}" in
  sample) sample ;;
  snapshot) snapshot ;;
  *) echo "usage: node-sampler.sh sample|snapshot" >&2; exit 2 ;;
esac
