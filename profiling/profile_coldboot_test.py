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

"""Tests for golden_records in profile_coldboot. Run: python3 -m unittest profile_coldboot_test"""

import json
import unittest

import profile_coldboot as pc
import profile_lifecycle as pl

LABELS = "logging.googleapis.com/labels"


def line(t, msg, **fields):
    return json.dumps({"time": f"2026-10-15T17:00:{t:06.3f}Z", "msg": msg, **fields})


def starting(t, uid, trace):
    return json.dumps({"time": f"2026-10-15T17:00:{t:06.3f}Z", "message": "Actor starting",
                       "trace_id": trace, LABELS: {"ate.actor.name": uid}})


def picked(t, pod, trace):
    return line(t, "Picked worker", worker=f'worker_pod:"{pod}"  node_name:"n-{pod}"', trace_id=trace)


def parse(*lines):
    return pl.parse_log_lines("\n".join(lines))


# The stream arm's golden actor (uid "stream-uid") boots while a leftover pull
# template's golden actor ("pull-uid") boots too, from the same image.
API = parse(
    picked(1.0, "w-pull", "p-pick"),
    line(1.1, "Actor has no snapshot; Booting from ActorTemplate spec", trace_id="t-pull"),
    picked(2.0, "w-stream", "s-pick"),
    line(2.1, "Actor has no snapshot; Booting from ActorTemplate spec", trace_id="t-stream"),
)
LET = parse(
    line(1.2, "Image cache miss", digest="sha256:aa", trace_id="t-pull"),
    line(2.2, "PrepareLayers timing breakdown", digest="sha256:aa", trace_id="t-stream"),
)
WRK = pl.parse_log_lines("\n".join([
    f"[pod/w-pull/ateom] {starting(9.0, 'pull-uid', 't-pull')}",
    f"[pod/w-stream/ateom] {starting(4.0, 'stream-uid', 't-stream')}",
]), prefix=True)


class GoldenRecordsTest(unittest.TestCase):
    def test_keeps_only_the_golden_actors_records(self):
        api, let, notes = pc.golden_records(API, LET, WRK, "stream-uid")
        self.assertEqual(notes, [])
        self.assertEqual([r["msg"] for r in api],
                         ["Picked worker", "Actor has no snapshot; Booting from ActorTemplate spec"])
        self.assertIn('worker_pod:"w-stream"', api[0]["worker"])
        self.assertEqual(api[1]["trace_id"], "t-stream")
        self.assertEqual([r["msg"] for r in let], ["PrepareLayers timing breakdown"])

    def test_boot_path_sees_one_worker(self):
        api, _, _ = pc.golden_records(API, LET, WRK, "stream-uid")
        _, notes = pc.boot_path(api)
        self.assertEqual([n for n in notes if n.startswith("worker ")], ["worker w-stream on n-w-stream"])

    def test_picked_worker_is_the_last_one_before_the_boot(self):
        api = API + parse(picked(0.5, "w-stream", "old"), picked(5.0, "w-stream", "after-boot"))
        got, _, _ = pc.golden_records(api, LET, WRK, "stream-uid")
        self.assertEqual(got[0]["trace_id"], "s-pick")

    def test_falls_back_to_the_window_without_an_ateom_record(self):
        api, let, notes = pc.golden_records(API, LET, [], "stream-uid")
        self.assertEqual((api, let), (API, LET))
        self.assertEqual(len(notes), 1)


if __name__ == "__main__":
    unittest.main()
