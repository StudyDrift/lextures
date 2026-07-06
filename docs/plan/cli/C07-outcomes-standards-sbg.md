# C07 — Outcomes, standards & SBG report cards

> CLI parity plan. Source: `courses/{id}/outcomes`, `registerStandardsRoutes`, `registerSBGReportRoutes`, `registerReportCardRoutes`, `course_outcomes_report.go`, `sbg`. Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C07 |
| **Section** | Assessment & grading |
| **Severity** | MAJOR |
| **Markets** | K12 / HE |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Assessment / CLI |
| **Depends on** | C06, C40 |
| **Unblocks** | C27, C31 |

---

## 1. Problem Statement

Standards/outcomes alignment, standards-based grading (SBG) and report cards are core to K-12 and outcomes-driven HE, but none are reachable from the CLI. Districts cannot bulk-load standards frameworks, align outcomes to assignments, or export report cards for their SIS/print pipeline.

## 2. Goals

- Import/align standards and outcomes to courses and assignments in bulk.
- Read outcome mastery and SBG rollups programmatically.
- Generate/export report cards for a section or student.

## 3. Non-Goals

- Authoring the assignment/quiz that an outcome aligns to (C03/C04).
- Transcript generation (C31) — report cards here are term progress reports.

## 4. Personas & User Stories

- **As a district admin**, I want `standards import --file framework.json` to load a standards set.
- **As a curriculum lead**, I want `outcomes align` to map outcomes to assignments in bulk.
- **As a teacher**, I want `report-cards export --section S` to produce printable report cards.
- **As an analyst**, I want `outcomes report <course>` to pull mastery data for BI.

## 5. Functional Requirements

- **FR-1.** MUST add `standards list|import|get` (`registerStandardsRoutes`, `--file` framework).
- **FR-2.** MUST add `outcomes list|create|align|report <course>` (`course_outcomes_report.go`).
- **FR-3.** MUST add `sbg get <course>` and `report-cards list|get|export <course>` (`--section`, `--user`, `--format pdf|csv|json`).
- **FR-4.** SHOULD add `outcomes mastery <course> --user <u>` (student rollup).
- **FR-5.** MAY add `standards align-suggest` (server-suggested alignments) if the endpoint exists.

## 6. Non-Functional Requirements

- **Performance** — report-card export for a section streams; p95 < 3 s per section.
- **Security** — outcomes-manage / report scope; report cards FERPA-covered → `--yes` on bulk export.
- **Privacy & Compliance** — student mastery data is FERPA; redact on shared output.
- **Reliability** — import idempotent by standards code.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a framework file, *When* `standards import`, *Then* standards are created and `standards list` shows them.
- **AC-2.** *Given* aligned outcomes, *When* `outcomes report --json`, *Then* mastery rollups are emitted.
- **AC-3.** *Given* a section, *When* `report-cards export --format pdf --out d`, *Then* PDFs are written with a summary.

## 8. Data Model

- None client-side. Document framework import JSON and report-card CSV schema.

## 9. API Surface

- `standards` list/import; `courses/{c}/outcomes` CRUD + align + report; `sbg`; `report-cards` list/get/export.

## 10. UI / UX

- `lextures standards ...`, `lextures outcomes ...`, `lextures report-cards ...`.
- Export writes files; `--json` for data.

## 11. AI / ML Considerations

- Optional `align-suggest` may be AI-backed server-side; CLI only triggers/reads. No CLI model calls.

## 12. Integration Points

- Server standards/outcomes/SBG/report-card handlers.
- Internal: new command files.

## 13. Dependencies & Sequencing

- After: C06 (grades feed SBG), C40.
- Before: C27 (reports include outcomes), C31.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Framework import format (CASE/IMS) complex | M | M | Support the server's native JSON first; note CASE as follow-up |
| Report-card PDF generation is async | M | M | Return job id; C40 `--wait` |

## 15. Rollout Plan

- Ship standards/outcomes read+import first, then SBG/report-card export.
- Rollback: additive.

## 16. Test Plan

- **Unit** — framework parse; align mapping.
- **Integration** — outcomes report shape; report-card export manifest.
- **E2E** — import→align→export report card.

## 17. Documentation & Training

- "Load a standards framework and align outcomes" recipe.

## 18. Open Questions

1. What framework import format does the server accept (native vs IMS CASE)?
2. Is report-card export synchronous?

## 19. References

- `registerStandardsRoutes`, `registerSBGReportRoutes`, `registerReportCardRoutes`, `course_outcomes_report.go`.
- Related: [C06](C06-gradebook-final-grades.md), [C27](C27-reports-exports.md), [C31](C31-credentials-transcripts-advising.md).
