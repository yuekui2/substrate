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

# Image streaming matrix: pull vs. Riptide v2 vs. Spegel vs. Dragonfly+Nydus.
# Run from the substrate repo root against one cluster (kubectl's current
# context); run one copy per cluster to split the workloads across clusters.
#
#   WORKLOADS=small START_REP=1 N=1 profiling/repeat_matrix.sh
#
# Each rep runs every arm in one of four counterbalanced orders (a 4x4
# Williams square: each arm once in every position, and directly after every
# other arm once, per four reps). For each arm, untimed: remove the previous
# arm's WorkerPool, adapter and node pool, install this arm's adapter, create
# a fresh node pool (empty caches) and a WorkerPool on it. Then timed: cold
# boot from spec (profile_coldboot.py) and resume/suspend/resume
# (profile_lifecycle.py).
#
# Required env: CLUSTER ZONE NODE_VERSION VER DS BUCKET_NAME KO_DOCKER_REPO
# SWEBENCH_IMG SWEBENCH_NYDUS_IMG (repo@digest). RENDER_ONLY=1 renders the
# adapter manifests into $MANIFESTS and exits.
set -uo pipefail

WORKLOADS="${WORKLOADS:-small medium large}"
ARMS="${ARMS:-pull riptidev2 spegel dragonfly-nydus}"  # a subset keeps the sequence order of the rest
START_REP="${START_REP:-1}"
N="${N:-16}"                                 # reps START_REP .. START_REP+N-1
GAP="${GAP:-0}"
SETTLE="${SETTLE:-10}"
WORKLOAD_TIMEOUT="${WORKLOAD_TIMEOUT:-600}"
IM="${IM:-matplotlib__matplotlib-23476}"
SLUG_M="swebench-$(echo "$IM" | tr '_' '-' | tr -s '-')"
SYSTEM_POOL="${SYSTEM_POOL:-substrate-system-e2}"
POOL=pool-arm
MACHINE_TYPE="${MACHINE_TYPE:-c3-standard-22}"
DISK_TYPE="${DISK_TYPE:-pd-ssd}"
DISK_SIZE="${DISK_SIZE:-300}"
IMAGE_TYPE="${IMAGE_TYPE:-UBUNTU_CONTAINERD}"  # Nydus needs EROFS on-demand and cachefiles
VER_SUFFIX="${VER_SUFFIX:-$VER}"
PROJECT_ID="${PROJECT_ID:-$(gcloud config get-value project 2>/dev/null)}"
# Shared by every cluster so all arm runs use one adapter build.
MANIFESTS="${MANIFESTS:-profiling/results/matrix/manifests-$VER}"
: "${CLUSTER:?}" "${ZONE:?}" "${NODE_VERSION:?}" "${VER:?}" "${DS:?}" "${BUCKET_NAME:?}" "${KO_DOCKER_REPO:?}"

declare -A SCRIPT=([small]=file-read [medium]=compile [large]=read-all)
# The label values are the adapter names each adapter manifest's node affinity
# matches; none matches no adapter, so atelet finds no socket and pulls.
declare -A LABEL=([pull]=none [riptidev2]=riptide [spegel]=spegel [dragonfly-nydus]=nydus-dragonfly)
declare -A ADAPTER=([pull]="" [riptidev2]=riptide [spegel]=spegel [dragonfly-nydus]=nydus-dragonfly)
SEQUENCES=(
  "pull riptidev2 dragonfly-nydus spegel"
  "riptidev2 spegel pull dragonfly-nydus"
  "spegel dragonfly-nydus riptidev2 pull"
  "dragonfly-nydus pull spegel riptidev2"
)
DRAGONFLY_CONTROL_PLANE="dragonfly-mysql dragonfly-redis dragonfly-manager dragonfly-scheduler"

