# CC.7 — Web: Checklist Page, Nav Entry & Outstanding-Items Badge

> Implementation plan. Source: Course Checklist product request. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CC.7 |
| **Section** | Course Checklist |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web client team |
| **Depends on** | CC.2 (API), CC.3 (real items to render) |
| **Unblocks** | CC.8, CC.10 |

---

## 1. Problem Statement

The course side navigation (`side-nav-course-links.tsx`) has 30+ destinations grouped into Content,
Collaboration, Your learning, Assessment, Grades & insights, People and Manage — and none of them answers
"what is left to do on this course?". CC.1–CC.6 produce the answer server-side; CC.7 is where a teacher
actually sees it: a **Checklist** entry directly under **Dashboard**, a badge showing the number of
outstanding essential items, and a page that crosses off what is done, expands a table of offenders for what
is not, sends the instructor straight to the fix, and keeps a dismissed pile for the items they have
consciously set aside.

## 2. Goals

- Add a **Checklist** nav item immediately below Dashboard, visible **only** to teachers-and-higher, with a
  live count badge when work remains.
- Render the checklist grouped by category with **crossed-off** done items, actionable todo items, progress
  counters and rubric citations.
- Support both interaction modes the product asked for: **expand a table** of offending entities, or
  **navigate directly** to the fix (CC.8 supplies the highlight).
- Provide **dismiss / restore** with a reason, and a **Dismissed** section that keeps set-aside items
  visible but out of the way.
- Meet WCAG 2.1 AA — this is a checklist about course quality; it cannot itself be inaccessible.

## 3. Non-Goals

- No mobile app work (CC.9); no server work (CC.1–CC.6).
- No deep-link highlight implementation — CC.7 consumes `target` and calls the CC.8 helper; if CC.8 has not
  shipped, targets navigate without a highlight.
- No AI remediation buttons (CC.10).
- No cross-course rollup view.
- No redesign of the existing side nav grouping beyond inserting one item.

## 4. Personas & User Stories

- **As a teacher**, I want a Checklist item right under Dashboard with a number on it, so that I know at a
  glance whether my course needs work.
- **As a teacher**, I want done items visibly crossed off, so that I feel progress rather than an endless
  list.
- **As a teacher**, I want to expand "11 assessments aren't mapped" into the actual eleven, so that I can
  work the list.
- **As a teacher**, I want clicking an item to take me to the exact control, so that I am not hunting
  through settings tabs.
- **As a teacher**, I want to dismiss an item that will never apply, with a reason my co-teacher can see, so
  that the list stays honest.
- **As a student**, I want to never see this nav item, so that instructor to-dos stay behind the curtain.
- **As a screen-reader user teaching a course**, I want the checklist to announce status and progress
  without relying on colour or strikethrough alone.

## 5. Functional Requirements

### Navigation

- **FR-1.** `side-nav-course-links.tsx` MUST render a `Checklist` link at
  `/courses/{courseCode}/checklist` **immediately after the Dashboard link**, before the Content section
  label, using the `ListChecks` icon (or `ClipboardCheck` if `ListChecks` collides with Question bank).
- **FR-2.** The link MUST render only when `canManageCourse` is true — the existing
  `allows(courseItemCreatePermission(courseCode))` predicate — matching the CC.2 FR-1 server guard, and MUST
  be hidden while permissions are loading (no flash).
- **FR-3.** When "View as: Student" is active (`useCourseViewAs`), the link MUST be hidden, consistent with
  how the preview hides staff surfaces.
- **FR-4.** The link MUST show a badge with `summary.outstandingEssential` when > 0, capped at `99+`, using
  the existing `badge` prop on `SideNavLink`. Zero outstanding ⇒ no badge (not a "0" chip).
- **FR-5.** The badge MUST have an accessible name of the form "8 checklist items need attention" via
  `aria-label` on the badge element, and MUST NOT be conveyed by colour alone.
- **FR-6.** In the collapsed side nav the badge MUST render in the existing collapsed position (`absolute
  end-2 top-2`) and the tooltip MUST include the count.
