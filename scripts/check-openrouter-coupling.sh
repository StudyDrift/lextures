#!/usr/bin/env bash
# AP.9 FR-1 / AC-1 — fail CI if production code reintroduces OpenRouter coupling
# outside the allowlisted adapter / dual-read wiring paths.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT/server"

failures=0

allowed_import() {
  case "$1" in
    internal/service/openrouter/*) return 0 ;;
    internal/service/aiprovider/openrouter.go|\
    internal/service/aiprovider/factory.go|\
    internal/service/aiprovider/resolver.go|\
    internal/service/aiprovider/catalog.go|\
    internal/httpserver/server.go|\
    internal/httpserver/settings_ai.go|\
    internal/httpserver/ai_gateway.go|\
    internal/platformstate/platformstate.go|\
    internal/background/ai_provider.go|\
    internal/aidisclosure/disclosure.go|\
    internal/repos/aiusage/aiusage.go) return 0 ;;
  esac
  return 1
}

allowed_client() {
  case "$1" in
    internal/httpserver/server.go|\
    internal/httpserver/ai_configured.go|\
    internal/httpserver/settings_ai.go) return 0 ;;
  esac
  return 1
}

allowed_direct() {
  case "$1" in
    internal/service/openrouter/*) return 0 ;;
    internal/service/aiprovider/openrouter.go|\
    internal/service/aiprovider/catalog.go|\
    internal/service/aiprovider/factory.go|\
    internal/aidisclosure/disclosure.go|\
    internal/platformstate/platformstate.go|\
    internal/background/ai_provider.go|\
    internal/httpserver/server.go) return 0 ;;
  esac
  return 1
}

normalize_rel() {
  local p="$1"
  p="${p#./}"
  printf '%s' "$p"
}

check_matches() {
  local label="$1"
  local pattern="$2"
  local allow_fn="$3"
  local line rel
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    rel="$(normalize_rel "${line%%:*}")"
    case "$rel" in
      *_test.go) continue ;;
    esac
    if "$allow_fn" "$rel"; then
      continue
    fi
    echo "FAIL ($label): $line"
    failures=$((failures + 1))
  done < <(rg -n --glob '*.go' --glob '!*_test.go' "$pattern" . || true)
}

echo "Checking OpenRouter package imports outside allowlist..."
check_matches "import" 'lextures/server/internal/service/openrouter' allowed_import

echo "Checking openRouterClient() call sites outside allowlist..."
check_matches "openRouterClient" 'openRouterClient\(' allowed_client

echo "Checking direct OpenRouter client API calls outside adapter..."
check_matches "direct API" '\.ChatCompletion|\.ChatCompletionStream|\.ChatCompletionVision|\.GenerateImage\(|ListModelsByOutputModality|openrouter\.NewClient' allowed_direct

if [[ "$failures" -gt 0 ]]; then
  echo "$failures OpenRouter coupling check(s) failed (AP.9)."
  echo "Route AI through aiprovider.Resolver, or extend the allowlist with a documented dual-read reason."
  exit 1
fi

echo "OK: OpenRouter coupling allowlist satisfied."
