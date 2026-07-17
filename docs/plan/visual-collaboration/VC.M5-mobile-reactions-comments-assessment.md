# VC.M5 — Mobile Boards: Reactions, Comments & Assessment

> Implementation plan. Source: mobile parity for board engagement. Landscape: [visual-collaboration/README](README.md). Mirrors web [VC.5](../../completed/visual-collaboration/VC.5-reactions-comments-assessment.md); reuses the reaction/comment/grade REST endpoints and rides the VC.M4 notify channel for live counts.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.M5 |
| **Section** | Visual Collaboration Boards — Mobile |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile squad |
| **Depends on** | VC.M2 (enhanced by VC.M4, VC.M6) |
| **Unblocks** | — |

---

## 1. Problem Statement

A board becomes interactive when peers respond to each other's cards — a like, a vote, a star, a short
comment — and instructors can grade a card. On mobile, tapping a heart or dropping a quick comment is
frictionless and exactly the kind of micro-interaction phones excel at. VC.M5 brings the board's
reaction/comment/grade layer to iOS and Android, respecting the board's reaction mode and FERPA-safe grade
visibility.

## 2. Goals

- Render and drive the board's **reaction mode** (`none | like | vote | star | grade`) on each card: toggle a
  like/vote, set a 1–5 star, or (graders only) set a score.
- Show aggregate + personal reaction state pulled from the post-list aggregates (`reactionCount`, `myReaction`,
  `avgStars`, `commentCount`) — no N+1.
- Provide a mobile **comment thread** per card (open in a sheet): read, post, reply, edit/delete own, and
  manager hide.
- Support **grade** mode read/write for graders and FERPA-correct visibility for students (own grade only).
- Keep counts feeling live by refetching on the VC.M4 `board.changed` notify (reason `post.updated` /
  targeted `postId`).

## 3. Non-Goals

- Reaction/comment **transport internals** (VC.M4 provides live refetch; VC.M5 works with pull-to-refresh if
  VC.M4 is absent).
- Peer-review rubrics / calibrated grading (separate feature).
- Post moderation/approval (VC.M7); VC.M5 comment moderation is limited to hide/delete.
- Anonymous-attribution rules (VC.M6 owns identity/anonymity; VC.M5 respects the resolved capability).

## 4. Personas & User Stories

- **As a student**, I want to like a classmate's idea with one tap so good ones rise.
- **As a student**, I want to rate cards with stars when my instructor turns that on.
- **As a student**, I want to comment on a card and reply in a thread from my phone.
- **As an instructor**, I want to grade each student's card on mobile and see the tally live for a vote.
- **As an instructor**, I want to hide an off-topic comment from my phone.
- **As a student**, I want to see my own card's grade but never a peer's.

## 5. Functional Requirements

- **FR-1.** Each card MUST render the control for the board's `reactionMode`: like/vote toggle + count,
  5-star widget + average, or a grade chip; `none` shows nothing.
- **FR-2.** In like/vote, a tap MUST toggle exactly one reaction per card via
  `PUT …/posts/{id}/reaction {kind}` / `DELETE …/posts/{id}/reaction`; the control MUST show `myReaction`
  and the aggregate optimistically, reconciling on response.
- **FR-3.** In star mode, setting/updating a rating MUST `PUT …/reaction {kind:'star', value:1..5}`; the card
  shows average + count.
- **FR-4.** In grade mode, only graders (server-enforced) MAY set a score; students MUST see only their own
  card's grade — the client MUST NOT render another student's grade even if a value were present.
- **FR-5.** Comments MUST load via `GET …/posts/{id}/comments`, post/reply via `POST …/comments {body,
  parentId?}`, edit/delete own via `PATCH`/`DELETE`; managers MAY hide/delete any (soft-hide preserved
  server-side).
- **FR-6.** Comment bodies MUST render through the sanitized rich-text/markdown renderer (server sanitizes on
  write; client never renders raw HTML).
