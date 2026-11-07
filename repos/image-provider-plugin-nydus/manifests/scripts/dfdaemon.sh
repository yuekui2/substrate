#!/bin/sh

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

# dfdaemon.sh configures and runs the node's Dragonfly dfdaemon.
#
# The proxy serves nydusd and the plugin as a registry mirror. Blob
# requests become Dragonfly tasks keyed by the blob digest, without the
# Range header, so every Range read of a layer is a slice of one task.
# dfdaemon fetches only the pieces a Range read needs, from peers when they
# have them.
#
# Environment:
#   NODE_IP       the node's IP, which peers connect to.
#   SEED_PEER     "true" to register as a super seed peer.
#   REGISTRY_URL  the registry to mirror for requests without an
#                 X-Dragonfly-Registry header.

set -eu

: "${NODE_IP:?}" "${SEED_PEER:?}" "${REGISTRY_URL:?}"

mkdir -p /etc/dragonfly /var/run/dragonfly
cat > /etc/dragonfly/dfdaemon.yaml << EOF
host:
  schedulerClusterID: 1
  ip: ${NODE_IP}
download:
  server:
    socketPath: /var/run/dragonfly/dfdaemon.sock
  concurrentPieceCount: 16
  backToSourceConcurrentPieceCount: 16
upload:
  server:
    port: 4000
    requestRateLimit: 10000
manager:
  addr: http://dragonfly-manager.ate-system.svc.cluster.local:65003
scheduler:
  announceInterval: 1m
  enableBackToSource: true
seedPeer:
  enable: ${SEED_PEER}
  type: super
storage:
  server:
    tcpPort: 4005
    quicPort: 4006
  dir: /var/lib/dragonfly
  keep: true
# Tasks stay on disk for a day so peers can keep fetching pieces from them.
gc:
  policy:
    taskTTL: 24h
    diskHighThresholdPercent: 90
    diskLowThresholdPercent: 85
proxy:
  server:
    ip: 127.0.0.1
    port: 4001
    # nydusd treats a 429 from the proxy rate limiter as a failed read, so
    # keep the limit out of reach.
    requestRateLimit: 100000
  rules:
  # nydusd reads ranges of at most a few chunks. With the default auto
  # policy, dfdaemon fetches ranges up to 4 MiB straight from the registry
  # and never asks the seeds.
  - regex: 'blobs/sha256.*'
    schedulingPolicy: always
  registryMirror:
    addr: ${REGISTRY_URL}
    enableTaskIDBasedBlobDigest: true
  # Prefetch would fetch every whole layer that nydusd touches.
  prefetch: false
health:
  server:
    port: 4003
metrics:
  server:
    port: 4002
EOF
exec /usr/local/bin/dfdaemon --config /etc/dragonfly/dfdaemon.yaml --console
