# VC.8 — Templates, Duplication & Board Creation

> Implementation plan. Source: net-new capability (real-time visual collaboration boards). Landscape: [visual-collaboration/README](../../plan/visual-collaboration/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.8 |
| **Section** | Visual Collaboration Boards |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Collaboration squad |
| **Depends on** | VC.1, VC.3 |
| **Unblocks** | — |

---

## 1. Problem Statement

Teachers adopt a tool faster when they don't start from a blank canvas. The activities that make boards
valuable — a brainstorm wall, an exit ticket, a KWL chart, a discussion prompt, a gallery, a map, a timeline
— are repeatable patterns. VC.8 delivers **starter templates**, **board duplication**, and a lightweight
**template gallery** so an instructor can spin up a ready-to-use board in one click and reuse boards across
courses and terms.

## 2. Goals

- Ship a curated set of **starter templates** (layout + sections + prompt cards + recommended settings).
- Let instructors **duplicate** an existing board (structure only, or structure + posts) within a course.
- Let instructors **save a board as a template** (course-local, and optionally org-shared) for reuse.
- Provide a **create flow** that offers Blank / From template / Duplicate at board creation.
- Support **cross-course copy** so a board built in one course can seed another the instructor owns.

## 3. Non-Goals

- Authoring the runtime layouts/sections themselves (VC.3 owns those; VC.8 seeds them).
- A public template marketplace / community sharing (future; org-shared only in v1).
- AI-generated boards from a prompt (note as Open Question / future).
- Import from third-party board formats (out of scope).

## 4. Personas & User Stories

- **As an instructor**, I want to create an "Exit ticket" board in one click with the prompt already set up.
- **As an instructor**, I want to duplicate last term's brainstorm board into this term's course.
- **As an instructor**, I want to save my custom board as a template so my department can reuse it.
- **As a department lead**, I want org-shared templates so every teacher starts consistent.
- **As a self-learner**, I want a "research board" template to organize my study.

## 5. Functional Requirements

- **FR-1.** The system MUST ship built-in templates, each defining: `layout` (VC.3), sections, seed posts
  (prompt cards, typically `text`), and default settings (reaction mode, moderation, attribution).
- **FR-2.** The create flow MUST offer **Blank**, **From template** (built-in or saved), and **Duplicate an
  existing board**.
- **FR-3.** Creating **from a template** MUST instantiate a new board copying the template's layout,
  sections, and seed posts; seed posts MUST be attributed to the creating instructor (not to the template).
- **FR-4.** **Duplicating a board** MUST support two modes: *structure only* (layout + sections + settings,
  no student posts) and *full copy* (also copy posts/attachments); full copy MUST re-reference or copy
  attachment objects safely (no shared mutable object between boards).
- **FR-5.** **Save as template** MUST persist a `board_templates` row capturing the source board's layout,
  sections, chosen seed posts, and settings, scoped `course` or `org`.
- **FR-6.** Templates MUST be listable in a **gallery** filtered by scope (built-in, this course, org),
  searchable by title/description/tag.
- **FR-7.** Cross-course duplication MUST be allowed only when the instructor has create permission in the
  **target** course; content copied MUST respect the source's access constraints (do not copy student PII
  into a template unless the instructor explicitly includes posts).
- **FR-8.** Copying MUST NOT carry over reactions, comments, reports, moderation log, or share links —
  a duplicate/template is a fresh board.
- **FR-9.** Built-in templates MUST be i18n-ready (prompt copy externalised or shipped per-locale).

## 6. Non-Functional Requirements

- **Performance** — instantiate-from-template completes < 1 s for typical templates; full board copy is a
  background job when large (many attachments), with progress.
- **Security** — cross-course copy re-checks target-course permission; attachment copy stays within the
  tenant's storage; org templates readable only within the org.
- **Privacy & Compliance** — "save as template" defaults to **structure only** (no student content); saving
  student posts into an org template requires explicit confirmation and respects FERPA (warn about sharing
  student work); align with [S02](../standards/S02-data-retention-deletion-engine.md).
- **Accessibility** — template gallery and create flow fully accessible; template cards have descriptions.
- **Scalability** — templates are small JSON blobs; full-copy heavy work offloaded to the job queue.
- **Reliability** — copy is transactional or resumable; partial copies are cleaned up on failure.
- **Observability** — counters for template usage, duplications, saves; most-used templates surfaced.
- **Maintainability** — one serializer defines the board→template→board mapping, reused by all three flows.
- **Internationalization** — built-in template content localised.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* the create flow, *when* an instructor picks the "Exit ticket" template, *then* a new
  board is created with the template's layout, section(s), and prompt card, attributed to the instructor.
- **AC-2.** *Given* an existing board, *when* the instructor duplicates it *structure only*, *then* the copy
  has the layout/sections/settings but zero student posts.
- **AC-3.** *Given* a full copy of a board with image cards, *when* copy completes, *then* the new board's
  images render and are independent objects (deleting one board's image doesn't affect the other).
- **AC-4.** *Given* "save as template (org)", *when* saved with structure only, *then* it appears in the org
  gallery for other instructors and contains no student content.
- **AC-5.** *Given* cross-course duplication, *when* the instructor lacks create permission in the target
  course, *then* the copy is refused.
- **AC-6.** *Given* any duplicate/template instantiation, *when* created, *then* it carries no reactions,
  comments, reports, or share links from the source.
