#!/usr/bin/env bash
# TD.5 FR-8 — flag new in-handler method dispatch on single-method handlers.
#
# The chi router owns method dispatch for single-method routes (r.Get/Post/…).
# OPTIONS preflight is handled by corsAll. In-handler MethodOptions /
# MethodNotAllowed prologues on single-method handlers are forbidden.
#
# Multi-method handlers (registered under more than one verb, or Handle/HandleFunc)
# may keep dispatch; their files are listed below.
#
# Usage:
#   scripts/check-handler-method-dispatch.sh
#   scripts/check-handler-method-dispatch.sh --report
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

REPORT=0
for arg in "$@"; do
  case "$arg" in
    --report) REPORT=1 ;;
    -h|--help)
      sed -n '2,16p' "$0" | sed 's/^# \?//'
      exit 0
      ;;
  esac
done

# FR-4 / FR-6 residuals + central infrastructure (must keep method checks).
ALLOW_OPTIONS=(
  server/internal/httpserver/cors.go
  server/internal/httpserver/not_found_response.go
  server/internal/httpserver/unimplemented_v1.go
  server/internal/httpserver/course_sections.go
  server/internal/httpserver/assignment_overrides_http.go
  server/internal/httpserver/calendar_http.go
)

# Files that may still contain StatusMethodNotAllowed / Method != (multi-method or helpers).
ALLOW_METHOD_CHECK=(
  server/internal/httpserver/cors.go
  server/internal/httpserver/not_found_response.go
  server/internal/httpserver/unimplemented_v1.go
  server/internal/httpserver/calendar_http.go
  server/internal/httpserver/auth.go
  server/internal/httpserver/mobile_link_policy_http.go
  server/internal/httpserver/course_sections.go
  server/internal/httpserver/assignment_overrides_http.go
  server/internal/httpserver/scim_http.go
  server/internal/httpserver/attendance_http.go
  server/internal/httpserver/behavior_http.go
  server/internal/httpserver/library_http.go
  server/internal/httpserver/admin.go
  server/internal/httpserver/admin_console.go
  server/internal/httpserver/admin_orgs.go
  server/internal/httpserver/admin_org_units.go
  server/internal/httpserver/admin_jobs.go
  server/internal/httpserver/org_branding_http.go
  server/internal/httpserver/org_role_grants.go
  server/internal/httpserver/org_type_http.go
  server/internal/httpserver/tax_http.go
  server/internal/httpserver/sis_http.go
  server/internal/httpserver/report_cards_http.go
  server/internal/httpserver/content_filter_http.go
  server/internal/httpserver/content_tools_governance.go
  server/internal/httpserver/broadcasts_http.go
  server/internal/httpserver/demographics_http.go
  server/internal/httpserver/parent_http.go
  server/internal/httpserver/board_admin_http.go
  server/internal/httpserver/quizgame_admin.go
  server/internal/httpserver/course_whiteboard.go
  server/internal/httpserver/course_evaluations_admin.go
  server/internal/httpserver/support_widget_http.go
  server/internal/httpserver/saml_lti.go
)

in_list() {
  local needle=$1
  shift
  local x
  for x in "$@"; do
    [[ "$x" == "$needle" ]] && return 0
  done
  return 1
}

violations=0

while IFS= read -r f; do
  [[ "$f" == *_test.go ]] && continue
  if rg -q 'r\.Method == http\.MethodOptions' "$f"; then
    if ! in_list "$f" "${ALLOW_OPTIONS[@]}"; then
      echo "TD.5: unexpected MethodOptions check in $f"
      violations=$((violations + 1))
    fi
  fi
  if rg -q 'r\.Method != http\.Method|StatusMethodNotAllowed' "$f"; then
    if ! in_list "$f" "${ALLOW_METHOD_CHECK[@]}"; then
      # Ignore pure status constants in comments / unrelated uses of the status name
      # only when there is an actual method comparison or Allow+405 pattern.
      if rg -q 'r\.Method != http\.Method|jobsMethodNotAllowed|http\.Error\(w, http\.StatusText\(http\.StatusMethodNotAllowed\)' "$f"; then
        echo "TD.5: unexpected in-handler method dispatch in $f"
        violations=$((violations + 1))
      fi
    fi
  fi
done < <(find server/internal/httpserver -name '*.go' ! -name '*_test.go' | sort)

# FR-5 map integrity: single-method claim never overlaps multi-method.
python3 scripts/analyze-handler-methods.py --assert-single-ok

if [[ "$REPORT" -eq 1 ]]; then
  echo "handler-method-dispatch violations: $violations"
  opt=$(rg -c 'r\.Method == http\.MethodOptions' server/internal/httpserver --glob '*.go' --glob '!*_test.go' | awk -F: '{s+=$2} END {print s+0}')
  mna=$(rg -c 'StatusMethodNotAllowed' server/internal/httpserver --glob '*.go' --glob '!*_test.go' | awk -F: '{s+=$2} END {print s+0}')
  echo "residual MethodOptions (non-test): $opt"
  echo "residual StatusMethodNotAllowed (non-test): $mna"
  exit 0
fi

if [[ "$violations" -gt 0 ]]; then
  echo "TD.5 FAIL: $violations file(s) reintroduced unreachable method dispatch." >&2
  echo "Handlers registered with a single r.Get/Post/… must not check r.Method." >&2
  echo "Multi-method handlers need an allowlist entry + explanatory comment (FR-4/FR-6)." >&2
  exit 1
fi
echo "TD.5 OK: no new single-method in-handler method dispatch"
