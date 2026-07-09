# W07 — Hide a Course from the Courses Page Views

> Implementation plan. Source: web catalog UX scan (2026-07-09) —
> `clients/web/src/pages/lms/courses.tsx` and the per-user catalog prefs layer.
> Follows [../_TEMPLATE.md](../_TEMPLATE.md).

## Implementation notes (2026-07)

- **Schema:** `367_user_course_catalog_hidden.sql` adds per-user hidden state; backfills legacy kanban `hidden` placements and removes them from `user_course_kanban_placement`.
- **API:** `PUT /api/v1/courses/catalog-hidden` (`SetUserCourseHidden`); enrolled list includes `catalogHidden` via `AttachUserCatalogMeta`. Hiding auto-unpins.
- **Kanban:** Hidden column is driven by `catalogHidden` plus schedule-derived `isCourseCatalogHidden`; kanban saves persist user-hidden rows only.
- **Web UI:** `⋯` actions menu (pin, rename, hide/unhide), toolbar **Show hidden (N)** toggle, all-hidden empty state, muted revealed rows with badge.
- **Tests:** Vitest (`course-catalog-hidden.test.ts`, `courses.test.tsx`); Go (`catalog_user_prefs_hidden_test.go`).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | W07 |
| **Section** | Web / Course Catalog UX |
| **Severity** | MINOR — quality-of-life declutter; RFP-parity with Canvas "favorites" / Classroom archive-from-list |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Frontend platform team |
| **Depends on** | Existing catalog prefs layer (`252_user_course_catalog_prefs`, `266_user_course_catalog_pins`) |
| **Unblocks** | A future "manage hidden courses" surface; mobile parity (separate story) |
| **Permission** | none — purely a per-user personal view preference; no course-level or roster impact |

---

## 1. Problem Statement

A learner or instructor with many enrollments (finished electives, TA shells, sandbox/test
courses, cross-listed sections) cannot declutter their own **Courses** page. The page renders every
enrolled, non-archived course across five views — **cards, list, gallery, table, status** — and only
the *status* (kanban) view offers a personal "Hidden" column. There is no way to say "keep this
course, but stop showing it to me here." The result is a noisy catalog where the courses a person
actually uses are buried. This is distinct from **archiving** (an org-wide, permission-gated,
lifecycle action on the course itself, `courses.archived`) and from **course visibility**
(`hiddenAt` / `visibleFrom`, an instructor setting that hides a course from *learners*). We need a
lightweight, per-user, reversible "hide from my catalog" that behaves identically in every view.

## 2. Goals

- Let any user **hide** an enrolled course from **all** courses-page views (cards, list, gallery,
  table) with one action, and **unhide** it just as easily.
- Make hidden state **consistent across views** — hiding in one view hides everywhere, and the
  existing kanban "Hidden" column becomes a projection of the same state (no more two parallel
  notions of "hidden").
- Keep hidden courses **one click away**, never lost: a "Show hidden (N)" toggle reveals them with a
  clear badge and an Unhide action.
- **Non-destructive & personal:** hiding changes nothing for anyone else, touches no enrollment,
  grade, or course record, and never blocks navigation to the course by direct link.
- Ship it feeling **native to the existing catalog** — same interaction language as Pin and Rename,
  optimistic + persisted per user.

## 3. Non-Goals

- **Not** archiving a course, deleting it, or dropping the enrollment (those exist elsewhere:
  settings Archive view, `archived-courses-api`).
- **Not** changing course-level visibility (`hiddenAt` / `visibleFrom`) — that stays instructor-only
  and learner-wide.
- **Not** hiding courses from the **sidebar pins**, **dashboard**, **calendar course filter**, or
  search in v1 (see §12 for the pin interaction; other surfaces are follow-ups).
- **No** bulk "hide all ended courses" automation in v1 (candidate follow-up, §18).
- **Not** a mobile change — iOS/Android parity is a separate story.

## 4. Personas & User Stories

- **As a student**, I want to hide last term's finished courses so my Courses page shows only what
  I'm taking now, without losing access to the old ones.
- **As an instructor**, I want to hide sandbox/test and co-taught shell courses so my catalog is just
  the sections I actively teach.
- **As a TA**, I want to hide the courses I only observe so they stop cluttering my cards view.
- **As a self-learner**, I want to tuck away completed courses but still reopen them for reference.
- **As any user**, I want to find a course I hid by mistake and restore it in one click.

## 5. Functional Requirements

