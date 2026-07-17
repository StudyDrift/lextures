# VC.M4 — Mobile Boards: Real-Time Collaboration & Presence (Same WebSocket)

> Implementation plan. Source: mobile parity for real-time boards over the **existing** relay. Landscape: [visual-collaboration/README](README.md). Mirrors web [VC.4](../../completed/visual-collaboration/VC.4-realtime-collaboration-and-presence.md) and connects to the **same** board WebSocket (`server/internal/httpserver/board_ws.go`), reusing the native `WebSocketClient` + per-screen socket pattern already shipped for feed/courses/notifications.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.M4 |
| **Section** | Visual Collaboration Boards — Mobile |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile squad + Collaboration squad |
| **Depends on** | VC.M1, VC.M2 (VC.M3 for live positions) |
| **Unblocks** | live UX for VC.M3, VC.M5 |

---

## 1. Problem Statement

The magic of a board is watching cards appear and move **as classmates add them**, live, without refreshing —
and that magic must work on the phone that most students actually hold. The server already runs a production
Y.js relay for boards (VC.4) that, crucially, also pushes a JSON **`board.changed`** text frame to every
connected peer on every REST mutation. VC.M4 connects the native apps to that **same** WebSocket so mobile
learners get live add/edit/move/delete — **without** porting a Y.js CRDT engine to Swift/Kotlin — and lays the
groundwork for presence.

## 2. Goals

- Connect the iOS and Android board detail screens to the **same** relay endpoint
  `GET /api/v1/courses/{code}/boards/{board_id}/ws` used by the web app, using the existing native
  `WebSocketClient` (JSON, `{"authToken":…}` handshake, 2 s reconnect).
- Deliver **live content updates** (new / edited / moved / deleted cards, section changes) by listening for the
  server's `board.changed` text frames and refetching the affected data — the "refetch-on-notify" model,
  identical in spirit to the shipped `FeedSocket` / `CourseStructureSocket`.
- Safely **ignore the binary Y.js frames** (replay snapshot, sync, awareness) the relay sends, so mobile
  interoperates with web peers without implementing CRDT.
- Keep mobile mutations flowing through the existing REST endpoints (VC.M2/VC.M3), which already trigger
  `notifyBoardPeers` → `board.changed` for every other peer (web and mobile).
