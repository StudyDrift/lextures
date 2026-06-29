#!/usr/bin/env python3
"""Canary analysis for production deploys (plan 17.9 FR-4 / AC-2 / AC-3).

Queries Prometheus for error rate and p95 latency on the canary (green) color,
compares against a pre-deploy baseline, and decides promote vs rollback.

Exit codes:
  0 — promote (thresholds met for the full analysis window)
  1 — rollback (error rate breach for 3 consecutive evaluation windows)
  2 — usage / configuration error
  3 — inconclusive (caller should fall back to time-based promote or abort)
"""

from __future__ import annotations

import argparse
import json
import math
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import asdict, dataclass
from typing import Any


# Plan 17.9 AC-2: rollback when canary 5xx rate > 1% for 3 consecutive minutes.
ERROR_RATE_ROLLBACK_THRESHOLD = 0.01
# Plan 17.9 AC-3: promote when error rate < 0.5% and p95 within 10% of baseline.
ERROR_RATE_PROMOTE_THRESHOLD = 0.005
P95_BASELINE_TOLERANCE = 0.10

ERROR_RATE_QUERY = """
sum(rate(lextures_http_requests_total{{deploy_color="{color}",status="5xx"}}[{window}]))
/
clamp_min(sum(rate(lextures_http_requests_total{{deploy_color="{color}"}}[{window}])), 1)
""".strip()

P95_QUERY = """
histogram_quantile(
  0.95,
  sum(rate(lextures_http_request_duration_seconds_bucket{{deploy_color="{color}"}}[{window}])) by (le)
)
""".strip()


@dataclass(frozen=True)
class Sample:
    error_rate: float | None
    p95_seconds: float | None


@dataclass(frozen=True)
class AnalysisResult:
    decision: str
    reason: str
    canary_color: str
    baseline_p95_seconds: float | None
    final_sample: Sample | None
    consecutive_failures: int
    samples: list[dict[str, Any]]


def _parse_vector(payload: dict[str, Any]) -> float | None:
    data = payload.get("data", {})
    result = data.get("result")
    if not result:
        return None
    value = result[0].get("value")
    if not value or len(value) < 2:
        return None
    raw = value[1]
    if raw in ("NaN", "+Inf", "-Inf"):
        return None
    try:
        v = float(raw)
    except (TypeError, ValueError):
        return None
    if math.isnan(v) or math.isinf(v):
        return None
    return v


class PrometheusClient:
    def __init__(self, base_url: str, timeout_secs: float = 15.0) -> None:
        self.base_url = base_url.rstrip("/")
        self.timeout_secs = timeout_secs

    def query(self, promql: str) -> float | None:
        params = urllib.parse.urlencode({"query": promql})
        url = f"{self.base_url}/api/v1/query?{params}"
        req = urllib.request.Request(url, headers={"Accept": "application/json"})
        try:
            with urllib.request.urlopen(req, timeout=self.timeout_secs) as resp:
                payload = json.load(resp)
        except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as exc:
            raise RuntimeError(f"prometheus query failed: {exc}") from exc
        if payload.get("status") != "success":
            raise RuntimeError(f"prometheus error: {payload.get('error', payload)}")
        return _parse_vector(payload)


def sample_canary(
    client: PrometheusClient,
    *,
    color: str,
    window: str,
) -> Sample:
    error_q = ERROR_RATE_QUERY.format(color=color, window=window)
    p95_q = P95_QUERY.format(color=color, window=window)
    return Sample(
        error_rate=client.query(error_q),
        p95_seconds=client.query(p95_q),
    )


def evaluate_promote(
  sample: Sample,
  *,
  baseline_p95: float | None,
) -> tuple[bool, str]:
    if sample.error_rate is None:
        return False, "missing canary error rate"
    if sample.error_rate >= ERROR_RATE_PROMOTE_THRESHOLD:
        return False, f"error rate {sample.error_rate:.4f} >= promote threshold {ERROR_RATE_PROMOTE_THRESHOLD}"
    if baseline_p95 is not None and baseline_p95 > 0:
        if sample.p95_seconds is None:
            return False, "missing canary p95 latency"
        upper = baseline_p95 * (1 + P95_BASELINE_TOLERANCE)
        if sample.p95_seconds > upper:
            return False, (
                f"p95 {sample.p95_seconds:.3f}s exceeds baseline {baseline_p95:.3f}s "
                f"+ {P95_BASELINE_TOLERANCE:.0%} tolerance"
            )
    return True, "canary healthy"