- **FR-1.** The system MUST provide a per-course **Hide** action reachable from every non-kanban view
  (cards, list, gallery, table) and MUST provide the inverse **Unhide** action wherever a hidden
  course is shown.
- **FR-2.** Hiding a course MUST remove it from the default rendering of cards, list, gallery, table,
  and from the term/section groupings (`catalogSections`), for **that user only**.
- **FR-3.** Hidden state MUST be **per-user, server-persisted**, and survive reload and device change.
- **FR-4.** The toolbar MUST expose a **"Show hidden (N)"** control that (a) shows the current hidden
  count and (b) toggles hidden courses back into the active view, visually marked as hidden with an
  Unhide affordance. When N = 0 the control MAY be omitted.
- **FR-5.** In the **status (kanban)** view, the existing "Hidden" column MUST be driven by the same
  per-user hidden state: dragging a card **into** Hidden MUST hide it everywhere; dragging it **out**
  MUST unhide it and restore its prior/derived column.
- **FR-6.** Hide/Unhide MUST be **optimistic** in the UI and reconcile with the server; on failure it
  MUST roll back and surface a non-blocking error (same pattern as Pin/reorder).
- **FR-7.** Hiding MUST NOT affect the course for other users, MUST NOT change enrollment/roster/grades,
  and MUST NOT prevent opening the course by direct URL.
- **FR-8.** When every course in a term section is hidden (and "Show hidden" is off), that section
  header MUST NOT render.
- **FR-9.** The empty state MUST distinguish "no courses" from "all your courses are hidden" and, in
  the latter case, offer a one-click "Show hidden" path.
- **FR-10.** A course that becomes **archived** or whose enrollment ends MUST NOT linger as a dangling
  hidden row (state is scoped by the enrolled-course query; orphaned prefs are ignored/cleaned).
- **FR-11.** Hide/Unhide SHOULD be reflected across open tabs on the next catalog fetch; real-time
  push is not required.

## 6. Non-Functional Requirements

- **Performance** — hidden filtering is client-side over the already-loaded catalog list; no extra
  round trip to render. Hide/Unhide write is a single small `PUT`; p95 < 300 ms. No added N+1 in the
  enrolled-list query (hidden joins one indexed per-user table, mirroring pins).
- **Security** — action authorizes on the **session user only**; server MUST verify the user is
  enrolled in the target course before recording hidden state (reuse the enrollment check used by
  nicknames). No permission scope required; no cross-user exposure.
- **Privacy & Compliance** — hidden state is personal UI metadata (FERPA-neutral, not an academic
  record); deleted with the user (`ON DELETE CASCADE`).
- **Accessibility** — WCAG 2.1 AA: Hide/Unhide controls have descriptive `aria-label`
  ("Hide {course} from your catalog" / "Show {course}"), are keyboard-reachable, ≥44px targets, and
  the "Show hidden" toggle exposes state via `aria-pressed`. Hidden rows announce their hidden status
  (badge text, not color alone).
- **Scalability** — bounded by a user's enrollment count (tens–low hundreds); the hidden table is
  keyed `(user_id, course_id)` and indexed for the "list my hidden" read.
- **Reliability** — idempotent upsert/delete; hiding an already-hidden course is a no-op.
- **Observability** — emit a `catalog.course_hidden` / `catalog.course_unhidden` analytics event
  (reuse the pin event pattern) to measure adoption; log write failures.
- **Maintainability** — hidden lives in the existing `pages/lms/course-catalog-*` module family and
  `lib/course-catalog-settings-api.ts`; no new top-level concept.
- **Internationalization** — all copy via i18n keys; RTL-safe (mirrors existing catalog controls).
- **Backward compatibility** — additive column/table + additive `CoursePublic.catalogHidden` field
  (defaults false when omitted by older servers, like `catalogPinned`).

## 7. Acceptance Criteria

- **AC-1.** *Given* a cards/list/gallery/table view, *When* I choose **Hide** on a course, *Then* it
  disappears from that view immediately and from the other three, and stays hidden after reload.
- **AC-2.** *Given* hidden courses exist, *When* I toggle **Show hidden (N)** on, *Then* they reappear
  marked "Hidden" with an **Unhide** action; toggling off removes them again.
- **AC-3.** *Given* the status view, *When* I drag a card into **Hidden**, *Then* switching to cards
  shows it gone; *When* I drag it back out, *Then* it reappears in cards and lands in its
  prior/derived kanban column.
- **AC-4.** *Given* a hide write fails, *Then* the course returns to visible and a non-blocking error
  shows; no partial state persists.
