# W05 — Human-Readable Labels for Entity IDs in Instructor & Review Surfaces

> Implementation plan. Source: web market-readiness scan (2026-07-06).

## Implementation notes (2026-07)

- **Shared helper:** `formatEntityLabel()` in `clients/web/src/lib/format-entity-label.ts` and `<EntityLabel>` in `components/ui/entity-label.tsx` centralize name → pseudonym → neutral fallback (never raw UUID prefixes).
- **Moderation API:** `GET .../reconciliation` now returns `studentName`, `submissionLabel`, and `graderName` per provisional row; respects blind-grading redaction server-side. Go handlers in `moderated_grading_http.go`; repo in `provisionalgrades/`.
- **Peer review API:** `GET .../peer-review/summary` adds `studentLabel` and `incompleteReviewerLabels` / `outlierReviewerLabels` (real names when `named`, stable pseudonyms when anonymous).
- **Assignment staff picker:** uses `entityLabel.unknownStaff` fallback instead of `Staff {id.slice(0,8)}…`.
- **Guard:** `npm test` runs `scripts/check-entity-labels.mjs` on the three W05 surfaces.
- **Tests:** Vitest (`format-entity-label.test.ts`); Go nodb test (`moderated_grading_nodb_test.go`).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | W05 |
| **Section** | Web / UX Polish |
| **Severity** | MINOR |
| **Markets** | HE / K12 |
| **Status (today)** | DONE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Frontend platform team |
| **Depends on** | none (parent surface handled by W02) |
| **Unblocks** | Legible moderation/peer-review workflows |

---

## 1. Problem Statement

Several user-facing surfaces render **raw truncated UUIDs** (`id.slice(0,8)…`) where a human name or
title belongs. The clearest cases are instructor/reviewer workflows: the **moderation dashboard**
(`pages/lms/moderation-dashboard.tsx`) shows submissions, students, and graders as ID prefixes; the
**peer-review summary** (`pages/lms/peer-review-summary-page.tsx`) shows reviewers as
`studentUserId.slice(0,8)…`; the **assignment page** falls back to `Staff {userId.slice(0,8)}…`. A grader
triaging held submissions cannot tell who or what a row refers to, which makes these workflows slow and
feels unfinished. (The K-12 parent grades case is the same pattern and is owned by [W02](W02-parent-guardian-portal-completeness.md).)

## 2. Goals

- Replace raw ID prefixes with human-readable labels (student/grader display name, submission/assignment
  title) on instructor- and reviewer-facing surfaces.
- Preserve deliberate anonymity where it is a feature (blind grading / anonymous peer review) — show a
  stable pseudonym ("Reviewer 3"), not a raw UUID.
- Provide a single, reusable "entity label" pattern so future surfaces don't reintroduce raw IDs.

## 3. Non-Goals

- Admin **audit/event-log** surfaces where an opaque actor/target ID is acceptable forensic detail
  (e.g. `AdminOverview` audit rows) — out of scope unless a name is trivially available.
- Changing anonymity/blind-grading rules (only how anonymized entities are *displayed*).
- The parent portal grade legibility fix (owned by W02).

## 4. Personas & User Stories

- **As an instructor moderating grades**, I want each row to show the student's name and the assignment
  title so that I can review without decoding IDs.
- **As a peer-review coordinator**, I want reviewers shown by name (or a stable pseudonym when the
  activity is anonymous) so that I can follow who reviewed what.
- **As an instructor**, I want a co-teacher shown by name, not "Staff a1b2c3d4…".

## 5. Functional Requirements

- **FR-1.** The moderation dashboard MUST display student and grader **display names** and a submission
  **label/title** (falling back to a short human label, never a raw UUID).
- **FR-2.** The peer-review summary MUST display reviewer names when the activity is non-anonymous, and a
  stable pseudonym (e.g. "Reviewer 3") when anonymous — not `studentUserId.slice(0,8)…`.
- **FR-3.** The assignment-page staff fallback MUST resolve a name; only if truly unknown may it show a
  neutral label ("Unknown staff"), not an ID prefix.
- **FR-4.** Where the API does not currently return names/titles, it MUST be extended (or a batch
  name-resolution endpoint used) so the client has labels without per-row fetches.
- **FR-5.** A shared `<EntityLabel>` / `formatEntityLabel()` helper SHOULD centralize the
  name → fallback → pseudonym logic and be reused by these surfaces.

## 6. Non-Functional Requirements

- **Performance** — Names resolved in the list payload or one batch call; no N+1 per row.
- **Security** — Name disclosure MUST respect anonymity settings (blind grading, anonymous peer review)
  and the viewer's authorization; do not leak a name the viewer isn't allowed to see.
