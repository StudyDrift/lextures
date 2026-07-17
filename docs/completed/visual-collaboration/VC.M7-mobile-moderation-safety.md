# VC.M7 — Mobile Boards: Moderation, Safety & Governance Surfaces

> Implementation plan. Source: mobile parity for board moderation/safety. Landscape: [visual-collaboration/README](../../plan/visual-collaboration/README.md). Mirrors web [VC.7](VC.7-moderation-safety-governance.md); consumes the moderation REST endpoints and honors lock/freeze/approval state (including over the VC.M4 WebSocket).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.M7 |
| **Section** | Visual Collaboration Boards — Mobile |
| **Severity** | BLOCKER (for K-12 GA / external sharing) |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile squad + Trust & Safety |
| **Depends on** | VC.M2, VC.M6 |
| **Unblocks** | mobile external-sharing UI (VC.M6 public/link) |

---

## 1. Problem Statement

A class wall on a phone is a safety surface: it can carry bullying, profanity, inappropriate images, or spam,
and a student is far more likely to be *on* their phone when something goes wrong. K-12 buyers will not enable
Boards without teacher control, and the mobile apps must not expose posting or external-share UI that the
web-side governance (approval, filtering, report/flag, hide/lock/freeze, AV-scan) doesn't back. VC.M7 brings
the student-facing **report** flow and safety states, and the instructor-facing **moderation controls +
queue**, to iOS and Android.

## 2. Goals

- Let any member **report** a card or comment with a reason from the phone.
- Honor **approval mode**: a held ("pending approval") post is invisible to peers and shows the author their
  own pending badge; managers can approve/reject.
- Honor **lock** (read-only) and **freeze** (temporary) board states in the mobile UI — disabling compose/
  react/arrange — and gracefully handle the server's `board_locked_or_frozen` rejection (including the WS
  text frame from VC.M4).
- Render safety states: "pending approval", "removed by instructor", "this file was blocked" (AV), filtered
  content messaging.
- Give managers a **moderation queue** (pending / reported / flagged) with approve / reject / hide / remove /
  dismiss, and board controls (moderation mode, filter action, lock, freeze) — enforcing the org policy floor
  the server signals.

## 3. Non-Goals

- The access/permission model (VC.M6) — VC.M7 governs content, not who may act.
- The comment feature itself (VC.M5) — VC.M7 adds moderation over comments.
- Client-side profanity filtering — filtering is server-side; the mobile client only surfaces the result
  (blocked message / flagged state).
- AI toxicity/image-safety pre-screening (server-side future; mobile just reflects outcomes).
- Platform-wide abuse analytics (web VC.10 / admin).

## 4. Personas & User Stories

- **As a student**, I want to report a hurtful card so a teacher reviews it.
- **As a student**, I want to see that my post is "pending approval" so I know it's not lost.
- **As an instructor**, I want to approve student posts from my phone before the class sees them.
- **As an instructor**, I want to lock the board after an activity, or freeze posting while I give
  instructions, from my phone.
- **As an instructor**, I want to hide/remove an inappropriate card and see the report that flagged it.

## 5. Functional Requirements

- **FR-1.** Any member MUST be able to report a post or comment via `POST …/boards/{id}/reports {postId?|
  commentId?, reason?}` from a card/comment overflow action; the app confirms submission and the reported item
  enters the server queue.
- **FR-2.** In `approval` mode, a non-manager's new post created via mobile MUST reflect `status = pending`:
  the author sees a "Pending approval" badge on their own card, and peers do not see it (the server filters the
  post list — the client renders whatever the server returns).
- **FR-3.** Managers MUST get a **moderation queue** (`GET …/boards/{id}/moderation/queue`) with Pending /
  Reported / Flagged sections and actions: approve/reject (`…/posts/{id}/approve|reject`), hide/remove
  (`…/posts/{id}/hide|remove`, also comments), and resolve report (`…/reports/{id}/resolve {action}`).
- **FR-4.** Managers MUST be able to set `moderationMode`, `filterAction`, `locked`, and `frozenUntil` via
  `PATCH …/boards/{id}`; where the server signals an org floor (COPPA/minor), those controls MUST render
  disabled/locked and the client MUST NOT attempt to loosen them.
- **FR-5.** When a board is **locked** or **frozen**, the mobile UI MUST disable compose/react/arrange for
  non-managers and show the state; a write attempt MUST be prevented client-side and the server's rejection —
  REST error or the VC.M4 WS `{"error":"board_locked_or_frozen"}` text frame — MUST surface a clear,
  non-blocking notice.
- **FR-6.** The client MUST render safety states honestly: `scanStatus == 'pending'` → "checking file";
  `'blocked'` → "this file was blocked" and never fetch bytes (shared with VC.M2); hidden/removed content →
  "removed by instructor" placeholder for managers, invisible to peers; filtered-blocked submission → a
  generic "couldn't post" message (managers see specifics via the queue).
