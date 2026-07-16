# IQ.1 — Foundation, Data Model & Feature Flag

> Implementation plan. Source: net-new capability (interactive game-based quizzing). Landscape: [interactive-quizzes/README](../../plan/interactive-quizzes/README.md). Mirrors the feature-flag + dedicated-schema pattern proven by Collaboration Boards (VC.1, migration `378`) and the Whiteboard (`whiteboard_enabled`).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IQ.1 |
| **Section** | Interactive Quizzes |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Assessment squad |
| **Depends on** | — |
| **Unblocks** | IQ.2, IQ.3, IQ.8, IQ.11 |

---

## 1. Problem Statement

Lextures has a strong static-assessment stack (question bank, module quizzes, item analysis) but nothing for
**live, gamified, whole-class quizzing** — the "everyone answers on their phone against a countdown while a
leaderboard updates on the projector" experience teachers currently buy from a third-party per-seat vendor.
IQ.1 lays the foundation: a per-course feature flag, a self-contained `quizgame` schema, the top-level "quiz
kit" object with CRUD, and the empty "Live Quizzes" page and nav entry — the scaffold every later IQ story
builds on. Without it, nothing else in the section can land.

## 2. Goals

- Add a per-course flag `interactive_quizzes_enabled` (default off) plus a platform master flag
  `FFInteractiveQuizzes`, wired end-to-end (server merge → API → web context → course-features toggle → nav).
- Stand up a dedicated `quizgame` Postgres schema and the `quizgame.kits` root table (a quiz kit =
  reusable, course-scoped, titled collection of questions), containment-scoped so it can be dropped behind
  the flag.
- Ship kit list/create/rename/duplicate-stub/archive CRUD (REST) with course-role authorization.
- Ship the "Live Quizzes" nav link and page shell: a kit gallery with empty/loading/error states and a
  "Create kit" affordance (question authoring lands in IQ.2).
- Establish the repo/handler/route/i18n conventions the rest of the section follows.

## 3. Non-Goals

