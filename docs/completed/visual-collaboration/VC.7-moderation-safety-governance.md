# VC.7 — Moderation, Safety & Content Governance

> Implementation plan. Source: net-new capability (real-time visual collaboration boards). Landscape: [visual-collaboration/README](../../plan/visual-collaboration/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.7 |
| **Section** | Visual Collaboration Boards |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Collaboration squad + Trust & Safety |
| **Depends on** | VC.2, VC.6 |
| **Unblocks** | (gates external sharing exposure) |

---

## 1. Problem Statement

A wall where any student can instantly post text and media to their whole class is a safety surface: it can
carry bullying, profanity, inappropriate images, or off-topic spam. K-12 buyers will not enable Boards
without teacher control, and no external share link can be exposed until governance exists. VC.7 delivers
**pre-moderation (approval queue), content filtering, reporting/flagging, teacher controls (hide/lock/
freeze), and attachment scanning** so instructors stay in control.

## 2. Goals

- Optional **approval mode**: student posts are held pending until an instructor approves them.
- **Content filtering**: a profanity/blocklist screen on post/comment text, configurable per board.
- **Report/flag** flow so any member can flag a card/comment for instructor review.
- **Teacher controls**: hide/remove any post or comment, **lock** a board (read-only), and **freeze**
  posting temporarily.
- Ensure attachment **AV scanning** (from VC.2) blocks unsafe files before they render.
- Full **audit trail** of moderation actions for FERPA/incident review.

## 3. Non-Goals

- Access/permission model (VC.6) — VC.7 governs content, not who may act.
- Reaction/comment mechanics (VC.5) — VC.7 adds moderation over comments, not the comment feature itself.
- Automated ML image classification for nudity/violence (note as optional future; v1 relies on AV scan +
  human review + report flow).
- Platform-wide abuse analytics (VC.10).

## 4. Personas & User Stories

- **As an instructor**, I want to approve student posts before the class sees them for a sensitive topic.
- **As an instructor**, I want profanity blocked automatically so I don't have to police every card.
- **As a student**, I want to report a hurtful card so a teacher reviews it.
- **As an instructor**, I want to lock a board after an activity so it becomes read-only.
- **As an instructor**, I want to freeze posting for five minutes while I give instructions.
- **As an admin**, I want an audit log of who hid/removed what, for incident response.

## 5. Functional Requirements

- **FR-1.** Each board MUST support `moderation_mode ∈ {open, approval}` (default `open`); in `approval`,
  a non-manager's new post is created with `status = pending` and is invisible to peers until a manager sets
  `status = approved` (or `rejected`).
- **FR-2.** Managers MUST have an **approval queue** listing pending posts with approve/reject actions;
  rejecting MUST retain the record (soft state) for audit, not hard-delete.
- **FR-3.** The system MUST screen post and comment text against a configurable **blocklist/profanity
  filter** on write; a match MUST either block submission or auto-flag for review per board setting
  (`filter_action ∈ {block, flag}`).
- **FR-4.** Any member MUST be able to **report** a post or comment with an optional reason; reports create a
  `board_reports` row and notify managers; a reported item MUST surface in the moderation queue.
- **FR-5.** Managers MUST be able to **hide** (soft, reversible) or **remove** (soft-delete) any post or
  comment; hidden/removed content MUST be invisible to peers but retrievable for audit.
- **FR-6.** Managers MUST be able to **lock** a board (fully read-only) and **freeze** posting (temporary,
  optional auto-expiry); locked/frozen state MUST be enforced server-side on REST and WS writes.
- **FR-7.** Attachment posts MUST respect the AV `scan_status` from VC.2: `pending` renders a placeholder,
  `blocked` never serves the file and auto-flags for review.
- **FR-8.** Every moderation action (approve/reject/hide/remove/lock/freeze/filter-hit/report-resolve) MUST
  be written to an **audit log** with actor, target, timestamp, and reason, integrating with the existing
  admin audit log where applicable.
- **FR-9.** External share links (VC.6) MUST NOT be exposable in the UI unless the board is in a state where
  moderation controls are available (i.e., VC.7 shipped); `contribute` links MUST honour `approval` mode and
  content filtering for anonymous contributors too.
- **FR-10.** Notifications MUST reach managers on new reports and on pending posts (reuse the existing
  notification system), respecting user notification preferences.
- **FR-11.** COPPA/minor contexts MUST be able to force `approval` mode and `filter_action = block` as a
  policy default that instructors cannot loosen below the org floor.

## 6. Non-Functional Requirements

- **Performance** — filter check adds < 20 ms to a write; queue lists paginate; lock/freeze checks are O(1)
  board-state reads.
- **Security** — moderation actions require manager authz; audit entries are tamper-evident (append-only);
  hidden content never leaks via any read path/export/WS.
- **Privacy & Compliance** — retain rejected/removed content per retention policy (not immediate purge) for
  incident/FERPA needs; align to [S02 retention](../../plan/standards/S02-data-retention-deletion-engine.md),
  [S03 breach/incident](../../plan/standards/S03-global-breach-notification-incident-response.md), and
  [S08 children's privacy](../../plan/standards/S08-childrens-privacy-age-assurance-design-codes.md).
- **Accessibility** — queue, report dialog, and lock/freeze controls are fully accessible; status changes
  announced.
- **Scalability** — reports/audit indexed by board; queue queries bounded.
- **Reliability** — lock/freeze is authoritative and immediate on WS (VC.4) and REST.
- **Observability** — counters for filter hits, reports, approvals/rejections, hides/removes; alert on
  report spikes.
- **Maintainability** — one server-side gate function checks (lock, freeze, moderation_mode, filter) for all
  write paths.
- **Internationalization** — filter supports locale word lists; moderation copy externalised.
- **Backward compatibility** — additive; boards default `open`, unlocked, unfrozen.

## 7. Acceptance Criteria

- **AC-1.** *Given* `approval` mode, *when* a student posts, *then* peers do not see it until a manager
  approves; the manager sees it in the queue.
- **AC-2.** *Given* `filter_action = block` and a profane submission, *when* submitted, *then* it is rejected
  with a clear message and no post is created.
- **AC-3.** *Given* `filter_action = flag`, *when* a flagged term is posted, *then* the post is created but
  appears in the moderation queue.
- **AC-4.** *Given* a reported card, *when* a manager opens the queue, *then* the report (with reason) is
  listed and can be resolved (dismiss / hide / remove).
- **AC-5.** *Given* a locked board, *when* any non-manager attempts to post/react/move (REST or WS), *then*
  the write is rejected.
- **AC-6.** *Given* a frozen board with a 5-minute timer, *when* the timer elapses, *then* posting resumes
  automatically.
- **AC-7.** *Given* a blocked attachment, *when* the card renders, *then* the file is never served and the
  item is auto-flagged.
- **AC-8.** *Given* any moderation action, *when* it completes, *then* an audit entry with actor/target/
  reason exists.
- **AC-9.** *Given* a minors' course, *when* an instructor opens board settings, *then* `approval` mode and
  blocking filter are enforced and cannot be turned below the org floor.

## 8. Data Model

Migration `389_board_moderation.sql` (plan originally reserved `384`; that number shipped as VC.6 access):

```sql
ALTER TABLE board.boards
  ADD COLUMN moderation_mode TEXT NOT NULL DEFAULT 'open',   -- open|approval
  ADD COLUMN filter_action   TEXT NOT NULL DEFAULT 'flag',   -- block|flag
  ADD COLUMN locked          BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN frozen_until    TIMESTAMPTZ;                     -- null = not frozen

ALTER TABLE board.posts
  ADD COLUMN status  TEXT NOT NULL DEFAULT 'approved',        -- approved|pending|rejected
  ADD COLUMN hidden  BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_posts_board_status ON board.posts (board_id, status);

CREATE TABLE board.reports (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id     UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    post_id      UUID REFERENCES board.posts (id) ON DELETE CASCADE,
    comment_id   UUID REFERENCES board.post_comments (id) ON DELETE CASCADE,
    reporter_id  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    reason       TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'open',                -- open|resolved|dismissed
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at  TIMESTAMPTZ,
    resolved_by  UUID REFERENCES "user".users (id) ON DELETE SET NULL
);
CREATE INDEX idx_reports_board_status ON board.reports (board_id, status);

CREATE TABLE board.moderation_log (
    id         BIGSERIAL PRIMARY KEY,
    board_id   UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    actor_id   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    action     TEXT NOT NULL,       -- approve|reject|hide|remove|lock|unlock|freeze|filter_hit|report_resolve
    target_type TEXT NOT NULL,      -- post|comment|board
    target_id  UUID,
    reason     TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_modlog_board ON board.moderation_log (board_id, created_at);
```

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| PATCH | `/boards/{id}` (extend) — `moderationMode`, `filterAction`, `locked`, `frozenUntil` | `item:create` (respect org floor) |
| GET | `/boards/{id}/moderation/queue` (pending + reports + filter-flags) | `item:create` |
| POST | `/boards/{id}/posts/{postId}/approve` \| `/reject` | `item:create` |
| POST | `/boards/{id}/posts/{postId}/hide` \| `/remove` (also `/comments/{cid}/…`) | `item:create` |
| POST | `/boards/{id}/reports` — `{postId?|commentId?, reason?}` | any member (incl. contribute-link) |
| POST | `/boards/{id}/reports/{reportId}/resolve` — `{action}` | `item:create` |
| GET | `/boards/{id}/moderation/log` | `item:create` |

- Rate-limit reports per user; filter check runs inside the shared write-gate for posts/comments.
- **OpenAPI**: moderation endpoints and states.

## 10. UI / UX

- **Moderation queue** (`components/boards/moderation-queue.tsx`): tabs for Pending / Reported / Flagged,
  each with approve/reject/hide/remove/dismiss and the offending content preview + reason.
- **Board controls**: lock toggle, freeze-for-N-minutes, moderation-mode + filter-action selectors in board
  settings (org floor rendered as locked/disabled where enforced).
- **Report dialog**: reason picker + free text on any card/comment.
- **Author-side states**: "Pending approval" badge on the author's own held card; "Removed by instructor"
  placeholder; "This file was blocked" state.
- **Mobile**: queue is a list; controls in an overflow menu.
- **Accessibility**: queue actions labelled; status changes announced; report dialog focus-trapped.
- **Copy & i18n**: `boards.moderation.*`, `boards.report.*` keys.

## 11. AI / ML Considerations

Optional/flagged for a later iteration: AI text toxicity and image-safety classification to pre-screen posts
before human review, using the platform's AI provider path with cost budget and PII handling per the AI
standards; must degrade to the human queue when unavailable. Not required for GA; v1 uses deterministic
blocklists + AV scan + report/queue.

## 12. Integration Points

- **Reuse**: AV-scan pipeline + `scan_status` (VC.2), notification system (manager alerts), admin audit log,
  `courseroles` manager checks, VC.4 write-gate for lock/freeze enforcement on WS, children's-privacy policy
  hook (S08).
- **New**: `server/internal/repos/board/moderation.go`, `board/reports.go`,
  `server/internal/service/boardfilter/` (blocklist), `server/internal/httpserver/board_moderation_http.go`,
  `clients/web/src/components/boards/moderation-queue.tsx`.

## 13. Dependencies & Sequencing

- Must ship after: VC.2 (content to moderate), VC.6 (who can act).
- Must ship before: exposing any VC.6 external share link in the UI, and before GA in K-12.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Approval mode not enforced on WS (VC.4) | M | H | Single write-gate consulted by REST and WS; tests on both |
| Filter false positives frustrate students | M | M | `flag` default; per-board word-list overrides; clear messaging |
| Removed content truly deleted, losing evidence | M | H | Soft-remove + retention; only retention engine purges |
| Report spam / abuse | M | L | Per-user rate limit; dedupe reports per target |
| Org floor bypass by instructor | L | H | Server enforces org floor; instructor UI cannot loosen below it |

## 15. Rollout Plan

- **Flag**: gated by `visual_boards_enabled`; org policy floor via platform settings (VC.10).
- **Sequencing**: migration `384` → ship lock/freeze/hide/remove + report/queue + filter → then unlock VC.6
  external-sharing UI.
- **Rollback**: force `moderation_mode = open`, unlock; audit/reports retained.

## 16. Test Plan

- **Unit** — filter matcher (locale word lists, evasion basics); write-gate (lock/freeze/approval); org-floor
  enforcement.
- **Integration** — pending post invisible to peers until approved; hide/remove hidden across REST/WS/export;
  report → queue → resolve; blocked attachment never served; audit entries written.
- **End-to-end** — Playwright: approval flow; profanity block; lock read-only; freeze auto-expiry; report a
  card as a student, resolve as instructor.
- **Security** — authz on all moderation routes; no leakage of hidden/rejected content; audit tamper checks.
- **Accessibility** — queue + dialogs axe/keyboard.
- **Manual** — minors course org floor; contribute-link post under approval mode.

## 17. Documentation & Training

- Instructor: turning on approval; filtering; locking/freezing; handling reports.
- Admin: org moderation floor for minors; audit log location.
- Runbook: incident response using the moderation/audit log ([S03](../../plan/standards/S03-global-breach-notification-incident-response.md)).
- API reference: moderation endpoints.

## 18. Open Questions

1. Ship AI toxicity/image-safety pre-screening in v1 or fast-follow? (Recommendation: fast-follow; ship
   deterministic + human first.)
2. Should students see *why* a post was blocked by the filter (the matched term)? (Recommendation: generic
   message to avoid gaming; managers see specifics.)
3. Default freeze duration and max? (Recommendation: 5-min default, 60-min max, manual unfreeze always
   available.)

## 19. References

- Existing files: AV-scan flag/pipeline, notification system, admin audit log, `courseroles`.
- Related plans: [VC.2](VC.2-posts-and-content-types.md), [VC.4](VC.4-realtime-collaboration-and-presence.md),
  [VC.6](VC.6-sharing-access-contributors.md),
  [S02 retention](../../plan/standards/S02-data-retention-deletion-engine.md),
  [S03 incident response](../../plan/standards/S03-global-breach-notification-incident-response.md),
  [S08 children's privacy](../../plan/standards/S08-childrens-privacy-age-assurance-design-codes.md).
