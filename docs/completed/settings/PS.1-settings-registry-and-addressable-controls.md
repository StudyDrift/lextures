# PS.1 — Settings Registry & Addressable Editor Controls

> Implementation plan. Source: authoring-UX gap — important assignment/quiz settings are buried inside collapsed accordions. Foundation for pinning. Folder overview: [README](README.md). Active backlog: [docs/plan/settings](../../plan/settings/).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | PS.1 |
| **Section** | Pinned Editor Settings |
| **Severity** | MINOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Web client team |
| **Depends on** | — |
| **Unblocks** | PS.2, PS.3, PS.4 |

---

## 1. Problem Statement

The assignment and quiz editors each render a settings sidebar built from hand-written accordions
(`assignment-page-settings-panel.tsx`, 1669 lines; `quiz-page-settings-panel.tsx`, 897 lines). Neither
panel has any notion of an individual, addressable setting: search matches whole *sections* against a
hard-coded keyword map (`SETTINGS_SECTION_SEARCH`), so typing "lockdown" expands the entire
Presentation accordion with eleven controls instead of surfacing the one control the instructor
wanted. Nothing outside the component file can reference "the lockdown mode control", which makes
per-user pinning (PS.3), per-setting telemetry (PS.4), and deep links from help content impossible.

## 2. Goals

- Give every user-facing control in both editor settings panels a **stable, string-addressable ID**
  that survives refactors and can be persisted in a database.
- Move the search index from section-level to **control-level**, so a query matches and reveals the
  specific setting.
- Refactor both panels to render each control through a shared wrapper so a control can be rendered
  **in its section or relocated** to another region of the panel without duplicating DOM or ids.
- Keep the change **behaviour-neutral** for users: same controls, same order, same copy, same save
  semantics as today.
- Establish a single place where a new setting is declared, so adding a setting later requires one
  registry entry instead of edits scattered across search maps and panel bodies.

## 3. Non-Goals

- No pin UI, no persistence, no API calls — those are PS.2 and PS.3.
- No redesign of the panels' visual language, accordion grouping, or copy.
- No change to how settings are *saved* (draft state still lives in the page components and is written
  by the existing toolbar save / immediate-patch paths).
- No extension to other settings surfaces (content page, module, discussion, survey, course settings);
  the registry is deliberately scoped to the two editor panels named in the request.
- No server-side mirror of the registry (see §18 Q1).

## 4. Personas & User Stories

- **As an instructor**, I want searching "kiosk" to show me the lockdown control itself, so that I stop
  scanning a 7-control accordion for the one row I need.
- **As an instructor**, I want the panel to behave exactly as it did yesterday when I am not searching,
  so that a refactor does not cost me a re-learning tax.
- **As a web engineer**, I want to declare a new quiz setting in one registry entry and have search,
  pinning, and telemetry work automatically, so that features stop drifting out of the search index.
- **As a support agent**, I want a stable identifier per setting so that help-center articles and
  in-product help can link to a specific control.
- **As a homeschool parent teaching solo**, I want the settings I actually use to be findable in one
  or two keystrokes, so that authoring a quiz does not require an LMS tutorial.

## 5. Functional Requirements

- **FR-1.** The system MUST define a registry module (`clients/web/src/lib/settings-registry.ts`)
  exporting a `SETTINGS_REGISTRY: SettingDescriptor[]` covering every user-facing control currently
  rendered by `assignment-page-settings-panel.tsx` and `quiz-page-settings-panel.tsx`.
- **FR-2.** Each `SettingDescriptor` MUST carry: `id`, `surface` (`'assignment' | 'quiz'`), `section`
  (existing `SettingsSectionId`), `label` (identical to the rendered label), `keywords: string[]`, and
  `pinnable: boolean`.
- **FR-3.** Setting IDs MUST follow `{surface}.{section}.{control}` in lower-kebab segments
  (e.g. `quiz.presentation.lockdown-mode`) and MUST be treated as a persisted contract: once shipped,
  an ID is never re-pointed to a different control.
- **FR-4.** The registry MUST expose a `SETTING_ID_ALIASES: Record<string, string>` map so a renamed
  or merged control can resolve legacy IDs, and a `resolveSettingId(id): string | null` helper that
  returns the canonical ID or `null` when the ID is unknown/retired.
- **FR-5.** Registry lookups MUST be O(1) by ID via a derived `Map`, built once at module load.
- **FR-6.** Both panels MUST render each control inside a shared `<SettingRow settingId=… >` wrapper
  that (a) reads its descriptor from the registry, (b) applies the current search predicate, and
  (c) exposes the control to a panel-level layout context for relocation by PS.3.