render_manifests() {
  mkdir -p "$MANIFESTS"
  local a f
  for a in riptide spegel nydus-dragonfly; do
    f="$MANIFESTS/ocilayer-$a.yaml"
    [[ -s "$f" ]] && continue
    # The same substitution as hack/install-ate.sh substitute_version.
    hack/run-tool.sh ko resolve -f "cmd/ocilayer-$a/ocilayer-$a.yaml" \
      | sed -e "s/: \${SUBSTRATE_VERSION}\$/: \"${VER}\"/" \
            -e "s/\${SUBSTRATE_VERSION}/${VER}/g" \
            -e "s/\${SUBSTRATE_VERSION_SUFFIX}/${VER_SUFFIX}/g" > "$f.tmp" \
      && mv "$f.tmp" "$f" || { echo "rendering $a failed" >&2; exit 1; }
  done
}
render_manifests
# GKE Ubuntu kernels lack EROFS/cachefiles on-demand support, which Nydus
# needs; the dragonfly-nydus arm loads rebuilt modules (nydus-kmods.sh).
NYDUS_KERNEL="${NYDUS_KERNEL:-6.8.0-1061-gke}"   # the kernel of $NODE_VERSION's Ubuntu image
KMODS="$MANIFESTS/kmods-$NYDUS_KERNEL"
if [[ " $ARMS " == *" dragonfly-nydus "* && ! -s "$KMODS/cachefiles.ko" ]]; then
  profiling/nydus-kmods.sh "$NYDUS_KERNEL" "$KMODS" || { echo "building Nydus kernel modules failed" >&2; exit 1; }
fi
[[ "${RENDER_ONLY:-0}" == 1 ]] && { echo "rendered $MANIFESTS"; exit 0; }
: "${SWEBENCH_IMG:?}" "${SWEBENCH_NYDUS_IMG:?}"
declare -A IMAGE=([pull]=$SWEBENCH_IMG [riptidev2]=$SWEBENCH_IMG [spegel]=$SWEBENCH_IMG
                  [dragonfly-nydus]=$SWEBENCH_NYDUS_IMG)

OUT="${OUT:-profiling/results/matrix/$CLUSTER}"
mkdir -p "$OUT"
LOG="$OUT/run.log"
log() { echo "$(date -u +%T) $*" | tee -a "$LOG"; }

cat > "$OUT/config-$(date -u +%Y%m%dT%H%M%S).json" <<EOF
{
  "cluster": "$CLUSTER", "zone": "$ZONE", "node_version": "$NODE_VERSION",
  "substrate_version": "$VER", "machine_type": "$MACHINE_TYPE", "disk_type": "$DISK_TYPE",
  "disk_size_gb": $DISK_SIZE, "image_type": "$IMAGE_TYPE", "instance": "$IM",
  "workloads": "$WORKLOADS", "arms": "$ARMS", "start_rep": $START_REP, "n": $N,
  "gap_s": $GAP, "settle_s": $SETTLE, "workload_timeout_s": $WORKLOAD_TIMEOUT,
  "images": {"oci": "$SWEBENCH_IMG", "nydus": "$SWEBENCH_NYDUS_IMG"},
  "manifests": {$(for a in riptide spegel nydus-dragonfly; do
      printf '"%s": "%s", ' "$a" "$(sha256sum "$MANIFESTS/ocilayer-$a.yaml" | cut -c1-16)"; done | sed 's/, $//')},
  "sequences": [$(printf '"%s",' "${SEQUENCES[@]}" | sed 's/,$//')],
  "nydus_kmods": {"kernel": "$NYDUS_KERNEL", "sha256": "$(cat "$KMODS"/*.ko 2>/dev/null | sha256sum | cut -c1-16)"}
}
EOF

