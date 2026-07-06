# C04 — Quizzes & question banks

> CLI parity plan. Source: `courses/{id}/quizzes` (35 routes), `registerQuizDeliveryRoutes`, `registerQuizSubmitRoutes`, `registerQuizGradingRoutes`, `registerQuizGradeSyncRoutes`, `registerQuizCodeRunRoutes`. Baseline (partial): `clients/cli/cmd/questions.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C04 |
| **Section** | Assessment & grading |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETE |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Assessment / CLI |
| **Depends on** | C02, C40 |
| **Unblocks** | C06, C09 |

---

## 1. Problem Statement

The CLI exposes only question-bank list/create/import — there is no way to create or manage **quizzes**, add questions to them, publish, view attempts, or trigger grading (including autograded code questions). Assessment-heavy programs cannot author or operate quizzes from CI or scripts, and cannot export question banks for backup/migration.

## 2. Goals

- Author quizzes and attach questions (from banks or inline) programmatically.
- Manage quiz settings (time limits, attempts, availability, shuffle).
- List attempts, grade submissions, sync grades to the gradebook.
- Round-trip question banks (import already exists; add export).

## 3. Non-Goals

- Live proctoring UX; real-time delivery to students (student-side is a browser flow).
- Rubric authoring beyond quiz settings (see C07 outcomes/rubrics).

## 4. Personas & User Stories

- **As an author**, I want `quizzes create` + `quizzes questions add` to build a quiz from a bank.
- **As an instructor**, I want `quizzes publish` and `quizzes attempts list` to operate an exam.
- **As a grader**, I want `quizzes grade` and `quizzes grade-sync` to push results to the gradebook.
- **As an admin**, I want `questions export --bank <id>` for backup/migration (QTI).

## 5. Functional Requirements

- **FR-1.** MUST add `quizzes list|get|create|update|delete <course>` and `quizzes publish|unpublish`.
- **FR-2.** MUST add `quizzes questions add|remove|list|reorder <quiz>` (reference bank questions or inline).
- **FR-3.** MUST add `quizzes settings set <quiz>` (time limit, attempts, shuffle, availability window).
- **FR-4.** MUST add `quizzes attempts list <quiz>` and `quizzes attempts get <quiz> --user <u>`.
- **FR-5.** MUST add `quizzes grade <quiz>` (trigger/regrade) and `quizzes grade-sync <quiz>` (push to gradebook).
- **FR-6.** SHOULD add `quizzes code-run <quiz>` diagnostics for autograded code questions.
- **FR-7.** SHOULD add `questions export --bank <id> [--qti]` complementing existing `questions import`.
- **FR-8.** MAY add `questions banks list|create` for bank management.

## 6. Non-Functional Requirements

- **Performance** — attempts list paginated; large banks streamed.
- **Security** — quiz author/grader scopes; attempt data is FERPA-covered → same `--yes` gating as C03 for bulk export.
- **Reliability** — grade/grade-sync idempotent; regrade safe to re-run.
- **Observability** — grade-sync prints count synced/failed.
- **Maintainability** — new `cmd/quizzes.go`; extend `cmd/questions.go`.
- **Backward compatibility** — `questions import` unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* a bank, *When* `quizzes create` then `quizzes questions add --bank B --count 10`, *Then* the quiz has 10 questions.
- **AC-2.** *Given* a submitted quiz, *When* `quizzes grade-sync`, *Then* gradebook scores appear (verify via C06 `grades list`).
- **AC-3.** *Given* `--json`, attempts list emits an array of attempt DTOs.
- **AC-4.** *Given* a bank, *When* `questions export --qti`, *Then* a valid QTI .zip is produced.

## 8. Data Model

- None client-side. QTI export writes a `.zip`; import already parses QTI.

## 9. API Surface

- Quiz CRUD/publish under `/api/v1/courses/{c}/quizzes`; delivery/submit/grading/grade-sync/code-run route groups; question-bank list/create + export.

## 10. UI / UX

- `lextures quizzes ...` (course-scoped via `--course`), `lextures questions ...`.
- Table default; `--json` raw. Attempt export gated by `--yes`.

## 11. AI / ML Considerations

- Autograded code questions run server-side sandboxes; CLI only triggers/reads — no model calls from CLI. AI-assisted grading is deferred to C09.

## 12. Integration Points

- Server quiz + grade-sync handlers; gradebook (C06).
- Internal: `clients/cli/cmd/quizzes.go`, `questions.go`.

## 13. Dependencies & Sequencing

- After: C02 (place quiz in a module), C40.
- Before: C06 (grade-sync target), C09.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| 35 quiz routes → large surface | H | M | Phase: author → operate → grade; ship incrementally |
| QTI export fidelity | M | M | Round-trip test import↔export |

## 15. Rollout Plan

- Phase 1 authoring (CRUD/questions/settings), Phase 2 operate (publish/attempts), Phase 3 grade/sync/export.
- Rollback: additive per phase.

## 16. Test Plan

- **Unit** — question-add modes (bank ref vs inline); settings flags.
- **Integration** — httptest for grade-sync; QTI export golden file.
- **E2E** — author→publish→simulate attempt→grade-sync→verify grade.

## 17. Documentation & Training

- "Author a quiz from a question bank in CI" recipe; QTI backup guide.

## 18. Open Questions

1. Can questions be added inline, or only by bank reference?
2. Is grade-sync automatic on submit or an explicit action?

## 19. References

- `clients/cli/cmd/questions.go`; quiz route groups in `server/internal/httpserver/server.go`.
- `clients/cli/cmd/quizzes.go`, `quizzes_extend.go`, `quizzes_test.go`, `quizzes_extend_test.go`; `questions_extend.go`, `questions_extend_test.go`.
- Related: [C02](C02-modules-course-structure.md), [C06](../plan/cli/C06-gradebook-final-grades.md), [C09](../plan/cli/C09-ai-grading-agents.md).