- **FR-7.** Panel search MUST match a control when the query fuzzy-matches its `label`, any `keyword`,
  or its section title, using the existing `fuzzyMatches` helper in `clients/web/src/lib/fuzzy-match.ts`.
- **FR-8.** When a search is active, a section MUST render only its matching controls, and a section
  with zero matches MUST be hidden entirely; the existing "No settings match …" empty state MUST
  remain when no control in the panel matches.
- **FR-9.** Controls that are conditionally rendered today (e.g. `Max attempts` only when
  `unlimitedAttempts` is false; lockdown controls only when `lockdownDeliveryEnabled`) MUST remain
  conditional; the registry MUST NOT force a hidden control to render.
- **FR-10.** Composite/embedded editors (`Assign to`, `Outcomes`/`Outcomes mapping`, `Rubric`) MUST be
  registered as **section-level** descriptors with `pinnable: true` and MUST NOT be decomposed into
  per-field descriptors in this plan.
- **FR-11.** A unit test MUST assert registry integrity: IDs unique, IDs match the naming regex, every
  `section` value is a real section for that `surface`, no alias points at a missing canonical ID.
- **FR-12.** A test MUST assert **registry/DOM parity**: every `settingId` passed to `<SettingRow>` in
  either panel exists in the registry, and every registry entry for a surface is reachable from that
  panel under some prop combination.
- **FR-13.** Existing DOM `id` attributes on inputs (e.g. `quiz-settings-due`) MUST be preserved so
  current e2e selectors and `htmlFor` label bindings keep working.

## 6. Non-Functional Requirements

- **Performance** — The registry is a static array (~60 entries); building lookup maps MUST add < 1 ms
  at module load. Panel re-render on keystroke MUST stay under 16 ms p95 on a mid-tier laptop; search
  filtering MUST be memoised per `(surface, query)`.
- **Security** — No new data crosses a trust boundary; the registry contains only labels and keywords
  already visible in the DOM. No PII.
- **Privacy & Compliance** — None; no new data collection in PS.1.
- **Accessibility** — WCAG 2.1 AA maintained. `<SettingRow>` MUST NOT introduce an extra focusable
  element, MUST NOT alter DOM order within a section, and MUST preserve every existing
  `label`/`htmlFor` association and `aria-*` attribute. Keyboard tab order MUST be unchanged when no
  search is active.
- **Scalability** — Registry design MUST tolerate ~200 descriptors (adding content-page/module
  surfaces later) without a lookup strategy change.
- **Reliability** — An unknown `settingId` passed to `<SettingRow>` MUST render the control normally
  and log a `console.warn` in dev builds; it MUST NOT throw or blank the panel.
- **Observability** — None required in PS.1 (telemetry lands in PS.4). Dev-mode warnings only.
- **Maintainability** — One registry entry per control; `SETTINGS_SECTION_SEARCH` in both panels is
  deleted and its keywords migrated into descriptors. TypeScript `satisfies` used so the section field
  is checked against each panel's `SettingsSectionId` union.
- **Internationalization** — Labels/keywords are English literals today, matching the rest of the
  panels. The descriptor MUST keep `label` and `keywords` as data (not JSX) so a future i18n pass can
  swap them for message keys without touching the panels.
- **Backward compatibility** — Purely internal refactor; no API, no schema, no user-visible change.

## 7. Acceptance Criteria

- **AC-1.** *Given* the quiz settings panel with no search query, *When* it renders, *Then* the visible
  sections, control order, labels, and DOM ids are byte-identical to the pre-refactor snapshot.
- **AC-2.** *Given* the quiz panel, *When* the instructor types `lockdown`, *Then* the Presentation
  section shows only the `Lockdown delivery` control (plus its dependent `Focus-loss flag threshold`
  when kiosk is selected) and every other section is hidden.
- **AC-3.** *Given* the assignment panel, *When* the instructor types `zzzz`, *Then* the panel shows
  the existing "No settings match "zzzz"" empty state.
- **AC-4.** *Given* `unlimitedAttempts` is true, *When* the quiz panel renders, *Then* the
  `quiz.attempts-grading.max-attempts` control is absent from the DOM even though it exists in the
  registry.
- **AC-5.** *Given* the registry, *When* the integrity unit test runs, *Then* it fails if any ID is
  duplicated, malformed, points at a non-existent section, or if an alias resolves to a missing ID.
- **AC-6.** *Given* a `<SettingRow settingId="quiz.bogus.control">`, *When* it renders in a dev build,
  *Then* the child control still renders and exactly one `console.warn` is emitted.
