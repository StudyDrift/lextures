# VC.3 — Board Layouts & Arrangement

> Implementation plan. Source: net-new capability (real-time visual collaboration boards). Landscape: [visual-collaboration/README](../../plan/visual-collaboration/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.3 |
| **Section** | Visual Collaboration Boards |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Collaboration squad |
| **Depends on** | VC.1, VC.2 |
| **Unblocks** | VC.8, VC.9 |

---

## 1. Problem Statement

The same set of cards means very different things depending on how they are arranged: a free brainstorm
(scattered wall), a structured comparison (columns), a chronology (timeline), a field trip (map), or a
gallery. The incumbent tools win evaluations largely on this **layout flexibility**. VC.3 gives each board a
selectable **layout** and the arrangement mechanics (drag-to-position, columns/sections, sort, zoom/pan) so
one board can be reshaped to fit the pedagogical task.

## 2. Goals

- Offer per-board **layout modes**: Wall (masonry/grid), Stream (single feed), Grid (uniform), Columns
  (named sections / shelf), Freeform Canvas (drag anywhere, pan/zoom), Timeline (by date), and Map (by
  geo-point).
- Let contributors position/reorder cards within the active layout, persisted per board.
- Support **sections/columns** with titles that group cards (the "shelf" model).
- Provide sort/group controls (by recency, author, reactions, section) where the layout allows.
- Keep arrangement data on the existing `board.posts` layout columns from VC.2 plus a small
  sections/settings table.

## 3. Non-Goals

- Real-time propagation of moves (VC.4 broadcasts position changes; VC.3 defines the model + single-user
  interactions).
- The content of cards (VC.2).
- Templates that pre-seed a layout (VC.8 builds on this).
- Presentation/slideshow ordering (VC.9).

## 4. Personas & User Stories

- **As an instructor**, I want to switch a board to Columns so students sort ideas into "Pros / Cons /
  Questions".
- **As an instructor**, I want a Timeline board so students place events in chronological order.
- **As a student**, I want to drag my card where it belongs and have it stay there.
- **As an instructor**, I want a Map board so students pin locations for a geography unit.
- **As an instructor**, I want to lock the layout so students can post but not rearrange others' cards.
- **As a self-learner**, I want a Freeform canvas to cluster my research however I like.

## 5. Functional Requirements

- **FR-1.** Each board MUST have a `layout` setting ∈ `{wall, stream, grid, columns, canvas, timeline, map}`
  (default `wall`), changeable by users with `item:create`.
- **FR-2.** Changing layout MUST NOT lose card content; it re-interprets the arrangement fields (e.g.
  `columns` uses `section_id` + `sort_index`; `canvas` uses `position {x,y,w,h}`; `timeline` uses a card
  `event_date`; `map` uses `lat/lng`).
- **FR-3.** In `columns` layout, the board MUST support named **sections** (create, rename, reorder, delete);
  deleting a section MUST move its cards to an "Unsorted" section, not delete them.
- **FR-4.** Contributors MUST be able to drag a card to a new section and/or reorder it, persisted via a
  `PATCH` that sets `section_id` and `sort_index` (fractional indexing to avoid full renumbering).
- **FR-5.** In `canvas` layout, cards MUST be positioned by `{x, y}` and resizable by `{w, h}`, with
  board-level pan and zoom; positions persist per card.
- **FR-6.** In `timeline` layout, cards MUST expose an `event_date`; the board renders them ordered on an
  axis; cards without a date collect in an "Undated" tray.
- **FR-7.** In `map` layout, cards MUST expose `{lat, lng}` (and optional place label); the board renders
  pins on a map with clustering.
- **FR-8.** The board MUST support a **layout lock**: when locked, non-managers may add cards but may not
  move/reorder existing cards.
- **FR-9.** Sort controls MUST offer at least: newest, oldest, most-reacted (VC.5), and author; grouping by
  section where applicable.
- **FR-10.** All arrangement writes MUST be authorized (author of the card or `item:create`), and MUST
  respect the layout lock.

## 6. Non-Functional Requirements

- **Performance** — dragging is 60fps client-side; position persistence is debounced; a 300-card canvas
  pans/zooms without jank (virtualize off-screen cards).
- **Security** — arrangement endpoints are course-scoped and respect the layout lock and post ownership.
- **Privacy & Compliance** — geo-points on `map` boards are user-entered content; no device geolocation is
  collected without explicit action; treat coordinates as education-record content.
- **Accessibility** — every drag interaction MUST have a keyboard/menu alternative ("move to section…",
  "move up/down", "set position"); layout changes announce via ARIA live region.
- **Scalability** — fractional `sort_index` avoids O(n) renumbering; map clustering handles hundreds of pins.
- **Reliability** — concurrent moves reconcile deterministically (last-write-wins per field until VC.4 CRDT
  positions arrive).
- **Observability** — track layout usage distribution and move counts.
- **Maintainability** — one layout-renderer component per mode behind a shared `<BoardSurface>`.
- **Internationalization** — section titles are user content; axis/date formatting is locale-aware.
- **Backward compatibility** — additive columns; existing boards default to `wall`.

## 7. Acceptance Criteria

- **AC-1.** *Given* a wall board, *when* an instructor switches it to Columns, *then* cards remain and land
  in an "Unsorted" section; new sections can be created and cards dragged between them, persisting on reload.
- **AC-2.** *Given* a canvas board, *when* a student drags and resizes a card, *then* the position/size
  persist and re-render identically after reload.
- **AC-3.** *Given* a timeline board, *when* cards have `event_date`s, *then* they render in date order and
  undated cards appear in the tray.
- **AC-4.** *Given* a map board, *when* a card has coordinates, *then* a pin renders at that location and
  clusters with nearby pins at low zoom.
- **AC-5.** *Given* a locked layout, *when* a student tries to move another student's card, *then* the move
  is rejected (`403`) and the UI prevents the drag.
- **AC-6.** *Given* keyboard-only navigation, *when* a user opens a card's menu, *then* "move to section" and
  "reorder" options are available and functional.
- **AC-7.** *Given* two users reorder within the same section, *when* both save, *then* both cards keep
  distinct, stable `sort_index` values (no collision or reorder thrash).

## 8. Data Model

Migration `380_board_layout.sql`:

```sql
ALTER TABLE board.boards
  ADD COLUMN layout       TEXT NOT NULL DEFAULT 'wall',   -- wall|stream|grid|columns|canvas|timeline|map
  ADD COLUMN layout_locked BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN settings     JSONB NOT NULL DEFAULT '{}';    -- per-layout options (map center/zoom, axis range…)

CREATE TABLE board.sections (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id    UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    sort_index  DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sections_board ON board.sections (board_id);

ALTER TABLE board.posts
  ADD COLUMN event_date TIMESTAMPTZ,             -- timeline
  ADD COLUMN lat        DOUBLE PRECISION,        -- map
  ADD COLUMN lng        DOUBLE PRECISION;
-- section_id, sort_index, position already exist from VC.2 (379).
-- Add FK now that sections exists:
ALTER TABLE board.posts
  ADD CONSTRAINT fk_posts_section FOREIGN KEY (section_id)
  REFERENCES board.sections (id) ON DELETE SET NULL;
```

- **Fractional indexing**: `sort_index` is a double; inserting between two cards uses the midpoint; a
  periodic compaction job renormalizes when gaps get too small.

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| PATCH | `/boards/{board_id}` (extend VC.1) — set `layout`, `layoutLocked`, `settings` | `item:create` |
| POST | `/boards/{board_id}/sections` | `item:create` |
| PATCH | `/boards/{board_id}/sections/{section_id}` (rename/reorder) | `item:create` |
| DELETE | `/boards/{board_id}/sections/{section_id}` | `item:create` |
| PATCH | `/boards/{board_id}/posts/{post_id}/arrange` — `{sectionId?, sortIndex?, position?, eventDate?, lat?, lng?}` | author or `item:create` (blocked if `layout_locked` for non-managers) |

- **Rate limits**: arrange endpoint is high-frequency during drag; client debounces and batches; server
  applies a generous per-user limiter.
- **OpenAPI**: sections + arrange schemas.

## 10. UI / UX

- **`<BoardSurface>`** dispatches to one renderer per layout: `WallLayout`, `StreamLayout`, `GridLayout`,
  `ColumnsLayout`, `CanvasLayout`, `TimelineLayout`, `MapLayout` (all in `components/boards/layouts/`).
- **Layout switcher** in the board header (grid/columns/canvas/timeline/map icons) with a confirm when a
  switch would hide arrangement data (e.g., leaving canvas).
- **Drag**: pointer-based DnD with a keyboard/menu fallback on each card; a "Lock arrangement" toggle for
  managers.
- **Map**: renders with a self-hosted/tile provider that satisfies the CSP; no third-party script beyond
  approved tiles.
- **States**: empty section placeholder ("Drop cards here"), off-screen virtualization on canvas, undated
  tray on timeline.
- **Mobile**: columns become horizontally swipeable; canvas supports pinch-zoom; drag via long-press.
- **Accessibility**: ARIA `listbox`/`option` semantics for sortable lists; live-region announcements on
  move; respects reduced-motion.
- **Copy & i18n**: `boards.layout.*`, `boards.section.*` keys.

## 11. AI / ML Considerations

Not AI-touching. (Optional future: "auto-cluster cards into sections" — noted in VC.10.)

## 12. Integration Points

- **New**: `server/internal/repos/board/sections.go`, `board/arrange.go`,
  `server/internal/httpserver/board_layout_http.go`, `clients/web/src/components/boards/layouts/*`.
- **Reuse**: existing DnD utilities if present; whiteboard pan/zoom math (`use-whiteboard-canvas.ts`) as a
  reference for the canvas layout.

## 13. Dependencies & Sequencing

- Must ship after: VC.1, VC.2.
- Must ship before: VC.8 (templates seed layouts/sections), VC.9 (presentation order derives from layout).
- Interacts with VC.4: position/section fields become CRDT-synced fields once VC.4 lands.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Drag perf on large boards | M | M | Virtualize; debounce persistence; batch arrange writes |
| Fractional index exhaustion | L | M | Midpoint insert + periodic renormalization job |
| Map tiles violate CSP / cost | M | M | Self-host or use an approved tile source; lazy-load |
| Data loss switching layouts | M | H | Never delete arrangement fields on switch; confirm dialog |

## 15. Rollout Plan

- **Flag**: gated by `visual_boards_enabled`; individual layouts can hide behind board settings while
  stabilizing (ship wall/stream/grid/columns first, then canvas/timeline/map).
- **Sequencing**: migration `380` → deploy → enable advanced layouts after canvas perf validated.
- **Rollback**: force `layout = 'wall'` via settings; data preserved.

## 16. Test Plan

- **Unit** — fractional indexing insert/renormalize; layout-field interpretation; lock enforcement.
- **Integration** — arrange authz (author/manager/locked); section delete moves cards to Unsorted.
- **End-to-end** — Playwright: switch layouts; drag between columns; resize on canvas; timeline ordering;
  map pin placement; locked board blocks moves.
- **Security** — arrange endpoint respects lock + ownership; coordinate bounds validated.
- **Accessibility** — keyboard move flows; live-region announcements; reduced-motion.
- **Performance** — 300-card canvas pan/zoom; drag frame rate.
- **Manual** — mobile swipe columns; pinch-zoom canvas.

## 17. Documentation & Training

- End-user: "Choose a layout for your board" with a screenshot per mode.
- Instructor: when to use columns vs timeline vs map; locking arrangement.
- API reference: sections + arrange endpoints.

## 18. Open Questions

1. Do we need a **Table/grid-with-columns** layout (rows = contributors, columns = prompts) for v1, or is
   Columns sufficient? (Recommendation: defer Table to a fast-follow.)
2. Should map default center/zoom be instructor-set or auto-fit to pins? (Recommendation: auto-fit with
   manual override in `settings`.)
3. Per-section posting limits (one card per student per section)? (Defer to VC.7 governance.)

## 19. References

- Existing files: `clients/web/src/components/whiteboard/use-whiteboard-canvas.ts` (pan/zoom reference),
  `board.posts` layout columns from [VC.2](VC.2-posts-and-content-types.md).
- Related plans: [VC.4](../../plan/visual-collaboration/VC.4-realtime-collaboration-and-presence.md),
  [VC.8](../../plan/visual-collaboration/VC.8-templates-and-duplication.md),
  [VC.9](../../plan/visual-collaboration/VC.9-embedding-export-presentation.md).
