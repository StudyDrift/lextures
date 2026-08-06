# UX.11 — Data Table and Gradebook System

> Implementation plan. Source: [audit.md](audit.md) §5 G-11, G-15.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.11 |
| **Section** | UI/UX — Core Surfaces |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | THIN — 99 hand-rolled tables, no shared component, 3,000-line gradebook |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Web + Product Design |
| **Depends on** | UX.1, UX.2, UX.3, UX.5 |
| **Unblocks** | Instructor efficiency; admin operations; UX.14 |

---

## 1. Problem Statement

Tables are where instructors and administrators spend their working day, and the
product has no table system. **99 files contain `<table>`; 81 wrap it in
`overflow-x-auto`**, leaving ~18 that force horizontal page scroll on narrow
viewports. Column sizing, sticky headers, sorting, filtering, selection, keyboard
navigation, empty rows, pagination and density are re-solved in every one.
`gradebook-grid.tsx` is 2,030 lines with a separate 1,000-line transposed variant.
Per **R-31**, the 2025–2026 movement in professional tools is *back to excellent
tables* — dense, inline-editable, keyboard-driven — and this is precisely the
surface where Lextures competes against Canvas and Blackboard on instructor
efficiency.

## 2. Goals

- Ship one **`DataTable`** system covering sorting, filtering, selection,
  pagination, density, sticky headers/columns, virtualisation, keyboard grid
  navigation and export.
- Migrate the 99 tables onto it, guaranteeing responsive and accessible behaviour
  by construction.
- Rebuild the gradebook on it as a genuine **spreadsheet-grade** surface: keyboard
  entry, inline editing, bulk actions, undo.
- Make dense data legible via the UX.3 type scale and tabular numerals.

## 3. Non-Goals

- Changing grading logic, weighting, curving or what-if calculation.
- Replacing the gradebook's data model or endpoints.
- Building a general BI/reporting tool.
- Charts and visualisations (a separate future plan).
- Native client tables.

## 4. Personas & User Stories

- **As an instructor**, I want to enter a column of grades with the keyboard only,
  moving down with Enter, like a spreadsheet.
- **As an instructor**, I want student names and assignment headers pinned while I
  scroll a large gradebook.
- **As an instructor**, I want to filter to ungraded submissions and act on them
  in bulk.
- **As an instructor who mis-clicked**, I want to undo a grade change without a
  confirmation dialog on every keystroke.
- **As an administrator**, I want to sort, filter and export any list in the
  product using the same controls.
- **As a screen-reader user**, I want tables announced with proper headers so I
  know which student and which assignment a cell belongs to.
- **As a user on a laptop**, I want tables to be usable without horizontal page
  scroll.

## 5. Functional Requirements

### `DataTable` core

- **FR-1.** A single `DataTable` MUST provide: column definitions (header, accessor,
  width, alignment, sortability, stickiness), sorting (single and multi), filtering,
  global search, row selection (single/multi/range), pagination or infinite scroll,
  density (comfortable/compact), column visibility and reordering, and empty/
  loading/error states.
- **FR-2.** `DataTable` MUST render semantic `<table>` markup with `<th scope>`,
  `<caption>` (or `aria-labelledby`), and `aria-sort` on sortable headers.
- **FR-3.** Horizontal overflow MUST be contained **within the table container**,
  never the page. A lint rule MUST forbid `<table>` outside `DataTable`.
- **FR-4.** Row virtualisation MUST engage automatically beyond 200 rows.
- **FR-5.** Sticky headers and sticky leading columns MUST be supported and MUST
  respect the [UX.5](UX.5-wcag-2.2-aa-conformance-uplift.md) focus-not-obscured
  requirement.
- **FR-6.** Density MUST be user-selectable and persisted per table per user.
- **FR-7.** Column resize and reorder MUST have single-pointer and keyboard
  alternatives (UX.5 FR-5/FR-6).
- **FR-8.** Selection state MUST survive sorting, filtering and pagination, and the
  count MUST always be visible ("12 of 340 selected").