- **AC-7.** *Given* a locale, *when* a built-in template is instantiated, *then* prompt copy renders in that
  locale where a translation exists.

## 8. Data Model

Migration `394_board_templates.sql`:

```sql
CREATE TABLE board.board_templates (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope         TEXT NOT NULL,               -- builtin|course|org
    course_id     UUID REFERENCES course.courses (id) ON DELETE CASCADE,   -- for course scope
    org_id        UUID,                         -- for org scope (tenant)
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    tags          TEXT[] NOT NULL DEFAULT '{}',
    definition    JSONB NOT NULL,               -- {layout, settings, sections[], seedPosts[]}
    created_by    UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_board_templates_scope ON board.board_templates (scope, org_id, course_id);
```

- Built-in templates are seeded (migration insert or code registry) with `scope = 'builtin'`.
- `definition.seedPosts[]` contains only instructor-authored prompt cards; never student content.

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| GET | `/api/v1/board-templates?scope=builtin\|course\|org&courseCode=…` | authenticated (org/course scoped) |
| POST | `/api/v1/courses/{code}/boards?from=template:{templateId}` | `item:create` (target course) |
| POST | `/api/v1/courses/{code}/boards?from=board:{boardId}&mode=structure\|full` | `item:create` (target course) |
| POST | `/api/v1/courses/{code}/boards/{boardId}/save-as-template` — `{scope, includePosts}` | `item:create` |

- Extends VC.1's board-create with a `from` selector. Full-copy returns `202` + a job id when offloaded.
- **OpenAPI**: templates + copy flows.

## 10. UI / UX

- **Create dialog** (`components/boards/create-board-dialog.tsx`): three tabs — Blank, Templates (gallery),
  Duplicate (pick a board + mode). Template cards show a preview thumbnail, title, description, tags.
- **Template gallery**: filter chips (Built-in / This course / Organization), search box, empty state.
- **Save-as-template**: from a board's menu, choose scope (course/org) and whether to include posts (with a
  FERPA warning when including student content).
- **States**: copy-in-progress (progress bar for full copy), copy-failed (retry), no-templates empty.
- **Mobile**: dialog is a full-screen sheet.
- **Accessibility**: tabbed dialog with proper roles; gallery cards are buttons with descriptions.
- **Copy & i18n**: `boards.create.*`, `boards.template.*` keys; built-in template content localised.

## 11. AI / ML Considerations

Optional/future (flagged off): "Generate a board from a prompt" — an AI produces a layout + seed cards; would
use the AI provider path with cost budget and human review before creation. Out of scope for GA.

## 12. Integration Points

- **Reuse**: VC.1 board create, VC.3 layout/sections model, VC.2 attachment copy (object-store copy via
  `filestorage`), the background job queue for full copies, i18n catalog.
- **New**: `server/internal/repos/board/templates.go`, `board/copy.go` (serializer + copier),
  `server/internal/httpserver/board_templates_http.go`, `clients/web/src/components/boards/create-board-dialog.tsx`.

## 13. Dependencies & Sequencing

- Must ship after: VC.1, VC.3 (needs layouts/sections to seed). Benefits from VC.2 for seed media.
- Must ship before: nothing hard.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Full copy shares mutable attachment objects | M | H | Copy objects in storage; never alias; independence test (AC-3) |
| Student PII leaking into org templates | M | H | Default structure-only; explicit include-posts confirm + FERPA warning |
| Large full copies block requests | M | M | Offload to job queue (`202` + progress) |
| Built-in templates untranslated | L | L | i18n-ready definition; fall back to default locale |

## 15. Rollout Plan

- **Flag**: gated by `visual_boards_enabled`. Org-shared templates additionally require an org-templates
  setting (default on for HE, configurable).
- **Sequencing**: migration `394` + seed built-ins → ship Blank/Template/Duplicate → enable save-as-template
  (course) → enable org scope.
- **Rollback**: hide the templates tab; blank create still works; data retained.

## 16. Test Plan

- **Unit** — board↔template serializer; copy modes; scope filtering; i18n resolution.
- **Integration** — instantiate each built-in; structure-only vs full copy; cross-course permission;
  no reactions/comments/shares carried; attachment independence.
- **End-to-end** — Playwright: create from template; duplicate structure-only and full; save-as-template
  (course + org) and reuse.
- **Security** — target-course authz; org-template tenant isolation; PII-include confirmation.
- **Accessibility** — create dialog + gallery axe/keyboard.
- **Performance** — full-copy job with many attachments; progress reporting.
- **Manual** — localised template instantiation.

## 17. Documentation & Training

- End-user: creating from a template; duplicating a board; saving your own template.
- Admin: org template governance.
- API reference: templates + copy endpoints.

## 18. Open Questions

1. Ship a public/community template marketplace later? (Recommendation: org-only for v1; revisit.)
2. Should "save as template" snapshot attachments or reference them? (Recommendation: snapshot/copy for
   independence; dedupe by content hash to save storage.)
3. Which built-in templates ship at GA? (Proposed: Brainstorm wall, Exit ticket, KWL, Discussion, Gallery,
   Timeline, Map, Q&A — finalize with instructional design.)

## 19. References

- Existing files: `filestorage` (object copy), background job queue, i18n catalog.
- Related plans: [VC.1](VC.1-foundation-and-feature-flag.md),
  [VC.3](VC.3-board-layouts-and-arrangement.md), [VC.2](VC.2-posts-and-content-types.md).
