# C03 — Assignments (expand)

> CLI parity plan. Source: `courses/{id}/assignments` (45 routes), `assignment_overrides_http.go`, `assignment_submissions_http.go`, `assignment_submission_annotations_http.go`, `assignment_submission_grade_http.go`. Baseline: `clients/cli/cmd/assignments.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C03 |
| **Section** | Assessment & grading |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | PARTIAL (list, get, create, submit) |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Platform / CLI |
| **Depends on** | C01, C02, C40 |
| **Unblocks** | C06, C09 |

---

## 1. Problem Statement

The CLI can list/get/create/submit assignments but cannot update or delete them, manage differentiated **overrides** (assign-to / multiple due dates), list submissions, download student work in bulk, or annotate/comment. Graders and course teams cannot script the full assignment lifecycle or bulk-download submissions for offline grading.

## 2. Goals

- Full assignment CRUD and publish state from the CLI.
- Differentiated assignment overrides (assign-to section/student, alternate due dates).
- Bulk submission listing and download for offline workflows.
- Programmatic annotations/comments on submissions.

## 3. Non-Goals

- Score entry / gradebook math (see C06).
- AI grading agents (see C09); rubrics live under grading settings.

## 4. Personas & User Stories

- **As an instructor**, I want `assignments update/delete` so I can fix or retire tasks via script.
- **As an instructor**, I want `assignments override` to grant an extension to one student/section.
- **As a grader**, I want `assignments submissions download --all --out ./subs` for offline grading.
- **As a TA**, I want `assignments annotate` to attach feedback programmatically.

## 5. Functional Requirements

- **FR-1.** MUST add `assignments update <id>` and `assignments delete <id>` (PUT/DELETE).
- **FR-2.** MUST add `assignments publish|unpublish <id>`.
- **FR-3.** MUST add `assignments overrides list|set|delete <id>` mapping to `assignment_overrides_http.go` (`--section`, `--user`, `--due`, `--available-from/until`).
- **FR-4.** MUST add `assignments submissions list <id>` with filters (`--status`, `--user`, `--late`).
- **FR-5.** MUST add `assignments submissions get <id> --user <u>` and `submissions download <id> [--all] --out <dir>`.
- **FR-6.** SHOULD add `assignments submissions annotate` and `submissions comment` (annotations + text feedback).
- **FR-7.** SHOULD add `assignments grade-history <id>` (assignment_grade_history.go).

## 6. Non-Functional Requirements

- **Performance** — bulk download streams and shows progress; concurrency-limited (reuse upload pattern).
- **Security** — grading scopes enforced server-side; 403 → exit 2.
- **Privacy & Compliance** — submission downloads are FERPA-covered student records; command warns and requires `--yes` for `--all` bulk export.
- **Reliability** — download resumes/skip-existing on re-run.
- **Observability** — bulk ops print per-file success/failure summary.
- **Backward compatibility** — existing `submit` unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* an assignment, *When* `assignments override set --user U --due 2026-09-01`, *Then* that user's due date changes.
- **AC-2.** *Given* an assignment with 30 submissions, *When* `submissions download --all --out d`, *Then* 30 files land under `d/` with a summary.
- **AC-3.** *Given* `--all` without `--yes`, *Then* the command refuses and prints the FERPA warning.

## 8. Data Model

- None client-side. Uses existing submission/attachment DTOs.

## 9. API Surface

- `PUT|DELETE /api/v1/courses/{c}/assignments/{a}`; overrides CRUD; `GET .../submissions`; `GET .../submissions/{u}` + attachment download; annotations POST; grade-history GET.

## 10. UI / UX

- Extend `assignmentsCmd`. New sub-group `assignments submissions` and `assignments overrides`.
- Download shows a progress line per file; `--json` emits a manifest.

## 11. AI / ML Considerations

- None here (AI grading is C09).

## 12. Integration Points

- Server assignment/override/submission handlers.
- Internal: `clients/cli/cmd/assignments.go`.

## 13. Dependencies & Sequencing

- After: C40 (bulk/progress helpers), C02 (module placement of assignments).
- Before: C06 (grade entry references submissions), C09.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Attachment download requires signed URLs | M | M | Follow the existing `files download` signed-URL flow |
| Overrides schema (assign-to sets) complex | M | M | Mirror differentiated-assignments plan 2.15 shapes |

## 15. Rollout Plan

- Ship CRUD + overrides first, then submissions list/download, then annotations.
- Rollback: additive.

## 16. Test Plan

- **Unit** — override flag combinations; download path safety (no traversal).
- **Integration** — httptest for overrides body; submissions pagination.
- **Security** — `--all` gating; 403 handling.

## 17. Documentation & Training

- Recipe: "Bulk-download submissions for offline grading."

## 18. Open Questions

1. Are submission attachments fetched via signed URL or direct stream?
2. Override target model — section only, or section+student+group?

## 19. References

- `clients/cli/cmd/assignments.go`; `assignment_overrides_http.go`, `assignment_submissions_http.go`, `assignment_submission_annotations_http.go`.
- Related: [C06](C06-gradebook-final-grades.md), [C09](C09-ai-grading-agents.md).