- **FR-9.** Export (CSV) MUST be available on every table, respecting the current
  filter and column visibility.

### Grid navigation and editing

- **FR-10.** Editable tables MUST implement the **WAI-ARIA `grid` pattern**:
  arrow keys move between cells, `Tab` moves between widgets, `Enter` enters edit
  mode, `Escape` cancels, `Home`/`End` and `Ctrl+Home`/`Ctrl+End` jump.
- **FR-11.** In a grade column, `Enter` MUST commit and move **down**; `Tab` MUST
  commit and move **right** — the spreadsheet convention.
- **FR-12.** Inline edits MUST be **optimistic** (**R-23**): the cell updates
  immediately, reverts with an explanation on failure, and the failure is
  announced.
- **FR-13.** Cell state MUST be visible: saved, saving, failed, excused, missing,
  late, dropped, overridden — using icon + text, never colour alone.
- **FR-14.** Copy/paste of a column of values MUST be supported.
- **FR-15.** Bulk actions on a selection MUST be available (set grade, excuse,
  message students, post/hide grades).

### Undo instead of confirmation (**R-20/R-21**)

- **FR-16.** Reversible table actions (grade change, excuse, hide/post, bulk set)
  MUST use **toast + undo**, not a confirmation dialog.
- **FR-17.** Irreversible or high-blast-radius actions (delete a column with
  submissions, submit final grades to SIS) MUST use a confirmation dialog naming
  the object and consequence, with a verb-specific button (**R-22**).

### Accessibility of tabular data

- **FR-18.** Every data cell MUST be programmatically associated with its row and
  column headers so AT announces "Ada Lovelace, Quiz 3, 18 out of 20".
- **FR-19.** Sort changes, filter changes and bulk-action results MUST be
  announced via a live region.
- **FR-20.** Numeric columns MUST use tabular numerals
  ([UX.3](UX.3-typography-and-reading-system.md) FR-6) and right alignment.

### Responsive

- **FR-21.** Below `md`, tables MUST switch to a **card/list presentation** with
  the primary columns surfaced, or an explicitly horizontally-scrollable region
  with a visible affordance — never a silently clipped table.
- **FR-22.** Touch targets in tables MUST meet the UX.5 24px minimum (44px on
  touch-primary).

## 6. Non-Functional Requirements

- **Performance** — 1,000 rows × 50 columns MUST scroll at 60fps with no long task
  >50 ms. Cell edit INP ≤200 ms. Initial gradebook render ≤1.5 s for a 200-student
  course. `DataTable` MUST be a separate chunk (≤25 KB gzip) so tables do not tax
  the entry bundle.
- **Security** — Export MUST enforce the same authorisation as the view and MUST be
  audit-logged for gradebook and enrollment data. Bulk actions MUST re-check
  authorisation server-side per affected row, never trusting a client-supplied set.
- **Privacy & Compliance** — Gradebook and enrollment exports contain FERPA-covered
  education records. Exports MUST be logged with actor, scope and row count, per
  `../standards/S09-ferpa-hardening.md`.
- **Accessibility** — The `grid` pattern (FR-10) is the acceptance bar; a table
  that cannot be operated by keyboard is not shippable.
- **Scalability** — Adding a table means declaring columns. The system must serve
  the 99 existing tables plus growth.
- **Reliability** — Optimistic edits MUST reconcile correctly under concurrent
  edits by a co-teacher; conflicts MUST be surfaced, not silently overwritten.
- **Observability** — Emit `table_sort`, `table_filter`, `table_export`,
  `grade_cell_edit`, `grade_edit_failed`, `bulk_action`, `undo_used`. `undo_used`
  rate validates the FR-16 policy.
- **Maintainability** — FR-3 lint; gradebook decomposed per
  [`TD.14`](../tech_debt/TD.14-decompose-god-components.md) to ≤300 lines per file.
