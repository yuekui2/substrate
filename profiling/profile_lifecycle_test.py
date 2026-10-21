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

"""Tests for the log helpers in profile_lifecycle. Run: python3 -m unittest profile_lifecycle_test"""

import os
import stat
import tempfile
import time
import unittest
from unittest import mock

import profile_lifecycle as pl

LINE_A = '{"time":"2026-10-13T17:00:00.5Z","msg":"Picked worker"}'
LINE_B = '{"time":"2026-10-13T17:00:01Z","msg":"Handle RPC"}'


class ParseLogLinesTest(unittest.TestCase):
    def test_keeps_json_objects_and_adds_synthetic_keys(self):
        recs = pl.parse_log_lines(f"not json\n{LINE_A}\n[1, 2]\n")
        self.assertEqual(len(recs), 1)
        self.assertEqual(recs[0]["msg"], "Picked worker")
        self.assertEqual(recs[0]["_pod"], "")
        self.assertEqual(recs[0]["_raw"], LINE_A)
        self.assertAlmostEqual(recs[0]["_ts"] % 60, 0.5)

    def test_prefix_sets_pod_and_drops_unprefixed_lines(self):
        recs = pl.parse_log_lines(f"[pod/w-1/ateom] {LINE_A}\n{LINE_B}\n", prefix=True)
        self.assertEqual([r["_pod"] for r in recs], ["w-1"])


class MergeRecordsTest(unittest.TestCase):
    def test_drops_lines_seen_in_more_than_one_source(self):
        streamed = pl.parse_log_lines(f"{LINE_A}\n")
        scraped = pl.parse_log_lines(f"{LINE_A}\n{LINE_B}\n")
        merged = pl.merge_records(streamed, scraped)
        self.assertEqual([r["msg"] for r in merged], ["Picked worker", "Handle RPC"])

    def test_same_line_from_different_pods_is_kept(self):
        a = pl.parse_log_lines(f"[pod/p1/c] {LINE_A}\n", prefix=True)
        b = pl.parse_log_lines(f"[pod/p2/c] {LINE_A}\n", prefix=True)
        self.assertEqual(len(pl.merge_records(a, b)), 2)


class LogStreamTest(unittest.TestCase):
    """Runs LogStream against a fake kubectl that prints two lines and hangs,
    the way `kubectl logs -f` keeps following until it is stopped."""

    def setUp(self):
        self.dir = tempfile.TemporaryDirectory()
        fake = os.path.join(self.dir.name, "kubectl")
        with open(fake, "w") as f:
            f.write(f"#!/bin/sh\necho '{LINE_A}'\necho '{LINE_B}'\nexec sleep 60\n")
        os.chmod(fake, os.stat(fake).st_mode | stat.S_IEXEC)
        patcher = mock.patch.dict(os.environ, {"PATH": f"{self.dir.name}:{os.environ['PATH']}"})
        patcher.start()
        self.addCleanup(patcher.stop)
        self.addCleanup(self.dir.cleanup)

    def test_captures_lines_and_stops_follower(self):
        with pl.LogStream("app=x", "ns") as s:
            deadline = time.time() + 5
            while time.time() < deadline and len(s.snapshot()) < 2:
                time.sleep(0.1)
            self.assertIsNone(s._proc.poll(), "snapshot() stopped the follower")
            recs = s.records()
            self.assertIsNotNone(s._proc.poll(), "follower still running after records()")
        self.assertEqual([r["msg"] for r in recs], ["Picked worker", "Handle RPC"])

    def test_records_is_safe_to_call_twice(self):
        with pl.LogStream("app=x", "ns") as s:
            time.sleep(0.5)
            first = s.records()
            self.assertEqual(len(s.records()), len(first))

    def test_exit_removes_the_file(self):
        with pl.LogStream("app=x", "ns") as s:
            path = s._path
            self.assertTrue(os.path.exists(path))
        self.assertFalse(os.path.exists(path))


def workload_end(actor, ts):
    return {"msg": "workload end", "labels": {"ate.actor.name": actor}, "_ts": ts}


class FakeClock:
    """Advances only when the code under test sleeps."""

    def __init__(self):
        self.now = 1000.0

    def time(self):
        return self.now

    def sleep(self, seconds):
        self.now += seconds