- Question authoring and question types (IQ.2).
- Any live hosting, WebSocket, join code, or gameplay (IQ.3/IQ.4).
- Scoring, reports, sharing, moderation, AI generation (IQ.5–IQ.11).
- Reusing/migrating existing module-quiz content into kits (touched by IQ.2's "import from question bank").

## 4. Personas & User Stories

- **As an instructor**, I want to see all my course's quiz kits in one place and create a new one, so I can
  start building a live quiz.
- **As an instructor**, I want to rename, duplicate, and archive kits so my library stays organised.
- **As a course admin**, I want Live Quizzes to be off by default and enabled per course, so we roll it out
  deliberately.
- **As a platform admin**, I want a master switch so the whole capability can be disabled tenant-wide.
- **As a student**, I should see nothing until an instructor enables and shares a game (no student surface in
  IQ.1).

## 5. Functional Requirements

- **FR-1.** The system MUST add `course.courses.interactive_quizzes_enabled BOOLEAN NOT NULL DEFAULT FALSE`
  and expose/accept it through the course-features read and PATCH handlers
  (`server/internal/httpserver/course_features.go`), gated by `course:{code}:item:create` like sibling flags.
- **FR-2.** The system MUST add a platform master flag `FFInteractiveQuizzes`
  (`settings.platform_app_settings.ff_interactive_quizzes`), defaulted `false` via `mergeBool` in
  `platformconfig/features.go`, and surface it as `ffInteractiveQuizzes` in the web platform-features client.
- **FR-3.** A course's Live Quizzes surface MUST be usable only when **both** the platform flag and the
  per-course flag are on; otherwise the nav link is hidden and the API returns `403`/`404` consistently with
  other flag-gated features.
- **FR-4.** The system MUST create schema `quizgame` and table `quizgame.kits` (id, course_id, title,
  description, slug, cover_image_ref, status, visibility, tags, created_by, timestamps, archived).
- **FR-5.** The system MUST provide REST CRUD: list kits (course-scoped, excludes archived by default),
  create, get, rename/patch, duplicate (metadata-only stub in IQ.1 — deep copy defined in IQ.8), archive
  (soft), and restore.
- **FR-6.** Creation MUST derive a URL-safe `slug` unique per course and stamp `created_by` from the viewer.
- **FR-7.** All write routes MUST require `courseroles.UserHasPermission(..., "course:{code}:item:create")`;
  read routes require course access via `requireCourseAccess`.
- **FR-8.** Archiving a kit MUST be reversible and MUST NOT hard-delete; hard delete follows the retention
  engine (IQ.11 / [S02](../../plan/standards/S02-data-retention-deletion-engine.md)).
- **FR-9.** The web app MUST add a lazy-loaded "Live Quizzes" page and a course nav link, both flag-gated via
  `course-nav-features-context`.
- **FR-10.** The kit list SHOULD be paginated/searchable by title and tag (server-side filter params).

## 6. Non-Functional Requirements

- **Performance** — kit list p95 < 200 ms for 500 kits; create < 150 ms.
- **Security** — course-scoped authorization on every route; no cross-course kit access; slugs are opaque and
  not authorization-bearing.
- **Privacy & Compliance** — kits are instructor content (not yet student data); still subject to org
  data-residency and deletion policies.
- **Accessibility** — kit gallery keyboard-navigable; cards are semantic links; WCAG 2.1 AA.
- **Scalability** — single table with `(course_id, archived)` partial index; ready for thousands of kits/course.
- **Reliability** — idempotent create via optional client-supplied idempotency key; soft-archive is reversible.
- **Observability** — counters for kit create/archive; standard request metrics via telemetry middleware.
- **Maintainability** — repo package `quizgame` owns all SQL; handlers thin, following module-quiz handlers.
- **Internationalization** — all page/nav copy in i18n catalog (`liveQuiz.*` keys).
- **Backward compatibility** — additive migration; default-off flag means zero behaviour change on upgrade.

## 7. Acceptance Criteria

- **AC-1.** *Given* the platform and course flags are on, *when* an instructor opens the course, *then* a
  "Live Quizzes" nav link appears and routes to the kit gallery.
- **AC-2.** *Given* the course flag is off, *when* any user loads the course, *then* the nav link is hidden
  and `GET .../live-quizzes/kits` returns `403/404`.
- **AC-3.** *Given* an instructor on the gallery, *when* they click "Create kit" and name it, *then* a kit is
  created, appears in the list, and opens to an (empty) editor.
- **AC-4.** *Given* an existing kit, *when* the instructor archives it, *then* it disappears from the default
  list, remains restorable, and is not hard-deleted.
- **AC-5.** *Given* two kits created with the same title, *when* both are saved, *then* each gets a distinct
  unique slug within the course.
- **AC-6.** *Given* a student (non-instructor) with course access, *when* they call a write route, *then* the
  API returns `403`.
- **AC-7.** *Given* the platform master flag is off, *when* any course tries to use Live Quizzes, *then* it is
  unavailable regardless of the per-course flag.

## 8. Data Model

Migration `385_interactive_quizzes_foundation.sql` (renumbered on merge from reserved `390`):

```sql
ALTER TABLE course.courses
  ADD COLUMN IF NOT EXISTS interactive_quizzes_enabled BOOLEAN NOT NULL DEFAULT FALSE;
COMMENT ON COLUMN course.courses.interactive_quizzes_enabled IS
  'IQ.1: Enables live game-based quizzes for this course. Default off.';

CREATE SCHEMA IF NOT EXISTS quizgame;

CREATE TYPE quizgame.kit_visibility AS ENUM ('private', 'course', 'org', 'public');
CREATE TYPE quizgame.kit_status     AS ENUM ('draft', 'ready', 'archived');

CREATE TABLE quizgame.kits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id       UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    slug            TEXT NOT NULL,
    cover_image_ref TEXT,                      -- storage object key (IQ.2 media)
    status          quizgame.kit_status     NOT NULL DEFAULT 'draft',
    visibility      quizgame.kit_visibility NOT NULL DEFAULT 'course',
    tags            TEXT[] NOT NULL DEFAULT '{}',
    question_count  INTEGER NOT NULL DEFAULT 0, -- denormalised, maintained by IQ.2
    archived        BOOLEAN NOT NULL DEFAULT FALSE,
    created_by      UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (course_id, slug)
);
CREATE INDEX idx_quizgame_kits_course ON quizgame.kits (course_id) WHERE archived = FALSE;
CREATE INDEX idx_quizgame_kits_tags   ON quizgame.kits USING gin (tags);

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_interactive_quizzes BOOLEAN;
COMMENT ON COLUMN settings.platform_app_settings.ff_interactive_quizzes IS
    'IQ.1: Platform master switch for Live Quizzes. Default OFF.';
```

- **Indexes/constraints:** unique `(course_id, slug)`; partial index for active-list queries; GIN on tags.
- **Backfill:** none — new column defaults false; new schema empty.
- **Down migration:** drops table, enums, schema, and columns (mirrors `378_board_foundation.down.sql`).

## 9. API Surface

| Verb | Path | Auth scope |
|---|---|---|
| GET | `/api/v1/courses/{code}/live-quizzes/kits` | course access; `?q=&tag=&page=` |
| POST | `/api/v1/courses/{code}/live-quizzes/kits` | `course:{code}:item:create` |
| GET | `/api/v1/courses/{code}/live-quizzes/kits/{kit_id}` | course access |
| PATCH | `/api/v1/courses/{code}/live-quizzes/kits/{kit_id}` | `course:{code}:item:create` |
| POST | `/api/v1/courses/{code}/live-quizzes/kits/{kit_id}/duplicate` | `course:{code}:item:create` |
| POST | `/api/v1/courses/{code}/live-quizzes/kits/{kit_id}/archive` | `course:{code}:item:create` |
| POST | `/api/v1/courses/{code}/live-quizzes/kits/{kit_id}/restore` | `course:{code}:item:create` |

- **Response shape** (`Kit`): `{ id, title, description, slug, coverImageRef, status, visibility, tags,
  questionCount, archived, createdBy, createdAt, updatedAt }`.
- **Errors** use `apierr.WriteJSON` with the standard codes.
- **OpenAPI:** register the new paths/schemas in `server/internal/openapi/openapi.go`.
- **Rate-limit:** create/duplicate share the standard authenticated-write limiter.

## 10. UI / UX

- **New page:** `clients/web/src/pages/lms/live-quiz-kits-page.tsx` — kit gallery (cards: cover, title,
  question count, status chip, last-edited), "Create kit" primary button, search + tag filter.
- **New nav link:** course side-nav "Live Quizzes" entry (`side-nav-course-links.tsx`), flag-gated.
- **New API client:** `clients/web/src/lib/live-quiz-api.ts` with typed `Kit` and CRUD calls; schemas in
  `live-quiz-api-schemas.ts`.
- **Flows:** (1) open gallery → (2) create kit (name modal) → (3) land in editor (empty in IQ.1) →
  (4) back to gallery, archive/restore from card menu.
- **States:** empty ("No quiz kits yet — create your first"), loading (skeleton cards), error (retry),
  archived filter toggle.
- **Responsive:** gallery reflows 1–4 columns; touch targets ≥ 44px.
- **Accessibility:** cards are links with accessible names; menu is keyboard-operable; focus returns to the
  card after closing a menu.
- **Copy & i18n:** `liveQuiz.gallery.*`, `liveQuiz.kit.*` keys across `en`, `es`, `fr`.

## 11. AI / ML Considerations

Not AI-touching. (AI kit generation is IQ.10 and hooks into this CRUD.)

## 12. Integration Points

- **Server new:** `server/internal/repos/quizgame/kits.go`,
  `server/internal/httpserver/quizgame_kits.go`, route registration in `courses_routes.go`.
- **Server modified:** `course_features.go` (flag field), `platformconfig/features.go` +
  `platformconfig.go` (master flag), `openapi.go`.
- **Web modified:** `course-nav-features-context.tsx`, `platform-features.ts`,
  `course-features-section.tsx`, `side-nav-course-links.tsx`, `lazy-pages.ts`, `app.tsx` (route).
- **Reuse:** `requireCourseAccess`, `courseroles.UserHasPermission`, `apierr`, slug helper, storage-object
  refs (for later media).

## 13. Dependencies & Sequencing

- Must ship after: nothing.
- Must ship before: IQ.2 (adds questions to a kit), IQ.3 (hosts a kit), IQ.8/IQ.11.
- Shared infra: Postgres, existing auth/roles, telemetry middleware.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Flag wiring drift across the ~6 touchpoints | M | M | Follow the VC.1 checklist exactly; add an integration test asserting nav visibility ↔ flags |
| Migration number collides with in-flight board work | M | L | Reserve `390_*`; renumber on merge; CI checks sequence gaps |
| Naming confusion with module "Quizzes" | M | M | Distinct "Live Quizzes" label + `quizgame` schema + `live-quizzes` route prefix |
| Scope creep into authoring | M | M | Hard non-goal; editor is an empty shell in IQ.1 |

## 15. Rollout Plan

- **Flag:** `interactive_quizzes_enabled` (course) gated by `ff_interactive_quizzes` (platform), both default
  off.
- **Sequencing:** migration `390` → deploy server (flag readable, CRUD live) → deploy web (nav + gallery) →
  enable platform flag in a dogfood tenant → enable per-course for a pilot instructor.
- **Dogfood:** internal course; create/archive kits; verify authz and nav gating.
- **GA criteria:** CRUD + flag gating pass automated tests; no P1s in dogfood.
- **Rollback:** turn off the platform flag (nav + API disappear); schema retained, no data loss.

## 16. Test Plan

- **Unit** — slug uniqueness, archive/restore transitions, flag-merge defaults.
- **Integration** — CRUD authz matrix (instructor vs student vs cross-course); flag-off → 403/404; pagination.
- **End-to-end** — Playwright: enable flag, create kit, rename, archive, restore, verify gallery states.
- **Security** — cross-course kit id probing returns 404; write routes reject non-instructors.
- **Accessibility** — axe on the gallery; keyboard-only create/archive.
- **Performance** — list 500 kits under target; index used (EXPLAIN).
- **Manual** — nav appears/disappears as flags toggle.

## 17. Documentation & Training

- End-user: "Create your first quiz kit" quick-start.
- Admin/instructor: how to enable Live Quizzes per course; platform master switch.
- API reference: new kit endpoints in OpenAPI.
- Runbook: schema location, flag names, archive vs delete policy.

## 18. Open Questions

1. Should kits live at course scope only, or also support **org-level shared kits** from day one?
   (Recommendation: course scope in IQ.1; org/public visibility columns exist but are exercised by IQ.8.)
2. Do we reuse `course.question_bank_enabled` as an implicit dependency, or keep Live Quizzes independently
   flaggable? (Recommendation: independent flag; IQ.2 offers question-bank import only when that flag is on.)

## 19. References

- Existing files: `server/migrations/378_board_foundation.sql` (schema-drop pattern),
  `server/internal/httpserver/course_features.go`, `server/internal/repos/platformconfig/features.go`,
  `clients/web/src/context/course-nav-features-context.tsx`,
  `clients/web/src/pages/lms/course-features-section.tsx`.
- Related plans: [IQ.2 (completed)](IQ.2-kit-authoring-and-question-types.md), [VC.1 (completed)](../visual-collaboration/VC.1-foundation-and-feature-flag.md).
