# C06 — Gradebook & final grades (expand)

> CLI parity plan. Source: `courses/{id}` `gradebook`, `grading`, `grading-scheme`, `grading-backlog`, `final-grades`, `registerFinalGradeRoutes`, `curves`, what-if. Baseline: `clients/cli/cmd/grades.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C06 |
| **Section** | Assessment & grading |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | PARTIAL (list, update, export) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Assessment / CLI |
| **Depends on** | C03, C04, C40 |
| **Unblocks** | C07, C31 |

---

## 1. Problem Statement

The CLI can list grades, update a single score and export CSV, but cannot read the full gradebook matrix, post grades in bulk, manage the grading scheme, work the grading backlog, apply curves, or finalize/submit final grades. Registrars and instructors cannot script end-of-term grade posting or curve application — the highest-value grading automation.

## 2. Goals

- Bulk grade import/export round-trip (edit CSV offline → post).
- Read the whole gradebook as a matrix (students × items).
- Manage grading scheme and grading backlog from the terminal.
- Apply curves and finalize/submit final grades.

## 3. Non-Goals

- Rubric/outcome authoring (see C07).
- AI grading (see C09).

## 4. Personas & User Stories

- **As an instructor**, I want `gradebook export` → edit → `gradebook import` to post many grades at once.
- **As a registrar**, I want `final-grades submit <course>` to finalize the term.
- **As an instructor**, I want `grades curve` to apply a curve and preview the effect.
- **As a TA**, I want `grading-backlog list` to see what still needs grading.

## 5. Functional Requirements

- **FR-1.** MUST add `gradebook get <course>` (matrix; `--json` and CSV).
- **FR-2.** MUST add `gradebook import <course> --file grades.csv` (bulk upsert) and keep existing `grades export`.
- **FR-3.** MUST add `grades scheme get|set <course>` (grading-scheme).
- **FR-4.** MUST add `final-grades list|set|submit <course>` (`registerFinalGradeRoutes`).
- **FR-5.** SHOULD add `grades curve <course> --assignment <a> --method <linear|sqrt|...>` with `--dry-run`.
- **FR-6.** SHOULD add `grading-backlog list <course>` and `grades what-if` (student projection).
- **FR-7.** MAY add `grades history <course> --assignment <a> --user <u>` (assignment_grade_history).

## 6. Non-Functional Requirements

- **Performance** — gradebook matrix for large sections paginated/streamed; p95 < 2 s.
- **Security** — grade-manage scope; final-grade submit may require elevated role; 403 → exit 2.
- **Privacy & Compliance** — grades are FERPA records; bulk export gated by `--yes`; audit noted server-side.
- **Reliability** — bulk import is transactional per row with a summary; idempotent re-post.
- **Observability** — import prints posted/failed counts with row-level errors.
- **Backward compatibility** — existing `grades list/update/export` unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* a CSV of scores, *When* `gradebook import`, *Then* scores post and a summary shows counts.
- **AC-2.** *Given* a course, *When* `final-grades submit`, *Then* final grades are locked/submitted (verify `final-grades list`).
- **AC-3.** *Given* `grades curve --dry-run`, *Then* projected scores print and nothing changes.

## 8. Data Model

- None client-side. Document the gradebook CSV column contract (student id, item id/title, score).

## 9. API Surface

- `GET .../gradebook`; bulk grade post; `GET|PUT .../grading-scheme`; `final-grades` list/set/submit; `curves`; `grading-backlog`; grade-history.

## 10. UI / UX

- Extend `gradesCmd`; add `gradebook` and `final-grades` sub-groups.
- CSV is the default interchange for bulk; `--json` for programmatic.

## 11. AI / ML Considerations

- None (AI grading is C09; what-if is deterministic projection).

## 12. Integration Points

- Server gradebook/final-grade/curve handlers; quiz grade-sync (C04).
- Internal: `clients/cli/cmd/grades.go`.

## 13. Dependencies & Sequencing

- After: C03/C04 (submissions/attempts exist to grade), C40 (`--file`, `--yes`, `--dry-run`).
- Before: C07 (SBG builds on scheme), C31 (transcripts read final grades).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Final-grade submit irreversible | M | H | Require `--yes`; support `final-grades unlock` if server allows |
| CSV column drift between export/import | M | M | Share one schema; round-trip test |

## 15. Rollout Plan

- Ship gradebook get/import + scheme first, then final-grades, then curves/what-if.
- Rollback: additive.

## 16. Test Plan

- **Unit** — CSV parse/serialize round-trip; curve math dry-run.
- **Integration** — bulk post partial-failure summary.
- **E2E** — export→edit→import→final submit against dev stack.

## 17. Documentation & Training

- "End-of-term grade posting" and "Apply a curve" recipes.

## 18. Open Questions

1. Is final-grade submission reversible?
2. Does bulk post accept CSV or JSON on the wire?

## 19. References

- `clients/cli/cmd/grades.go`; `registerFinalGradeRoutes`, gradebook/curve handlers.
- Related: [C03](C03-assignments.md), [C04](C04-quizzes-question-banks.md), [C07](C07-outcomes-standards-sbg.md), [C31](C31-credentials-transcripts-advising.md).
