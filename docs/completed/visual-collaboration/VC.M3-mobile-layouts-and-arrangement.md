# VC.M3 — Mobile Boards: Layouts & Arrangement

> Implementation plan. Source: mobile parity for board layouts. Landscape: [visual-collaboration/README](../../plan/visual-collaboration/README.md). Mirrors web [VC.3](VC.3-board-layouts-and-arrangement.md), adapting the seven layout modes to a small touch screen and reusing the REST arrange/sections endpoints.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.M3 |
| **Section** | Visual Collaboration Boards — Mobile |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Mobile squad |
| **Depends on** | VC.M1, VC.M2 |
| **Unblocks** | (enhances VC.M4 sync of positions) |

---

## 1. Problem Statement

A pile of cards means different things depending on arrangement — a scattered brainstorm, sorted columns, a
timeline, a map. Web VC.3 gives a board seven layout modes; mobile must **honor the board's chosen layout**
(so a phone user sees the same structure as the class projector) and let contributors place their own cards.
The challenge is translating drag-on-a-canvas interactions to touch, where screens are small and precise
dragging is hard. VC.M3 renders each layout in a mobile-appropriate way and provides touch + menu-based
arrangement.

## 2. Goals

- Render each board `layout ∈ {wall, stream, grid, columns, canvas, timeline, map}` in a form suited to a
  phone, reading the same `layout` / `settings` the web board uses.
- Let contributors position/reorder their cards where the layout and permissions allow, persisted via the
  REST arrange endpoint (fractional `sortIndex`, `sectionId`, `position`, `eventDate`, `lat/lng`).