- **Internationalization** — Column headers from i18n keys; number, date and
  percentage formatting locale-aware; table mirrors in RTL with correct arrow-key
  inversion.
- **Backward compatibility** — No change to grading semantics. Existing exports
  keep their column order and format unless explicitly changed.

## 7. Acceptance Criteria

- **AC-1.** *Given* the codebase, *When* the lint runs, *Then* `<table>` outside
  `DataTable` occurs **0** times and the allowlist is empty.
- **AC-2.** *Given* any table at 390px, *When* rendered, *Then* the page does not
  scroll horizontally; the table either reflows to cards or scrolls within its own
  container with a visible affordance.
- **AC-3.** *Given* the gradebook, *When* an instructor types a grade and presses
  `Enter`, *Then* it saves optimistically and focus moves to the cell below.
- **AC-4.** *Given* a failed grade save, *When* it fails, *Then* the cell reverts,
  an explanation is shown and announced, and the entered value is recoverable.
- **AC-5.** *Given* a gradebook with 200 students × 50 columns, *When* scrolled,
  *Then* headers and the name column stay pinned and there is no long task >50 ms.
- **AC-6.** *Given* a screen reader on any data cell, *When* focused, *Then* the
  row and column headers are announced with the value.
- **AC-7.** *Given* a sort or filter change, *When* applied, *Then* the result is
  announced via a live region.
- **AC-8.** *Given* an instructor changes a grade, *When* it saves, *Then* a toast
  with **Undo** appears and undo restores the previous value.
- **AC-9.** *Given* an instructor deletes a gradebook column with submissions,
  *When* they act, *Then* a confirmation dialog names the column and the number of
  affected submissions, with a verb-specific button.
- **AC-10.** *Given* a selection of 12 rows, *When* the user sorts and filters,
  *Then* the selection persists and the count remains visible.
- **AC-11.** *Given* an export, *When* performed on gradebook data, *Then* it is
  audit-logged with actor, scope and row count.
- **AC-12.** *Given* keyboard-only operation, *When* a user navigates the
  gradebook, *Then* the full ARIA `grid` contract works, including RTL inversion.
- **AC-13.** *Given* the top 20 tables, *When* axe runs in all four themes,
  *Then* 0 violations.
- **AC-14.** *Given* the decomposition, *When* measured, *Then* no gradebook file
  exceeds 300 lines.
- **AC-15.** *Given* two teachers editing the same cell concurrently, *When* both
  save, *Then* the conflict is surfaced and neither value is silently lost.

## 8. Data Model

```sql
-- server/migrations/NNN_user_table_preferences.sql
CREATE TABLE user_table_preferences (
  user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  table_key   text        NOT NULL,       -- e.g. 'course.gradebook'
  density     text        NOT NULL DEFAULT 'comfortable',
  columns     jsonb       NOT NULL DEFAULT '[]'::jsonb,  -- [{id, visible, width, rank}]
  sort        jsonb       NOT NULL DEFAULT '[]'::jsonb,
  updated_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, table_key),
  CONSTRAINT user_table_preferences_density_chk
    CHECK (density IN ('comfortable', 'compact'))
);
```

- **Backfill** — none; absent row means defaults.
- Unknown column ids dropped on read against the column definitions.
- Export audit entries reuse the **existing** audit-log table; no new table.
- Cascade delete satisfies `../standards/S02-data-retention-deletion-engine.md`.

## 9. API Surface

```ts
// GET  /api/v1/users/me/table-preferences/{tableKey}    (auth: self)
// PUT  /api/v1/users/me/table-preferences/{tableKey}    (auth: self)
// DELETE /api/v1/users/me/table-preferences/{tableKey}  (auth: self) — reset

// Bulk grade mutation (replaces N single writes) — authz re-checked per row.
// POST /api/v1/courses/{code}/gradebook/bulk            (auth: gradebook write)
type BulkGradeRequest = {
  operations: { studentId: string; columnId: string; value: string | null; excused?: boolean }[]
  idempotencyKey: string
}
type BulkGradeResponse = {
  applied: number
  failed: { studentId: string; columnId: string; reason: string }[]
  undoToken: string            // consumed by POST .../gradebook/undo
}

// POST /api/v1/courses/{code}/gradebook/undo            (auth: gradebook write)
type UndoRequest = { undoToken: string }
```

