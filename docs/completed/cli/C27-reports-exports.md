# C27 — Reports & exports

> CLI parity plan. Source: `/api/v1/reports/learning-activity`, `report_export.go` (`registerReportExportRoutes`), `courses/{id}/analytics` (12), `/api/v1/analytics`, `admin_search.go`, `courses/{id}/reports`. Baseline: `clients/cli/cmd/reports.go`, `analytics.go`, `reports_exports_logic.go`, `reports_exports_test.go` (plus existing `grades export`).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C27 |
| **Section** | Reporting & insights |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Analytics / CLI |
| **Depends on** | C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

Reporting and data export — the classic reason institutions want a CLI — is almost absent. Beyond `grades export`, there is no way to pull learning-activity reports, course analytics, or run the report-export pipeline into CSV/JSON for a data warehouse. Data teams cannot schedule nightly extracts.

## 2. Goals

- Run and export the platform's canonical reports (learning activity, course analytics) to CSV/JSON.
- Drive the async report-export pipeline (submit → wait → download).
- Make every report scriptable for warehouse ETL/cron.

## 3. Non-Goals

- Building new report types (server-owned).
- BI visualization.

## 4. Personas & User Stories

- **As a data engineer**, I want `reports export learning-activity --from --to --out d` for nightly ETL.
- **As an analyst**, I want `analytics course <course> --json` for a course dashboard feed.
- **As an admin**, I want `reports run <type> --wait` to generate large exports.

## 5. Functional Requirements

- **FR-1.** MUST add `reports list` (available report types) and `reports run <type>` / `reports export <type>` (`registerReportExportRoutes`, async + `--wait`).
- **FR-2.** MUST add `reports learning-activity [--course|--org] --from --to` (`/reports/learning-activity`).
- **FR-3.** MUST add `analytics course <course>` and `analytics platform` (`courses/{id}/analytics`, `/analytics`).
- **FR-4.** SHOULD support `--format csv|json|ndjson` and `--out <dir>` for all exports.
- **FR-5.** SHOULD add `reports schedule` passthrough if the server supports scheduled exports.

## 6. Non-Functional Requirements

- **Performance** — large exports stream; async path with `--wait` for heavy reports.
- **Security** — reports scope (`global:app:reports:view` etc.); 403 → exit 2.
- **Privacy & Compliance** — reports contain student PII/grades (FERPA); export gated by `--yes`; PII redaction option where the server supports it.
- **Reliability** — export idempotent; resumable download of generated artifacts.
- **Observability** — `--wait` shows generation progress; row/record counts on completion.
- **Backward compatibility** — keep `grades export`; consider aliasing under `reports`.

## 7. Acceptance Criteria

- **AC-1.** *Given* a range, *When* `reports export learning-activity --out d`, *Then* a CSV lands in `d/` with a row count.
- **AC-2.** *Given* a heavy report, *When* `reports run <type> --wait`, *Then* it completes and downloads.
- **AC-3.** *Given* `analytics course --json`, *Then* analytics metrics emit as JSON.

## 8. Data Model

- None client-side.

## 9. API Surface

- `registerReportExportRoutes`; `/reports/learning-activity`; `courses/{id}/analytics` + `/analytics`; `courses/{id}/reports`.

## 10. UI / UX

- `lextures reports ...`, `lextures analytics ...`. CSV default for exports; `--json`/`--ndjson` for pipelines.

## 11. AI / ML Considerations

- None (insights/at-risk AI lives in C28).

## 12. Integration Points

- Server report/analytics handlers; job queue (`--wait`, C18/C40).

## 13. Dependencies & Sequencing

- After: C40 (`--wait`, `--out`, `--format`).
- Before: none (but complements C06/C07/C12 data producers).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Report catalog undiscoverable | M | M | `reports list` enumerates types + params |
| Huge exports | M | M | Streamed + async + resumable download |

## 15. Rollout Plan

- Ship learning-activity + course analytics + report-export pipeline first.
- Rollback: additive.

## 16. Test Plan

- **Unit** — format/out flags; range params.
- **Integration** — export job submit/poll/download.
- **Security** — reports scope; `--yes` gate.
- **E2E** — nightly ETL simulation into a temp dir.

## 17. Documentation & Training

- "Schedule nightly report exports to your warehouse" runbook.

## 18. Open Questions

1. Is there a discoverable catalog of report types + parameters?
2. Which reports are sync vs async?

## 19. References

- `report_export.go`, `courses/{id}/analytics` handlers, `/reports/learning-activity`.
- Related: [C06](C06-gradebook-final-grades.md), [C12](C12-attendance-behavior.md), [C28](C28-insights-at-risk.md), [C40](C40-cli-framework.md).