- **AC-7.** *Given* the assignment panel, *When* a keyboard user tabs through it with no active search,
  *Then* focus order matches the pre-refactor order recorded in the a11y test.
- **AC-8.** *Given* a search matches a control inside a collapsed accordion, *When* results render,
  *Then* that accordion is force-open (existing `forceOpen` behaviour) and the matched control is
  visible without further interaction.

## 8. Data Model

No database changes in PS.1. The persisted artefact introduced here is the **setting ID namespace**,
which PS.2 stores.

- Naming regex (enforced by test): `^(assignment|quiz)\.[a-z0-9]+(?:-[a-z0-9]+)*\.[a-z0-9]+(?:-[a-z0-9]+)*$`.
- Max ID length: **96 characters** (matches the column constraint PS.2 will add).
- Retirement policy: an ID is never deleted outright — it moves to `SETTING_ID_ALIASES` pointing at its
  replacement, or to a `RETIRED_SETTING_IDS: Set<string>` when the control is gone for good.
  `resolveSettingId` returns `null` for retired IDs so PS.3 can prune stale pins silently.

### Initial catalog (abridged; the full list ships in the registry module)

| Surface | Section | Setting ID | Label |
|---|---|---|---|
| quiz | scheduling | `quiz.scheduling.due-date` | Due date |
| quiz | scheduling | `quiz.scheduling.visible-from` | Visibility start |
| quiz | scheduling | `quiz.scheduling.visible-until` | Visibility end |
| quiz | attempts-grading | `quiz.attempts-grading.unlimited-attempts` | Unlimited attempts |
| quiz | attempts-grading | `quiz.attempts-grading.max-attempts` | Max attempts |
| quiz | attempts-grading | `quiz.attempts-grading.grade-policy` | Grade uses |
| quiz | attempts-grading | `quiz.attempts-grading.passing-score` | Passing score (%) |
| quiz | attempts-grading | `quiz.attempts-grading.points-worth` | Points worth |
| quiz | attempts-grading | `quiz.attempts-grading.late-policy` | Late submission (after due) |
| quiz | attempts-grading | `quiz.attempts-grading.late-penalty` | Late penalty (% of points) |
| quiz | grading | `quiz.grading.assignment-group` | Assignment group |
| quiz | grading | `quiz.grading.never-drop` | Never drop this score |
| quiz | grading | `quiz.grading.replace-with-final` | Use as final for replace-lowest |
| quiz | time-limits | `quiz.time-limits.total-minutes` | Total time limit (minutes) |
| quiz | time-limits | `quiz.time-limits.pause-when-hidden` | Pause timer when tab is hidden |
| quiz | time-limits | `quiz.time-limits.per-question-seconds` | Per-question time limit (seconds) |
| quiz | scores-review | `quiz.scores-review.show-score-timing` | When to show score |
| quiz | scores-review | `quiz.scores-review.visibility` | What learners can see |
| quiz | scores-review | `quiz.scores-review.when` | When they can review |
| quiz | presentation | `quiz.presentation.one-question-at-a-time` | One question at a time |
| quiz | presentation | `quiz.presentation.shuffle-questions` | Shuffle question order |
| quiz | presentation | `quiz.presentation.shuffle-choices` | Shuffle answer choices |
| quiz | presentation | `quiz.presentation.back-navigation` | Allow back navigation |
| quiz | presentation | `quiz.presentation.lockdown-mode` | Lockdown delivery |
| quiz | presentation | `quiz.presentation.focus-loss-threshold` | Focus-loss flag threshold |
| quiz | presentation | `quiz.presentation.random-pool-size` | Random question pool size |
| quiz | outcomes | `quiz.outcomes.mapping` | Outcomes *(composite)* |
| quiz | assign-to | `quiz.assign-to.editor` | Assign to *(composite)* |
| quiz | access | `quiz.access.access-code` | Quiz access code |
| quiz | adaptive-ai | `quiz.adaptive-ai.difficulty` | Difficulty target |
| quiz | adaptive-ai | `quiz.adaptive-ai.topic-balance` | Balance topics across sources |
| quiz | adaptive-ai | `quiz.adaptive-ai.stop-rule` | Stop rule |
| assignment | scheduling | `assignment.scheduling.due-date` | Due date |
| assignment | scheduling | `assignment.scheduling.visible-from` | Visibility start |
| assignment | scheduling | `assignment.scheduling.visible-until` | Visibility end |
| assignment | submission-type | `assignment.submission-type.text-entry` | Text entry |
| assignment | submission-type | `assignment.submission-type.file-upload` | File upload |
| assignment | submission-type | `assignment.submission-type.url` | Website URL |
| assignment | academic-integrity | `assignment.academic-integrity.originality-mode` | Originality checks |
| assignment | academic-integrity | `assignment.academic-integrity.student-visibility` | Student score visibility |
| assignment | late-submission | `assignment.late-submission.policy` | Policy |
| assignment | late-submission | `assignment.late-submission.penalty` | Late penalty (% of points) |
| assignment | grade-posting | `assignment.grade-posting.policy` | Posting policy (automatic/manual) |
| assignment | grade-posting | `assignment.grade-posting.release-at` | Release grades at |
| assignment | grading | `assignment.grading.blind-grading` | Blind grading |
| assignment | grading | `assignment.grading.moderated-grading` | Moderated grading |
| assignment | grading | `assignment.grading.agreement-threshold` | Agreement threshold (% of points) |
| assignment | grading | `assignment.grading.moderator` | Moderator |
| assignment | grading | `assignment.grading.points-worth` | Points worth |
| assignment | grading | `assignment.grading.assignment-group` | Assignment group |
| assignment | grading | `assignment.grading.never-drop` | Never drop this score |
| assignment | grading | `assignment.grading.replace-with-final` | Use as final for replace-lowest |
| assignment | grading | `assignment.grading.display-override` | Grade display override |
| assignment | rubric | `assignment.rubric.editor` | Rubric *(composite)* |
| assignment | outcomes-mapping | `assignment.outcomes-mapping.editor` | Outcomes mapping *(composite)* |
| assignment | assign-to | `assignment.assign-to.editor` | Assign to *(composite)* |
| assignment | access | `assignment.access.access-code` | Assignment access code |