- Provide connection status UX (connecting / live / reconnecting / offline) and a manual-refresh fallback.
- Scope **presence** (who's here, live cursors) explicitly: a lightweight roster if achievable without CRDT,
  otherwise deferred to a follow-up (see §11/§18) — content sync is the BLOCKER, presence is the nice-to-have.

## 3. Non-Goals

- Porting Y.js / y-crdt to native (binary sync + awareness). VC.M4 is deliberately a **JSON listener**; a
  future story may add a native CRDT binding for live cursors and sub-second position streaming.
- Changing the server protocol (the `board.changed` frame and REST notify hooks already exist). One **small,
  optional** server enhancement is proposed in §18 (emit `board.changed` on CRDT-only moves) but is not
  required for correctness.
- Offline-first authoring beyond the app's existing reconnect behaviour (basic refetch on reconnect).
- Reactions/comments transport (VC.M5 rides the same notify channel but owns its events).

## 4. Personas & User Stories

- **As a student on my phone**, I want a classmate's new card to appear without me pulling to refresh.
- **As a student**, I want a card that was moved to another column on the projector to update on my phone.
- **As a student on a flaky connection**, I want the board to resync when I reconnect, with no duplicate cards.
- **As an instructor**, I want to see the board update live while I facilitate from my phone.
- **As a student**, I'd like to see who else is on the board right now (presence — stretch).

## 5. Functional Requirements

- **FR-1.** A per-screen **`BoardSocket`** (iOS `Features/Boards/BoardSocket.swift`, Android
  `core/realtime/BoardSocket.kt`) MUST connect to `/api/v1/courses/{code}/boards/{board_id}/ws` via the
  existing `WebSocketClient`, created on board-open and torn down on board-close — the exact lifecycle of
  `FeedSocket`.
- **FR-2.** The socket MUST rely on the shipped handshake: `WebSocketClient` already sends `{"authToken":…}`
  as the first text frame and reconnects after 2 s; VC.M4 MUST NOT re-implement auth.
- **FR-3.** The socket MUST parse **text/JSON** frames and act only on `{"type":"board.changed","reason":…,
  "postId":…}`, bumping a revision counter (and, when present, recording the affected `postId`). It MUST also
  handle the `{"error":"board_locked_or_frozen"}` text frame by surfacing a non-blocking notice.
- **FR-4.** The socket MUST **ignore binary frames** (the relay's Y.js replay/sync/awareness) — decoding a
  binary frame as JSON fails and MUST be a safe no-op, never a crash (the existing `try?`/`runCatching` decode
  pattern already gives this).
- **FR-5.** The board detail screen MUST observe the revision counter and **refetch** on bump: a general bump
  refetches the post list (and sections); a bump carrying a `postId` MAY refetch just that post as an
  optimization. Refetch MUST be de-duplicated/debounced to coalesce bursts.
- **FR-6.** Mobile mutations (create/edit/delete/arrange from VC.M2/VC.M3) MUST continue to use REST; because
  the server calls `notifyBoardPeers` on those routes, **no extra publish step** is needed for peers to see
  mobile changes.
- **FR-7.** On (re)connect, the screen MUST perform a full refetch so a reconnecting client reconstructs
  current state with **no duplicates** (REST is the source of truth; the client keys posts by id).
- **FR-8.** The screen MUST show connection state — **connecting**, **live**, **reconnecting**, **offline** —
  and MUST keep working read-only + manual pull-to-refresh when the socket is down.
- **FR-9.** When the board WS refuses the upgrade (realtime flag off, board archived, or no access), the app
  MUST **not** hammer reconnect: cap/stop retries for a permanent refusal and fall back to
  refetch-on-focus + pull-to-refresh (avoid the 2 s reconnect loop for a hard `404`).
- **FR-10.** Incoming remote changes MUST be announced to assistive tech via the platform live-region
  equivalent (e.g., "3 new cards"), politely, without stealing focus.
- **FR-11 (presence, conditional).** IF presence ships in this story, connected members MUST appear as a
  presence bar; live cursors are canvas-only. IF the JSON channel cannot carry presence without a native CRDT
  binding, presence MUST be split into a follow-up and this story ships content-sync only (still BLOCKER-complete).

## 6. Non-Functional Requirements

- **Performance** — a peer's change appears on mobile within ~1 refetch round-trip of the `board.changed`
  frame (target < 500 ms on a warm connection); refetch coalescing prevents thundering-herd on bursty boards.
- **Security** — token-verified, enrollment- and board-access-checked by the server on upgrade (unchanged);
  the client never trusts frame contents for authorization, only as a refetch trigger; awareness/identity
  spoofing is a server concern already handled (server stamps identity).
- **Privacy & Compliance** — presence, if shown, reveals only in-course identity already visible on the
  roster; anonymous-attribution boards (VC.M6) MUST NOT leak authorship via any live update or presence label.
- **Accessibility** — live inserts announced politely; connection-state changes are perceivable; presence
  (if any) is supplementary, never required to operate the board.
- **Scalability** — one socket per open board; refetch load bounded by coalescing; matches the existing
  per-screen socket footprint (feed already does this).
- **Reliability** — reconnect → full refetch tolerates missed frames and duplicate delivery (id-keyed);
  no client-side persistence of WS frames.
- **Battery/network** — the socket is torn down when the board screen is backgrounded/left; no background WS.
- **Internationalization** — status + announcement copy externalised; RTL-safe.
- **Backward compatibility** — additive; if realtime is disabled server-side, the app degrades to REST refetch
  with no error surface beyond the "offline" chip.

## 7. Acceptance Criteria

- **AC-1.** *Given* a web user and a mobile user on the same board, *when* the web user adds a card, *then* it
  appears on mobile within a refetch round-trip, with no manual refresh.
- **AC-2.** *Given* a peer moves a card between columns (web calls REST arrange → `board.changed`), *then* the
  mobile board reflects the new placement.
- **AC-3.** *Given* the mobile app receives the relay's binary replay frames on connect, *then* they are
  ignored with no crash and no malformed state.
- **AC-4.** *Given* a mobile client disconnects and reconnects, *when* it rejoins, *then* it refetches and
  shows the full current state with no duplicate cards.
- **AC-5.** *Given* a mobile user creates/deletes a card, *then* connected web and mobile peers see the change
  (proving mobile REST mutations reach peers via `notifyBoardPeers`).
- **AC-6.** *Given* the realtime flag is off / board archived, *when* the WS upgrade is refused, *then* the app
  stops retrying, shows "offline", and pull-to-refresh still works — no reconnect storm.
- **AC-7.** *Given* a burst of ten `board.changed` frames in a second, *then* the app coalesces them into a
  bounded number of refetches.
- **AC-8 (presence, if in scope).** *Given* members on the board, *when* the presence bar renders, *then* it
  shows connected members; anonymous boards show no authorship.

## 8. Data Model

No server schema or migration change — VC.4's `board.board_updates` / `board.board_snapshots` and the
`FFBoardsRealtime` flag already exist. Client state only: a `BoardSocket` revision counter + last-changed
`postId`, and (if presence ships) a transient in-memory presence roster.

## 9. API Surface

No new endpoints. Mobile uses the existing relay + REST:

| Verb | Path | Auth | Mobile use |
|---|---|---|---|
| GET (upgrade) | `/boards/{board_id}/ws` | first-frame `authToken` → JWT + enrollment + access | connect; read `board.changed` text frames; ignore binary |
| GET | `/boards/{id}/posts`, `/sections` | course access | refetch on bump / reconnect |
| (VC.M2/M3 writes) | `/posts`, `/posts/{id}`, `/arrange`, `/sections…` | author/manager | mutate via REST → server notifies peers |

`board.changed` frame (server → client): `{ "type": "board.changed", "reason": string, "postId"?: string }`.
Reasons observed: `post.created`, `post.updated`, `post.deleted`, `post.arranged`, `post.moderated`,
`section.created`, `section.updated`, `section.deleted`.

## 10. UI / UX

- **`BoardSocket`** owned by the board detail view model; exposes `connectionState` + `revision` (Observable /
  StateFlow), mirroring `FeedSocket`'s `channelsRevision`/`isConnected`.
- **Connection chip** in the board header: subtle "Live" dot when connected, "Reconnecting…" while retrying,
  "Offline — pull to refresh" when down or refused.
- **Optimistic UX**: local mutations apply immediately (VC.M2/M3) and are confirmed by the subsequent refetch;
  remote changes fade in.
- **Presence (if in scope)**: stacked member avatars in the header with an overflow count; live cursors are
  canvas-only and decorative.
- **Accessibility**: incoming-change announcements are polite; connection-state is exposed as text, not
  color-only.
- **Copy & i18n**: `boards.sync.*`, `boards.presence.*` keys.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **Reuse (iOS)**: `Core/Realtime/WebSocketClient.swift`, `Core/Config/AppConfiguration.webSocketURL`; the
  `FeedSocket` pattern as the template for `BoardSocket`.
- **Reuse (Android)**: `core/realtime/WebSocketClient.kt`, `FeedSocket.kt` pattern.
- **Server (unchanged, for reference)**: `server/internal/httpserver/board_ws.go` (`notifyBoardPeers`,
  `board.changed`), `server/internal/yrelay` (relay core). No server change required for content sync.
- **New (iOS)**: `Features/Boards/BoardSocket.swift` + board detail view-model wiring → regenerate project.
- **New (Android)**: `core/realtime/BoardSocket.kt` + board detail state wiring.

## 13. Dependencies & Sequencing

- Must ship after: VC.M1, VC.M2 (posts to refetch); ideally alongside VC.M3 (so moved positions update live).
- Must ship before: nothing hard, but it makes VC.M3/VC.M5 feel live.
- Shared infra: the existing board WS (already deployed for web).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Binary Y.js frames crash the JSON decoder | M | H | Decode with `try?`/`runCatching`; ignore non-JSON; explicit test feeding a binary frame |
| Reconnect storm on a hard-refused board (flag off) | M | M | Detect permanent refusal; cap/stop retries; fall back to refetch-on-focus (FR-9) |
| Refetch stampede on bursty boards | M | M | Coalesce/debounce refetches; per-`postId` targeted refetch |
| Web CRDT-only move not carried by `board.changed` | L | M | Web already REST-persists arrange (fires `board.changed`); backstop refetch-on-focus; optional server hook (§18) |
| Presence needs CRDT the client lacks | M | L | Ship content-sync as BLOCKER; split presence to a follow-up if needed |
| Socket kept alive in background drains battery | M | M | Tear down on background/leave (FeedSocket lifecycle) |

## 15. Rollout Plan

- **Flag**: gated by `visualBoardsEnabled` + the server `FFBoardsRealtime` sub-flag (already controls the WS).
  A client kill-switch can disable the `BoardSocket` and fall back to pure refetch if needed.
- **Sequencing**: land `BoardSocket` + refetch-on-notify → validate cross-client (web↔mobile) → enable in the
  cohort → (optional) presence follow-up.
- **Rollback**: disable the client socket (refetch-on-focus only) or the server realtime flag; no data loss —
  REST/DB is the durable store.

## 16. Test Plan

- **Unit** — `board.changed` frame parsing; binary-frame ignore; refetch coalescing; permanent-refusal
  retry-stop logic.
- **Integration** — connect → replay-binary ignored → `board.changed` → refetch; reconnect → full refetch,
  no dupes; mobile REST mutation → peer sees change.
- **End-to-end (multi-client)** — web adds/moves/deletes → mobile updates live; two mobile devices; flaky
  reconnect; flag-off refusal.
- **Security** — WS auth/enrollment enforced (server); no author leak on anonymous boards via live updates.
- **Accessibility** — polite announcements; connection-state text.
- **Performance / battery** — burst coalescing; socket torn down on background.
- **Manual** — network partition and heal; airplane-mode toggle; app backgrounding mid-session.

## 17. Documentation & Training

- End-user: "Boards update live on mobile — and what the Live/Offline chip means."
- Runbook: mobile uses the same board WS; content sync is refetch-on-`board.changed`, binary CRDT frames are
  ignored client-side; presence status.
- Update the mobile READMEs' realtime section (add `BoardSocket` to the socket list).

## 18. Open Questions

1. Ship **presence** in VC.M4 or split it out? (Recommendation: split — content sync is the BLOCKER; presence
   needs either a native CRDT/awareness binding or a new lightweight JSON presence channel.)
2. Should the server **also emit `board.changed` on CRDT-only sync writes** (byte-0 sync path in
   `board_ws.go`), so a JSON-only client never misses a web drag even if web skipped REST? (Recommendation:
   small, safe server addition — do it if we see any missed-move gaps; otherwise the existing web REST-persist
   covers it.)
3. Do we want a native **y-crdt binding** (yswift / y-crdt Kotlin) later for live cursors + sub-second
   positions? (Recommendation: evaluate after content-sync ships; only if presence/cursor demand is real.)

## 19. References

- Web plan: [VC.4](../../completed/visual-collaboration/VC.4-realtime-collaboration-and-presence.md); web hook
  `clients/web/src/lib/boards-realtime.ts` (note the `board.changed` REST-notify design).
- Server: `server/internal/httpserver/board_ws.go`, `server/internal/yrelay/*`, `notifyBoardPeers` callers in
  `board_posts_http.go` / `board_layout_http.go` / `board_moderation_http.go` / `board_links_http.go`.
- Existing mobile realtime: `clients/ios/Lextures/Core/Realtime/{WebSocketClient,RealtimeManager}.swift`,
  `clients/ios/Lextures/Features/Feed/FeedSocket.swift`,
  `clients/android/app/src/main/kotlin/com/lextures/android/core/realtime/{WebSocketClient,FeedSocket}.kt`.
- Related mobile plans: [VC.M2](VC.M2-mobile-posts-and-content.md), [VC.M3](VC.M3-mobile-layouts-and-arrangement.md), [VC.M5](VC.M5-mobile-reactions-comments-assessment.md).
