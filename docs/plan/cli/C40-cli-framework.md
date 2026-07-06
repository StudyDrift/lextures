# C40 — CLI framework & ergonomics

> Foundational plan (cross-cutting). Source: `clients/cli/cmd/root.go`, `internal/client/client.go`, `internal/config`, `internal/auth`. Enables every other C-plan.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C40 |
| **Section** | CLI framework |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | THIN (basic Cobra + JSON/table + retry) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Platform / CLI |
| **Depends on** | — |
| **Unblocks** | all C-plans |

---

## 1. Problem Statement

The CLI has good bones (Cobra, profiles, `--json`, retry, keychain auth) but lacks the cross-cutting primitives every new command needs: consistent pagination, async **job wait**, server-side vs client-side output shaping, `--file`/stdin input, `--dry-run`/`--yes` gating, bulk-op summaries, streaming (SSE/WebSocket), and shell completion. Building these once, well, prevents 39 inconsistent re-implementations.

## 2. Goals

- Standardize output (table/JSON/NDJSON/CSV), pagination, and error handling.
- Provide shared primitives: `--wait` for jobs, `--file`/stdin, `--dry-run`, `--yes`, idempotency keys, bulk summaries.
- Add a streaming (SSE/WebSocket) helper for tutor/grading dry-run/audit-tail.
- Ship shell completion, `--output` templating, and robust config/profile UX.

## 3. Non-Goals

- Any specific domain command (those are C01–C39).
- Rewriting the auth store (already solid) beyond adding access-key support hooks.

## 4. Personas & User Stories

- **As a CLI author**, I want a `runList`/`runGet`/`runMutate` helper set so new commands are 20 lines.
- **As a scripter**, I want `--output json|table|csv|ndjson` and `--jq`/`--template` for shaping.
- **As a CI engineer**, I want `--wait --timeout` to block on async jobs with proper exit codes.
- **As a user**, I want `completion bash|zsh|fish` and `--no-color`/`--quiet`.

## 5. Functional Requirements

- **FR-1.** MUST add a shared **output layer**: `--output table|json|ndjson|csv`, `--no-headers`, `--quiet`, `--no-color`, honoring existing `--json` as an alias.
- **FR-2.** MUST add **pagination** helpers (cursor + page/limit) with `--all` to auto-follow pages.
- **FR-3.** MUST add a **job `--wait`** primitive (poll or stream; `--timeout`; exit 0 success / 2 failure) reused by C15/C18/C22/C24/C27/C29/C33.
- **FR-4.** MUST add **input helpers**: `--file <path|->` (JSON/YAML/CSV) and consistent stdin handling.
- **FR-5.** MUST add **safety gates**: `--dry-run` (server preview where supported), `--yes`/`--force` for destructive/FERPA-export ops, and an idempotency-key header helper.
- **FR-6.** MUST add **bulk-op summary** rendering (created/updated/skipped/failed with row errors + non-zero exit on any failure unless `--continue-on-error`).
- **FR-7.** SHOULD add a **streaming helper** (SSE + WebSocket) for C09/C19/C36.
- **FR-8.** SHOULD add **shell completion** (`completion` command) and dynamic completion for ids where cheap.
- **FR-9.** SHOULD add `--server`/profile UX niceties: `config set/get/list`, `env` diagnostics, `whoami` (alias to C39 `me`).
- **FR-10.** SHOULD standardize **error envelope** parsing so `--json` errors are consistent (`{error, code, request_id}`) and human errors include remediation hints.

## 6. Non-Functional Requirements