- **FR-7.** A filter **block** on submit MUST show the author a generic message and create no post; a filter
  **flag** MUST create the post but surface it in the manager queue (server behaviour; client reflects).
- **FR-8.** Report actions MUST be rate-limited (server-enforced); the client SHOULD debounce repeat reports on
  the same target and reflect an already-reported state.
- **FR-9.** The mobile external-share UI (VC.M6 public/link) MUST remain hidden until VC.M7 ships, matching the
  web sequencing (never expose link-sharing without moderation controls present).

## 6. Non-Functional Requirements

- **Performance** — queue lists paginate; lock/freeze checks are O(1) state reads; report submit is a single
  small write.
- **Security** — moderation actions require the manager capability (`canManage`, server-enforced); the client
  never bypasses lock/freeze/approval; hidden/rejected content never rendered to peers.
- **Privacy & Compliance** — rejected/removed content is retained server-side (not purged) for incident/FERPA
  needs (S02/S03); reporter identity handled per policy; minors' org floor honored (S08); anonymous-board
  authorship never revealed in the queue to non-managers.
- **Accessibility** — report dialog focus-trapped; queue actions labelled; lock/freeze state changes
  announced; safety-state placeholders have text, not color-only.
- **Reliability** — lock/freeze is authoritative server-side and immediate on WS; the client re-reads state on
  the VC.M4 notify.
- **Observability** — reuse app error logging; no client moderation telemetry beyond that.
- **Internationalization** — moderation/report copy externalised; RTL-safe; filter word lists are server-side.
- **Backward compatibility** — additive; boards default `open`, unlocked, unfrozen.

## 7. Acceptance Criteria

- **AC-1.** *Given* any member, *when* they report a card with a reason, *then* it is submitted and appears in
  the manager queue.
- **AC-2.** *Given* `approval` mode, *when* a student posts from mobile, *then* peers don't see it, the author
  sees a "Pending approval" badge, and the manager can approve/reject from the queue.
- **AC-3.** *Given* a locked board, *when* a non-manager tries to post/react/move (REST or via the VC.M4 WS),
  *then* the write is prevented client-side and any server rejection surfaces a clear notice.
- **AC-4.** *Given* a frozen board with a timer, *when* the timer elapses, *then* posting re-enables (the
  client re-reads state and updates controls).
- **AC-5.** *Given* a blocked attachment, *when* the card renders, *then* the file is never fetched and a
  "blocked" state shows.
- **AC-6.** *Given* a minors' course org floor, *when* a manager opens board settings, *then* approval + block
  filter render enforced and cannot be loosened.
- **AC-7.** *Given* a manager hides/removes a card, *then* it disappears for peers and is retrievable in the
  queue/audit (server-side).
- **AC-8.** *Given* both platforms, *when* CI runs, *then* iOS build + Android compile are green.

## 8. Data Model

No server schema change — VC.7's `board.reports`, `board.moderation_log`, the board moderation columns
(`moderation_mode`, `filter_action`, `locked`, `frozen_until`) and post `status`/`hidden` already exist. The
client extends its `BoardPost`/`Board` models with `status`, `hidden`, moderation state, and adds `Report` /
`QueueItem` models.

## 9. API Surface

No new endpoints. Mobile consumes web VC.7's routes:

| Verb | Path | Auth |
|---|---|---|
| PATCH | `/boards/{id}` — `moderationMode`, `filterAction`, `locked`, `frozenUntil` | `item:create` (respect org floor) |
| GET | `/boards/{id}/moderation/queue` | `item:create` |
| POST | `/boards/{id}/posts/{postId}/approve` \| `/reject` | `item:create` |
| POST | `/boards/{id}/posts/{postId}/hide` \| `/remove` (also `/comments/{cid}/…`) | `item:create` |
| POST | `/boards/{id}/reports` — `{postId?|commentId?, reason?}` | any member |
| POST | `/boards/{id}/reports/{reportId}/resolve` — `{action}` | `item:create` |
| GET | `/boards/{id}/moderation/log` | `item:create` (optional on mobile v1) |

The board/post responses already carry `status`, `hidden`, `locked`, `frozenUntil`, and org-floor hints —
mobile consumes them.

## 10. UI / UX

- **Report dialog** — reason picker + optional free text on any card/comment overflow; confirmation toast;
  already-reported reflected.
- **Author safety states** — "Pending approval" badge on the author's own held card; "Removed by instructor"
  placeholder; "This file was blocked" (AV); generic "couldn't post" on filter-block.
- **Moderation queue** (managers) — a list with Pending / Reported / Flagged chips (`LMSSegmentedChips`),
  content preview + reason, and approve/reject/hide/remove/dismiss actions.
- **Board controls** (managers) — lock toggle, freeze-for-N-minutes, moderation-mode + filter-action selectors
  in board settings; org-floor items rendered disabled with an explainer.
