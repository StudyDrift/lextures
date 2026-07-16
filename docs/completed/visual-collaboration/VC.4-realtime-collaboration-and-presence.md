# VC.4 — Real-Time Collaboration & Presence

> Completed implementation plan. Source: net-new capability (real-time visual collaboration boards). Landscape: [visual-collaboration/README](../../plan/visual-collaboration/README.md). Directly reuses the Y.js CRDT WebSocket relay built for Collaborative Documents (`server/internal/httpserver/collab_docs_ws.go`, plan 6.5); shared helpers live in `server/internal/yrelay`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.4 |
| **Section** | Visual Collaboration Boards |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Collaboration squad |
| **Depends on** | VC.1, VC.2 |
| **Unblocks** | (enhances VC.3, VC.5) |

---

## 1. Problem Statement

The magic moment of a shared board is watching cards appear and move **as your classmates add them**,
live, without refreshing. Without real-time sync, a board is just a slow forum. Lextures already runs a
production **Y.js CRDT relay** for Collaborative Documents; VC.4 stands up the same relay for boards so
adds, edits, moves, and deletes propagate instantly, with **presence** (who's here, live cursors) on top.

## 2. Goals

- Provide a per-board WebSocket relay that broadcasts card add/edit/move/delete to all connected peers in
  real time, using the existing Y.js sync + awareness protocol as the transport.
- Model the board's shared state (posts, positions, sections) as a CRDT document so concurrent edits merge
  without conflict.
- Show **presence**: avatars of connected members and live cursors/selection in canvas layouts.
- Persist CRDT updates so a late-joining or reconnecting client reconstructs current state (same
  store-and-replay approach as collab docs).
- Reconcile the CRDT with the REST/DB representation from VC.2/VC.3 so both views stay consistent.

## 3. Non-Goals

- The card content model (VC.2) and layout mechanics (VC.3) — VC.4 syncs them, it does not define them.
- Comments/reactions transport (VC.5 may reuse this relay but owns its own events).
- Voice/video presence (out of scope; live sessions is a separate feature, plan 6.4).
- Offline-first authoring beyond reconnect replay (basic offline queue only).

## 4. Personas & User Stories

- **As a student**, I want to see classmates' cards appear the instant they post, without refreshing.
- **As a student**, I want to see who else is on the board right now.
- **As an instructor**, I want to watch cards move live while facilitating a sort activity.
- **As a student on a flaky connection**, I want my changes to sync when I reconnect, not get lost.
- **As an instructor**, I want live cursors on a canvas board so I can point while I talk.

## 5. Functional Requirements

- **FR-1.** The system MUST expose `GET /api/v1/courses/{code}/boards/{board_id}/ws` that upgrades to a
  WebSocket and speaks the same minimal Y.js protocol as `collab_docs_ws.go` (byte-0 sync, byte-1
  awareness).
- **FR-2.** The WS MUST authenticate exactly like the collab-doc WS: first text message `{"authToken":…}`,
  verified via `JWTSigner`, then `enrollment.UserHasAccess(courseCode, userID)` and board-belongs-to-course
  checks.
- **FR-3.** The relay MUST maintain an in-process **room per board** (map of connected clients), joining a
  new client to the room and removing it on disconnect, mirroring `collabRoom`/`getOrCreateRoom`.
- **FR-4.** Sync updates (byte 0) MUST be persisted to a `board.board_updates` table and rebroadcast to
  other clients; awareness updates (byte 1) MUST be relayed only, never persisted.
- **FR-5.** On connect, the server MUST replay all stored updates so the client reconstructs current board
  state, then request the client's state vector (empty syncStep1), exactly as collab docs does.
- **FR-6.** The shared CRDT document MUST represent the board's posts (id, content ref, section, sort_index,
  position, event_date, lat/lng) so add/move/edit/delete converge; card **bodies/attachments** remain in
  the REST/DB model and are referenced by id from the CRDT.
- **FR-7.** A server-side (or scheduled) **reconciler** MUST fold CRDT arrangement state back into
  `board.posts` (and forward DB-side changes into the CRDT) so REST reads (VC.2/VC.3), export (VC.9), and
  read-only embeds stay correct; conflicts resolve via the CRDT as source of truth for arrangement.
- **FR-8.** Presence/awareness MUST carry `{userId, displayName, color, cursor?, selectionPostId?}`; the UI
  MUST render connected avatars and, on canvas layouts, live cursors.
- **FR-9.** The relay MUST cap message size and per-connection rate, and MUST drop/limit clients that exceed
  it, to prevent a single peer from flooding a room.
