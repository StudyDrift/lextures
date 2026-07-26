# TD.14 — Decompose God Components

> Implementation plan. Source: technical-debt static analysis, 2026-07-25. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | TD.14 |
| **Section** | Technical Debt Remediation |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | XL (>2mo) |
| **Owner (proposed)** | Web platform team, with a named engineer per screen |
| **Depends on** | TD.13 |
| **Unblocks** | Sustainable feature velocity in the course area |

---

## 1. Problem Statement

The web client's core screens have grown past the point of safe modification. `course-modules.tsx` is **3,372 lines with 99 `useState` and 36 `useCallback`**; `course-module-quiz-page.tsx` is **3,383 lines with 99 `useState`**; `quiz-student-take-panel.tsx` is 2,196; `gradebook-grid.tsx` 2,030; `course-enrollments.tsx` 1,907; `course-settings.tsx` 1,821. These are the screens students and instructors use most, so they attract the most change — and each change is riskier than the last. No one can hold 99 pieces of state in mind, so bugs are found by users rather than review; the components cannot be unit-tested in any meaningful unit; and they cannot be reused, so adjacent features copy from them, spreading the pattern. Only **16 shared hooks** exist for **718 components and pages**, confirming that almost nothing has been extracted.

[TD.13](TD.13-adopt-server-state-management.md) removes the largest single contributor — the hand-rolled fetch/loading/error state. This story addresses what remains: genuine UI composition.

## 2. Goals

- Bring the largest screens under the TD.2 size budget by extracting real, reusable units.
- Extract shared behaviour into hooks and shared UI into components, growing the shared layer beyond 16 hooks.
- Make the extracted pieces independently testable, raising meaningful coverage on the most-used screens.
- Preserve every screen's behaviour, appearance, and accessibility exactly.
- Leave a repeatable decomposition pattern the team applies to future screens.

## 3. Non-Goals

- Redesigning any screen. This is structural, not visual — pixels must not move.
- Changing feature behaviour, permissions, or copy.
- Migrating fetching to the query library — that is TD.13, which lands first.
- Decomposing all 718 components — only those over budget, in pain order.
- Introducing a component library or design-system rewrite.

## 4. Personas & User Stories

- **As a student taking a quiz**, I want the screen to keep working exactly as it does, so that a refactor never costs me an attempt.
- **As an instructor managing modules**, I want new features to arrive faster and with fewer regressions.
- **As a web engineer**, I want to change the module list without reading 3,372 lines, so that a small feature is a small change.
- **As a reviewer**, I want a diff scoped to one extracted component, so that I can actually review it.
- **As a QA engineer**, I want testable units, so that regressions are caught by tests rather than by exploratory passes.

## 5. Functional Requirements

- **FR-1.** Each target screen MUST be decomposed until every resulting file is within the TD.2 size budget (500 lines).
- **FR-2.** Decomposition MUST be **behaviour-preserving**: no change to rendered output, interaction, permissions, or copy.
- **FR-3.** Extractions MUST follow real seams — a section of UI with its own state and responsibility — not arbitrary line-count splits.
- **FR-4.** Shared behaviour MUST be extracted into hooks under `src/hooks/` or beside the feature; shared UI into components, with reuse across screens where genuine.
- **FR-5.** Each extracted component MUST have at least one test covering its primary interaction.
- **FR-6.** Each screen's accessibility MUST be verified before and after — focus order, ARIA relationships, keyboard navigation, and screen-reader announcements.
- **FR-7.** Decomposition MUST proceed **one screen per PR series**, in pain order, with the screen releasable at each step.
- **FR-8.** Target screens, in priority order: `course-modules.tsx` (3,372), `course-module-quiz-page.tsx` (3,383), `quiz-student-take-panel.tsx` (2,196), `gradebook-grid.tsx` (2,030), `course-enrollments.tsx` (1,907), `course-settings.tsx` (1,821), `courses.tsx` (1,720), `assignment-page-settings-panel.tsx` (1,656).
- **FR-9.** Before decomposing a screen, its current behaviour MUST be captured by tests sufficient to detect regression — characterization tests, mirroring TD.1's approach on the backend.
- **FR-10.** Extracted pieces MUST NOT increase render count or introduce unnecessary re-renders; verify with React DevTools profiler on the heaviest screens.
- **FR-11.** File naming MUST follow the kebab-case convention (TD.2 FR-5); this is an opportunity to retire entries from the naming allowlist.
- **FR-12.** The TD.2 file-size allowlist MUST shrink with each completed screen.

