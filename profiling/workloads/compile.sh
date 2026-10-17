#!/bin/bash
# Compiles $D at boot, then again after each restore, logging start/end.
set -u
D=${COMPILE_DIR:-/testbed/lib/matplotlib} 

mark() { echo "{\"msg\":\"workload $1\",\"workload\":\"compile\",\"trigger\":\"$2\",\"dir\":\"$D\"$3}"; }

work() { # compile workload 
  mark start "$1" ""
  t0=${EPOCHREALTIME/./}
  /opt/miniconda3/envs/testbed/bin/python -m compileall -f -q "$D" > /dev/null
  t1=${EPOCHREALTIME/./}
  mark end "$1" ",\"elapsed_us\":$((t1 - t0))"
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
