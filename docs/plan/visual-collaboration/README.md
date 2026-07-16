# Visual Collaboration Boards — Implementation Plans

> Goal: ship an in-house, real-time **visual collaboration board** so institutions no longer pay for a
> separate third-party wall/canvas tool. A *board* is a shared surface where many people post cards
> (text, image, file, link, video, audio, sketch), arrange them in a chosen layout, and react/comment in
> real time. It is delivered as a **per-course feature flag** — the same on/off model as the existing
> Whiteboard (`whiteboard_enabled`) and Collaborative Documents (`collab_docs_enabled`) apps.

## Why this folder exists

Lextures already ships two collaboration surfaces we can build on:

- **Whiteboard** (`course.whiteboards`, migration `230`, flag `whiteboard_enabled`) — a single-canvas drawing
  tool with a per-course boolean flag, a nav link, a page, and REST CRUD.
- **Collaborative Documents** (`collab.collaborative_documents`, flag `collab_docs_enabled`, plan 6.5) — a
  **Y.js CRDT** WebSocket relay (`server/internal/httpserver/collab_docs_ws.go`) that persists binary
  updates and rebroadcasts presence/awareness to every connected peer.

Neither one is a **multi-format, multi-contributor visual board**: many small cards contributed by every
learner, arranged on a wall/grid/canvas/timeline/map, with reactions, comments, moderation, sharing links,
templates, embedding, and export. That is the gap these plans close, reusing the flag pattern from the
Whiteboard and the real-time engine from Collaborative Documents.

## Product naming

- **User-facing:** "Boards" (menu label) / "Collaboration Boards".
- **Internal id / flag:** `visual_boards` — per-course column `course.courses.visual_boards_enabled`
  (default `FALSE`), plus a platform master flag `VisualBoardsEnabled` in
  `server/internal/repos/platformconfig/features.go`.
- **Feature-ID prefix:** `VC` (Visual Collaboration), mirroring `W##`/`M##`/`S##`/`AP.#`.

## Conventions

- **File naming:** `VC.{N}-{kebab-slug}.md`. Every plan follows [`../_TEMPLATE.md`](../_TEMPLATE.md).
- A plan is **ready** when every template section is filled (no `…` placeholders).
- **Schema:** new tables live in a dedicated `board` Postgres schema (`board.boards`, `board.posts`, …),
  keeping the surface self-contained and easy to drop behind the flag.
- **Migrations** continue the repo's global sequence. The highest existing is `377_*`
  (`377_ai_usage_cost_estimated.sql`), so these plans reserve `378_*` onward; each plan states its number.
  Renumber on merge if the sequence has advanced.
- **HTTP:** handlers in `server/internal/httpserver/board_*.go`, repos in `server/internal/repos/board/`,
  routes under `/api/v1/courses/{course_code}/boards/*`, using `apierr.WriteJSON`, `requireCourseAccess`,
  and `courseroles.UserHasPermission` exactly as the Whiteboard/Collab-Docs handlers do.
- **Web:** page components in `clients/web/src/pages/lms/course-boards-*.tsx`, shared UI in
  `clients/web/src/components/boards/`, API client in `clients/web/src/lib/boards-api.ts`, flag surfaced
  through `clients/web/src/context/course-nav-features-context.tsx` and toggled in
  `clients/web/src/pages/lms/course-features-section.tsx`.

## Severity legend

- **BLOCKER** — an institution cannot retire its incumbent wall tool (and its licence spend) without this.
- **MAJOR** — parity gap that loses the head-to-head evaluation.
- **MINOR** — polish / nice-to-have / defence-in-depth.

## Story index

| ID | Plan | Severity | Depends on | Est. |
|---|---|---|---|---|
| VC.1 | ~~Foundation, data model & feature flag~~ → [completed](../../completed/visual-collaboration/VC.1-foundation-and-feature-flag.md) | BLOCKER | — | M |
| VC.2 | ~~Posts & multi-format content~~ → [completed](../../completed/visual-collaboration/VC.2-posts-and-content-types.md) | BLOCKER | VC.1 | L |
| VC.3 | ~~Board layouts & arrangement~~ → [completed](../../completed/visual-collaboration/VC.3-board-layouts-and-arrangement.md) | BLOCKER | VC.1, VC.2 | L |
| VC.4 | ~~Real-time collaboration & presence~~ → [completed](../../completed/visual-collaboration/VC.4-realtime-collaboration-and-presence.md) | BLOCKER | VC.1, VC.2 | L |
| VC.5 | ~~Reactions, comments & assessment~~ → [completed](../../completed/visual-collaboration/VC.5-reactions-comments-assessment.md) | MAJOR | VC.2 | M |
| VC.6 | ~~Sharing, access control & contributors~~ → [completed](../../completed/visual-collaboration/VC.6-sharing-access-contributors.md) | BLOCKER | VC.1 | L |
| VC.7 | ~~Moderation, safety & content governance~~ → [completed](../../completed/visual-collaboration/VC.7-moderation-safety-governance.md) | BLOCKER | VC.2, VC.6 | M |
| VC.8 | ~~Templates, duplication & board creation~~ → [completed](../../completed/visual-collaboration/VC.8-templates-and-duplication.md) | MAJOR | VC.1, VC.3 | M |
| VC.9 | [Embedding, export & presentation](VC.9-embedding-export-presentation.md) | MAJOR | VC.2, VC.3 | M |
| VC.10 | [Admin governance, analytics, quotas & lifecycle](VC.10-admin-analytics-quotas-lifecycle.md) | MAJOR | VC.1 | M |

## Recommended sequencing

1. **VC.1** ships the flag, schema, board list, and an empty board page — nothing else can land without it.
2. **VC.2 → VC.3 → VC.4** turn the empty board into a usable, multi-format, real-time wall. These three are
   the MVP that lets a class replace its incumbent tool for a brainstorm/exit-ticket use case.
3. **VC.6 + VC.7** must ship before any external sharing is exposed (never open link-sharing without the
   moderation and access-control controls).
4. **VC.5, VC.8, VC.9, VC.10** are parity/polish layers that can land in any order once the MVP is stable.

## Cross-cutting requirements (apply to every plan)

- **Privacy / FERPA / COPPA:** student-authored content and attribution are education records; deletion,
  export, and retention must honour the shipped compliance engines (see
  [`../standards/`](../standards/) — especially [S01 DSAR](../standards/S01-unified-data-subject-rights-orchestration.md),
  [S02 retention](../standards/S02-data-retention-deletion-engine.md), and
  [S08 children's privacy](../standards/S08-childrens-privacy-age-assurance-design-codes.md)).
- **Accessibility:** WCAG 2.1 AA for every surface — keyboard-reachable cards, drag alternatives, ARIA
  live regions for real-time updates, and reduced-motion support.
- **Internationalization:** all copy externalised to the web i18n catalog; timezone/locale-aware timestamps.
- **Observability:** metrics, traces, and structured logs via `server/internal/telemetry`.