class WaitForWorkloadEndTest(unittest.TestCase):
    def wait(self, batches, actor="a1", after=1000.0, timeout=10):
        """Feeds one batch of records per poll; the last batch repeats."""
        clock, calls = FakeClock(), []

        def read():
            calls.append(clock.now)
            return batches[min(len(calls), len(batches)) - 1]

        hit = pl.wait_for_workload_end(read, actor, after, timeout, clock=clock.time, sleep=clock.sleep)
        return hit, calls

    def test_returns_once_the_line_appears(self):
        hit, calls = self.wait([[], [], [workload_end("a1", 1002.0)]])
        self.assertEqual(hit["_ts"], 1002.0)
        self.assertEqual(len(calls), 3)

    def test_ignores_other_actors_and_earlier_lines(self):
        stale = [workload_end("a2", 1001.0), workload_end("a1", 999.0)]
        hit, _ = self.wait([stale, stale + [workload_end("a1", 1003.0)]])
        self.assertEqual(hit["_ts"], 1003.0)

    def test_ignores_workload_start(self):
        start = {"msg": "workload start", "labels": {"ate.actor.name": "a1"}, "_ts": 1001.0}
        hit, _ = self.wait([[start]], timeout=3)
        self.assertIsNone(hit)

    def test_times_out(self):
        hit, calls = self.wait([[]], timeout=5)
        self.assertIsNone(hit)
        self.assertLessEqual(calls[-1] - calls[0], 5)


class WorkloadTest(unittest.TestCase):
    def test_reports_end_bytes_and_rc(self):
        recs = pl.parse_log_lines("\n".join([
            '{"time":"2026-10-15T17:00:01Z","msg":"workload start","trigger":"resume",'
            '"labels":{"ate.actor.name":"a1"}}',
            '{"time":"2026-10-15T17:00:03Z","msg":"workload end","trigger":"resume","bytes":9513044055,'
            '"rc":0,"elapsed_us":2000000,"labels":{"ate.actor.name":"a1"}}',
        ]))
        out = pl.workload(recs, "a1", recs[0]["_ts"] - 1, "picked worker")
        self.assertEqual(out["workload bytes (resume)"], 9513044055)
        self.assertEqual(out["workload rc (resume)"], 0)
        self.assertAlmostEqual(out["workload elapsed in sandbox (resume)"], 2.0)
        self.assertAlmostEqual(out["picked worker -> workload end (resume)"], 3.0)


DIGEST = "sha256:64bb043b17a25dbeeba766c82d1d5f66ca446ca32da12c88418ded9e0d79cfdc"
IMAGE = f"us-west1-docker.pkg.dev/p/swebench-mirror/swebench-verified@{DIGEST}"
# Verbatim from ate-ocilayer-riptide (project name replaced): Go's default
# logger, second-resolution UTC timestamp, unquoted message.
ADAPTER_LINE = (f"2026/10/19 17:01:56 INFO PrepareImage timing breakdown provider=riptide "
                f"image={IMAGE} digest={DIGEST} layers=10 resolve_ms=399.442 stat_ms=17.194 "
                "prepare_ms=1983.077 view_ms=20.614 listable_ms=2.868 wrapper_ms=2.528 total_ms=2427.578")
STREAMED_LINE = ('{"time":"2026-10-19T17:01:56.95Z","msg":"Image streamed via OCILayerService",'
                 f'"image":"{IMAGE}","layers":10,"duration":2512811227}}')