- Support **columns/sections** as horizontally swipeable lanes with drag-between and a menu ("move to
  section…") alternative.
- Support **timeline** (date-ordered list/axis) and **map** (native map pins with clustering) reads, and
  card date/geo entry where appropriate.
- Respect the board **layout lock**: when locked, non-managers may post but not rearrange others' cards.

## 3. Non-Goals

- Real-time propagation of moves (VC.M4 broadcasts/refetches; VC.M3 defines the mobile interactions).
- Card content (VC.M2).
- A full free-pan/zoom **canvas authoring** experience matching desktop precision — mobile canvas is
  pan/pinch **view** + long-press move, not pixel-perfect resize (resize is optional/deferred).
- Templates that pre-seed layouts (web VC.8; deferred on mobile).

## 4. Personas & User Stories

- **As a student**, I want the board to look the way my instructor set it up (columns, timeline, map) when I
  open it on my phone.
- **As a student**, I want to drag my card into the right column with a long-press, or use a menu if dragging
  is fiddly.
- **As a student**, I want to place my card on the timeline by setting its date.
- **As a student**, I want to drop a pin on the map for my location card.
- **As an instructor**, I want a locked layout so students post but don't rearrange each other's cards.

## 5. Functional Requirements

- **FR-1.** The board surface MUST read `board.layout` and render the matching mobile view: **wall** → masonry
  /2-col grid; **stream** → single vertical feed; **grid** → uniform grid; **columns** → horizontally
  swipeable lanes; **canvas** → pan/pinch surface with absolutely-positioned cards; **timeline** →
  date-ordered list with an axis and an "Undated" tray; **map** → native map with pins.
- **FR-2.** Switching layout is a **manager** action (`item:create`) via `PATCH …/boards/{id}` (`layout`);
  switching MUST NOT lose card content — it re-interprets arrangement fields exactly as web does.
- **FR-3.** In **columns**, the app MUST list sections, allow create/rename/reorder/delete for managers
  (`…/sections` endpoints); deleting a section moves its cards to "Unsorted" (server behaviour), and the UI
  reflects that.
- **FR-4.** Contributors MUST be able to move a card to another section and/or reorder it via long-press drag
  **and** a card menu ("Move to section…", "Move up/down"), persisted with
  `PATCH …/posts/{id}/arrange {sectionId?, sortIndex?}` using midpoint fractional indexing computed
  client-side.
- **FR-5.** In **canvas**, cards render at their `{x,y}` (and `{w,h}`); the surface supports pan and
  pinch-zoom; a card can be repositioned by long-press-drag, persisting `position`. Resize is optional in v1.
- **FR-6.** In **timeline**, a card exposes an editable `eventDate` (native date picker); undated cards go to
  the "Undated" tray; the axis orders by date.
- **FR-7.** In **map**, a card exposes `{lat,lng}` (drop-a-pin or "use current location" with explicit
  consent); pins cluster at low zoom using the native map's clustering.
- **FR-8.** When `board.layoutLocked` is true (or the resolved `canArrange` is false), the app MUST disable
  drag/reorder for non-managers and hide arrange affordances; attempted arrange MUST be prevented client-side
  and any server `403` handled.
- **FR-9.** Sort controls MUST offer newest / oldest / most-reacted (VC.M5) / author where the layout allows.
- **FR-10.** Arrange writes MUST be debounced/batched (drag emits one persisted write on release, not per
  frame).

## 6. Non-Functional Requirements

- **Performance** — 60fps touch drag; a 300-card canvas pans/pinches without jank (virtualize off-screen
  cards); arrange persistence debounced.
- **Security** — arrange endpoints are course-scoped and enforce lock + ownership server-side; the client
  never assumes it can move a card it couldn't.
- **Privacy & Compliance** — device geolocation for map cards is collected **only** on explicit action, with a
  permission prompt; coordinates are education-record content.
- **Accessibility** — every drag has a menu/keyboard alternative; layout changes announce via the platform
  live-region equivalent; respects Reduce Motion; map has a list fallback for pins.
- **Scalability** — fractional `sortIndex` avoids renumbering; map clustering handles hundreds of pins.
- **Reliability** — concurrent moves reconcile last-write-wins per field (until VC.M4/CRDT positions arrive);
  a failed arrange rolls back the optimistic move.
- **Internationalization** — section titles are user content; date/axis formatting locale-aware; RTL flips
  column order correctly.
- **Backward compatibility** — additive; a board with an unknown/future layout falls back to **stream**.

## 7. Acceptance Criteria

- **AC-1.** *Given* a columns board, *when* opened on mobile, *then* sections render as swipeable lanes and a
  student can drag or menu-move their card between them, persisting on reload.
- **AC-2.** *Given* a canvas board, *when* a student long-press-drags a card, *then* the position persists and
  re-renders identically after reload.
- **AC-3.** *Given* a timeline board, *when* a card has an `eventDate`, *then* it renders in date order and
  undated cards appear in the tray.
- **AC-4.** *Given* a map board, *when* a card has coordinates, *then* a pin renders and clusters at low zoom;
  a list fallback lists the same pins.
- **AC-5.** *Given* a locked layout, *when* a student tries to move another student's card, *then* the drag is
  prevented and any server attempt returns `403`.
- **AC-6.** *Given* assistive tech, *when* a user opens a card menu, *then* "Move to section" / "Move up/down"
  are available and functional.
- **AC-7.** *Given* an unknown layout value, *when* rendered, *then* the board falls back to stream (no crash).

## 8. Data Model

No server schema change — VC.3's `board.sections` + `board.posts` layout columns (`section_id`, `sort_index`,
`position`, `event_date`, `lat`, `lng`) already exist. Client adds a `Section` model and arrange-input types
mirroring web `boards-api.ts` (`ArrangeBoardPostInput`, `BoardPostPosition`).

## 9. API Surface

No new endpoints. Mobile consumes web VC.3's routes:

| Verb | Path | Auth |
|---|---|---|
| PATCH | `/boards/{id}` — `layout`, `layoutLocked`, `settings` | `item:create` |
| POST | `/boards/{id}/sections` | `item:create` |
| PATCH | `/boards/{id}/sections/{sid}` (rename/reorder) | `item:create` |
| DELETE | `/boards/{id}/sections/{sid}` | `item:create` |
| PATCH | `/boards/{id}/posts/{postId}/arrange` — `{sectionId?, sortIndex?, position?, eventDate?, lat?, lng?}` | author or `item:create`; blocked when locked for non-managers |

## 10. UI / UX