## 6. Non-Functional Requirements

- **Performance** — no regression in render performance or route chunk size. The heaviest screens are already the slowest; measure with the existing Lighthouse harness and the profiler (FR-10). Decomposition may *improve* things via better memoisation boundaries, but that is a bonus, not a licence to change behaviour.
- **Security** — permission-gated UI is dense in these screens (instructor vs student vs admin views). An extraction that drops a permission check would expose controls to the wrong role. Every extracted component that renders conditionally on permission MUST have a test asserting the negative case.
- **Privacy & Compliance** — no change to data displayed; FERPA-relevant views (gradebook, enrollments) must render identically per role.
- **Accessibility** — WCAG 2.1 AA must be preserved exactly. These screens contain complex widgets (grids, drag-and-drop module ordering, quiz-taking) where focus management and ARIA relationships are easy to break during extraction. FR-6 is a hard gate, not a review nicety.
- **Scalability** — n/a.
- **Reliability** — the quiz-taking screen is the highest-stakes surface in the product: a regression can cost a student a graded attempt. It gets the most conservative treatment and the deepest test coverage before any change.
- **Observability** — no change; verify any analytics or telemetry events fire identically after extraction.
- **Maintainability** — the goal.
- **Internationalization** — extracted components must keep using the existing i18n keys; no string may be inlined during extraction.
- **Backward compatibility** — user-visible behaviour unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* a decomposed screen, *When* its characterization tests run, *Then* rendered output and interactions match pre-decomposition behaviour.
- **AC-2.** *Given* a decomposed screen, *When* file sizes are measured, *Then* every resulting file is within the TD.2 budget.
- **AC-3.** *Given* a decomposed screen, *When* audited with axe and a screen reader, *Then* results match the pre-decomposition audit — same focus order, same announcements, no new violations.
- **AC-4.** *Given* an extracted component rendering permission-gated controls, *When* a user without the permission renders it, *Then* the controls are absent (explicit negative test).
- **AC-5.** *Given* each extracted component, *When* the test suite runs, *Then* it has at least one interaction test.
- **AC-6.** *Given* a decomposed screen, *When* profiled, *Then* render count and time are no worse than before.
- **AC-7.** *Given* a decomposed screen, *When* `make e2e` runs, *Then* all flows for that screen pass unchanged.
- **AC-8.** *Given* all target screens are complete, *When* the TD.2 allowlists are inspected, *Then* the file-size and naming allowlists have shrunk accordingly.
- **AC-9.** *Given* the quiz-taking screen specifically, *When* a full attempt is exercised end-to-end (start, answer, save, submit, review, and an interrupted-then-resumed attempt), *Then* behaviour is identical to pre-decomposition.
- **AC-10.** *Given* i18n checks, *When* `npm run i18n:check` runs, *Then* it passes with no newly inlined strings.

## 8. Data Model

No schema change. Target structure per screen, illustrated for course modules:

```
clients/web/src/pages/lms/course-modules/
  index.tsx                 # composition + routing only, well under budget
  use-module-reorder.ts     # extracted behaviour
  use-module-selection.ts
  module-list.tsx
  module-item.tsx
  module-toolbar.tsx
  module-bulk-actions.tsx
  __tests__/
```

Shared extractions graduate to `src/hooks/` or `src/components/` when genuinely reused by more than one screen — not speculatively.

