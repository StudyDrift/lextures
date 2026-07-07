# C38 — Portfolios & eportfolio

> CLI parity plan. Source: `registerEportfolioRoutes` (`me/portfolios`, 8; `portfolios`), `me/ccr` (adjacency to C31). Baseline: `clients/cli/cmd/portfolios.go`, `portfolios_logic.go`, `portfolios_test.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C38 |
| **Section** | Student experience |
| **Severity** | MINOR |
| **Markets** | HE / SL / K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Learner XP / CLI |
| **Depends on** | C40 |
| **Unblocks** | — |

---

## 1. Problem Statement

E-portfolios (student showcases of work, artifacts, reflections) have no CLI. Students and program admins cannot bulk-add artifacts, export a portfolio for archival/accreditation, or publish/share it programmatically.

## 2. Goals

- Create portfolios and add artifacts (files, links, reflections) in bulk.
- Publish/share and export a portfolio for accreditation/archival.

## 3. Non-Goals

- Rich portfolio theming/layout (browser).

## 4. Personas & User Stories

- **As a student**, I want `portfolios add-artifact --file project.pdf` to build my showcase.
- **As a program admin**, I want `portfolios export --user U --out d` for accreditation evidence.
- **As a student**, I want `portfolios publish <id>` to share a public link.

## 5. Functional Requirements

- **FR-1.** MUST add `portfolios list|get|create|delete` (`me/portfolios`).
- **FR-2.** MUST add `portfolios add-artifact|remove-artifact <id>` (`--file`, `--link`, `--reflection`).
- **FR-3.** SHOULD add `portfolios publish|unpublish <id>` and `portfolios export <id> --out <dir>`.
- **FR-4.** MAY add `portfolios share <id> --with <user>`.

## 6. Non-Functional Requirements

- **Performance** — artifact upload streams (reuse `files upload`).
- **Security** — self-scope; admin export requires elevated scope.
- **Privacy & Compliance** — portfolios contain student work (FERPA); publish makes content public — CLI warns and requires `--yes`.
- **Reliability** — add-artifact idempotent by content hash.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a file, *When* `portfolios add-artifact --file`, *Then* it's attached and listed.
- **AC-2.** *Given* a portfolio, *When* `portfolios publish --yes`, *Then* a public URL prints.
- **AC-3.** *Given* a portfolio, *When* `portfolios export`, *Then* artifacts + metadata download.

## 8. Data Model

- None client-side.

## 9. API Surface

- `registerEportfolioRoutes` (`me/portfolios`, `portfolios`).

## 10. UI / UX

- `lextures portfolios ...`.

## 11. AI / ML Considerations

- None.

## 12. Integration Points

- Server eportfolio handlers; file upload (existing); CCR (C31).

## 13. Dependencies & Sequencing

- After: C40.
- Before: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Accidental public publish | M | M | `--yes` + explicit URL notice |

## 15. Rollout Plan

- Ship create + add-artifact first, then publish/export.
- Rollback: additive.

## 16. Test Plan

- **Unit** — artifact type flags; idempotency.
- **Integration** — add-artifact; publish.
- **E2E** — build a portfolio → export.

## 17. Documentation & Training

- "Export portfolios for accreditation" recipe.

## 18. Open Questions

1. Is export a bundled archive or per-artifact URLs?

## 19. References

- `registerEportfolioRoutes`.
- Related: [C31](C31-credentials-transcripts-advising.md), [C37](C37-student-workspace.md).
