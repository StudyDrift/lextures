# VC.5 — Reactions, Comments & Assessment

> Implementation plan. Source: net-new capability (real-time visual collaboration boards). Landscape: [visual-collaboration/README](../../plan/visual-collaboration/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.5 |
| **Section** | Visual Collaboration Boards |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Collaboration squad |
| **Depends on** | VC.2 |
| **Unblocks** | (enhances VC.9 export) |

---

## 1. Problem Statement

Boards become interactive when peers can respond to each other's cards — a like, a vote, a star rating, a
short comment — and instructors can turn that response layer into lightweight assessment (grade a card,
tally votes). Without a reaction/comment layer a board is a bulletin board, not a collaboration space. VC.5
adds configurable **reactions**, threaded **comments**, and optional **grading** per card.

## 2. Goals

- Let a board owner choose a **reaction mode**: none, like (heart/thumb), upvote (count), star rating
  (1–5), or grade (points/letter), applied uniformly to that board's cards.
- Record one reaction per user per card (idempotent toggle) and show aggregate + personal state.
- Provide threaded **comments** on a card with edit/delete and instructor moderation.
- Optionally push a card's grade into the existing **gradebook** when the board is tied to an assignment.
- Keep everything real-time-friendly (reactions/counts update live via VC.4 where available).

## 3. Non-Goals

- The reaction/comment **transport** internals (VC.4 provides live updates; VC.5 works with polling if VC.4
  is absent).
- Peer-review rubrics and calibrated grading (that's the peer-review feature, plan 3.15) — VC.5 grading is a
  simple per-card score.
- Moderation/approval of posts themselves (VC.7); VC.5 comment moderation is limited to hide/delete.
- Anonymous-attribution rules (VC.6 owns identity/anonymity).

## 4. Personas & User Stories

- **As a student**, I want to "like" a classmate's idea so good ones rise to the top.
- **As an instructor**, I want to enable star ratings so the class can rank submissions.
- **As an instructor**, I want to run a vote and see the tally live.
- **As a student**, I want to comment on a card to ask a follow-up question.
- **As an instructor**, I want to grade each student's card and (optionally) send the score to the gradebook.
- **As an instructor**, I want to hide an off-topic comment.

## 5. Functional Requirements

- **FR-1.** Each board MUST have a `reaction_mode ∈ {none, like, vote, star, grade}` (default `none`),
  settable by `item:create`.
- **FR-2.** In `like`/`vote` modes, a user MUST be able to toggle exactly one reaction per card; the card
  MUST show the aggregate count and whether the current user reacted.
- **FR-3.** In `star` mode, a user MUST be able to set a 1–5 rating per card (updatable); the card MUST show
  the average and count.
- **FR-4.** In `grade` mode, only users with a grading permission (`item:create` / grader role) MAY set a
  numeric/letter score per card; students see their own card's grade, not others'.
- **FR-5.** The system MUST support threaded comments per card: `POST/GET/PATCH/DELETE` with author-or-manager
  authorization; comment bodies MUST be sanitized (same policy as post bodies, VC.2 FR-12).
- **FR-6.** Comment authors MUST be able to edit/delete their own comments; `item:create` MAY hide/delete any
  comment (soft-hide preserves the record for audit/FERPA).
- **FR-7.** When a board is linked to an assignment (optional `assignment_id` on the board), `grade` mode
  MUST offer "send to gradebook", writing through the existing grading path; unlinking or regrading MUST be
  supported.
- **FR-8.** Reaction and comment counts MUST be queryable in bulk with the post list (avoid N+1); the post
  list response MUST include `{reactionCount, myReaction, avgStars, commentCount}` as applicable.
- **FR-9.** Sorting a board by "most reacted" (VC.3 FR-9) MUST use these aggregates.
- **FR-10.** All reaction/comment writes MUST respect board access and the board's posting/interaction
  permissions (VC.6); a read-only viewer MUST NOT be able to react or comment.

## 6. Non-Functional Requirements

- **Performance** — aggregates precomputed or single-query joined with the post list; react toggle p95 < 120
  ms.
- **Security** — one-reaction-per-user enforced by a unique constraint, not just UI; grades visible only to
  the owner and graders.
- **Privacy & Compliance** — comments and grades are education records; grade visibility respects FERPA
  (never expose a student's grade to peers); deletion/export via [S02](../standards/S02-data-retention-deletion-engine.md).
- **Accessibility** — reaction buttons are labelled toggle buttons with state; star rating is keyboard-
  operable; comment threads have proper heading/list semantics.
- **Scalability** — reactions table indexed by post; counts via aggregate or maintained counters.
- **Reliability** — idempotent toggle (re-clicking removes); grade writes are transactional.
- **Observability** — counters for reactions/comments/grade-sync; failed gradebook writes alert.
- **Maintainability** — reuse the existing gradebook write API; do not fork grading logic.
- **Internationalization** — reaction labels and comment UI externalised; grade formats locale-aware.
- **Backward compatibility** — additive; boards default `reaction_mode = none`.

## 7. Acceptance Criteria

- **AC-1.** *Given* like mode, *when* a student clicks like then clicks again, *then* the count increments
  then decrements and their personal state toggles.
- **AC-2.** *Given* star mode, *when* three students rate a card 4, 5, 3, *then* the card shows avg 4.0 (n=3).
- **AC-3.** *Given* grade mode, *when* a student views the board, *then* they see their own card's grade but
  no one else's.
- **AC-4.** *Given* a linked assignment and a graded card, *when* the instructor clicks "send to gradebook",
  *then* the score appears in the gradebook for that student.
- **AC-5.** *Given* a comment thread, *when* a student replies, *then* the reply nests under the parent and
  the card's comment count increments.
- **AC-6.** *Given* an off-topic comment, *when* the instructor hides it, *then* it disappears for students
  but remains retrievable for audit.
- **AC-7.** *Given* a read-only viewer (VC.6), *when* they try to react/comment, *then* the API returns
  `403`.
- **AC-8.** *Given* "sort by most reacted", *when* applied, *then* cards order by aggregate reaction/score.

## 8. Data Model

Migration `382_board_reactions_comments.sql`:

```sql
ALTER TABLE board.boards
  ADD COLUMN reaction_mode TEXT NOT NULL DEFAULT 'none',   -- none|like|vote|star|grade
  ADD COLUMN assignment_id UUID;                            -- optional link for grade sync

CREATE TABLE board.post_reactions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id    UUID NOT NULL REFERENCES board.posts (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    kind       TEXT NOT NULL,               -- like|vote|star|grade
    value      DOUBLE PRECISION,            -- stars 1-5, grade points; null for like/vote
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (post_id, user_id, kind)         -- one reaction of a kind per user per card
);
CREATE INDEX idx_reactions_post ON board.post_reactions (post_id);

CREATE TABLE board.post_comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id    UUID NOT NULL REFERENCES board.posts (id) ON DELETE CASCADE,
    parent_id  UUID REFERENCES board.post_comments (id) ON DELETE CASCADE,
    author_id  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    body       JSONB NOT NULL,              -- sanitized rich text
    hidden     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_comments_post ON board.post_comments (post_id);
```

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| PUT | `/boards/{id}/posts/{postId}/reaction` — `{kind, value?}` (idempotent set/toggle) | interaction-permitted member; `grade` requires grader |
| DELETE | `/boards/{id}/posts/{postId}/reaction` | reacting user |
| GET | `/boards/{id}/posts/{postId}/comments` | course access |
| POST | `/boards/{id}/posts/{postId}/comments` — `{body, parentId?}` | interaction-permitted member |
| PATCH | `/boards/{id}/posts/{postId}/comments/{commentId}` | author or `item:create` |
| DELETE | `/boards/{id}/posts/{postId}/comments/{commentId}` | author or `item:create` |
| POST | `/boards/{id}/posts/{postId}/grade-sync` | grader (requires linked `assignment_id`) |

- Post-list responses (VC.2) extended with aggregates. Rate-limit comment creation per user.

## 10. UI / UX

- **Reaction control** on each card: heart/thumb (like), up-arrow + count (vote), 5-star widget (star), or a
  grade chip (grade). Shows personal + aggregate state.
- **Comment thread**: expandable panel under/behind a card with nested replies, edit/delete, and a manager
  "hide" action.
- **Board settings**: reaction-mode selector; "link to assignment" + "send grades to gradebook".
- **States**: no-reactions empty, optimistic toggle, hidden-comment placeholder for managers.
- **Mobile**: comments open in a sheet; reactions are large tap targets.
- **Accessibility**: reaction buttons expose `aria-pressed`; star rating is a labelled rad/slider; threads
  use list/heading semantics; live count updates announced politely.
- **Copy & i18n**: `boards.react.*`, `boards.comment.*` keys.

## 11. AI / ML Considerations

Optional (flagged, off by default): comment toxicity pre-screen reusing the profanity/safety path from VC.7;
AI "summarize the top-voted cards" for instructors. Both degrade gracefully to manual. Out of scope for GA.

## 12. Integration Points

- **Reuse**: existing gradebook write path (grade sync), rich-text sanitizer (VC.2), VC.4 relay for live
  count updates.
- **New**: `server/internal/repos/board/reactions.go`, `board/comments.go`,
  `server/internal/httpserver/board_engagement_http.go`, `clients/web/src/components/boards/reactions/*`.

## 13. Dependencies & Sequencing

- Must ship after: VC.2 (posts). Enhanced by VC.4 (live) and VC.6 (interaction permissions).
- Must ship before: nothing hard; improves VC.9 export (include counts/comments).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Grade leakage to peers | L | H | Grade rows filtered server-side to owner+graders; explicit tests |
| N+1 on counts | M | M | Join aggregates into the post-list query / maintained counters |
| Comment spam | M | M | Per-user rate limit; VC.7 moderation tools |
| Double-reaction race | L | L | `UNIQUE (post_id,user_id,kind)` + upsert |

## 15. Rollout Plan

- **Flag**: gated by `visual_boards_enabled`; grade-sync additionally requires the gradebook feature.
- **Sequencing**: migration `382` → ship like/vote/star → add grade + gradebook sync.
- **Rollback**: set `reaction_mode = none`; comments hideable; data retained.

## 16. Test Plan

- **Unit** — toggle idempotency; star average; grade visibility filter; comment nesting.
- **Integration** — reaction uniqueness; grade-sync writes to gradebook; hide preserves audit row.
- **End-to-end** — Playwright: like toggle; star rating; vote tally; comment thread; instructor grade →
  gradebook; read-only viewer blocked.
- **Security** — grade visibility; interaction-permission enforcement; XSS in comments.
- **Accessibility** — reaction/star keyboard + SR; thread semantics.
- **Performance** — post-list-with-aggregates query plan.
- **Manual** — live count updates with two clients.

## 17. Documentation & Training

- End-user: reacting and commenting on cards.
- Instructor: choosing a reaction mode; grading cards; sending grades to the gradebook.
- API reference: reaction/comment/grade-sync endpoints.

## 18. Open Questions

1. Multiple reaction kinds at once (emoji palette) or a single mode per board? (Recommendation: single mode
   per board for v1; emoji palette as fast-follow.)
2. Should students be able to see the vote tally live, or only after the instructor reveals it? (Add a
   "hide results until revealed" board setting.)
3. Grade scale source — reuse the assignment's scheme when linked, else points? (Recommendation: inherit
   from linked assignment; points otherwise.)

## 19. References

- Existing files: gradebook write path (grading handlers/repos), rich-text sanitizer from
  [VC.2](VC.2-posts-and-content-types.md).
- Related plans: [VC.4](VC.4-realtime-collaboration-and-presence.md),
  [VC.6](../../plan/visual-collaboration/VC.6-sharing-access-contributors.md),
  [VC.7](../../plan/visual-collaboration/VC.7-moderation-safety-governance.md),
  [VC.9](../../plan/visual-collaboration/VC.9-embedding-export-presentation.md); peer review (plan 3.15).