- **AC-5.** *Given* a term section whose only course I hide, *When* "Show hidden" is off, *Then* the
  section header is gone; *When* on, both header and course return.
- **AC-6.** *Given* I hide every course, *Then* the empty state says my courses are hidden and offers
  "Show hidden," not "You have no courses."
- **AC-7.** *Given* a course I hid, *When* I open it by direct link or from the sidebar, *Then* it
  opens normally (hiding is catalog-only).
- **AC-8.** *Given* user A hides a course, *Then* user B's catalog is unchanged.

## 8. Data Model

Recommended: hidden is a **first-class per-user state, orthogonal to kanban column**, so unhiding
restores a course's prior manual/derived column instead of stranding it.

- New table (`server/migrations/367_user_course_catalog_hidden.sql`):

  ```sql
  CREATE TABLE course.user_course_catalog_hidden (
      user_id   UUID NOT NULL REFERENCES "user".users (id)   ON DELETE CASCADE,
      course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
      hidden_at TIMESTAMPTZ NOT NULL DEFAULT now(),
      PRIMARY KEY (user_id, course_id)
  );
  CREATE INDEX idx_user_course_catalog_hidden_user
      ON course.user_course_catalog_hidden (user_id, hidden_at DESC);
  ```

- **`CoursePublic`**: add `catalogHidden bool` (Go `CatalogHidden bool json:"catalogHidden,omitempty"`;
  TS `catalogHidden?: boolean`). Populate via `LEFT JOIN course.user_course_catalog_hidden h ON
  h.user_id = $1 AND h.course_id = c.id` in `coursePublicSelect`/`coursePublicFrom`
  (`repos/course/list_enrolled.go`) — same shape as `catalogPinned`.
- **Kanban unification**: the status view's "Hidden" column is derived from `catalogHidden` (plus the
  legacy schedule-derived hidden in `isCourseCatalogHidden`) rather than from
  `user_course_kanban_placement.column_id = 'hidden'`. Placement rows keep only `todo/in-progress/done`;
  a data migration drops/ignores existing `hidden` placements after copying them into the new table
  (backfill below).
- **Backfill**: `INSERT INTO course.user_course_catalog_hidden (user_id, course_id) SELECT user_id,
  course_id FROM course.user_course_kanban_placement WHERE column_id = 'hidden' ON CONFLICT DO
  NOTHING;` so users who already parked courses in the kanban Hidden column keep them hidden.
- **Alternative (documented, not chosen):** keep `hidden` as a kanban placement column and treat it as
  the source of truth across all views — no new table, but unhide can't restore the prior column and
  the two features stay semantically entangled. See §18.

## 9. API Surface

- **`PUT /api/v1/courses/catalog-hidden`** — body `{ courseId: string, hidden: boolean }`; auth = session
  user; verifies enrollment; upserts or deletes the row; returns `204`/`{ok:true}`. Mirrors
  `PUT /api/v1/courses/catalog-pin` in `courses_routes.go`.
- Enrolled-list endpoints (`GET /api/v1/courses` and org-scoped variants) now include `catalogHidden`
  per course — no new list endpoint needed; the client filters locally.
- Optional read for a future dedicated surface: reuse the list; no separate "list hidden" endpoint in
  v1.
- Rate-limit: same bucket as other catalog pref writes. OpenAPI: document the new route and the added
  `catalogHidden` field.
- Repo functions in `repos/course/catalog_user_prefs.go`: `SetUserCourseHidden(ctx, pool, userID,
  courseID, hidden bool)` with the enrollment guard used by `UpsertUserCatalogNickname`.

## 10. UI / UX

Keep the interaction language identical to **Pin** and **Rename** so it feels native.

**Per-course Hide affordance.** Introduce a small **course action menu** (kebab `⋯`) on each card /
gallery tile / list row / table row that consolidates the per-course actions — **Pin/Unpin**, **Rename**,
**Hide from catalog** (`EyeOff` icon) — and, for hidden courses, **Unhide** (`Eye`). This avoids a
third always-visible icon button while giving Hide a discoverable home. (The existing inline Pin button
may remain on hover for cards; Hide lives in the menu.) New components:
`course-catalog-actions-menu.tsx` and `course-catalog-hide-button.tsx` (parallel to
`course-catalog-pin-button.tsx`), wired through a `useCatalogHidden` hook / context mirroring
`course-pinned-context`.

