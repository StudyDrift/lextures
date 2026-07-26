#!/usr/bin/env bash
# TD.3 — OpenAPI contract validity + documentation coverage ratchet.
#
# Verifies:
#   1. server/internal/openapi/openapi.json is valid JSON with no trailing data
#   2. Required OpenAPI 3.0.3 root shape (openapi, info, paths, components)
#   3. components.securitySchemes.bearerAuth is present
#   4. Every local $ref resolves
#   5. Documented path count ≥ scripts/allowlists/openapi-coverage.txt baseline
#   6. Every documented path matches a TD.1 route-inventory pattern
#      (exact / trailing-slash-normalized / chi wildcard prefix)
#
# Usage:
#   scripts/check-openapi-coverage.sh
#   scripts/check-openapi-coverage.sh --self-test
#   make openapi-check
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SPEC="${ROOT}/server/internal/openapi/openapi.json"
BASELINE="${ROOT}/scripts/allowlists/openapi-coverage.txt"
INVENTORY="${ROOT}/server/internal/httpserver/testdata/route_inventory.golden"

SELF_TEST=0
for arg in "$@"; do
  case "$arg" in
    --self-test) SELF_TEST=1 ;;
    -h|--help)
      sed -n '2,20p' "$0" | sed 's/^# \?//'
      exit 0
      ;;
    *)
      echo "Unknown argument: $arg" >&2
      exit 2
      ;;
  esac
done

run_self_test() {
  local failures=0
  local tmp
  tmp="$(mktemp -d)"
  # valid minimal openapi-ish doc
  cat >"$tmp/ok.json" <<'EOF'
{
  "openapi": "3.0.3",
  "info": { "title": "t", "version": "0" },
  "paths": { "/health": { "get": { "responses": { "200": { "description": "ok" } } } } },
  "components": {
    "securitySchemes": {
      "bearerAuth": { "type": "http", "scheme": "bearer" }
    },
    "schemas": {
      "X": { "type": "object" }
    }
  }
}
EOF
  # trailing data
  printf '%s\nTRAILING\n' "$(cat "$tmp/ok.json")" >"$tmp/trail.json"
  # broken ref
  cat >"$tmp/badref.json" <<'EOF'
{
  "openapi": "3.0.3",
  "info": { "title": "t", "version": "0" },
  "paths": {
    "/x": {
      "get": {
        "responses": {
          "200": {
            "description": "ok",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/Missing" }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "securitySchemes": { "bearerAuth": { "type": "http", "scheme": "bearer" } },
    "schemas": {}
  }
}
EOF
  echo "min_documented_paths=1" >"$tmp/base.txt"
  printf 'GET\t/health\tanonymous\n' >"$tmp/inv.golden"

  if ! SPEC="$tmp/ok.json" BASELINE="$tmp/base.txt" INVENTORY="$tmp/inv.golden" \
      python3 "$ROOT/scripts/lib/openapi_check.py"; then
    echo "FAIL: valid fixture should pass"
    failures=$((failures + 1))
  else
    echo "OK: valid fixture passes"
  fi

  if SPEC="$tmp/trail.json" BASELINE="$tmp/base.txt" INVENTORY="$tmp/inv.golden" \
      python3 "$ROOT/scripts/lib/openapi_check.py" 2>/dev/null; then
    echo "FAIL: trailing data should fail"
    failures=$((failures + 1))
  else
    echo "OK: trailing data fails"
  fi

  if SPEC="$tmp/badref.json" BASELINE="$tmp/base.txt" INVENTORY="$tmp/inv.golden" \
      python3 "$ROOT/scripts/lib/openapi_check.py" 2>/dev/null; then
    echo "FAIL: bad \$ref should fail"
    failures=$((failures + 1))
  else
    echo "OK: bad \$ref fails"
  fi

  # coverage decrease
  echo "min_documented_paths=99" >"$tmp/base-high.txt"
  if SPEC="$tmp/ok.json" BASELINE="$tmp/base-high.txt" INVENTORY="$tmp/inv.golden" \
      python3 "$ROOT/scripts/lib/openapi_check.py" 2>/dev/null; then
    echo "FAIL: coverage drop should fail"
    failures=$((failures + 1))
  else
    echo "OK: coverage drop fails"
  fi

  rm -rf "$tmp"
  if [[ "$failures" -ne 0 ]]; then
    echo "self-test FAILED ($failures)"
    exit 1
  fi
  echo "self-test PASSED"
  exit 0
}

if [[ "$SELF_TEST" -eq 1 ]]; then
  run_self_test
fi

export SPEC BASELINE INVENTORY
python3 "$ROOT/scripts/lib/openapi_check.py"
