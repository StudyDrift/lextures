# SEC-26 — No secret scanning / SAST in CI

- **Severity:** Informational
- **Status:** Partially addressed
- **Area:** CI
- **File:** [.github/workflows/ci.yml](../../.github/workflows/ci.yml)

## Problem

CI already runs `govulncheck` (Go vuln DB) and `npm audit --audit-level=high` (JS dependencies) — good, and worth keeping. What's missing:

- **Secret scanning.** No `gitleaks` / `trufflehog` step. The repo history has previously contained live-looking API keys in a local `.env` (rotated since), exactly the class of leak a pre-commit/CI scanner catches before it lands.
- **Go SAST.** No `gosec` / `staticcheck -checks=SA*` to catch the kinds of issues in this audit (unsafe file serving, missing timeouts, weak crypto) at PR time.

## Risk

Informational — process gap rather than a live vulnerability. Without secret scanning, the next pasted token or `.env` is caught only by luck. Without SAST, regressions of the findings in this folder can re-enter silently.

## Fix

1. Add a `gitleaks` (or `trufflehog`) job to `ci.yml` and a pre-commit hook; fail the build on findings.
2. Add `gosec ./...` and `staticcheck` jobs; start non-blocking, then gate on high-severity once the backlog is clean.
3. Keep Dependabot/renovate current for `go.mod` and `package.json`.

## Verification

- A PR that adds a fake AWS key is blocked by the secret-scanning job.
- `gosec` runs on every PR and reports findings.