**Toolbar "Show hidden (N)".** Add a toggle to the courses toolbar (next to the View menu and term/
grade filters). Off by default. Shows the count; hidden = `aria-pressed`. When on, hidden courses
render in every view at reduced emphasis (muted, `Hidden` badge) with the Unhide action inline.

**Status (kanban) view.** No visual change — the existing "Hidden" column now reflects `catalogHidden`.
Drag in ⇒ hide; drag out ⇒ unhide and restore prior column. The column's collapse state keeps using
`hidden_column_expanded`.

**Key flows.**
1. Hover/focus a course → open `⋯` → **Hide from catalog** → course animates out; toast "Hidden. Undo"
   (undo restores instantly). Toolbar count increments.
2. Toolbar → **Show hidden (3)** → hidden courses fade in with badge → **Unhide** on one → it rejoins
   the active list and the count drops.
3. Status view → drag card to **Hidden** → same underlying state.

**States.**
- *Empty (no courses):* existing `EmptyState` ("No courses yet").
- *Empty (all hidden):* new copy — "All your courses are hidden" + **Show hidden** button.
- *Loading:* existing `CoursesCatalogSkeleton`.
- *Error:* existing inline error banner; hide/unhide roll back optimistically.

**Responsive / mobile-web.** Kebab menu is touch-friendly (≥44px); toolbar toggle wraps with the
existing filter row.

**Accessibility annotations.** Menu is `role="menu"` with `menuitem`s (reuse the view-menu pattern);
Hide/Unhide carry course-specific labels; hidden badge is text + icon, not color alone; focus returns
to the triggering row after hide (or to the toolbar toggle if the row leaves the DOM).

**Copy & i18n keys** (new): `courses.actions.hide`, `courses.actions.unhide`, `courses.hidden.badge`,
`courses.hidden.toggle` (`Show hidden ({count})`), `courses.hidden.emptyTitle`,
`courses.hidden.emptyAction`, `courses.hidden.undo`, and `aria` variants.

## 11. AI / ML Considerations

- None. (A future "suggest courses to hide" based on inactivity is out of scope; §18.)

## 12. Integration Points

- **Web page:** `clients/web/src/pages/lms/courses.tsx` — filter `courses` by `catalogHidden` before
  `catalogSections`/`renderCourseItems`; add the "Show hidden" toggle state; extend `renderSortableCourse`
  rows with the actions menu.
- **Catalog components:** `course-catalog-view-menu.tsx` (unchanged), `course-catalog-kanban.tsx` +
  `course-catalog-status.ts` (`buildKanbanBoardState`/`courseKanbanColumn` read `catalogHidden`),
  new `course-catalog-actions-menu.tsx`, `course-catalog-hide-button.tsx`.
- **API client:** `lib/course-catalog-settings-api.ts` — add `putCourseCatalogHidden(courseId, hidden)`;
  `lib/courses-api.ts` — add `catalogHidden` to `CoursePublic`.
- **Context:** new `context/course-hidden-context.tsx` (or extend an existing catalog context) mirroring
  `course-pinned-context` for optimistic toggling.
- **Server:** `repos/course/catalog_user_prefs.go` (`SetUserCourseHidden`), `repos/course/list_enrolled.go`
  (join + scan), `httpserver/courses_routes.go` (new route).
- **Pin interaction (decision):** hiding a course that is **pinned to the sidebar** SHOULD also unpin it
  (a hidden-but-pinned course is contradictory and re-clutters). Recommended: hide → unpin; unhide does
  **not** auto-repin. Flag as confirm-with-design (§18).
- **Migrations:** `367_user_course_catalog_hidden.sql` (+ `.down.sql`), with the kanban-hidden backfill.

## 13. Dependencies & Sequencing

- Ships on top of the existing catalog prefs stack (pins, nicknames, kanban). No feature depends on it
  to ship first.
- Shared infra: Postgres (`course` schema), existing analytics event pipeline. No object storage, queue,
  or email.
- Sequence: schema + backfill → server route + list field → API client + context → UI (menu, toggle,
  kanban wiring) → i18n → tests.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Users "lose" courses and think they're deleted | M | M | Always-visible "Show hidden (N)" count; hide toast has Undo; hidden ≠ archived copy |
| Kanban "Hidden" column semantics change under existing users | M | M | Backfill kanban-hidden rows into new table; keep the column visually identical |
| Hidden filtering diverges between views | L | M | Single `catalogHidden` predicate applied once, shared by all renderers + kanban builder |
| Pin/hide contradiction (pinned course hidden) | M | L | Hide auto-unpins (recommended); covered by test |
| Orphaned hidden rows after unenroll/archive | L | L | State scoped by enrolled-course query; `ON DELETE CASCADE`; periodic cleanup not required |
| Extra join slows enrolled-list query | L | L | Indexed `(user_id, course_id)` join, identical cost profile to pins |