> The implementer MUST reconcile this table against the panels at build time; FR-12's parity test is
> the enforcement mechanism, not this document.

## 9. API Surface

None. PS.1 ships no HTTP routes, no WebSocket events, and no OpenAPI changes.

## 10. UI / UX

- **New components** (`clients/web/src/components/settings-panel/`):
  - `setting-row.tsx` — `<SettingRow settingId, children>`; looks up the descriptor, evaluates the
    search predicate from context, renders `children` or `null`.
  - `settings-panel-context.tsx` — provides `{ surface, query, matches(id), register(id) }` to rows.
- **Modified components**: `assignment-page-settings-panel.tsx`, `quiz-page-settings-panel.tsx` —
  each control body wrapped in `<SettingRow>`; local `SETTINGS_SECTION_SEARCH` maps deleted;
  `settingsSectionVisible` replaced by "section has ≥ 1 matching row".
- **Key user flows**
  1. Instructor opens the settings sidebar → identical to today.
  2. Instructor types a query → matching controls appear, non-matching rows and empty sections
     disappear, matching sections force-open.
  3. Instructor clears the query → full panel restored with accordions back to manual open state.
- **Empty / loading / error states** — Reuses the existing "No settings match …" panel. Composite
  editors keep their own internal loading states. No new error state.
- **Mobile / responsive** — Unchanged; the sidebar's existing responsive behaviour is untouched.
- **Accessibility annotations** — `<SettingRow>` renders a plain `<div>` (or a fragment where the
  parent uses `divide-y` spacing so dividers stay correct); it adds no roles, no tabindex, no
  landmarks. Section headings, `htmlFor`, and `role="switch"` toggles are unchanged.
- **Copy & i18n keys** — No new copy. Labels and keywords move verbatim from the panels into the
  registry.

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- Internal modules touched:
  - `clients/web/src/components/assignment/assignment-page-settings-panel.tsx`
  - `clients/web/src/components/quiz/quiz-page-settings-panel.tsx`
  - `clients/web/src/lib/fuzzy-match.ts` (consumed, unchanged)
  - New: `clients/web/src/lib/settings-registry.ts`,
    `clients/web/src/components/settings-panel/{setting-row,settings-panel-context}.tsx`
- Consumers of the panels (must compile unchanged): `clients/web/src/pages/lms/course-module-assignment-page.tsx`,
  `clients/web/src/pages/lms/course-module-quiz-page.tsx`.
- No external services, no webhooks, no events.

## 13. Dependencies & Sequencing

- Must ship after: nothing.
- Must ship before: **PS.2** (needs the ID namespace to validate against), **PS.3** (needs
  `<SettingRow>` relocation), **PS.4** (needs IDs for telemetry).
