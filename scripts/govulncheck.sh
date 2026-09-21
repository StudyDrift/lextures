#!/usr/bin/env bash
# Run govulncheck in the current module and fail on called vulnerabilities.
#
# GO-2026-6452 (excelize negative shared-string index) has no fixed version in
# the Go vuln DB, but the bounds check from
# https://github.com/qax-os/excelize/commit/93f0b3caed37f21ef5079e3259c6c21dcfe68453
# is present in github.com/xuri/excelize/v2 v2.11.0. Called findings for that
# ID are allowed only when the selected module is v2.11.0 or newer.
#
# Usage (from a module directory):
#   scripts/govulncheck.sh
#   scripts/govulncheck.sh --self-test
set -euo pipefail

if [[ "${1:-}" == "--self-test" ]]; then
  python3 - "$0" <<'PY'
import json, os, subprocess, sys, tempfile

script = sys.argv[1]
cases = {
    "allowed": {
        "finding": {
            "osv": "GO-2026-6452",
            "trace": [
                {"module": "github.com/xuri/excelize/v2", "version": "v2.11.0", "function": "File.GetRows"},
                {"module": "example.com/app", "version": "v0.0.0", "function": "main"},
            ],
        }
    },
    "old": {
        "finding": {
            "osv": "GO-2026-6452",
            "trace": [
                {"module": "github.com/xuri/excelize/v2", "version": "v2.10.2", "function": "File.GetRows"},
            ],
        }
    },
    "other": {
        "finding": {
            "osv": "GO-2026-0000",
            "trace": [
                {"module": "example.com/mod", "version": "v1.2.3", "function": "Bad"},
            ],
        }
    },
    "uncalled": {
        "finding": {
            "osv": "GO-2026-0000",
            "trace": [{"module": "example.com/mod", "version": "v1.2.3"}],
        }
    },
}

def run(payload):
    with tempfile.NamedTemporaryFile("w", delete=False) as fh:
        json.dump(payload, fh)
        name = fh.name
    env = os.environ.copy()
    env["GOVULNCHECK_JSON"] = name
    result = subprocess.run(["bash", script, "--filter-only"], env=env, capture_output=True, text=True)
    os.unlink(name)
    return result.returncode

assert run(cases["allowed"]) == 0, "patched excelize should pass"
assert run(cases["uncalled"]) == 0, "uncalled findings should pass"
assert run(cases["old"]) != 0, "excelize before v2.11.0 should fail"
assert run(cases["other"]) != 0, "other called vulns should fail"
print("govulncheck filter self-test ok")
PY
  exit 0
fi

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

if [[ "${1:-}" == "--filter-only" ]]; then
  cp "${GOVULNCHECK_JSON:?}" "$tmp"
  status=3
else
  set +e
  go run golang.org/x/vuln/cmd/govulncheck@latest -json ./... >"$tmp"
  status=$?
  set -e
  if [[ "$status" -ne 0 && "$status" -ne 3 ]]; then
    echo "govulncheck failed to run (exit $status)" >&2
    exit "$status"
  fi
fi

python3 - "$tmp" <<'PY'
import json
import sys

ALLOW = {
    # Shared-string panic is fixed in v2.11.0; the vuln DB has no fixed version.
    "GO-2026-6452": ("github.com/xuri/excelize/v2", (2, 11, 0)),
}

def version_at_least(version, minimum):
    raw = version[1:] if version.startswith("v") else version
    core, _, pre = raw.partition("-")
    parts = []
    for piece in core.split(".")[:3]:
        if not piece.isdigit():
            return False
        parts.append(int(piece))
    while len(parts) < 3:
        parts.append(0)
    got = tuple(parts)
    if got > minimum:
        return True
    if got < minimum:
        return False
    return pre == ""

decoder = json.JSONDecoder()
data = open(sys.argv[1], encoding="utf-8").read()
index = 0
failures = []
allowed = []
while index < len(data):
    while index < len(data) and data[index].isspace():
        index += 1
    if index >= len(data):
        break
    try:
        obj, index = decoder.raw_decode(data, index)
    except json.JSONDecodeError as err:
        sys.stderr.write(f"govulncheck JSON parse error: {err}\n")
        sys.exit(1)
    finding = obj.get("finding") if isinstance(obj, dict) else None
    if not finding:
        continue
    trace = finding.get("trace") or []
    if not trace or not trace[0].get("function"):
        continue
    osv = finding.get("osv") or ""
    module = trace[0].get("module") or ""
    version = trace[0].get("version") or ""
    rule = ALLOW.get(osv)
    if rule and module == rule[0] and version_at_least(version, rule[1]):
        allowed.append(f"{osv} {module}@{version}")
        continue
    failures.append(f"{osv} {module}@{version} {trace[0].get('function')}")

for item in allowed:
    print(f"allowed known-fixed finding: {item}")
if failures:
    print("called vulnerabilities:", file=sys.stderr)
    for item in failures:
        print(f"  {item}", file=sys.stderr)
    sys.exit(1)
print("govulncheck: no unfixed called vulnerabilities")
PY