def evaluate_rollback(sample: Sample) -> tuple[bool, str]:
    if sample.error_rate is None:
        return False, "missing error rate (not a rollback signal)"
    if sample.error_rate > ERROR_RATE_ROLLBACK_THRESHOLD:
        return (
            True,
            f"error rate {sample.error_rate:.4f} > rollback threshold {ERROR_RATE_ROLLBACK_THRESHOLD}",
        )
    return False, "error rate within rollback threshold"


def run_analysis(
    client: PrometheusClient,
    *,
    canary_color: str,
    baseline_color: str,
    window_minutes: int,
    eval_interval_secs: int,
    rollback_streak: int,
) -> AnalysisResult:
    window = f"{window_minutes}m"
    eval_window = "1m"
    deadline = time.monotonic() + window_minutes * 60
    consecutive_failures = 0
    samples: list[dict[str, Any]] = []

    baseline_p95 = None
    try:
        baseline_p95 = client.query(P95_QUERY.format(color=baseline_color, window=eval_window))
    except RuntimeError:
        baseline_p95 = None

    final_sample: Sample | None = None

    while time.monotonic() < deadline:
        try:
            sample = sample_canary(client, color=canary_color, window=eval_window)
        except RuntimeError as exc:
            return AnalysisResult(
                decision="inconclusive",
                reason=str(exc),
                canary_color=canary_color,
                baseline_p95_seconds=baseline_p95,
                final_sample=None,
                consecutive_failures=consecutive_failures,
                samples=samples,
            )

        final_sample = sample
        entry = {
            "ts": time.time(),
            "error_rate": sample.error_rate,
            "p95_seconds": sample.p95_seconds,
        }
        samples.append(entry)

        should_rollback, rollback_reason = evaluate_rollback(sample)
        if should_rollback:
            consecutive_failures += 1
            if consecutive_failures >= rollback_streak:
                return AnalysisResult(
                    decision="rollback",
                    reason=rollback_reason,
                    canary_color=canary_color,
                    baseline_p95_seconds=baseline_p95,
                    final_sample=sample,
                    consecutive_failures=consecutive_failures,
                    samples=samples,
                )
        else:
            consecutive_failures = 0

        time.sleep(eval_interval_secs)

    if final_sample is None:
        return AnalysisResult(
            decision="inconclusive",
            reason="no samples collected",
            canary_color=canary_color,
            baseline_p95_seconds=baseline_p95,
            final_sample=None,
            consecutive_failures=0,
            samples=samples,
        )

    ok, promote_reason = evaluate_promote(final_sample, baseline_p95=baseline_p95)
    if ok:
        return AnalysisResult(
            decision="promote",
            reason=promote_reason,
            canary_color=canary_color,
            baseline_p95_seconds=baseline_p95,
            final_sample=final_sample,
            consecutive_failures=0,
            samples=samples,
        )

    return AnalysisResult(
        decision="rollback",
        reason=promote_reason,
        canary_color=canary_color,
        baseline_p95_seconds=baseline_p95,
        final_sample=final_sample,
        consecutive_failures=consecutive_failures,
        samples=samples,
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Canary analysis for Lextures deploys (plan 17.9)")
    parser.add_argument("--prometheus-url", default="http://localhost:9090")
    parser.add_argument("--canary-color", default="green")
    parser.add_argument("--baseline-color", default="blue")
    parser.add_argument("--window-minutes", type=int, default=10)
    parser.add_argument("--eval-interval-secs", type=int, default=60)
    parser.add_argument("--rollback-streak", type=int, default=3)
    parser.add_argument("--artifact", help="Write JSON analysis artifact to this path")
    parser.add_argument(
        "--fallback-promote-on-unavailable",
        action="store_true",
        help="Exit 0 when Prometheus is unreachable (plan 17.9 risk mitigation)",
    )
    args = parser.parse_args(argv)

    client = PrometheusClient(args.prometheus_url)
    try:
        result = run_analysis(
            client,
            canary_color=args.canary_color,
            baseline_color=args.baseline_color,
            window_minutes=args.window_minutes,
            eval_interval_secs=args.eval_interval_secs,
            rollback_streak=args.rollback_streak,
        )
    except RuntimeError as exc:
        if args.fallback_promote_on_unavailable:
            payload = {"decision": "promote", "reason": f"fallback: {exc}"}
            if args.artifact:
                with open(args.artifact, "w", encoding="utf-8") as fh:
                    json.dump(payload, fh, indent=2)
            print(json.dumps(payload))
            return 0
        print(str(exc), file=sys.stderr)
        return 3

    payload = asdict(result)
    print(json.dumps(payload, indent=2))
    if args.artifact:
        with open(args.artifact, "w", encoding="utf-8") as fh:
            json.dump(payload, fh, indent=2)

    if result.decision == "promote":
        return 0
    if result.decision == "rollback":
        return 1
    return 3


if __name__ == "__main__":
    raise SystemExit(main())