## 9. API Surface

**No API change.**

## 10. UI / UX

**No intended visual or behavioural change** — this is the story's defining constraint.

Per screen, verify and preserve:

1. **Key flows** — enumerate before starting; they become the characterization tests (FR-9).
2. **Empty / loading / error / offline states** — all preserved, including the offline behaviour on offline-capable screens.
3. **Mobile / responsive** — breakpoint behaviour identical; verify at each breakpoint.
4. **Accessibility** — focus order, ARIA relationships, keyboard paths, live-region announcements. Drag-and-drop module ordering and the gradebook grid have bespoke keyboard handling (`src/lib/dnd/`) that is easy to break.
5. **Copy & i18n** — keys unchanged; no inlined strings (AC-10).
6. **Motion** — screens participating in the AN motion plans must keep their transitions.

## 11. AI / ML Considerations

`assignment-annotation-workbench.tsx` (1,424) and the grader-agent components (`use-grader-agent-workflow.ts` 1,251, `workflow-nodes.tsx` 985) are AI-driven surfaces. They are **not** in the FR-8 priority list because their AI interaction patterns are settled by TD.13 §11 first. Revisit them once that pattern is documented.

## 12. Integration Points

- `clients/web/src/pages/lms/` — the target screens.
- `clients/web/src/components/` — destination for shared extractions.
- `clients/web/src/hooks/` — currently 16 hooks; expected to grow.
- `clients/web/src/lib/courses/queries.ts` (TD.13) — screens consume query hooks rather than managing fetch state.
- `clients/web/src/lib/dnd/` — drag-and-drop keyboard handling (§10.4).
- `clients/web/src/lazy-pages.ts` — route splitting; update paths as screens become directories.
- CI: TD.2 allowlists, `i18n:check`, `contrast:check`, `interface-polish:check`, `bundle:check`, Lighthouse harness.
- `e2e/tests/` — existing specs for these screens.

## 13. Dependencies & Sequencing

- Must ship after: **TD.13** — migrating fetch state first removes the majority of the `useState` count and prevents decomposing machinery that is about to be deleted.
- Should follow: **TD.12** (screens import from split modules).
- Must ship before: nothing hard; this is the programme's tail.
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Accessibility regression in a complex widget (grid, drag-and-drop, quiz) | **H** | **H** | FR-6/AC-3 before-and-after audits as a hard gate; a11y specialist review for the grid and DnD screens |
| A permission-gated control becomes visible to the wrong role | M | **H** | AC-4 explicit negative tests for every permission-conditional extraction |
| Quiz-taking regression costs a student a graded attempt | M | **H** | Deepest characterization coverage before any change (FR-9); AC-9 full-attempt e2e including interrupted-and-resumed; consider a staged rollout on this screen specifically |
| Extraction changes re-render behaviour, degrading performance | M | M | FR-10/AC-6 profiler verification; avoid inline object/callback props at new boundaries |
| Screens are split arbitrarily to satisfy the budget, producing worse code | **H** | M | FR-3 requires real seams; reviewers reject line-count-driven splits; a screen that resists decomposition is a design conversation, not a mechanical one |
| Merge conflicts with active feature work on the most-changed files | **H** | M | FR-7 one screen at a time; announce windows; sequence against the roadmap |
| Inlined strings during extraction break i18n | M | M | AC-10 `i18n:check` gate |
| Story never finishes; some screens stay huge | M | M | FR-8 pain-ordered list; each completed screen is independently valuable; partial completion is an acceptable outcome |

## 15. Rollout Plan

- **Feature flag** — none by default. **Exception**: consider a flag for the quiz-taking screen given §14's stakes, allowing instant revert without a deploy.
- **Sequencing** — per screen: (1) enumerate flows and write characterization tests; (2) capture the accessibility baseline; (3) extract in small PRs, screen releasable at each; (4) verify a11y, profiler, e2e; (5) shrink the TD.2 allowlists; (6) next screen.
- **Dogfood** — each completed screen soaks on staging with internal users before the next begins.
- **GA criteria** — per screen: tests green, a11y audit matched, one week in production with no attributable regression.
- **Rollback** — per-PR revert; per-screen flag where used.

