# C12 — Attendance, behavior & seat-time

> CLI parity plan. Source: `attendance_http.go` (`courses/{id}/attendance`), `behavior_http.go` (`behavior`, `pbis`), `registerSeatTimeRoutes` (`seat-time`), hall pass. Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C12 |
| **Section** | Roster & classroom |
| **Severity** | MAJOR |
| **Markets** | K12 (primary) / HE |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | K12 / CLI |
| **Depends on** | C11, C40 |
| **Unblocks** | C27, C28 |

---

## 1. Problem Statement

Attendance, behavior/PBIS records and seat-time tracking are UI-only. K-12 districts that must report attendance to the state cannot export it via CLI, and cannot bulk-import attendance captured by another system (e.g. a door scanner).

## 2. Goals

- Record and export attendance per course/section/date.
- Read behavior/PBIS points and incidents; export for reporting.
- Pull seat-time reports for compliance (e.g. state seat-time mandates).

## 3. Non-Goals

- Real-time hall-pass issuance UX (browser/mobile flow) beyond a simple issue/return command.
- Discipline case management workflow.

## 4. Personas & User Stories

- **As an attendance clerk**, I want `attendance import --file day.csv` to bulk-record a day.
- **As a state reporter**, I want `attendance export --course C --from --to` for the ADA report.
- **As a dean**, I want `behavior export --course C` to pull PBIS points/incidents.
- **As a compliance officer**, I want `seat-time report --course C` for seat-time mandates.

## 5. Functional Requirements

- **FR-1.** MUST add `attendance list|record|import|export <course>` (`--date`, `--user`, `--status present|absent|tardy|excused`, `--file`).
- **FR-2.** MUST add `behavior list|export <course>` and `behavior award --user U --points N` (PBIS).
- **FR-3.** MUST add `seat-time report <course> [--user]` (`courses/{id}/seat-time-report`, `me/seat-time`).
- **FR-4.** SHOULD add `hall-pass issue|return|list` if endpoints exist.
- **FR-5.** SHOULD add `attendance summary <course>` (rollup per student).

## 6. Non-Functional Requirements

- **Performance** — day/section import chunked; export streamed.
- **Security** — attendance/behavior scope; K12 role gating server-side.
- **Privacy & Compliance** — attendance/behavior are FERPA (and sometimes state-reported) records → export gated by `--yes`; COPPA for under-13.
- **Reliability** — record/import idempotent per (student, date, period).
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a day CSV, *When* `attendance import`, *Then* records post; re-import updates in place, no dups.
- **AC-2.** *Given* a date range, *When* `attendance export --json`, *Then* records are emitted for reporting.
- **AC-3.** *Given* a course, *When* `seat-time report`, *Then* per-student minutes print.

## 8. Data Model

- None client-side. Document attendance CSV (student, date, period, status).

## 9. API Surface

- `courses/{c}/attendance` CRUD/import/export; `behavior`/`pbis`; `seat-time`/`seat-time-report`; hall-pass endpoints.

## 10. UI / UX

- `lextures attendance ...`, `lextures behavior ...`, `lextures seat-time ...`.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server attendance/behavior/seat-time handlers; state-reporting export (C27).

## 13. Dependencies & Sequencing

- After: C11 (roster), C40.
- Before: C27/C28 (attendance feeds reports/at-risk).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Period/block model varies by district | M | M | Support `--period` optional; default single daily record |
| State export formats differ | M | M | Provide neutral CSV/JSON; state-specific formatting in C27 |

## 15. Rollout Plan

- Ship attendance record/export first, then behavior/seat-time.
- Rollback: additive.

## 16. Test Plan

- **Unit** — status enum; idempotency key.
- **Integration** — import summary; export range filter.
- **E2E** — record attendance → export → verify.

## 17. Documentation & Training

- "Export attendance for state reporting" recipe.

## 18. Open Questions

1. Does the model support periods/blocks or only daily attendance?
2. Is hall-pass exposed via REST?

## 19. References

- `attendance_http.go`, `behavior_http.go`, `registerSeatTimeRoutes`.
- Related: [C11](C11-enrollments-sections.md), [C27](C27-reports-exports.md), [C28](C28-insights-at-risk.md).
