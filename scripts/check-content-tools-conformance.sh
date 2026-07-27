#!/usr/bin/env bash
# CT.8 — Content Tools conformance gate (CI job: tools-conformance).
# Fails when any registered tool is missing a data sheet, projection, i18n, or a11y declaration.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export PATH="/usr/local/go/bin:${PATH:-}"
cd "$ROOT/server"
go test ./internal/service/contenttools/ -count=1 -short -run 'TestConformanceGate_BuiltinsPass|TestRegistryContract|TestValidateDataSheet_Required' -timeout=2m
echo "content tools conformance gate: OK"
