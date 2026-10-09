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
# A restore shows up as a wall-clock jump much larger than one ~10 ms pass.
exec 3<> <(:) # fd that never has data or EOF, so read -t just waits
last=${EPOCHREALTIME/./}
while :; do
  read -t 0.01 -u 3 # built-in timed wait, no process started
  now=${EPOCHREALTIME/./}
  if (( now - last > 1000000 )); then work resume; now=${EPOCHREALTIME/./}; fi
  last=$now
done
