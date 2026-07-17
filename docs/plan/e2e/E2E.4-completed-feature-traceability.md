# E2E.4 — Completed Feature Traceability and Coverage Gate

> Implementation plan. Source: audit of 482 files in `docs/completed` against Playwright specs in `e2e/tests` (2026-07-17).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | E2E.4 |
| **Section** | End-to-End Coverage |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | QA / Developer Experience |
| **Depends on** | None |
| **Unblocks** | E2E.1, E2E.2, E2E.3 |

---

## 1. Problem Statement

Completed plans and E2E specs have no machine-readable relationship. Filename similarity shows broad coverage but cannot distinguish a full journey, API-only test, smoke assertion, feature-flag lifecycle, intentionally manual coverage, or a completed feature that silently lacks E2E tests.

## 2. Goals

- Create a reviewed manifest mapping every completed story to automated and manual coverage.
- Define coverage levels consistently.
- Fail CI when a newly completed story has no disposition.
- Generate a readable coverage report by market, section, and flag family.

## 3. Non-Goals

- Require Playwright for stories that are purely internal, mobile-only, operational, legal-document, or unit/integration-test appropriate.
- Claim code coverage percentages from story mapping.
- Rewrite historical completed plans.

## 4. Personas & User Stories

- **As a product owner**, I want to see which shipped stories have real user-journey protection.
- **As a developer**, I want completion criteria to identify the exact E2E responsibility.
- **As QA**, I want risk-based gaps rather than guesses from filenames.
- **As an operator**, I want flagged features to identify their rollback test.

## 5. Functional Requirements

- **FR-1.** The manifest MUST include every Markdown story under `docs/completed`, excluding indexes/assets with an explicit rule.
- **FR-2.** Each story MUST be classified as `journey`, `smoke`, `api-contract`, `covered-by-parent`, `manual`, `not-applicable`, or `missing` with rationale.
- **FR-3.** Automated entries MUST link exact Playwright spec paths and, where useful, test titles.
- **FR-4.** Flagged stories MUST separately identify settings-toggle, disabled-state, enabled journey, authorization, dependency, and rollback coverage.
- **FR-5.** CI MUST reject missing manifest entries, broken file links, unknown classifications, and unowned `missing` entries.
- **FR-6.** A generator SHOULD publish summary counts without rewriting reviewer-authored rationale.

## 6. Non-Functional Requirements

- **Performance** — validation completes in seconds and does not launch browsers.
- **Security** — no tokens or runtime secrets in the manifest/report.
- **Privacy & Compliance** — compliance stories may point to controlled manual evidence with an owner and cadence.
- **Accessibility** — generated report is semantic Markdown/HTML.
- **Scalability** — supports hundreds of stories and multiple client platforms.
- **Reliability** — deterministic sorted output and actionable errors.
- **Observability** — CI artifact shows additions, removals, and coverage-level changes.
- **Maintainability** — stable IDs and paths; aliases for renamed stories.
- **Internationalization** — not applicable to the internal manifest; linked journeys retain product i18n requirements.
- **Backward compatibility** — bootstrap historical stories as reviewed dispositions rather than failing all at once.

## 7. Acceptance Criteria

- **AC-1.** *Given* the current completed-doc tree, *When* validation runs, *Then* every eligible story has exactly one manifest entry.
- **AC-2.** *Given* an automated entry, *When* its spec is renamed or deleted, *Then* validation fails with the story ID and broken path.
- **AC-3.** *Given* a completed story with a feature flag, *When* its entry is reviewed, *Then* the six flag-coverage dimensions are explicit.
- **AC-4.** *Given* a new story moves into `docs/completed`, *When* CI runs without a disposition, *Then* CI fails.
- **AC-5.** *Given* a `missing` entry, *When* the report is generated, *Then* it includes severity, owner, and target milestone.

## 8. Data Model

Add a YAML or JSON manifest keyed by stable story ID with document path, markets, risk, flags, coverage classification, spec links, manual evidence, owner, and notes. Generated summaries are artifacts, not hand-edited sources.

## 9. API Surface

No product API. Provide a repository command such as `npm run e2e:coverage:check` and `npm run e2e:coverage:report` from `e2e/`.

## 10. UI / UX

The generated report should show section totals and filters for missing journeys, flag lifecycle gaps, market, severity, and client. It must use relative repository links and remain readable as plain Markdown.

## 11. AI / ML Considerations

AI may suggest initial mappings, but a human MUST review classifications and rationales. CI validation is deterministic and does not call a model.

## 12. Integration Points

- `docs/completed/**`
- `e2e/tests/*.spec.ts`
- `e2e/package.json`
- CI workflow and pull-request template/completion checklist.
- E2E.1/E2E.2 flag manifests may be referenced rather than duplicated.

## 13. Dependencies & Sequencing

Define schema and exclusion rules, bootstrap mappings by section, review all `missing` and `not-applicable` entries, add validation, then make the new-completed-story gate required.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Filename-based bootstrap produces false matches | H | M | Require human review and exact links |
| Manifest becomes checkbox bureaucracy | M | M | Keep schema small and generate reports automatically |
| “Covered by parent” hides gaps | M | H | Require parent story/spec and rationale |
| Mobile/ops docs distort web E2E totals | H | M | Explicit client and not-applicable classifications |

## 15. Rollout Plan

No product flag. Introduce validation in report-only mode, review the baseline, then gate only new completed stories. After historical missing entries have owners, enable full integrity checks. Rollback is making the CI check advisory while preserving the manifest.

## 16. Test Plan

- **Unit** — parser, exclusions, schema, stable sorting, broken-link detection.
- **Integration** — run against a fixture completed/spec tree with adds, moves, and deletes.
- **End-to-end** — CI command produces the report and correct exit status.
- **Security** — reject secret-like fields and external evidence URLs where policy requires internal storage.
- **Accessibility** — validate semantic headings/tables in generated output.
- **Performance / load** — complete current 482-document scan in under 10 seconds locally.
- **Manual exploratory** — product/QA review of all initial `missing` and `not-applicable` classifications.

## 17. Documentation & Training

Document the coverage levels, how to register a story/spec, flag lifecycle expectations, exclusions, and the workflow when a plan moves to `docs/completed`.

## 18. Open Questions

1. Should the source manifest be YAML for reviewability or JSON/TypeScript for stronger tooling?
2. Should test titles carry story IDs, or are exact spec-path links sufficient?
3. Who approves `not-applicable` and `manual` dispositions for high-risk compliance stories?

## 19. References

- `docs/plan/_TEMPLATE.md`
- `docs/completed/`
- `e2e/tests/`
- `e2e/playwright.config.ts`
- Related plans: `E2E.1-course-feature-flag-matrix.md`, `E2E.2-platform-feature-flag-contract.md`, `E2E.3-flagged-feature-rollback-and-dependencies.md`.
