#!/bin/bash
# Reads one file at boot, then again after each restore, logging start/end.
set -u
F=${READ_FILE:-/testbed/lib/matplotlib/pyplot.py} # small file = pyplot.py, big file = pytest 

mark() { echo "{\"msg\":\"workload $1\",\"workload\":\"readfile\",\"trigger\":\"$2\",\"file\":\"$F\"$3}"; }

work() {
  mark start "$1" ""
  t0=${EPOCHREALTIME/./}
  cat -- "$F" > /dev/null
  t1=${EPOCHREALTIME/./}
  mark end "$1" ",\"bytes\":$(stat -c %s -- "$F"),\"elapsed_us\":$((t1 - t0))"
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
