# VC.1 — Collaboration Boards: Foundation, Data Model & Feature Flag

> Implementation plan. Source: net-new capability (real-time visual collaboration boards). Landscape: [visual-collaboration/README](../../plan/visual-collaboration/README.md). Builds on the Whiteboard app (`course.whiteboards`, migration `230`, flag `whiteboard_enabled`) and Collaborative Documents (plan 6.5).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.1 |
| **Section** | Visual Collaboration Boards |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Collaboration squad |
| **Depends on** | — |
| **Unblocks** | VC.2, VC.3, VC.4, VC.5, VC.6, VC.7, VC.8, VC.9, VC.10 |

---

## 1. Problem Statement

Instructors who want a shared "wall" where every learner drops a card (an idea, an image, a link, a
question) currently pay for an external tool per seat, and that content lives outside Lextures — outside
our roster, permissions, gradebook, and privacy controls. Lextures has a single-author Whiteboard and a
Y.js collaborative-document surface, but no **multi-contributor visual board**. VC.1 lays the foundation:
the `board` schema, the per-course feature flag, board CRUD, the navigation entry, and an (initially empty)
board page — the base every other VC story builds on.

## 2. Goals

- Introduce a per-course `visual_boards_enabled` flag that follows the exact Whiteboard flag pattern (DB
  column → API → nav context → settings toggle → nav link).
- Persist boards in a self-contained `board` Postgres schema so the whole feature can live behind the flag.
- Ship board CRUD (create, list, read, rename, archive/delete) with course-scoped authorization.
- Render a boards **list page** and an **empty board page** shell that later stories fill in.
- Establish the API/URL/permission conventions the rest of the VC stories reuse.

## 3. Non-Goals

- Posts / cards and their content types (VC.2).
- Layouts beyond a default "wall" placeholder (VC.3).
- Real-time sync and presence (VC.4).
- Reactions, comments, sharing links, moderation, templates, export (VC.5–VC.9).
- Platform-level analytics and quotas (VC.10) — this plan adds only the platform master flag.

## 4. Personas & User Stories

- **As an instructor**, I want to create a named board inside my course so that I have a shared space to
  collect contributions.
- **As an instructor**, I want to enable/disable Boards per course from course settings so the menu only
  appears when I use it.
- **As a student**, I want the "Boards" menu item to appear only when my instructor has turned it on, and
  to open a board I have access to.
- **As an admin**, I want a platform master switch so the whole capability can be dark-launched or disabled
  organisation-wide.
- **As a self-learner** (solo course owner), I want to create a personal board for my own study course.

## 5. Functional Requirements

- **FR-1.** The system MUST add a boolean column `course.courses.visual_boards_enabled` defaulting to
  `FALSE`, exposed on the course payload as `visualBoardsEnabled` and patchable via the existing course
  features endpoint (`PatchFeatures`).
- **FR-2.** The system MUST add a platform master flag `VisualBoardsEnabled` (default `FALSE`) in
  `platformconfig`; when the master flag is off, all board routes MUST behave as if the feature does not
  exist (404) regardless of the per-course flag.
- **FR-3.** The system MUST provide `POST /api/v1/courses/{course_code}/boards` to create a board with a
  `title` (required, ≤200 chars) and optional `description`, returning the created row.
- **FR-4.** The system MUST provide `GET /api/v1/courses/{course_code}/boards` returning boards for the
  course ordered by `updated_at DESC`, excluding archived boards unless `?includeArchived=true`.
- **FR-5.** The system MUST provide `GET /api/v1/courses/{course_code}/boards/{board_id}` returning a
  single board or 404.
- **FR-6.** The system MUST provide `PATCH /api/v1/courses/{course_code}/boards/{board_id}` to update
  `title`, `description`, and `archived` state.
- **FR-7.** The system MUST provide `DELETE /api/v1/courses/{course_code}/boards/{board_id}` to soft-delete
  (archive) by default, and hard-delete when the caller has course-manage permission and passes
  `?hard=true`.
- **FR-8.** Creating, renaming, archiving, and deleting a board MUST require the
  `course:{code}:item:create` permission (the permission the Whiteboard handlers already use); listing and
  reading MUST require course access (`requireCourseAccess`).
- **FR-9.** All board routes MUST return `404 not found` when `visual_boards_enabled` is false for the
  course, matching the `collabDocsFeatureOff` guard pattern.
- **FR-10.** The web app MUST show a "Boards" nav link under a course only when both the master flag and
  the per-course flag are on, gated the same way the Whiteboard link is gated in `side-nav-course-links`.
- **FR-11.** Every board MUST record `created_by`, `created_at`, `updated_at`, and a `slug`/short id usable
  in URLs; deleting the course MUST cascade-delete its boards.

## 6. Non-Functional Requirements

- **Performance** — board list p95 < 150 ms for ≤500 boards; single-board fetch p95 < 100 ms.
- **Security** — course-scoped authz on every route; board ids are UUIDs; no cross-course leakage (every
  query joins on `course_code`, as `ListWhiteboards` does).
