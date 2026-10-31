#!/usr/bin/env bash
# Samples the Dragonfly scheduler every 5 s into $1 until killed.
set -uo pipefail
out=$1
while true; do
  p=$(kubectl -n ate-system get pods -l app=dragonfly-scheduler -o name 2>/dev/null | head -1 | cut -d/ -f2)
  ts=$(date +%s)
  m=$(kubectl get --raw "/api/v1/namespaces/ate-system/pods/$p:8000/proxy/metrics" 2>/dev/null)
  cpu=$(awk '$1=="process_cpu_seconds_total"{print $2}' <<<"$m")
  gor=$(awk '$1=="go_goroutines"{print $2}' <<<"$m")
  started=$(awk '/^grpc_server_started_total/{s+=$2} END{print s+0}' <<<"$m")
  top=$(kubectl -n ate-system top pod "$p" --no-headers 2>/dev/null | awk '{print $2, $3}')
  echo "$ts cpu_s=$cpu goroutines=$gor grpc_started=$started top=$top" >>"$out"
  sleep 5
done