## 16. Test Plan

- **Unit** — each extracted component and hook: rendering, interaction, permission negative cases (AC-4).
- **Integration** — full screen composition against mocked queries; characterization tests from FR-9.
- **End-to-end** — existing Playwright specs must pass unchanged; add the AC-9 full quiz-attempt flow including interruption and resume.
- **Security** — role matrix per screen: render as student, instructor, admin, and (where relevant) parent; assert control visibility matches pre-decomposition exactly.
- **Accessibility** — axe automated scan plus manual screen-reader script per screen, before and after (AC-3); explicit keyboard-path testing for drag-and-drop module ordering and the gradebook grid.
- **Performance / load** — React profiler render counts (AC-6); Lighthouse via `npm run lighthouse:dashboard:dark` for affected routes; `bundle:check`.
- **Manual exploratory** — per-screen QA checklist at each breakpoint, plus offline where applicable.

Baseline:

```bash
cd clients/web
for f in src/pages/lms/course-modules.tsx src/pages/lms/course-module-quiz-page.tsx \
         src/components/quiz/quiz-student-take-panel.tsx src/pages/lms/gradebook/gradebook-grid.tsx \
         src/pages/lms/course-enrollments.tsx src/pages/lms/course-settings.tsx; do
  echo "$f lines=$(wc -l < $f) useState=$(grep -oE 'useState[<(]' $f | wc -l)"
done
```

## 17. Documentation & Training

- `docs/ARCHITECTURE_CONVENTIONS.md` (TD.2) — component size budget; extract on real seams; hooks for behaviour, components for UI.
- Decomposition playbook: enumerate flows → characterize → capture a11y baseline → extract → verify. Written after the first screen, from what was actually learned.
- Per-screen notes recording the seams found, useful to whoever next changes that area.
- `docs/design.md` / `docs/design-tokens.md` — update if extraction surfaces genuinely reusable UI worth promoting to the shared layer.

## 18. Open Questions

1. Should the quiz-taking screen be flagged for staged rollout given §14's stakes? (Leaning yes — it is the only screen where a regression has an academic consequence.)
2. How much of the 99 `useState` count actually disappears with TD.13? Measure after TD.13 lands on the priority screens; the answer determines this story's real size and may reorder FR-8.
3. Which extractions are genuinely shared versus screen-local? Resist premature promotion to `src/components/` — a shared component with one caller is a liability.
4. Do the AI-driven components (§11) belong in this story at all, or in a follow-up once TD.13's AI pattern is settled?
5. Is there an accessibility specialist available for the grid and drag-and-drop screens, or does that capability need sourcing before those screens start?
6. Should `index.css` (1,819 lines, 193 custom classes) be decomposed alongside the components that use it, or handled separately?

## 19. References

- `clients/web/src/pages/lms/course-modules.tsx` — 3,372 LOC, 99 `useState`, 36 `useCallback`
- `clients/web/src/pages/lms/course-module-quiz-page.tsx` — 3,383 LOC, 99 `useState`
- `clients/web/src/components/quiz/quiz-student-take-panel.tsx` — 2,196 LOC
- `clients/web/src/pages/lms/gradebook/gradebook-grid.tsx` — 2,030 LOC
- `clients/web/src/lib/dnd/` — drag-and-drop keyboard handling
- `clients/web/src/hooks/` — 16 shared hooks for 718 components and pages
- `docs/accessibility/`, `docs/vpat/` — conformance obligations
- Related plans: [TD.13](TD.13-adopt-server-state-management.md), [TD.12](TD.12-split-courses-api-module.md), [TD.2](../../completed/tech_debt/TD.2-convention-charter-and-enforcement.md)
