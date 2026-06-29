"""Unit tests for deploy/scripts/canary_analyze.py (plan 17.9)."""

from __future__ import annotations

import json
import unittest
from unittest import mock

from canary_analyze import (
    ERROR_RATE_PROMOTE_THRESHOLD,
    ERROR_RATE_ROLLBACK_THRESHOLD,
    AnalysisResult,
    PrometheusClient,
    Sample,
    _parse_vector,
    evaluate_promote,
    evaluate_rollback,
    run_analysis,
    sample_canary,
)


class ParseVectorTests(unittest.TestCase):
    def test_parses_scalar(self) -> None:
        payload = {"data": {"result": [{"value": [1, "0.004"]}]}}
        self.assertAlmostEqual(_parse_vector(payload), 0.004)

    def test_missing_result(self) -> None:
        self.assertIsNone(_parse_vector({"data": {"result": []}}))

    def test_nan(self) -> None:
        payload = {"data": {"result": [{"value": [1, "NaN"]}]}}
        self.assertIsNone(_parse_vector(payload))


class ThresholdTests(unittest.TestCase):
    def test_rollback_above_one_percent(self) -> None:
        ok, reason = evaluate_rollback(Sample(error_rate=0.011, p95_seconds=0.2))
        self.assertTrue(ok)
        self.assertIn("rollback threshold", reason)

    def test_no_rollback_at_threshold(self) -> None:
        ok, _ = evaluate_rollback(Sample(error_rate=ERROR_RATE_ROLLBACK_THRESHOLD, p95_seconds=0.2))
        self.assertFalse(ok)

    def test_promote_healthy_canary(self) -> None:
        ok, reason = evaluate_promote(
            Sample(error_rate=0.002, p95_seconds=0.25),
            baseline_p95=0.24,
        )
        self.assertTrue(ok, reason)

    def test_promote_rejects_high_error_rate(self) -> None:
        ok, reason = evaluate_promote(
            Sample(error_rate=ERROR_RATE_PROMOTE_THRESHOLD, p95_seconds=0.2),
            baseline_p95=0.2,
        )
        self.assertFalse(ok)
        self.assertIn("promote threshold", reason)

    def test_promote_rejects_latency_regression(self) -> None:
        ok, reason = evaluate_promote(
            Sample(error_rate=0.001, p95_seconds=0.50),
            baseline_p95=0.40,
        )
        self.assertFalse(ok)
        self.assertIn("p95", reason)


class PrometheusClientTests(unittest.TestCase):
    def test_query_parses_response(self) -> None:
        body = json.dumps({"status": "success", "data": {"result": [{"value": [1, "0.5"]}]}}).encode()
        with mock.patch("urllib.request.urlopen") as urlopen:
            urlopen.return_value.__enter__.return_value.read.return_value = body
            client = PrometheusClient("http://prom:9090")
            self.assertAlmostEqual(client.query("up"), 0.5)


class RunAnalysisTests(unittest.TestCase):
    def test_three_consecutive_failures_trigger_rollback(self) -> None:
        client = mock.Mock(spec=PrometheusClient)
        client.query.side_effect = [
            0.2,  # baseline p95
            0.02, 0.3,  # minute 1 — error rate 2%
            0.02, 0.3,  # minute 2
            0.02, 0.3,  # minute 3 — rollback
        ]

        with mock.patch("canary_analyze.time.sleep"):
            result = run_analysis(
                client,
                canary_color="green",
                baseline_color="blue",
                window_minutes=1,
                eval_interval_secs=0,
                rollback_streak=3,
            )

        self.assertEqual(result.decision, "rollback")
        self.assertGreaterEqual(result.consecutive_failures, 3)

    def test_promote_after_healthy_window(self) -> None:
        client = mock.Mock(spec=PrometheusClient)
        # baseline p95, then one healthy minute (error + p95)
        client.query.side_effect = [0.2, 0.001, 0.21]

        with mock.patch("canary_analyze.time.monotonic", side_effect=[0, 0, 70]):
            with mock.patch("canary_analyze.time.sleep"):
                result = run_analysis(
                    client,
                    canary_color="green",
                    baseline_color="blue",
                    window_minutes=1,
                    eval_interval_secs=0,
                    rollback_streak=3,
                )

        self.assertEqual(result.decision, "promote")


class SampleCanaryTests(unittest.TestCase):
    def test_sample_canary_queries(self) -> None:
        client = mock.Mock(spec=PrometheusClient)
        client.query.side_effect = [0.01, 0.3]
        sample = sample_canary(client, color="green", window="1m")
        self.assertEqual(sample.error_rate, 0.01)
        self.assertEqual(sample.p95_seconds, 0.3)
        self.assertEqual(client.query.call_count, 2)


if __name__ == "__main__":
    unittest.main()