- `idempotencyKey` makes bulk writes safe to retry.
- `undoToken` MUST expire (suggested 60 s, matching the toast duration) and be
  single-use.
- Existing single-cell endpoints remain for compatibility.
- **OpenAPI** — all new routes documented; `make openapi-check` passes.

## 10. UI / UX

- **New pages** — none. The UX.2 gallery gains a "Tables" section.
- **Modified pages** — all 99 table-bearing files; most significantly
  `gradebook-grid.tsx` (2,030 lines), `gradebook-grid-transposed.tsx`,
  `course-gradebook.tsx`, `course-enrollments.tsx` (1,913 lines),
  `course-standards-gradebook.tsx`, `pages/admin/**` lists.
- **Key user flows**
  1. Instructor opens gradebook → filters to "ungraded" → types down a column with
     `Enter` → each cell saves optimistically → a mistake is undone from the toast.
  2. Instructor selects 12 students → "Excuse assignment" → one toast, one undo.
  3. Admin sorts a user list, hides three columns, exports the filtered view.
  4. Instructor on a laptop opens the gradebook → headers pinned, no page-level
     horizontal scroll.
- **States** — table: loading (skeleton rows matching expected count), empty
  (onboarding copy + action, **R-18**), filtered-empty (distinct from truly empty:
  "No rows match — clear filters"), error (retry), offline (read-only cached with
  indicator). Cell: default, editing, saving, saved, failed, excused, missing,
  late, dropped, overridden, conflicted.
- **Mobile/responsive** — below `md`, list/card presentation with primary columns;
  the gradebook specifically offers a single-student or single-assignment view
  rather than a clipped grid.
- **Accessibility annotations** — `role="grid"` with `aria-rowcount`/
  `aria-colcount` for virtualised tables; `aria-sort`; `aria-selected` on rows;
  live-region announcements for sort/filter/bulk results; focus never lost during
  virtualisation.
- **Copy & i18n** — column headers, cell state labels and bulk-action copy from
  i18n keys at parity across four locales. Cell-state labels reuse
  `components/ui/status-vocabulary.tsx`.

## 11. AI / ML Considerations

Not AI-touching in v1. The gradebook surfaces output from the **existing** grading
agent (`components/annotation/grader-agent/`) as ordinary cell state — an
AI-suggested grade MUST be visibly distinguished from a human-entered one and MUST
require explicit acceptance. No new model, prompt, or inference is introduced by
this plan; the fidelity and oversight requirements of the shipped grading-agent
work continue to apply unchanged.

## 12. Integration Points

- **External** — a virtualisation library (see §18 Q1). No other new dependency.
- **Internal**
  - `clients/web/src/components/ui/` — `DataTable` and cell primitives
  - `clients/web/src/pages/lms/gradebook/**` (4 files, ~4,000 lines)
  - `clients/web/src/pages/lms/course-gradebook.tsx`,
    `course-standards-gradebook.tsx`, `course-enrollments.tsx`,
    `course-my-grades.tsx`, `course-mastery-heatmap.tsx`
  - `clients/web/src/pages/admin/**` — admin lists
  - `clients/web/src/lib/courses-api.ts` — bulk/undo endpoints
    (coordinates with [`TD.12`](../tech_debt/TD.12-split-courses-api-module.md))
  - `clients/web/src/components/annotation/grader-agent/**` — AI-suggested cells
  - `clients/web/src/lib/lms-toast.ts` — `toastWithUndo` becomes load-bearing
  - `server/internal/httpserver` — bulk, undo, table-preference routes