install_adapter() {  # $1 = adapter, empty for none
  [[ -n "$1" ]] || return 0
  [[ "$1" != nydus-dragonfly ]] || install_nydus_kmods || return 1
  kubectl apply -f "$MANIFESTS/ocilayer-$1.yaml" || return 1
  if [[ "$1" == nydus-dragonfly ]]; then
    patch_nydus_kmods || return 1
    # The control plane has no node selector; keep it off the measured nodes.
    local d
    for d in $DRAGONFLY_CONTROL_PLANE; do
      kubectl -n ate-system patch deploy "$d" --type merge -p \
        "{\"spec\":{\"template\":{\"spec\":{\"nodeSelector\":{\"cloud.google.com/gke-nodepool\":\"$SYSTEM_POOL\"}}}}}" || return 1
    done
    for d in $DRAGONFLY_CONTROL_PLANE; do
      kubectl -n ate-system rollout status deploy/"$d" --timeout=10m || return 1
    done
  fi
}
uninstall_adapter() {  # $1 = adapter, empty for none
  [[ -n "$1" ]] || return 0
  kubectl delete -f "$MANIFESTS/ocilayer-$1.yaml" --wait --ignore-not-found
  [[ "$1" != nydus-dragonfly ]] || kubectl -n ate-system delete configmap nydus-kmods --ignore-not-found
}
install_nydus_kmods() {  # the rebuilt modules, as a ConfigMap (create, not apply: no size-doubling annotation)
  kubectl -n ate-system delete configmap nydus-kmods --ignore-not-found >/dev/null
  kubectl -n ate-system create configmap nydus-kmods --from-file="$KMODS"
}
# An init container on the adapter DaemonSet swaps the stock erofs and
# cachefiles modules for the rebuilt ones on every new node, before the
# snapshotter starts nydusd, and fails the pod unless "bind ondemand" on
# /dev/cachefiles then works. The manifest's own swap then finds on-demand
# symbols and leaves them loaded; it isn't relied on because its kallsyms test
# also matches the stock module's trace events and has no functional check.
patch_nydus_kmods() {
  local ds img
  ds=$(kubectl -n ate-system get ds -l app=ate-ocilayer-adapter,ate.dev/oci-layer-provider=nydus-dragonfly -o name)
  img=$(kubectl -n ate-system get "$ds" \
    -o jsonpath='{.spec.template.spec.containers[?(@.name=="nydus-snapshotter")].image}')
  [[ -n "$ds" && -n "$img" ]] || return 1
  sed "s|__IMAGE__|$img|" <<'EOF' | kubectl -n ate-system patch "$ds" --type strategic --patch-file /dev/stdin
spec:
  template:
    spec:
      volumes:
      - name: nydus-kmods
        configMap:
          name: nydus-kmods
      initContainers:
      - name: load-nydus-kmods
        image: __IMAGE__
        securityContext:
          privileged: true
        volumeMounts:
        - name: nydus-kmods
          mountPath: /kmods
        - name: kube-home
          mountPath: /home/kubernetes
        command: ["/bin/bash", "-ec"]
        args:
        - |
          want=$(cat /kmods/kernel); have=$(uname -r)
          if [[ "$want" != "$have" ]]; then
            echo "FATAL: node kernel $have, Nydus modules built for $want (set NYDUS_KERNEL)" >&2; exit 1
          fi
          mkdir -p /home/kubernetes/bin/modules
          cp /kmods/erofs.ko /kmods/cachefiles.ko /home/kubernetes/bin/modules/
          exec nsenter -t 1 -m -u -i -n -- /bin/sh -ec '
            M=/home/kubernetes/bin/modules
            modprobe netfs; modprobe erofs; modprobe cachefiles   # loads their dependencies
            rmmod cachefiles; rmmod erofs
            insmod $M/erofs.ko; insmod $M/cachefiles.ko
            d=$(mktemp -d /var/lib/cf-probe.XXXXXX)
            exec 3<>/dev/cachefiles
            echo "dir $d" >&3
            echo "bind ondemand" >&3
            exec 3>&-
            rm -rf "$d"
            grep -qw erofs_fscache_register_fs /proc/kallsyms
            echo "on-demand erofs and cachefiles loaded on $(uname -r)"'
EOF
}
create_pool() {  # $1 = arm. Retries cover transient stockouts.
  local try
  for try in 1 2 3; do
    gcloud container node-pools create "$POOL" --cluster "$CLUSTER" --zone "$ZONE" \
      --machine-type "$MACHINE_TYPE" --disk-type "$DISK_TYPE" --disk-size "$DISK_SIZE" \
      --image-type "$IMAGE_TYPE" --node-version "$NODE_VERSION" --num-nodes 2 \
      --workload-metadata GKE_METADATA \
      --node-labels "ate.dev/substrate-version=$VER,bench-arm=$1,ate.dev/oci-layer-service=${LABEL[$1]}" \
      --no-enable-autoupgrade --no-enable-autorepair --no-enable-image-streaming --quiet && return 0
    log "node pool create failed (try $try); retrying"
    gcloud container node-pools delete "$POOL" --cluster "$CLUSTER" --zone "$ZONE" --quiet || true
    sleep 60
  done
  return 1
}
delete_pool() {
  gcloud container node-pools describe "$POOL" --cluster "$CLUSTER" --zone "$ZONE" >/dev/null 2>&1 || return 0
  gcloud container node-pools delete "$POOL" --cluster "$CLUSTER" --zone "$ZONE" --quiet || true
}
wait_adapter() {  # $1 = adapter: Ready on every arm node; with none, no adapter pod at all
  local n sel="app=ate-ocilayer-adapter${1:+,ate.dev/oci-layer-provider=$1}"
  for n in $(kubectl get nodes -l "cloud.google.com/gke-nodepool=$POOL" -o jsonpath='{.items[*].metadata.name}'); do
    if [[ -z "$1" ]]; then
      if kubectl -n ate-system get pods -l "$sel" --field-selector "spec.nodeName=$n" -o name | grep . >/dev/null; then
        log "adapter pod found on pull node $n"; return 1
      fi
      continue
    fi
    for _ in $(seq 60); do
      kubectl -n ate-system get pods -l "$sel" --field-selector "spec.nodeName=$n" -o name | grep . >/dev/null && break
      sleep 5
    done
    kubectl -n ate-system wait pod --for=condition=Ready --timeout=10m -l "$sel" \
      --field-selector "spec.nodeName=$n" || return 1
  done
}
mkpool() {  # $1 = WorkerPool
  sed -e "s|name: profiling$|name: $1|" -e "s|workload: profiling$|workload: $1|" \
      -e "s|\${SUBSTRATE_VERSION}|$VER|g" \
      -e "s|\${WORKER_COUNT}|3|g; s|\${WORKER_CPU}|1|g; s|\${WORKER_MEMORY}|2Gi|g" \
      -e "s|^\(      ate.dev/substrate-version: .*\)$|\1\n      cloud.google.com/gke-nodepool: $POOL|" \
      profiling/profiling-pool.yaml.tmpl
}
delete_golden() {  # $1 = arm
  local a
  for a in $(kubectl-ate get actor -a ate-golden 2>/dev/null | awk -v t="ate-prof-$1/$SLUG_M" '$3 == t {print $2}'); do
    kubectl-ate delete actor "$a" -a ate-golden --any-state || true
  done
}
reset_arm() {  # $1 = arm: delete the arm's actors, template and golden actors
  local as="ate-prof-$1" a
  for a in $(kubectl-ate get actor -a "$as" 2>/dev/null | awk 'NR > 1 {print $2}'); do
    kubectl-ate delete actor "$a" -a "$as" --any-state || true
  done
  # Deleting the template first stops the reconciler from rebuilding the
  # golden actor deleted next.
  if kubectl-ate get actor-template "$SLUG_M" -a "$as" >/dev/null 2>&1; then
    kubectl-ate delete actor-template "$SLUG_M" -a "$as" || true
    for _ in $(seq 60); do
      kubectl-ate get actor-template "$SLUG_M" -a "$as" >/dev/null 2>&1 || break
      sleep 2
    done
  fi
  delete_golden "$1"
}