- **FR-7.** The post list MUST use the bulk aggregates already returned by VC.2/VC.5 (`reactionCount`,
  `myReaction`, `avgStars`, `commentCount`) — the client MUST NOT issue per-card count requests.
- **FR-8.** "Sort by most reacted" (VC.M3 sort control) MUST use these aggregates.
- **FR-9.** All reaction/comment writes MUST respect the board's interaction capability (`canInteract` from
  VC.M6); a read-only viewer MUST get no reaction/comment controls, and any server `403` MUST be handled.
- **FR-10.** When VC.M4 is connected, a `board.changed` bump MUST refresh the affected card's counts/comment
  count; otherwise pull-to-refresh updates them.

## 6. Non-Functional Requirements

- **Performance** — reaction toggle feels instant (optimistic); comment thread opens < 300 ms; aggregates come
  with the post list (single query).
- **Security** — one-reaction-per-user is server-enforced (`UNIQUE`), the client just reflects it; grades are
  filtered server-side to owner+graders and the client renders only what it receives.
- **Privacy & Compliance** — comments and grades are education records; grade visibility respects FERPA
  (never a peer's grade); anonymous boards hide comment authorship per VC.M6; deletion/export via S02.
- **Accessibility** — reaction controls are labelled toggle buttons exposing pressed state; star rating is
  AT-operable; comment threads use list/heading semantics; live count updates announced politely.
- **Scalability** — aggregates precomputed/joined server-side; comment paging for long threads.
- **Reliability** — idempotent toggle (re-tap removes); optimistic updates roll back on error; grade writes
  transactional server-side.
- **Internationalization** — reaction labels + comment UI externalised; grade formats locale-aware; RTL-safe.
- **Backward compatibility** — additive; `reactionMode = none` boards show no controls.

## 7. Acceptance Criteria

- **AC-1.** *Given* like mode, *when* a student taps like then taps again, *then* the count +1 then −1 and the
  personal state toggles.
- **AC-2.** *Given* star mode with ratings 4,5,3, *then* the card shows avg 4.0 (n=3).
- **AC-3.** *Given* grade mode, *when* a student views the board, *then* they see only their own card's grade.
- **AC-4.** *Given* a comment thread, *when* a student replies, *then* it nests under the parent and the card's
  comment count increments.
- **AC-5.** *Given* an off-topic comment, *when* an instructor hides it, *then* it disappears for students but
  remains retrievable server-side.
- **AC-6.** *Given* a read-only viewer, *when* they open a card, *then* no reaction/comment controls appear and
  any server write returns `403`.
- **AC-7.** *Given* VC.M4 connected, *when* a peer reacts, *then* the mobile card's count refreshes without a
  manual refresh.
- **AC-8.** *Given* both platforms, *when* CI runs, *then* iOS build + Android compile are green.

## 8. Data Model

No server schema change — VC.5's `board.post_reactions` / `board.post_comments` and the board
`reaction_mode` / `assignment_id` columns already exist. Client adds `Reaction`, `Comment` models and folds
the aggregates into the existing `BoardPost` model.

## 9. API Surface

No new endpoints. Mobile consumes web VC.5's routes:

| Verb | Path | Auth |
|---|---|---|
| PUT | `/boards/{id}/posts/{postId}/reaction` — `{kind, value?}` | interaction-permitted; `grade` needs grader |
| DELETE | `/boards/{id}/posts/{postId}/reaction` | reacting user |
| GET | `/boards/{id}/posts/{postId}/comments` | course access |
| POST | `/boards/{id}/posts/{postId}/comments` — `{body, parentId?}` | interaction-permitted |
| PATCH/DELETE | `/boards/{id}/posts/{postId}/comments/{commentId}` | author or `item:create` |
| POST | `/boards/{id}/posts/{postId}/grade-sync` | grader (linked assignment) |

Grade-sync (push a card's grade to the gradebook) is **manager-only** and optional on mobile v1 (Open
Question 1).

## 10. UI / UX

- **Reaction control** on each card: heart/thumb (like), up-arrow + count (vote), a compact 5-star widget
  (star), or a grade chip (grade), with large tap targets.
- **Comment sheet**: bottom sheet with the nested thread, a composer, edit/delete on own comments, and a
  manager "hide" action; comment count on the card opens it.
- **Grade sheet** (graders): score field validated ≤ max, optional instructor comment, save; "send to
  gradebook" only when a board is linked to an assignment (if in scope).
- **States**: no-reactions empty, optimistic toggle, hidden-comment placeholder for managers.
- **Accessibility**: reaction buttons expose pressed state; star rating AT-operable; thread list/heading
  semantics; polite live-count announcements.
- **Copy & i18n**: `boards.react.*`, `boards.comment.*` keys.

## 11. AI / ML Considerations

Optional (off by default, mirrors web): comment toxicity pre-screen via the VC.7 safety path; "summarize
top-voted cards" for instructors. Both degrade to manual; out of scope for GA.

## 12. Integration Points

- **Reuse**: the sanitized rich-text renderer (VC.M2); VC.M4 `BoardSocket` for live count refresh; the
  gradebook write path server-side (grade-sync, unchanged).
- **New (iOS)**: `Core/LMS/LMSAPIBoardEngagement.swift`, `Features/Boards/{ReactionControl,CommentSheet,
  GradeSheet}.swift` → regenerate project.
- **New (Android)**: `core/lms/BoardEngagementApi.kt`, `features/boards/{ReactionControl,CommentSheet,
  GradeSheet}.kt`.

## 13. Dependencies & Sequencing

- Must ship after: VC.M2 (posts). Enhanced by VC.M4 (live counts) and VC.M6 (interaction permissions).
- Must ship before: nothing hard.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Grade leaks to a peer on the client | L | H | Render only the grade the server returns for the viewer; explicit test for student view |
| Double-reaction race on flaky taps | M | L | Server `UNIQUE` + upsert; client debounces taps |
| Long comment threads janky | M | M | Page comments; lazy render; collapse deep replies |
| Read-only viewer sees write controls | M | M | Gate controls on `canInteract`; handle `403` |

## 15. Rollout Plan

- **Flag**: gated by `visualBoardsEnabled`; grade-sync additionally requires the gradebook feature.
- **Sequencing**: like/vote/star + comments → grade read → (optional) grade write + gradebook sync.
- **Rollback**: board `reactionMode = none` hides controls; comments hideable; data retained.

## 16. Test Plan

- **Unit** — toggle idempotency; star average; grade-visibility filter (client); comment nesting.
- **Integration** — reaction uniqueness reflected; comment CRUD authz; hide preserves record.
- **End-to-end (device)** — like toggle; star; vote tally live with VC.M4; comment thread; instructor hide;
  read-only viewer blocked.
- **Security** — grade visibility; interaction-permission enforcement; sanitized-body rendering.
- **Accessibility** — reaction/star AT operation; thread semantics; live announcements.
- **Manual** — two-client live counts; long thread.

## 17. Documentation & Training

- End-user: reacting and commenting on cards from mobile.
- Instructor: choosing a reaction mode; grading a card on mobile; sending grades to the gradebook.

## 18. Open Questions

1. Ship **grade write + gradebook sync** on mobile v1, or grade **read-only** first? (Recommendation:
   read-only first; add write once the mobile grading surfaces from M6.1 are aligned.)
2. Show a live vote **tally** to students immediately, or only after the instructor reveals? (Mirror the web
   "hide results until revealed" board setting.)
3. Emoji-palette reactions vs. single mode per board? (Follow web: single mode for v1.)

## 19. References

- Web plan: [VC.5](../../completed/visual-collaboration/VC.5-reactions-comments-assessment.md); web components
  `clients/web/src/components/boards/reactions/*`.
- Related mobile plans: [VC.M2](VC.M2-mobile-posts-and-content.md), [VC.M4](VC.M4-mobile-realtime-and-presence.md), [VC.M6](VC.M6-mobile-sharing-access-attribution.md); grades [M6.1](../../completed/mobile/M6.1-grades-feedback.md).