- **FR-7.** The course **Dashboard** page (`course-detail.tsx`) MUST show a compact checklist card for staff
  — progress ("18 of 26 done"), the top 3 outstanding essential items, and a link to the full page — placed
  above the grading-backlog card. Zero outstanding ⇒ a single "Your course checklist is complete" line.
- **FR-8.** The command palette (`command-palette-dialog.tsx`) MUST expose "Course checklist" as a
  destination for staff.

### Badge data

- **FR-9.** A `CourseChecklistSummaryProvider` context MUST fetch `GET .../checklist/summary` once per
  course, memoise for 60 s, and expose `{ summary, loading, refresh }`.
- **FR-10.** The provider MUST refresh on: course change, window focus after > 60 s idle, an explicit
  `refresh()` from the checklist page after a dismiss/restore/recheck, and after any successful mutation on
  a page whose route matches a known checklist target (a lightweight `invalidateChecklist()` hook called
  from course settings, modules, outcomes and enrollment save paths).
- **FR-11.** Summary fetch failures MUST be silent: no badge, no error toast, one `console.warn` in dev.
- **FR-12.** The provider MUST NOT fetch when the viewer lacks `canManageCourse`.

### Checklist page

- **FR-13.** Route `/courses/:courseCode/checklist` MUST be registered in `app.tsx` under the course layout
  and lazily loaded via `lazy-pages.ts`, consistent with the other course pages.
- **FR-14.** The page MUST render a header with: course name, an overall progress indicator
  (`done / total` plus a progress bar), `computedAt` as relative time, and a **Re-check** button calling
  `POST /checklist/refresh`.
- **FR-15.** Categories MUST render in server order as collapsible sections with `n outstanding` counts.
  Categories with zero outstanding items MUST default to collapsed; others expanded.
- **FR-16.** Each item row MUST show: status affordance, title, `why` (one line, expandable), `detail`,
  `progress` when present, tier indicator for `essential`, source citation chips, and an overflow menu.
- **FR-17.** `done` items MUST render with a **line-through title**, a check icon, and reduced emphasis —
  and MUST also carry a visually-hidden "Completed" text so status is not conveyed by decoration alone.
- **FR-18.** `todo` and `in_progress` items MUST be interactive. Interaction resolution:
  - if the item has `evidence` with ≥ 1 row → the row is a **disclosure** that expands an evidence table;
  - else if the item has a `target` → the row is a **link** that navigates (CC.8 highlight);
  - else → the row is static with its `detail` only.
- **FR-19.** The evidence table MUST render `evidence.columns` as `<th scope="col">`, one row per entity,
  with the first cell a link to that row's `target` (falling back to the item target). When
  `truncatedAt` is set, the table MUST show "Showing first 200 of N".
- **FR-20.** `unknown` items MUST render muted with "Couldn't check this right now" and a **Re-check**
  action calling `POST /checklist/items/{id}/recheck`.
- **FR-21.** Each item's overflow menu MUST offer **Dismiss** (opens a reason dialog: reason select +
  optional ≤ 500-char note) and, for done items, nothing else.
- **FR-22.** A **Dismissed** section MUST render below all categories, collapsed by default, showing each
  dismissed item with who dismissed it, when, the reason, the note, and a **Restore** action.
- **FR-23.** Dismiss and restore MUST update optimistically, roll back on failure with an inline error, and
  call `refresh()` on the summary provider so the badge stays in sync.
- **FR-24.** The page MUST support deep-linking to a single item via `#item-{itemId}`, scrolling it into
  view and expanding its category.
- **FR-25.** Empty states: no outstanding items ⇒ a celebratory but plain "Everything on the checklist is
  done" panel that still lists completed items behind a "Show completed" toggle; catalog empty ⇒ "Nothing to
  check right now."
- **FR-26.** Loading ⇒ skeleton rows matching the final layout (per the AN motion plan's skeleton→content
  choreography). Error ⇒ inline retry panel, never a blank page.
