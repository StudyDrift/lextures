# C10 — Plagiarism & originality

> CLI parity plan. Source: `courses/{id}/plagiarism-settings` (3), `originality_http.go`, `webhooks_originality.go`, `registerFERPARoutes` (integrity adjacency). Baseline: none.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C10 |
| **Section** | Assessment & grading |
| **Severity** | MAJOR |
| **Markets** | HE / K12 |
| **Status (today)** | MISSING |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Integrity / CLI |
| **Depends on** | C03, C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

Originality/plagiarism configuration and report retrieval are UI-only. Institutions with academic-integrity policies cannot standardize plagiarism settings across courses via script, nor pull originality scores/reports for audit.

## 2. Goals

- Configure per-course plagiarism settings consistently across many courses.
- Trigger originality checks and retrieve scores/reports for submissions.

## 3. Non-Goals

- Implementing a detector (third-party providers do this).
- Adjudicating integrity cases (a human/process concern).

## 4. Personas & User Stories

- **As an integrity officer**, I want `plagiarism settings set --file policy.json` across courses.
- **As an instructor**, I want `originality get <assignment> --user U` to pull a report link/score.
- **As an auditor**, I want `originality list <assignment>` to export scores for a cohort.

## 5. Functional Requirements

- **FR-1.** MUST add `plagiarism settings get|set <course>` (`plagiarism-settings`, `--file`).
- **FR-2.** MUST add `originality status|get <assignment> --user <u>` and `originality list <assignment>`.
- **FR-3.** SHOULD add `originality submit <assignment> --user <u>` to (re)trigger a check.
- **FR-4.** MAY add `originality export <assignment> --out file.csv`.

## 6. Non-Functional Requirements

- **Performance** — list paginated.
- **Security** — integrity/report scope; provider credentials never printed.
- **Privacy & Compliance** — originality reports are FERPA records; export gated by `--yes`.
- **Reliability** — (re)submit idempotent while a check is pending.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a policy file, *When* `plagiarism settings set`, *Then* `get` reflects it.
- **AC-2.** *Given* a checked submission, *When* `originality get`, *Then* score + report reference print.
- **AC-3.** *Given* `originality export` without `--yes`, *Then* it refuses.

## 8. Data Model

- None client-side. Document policy JSON schema.

## 9. API Surface

- `courses/{c}/plagiarism-settings` get/set; `originality_http.go` status/get/list/submit. (Inbound `webhooks_originality.go` is server-side, not CLI.)

## 10. UI / UX

- `lextures plagiarism settings ...`, `lextures originality ...`.

## 11. AI / ML Considerations

- Some originality is AI-detection; CLI only reads provider results.

## 12. Integration Points

- Server originality + plagiarism-settings handlers; third-party provider (Turnitin/etc.) via server.

## 13. Dependencies & Sequencing

- After: C03 (submissions), C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Provider variance in report shape | M | M | Normalize to score + report URL; passthrough raw in `--json` |

## 15. Rollout Plan

- Ship settings get/set + originality get/list; add submit/export next.
- Rollback: additive.

## 16. Test Plan

- **Unit** — policy parse; `--yes` gate.
- **Integration** — originality get/list shapes.
- **E2E** — set policy → submit → get score.

## 17. Documentation & Training

- "Standardize plagiarism policy across a term" recipe.

## 18. Open Questions

1. Which providers are enabled, and do their report shapes differ enough to need per-provider handling?

## 19. References

- `originality_http.go`, `webhooks_originality.go`, `plagiarism-settings` handlers.
- Related: [C03](C03-assignments.md), [C29](C29-compliance-privacy.md).