- **Privacy & Compliance** — a board and its future posts are education records; the schema MUST carry the
  columns (owner, timestamps) needed by the deletion/export engines ([S01](../../plan/standards/S01-unified-data-subject-rights-orchestration.md)/[S02](../../plan/standards/S02-data-retention-deletion-engine.md)).
- **Accessibility** — list and empty-state pages meet WCAG 2.1 AA; nav link is keyboard-reachable.
- **Scalability** — schema designed so post volume (VC.2) scales independently of board count.
- **Reliability** — idempotent create is not required, but create MUST be transactional; archive is a
  reversible soft-delete.
- **Observability** — emit `board.created`, `board.archived`, `board.deleted` counters and a request span
  per handler via `server/internal/telemetry`.
- **Maintainability** — repo lives in `server/internal/repos/board/`, mirroring `repos/collabdocs`.
- **Internationalization** — all web copy via i18n catalog; timestamps rendered in viewer locale/tz.
- **Backward compatibility** — additive migration; flag defaults off, so no existing course changes.

## 7. Acceptance Criteria

- **AC-1.** *Given* the master flag on and a course with `visual_boards_enabled = true`, *when* an
  instructor POSTs a board with a title, *then* a `201` returns the board and it appears in the list.
- **AC-2.** *Given* `visual_boards_enabled = false`, *when* any `/boards` route is called, *then* the API
  returns `404` with an `apierr` body.
- **AC-3.** *Given* the master flag off, *when* any `/boards` route is called, *then* it returns `404` even
  if the per-course flag is true.
- **AC-4.** *Given* a student without `item:create`, *when* they POST a board, *then* the API returns `403`.
- **AC-5.** *Given* an instructor archives a board, *when* the list is fetched without `includeArchived`,
  *then* the board is absent; *when* fetched with `includeArchived=true`, *then* it is present.
- **AC-6.** *Given* the flag is on, *when* a course member opens the course, *then* a "Boards" nav link is
  visible and routes to the boards list.
- **AC-7.** *Given* a course is deleted, *when* the cascade runs, *then* its boards are removed.

## 8. Data Model

New schema `board`. Migration `378_board_foundation.sql`:

```sql
-- 378_board_foundation.sql
ALTER TABLE course.courses
  ADD COLUMN visual_boards_enabled BOOLEAN NOT NULL DEFAULT FALSE;

CREATE SCHEMA IF NOT EXISTS board;

CREATE TABLE board.boards (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id     UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    slug          TEXT NOT NULL,               -- short, URL-safe; unique per course
    archived      BOOLEAN NOT NULL DEFAULT FALSE,
    created_by    UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (course_id, slug)
);

CREATE INDEX idx_boards_course ON board.boards (course_id) WHERE archived = FALSE;
```

- Platform master flag column added to the `platform` settings table alongside the other DB-managed feature
  flags (see `platformconfig.applyPlatformBools`); no env seed, defaults `FALSE`.
- **Migration file naming**: `server/migrations/378_board_foundation.sql` (+ `.down.sql` following the repo
  convention — a rollback stub pointing at the runbook, as `230_course_whiteboard.down.sql` does).
- **Backfill**: none; the new column defaults `FALSE`, so every existing course starts with Boards off.

## 9. API Surface

All under `/api/v1/courses/{course_code}/boards`, registered in `courses_routes.go` next to the whiteboard
routes.

| Verb | Path | Auth scope |
|---|---|---|
| GET | `/boards` | course access |
| POST | `/boards` | `course:{code}:item:create` |
| GET | `/boards/{board_id}` | course access |
| PATCH | `/boards/{board_id}` | `course:{code}:item:create` |
| DELETE | `/boards/{board_id}` | `course:{code}:item:create` (hard-delete needs course-manage) |

```ts
// Board (response)
type Board = {
  id: string
  courseId: string
  title: string
  description: string
  slug: string
  archived: boolean
  createdBy: string | null
  createdAt: string   // RFC3339
  updatedAt: string
}
// POST body:  { title: string; description?: string }
// PATCH body: { title?: string; description?: string; archived?: boolean }
```

- **Rate limits**: reuse the standard authenticated-write limiter; board create is low-volume.
- **OpenAPI**: register the new paths/schemas in `server/internal/openapi/openapi.go`.

## 10. UI / UX

- **New page — Boards list** (`clients/web/src/pages/lms/course-boards-page.tsx`): grid of board cards
  (title, description, updated-at, contributor count placeholder), a "New board" button for users with
  create permission, and an empty state ("No boards yet — create one to get started").
- **New page — Board detail shell** (`course-board-detail-page.tsx`): header (title, rename, archive/menu)
  plus a placeholder canvas area that VC.2/VC.3 fill. Loading / error / empty states included.
- **Nav**: add a "Boards" `SideNavLink` (grid icon) in `side-nav-course-links.tsx`, gated on
  `visualBoardsEnabled` (and role where appropriate), mirroring the Whiteboard link.