- **Lock/freeze feedback** — a banner on the board when locked/frozen; compose/react/arrange controls disabled;
  the VC.M4 `board_locked_or_frozen` frame shows a transient notice.
- **Accessibility** — dialog focus-trap; labelled queue actions; announced state changes; text placeholders.
- **Copy & i18n** — `boards.moderation.*`, `boards.report.*` keys.

## 11. AI / ML Considerations

None client-side. AI toxicity/image-safety pre-screening is a server-side future; mobile only reflects
outcomes (queue/blocked states).

## 12. Integration Points

- **Reuse**: AV `scanStatus` handling (VC.M2); VC.M4 `BoardSocket` (lock/freeze notice, re-read state); the
  server write-gate + moderation repos (unchanged); notification system for manager alerts.
- **New (iOS)**: `Core/LMS/LMSAPIBoardModeration.swift`, `Features/Boards/{ReportDialog,ModerationQueueView,
  BoardModerationSettings}.swift` → regenerate project.
- **New (Android)**: `core/lms/BoardModerationApi.kt`, `features/boards/{ReportDialog,ModerationQueueScreen,
  BoardModerationSettings}.kt`.

## 13. Dependencies & Sequencing

- Must ship after: VC.M2 (content to moderate), VC.M6 (who can act / capabilities); benefits from VC.M4
  (lock/freeze frame).
- Must ship before: exposing any VC.M6 external-share (public/link) UI on mobile, and before K-12 mobile GA.
- Shared infra: notifications; server write-gate.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Lock/freeze not enforced on the mobile write path (esp. WS) | M | H | Client disables controls **and** relies on server write-gate; handle the WS `board_locked_or_frozen` frame; tests on REST + WS |
| Hidden/rejected content leaks to peers on some mobile view | M | H | Render only what the server returns; audit list/card/comment/queue paths |
| Report spam from mobile | M | L | Server rate-limit + client debounce/already-reported state |
| Org floor loosened via mobile settings | L | H | Server enforces; client renders floor items disabled, never attempts |
| External-share UI exposed before moderation lands | L | H | Keep VC.M6 public/link UI gated on VC.M7 presence (FR-9) |

## 15. Rollout Plan

- **Flag**: gated by `visualBoardsEnabled`; org policy floor via the platform settings the server exposes.
- **Sequencing**: report flow + safety states (student-facing) → lock/freeze honoring → manager queue +
  controls → then unlock the VC.M6 external-share UI on mobile.
- **Rollback**: hide manager moderation UI (server still governs); report flow can stay; force `open`/unlocked
  server-side if needed.

## 16. Test Plan

- **Unit** — report submit; safety-state rendering (pending/removed/blocked/filtered); org-floor disabled
  rendering; lock/freeze control gating.
- **Integration** — pending post invisible to peers; hide/remove hidden across list + queue; report → queue →
  resolve; blocked attachment never fetched; WS lock/freeze frame handled.
- **End-to-end (device)** — student reports a card → instructor resolves; approval flow; lock read-only;
  freeze auto-expiry; filter-block message.
- **Security** — manager authz on moderation routes; no leakage of hidden/rejected content; anonymous
  authorship not revealed in queue to non-managers.
- **Accessibility** — report dialog + queue AT/keyboard; announced state changes.
- **Manual** — minors org floor; contribute-link post under approval mode (with VC.M6).

## 17. Documentation & Training

- Student: reporting a card from mobile; what "pending approval" means.
- Instructor: approving posts, locking/freezing, handling reports from the phone.
- Admin: org moderation floor for minors (shared with web); audit/incident runbook (S03).

## 18. Open Questions

1. Ship the full **manager queue** on mobile v1, or student **report** + safety states first and queue in a
   fast-follow? (Recommendation: report + safety states first — they are the student-safety BLOCKER; queue
   next.)
2. Show a mobile **moderation log** view, or leave audit review to web/admin? (Recommendation: web/admin for
   v1.)
3. Freeze duration presets on mobile — mirror web (5-min default, 60-min max)? (Recommendation: yes, identical.)

## 19. References

- Web plan: [VC.7](../../completed/visual-collaboration/VC.7-moderation-safety-governance.md); server
  `server/internal/httpserver/board_moderation_http.go`, `server/internal/repos/board/{moderation,reports}.go`,
  write-gate `board.CheckWriteAllowed` (used by `board_ws.go`).
- Related mobile plans: [VC.M2](VC.M2-mobile-posts-and-content.md), [VC.M4](VC.M4-mobile-realtime-and-presence.md), [VC.M6](VC.M6-mobile-sharing-access-attribution.md).
- Standards: [S02 retention](../../plan/standards/S02-data-retention-deletion-engine.md), [S03 incident response](../../plan/standards/S03-global-breach-notification-incident-response.md), [S08 children's privacy](../../plan/standards/S08-childrens-privacy-age-assurance-design-codes.md).
