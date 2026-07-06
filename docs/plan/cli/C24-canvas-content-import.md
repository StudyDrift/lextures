# C24 — Canvas & content import

> CLI parity plan. Source: `registerCanvasImportRoutes`, `canvas_import_queue.go`, `canvas_catalog.go`, `canvas_enrollment_import.go`, `canvas_grade_import.go`, `canvas_assignment_submissions_import.go`, `canvas_announcements_import.go`, `registerCanvasSubmissionSyncRoutes`, `courses/{id}/canvas-link`, `registerImportRoutes` (`imports`). Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C24 |
| **Section** | Integrations & interoperability |
| **Severity** | MAJOR |
| **Markets** | K12 / HE |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Migration / CLI |
| **Depends on** | C18, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

Canvas migration (courses, enrollments, grades, submissions, announcements) and generic content import are UI-only. Migration teams onboarding an institution from Canvas cannot script or monitor the import queue, retry failures, or batch many courses — the slowest, most error-prone part of onboarding.

## 2. Goals

- Browse the Canvas catalog and queue course imports in bulk.
- Drive the import queue (submit, monitor, retry) with `--wait`.
- Import enrollments, grades, submissions, announcements selectively.
- Link an existing course to Canvas for ongoing sync.

## 3. Non-Goals

- Building the Canvas connector (server owns it).
- Non-Canvas LMS imports beyond the generic `imports` surface.

## 4. Personas & User Stories

- **As a migration engineer**, I want `canvas catalog list` then `canvas import course <id> --wait`.
- **As an engineer**, I want `canvas import queue` to see status and `... retry <id>` failures.
- **As a registrar**, I want `canvas import grades --course C` to bring over historical grades.
- **As an admin**, I want `imports submit --file pkg.imscc` for generic Common Cartridge import.

## 5. Functional Requirements

- **FR-1.** MUST add `canvas catalog list|search` (`canvas_catalog.go`).
- **FR-2.** MUST add `canvas import course|enrollments|grades|submissions|announcements` with `--course`/`--wait`.
- **FR-3.** MUST add `canvas import queue|status|retry|cancel` (`canvas_import_queue.go`).
- **FR-4.** SHOULD add `canvas link set|status <course>` (`courses/{id}/canvas-link`) + submission sync.
- **FR-5.** SHOULD add `imports submit|status|list --file <pkg>` for generic (Common Cartridge/QTI) import (`registerImportRoutes`).

## 6. Non-Functional Requirements

- **Performance** — imports are async/job-backed; `--wait` streams progress; batch submit chunks.
- **Security** — migration-admin scope; Canvas API tokens via file/stdin, redacted.
- **Privacy & Compliance** — imported data includes student PII/grades (FERPA); reports gated by `--yes`.
- **Reliability** — import idempotent per source id; retry safe; partial-failure surfaced.
- **Observability** — queue status shows per-item state; final job exit code.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a Canvas course id, *When* `canvas import course --wait`, *Then* the job completes and a summary prints.
- **AC-2.** *Given* failures in the queue, *When* `canvas import retry <id>`, *Then* the item re-runs.
- **AC-3.** *Given* an `.imscc`, *When* `imports submit --wait`, *Then* content imports with a report.

## 8. Data Model

- None client-side.

## 9. API Surface

- Canvas import/catalog/queue/submission-sync endpoints; `courses/{c}/canvas-link`; `registerImportRoutes` generic import.

## 10. UI / UX

- `lextures canvas ...`, `lextures imports ...`.
- Shared `--wait` job primitive (C18/C40).

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server Canvas + import handlers; job queue (C18); course structure (C02).

## 13. Dependencies & Sequencing

- After: C18 (`--wait`), C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Large migrations time out CI | M | M | Async job path + `--wait --timeout`; resume via queue id |
| Duplicate imports | M | M | Idempotency by Canvas source id; `--skip-existing` |

## 15. Rollout Plan

- Ship catalog + course import + queue first, then per-artifact imports + generic import.
- Rollback: additive.

## 16. Test Plan

- **Unit** — token redaction; queue status parsing.
- **Integration** — import submit/poll; retry.
- **E2E** — import a sample Canvas course → verify structure.

## 17. Documentation & Training

- "Migrate courses from Canvas at scale" runbook.

## 18. Open Questions

1. Does course import pull all artifacts or require per-artifact calls?

## 19. References

- `canvas_import_queue.go`, `canvas_catalog.go`, `canvas_*_import.go`, `registerImportRoutes`.
- Related: [C02](C02-modules-course-structure.md), [C18](C18-jobs-scheduler-backups.md), [C04](C04-quizzes-question-banks.md).