CURRENT_ARM=""
teardown() {  # remove the current arm's WorkerPool, adapter and nodes
  [[ -n "$CURRENT_ARM" ]] || return 0
  kubectl -n ate-profiling delete workerpool "profiling-$CURRENT_ARM" --ignore-not-found
  uninstall_adapter "${ADAPTER[$CURRENT_ARM]}"
  delete_pool
  CURRENT_ARM=""
}
# Ctrl-C stops the whole run instead of moving on to the next arm.
trap 'echo interrupted; teardown; exit 130' INT TERM

prepare_arm() {  # $1 = arm, $2 = rep dir. Untimed.
  local arm=$1 dir=$2
  teardown
  delete_pool  # left by an interrupted run
  CURRENT_ARM=$arm
  # Installed before the nodes exist, so its pods start as they register.
  install_adapter "${ADAPTER[$arm]}" || return 1
  create_pool "$arm" || return 1
  kubectl -n ate-system rollout status ds/"$DS" --timeout=10m || return 1
  wait_adapter "${ADAPTER[$arm]}" || return 1
  if [[ "$arm" == dragonfly-nydus ]]; then
    # The adapter turns Ready even when nydusd failed to start.
    sleep 5
    local p
    for p in $(kubectl -n ate-system get pods -l app=ate-ocilayer-adapter,ate.dev/oci-layer-provider=nydus-dragonfly -o name); do
      kubectl -n ate-system logs "$p" -c load-nydus-kmods 2>/dev/null | tail -1
      if kubectl -n ate-system logs "$p" -c nydus-snapshotter 2>/dev/null \
          | grep -E 'Failed to start singleton daemon|probably not started' >/dev/null; then
        log "nydusd failed to start in $p"; return 1
      fi
    done
  fi
  mkpool "profiling-$arm" | hack/run-tool.sh ko apply -f - || return 1
  kubectl -n ate-profiling wait --for=jsonpath='{.status.readyReplicas}'=3 \
    workerpool/"profiling-$arm" --timeout=10m || return 1
  { kubectl -n ate-profiling get pods -o wide
    echo; kubectl -n ate-system get pods -l app=ate-ocilayer-adapter -o wide
    echo; kubectl get nodes -l "cloud.google.com/gke-nodepool=$POOL" \
      -L ate.dev/oci-layer-service,bench-arm,node.kubernetes.io/instance-type
    echo; echo "NODE CPU_PLATFORM"
    gcloud compute instances list --project "$PROJECT_ID" \
      --filter="name:gke-${CLUSTER}-${POOL}-*" --format='value(name,cpuPlatform)'
  } > "$dir/$arm-workers.txt"
  reset_arm "$arm"
}
run_arm() {  # $1 = arm, $2 = rep dir, $3 = workload script. Timed.
  local arm=$1 dir=$2 script=$3
  local env="WORKLOAD=profiling-$arm ACTOR_SCRIPT=$script IMAGE_REF=${IMAGE[$arm]}"
  env+=" STORAGE_LOCATION=gs://$BUCKET_NAME/$CLUSTER/ate-prof-$arm/"
  profiling/profile_coldboot.py --instance "$IM" --atespace "ate-prof-$arm" \
    --worker-namespace ate-profiling --worker-pool "profiling-$arm" --adapter "${ADAPTER[$arm]}" \
    --env "$env" --json "$dir/$arm-coldboot-$IM.json"
  sleep "$GAP"
  profiling/profile_lifecycle.py --atespace "ate-prof-$arm" --template "ate-prof-$arm/$SLUG_M" \
    --worker-namespace ate-profiling --worker-pool "profiling-$arm" --adapter "${ADAPTER[$arm]}" \
    --step-gap "$GAP" --settle "$SETTLE" --workload-timeout "$WORKLOAD_TIMEOUT" \
    --json "$dir/$arm-lifecycle-$IM.json"
}
check_arm() {  # $1 = arm, $2 = rep dir: log what the cold boot measured
  local f="$2/$1-coldboot-$IM.json"
  [[ -s "$f" ]] || { log "CHECK $1: no coldboot result"; return; }
  log "CHECK $1: $(jq -r '[(.streaming["swebench-verified path"] // ("image " + (.images["swebench-verified"] // "MISSING"))),
     ("bytes " + ((.workload["workload bytes (boot)"] // "MISSING") | tostring)),
     (.streaming["layer source"] // empty | "layer source " + .)] | join(", ")' "$f")"
}

# Expired gcloud credentials fail every pool operation; find out now.
gcloud auth print-access-token >/dev/null || { echo "gcloud auth failed" >&2; exit 1; }
# Only the measured arm's adapter may exist: remove any left from earlier
# installs (other versions included) or an interrupted run.
for a in riptide spegel nydus-dragonfly; do uninstall_adapter "$a"; done
kubectl -n ate-system delete ds -l app=ate-ocilayer-adapter --wait --ignore-not-found
for arm in "${!LABEL[@]}"; do reset_arm "$arm"; done

for w in $WORKLOADS; do
  script="$PWD/profiling/workloads/${SCRIPT[$w]:?unknown workload $w}.sh"
  for i in $(seq "$START_REP" $((START_REP + N - 1))); do
    seq_i=$(( (i - 1) % 4 ))
    dir="$OUT/$w/rep-$(printf %02d "$i")"; mkdir -p "$dir"
    printf 'sequence %d: %s\n' $((seq_i + 1)) "${SEQUENCES[$seq_i]}" > "$dir/order.txt"
    log "=== $w rep $i, sequence $((seq_i + 1)): ${SEQUENCES[$seq_i]} -> $dir"
    for arm in ${SEQUENCES[$seq_i]}; do
      [[ " $ARMS " == *" $arm "* ]] || continue
      log "--- $arm: preparing"
      # A failed preparation (auth, quota, stockout after retries) would fail
      # every later arm the same way, so stop rather than record empty arms.
      if ! prepare_arm "$arm" "$dir"; then
        log "STOP: preparing $arm failed in $w rep $i; rerun with WORKLOADS=$w START_REP=$i"
        teardown
        exit 1
      fi
      log "--- $arm: measuring"
      run_arm "$arm" "$dir" "$script"
      check_arm "$arm" "$dir"
    done
  done
done
teardown
log "done: $OUT"
