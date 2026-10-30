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

"""Guest for the image plugin benchmark.

Runs inside each SWE-bench image as the actor's only process, with the
image's own Python. The loader calls it once per turn through atenet-router:

  GET /readyz  wakeup probe
  GET /probe   the turn's first command
  GET /work    reads N .py files under /testbed and writes M .py files,
               and returns the timings as JSON

Only the standard library is used, so it runs on every image's Python.
"""

import http.server
import json
import os
import sys
import time
import urllib.parse

ROOT = "/testbed"
WRITE_DIR = "/testbed/.bench"


def list_py(root, limit):
    found = []
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = sorted(d for d in dirnames if not d.startswith("."))
        for name in sorted(filenames):
            if name.endswith(".py"):
                found.append(os.path.join(dirpath, name))
                if len(found) >= limit:
                    return found
    return found


def work(reads, writes, turn):
    t0 = time.monotonic()
    files = list_py(ROOT, reads)
    t1 = time.monotonic()
    read_bytes = 0
    for path in files:
        with open(path, "rb") as f:
            read_bytes += len(f.read())
    t2 = time.monotonic()
    os.makedirs(WRITE_DIR, exist_ok=True)
    written = 0
    for i in range(writes):
        src = files[i % len(files)] if files else None
        body = b""
        if src:
            with open(src, "rb") as f:
                body = f.read()
        path = os.path.join(WRITE_DIR, "turn_%d_%d.py" % (turn, i))
        with open(path, "wb") as f:
            f.write(b"# turn %d\n" % turn)
            f.write(body)
            f.flush()
            os.fsync(f.fileno())
        written += len(body)
    t3 = time.monotonic()
    return {
        "files_read": len(files),
        "bytes_read": read_bytes,
        "files_written": writes,
        "bytes_written": written,
        "list_s": t1 - t0,
        "read_s": t2 - t1,
        "write_s": t3 - t2,
        "work_s": t3 - t0,
    }


class Handler(http.server.BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        pass

    def reply(self, code, body):
        data = json.dumps(body).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        url = urllib.parse.urlparse(self.path)
        if url.path in ("/readyz", "/probe"):
            self.reply(200, {"ok": True})
            return
        if url.path == "/work":
            q = urllib.parse.parse_qs(url.query)
            reads = int(q.get("reads", ["200"])[0])
            writes = int(q.get("writes", ["2"])[0])
            turn = int(q.get("turn", ["0"])[0])
            try:
                self.reply(200, work(reads, writes, turn))
            except Exception as e:  # report, don't crash the guest
                self.reply(500, {"error": repr(e)})
            return
        self.reply(404, {"error": "not found"})


def main():
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 80
    server = http.server.HTTPServer(("0.0.0.0", port), Handler)
    server.serve_forever()


if __name__ == "__main__":
    main()
