#!/usr/bin/env python3
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

"""Turns on the nydus-blob-seed access log.

  seed-access-log.py < manifests > manifests

nginx then logs one JSON line per request to stdout, with its own service
time ($request_time) and the range, for run.sh to collect. The buffer keeps
it to one write per 64 KB or per second.
"""
import sys

import yaml

LOG = """log_format seed escape=json '{"msec":$msec,"rt":$request_time,"st":$status,\
"b":$body_bytes_sent,"range":"$http_range","uri":"$uri","ra":"$remote_addr",\
"cr":$connection_requests}';
  access_log /var/log/nginx/access.log seed buffer=64k flush=1s;"""

docs = list(yaml.safe_load_all(sys.stdin))
patched = 0
for d in docs:
    if d and d.get("kind") == "ConfigMap" and d["metadata"]["name"] == "nydus-blob-seed":
        conf = d["data"]["nginx.conf"]
        if "access_log off;" not in conf:
            sys.exit("nydus-blob-seed: no 'access_log off;' to replace")
        d["data"]["nginx.conf"] = conf.replace("access_log off;", LOG)
        patched += 1
if patched != 1:
    sys.exit(f"found {patched} nydus-blob-seed ConfigMaps, want 1")
yaml.safe_dump_all(docs, sys.stdout, sort_keys=False)