- **Events** — table telemetry into `server/internal/telemetry`; export events into
  the existing audit log.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.1](UX.1-semantic-design-token-system.md),
  [UX.2](UX.2-core-component-library-and-adoption-ratchet.md) (Table primitives),
  [UX.3](UX.3-typography-and-reading-system.md) (tabular numerals, dense scale),
  [UX.5](UX.5-wcag-2.2-aa-conformance-uplift.md) (target size, drag alternatives).
- **Should ship alongside** — [UX.13](UX.13-feedback-undo-and-destructive-actions.md)
  (undo policy) and [`TD.14`](../tech_debt/TD.14-decompose-god-components.md).
- **Must ship before** — [UX.14](UX.14-responsive-and-small-viewport-experience.md)
  can close its table findings.
- **Shared infra** — instructor participant recruitment for usability testing.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Gradebook regressions affect grades — the highest-stakes data in the product | M | **Critical** | Characterisation tests on the existing grid before any change; parallel-run validation (old and new compute the same values) on seeded and pilot data; flag-gated per course; no GA until 30 days of zero grade-discrepancy reports |
| Optimistic edits diverge from server truth under concurrency | M | **H** | Version/ETag per cell; explicit conflict state (FR-13, AC-15); never silently overwrite |
| Undo token expiry leaves users believing an action is reversible when it is not | M | M | Toast shows a countdown; after expiry the action is described as final; undo failure is explicit, never silent |
| 99-table migration stalls | H | M | Ratcheting lint (FR-3); migrate by directory; admin lists first (simple), gradebook last (hard) |
| Virtualisation breaks screen-reader row counts and focus | M | **H** | `aria-rowcount`/`aria-colcount` on the virtual grid; explicit AT testing on a 1,000-row table; focus preservation tests |
| Bulk actions become a foot-gun at scale | M | **H** | Server re-checks authz per row; result summary shows applied vs failed; undo covers the whole batch |
| Card reflow on mobile loses the comparative value of a table | M | M | For the gradebook specifically, offer single-student / single-assignment views rather than a card list |

## 15. Rollout Plan

- **Feature flag** — `ffDataTable` for the general migration (per directory) and a
  separate `ffGradebookV2` gated **per course**, so a pilot instructor can opt in.
- **Sequencing**
  1. `DataTable` built, gallery entries, axe and keyboard conformance green.
  2. Migrate admin lists (simple, low risk) — proves the component.
  3. Migrate enrollments, my-grades, reports.
  4. Bulk + undo endpoints server-side.
  5. Gradebook rebuilt behind `ffGradebookV2`, with parallel-run validation.
  6. Pilot courses → 10% of courses → 50% → GA.
  7. Lint flipped to error; allowlist deleted.
- **Dogfood** — internal org for admin lists; volunteer instructors for the
  gradebook, with an explicit opt-out.
- **GA criteria** — AC-1…AC-15 green; zero grade discrepancies in parallel-run;
  instructor task-time on "grade a column of 30" improved vs baseline.
- **Rollback** — `ffGradebookV2` off per course restores the current grid. Bulk and
  undo endpoints are additive and remain.

## 16. Test Plan

- **Unit** — column definition resolution; sort/filter/selection state machines;
  selection persistence across sort and pagination; virtualisation windowing;
  grid keyboard navigation including RTL; optimistic edit apply/revert; undo token
  lifecycle; density and preference merge.
- **Integration** — bulk endpoint authz per row (including a row the actor may not
  grade); idempotency on retry; undo token single-use and expiry; conflict
  detection with concurrent writers; export authz and audit logging.
- **End-to-end** — Playwright: type down a grade column with `Enter`; paste a
  column; select-filter-sort-bulk-undo; export a filtered view; 390px table
  behaviour; concurrent-edit conflict between two sessions.
- **Security** — bulk operation with a forged student id; export by an
  under-privileged user; audit-log completeness for exports; undo token replay.