- **FR-27.** A non-staff user hitting the route directly MUST see the standard "you don't have access to
  this page" state (the API returns 403) and MUST NOT see any item titles.

## 6. Non-Functional Requirements

- **Performance** — First contentful render of the page < 400 ms on a warm API; the page MUST NOT block on
  the summary provider. Badge fetch adds one request per course visit (memoised 60 s). Evidence tables
  virtualise above 100 rows. Bundle impact ≤ 25 KB gzipped for the lazy chunk.
- **Security** — All gating is server-enforced (CC.2); client gating is UX only. No checklist data is
  persisted to `localStorage` or IndexedDB beyond the in-memory 60 s memo (evidence can name students).
- **Privacy & Compliance** — Evidence rows may contain student display names (CC.3 FR-31/33); the page MUST
  NOT include them in any analytics payload (CC.10 sends item IDs and statuses only).
- **Accessibility** — WCAG 2.1 AA. Specifically: categories are `<section>` with `<h2>`; each item is a
  list item in a `<ul>`; status is exposed as text, not only icon/colour/strikethrough (FR-17); disclosure
  buttons use `aria-expanded`/`aria-controls`; evidence tables are real tables with scoped headers; the
  progress bar uses `role="progressbar"` with `aria-valuenow/min/max` and an accessible name; the dismiss
  dialog is a focus-trapped `role="dialog"` returning focus to its trigger; live regions announce
  "Item dismissed" / "Item restored" / "Re-checked"; all interactive targets ≥ 44×44 CSS px; strikethrough is
  paired with `text-decoration` **and** a visually-hidden label because some AT ignores decoration.
- **Scalability** — Renders 120 items across 10 categories without jank; evidence virtualisation above 100
  rows; category collapse state persisted per course in `sessionStorage`.
- **Reliability** — Optimistic updates roll back; a failed summary fetch never breaks the shell; the page
  works when `evidence` is absent, when `target` is null, and when a category is empty.
- **Observability** — Client events (CC.10 defines the dictionary): page view, item expand, evidence-row
  click, target navigation, dismiss (with reason), restore, recheck, refresh. No PII in payloads.