- Shared infra needed: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Refactoring a 1669-line panel silently drops or reorders a control | M | H | Registry/DOM parity test (FR-12) + pre/post render snapshot test per panel (AC-1); land as one reviewable PR per panel |
| Setting IDs churn after PS.2 persists them, orphaning pins | M | M | IDs frozen by test + alias map (FR-4); `resolveSettingId` prunes unknown IDs client-side |
| Control-level search feels *worse* (too aggressive hiding) than section-level | L | M | Section title also matches (FR-7), so "presentation" still returns the whole section; dogfood before flipping PS.3's flag |
| `<SettingRow>` wrapper breaks Tailwind `divide-y` / `space-y` sibling spacing | M | L | Row renders a fragment where the parent relies on sibling selectors; visual regression check on both panels |
| Composite editors (Rubric, Assign to) fetch on mount and get remounted by filtering | M | M | Filtering hides via conditional render only at section level for composites; composites keep their existing mount lifecycle |

## 15. Rollout Plan

- **Feature flag** — None. PS.1 is behaviour-neutral and ships unflagged; the user-visible pinning
  behind `ff_pinned_settings` arrives in PS.2/PS.3.
- **Sequencing** — (1) registry module + tests, (2) quiz panel refactor (smaller, proves the pattern),
  (3) assignment panel refactor, (4) delete the old `SETTINGS_SECTION_SEARCH` maps.
- **Dogfood** — Internal instructors use the editors for one sprint before PS.3 lands.
- **GA criteria** — Both panels refactored, parity + snapshot tests green, no visual diff.
- **Rollback path** — Revert the PR; nothing persisted, nothing migrated.

## 16. Test Plan

- **Unit** — Registry integrity (uniqueness, regex, section validity, alias resolution, ID length ≤ 96);
  `resolveSettingId` for canonical / alias / retired / unknown; search predicate matching label,
  keyword, and section title; memoisation returns a stable reference for an unchanged query.
- **Integration (component)** — Render each panel with representative prop sets (adaptive on/off,
  lockdown on/off, with/without `courseCode` + item id) and assert: no-query snapshot parity,
  query filtering per AC-2/AC-3, conditional controls per AC-4, dev warning per AC-6.
- **End-to-end** — Existing `e2e/tests` specs that touch quiz and assignment settings MUST pass
  unchanged (they rely on the preserved DOM ids, FR-13). No new spec in PS.1.
- **Security** — Not applicable (no data flow change).
- **Accessibility** — `axe` on both panels with and without an active query; focus-order test per AC-7;
  screen-reader spot-check that section headings and control labels are announced as before.
- **Performance** — React Profiler check: keystroke-to-paint p95 < 16 ms with the assignment panel
  (largest) fully expanded.
- **Manual exploratory** — QA checklist: every accordion opened, every conditional control toggled,
  search for each section title, search for a control label, clear search, resize to mobile width.

## 17. Documentation & Training

- End-user docs: none (no visible change); note the sharper search in release notes only if PS.3
  ships in the same release.
- Internal: `clients/web/src/lib/settings-registry.ts` header comment documents the ID contract,
  the alias/retirement policy, and "add a setting" steps.
- API reference: no change.
- Runbook: none.

## 18. Open Questions

1. Should the server hold a mirrored registry to validate pinned IDs, or validate shape-only and let
   the client prune unknowns? **Proposed:** shape-only on the server (avoids Go/TS drift); decided in
   PS.2 §9.
2. Should composite editors (Rubric, Assign to, Outcomes) eventually decompose into per-field
   descriptors? Deferred until pin telemetry (PS.4) shows demand.
3. Does the `content-page` editor's settings surface warrant the same registry treatment in a
   follow-up (`PS.5`)? Out of scope here; revisit after GA.
4. Should section titles remain searchable once controls are individually searchable, or does that
   re-create today's "whole section explodes" complaint? Ship with section titles matching; revisit
   with PS.4 telemetry.

## 19. References

- Existing files: `clients/web/src/components/assignment/assignment-page-settings-panel.tsx`,
  `clients/web/src/components/quiz/quiz-page-settings-panel.tsx`,
  `clients/web/src/lib/fuzzy-match.ts`, `clients/web/src/lib/__tests__/fuzzy-match.test.ts`,
  `clients/web/src/pages/lms/course-module-quiz-page.tsx`,
  `clients/web/src/pages/lms/course-module-assignment-page.tsx`.
- Related plans: [PS.2](PS.2-pinned-settings-data-model-and-api.md),
  [PS.3](PS.3-pin-and-reorder-ux-in-editor-panels.md),
  [PS.4](../../plan/settings/PS.4-suggested-pins-telemetry-and-rollout.md).
- External standards: WCAG 2.1 AA (1.3.1 Info and Relationships, 2.4.3 Focus Order), RFC 2119.
