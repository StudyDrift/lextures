# C08 — Peer review, evaluations & surveys

> CLI parity plan. Source: `registerPeerReviewRoutes` (`peer-review`), `courses/{id}/reviews`/`evaluations`, `admin/evaluation-templates`, `registerSurveyRoutes` (`surveys`). Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C08 |
| **Section** | Assessment & grading |
| **Severity** | MINOR |
| **Markets** | HE / K12 / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Assessment / CLI |
| **Depends on** | C03, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

Peer review, course/instructor evaluations, evaluation templates and surveys have no CLI presence. Program admins cannot bulk-configure end-of-term evaluations or export survey results for analysis, and instructors cannot script peer-review assignment/allocation.

## 2. Goals

- Configure and allocate peer reviews for an assignment.
- Manage evaluation templates and launch course/instructor evaluations.
- Create surveys and export responses for BI.

## 3. Non-Goals

- Building the student response UX (browser flow).
- Rubric authoring beyond template fields.

## 4. Personas & User Stories

- **As an instructor**, I want `peer-review allocate <assignment>` to assign reviewers.
- **As a program admin**, I want `evaluations launch --template T --course C` at term end.
- **As an analyst**, I want `surveys results export <id>` to pull responses.

## 5. Functional Requirements

- **FR-1.** MUST add `peer-review status|allocate|list <assignment>` (`registerPeerReviewRoutes`).
- **FR-2.** MUST add `evaluation-templates list|create|get` (admin) and `evaluations launch|list|get <course>`.
- **FR-3.** MUST add `surveys list|create|get|results <id>` with `results export --format csv|json`.
- **FR-4.** SHOULD add `evaluations results export` for completed evaluations.
- **FR-5.** MAY add `peer-review reminders send` if the endpoint exists.

## 6. Non-Functional Requirements

- **Performance** — results export paginated/streamed.
- **Security** — evaluation/survey admin scope; anonymity preserved in exports.
- **Privacy & Compliance** — evaluations often anonymous → CLI MUST NOT expose respondent identity when the server marks a survey anonymous.
- **Reliability** — allocation idempotent.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* submissions, *When* `peer-review allocate --per 2`, *Then* each submission gets 2 reviewers.
- **AC-2.** *Given* a template, *When* `evaluations launch`, *Then* an evaluation id is returned.
- **AC-3.** *Given* an anonymous survey, *When* `surveys results export`, *Then* no respondent identity is present.

## 8. Data Model

- None client-side.

## 9. API Surface

- `peer-review` status/allocate/list; `admin/evaluation-templates`; `courses/{c}/evaluations`; `surveys` CRUD + results.

## 10. UI / UX

- `lextures peer-review ...`, `lextures evaluations ...`, `lextures surveys ...`.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server peer-review/evaluation/survey handlers.

## 13. Dependencies & Sequencing

- After: C03 (peer review targets submissions), C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Anonymity leakage in export | L | H | Assert server anonymity flag; strip identity fields; test |

## 15. Rollout Plan

- Ship surveys + evaluations first (broader use), then peer-review allocation.
- Rollback: additive.

## 16. Test Plan

- **Unit** — allocation flag math; anonymity stripping.
- **Integration** — results export shape.
- **E2E** — allocate peer review; export survey.

## 17. Documentation & Training

- "Launch end-of-term evaluations" recipe.

## 18. Open Questions

1. Is peer-review allocation server-driven or does the CLI supply pairings?

## 19. References

- `registerPeerReviewRoutes`, `registerSurveyRoutes`, evaluation-template handlers.
- Related: [C03](C03-assignments.md), [C27](C27-reports-exports.md).