- **Privacy & Compliance** — FERPA: names shown only to authorized staff for their own course/section.
- **Accessibility** — Labels are readable text (not title-attribute only); pseudonyms are consistent so
  screen-reader users can track an entity across a table.
- **Reliability** — Missing name degrades to a neutral label, never a crash or a raw UUID.
- **Observability** — Optional: `entity_label_fallback_rate` to catch surfaces still lacking names.
- **Maintainability** — One helper; lint/grep guard against new `id.slice(0, 8)` in JSX display paths.
- **Internationalization** — Fallback/pseudonym strings via `t()` (W01).
- **Backward compatibility** — Additive API fields; anonymized flows visually unchanged in intent.

## 7. Acceptance Criteria

- **AC-1.** *Given* the moderation dashboard with held submissions, *When* it renders, *Then* each row
  shows student name, grader name, and a submission label — no `id.slice(0,8)…`.
- **AC-2.** *Given* a non-anonymous peer-review activity, *When* the summary renders, *Then* reviewers
  show by name; *Given* an anonymous activity, *Then* reviewers show a stable pseudonym.
- **AC-3.** *Given* a co-teacher on an assignment, *When* the staff picker renders, *Then* their name
  shows (not "Staff a1b2c3d4…").
- **AC-4.** *Given* a blind-grading context, *When* labels render, *Then* no student identity leaks
  before identities are revealed.

## 8. Data Model

- No new tables. May add a name-resolution/batch endpoint or enrich existing list responses with
  `displayName` / `title` / `label` fields.

## 9. API Surface

- Extend moderation and peer-review list responses with `studentName`, `graderName`, `submissionLabel`,
  `reviewerName`/`reviewerAlias`. Alternatively add `POST /api/v1/users:resolve-names` (batch id→name,
  authorization-checked) for reuse.
- Anonymity flags MUST gate whether real names are returned.

## 10. UI / UX

- **Modified:** `pages/lms/moderation-dashboard.tsx`, `pages/lms/peer-review-summary-page.tsx`,
  `pages/lms/course-module-assignment-page.tsx` (staff fallback).
- **New:** `components/ui/entity-label.tsx` (or a `lib/format` helper).
- **States:** name loading → skeleton; unknown → neutral label; anonymous → pseudonym.
- **Accessibility:** consistent pseudonyms across the table.
- **Copy & i18n:** "Reviewer {n}", "Unknown staff" via `t()`.

## 11. AI / ML Considerations

- Not applicable.

## 12. Integration Points

- The three pages above; a shared label helper; server list handlers for moderation/peer-review/assignment
  staff; anonymity settings (blind grading / anonymous peer review).

## 13. Dependencies & Sequencing

- **Must ship after:** none.
- **Must ship before:** none (independent polish).
- **Shared infra:** optional batch name-resolution endpoint.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Name leak breaks blind grading / anonymous peer review | M | H | Anonymity flag gates name resolution server-side; AC-4 test. |
| N+1 name lookups | M | M | Enrich list payloads or one batch call. |
| New surfaces reintroduce raw IDs | M | L | Shared helper + grep/lint guard on `.slice(0, 8)` in JSX. |

## 15. Rollout Plan

- **Feature flag:** none — incremental per surface.
- **Sequencing:** moderation dashboard → peer-review summary → assignment staff fallback.
- **GA criteria:** ACs pass; no raw ID prefixes on the three surfaces.
- **Rollback:** per-surface revert.

## 16. Test Plan

- **Unit** — `formatEntityLabel`: name → fallback → pseudonym; anonymity path.
- **Integration** — moderation/peer-review responses carry names; anonymity gating.
- **End-to-end** — Playwright asserts no `…`-truncated UUID text on the three surfaces.
- **Security** — anonymity/authorization matrix (blind grading, anonymous peer review).
- **Accessibility** — pseudonym stability across rows.

## 17. Documentation & Training

- Engineering: "Never render a raw ID to users — use `<EntityLabel>`" note in the web README.

## 18. Open Questions

1. Batch name-resolution endpoint vs. enriching each list response — which does the backend prefer?
2. Are admin audit-log ID prefixes explicitly in-scope for a follow-up, or intentionally opaque?

## 19. References

- `clients/web/src/pages/lms/moderation-dashboard.tsx` (submission/student/grader `.slice(0,8)…`).
- `clients/web/src/pages/lms/peer-review-summary-page.tsx:116` (`studentUserId.slice(0,8)…`).
- `clients/web/src/pages/lms/course-module-assignment-page.tsx:669` (`Staff {userId.slice(0,8)}…`).
- Related plans: [W02](W02-parent-guardian-portal-completeness.md) (parent grade legibility),
  [W01](W01-i18n-application-coverage.md).