- **FR-10.** Periodic **compaction** MUST merge the append-only `board_updates` into a snapshot to bound
  replay cost (a maintenance job), analogous to collab-doc snapshots.
- **FR-11.** When the feature/course flag is off or the board is archived, the WS MUST refuse the upgrade.

## 6. Non-Functional Requirements

- **Performance** — update fan-out p95 < 150 ms within a room of 40 peers; connect+replay < 1.5 s for a
  200-card board (post-compaction).
- **Security** — token-verified, enrollment-checked, board-scoped; awareness cannot spoof another user's id
  (server stamps identity, does not trust client-claimed userId for authz); origin patterns configured.
- **Privacy & Compliance** — presence reveals only in-course identity already visible to the roster; no
  cross-course leakage; updates are education-record content (deletion/export via [S02](../../plan/standards/S02-data-retention-deletion-engine.md)).
- **Accessibility** — real-time inserts announce via ARIA live region (polite); cursors are decorative and
  not required to operate the board.
- **Scalability** — single-process rooms match the existing collab-doc design; a horizontal-scale path
  (shared pub/sub such as Postgres LISTEN/NOTIFY or Redis) is documented as a follow-up if multi-instance WS
  fan-out is needed.
- **Reliability** — reconnect replays missed updates; CRDT idempotency tolerates duplicate delivery;
  compaction prevents unbounded growth.
- **Observability** — gauges for rooms/clients, counters for updates relayed/persisted, replay latency.
- **Maintainability** — factor the shared Y.js relay helpers (`writeVarUint`, `encodeSyncUpdate`,
  `extractUpdateFromMsg`, room registry) out of `collab_docs_ws.go` into a reusable package so boards and
  docs share one implementation.
- **Internationalization** — presence display names respect the user's display preferences.
- **Backward compatibility** — additive; no change to collab docs behaviour.

## 7. Acceptance Criteria

- **AC-1.** *Given* two members on the same board, *when* one adds a card, *then* it appears on the other's
  screen within ~150 ms without a refresh.
- **AC-2.** *Given* a member drags a card on a canvas board, *when* they release, *then* peers see the card
  move live.
- **AC-3.** *Given* a client disconnects and reconnects, *when* it rejoins, *then* it reconstructs the full
  current board state and shows no duplicate cards.
- **AC-4.** *Given* two members edit different cards simultaneously, *when* both sync, *then* both edits
  survive (no lost update).
- **AC-5.** *Given* members on the board, *when* the presence panel renders, *then* it shows each connected
  member's avatar, and canvas layouts show their live cursors.
- **AC-6.** *Given* a REST read (or export) after live edits, *when* the reconciler has run, *then* the DB
  representation matches what live clients see.
- **AC-7.** *Given* the course flag is off, *when* a client tries to open the board WS, *then* the upgrade is
  refused.
- **AC-8.** *Given* a client sends an oversized/flooding stream, *when* the cap is exceeded, *then* the
  server throttles/closes that connection without affecting others.

## 8. Data Model

Migration `381_board_realtime.sql`:

```sql
CREATE TABLE board.board_updates (
    id         BIGSERIAL PRIMARY KEY,
    board_id   UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    author_id  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    update     BYTEA NOT NULL,               -- raw Y.js binary update
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_board_updates_board ON board.board_updates (board_id, created_at);

CREATE TABLE board.board_snapshots (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id   UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    state      BYTEA NOT NULL,               -- compacted Y.js document state
    taken_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_board_snapshots_board ON board.board_snapshots (board_id, taken_at);
```

Mirrors `collab.collab_doc_updates` / `collab.collab_doc_snapshots`. Compaction writes a snapshot and prunes
folded updates.

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| GET (upgrade) | `/boards/{board_id}/ws` | first-message `authToken` → JWT + enrollment + board-in-course |

- **Protocol**: identical framing to collab docs — binary `[0, subType, varLen, …update]` for sync, `[1, …]`
  for awareness. Server persists sync `subType ∈ {1,2}` payloads and relays; relays awareness.
- **Rate-limit**: per-connection message-size and message-rate caps.
- **OpenAPI**: document the WS endpoint and handshake (as collab docs is documented).

## 10. UI / UX

- **`useBoardRealtime` hook** (`clients/web/src/lib/boards-realtime.ts`): wraps a Y.js doc + `y-websocket`-
  style provider pointed at the board WS, exposing the shared posts map and awareness.