- **`BoardSurface`** dispatches to one mobile renderer per layout (iOS `Features/Boards/Layouts/*`, Android
  `features/boards/layouts/*`): `WallLayout`, `StreamLayout`, `GridLayout`, `ColumnsLayout` (swipeable lanes),
  `CanvasLayout` (pan/pinch), `TimelineLayout`, `MapLayout` (native `MapKit` / Google Maps Compose).
- **Layout switcher** (managers): an overflow action with icons; confirm dialog when a switch would hide
  arrangement data (e.g. leaving canvas).
- **Arrange**: long-press to pick up a card (haptic), drag to reorder/move; every card also has a menu
  fallback. A manager "Lock arrangement" toggle.
- **States**: empty-section placeholder ("Drop cards here"), off-screen virtualization on canvas, "Undated"
  timeline tray, map list-fallback.
- **Accessibility**: reorder actions in the card menu; live-region announcements on move; Reduce Motion
  respected; map pins have accessible labels + list alternative.
- **Copy & i18n**: `boards.layout.*`, `boards.section.*` keys.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **Reuse**: native map SDK already permitted by the app (or add via the platform's first-party map — MapKit
  iOS / Maps Compose Android) respecting existing map usage; whiteboard pan/zoom math as a reference for
  canvas.
- **New (iOS)**: `Core/LMS/LMSAPIBoardLayout.swift`, `Features/Boards/Layouts/*.swift` → regenerate project.
- **New (Android)**: `core/lms/BoardLayoutApi.kt`, `features/boards/layouts/*.kt`.

## 13. Dependencies & Sequencing

- Must ship after: VC.M1, VC.M2.
- Interacts with VC.M4: position/section fields become CRDT-synced (or refetch-driven) once realtime lands.
- Shared infra: native maps; no server change.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Touch drag precision on small screens | H | M | Long-press pickup + menu fallback; generous drop targets; snap-to-lane |
| Canvas perf on large boards | M | M | Virtualize off-screen cards; debounce persistence |
| Map SDK / API-key / CSP constraints | M | M | Use the platform first-party map; key via existing config; list fallback |
| Fractional index exhaustion | L | M | Midpoint insert; rely on server renormalization job |
| Data loss switching layouts | M | H | Never clear arrangement fields client-side; confirm dialog before hiding layouts |

## 15. Rollout Plan

- **Flag**: gated by `visualBoardsEnabled`; advanced layouts (canvas/timeline/map) can ship after wall/stream/
  grid/columns are validated on device.
- **Sequencing**: list-style layouts (wall/stream/grid/columns) → canvas → timeline → map.
- **Rollback**: force stream rendering client-side if a layout renderer misbehaves; data preserved.

## 16. Test Plan

- **Unit** — fractional-index midpoint; layout-field interpretation; lock enforcement; unknown-layout fallback.
- **Integration** — arrange authz (author/manager/locked); section delete → Unsorted; date/geo persistence.
- **End-to-end (device)** — drag between columns; long-press move on canvas; timeline ordering; map pin drop;
  locked board blocks moves.
- **Accessibility** — menu move flows; live-region announcements; Reduce Motion; map list fallback.
- **Performance** — 300-card canvas pan/pinch; drag frame rate.
- **Manual** — swipe columns; pinch-zoom canvas; geolocation consent path.

## 17. Documentation & Training

- End-user: "Arrange cards on mobile" (drag vs. menu) and per-layout notes.
- Instructor: choosing/locking a layout from the phone.

## 18. Open Questions

1. Do we support **canvas card resize** on mobile v1, or move-only? (Recommendation: move-only; resize is a
   fast-follow.)
2. Which map SDK — first-party per platform (MapKit / Maps Compose) to avoid extra keys/CSP? (Recommendation:
   first-party.)
3. Should mobile allow **layout switching** at all, or is that instructor-on-web only? (Recommendation: allow
   for managers; it is one PATCH.)

## 19. References

- Web plan: [VC.3](VC.3-board-layouts-and-arrangement.md); web layouts
  `clients/web/src/components/boards/layouts/*`.
- Related mobile plans: [VC.M2](VC.M2-mobile-posts-and-content.md), [VC.M4](VC.M4-mobile-realtime-and-presence.md).