## 15. Rollout Plan

- **Feature flag:** `web_catalog_hide_courses` (client-side gate on the menu item + toggle), default
  **on** in dev/staging, **off** in prod until dogfood passes. Server route + column ship unflagged
  (inert without UI).
- **Sequencing:** migrate schema → backfill kanban-hidden → deploy server → deploy client behind flag →
  dogfood (internal org) → flip flag to GA.
- **GA criteria:** AC-1…AC-8 green; no rise in "where did my course go?" support signals during dogfood.
- **Rollback:** flip the flag off (UI disappears; stored hidden state is inert and preserved). Full
  revert = `.down.sql` drops the table; kanban placements remain intact.

## 16. Test Plan

- **Unit** — hidden filter predicate; `catalogSections` drops all-hidden sections; kanban builder maps
  `catalogHidden` → Hidden column; empty-state selection (none vs all-hidden); optimistic rollback.
- **Integration (Go)** — `SetUserCourseHidden` upsert/delete + enrollment guard; enrolled-list returns
  `catalogHidden`; hide auto-unpin; backfill migration copies kanban-hidden rows.
- **E2E (Playwright)** — hide in cards → gone in list/gallery/table; Show hidden reveals + Unhide;
  kanban drag in/out matches other views; hide → reload persists; direct-link still opens a hidden
  course; user-A hide invisible to user-B.
- **Security** — hidden write requires enrollment; cannot hide on behalf of another user; no permission
  escalation surface.
- **Accessibility** — axe on the courses page with the menu open and hidden shown; keyboard-only
  hide/unhide/toggle; screen-reader announces hidden badge and control state.
- **Performance** — enrolled-list query timing with the added join at p95 target; render with 100+
  courses, half hidden.
- **Manual exploratory** — hide a pinned course; hide the last visible course; toggle churn; multi-tab
  reconcile.

## 17. Documentation & Training

- Help center: "Hide a course from your Courses page (and get it back)" — clarifies hide vs. archive vs.
  drop.
- In-product: tooltip on the toolbar toggle; empty-state copy is self-explanatory.
- API reference: new `PUT /courses/catalog-hidden` and the `catalogHidden` field.
- Internal runbook: note the kanban-hidden → hidden-table migration in the release notes.

## 18. Open Questions

1. **Hidden state model** — dedicated orthogonal `hidden_at` table (recommended, §8) vs. reuse the
   kanban `hidden` placement column? Recommendation: dedicated, for clean unhide/restore.
2. **Pin interaction** — auto-unpin on hide (recommended) vs. allow pinned-and-hidden vs. block hiding a
   pinned course? Confirm with design.
3. **Hide affordance shape** — consolidated `⋯` actions menu (recommended) vs. a dedicated always-visible
   Hide icon next to Pin?
4. **Scope creep** — should hidden courses also drop out of the **sidebar pins list is separate**, the
   **calendar course filter**, and the **dashboard**, or stay courses-page-only for v1 (recommended)?
5. **Bulk hide** — offer "Hide all ended courses" later? (Follow-up, not v1.)
6. **Default reveal per view** — should the status/kanban Hidden column stay always-present while the
   other views default-hide, or gain a matching "Show hidden" affordance? (Recommended: kanban keeps its
   column; toggle governs the other four.)

## 19. References

- Web: `clients/web/src/pages/lms/courses.tsx`, `course-catalog-status.ts`, `course-catalog-kanban.tsx`,
  `course-catalog-pin-button.tsx`, `course-catalog-view-menu.tsx`,
  `clients/web/src/lib/course-catalog-settings-api.ts`, `lib/course-catalog-types.ts`,
  `lib/courses-api.ts`, `context/course-pinned-context.ts`.
- Server: `server/internal/repos/course/catalog_user_prefs.go`, `repos/course/list_enrolled.go`,
  `httpserver/courses_routes.go`; migrations `252_user_course_catalog_prefs.sql`,
  `266_user_course_catalog_pins.sql`.
- Related plans: [W05 — Human-readable entity labels](../../completed/web/W05-human-readable-entity-labels.md);
  [M14.10 — Global archived courses (mobile)](../mobile/M14.10-global-archived-courses.md) (archive is a
  distinct, org-wide action).
</content>
</invoke>
