# C18 — Jobs, scheduler, quarantine & backups

> CLI parity plan. Source: `admin_jobs.go` (`admin/jobs`), `admin_scheduler.go` (`admin/scheduler`), `admin/quarantine`, `backup_ops_http.go`, `av_scan.go`, `admin/lrs-dead-letter`. Baseline: `clients/cli/cmd/jobs.go`, `jobs_ops_logic.go`, `scheduler.go`, `quarantine.go`, `backups.go`, `lrs_deadletter.go`, `jobs_ops_test.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C18 |
| **Section** | Admin & governance |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Platform / SRE |
| **Depends on** | C40 |
| **Unblocks** | C15, C24 |

---

## 1. Problem Statement

Operational surfaces — background job status, scheduled tasks, the AV-scan quarantine, dead-letter queues, and backup operations — are UI-only. SRE and admins cannot monitor or drive jobs from scripts/CI, retry failed items, or trigger/inspect backups, which blocks operational automation and incident response.

## 2. Goals

- Inspect and control background jobs (list, get, retry, cancel) with `--wait`.
- Manage scheduled tasks (list, run-now, enable/disable).
- Triage the AV quarantine and dead-letter queues.
- Trigger and inspect backup operations.

## 3. Non-Goals

- Building the job engine (server concern).
- Infra-level DB backups outside the app's backup-ops surface.

## 4. Personas & User Stories

- **As an SRE**, I want `jobs list --status failed` and `jobs retry <id>`.
- **As an admin**, I want `imports status --wait` (shared with C15) to block until a job finishes.
- **As a scheduler owner**, I want `scheduler run-now <task>` to force a run.
- **As a security admin**, I want `quarantine list` and `quarantine release <id>`.
- **As an SRE**, I want `backups create` and `backups status`.

## 5. Functional Requirements

- **FR-1.** MUST add `jobs list|get|retry|cancel` (`admin_jobs.go`) with `--status` filter and `--wait`.
- **FR-2.** MUST add `scheduler list|run-now|enable|disable` (`admin_scheduler.go`).
- **FR-3.** MUST add `quarantine list|get|release|delete` (`admin/quarantine`, `av_scan.go`).
- **FR-4.** SHOULD add `backups create|status|list` (`backup_ops_http.go`).
- **FR-5.** SHOULD add `dead-letter list|retry` (`lrs-dead-letter`, ties to C26).

## 6. Non-Functional Requirements

- **Performance** — job polling backs off; `--wait` uses server-recommended interval.
- **Security** — ops/admin scope; quarantine release requires `--yes`.
- **Privacy & Compliance** — quarantined files may contain malware/PII; never downloaded to stdout.
- **Reliability** — retry idempotent; cancel safe on already-finished jobs.
- **Observability** — `--wait` streams status transitions; final state → exit code (0 success, 2 failed).
- **Backward compatibility** — additive; `--wait` primitive shared via C40.

## 7. Acceptance Criteria

- **AC-1.** *Given* a running job, *When* `jobs get <id> --wait`, *Then* the command blocks then exits 0 on success / 2 on failure.
- **AC-2.** *Given* a failed job, *When* `jobs retry`, *Then* a new run is queued.
- **AC-3.** *Given* a quarantined file, *When* `quarantine release --yes`, *Then* it is released.

## 8. Data Model

- None client-side.

## 9. API Surface

- `admin/jobs` list/get/retry/cancel; `admin/scheduler`; `admin/quarantine` + `av_scan`; `backup_ops_http.go`; `admin/lrs-dead-letter`.

## 10. UI / UX

- `lextures jobs ...`, `lextures scheduler ...`, `lextures quarantine ...`, `lextures backups ...`.
- `--wait` renders a live status line; `--json` final state.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server job/scheduler/quarantine/backup handlers; shared `--wait` primitive (C40); import jobs (C15), LRS DLQ (C26).

## 13. Dependencies & Sequencing

- After: C40 (`--wait`).
- Before: C15/C24 rely on the shared job-wait primitive.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Long polling ties up CI | M | M | `--wait --timeout`; exit 2 on timeout |
| Quarantine release risk | M | H | `--yes` + audit; never stream file bytes |

## 15. Rollout Plan

- Ship jobs + scheduler first (highest ops value), then quarantine + backups + DLQ.
- Rollback: additive.

## 16. Test Plan

- **Unit** — poll/backoff; exit-code mapping.
- **Integration** — job state transitions; retry.
- **E2E** — submit import (C15) → `jobs --wait` → success.

## 17. Documentation & Training

- Runbook: "Monitor and retry background jobs from CI."

## 18. Open Questions

1. Does the server expose a recommended poll interval / job progress percentage?

## 19. References

- `admin_jobs.go`, `admin_scheduler.go`, `backup_ops_http.go`, `av_scan.go`.
- Related: [C15](C15-people-provisioning.md), [C24](C24-canvas-content-import.md), [C26](C26-xapi-lrs-engagement.md), [C40](C40-cli-framework.md).