class AdapterTest(unittest.TestCase):
    def test_parses_adapter_lines(self):
        r, = pl.parse_log_lines(f"[pod/ada-1/ocilayer-riptide] {ADAPTER_LINE}\n", prefix=True)
        self.assertEqual((r["_pod"], r["level"], r["msg"], r["layers"], r["prepare_ms"], r["digest"]),
                         ("ada-1", "INFO", "PrepareImage timing breakdown", 10, 1983.077, DIGEST))
        self.assertEqual(r["_ts"], pl.parse_ts("2026-10-19T17:01:56Z"))

    def test_quoted_values_and_bare_message(self):
        r = pl.parse_go_log('2026/10/19 17:00:00 WARN Something failed err="a \\"b\\" c=d" n=3')
        self.assertEqual((r["msg"], r["err"], r["n"]), ("Something failed", 'a "b" c=d', 3))
        self.assertEqual(pl.parse_go_log("2026/10/19 17:00:00 INFO no attributes")["msg"], "no attributes")
        self.assertIsNone(pl.parse_go_log("gcfsd: some other output"))

    def test_adapter_records_by_window_and_digest(self):
        ada, = pl.parse_log_lines(ADAPTER_LINE)
        t = ada["_ts"]
        self.assertEqual(pl.adapter_records([ada], t - 1, t + 1), [ada])
        self.assertEqual(pl.adapter_records([ada], t + 1, t + 2), [])
        self.assertEqual(pl.adapter_records([ada], t - 1, t + 1, {"sha256:bb"}), [])

    def test_cold_stream_from_adapter_and_atelet(self):
        recs = sorted(pl.parse_log_lines(f"{STREAMED_LINE}\n{ADAPTER_LINE}"), key=lambda r: r["_ts"])
        out, notes, streamed = pl.atelet_streaming(recs)
        self.assertEqual(out["swebench-verified path"], "STREAMED cold, 10 layers")
        self.assertAlmostEqual(out["  prepare_layers.prepare"], 1.983077)
        self.assertAlmostEqual(out["stream swebench-verified (atelet total)"], 2.512811227)
        self.assertEqual((notes, streamed), ([], {DIGEST}))

    def test_fallback(self):
        out, _, streamed = pl.atelet_streaming(pl.parse_log_lines(
            '{"time":"2026-10-15T17:00:02Z","msg":"OCILayerService layer preparation failed; '
            'falling back to cache","image":"r/img@sha256:aa","err":"boom"}'))
        self.assertEqual(out["img path"], "FALLBACK to pull: OCILayerService layer preparation failed (boom)")
        self.assertEqual(streamed, set())

    def test_declined_is_ignored_without_an_adapter(self):
        recs = pl.parse_log_lines(
            '{"time":"2026-10-15T17:00:02Z","msg":"OCILayerService declined image; '
            'falling back to cache","image":"r/img@sha256:aa"}')
        self.assertIn("img path", pl.atelet_streaming(recs)[0])
        self.assertEqual(pl.atelet_streaming(recs, ignore_declined=True)[0], {})

    def test_adapter_source(self):
        self.assertIsNone(pl.adapter_source(""))
        self.assertEqual(pl.adapter_source("spegel"),
                         ("app=ate-ocilayer-adapter,ate.dev/oci-layer-provider=spegel", "ocilayer-spegel"))
        self.assertIsInstance(pl.adapter_stream(""), pl.NoLogs)
        self.assertEqual(pl.adapter_logs("", 60), [])


SPEGEL_LINES = (
    "2026/10/19 17:01:50 INFO Materialized Spegel OCI layer layerIndex=0 blobDigest=sha256:01 "
    "source=upstream:us-west1-docker.pkg.dev layerFSDir=/x/0\n"
    "2026/10/19 17:01:51 INFO Materialized Spegel OCI layer layerIndex=1 blobDigest=sha256:02 "
    "source=spegel-p2p:10.0.0.7:5000 layerFSDir=/x/1\n"
    "2026/10/19 17:01:51 INFO Streamed OCI layer blob from remote Spegel P2P peer "
    "peer=10.0.0.7:5000 digest=sha256:02\n"
)


class AdapterNotesTest(unittest.TestCase):
    def test_spegel_sources_and_peers(self):
        notes, facts = pl.adapter_notes(pl.parse_log_lines(SPEGEL_LINES))
        self.assertEqual(facts, {"layer source": "registry+peer"})
        self.assertEqual(notes, ["spegel materialized 2 layers: 1 from registry, 1 from peer, 0 from local",
                                 "spegel streamed 1 blobs from peers ['10.0.0.7:5000']"])

    def test_local_store_is_not_a_peer(self):
        _, facts = pl.adapter_notes(pl.parse_log_lines(
            "2026/10/19 17:01:50 INFO Materialized Spegel OCI layer layerIndex=0 "
            "blobDigest=sha256:01 source=local-store\n"))
        self.assertEqual(facts, {"layer source": "local"})

    def test_nydus(self):
        notes, facts = pl.adapter_notes(pl.parse_log_lines(
            "2026/10/19 17:01:50 INFO Materialized and committed Nydus RAFS v6 EROFS bootstrap layer "
            "layerIndex=0 layerDigest=sha256:01\n"))
        self.assertEqual((notes, facts), (["nydus materialized 1 EROFS bootstrap layers"], {}))

    def test_digest_filter_keeps_layer_lines(self):
        recs = pl.parse_log_lines(SPEGEL_LINES)
        t = recs[0]["_ts"]
        self.assertEqual(len(pl.adapter_records(recs, t - 1, t + 2, {DIGEST})), 3)


if __name__ == "__main__":
    unittest.main()
