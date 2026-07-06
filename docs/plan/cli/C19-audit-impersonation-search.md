# C19 — Audit log, impersonation & admin search

> CLI parity plan. Source: `admin_audit_log_http.go` (`admin/audit-log`, `compliance/audit-log`), `registerImpersonationRoutes` (`admin-console/impersonate`), `admin_search.go` (`admin/search`), `impersonationWriteBlockMiddleware`. Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C19 |
| **Section** | Admin & governance |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Security / CLI |
| **Depends on** | C40 |
| **Unblocks** | C29 |

---

## 1. Problem Statement

The admin audit log, impersonation (act-as), and admin search are UI-only. Security/compliance teams cannot pull audit trails for SIEM ingestion or investigations, and support staff cannot script "act as user" flows for reproducing issues.

## 2. Goals

- Export the audit log with filters for SIEM/compliance.
- Start/stop impersonation sessions for support (read-only by default — writes are server-blocked).
- Run admin search to locate users/courses/orgs quickly.

## 3. Non-Goals

- Building SIEM integration (this provides the export; C25/C26 handle streaming).
- Bypassing impersonation write-block (server enforces).

## 4. Personas & User Stories

- **As a compliance officer**, I want `audit-log export --from --to --actor U` for an investigation.
- **As support**, I want `impersonate start --user U` to reproduce a bug (read-only).
- **As an admin**, I want `admin search "jane"` to find a user/course/org fast.

## 5. Functional Requirements

- **FR-1.** MUST add `audit-log list|export` (`--actor`, `--action`, `--from`, `--to`, `--target`; CSV/JSON).
- **FR-2.** MUST add `impersonate start|stop|whoami` (`admin-console/impersonate`); CLI clearly labels impersonated sessions and honors the server write-block.
- **FR-3.** MUST add `admin search <query> [--type user|course|org]` (`admin_search.go`).
- **FR-4.** SHOULD add `audit-log tail` (poll for new entries) for near-real-time monitoring.

## 6. Non-Functional Requirements

- **Performance** — audit export paginated/streamed; tail backs off.
- **Security** — audit-view / impersonation scope; impersonation tokens stored separately and never written to the default profile.
- **Privacy & Compliance** — audit entries + impersonation are themselves audited (SOC 2); export gated by `--yes`.
- **Reliability** — impersonation `stop` always restores the real identity; CLI warns if a session is active.
- **Observability** — `whoami` shows real vs effective identity.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a range, *When* `audit-log export --json`, *Then* entries stream to stdout.
- **AC-2.** *Given* impersonation active, *When* a write command runs, *Then* the server blocks it and the CLI surfaces the block.
- **AC-3.** *Given* `impersonate whoami`, *Then* both real and effective users print.

## 8. Data Model

- Client stores impersonation token transiently (separate from `~/.lextures` profile token).

## 9. API Surface

- `admin/audit-log` + `compliance/audit-log` list/export; `admin-console/impersonate` start/stop; `admin/search`.

## 10. UI / UX

- `lextures audit-log ...`, `lextures impersonate ...`, `lextures admin search`.
- Impersonated sessions render a persistent banner-style notice in output.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server audit/impersonation/search handlers; `impersonationWriteBlockMiddleware`.

## 13. Dependencies & Sequencing

- After: C40.
- Before: C29 (compliance exports reuse audit-log export).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Forgotten active impersonation | M | H | `whoami` warns; auto-expire; `stop` idempotent |
| Audit export volume | M | M | Streamed pagination + date-range required |

## 15. Rollout Plan

- Ship audit-log export + admin search first, then impersonation.
- Rollback: additive.

## 16. Test Plan

- **Unit** — filter param building; identity display.
- **Integration** — export pagination; impersonation start/stop token handling.
- **Security** — write-block honored; token isolation.
- **E2E** — export audit → verify fields.

## 17. Documentation & Training

- "Export audit logs for your SIEM" runbook.

## 18. Open Questions

1. Does impersonation issue a distinct token, or a session header?

## 19. References

- `admin_audit_log_http.go`, `admin_search.go`, `registerImpersonationRoutes`.
- Related: [C16](C16-roles-permissions.md), [C29](C29-compliance-privacy.md).