- **Maintainability** — New files under `clients/web/src/pages/lms/course-checklist/` and
  `clients/web/src/components/checklist/`; API client in `clients/web/src/lib/course-checklist-api.ts`
  (a **new module**, not an addition to the 8.6K-line `courses-api.ts`, per TD.12's direction).
- **Internationalization** — All copy through the existing i18n layer; render `titleKey`/`whyKey` when a
  translation exists, else the server's English default. RTL verified (the repo ships `ar`); strikethrough,
  progress bar and badge position must mirror correctly.
- **Backward compatibility** — Unknown item fields ignored; unknown `status` values render as `unknown`;
  `catalogVersion` change busts the client memo.

## 7. Acceptance Criteria

- **AC-1.** *Given* a teacher on a course, *When* the side nav renders, *Then* a Checklist link appears
  directly below Dashboard with a badge showing the outstanding-essential count.
- **AC-2.** *Given* a student, *Then* no Checklist link renders and navigating to `/checklist` shows the
  no-access state with no item titles in the DOM.
- **AC-3.** *Given* "View as: Student" is active, *Then* the Checklist link is hidden.
- **AC-4.** *Given* `outstandingEssential = 0`, *Then* no badge renders (not a zero chip).
- **AC-5.** *Given* `outstandingEssential = 137`, *Then* the badge reads `99+` and its `aria-label` reads
  "137 checklist items need attention".
- **AC-6.** *Given* a done item, *Then* its title is struck through **and** a visually-hidden "Completed"
  label is present (asserted in the a11y test).
- **AC-7.** *Given* `outcomes.assessment-mapping` with 11 evidence rows, *When* the row is activated, *Then*
  a table expands with the declared column headers and 11 linked rows, and `aria-expanded` becomes `true`.
- **AC-8.** *Given* an evidence row is clicked, *Then* the app navigates to that row's target.
- **AC-9.** *Given* an item with a target and no evidence, *When* the row is activated, *Then* the app
  navigates to the item target.
- **AC-10.** *Given* the user dismisses an item with reason "Not applicable", *Then* the item moves to the
  Dismissed section, the badge decrements, and a live region announces the change.
- **AC-11.** *Given* the dismiss request fails, *Then* the item returns to its category and an inline error
  appears; the badge is unchanged.
- **AC-12.** *Given* a dismissed item is restored, *Then* it returns to its category with its live status.
- **AC-13.** *Given* an `unknown` item, *Then* a Re-check action is offered and, on success, the item
  updates in place without a full page reload.
- **AC-14.** *Given* the page in a loading state, *Then* skeleton rows render and no layout shift occurs when
  data arrives (CLS < 0.1).
- **AC-15.** *Given* an axe scan of the page in both themes and in RTL, *Then* there are zero serious or
  critical violations.
- **AC-16.** *Given* keyboard-only navigation, *Then* every item, disclosure, evidence link, overflow menu
  and dialog is reachable and operable, with visible focus and no traps.

## 8. Data Model

No client persistence beyond:

- `sessionStorage` key `checklist:categories:{courseCode}` — collapsed/expanded state, boolean map. Cleared
  on catalog-version change.
- In-memory summary memo (60 s) in the provider.

No `localStorage`, no IndexedDB, no offline cache — evidence can name students (§6 Privacy).

## 9. API Surface

Consumes CC.2 only; adds no endpoints. New client module
`clients/web/src/lib/course-checklist-api.ts`:

```ts
export async function fetchCourseChecklist(courseCode: string): Promise<ChecklistResponse>
export async function fetchCourseChecklistSummary(courseCode: string): Promise<ChecklistSummary>
export async function refreshCourseChecklist(courseCode: string): Promise<ChecklistResponse>
export async function dismissChecklistItem(
  courseCode: string, itemId: string,
  body: { reason: DismissReason; note?: string }): Promise<ChecklistItem>
export async function restoreChecklistItem(courseCode: string, itemId: string): Promise<ChecklistItem>
export async function recheckChecklistItem(courseCode: string, itemId: string): Promise<ChecklistItem>
```

Types mirror CC.2 §9 and live in `course-checklist-api-schemas.ts` with runtime validation matching the
existing `courses-api-schemas.ts` convention.

## 10. UI / UX

**Nav placement**

```
Back
Dashboard
Checklist            (8)      ← new, staff only
── Content ──
Files / Modules / Syllabus …
```

**Page layout**

```
Course checklist                                    [ Re-check ]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
18 of 26 done · 5 need attention · checked 4 minutes ago
[▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░]

▾ Foundations & orientation                        2 outstanding
  ✓  ̶S̶e̶t̶ ̶c̶o̶u̶r̶s̶e̶ ̶s̶t̶a̶r̶t̶ ̶a̶n̶d̶ ̶e̶n̶d̶ ̶d̶a̶t̶e̶s̶                  Completed
  ✓  ̶P̶o̶s̶t̶ ̶a̶ ̶w̶e̶l̶c̶o̶m̶e̶ ̶a̶n̶n̶o̶u̶n̶c̶e̶m̶e̶n̶t̶                     Completed
  ○  Publish your course                                      ⋯
     Students can't see the course. It starts in 3 days.   QM 1.2
  ○  Add students to the course                               ⋯
     3 invitations are pending.                    ▸ Show the 3

▾ Outcomes & alignment                             1 outstanding
  ◐  Map every assessment to an outcome        13 / 24         ⋯
     11 of 24 assessments aren't mapped.         QM 3.1 · NSQ C
     ▾ Show the 11
       ┌───────────────────┬──────────┬───────────┬────────┐
       │ Item              │ Type     │ Module    │ Points │
       ├───────────────────┼──────────┼───────────┼────────┤
       │ Essay 1           │ Assign.  │ Week 2    │ 100    │
       │ Midterm quiz      │ Quiz     │ Week 6    │ 150    │
       └───────────────────┴──────────┴───────────┴────────┘

▸ Dismissed (3)
```

**Interaction flows**

1. *Fix via table*: item → "Show the 11" → row → item editor with the outcomes section focused (CC.8) → map
   → back → item auto-rechecks → row disappears.
2. *Fix directly*: item → target route with the control highlighted → save → back → recheck.
3. *Dismiss*: overflow → Dismiss → reason + note → item moves to Dismissed, badge decrements.
4. *Restore*: Dismissed → Restore → item returns with live status.

**States** — loading skeleton; error retry panel; `unknown` muted with re-check; all-done celebration panel
with "Show completed"; no-access panel.

**Responsive** — Single column below `md`; evidence tables become stacked definition rows below `sm` with
each row a full-width link; the header progress bar wraps under the title; sticky category headers on
mobile widths.

**Motion** — Disclosure expand/collapse and the strike-through transition use the shared spring tokens from
the AN motion work; all animation is disabled under `prefers-reduced-motion`.

## 11. AI / ML Considerations

None in CC.7. The item row reserves a slot for an optional primary action button which CC.10 will populate
with AI-assisted fixes (e.g. "Suggest mappings", "Build a rubric"); CC.7 ships the slot rendering nothing.

## 12. Integration Points

- Modified: `clients/web/src/components/layout/side-nav-course-links.tsx` (nav item + badge),
  `clients/web/src/app.tsx` + `lazy-pages.ts` (route), `clients/web/src/pages/lms/course-detail.tsx`
  (dashboard card), `clients/web/src/components/command-palette/command-palette-dialog.tsx`,
  `clients/web/src/lib/search-course-features.ts` (make "checklist" findable in course search).
- New: `pages/lms/course-checklist/` (page, category, item, evidence table, dismiss dialog),
  `components/checklist/` (badge, progress, status affordance),
  `context/course-checklist-summary-context.tsx`, `lib/course-checklist-api.ts` + schemas.
- Reuses: `SideNavLink` `badge` prop, `usePermissions`, `useCourseViewAs`, `LmsPage`, existing dialog and
  table primitives, i18n layer, `formatTimeAgoFromIso`.
- No new third-party dependencies.

## 13. Dependencies & Sequencing

- Must ship after: CC.2 (API) and at least one rule pack (CC.3) so the page is not empty.
- Should ship with or before CC.8 so targets highlight; degrades gracefully if CC.8 is later.
- Must ship before: CC.9 (mobile parity references this IA), CC.10 (telemetry hooks into these components).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Badge fetch on every course page multiplies API load | M | M | 60 s memo + snapshot-served summary (CC.2); provider skips non-staff; metric on summary QPS |
| A long checklist feels like a wall of failure | **H** | M | Done items rendered and crossed off (progress is visible), categories with no outstanding items collapse, only essential items badge, dismissal always one click away |
| Strikethrough is invisible to some AT | M | M | FR-17 visually-hidden "Completed" label; AC-6 asserts it |
| Evidence tables break on narrow screens | M | M | Stacked-row responsive mode below `sm`; tested at 320 px |
| Nav item pushes an already-long side nav longer | M | L | Single item placed at the top where it is most useful; no new section label |
| `courses-api.ts` grows further | M | M | New dedicated API module (TD.12 direction), asserted by a lint rule on file size for the new module |

## 15. Rollout Plan

**No feature flag.** The nav item and page ship on for teachers-and-higher in one release. Controls:

- The catalog itself is the staging mechanism: with only CC.3's `recommended` rules live, the badge stays at
  0 and the page is informational, so shipping the UI early is low-risk.
- Sequence: CC.2 API → CC.3 rules (recommended) → CC.7 UI → tier promotions (badge starts showing numbers).
- Dogfood on internal courses for one week before the first `essential` promotion, so the first time a badge
  appears the page behind it is polished.
- GA criteria: axe clean in both themes and RTL, keyboard walkthrough signed off, p75 page render < 600 ms,
  summary endpoint p95 < 40 ms under real traffic.
- Rollback: revert the client release. Because the API is additive and the server keeps working, a client
  rollback is complete and safe.

## 16. Test Plan

- **Unit** (Vitest + RTL) — Nav visibility per role and per view-as state; badge cap and `aria-label`;
  item rendering per status; interaction resolution (evidence vs target vs static, FR-18); optimistic
  dismiss + rollback; category collapse persistence; catalog-version memo invalidation; unknown-status
  fallback.
- **Integration** — API client module against a mocked server: response validation, error mapping, 403
  handling, refresh rate-limit response.
- **End-to-end** (Playwright) — Teacher: nav badge visible → open page → expand evidence → click a row →
  land on the target → fix → return → item rechecks to done → badge decrements. Dismiss → appears in
  Dismissed → restore. Student: no nav item, `/checklist` shows no-access.
- **Security** — Assert no item titles or evidence appear in the DOM or network responses for a student;
  assert no checklist data written to `localStorage`/IndexedDB.
- **Accessibility** — axe on the page in light/dark and LTR/RTL (zero serious/critical); keyboard-only
  script covering every control; screen-reader script (VoiceOver + NVDA) verifying status announcement,
  progress bar, disclosure state and dialog focus return; reduced-motion verification.
- **Performance / load** — Lighthouse on the page (targets consistent with the existing LH remediation
  budgets); render benchmark at 120 items / 200 evidence rows; bundle-size assertion on the lazy chunk.
- **Manual exploratory** — QA matrix: 320 px width, RTL locale, collapsed side nav, 0-item course, all-done
  course, all-dismissed course, `unknown` items, slow network.

## 17. Documentation & Training

- Help-centre: "Using the course checklist" — what it is, who sees it, how dismissal works, that it is
  guidance not gatekeeping.
- Screenshot set for the instructor onboarding tour; add a step pointing at the new nav item for
  first-time course creators.
- `clients/web/README` component notes for the new checklist components.
- Release note copy for the in-app banner announcing the feature.

## 18. Open Questions

1. Should the Dashboard card (FR-7) be dismissable independently of the checklist items? Proposed: yes,
   per-user, stored in existing dashboard preferences — needs a decision.
2. Should the badge count `essential` only, or all outstanding items? Proposed: essential only, so the
   number stays actionable; the page shows both.
3. Live updates: poll `summary` on route change (proposed) or subscribe to the existing course-structure
   websocket? The websocket already fires on the mutations that matter; worth a spike.
4. Should completed items be hidden by default with a "Show completed" toggle, rather than always shown?
   Proposed: always shown (crossing-off is the requested behaviour and the reward), with the toggle only in
   the all-done state.
5. Does the checklist deserve a place in the course creation flow ("start your checklist") rather than only
   the nav? Deferred to CC.10 rollout.

## 19. References

- Existing files this work touches: `clients/web/src/components/layout/side-nav-course-links.tsx`,
  `.../side-nav-link.tsx` (badge prop already supported), `clients/web/src/app.tsx`,
  `clients/web/src/lazy-pages.ts`, `clients/web/src/pages/lms/course-detail.tsx`,
  `clients/web/src/components/command-palette/command-palette-dialog.tsx`,
  `clients/web/src/lib/search-course-features.ts`, `clients/web/src/context/use-permissions.ts`,
  `clients/web/src/lib/course-view-as.ts`.
- Precedent: the unread-badge pattern in `side-nav-main-links.tsx`; the grading-backlog card in
  `components/dashboard/grading-backlog-list.tsx`.
- Related plans: [CC.2](CC.2-checklist-state-api-and-dismissals.md),
  [CC.8](CC.8-deep-link-and-highlight-targeting.md), [CC.9](CC.9-mobile-checklist-ios-and-android.md),
  [CC.10](CC.10-analytics-guidance-and-rollout.md); motion tokens from
  [`docs/completed/animations/`](../../completed/animations/); frontend structure direction from
  [TD.12](../tech_debt/TD.12-split-courses-api-module.md).