- **Performance** — pagination auto-follow bounded by `--limit`; streaming renders incrementally; retries use existing `doWithRetry` with backoff.
- **Security** — secrets (keys, tokens, provider creds) never printed to stdout by default nor logged; `--file`/stdin for secret input; redaction utility used everywhere.
- **Privacy & Compliance** — a shared `confirmSensitiveExport()` gate powers every FERPA/financial/PII export across plans (`--yes`).
- **Reliability** — one HTTP client config (timeouts, retry, `User-Agent: lextures-cli/<ver>`); consistent exit codes (0/1/2) across all commands.
- **Observability** — optional `--verbose`/`LEXTURES_DEBUG` request logging (redacted); every request carries a client request id.
- **Maintainability** — helpers live in `internal/cli` (new) so C01–C39 depend on them; a linter/test asserts new commands use them.
- **Internationalization** — timezone/locale flags (`--tz`) resolved once and shared.
- **Backward compatibility** — existing flags (`--json`, `--server`, `--profile`, `--api-key`, `--config`) and exit codes preserved; `--json` remains an alias of `--output json`.

## 7. Acceptance Criteria

- **AC-1.** *Given* any list command, *When* `--all`, *Then* all pages are followed transparently.
- **AC-2.** *Given* an async job, *When* `--wait --timeout 60`, *Then* the command blocks then exits 0/2 by outcome (or 2 on timeout).
- **AC-3.** *Given* a destructive command without `--yes`, *Then* it refuses with a clear prompt-to-confirm message.
- **AC-4.** *Given* `completion zsh`, *Then* a valid completion script is emitted.
- **AC-5.** *Given* a secret-bearing create, *Then* the secret never appears in default output or debug logs.

## 8. Data Model

- No server changes. New client package `internal/cli` (output, paginate, wait, input, gates, stream).

## 9. API Surface

- No new routes. Consumes existing job/status endpoints for `--wait`; SSE/WS endpoints for streaming.

## 10. UI / UX

- Consistent table/JSON/CSV; color auto-off when non-TTY; `--quiet` for scripts; uniform error format; `completion` + `config` helper commands.

## 11. AI / ML Considerations

- Streaming helper underpins AI features (C09, C36); usage/cost display convention defined here.

## 12. Integration Points

- Refactors `clients/cli/cmd/root.go`, `internal/client/client.go`; new `internal/cli` package used by all commands.

## 13. Dependencies & Sequencing

- After: none — this is the foundation.
- Before: all of C01–C39 (they consume these primitives). Ship first.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Refactor churn across existing commands | M | M | Introduce helpers additively; migrate existing commands incrementally with tests |
| Streaming (WS) complexity | M | M | Start with SSE/polling; add WS when a consumer (C09/C36) lands |
| Over-abstraction | M | M | Keep helpers thin; driven by two real consumers before generalizing |

## 15. Rollout Plan

- Phase 1: output/pagination/input/gates/exit-codes + migrate existing 9 commands.
- Phase 2: `--wait` job primitive + bulk summary.
- Phase 3: streaming + completion + config helpers.
- Rollback: helpers are additive; existing behavior preserved throughout.

## 16. Test Plan

- **Unit** — output formatters; pagination follow; wait/backoff; gate logic; redaction.
- **Integration** — `--wait` against a mock job endpoint; `--all` pagination; SSE stream parse.
- **Security** — secret redaction across output/debug; `--yes` gating.
- **E2E** — migrate `courses list` to the new output layer with unchanged observable output (golden test).

## 17. Documentation & Training

- CLI conventions doc (flags, exit codes, scripting patterns); "Writing a new command" contributor guide.

## 18. Open Questions

1. Do async endpoints expose a uniform job-status shape, or per-domain? (Drives `--wait` abstraction.)
2. SSE vs WebSocket for streaming endpoints — is it consistent server-side?
3. Adopt YAML input in addition to JSON/CSV?

## 19. References

- `clients/cli/cmd/root.go`, `clients/cli/internal/client/client.go`, `internal/config`, `internal/auth`.
- Related: consumed by [C01](C01-courses.md)–[C39](C39-profile-account-personas.md); especially [C15](C15-people-provisioning.md), [C18](C18-jobs-scheduler-backups.md), [C27](C27-reports-exports.md), [C09](C09-ai-grading-agents.md), [C36](C36-tutor-study-buddy.md).
