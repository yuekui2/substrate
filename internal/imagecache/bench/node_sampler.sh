#!/usr/bin/env bash

# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit -o nounset -o pipefail

LABEL="$1"
shift

SAMPLE_FILE="/tmp/pull_samples_$$.txt"
rm -f "${SAMPLE_FILE}"

NUM_CPUS="$(grep -c '^processor' /proc/cpuinfo)"

sample_once() {
  awk '
    FILENAME == "/proc/uptime" { up = $1 }
    FILENAME == "/proc/stat" && $1 == "cpu" {
      usr_sys = $2 + $3 + $4 + $7 + $8 + $9
      iowait = $6
      total = $2 + $3 + $4 + $5 + $6 + $7 + $8 + $9
    }
    FILENAME == "/proc/net/dev" && ($1 ~ /^(eth0|ens4):/) {
      sub(/^[^:]+:/, "", $0)
      rx += $1
      tx += $9
    }
    FILENAME == "/proc/diskstats" && ($3 == "sda" || $3 == "nvme0n1") {
      sect_r += $6
      sect_w += $10
    }
    END {
      printf "%.3f %.0f %.0f %.0f %.0f %.0f %.0f\n", up, usr_sys, iowait, total, rx, sect_r, sect_w
    }
  ' /proc/uptime /proc/stat /proc/net/dev /proc/diskstats >> "${SAMPLE_FILE}"
}

sync
echo 3 > /proc/sys/vm/drop_caches 2>/dev/null || true

sample_once
(
  while true; do
    sleep 0.5
    sample_once
  done
) &
SAMPLER_PID=$!

cleanup() {
  kill "${SAMPLER_PID}" 2>/dev/null || true
  wait "${SAMPLER_PID}" 2>/dev/null || true
}
trap cleanup EXIT

T0="$(awk '{print $1}' /proc/uptime)"
"$@"
T1="$(awk '{print $1}' /proc/uptime)"

kill "${SAMPLER_PID}" 2>/dev/null || true
wait "${SAMPLER_PID}" 2>/dev/null || true
sample_once

awk -v label="${LABEL}" -v ncpu="${NUM_CPUS}" -v t0="${T0}" -v t1="${T1}" '
  NR == 1 {
    up0 = $1; us0 = $2; io0 = $3; tot0 = $4; rx0 = $5; sr0 = $6; sw0 = $7
    prev_up = up0; prev_us = us0; prev_io = io0; prev_tot = tot0; prev_rx = rx0; prev_sw = sw0
    printf "\n=== TELEMETRY: %s (CPUs=%d) ===\n", label, ncpu
    printf "%6s | %10s | %10s | %12s | %12s\n", "t(s)", "CPU(cores)", "IOw(cores)", "NetRx(MB/s)", "DiskW(MB/s)"
    printf "-------+------------+------------+--------------+-------------\n"
    next
  }
  {
    dt = $1 - prev_up
    if (dt <= 0.05) next
    dtot = $4 - prev_tot
    if (dtot <= 0) dtot = 1
    cores = (($2 - prev_us) / dtot) * ncpu
    iocores = (($3 - prev_io) / dtot) * ncpu
    rx_mbps = (($5 - prev_rx) / (1000.0 * 1000.0)) / dt
    dw_mbps = ((($7 - prev_sw) * 512.0) / (1000.0 * 1000.0)) / dt

    if (cores > peak_cores) peak_cores = cores
    if (iocores > peak_iocores) peak_iocores = iocores
    if (rx_mbps > peak_rx) peak_rx = rx_mbps
    if (dw_mbps > peak_dw) peak_dw = dw_mbps

    if (rx_mbps > 5.0) {
      active_net_t += dt
      active_net_bytes += ($5 - prev_rx)
    }

    printf "%6.2f | %10.2f | %10.2f | %12.2f | %12.2f\n", ($1 - up0), cores, iocores, rx_mbps, dw_mbps

    prev_up = $1; prev_us = $2; prev_io = $3; prev_tot = $4; prev_rx = $5; prev_sw = $7
    last_us = $2; last_io = $3; last_tot = $4; last_rx = $5; last_sw = $7
  }
  END {
    wall = t1 - t0
    if (wall <= 0) wall = prev_up - up0
    tot_ticks = last_tot - tot0
    if (tot_ticks <= 0) tot_ticks = 1
    avg_cores = ((last_us - us0) / tot_ticks) * ncpu
    avg_iocores = ((last_io - io0) / tot_ticks) * ncpu
    core_sec = avg_cores * wall
    tot_rx_mb = (last_rx - rx0) / (1000.0 * 1000.0)
    avg_rx_mbps = tot_rx_mb / wall
    active_rx_mbps = (active_net_t > 0) ? ((active_net_bytes / (1000.0 * 1000.0)) / active_net_t) : 0
    tot_dw_mb = ((last_sw - sw0) * 512.0) / (1000.0 * 1000.0)
    avg_dw_mbps = tot_dw_mb / wall

    printf "-------+------------+------------+--------------+-------------\n"
    printf "SUMMARY [%s]:\n", label
    printf "  Wall-Clock Duration      : %.2f s\n", wall
    printf "  CPU Utilization (usr+sys): avg %.2f cores (%.1f%% of %d cores), peak %.2f cores, total %.2f core-s\n", avg_cores, (avg_cores/ncpu)*100.0, ncpu, peak_cores, core_sec
    printf "  CPU I/O Wait             : avg %.2f cores, peak %.2f cores\n", avg_iocores, peak_iocores
    printf "  Network Rx Total         : %.2f MB\n", tot_rx_mb
    printf "  Network Rx Throughput    : avg %.2f MB/s (%.2f Gbps), peak %.2f MB/s (%.2f Gbps)\n", avg_rx_mbps, (avg_rx_mbps*8)/1000.0, peak_rx, (peak_rx*8)/1000.0
    printf "  Network Active Window    : %.2f s (>5 MB/s), active avg %.2f MB/s (%.2f Gbps)\n", active_net_t, active_rx_mbps, (active_rx_mbps*8)/1000.0
    printf "  Disk Write Total         : %.2f MB\n", tot_dw_mb
    printf "  Disk Write Throughput    : avg %.2f MB/s, peak %.2f MB/s\n", avg_dw_mbps, peak_dw
    printf "==============================================================\n"
  }
' "${SAMPLE_FILE}"

rm -f "${SAMPLE_FILE}"