- **Presence bar**: stacked avatars of connected members with tooltips; overflow count.
- **Live cursors**: colored cursors + name labels on canvas layouts; hidden in non-spatial layouts.
- **States**: connecting (subtle banner), reconnecting (toast), offline (queued changes indicator), synced.
- **Optimistic UX**: local changes apply immediately; the CRDT converges; no spinner per card.
- **Accessibility**: incoming cards announced politely; presence bar labelled; cursors decorative
  (`aria-hidden`).
- **Copy & i18n**: `boards.presence.*`, `boards.sync.*` keys.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **Reuse / refactor**: `server/internal/httpserver/collab_docs_ws.go` — extract the relay core into a shared
  package (e.g. `server/internal/yrelay`) consumed by both collab docs and boards; reuse `JWTSigner`,
  `enrollment.UserHasAccess`, `course.GetIDByCourseCode`.
- **New**: `server/internal/httpserver/board_ws.go`, `server/internal/repos/board/updates.go`,
  reconciler + compaction jobs (`server/internal/background/`), `clients/web/src/lib/boards-realtime.ts`.
- **Client dep**: Y.js is already used (collab docs); reuse the same version.

## 13. Dependencies & Sequencing

- Must ship after: VC.1, VC.2 (posts to sync); ideally alongside VC.3 (positions to sync).
- Must ship before: nothing hard-depends on it, but it dramatically improves VC.3/VC.5 UX.
- Shared infra: WebSocket support (already in `server.go`), background job runner.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Single-process rooms don't fan out across instances | M | H | Document + implement Postgres LISTEN/NOTIFY (or Redis) pub/sub bridge before multi-instance WS scale-out |
| Unbounded `board_updates` growth | H | M | Snapshot + compaction job (as collab docs) |
| CRDT ↔ DB divergence | M | H | Single reconciler owns arrangement; CRDT authoritative for positions; tests assert convergence |
| Awareness identity spoofing | M | M | Server stamps identity from JWT; never trust client-claimed userId for authz |
| Message flooding by one peer | M | M | Size/rate caps; throttle/close offender |

## 15. Rollout Plan

- **Flag**: gated by `visual_boards_enabled`; a `boards_realtime` sub-flag lets us ship VC.2/VC.3 with
  refetch-on-focus first, then flip realtime on.
- **Sequencing**: migration `381` → deploy relay behind sub-flag → dogfood in a live class → enable.
- **Rollback**: disable the realtime sub-flag; boards fall back to REST refetch; no data loss (DB is the
  durable store, updates table retained).

## 16. Test Plan

- **Unit** — varint/sync encoders (shared package); reconciler folding; compaction correctness.
- **Integration** — two-client relay (add/move/edit/delete converge); reconnect replay; auth handshake
  failures; oversized-message handling.
- **End-to-end** — Playwright multi-context: user A adds/moves, user B sees it live; presence avatars; flaky
  reconnect.
- **Security** — token/enrollment enforcement; spoofed awareness cannot escalate; origin checks.
- **Accessibility** — live-region announcements; keyboard operation unaffected by cursors.
- **Performance / load** — 40-peer room fan-out latency; connect+replay time pre/post compaction.
- **Manual** — network partition and heal; two tabs same user.

## 17. Documentation & Training

- End-user: "Boards update live — see who's here."
- API reference: the board WS handshake/protocol.
- Runbook: rooms/updates/snapshots operations, compaction job, and the multi-instance fan-out plan.

## 18. Open Questions

1. Ship the horizontal fan-out (LISTEN/NOTIFY vs Redis) in v1, or accept single-instance WS until scale
   demands it? (Recommendation: single-instance for GA; land the bridge before multi-node WS.)
2. Should comments/reactions (VC.5) ride this relay or use their own lighter channel? (Recommendation:
   reactions via awareness/light events; comments via REST + this relay's notify.)
3. Snapshot/compaction cadence — time-based, update-count-based, or on-idle? (Recommendation: update-count
   threshold with an idle flush.)

## 19. References

- Existing files: `server/internal/httpserver/collab_docs_ws.go`,
  `server/internal/repos/collabdocs/collabdocs.go` (updates/snapshots pattern),
  `server/internal/httpserver/server.go` (WS setup), `enrollment.UserHasAccess`.
- Related plans: [VC.2](VC.2-posts-and-content-types.md), [VC.3](VC.3-board-layouts-and-arrangement.md),
  [VC.5](VC.5-reactions-comments-assessment.md); Collaborative Documents (plan 6.5).