- **Settings**: add a "Collaboration boards" toggle row in `course-features-section.tsx` with copy such as
  *"A shared wall where students post cards — text, images, links, and more — and react in real time."*
- **Routing**: register lazy routes in `lazy-pages.ts` / `app.tsx` (`/courses/:code/boards` and
  `/courses/:code/boards/:boardId`).
- **Mobile**: list is a single-column card stack; board shell is full-width.
- **Accessibility**: board cards are links with visible focus; the create button has an accessible label.
- **Copy & i18n**: all strings via the i18n catalog (`boards.*` keys).

## 11. AI / ML Considerations

Not AI-touching. (AI-assisted board summaries/idea-clustering are a future enhancement noted in VC.10 Open
Questions, out of scope here.)

## 12. Integration Points

- **Internal**: `server/internal/repos/course/features.go` (`PatchFeatures` gains the new flag arg),
  `server/internal/httpserver/course_features.go`, `courses_routes.go`, `platformconfig/features.go`,
  `clients/web/src/context/course-nav-features-context.tsx`, `course-features-section.tsx`,
  `side-nav-course-links.tsx`, `lib/courses-api-schemas.ts`.
- **New**: `server/internal/repos/board/board.go`, `server/internal/httpserver/board_http.go`,
  `clients/web/src/lib/boards-api.ts`.
- **Events**: emit board lifecycle telemetry; no external webhooks in VC.1.

## 13. Dependencies & Sequencing

- Must ship after: — (this is the root story).
- Must ship before: every other VC story.
- Shared infra needed: Postgres (new schema), existing telemetry.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Flag-plumbing drift (adding one boolean touches ~8 files) | M | L | Follow the Whiteboard flag PR as a checklist; add a features-round-trip test |
| `PatchFeatures` signature grows unwieldy (already 22 bools) | M | M | Note tech-debt; consider a features struct refactor (tracked, not blocking) |
| Slug collisions within a course | L | L | `UNIQUE (course_id, slug)` + retry-with-suffix on insert |

## 15. Rollout Plan

- **Flag**: per-course `visual_boards_enabled` (default off) + platform `VisualBoardsEnabled` (default off).
- **Sequencing**: migration `378` → deploy code (routes behind both flags) → enable master flag in staging
  → dogfood with an internal course → GA the master flag once VC.2–VC.4 land.
- **Dogfood**: internal "Boards" course.
- **GA criteria**: create/list/read/rename/archive all green in E2E; nav gating verified.
- **Rollback**: turn off the master flag; the migration is additive and inert when the flag is off.

## 16. Test Plan

- **Unit** — repo CRUD (create/list/get/patch/archive/hard-delete); slug uniqueness/retry.
- **Integration** — feature-flag guard returns 404 when off (per-course and master); authz matrix
  (student vs instructor); cascade-on-course-delete.
- **End-to-end** — Playwright: toggle the flag in settings → "Boards" appears → create a board → it lists →
  rename → archive → disappears from default list.
- **Security** — cross-course access attempt returns 404/403; hard-delete requires manage permission.
- **Accessibility** — axe on list + shell pages; keyboard nav to the "Boards" link and create button.
- **Performance** — list query plan uses `idx_boards_course`.
- **Manual** — flag on/off transitions; deep-link to an archived board.

## 17. Documentation & Training

- End-user: "Create your first board" help-center article.
- Instructor: how to enable Boards for a course.
- Admin: the platform master flag and what it gates.
- API reference: new `/boards` endpoints in the OpenAPI doc.
- Runbook: note the `board` schema and the archive-vs-hard-delete semantics.

## 18. Open Questions

1. Should boards be creatable outside a course (org-level / personal), or course-scoped only for v1?
   (Recommendation: course-scoped only; revisit in VC.10.)
2. Do we want a per-course board **cap** at the free tier? (Deferred to VC.10 quotas.)
3. Should `slug` be user-editable or always derived from the title? (Recommendation: derived, with suffix
   on collision.)

## 19. References

- Existing files this work touches: `server/internal/repos/course/features.go`,
  `server/internal/httpserver/course_features.go`, `server/internal/httpserver/courses_routes.go`,
  `server/internal/httpserver/course_whiteboard.go` (pattern), `server/migrations/230_course_whiteboard.sql`
  (pattern), `server/internal/repos/platformconfig/features.go`,
  `clients/web/src/context/course-nav-features-context.tsx`,
  `clients/web/src/pages/lms/course-features-section.tsx`,
  `clients/web/src/components/layout/side-nav-course-links.tsx`, `clients/web/src/lazy-pages.ts`.
- Related plans: [VC.2](VC.2-posts-and-content-types.md),
  [VC.6](../../plan/visual-collaboration/VC.6-sharing-access-contributors.md),
  [VC.10](../../plan/visual-collaboration/VC.10-admin-analytics-quotas-lifecycle.md).