- **Accessibility** — axe on the top 20 tables × 4 themes (AC-13); screen-reader
  scripts: identify a cell's row and column headers; change sort and hear the
  announcement; navigate a virtualised 1,000-row grid; complete a grade entry
  keyboard-only; RTL arrow behaviour.
- **Performance / load** — 1,000 × 50 scroll profiling (AC-5); cell edit INP;
  gradebook initial render for 200 students; chunk size gate.
- **Correctness (grades)** — parallel-run harness comparing old and new grid
  computed values across seeded courses and, during pilot, live courses.
- **User research** — 6 moderated instructor sessions on grading a column, bulk
  excusing, and finding ungraded work; task-time baseline captured first.
- **Manual exploratory** — QA checklist per table type × density × theme × RTL ×
  offline.

## 17. Documentation & Training

- **End-user (student)** — none beyond "My grades" behaving consistently.
- **Instructor** — help-centre: "Grading with the keyboard", "Bulk actions and
  undo", "Exporting your gradebook"; a short in-product tour on first
  `ffGradebookV2` exposure.
- **Admin** — "Exporting data and what gets logged".
- **Engineer** — `docs/guides/data-table.md`: declaring columns, the grid keyboard
  contract, when virtualisation engages, optimistic-edit and conflict handling, the
  undo-vs-confirm rule.
- **API reference** — OpenAPI for bulk, undo and table-preference routes.
- **Runbook** — "An instructor reports a wrong grade after a bulk action": how to
  read the audit log and the bulk operation record.

## 18. Open Questions

1. Virtualisation library: TanStack Virtual, `react-window`, or in-house?
   *Recommendation: TanStack Virtual — small, headless, works with semantic
   tables. Confirm bundle impact against the 25 KB budget.*
2. Do we also adopt TanStack Table for state management, or keep table state
   in-house? *Recommendation: adopt — sorting/filtering/selection state machines
   are exactly the thing not worth hand-rolling 99 times.*
3. Should the transposed gradebook view survive, or become a density/orientation
   option of one grid? *Recommendation: one grid with an orientation toggle —
   1,000 lines of duplicate logic is the current cost.*
4. What is the correct undo window? 60 s matches the toast, but a grading session
   may want longer. Consider a per-action "Undo last change" in the toolbar as
   well.
5. Does bulk grading need a server-side job for very large courses (>1,000
   students), or is a synchronous request acceptable?
6. Do exports need watermarking or per-recipient tracking for FERPA purposes?
   Coordinate with `../standards/S09-ferpa-hardening.md`.

## 19. References

- Existing files: `clients/web/src/pages/lms/gradebook/gradebook-grid.tsx`
  (2,030 lines), `gradebook-grid-transposed.tsx`, `gradebook-cell-menu.tsx`,
  `gradebook-submission-grading-modal.tsx`,
  `clients/web/src/pages/lms/course-gradebook.tsx`,
  `course-enrollments.tsx` (1,913 lines), `course-standards-gradebook.tsx`,
  `clients/web/src/lib/lms-toast.ts` (`toastWithUndo`),
  `clients/web/src/components/ui/status-vocabulary.tsx`
- Research: [research.md](research.md) R-18, R-20, R-21, R-22, R-23, R-31, R-33
- Audit: [audit.md](audit.md) G-11, G-15, G-13, G-7
- External: [WAI-ARIA APG — Grid pattern](https://www.w3.org/WAI/ARIA/apg/patterns/grid/),
  [NN/g — Confirmation Dialogs](https://www.nngroup.com/articles/confirmation-dialog/)
- Related plans: [UX.2](UX.2-core-component-library-and-adoption-ratchet.md),
  [UX.5](UX.5-wcag-2.2-aa-conformance-uplift.md),
  [UX.13](UX.13-feedback-undo-and-destructive-actions.md),
  [UX.14](UX.14-responsive-and-small-viewport-experience.md),
  [`../tech_debt/TD.14-decompose-god-components.md`](../tech_debt/TD.14-decompose-god-components.md),
  `../standards/S09-ferpa-hardening.md`
