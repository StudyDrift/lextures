# CT.22 — Tool: Inline Discussion (a scoped thread anchored to this part of the page)

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](../plan/content_tools/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.22 |
| **Section** | Content Tools (CT) — tool shelf |
| **Severity** | MINOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Collaboration team |
| **Depends on** | CT.1, CT.2, CT.3, CT.8 (moderation is mandatory here) |
| **Unblocks** | Discussion where the confusion is, rather than in a forum nobody opens |

---

## 1. Problem Statement

Lextures has capable discussion forums (`course.discussion_forums` / `threads` / `posts`), and they sit
one navigation away from every moment a student actually wants to say something. The result is the
familiar dead forum: prompts answered dutifully once a week, disconnected from the material. Discussion
works best when it is *anchored* — attached to the paragraph, the diagram, the claim that provoked it —
and when the cost of contributing is one click rather than one navigation. This tool places a small,
scoped thread inline, reusing the shipped discussion data model rather than inventing a second one.

## 2. Goals

- Put a focused discussion thread inline, anchored to a section of content and to an author's prompt.
- Reuse the shipped forum/thread/post model and its moderation, notification and permission machinery.
- Keep it small on purpose: a prompt, replies, one level of nesting, upvotes — not a forum.
- Support the pedagogical patterns that make inline discussion work: post-before-you-see (commit first),
  required participation counts, and instructor endorsement of good answers.
- Ship with moderation, reporting and anonymity controls from day one, because peer-visible student text
  in a K-12 product is not a feature you add safety to later.

## 3. Non-Goals

- Replacing course forums for long-form, graded or cross-topic discussion.
- Real-time chat (the shipped live-session and messaging surfaces cover that).
- Group-restricted discussion (group spaces exist and own that).
- Threading beyond one reply level — depth is what makes forums unreadable.

## 4. Personas & User Stories

- **As an instructor**, I want a discussion prompt right under the passage it refers to so that replies
  are actually about it.
- **As an instructor**, I want students to post before they can read peers' answers so that I get thirty
  independent thoughts, not one thought echoed thirty times.
- **As an instructor**, I want to endorse a good student answer so that the class knows what good looks
  like.
- **As a student**, I want to ask about the specific sentence that confused me so that my question has
  context.
- **As a student**, I want to report an unkind reply so that it stops.
- **As a K-12 administrator**, I want peer-visible text moderated and reportable so that this feature
  passes our safeguarding review.

## 5. Functional Requirements

- **FR-1.** The author MUST configure: a prompt, whether posting is required, minimum posts/replies for
  completion, **post-before-you-see** on/off, anonymity mode (`named`, `anonymous_to_peers`), and
  whether replies are allowed.
- **FR-2.** Posts MUST be persisted in the shipped discussion tables (a thread per instance, posts as
  normal posts) — **not** in `state_json`. The tool's per-enrollment state holds only the learner's
  participation record (post ids, counts, read markers, completion).
- **FR-3.** The tool MUST create its backing thread lazily on first post, linked to the instance.
- **FR-4.** With post-before-you-see enabled, the peer posts endpoint MUST refuse to return other
  learners' posts until the caller has posted — enforced server-side, not by hiding in the UI.
- **FR-5.** Every post MUST pass the CT.8 content filter, with crisis-signal escalation, before becoming
  visible.
- **FR-6.** Students MUST be able to report a post; instructors MUST be able to hide, remove, restore and
  warn, with all actions audited (CT.8 moderation model).
- **FR-7.** `anonymous_to_peers` MUST hide author identity from peers while preserving it for
  instructors and moderation records; the UI MUST state this clearly to students before they post.
- **FR-8.** Instructors MUST be able to **endorse** a post; endorsed posts are pinned to the top with a
  clear label.
- **FR-9.** Learners MUST be able to edit their own post within a configurable window (default 5 min) and
  delete it (soft delete, visible to instructors) unless the course forbids it.
- **FR-10.** Notifications MUST reuse the shipped discussion notification preferences; a busy inline
  thread MUST NOT create a new notification firehose (digest by default).
- **FR-11.** Completion MUST be defined by the configured minimum posts/replies and reported to CT.7.
- **FR-12.** The thread MUST paginate (default 20 posts) with newest-first or oldest-first per config,
  and MUST render read-only when the instance is archived.
- **FR-13.** CT.4 reset MUST clear the learner's participation state and, per an explicit choice in the
  reset dialog, either keep or soft-delete their posts — with the consequence stated plainly.
- **FR-14.** Instructors MUST see participation at a glance (who posted, who replied, who read nothing)
  from the CT.4 roster.

## 6. Non-Functional Requirements

- **Performance** — Thread load p95 ≤ 200 ms for 20 posts; posting p95 ≤ 300 ms including filtering;
  renderer ≤ 30 KB gz.
- **Security** — Post visibility gated server-side (post-before-you-see, anonymity, moderation state);
  reporting is rate-limited; a removed post is unreachable through every read path including direct id
  access.
- **Privacy & Compliance** — Peer-visible student text triggers the full CT.8 set: filtering,
  escalation, moderation records, DSAR export of posts, retention, and (for K-12) an anonymity default
  decided by program type.
- **Accessibility** — WCAG 2.1 AA: the thread is a semantic list; the composer is a labelled textarea
  with clear submit; new posts are announced politely without stealing focus; endorsement and moderation
  states are conveyed in text; report and reply controls are keyboard-reachable with clear names.
- **Scalability** — Reuses the forum tables and their indexes; pagination prevents unbounded growth from
  reaching the client.
- **Reliability** — Draft composer text is autosaved locally; a failed post preserves the draft; the
  existing post idempotency table prevents duplicates.
- **Observability** — `lextures_content_tool_posts_total{outcome}`, filter-block rate, report rate,
  moderation-action rate, participation rate per instance.
- **Maintainability** — Thin adapter over the shipped discussion service; no second posting stack.
- **Internationalization** — Localized chrome; RTL threading layout; filter quality per language
  documented (CT.8).
- **Backward compatibility** — Additive. Threads created by this tool are marked as inline-scoped so
  they do not clutter the forum index unless the course opts to surface them.

## 7. Acceptance Criteria

- **AC-1.** *Given* post-before-you-see is on and the learner has not posted, *When* they request the
  thread, *Then* the API returns only the prompt and their own (absent) post — no peer content.
- **AC-2.** *Given* the learner posts, *Then* peer posts become available in the same response and the
  participation state records the post id.
- **AC-3.** *Given* `anonymous_to_peers`, *When* a peer views the thread, *Then* no author identity is
  present in the payload, while the instructor's view includes it.
- **AC-4.** *Given* a post containing filtered content and a `block` policy, *Then* the post is refused
  with guidance, the draft is preserved, and nothing becomes visible.
- **AC-5.** *Given* a student reports a post and the instructor removes it, *Then* it disappears for
  every viewer, direct id access returns 404, and the action is audited.
- **AC-6.** *Given* an instructor endorses a post, *Then* it pins to the top with a labelled badge for
  all viewers.
- **AC-7.** *Given* the edit window has passed, *When* the learner tries to edit, *Then* the API refuses
  and the UI offers no edit control.
- **AC-8.** *Given* completion requires 1 post and 2 replies, *Then* status becomes `completed` only when
  both are met, and CT.7 reflects it.
- **AC-9.** *Given* a CT.4 reset with "keep posts", *Then* participation state clears while posts remain;
  with "remove posts", posts are soft-deleted and the choice is recorded.
- **AC-10.** *Given* a screen-reader user posts, *Then* the new post is announced and focus returns to
  the composer, not to the top of the thread.

## 8. Data Model

**No migration.** Posts live in the shipped tables; the link is carried in existing columns plus the
tool's own state.

```ts
// configSchema
type InlineDiscussionConfig = {
  prompt: string
  postBeforeYouSee: boolean            // default true
  allowReplies: boolean                // default true
  requiredPosts: number                // default 1
  requiredReplies: number              // default 0
  anonymity: 'named' | 'anonymous_to_peers'    // default per program type (K-12: anonymous_to_peers)
  editWindowMinutes: number            // default 5
  allowDelete: boolean                 // default true
  sort: 'newest' | 'oldest'            // default 'oldest'
  pageSize: number                     // default 20
}

// stateSchema — participation only; content lives in discussion tables
type InlineDiscussionState = {
  v: 1
  threadId?: string
  myPostIds: string[]
  myReplyIds: string[]
  lastReadAt?: string
  completedAt?: string
}
```

`scoring.mode = 'none'` (optionally `manual`); `capabilities = ['state','peer_visible']`;
`maxStateBytes = 8000`.

The `peer_visible` capability is what makes CT.8's org policy able to deny this whole class of tool in
one switch — the reason capabilities exist.

## 9. API Surface

**No new routes for posting** — the tool actions wrap the shipped discussion service so permissions,
notifications and moderation are inherited:

- `POST .../actions/post` — `{text, parentPostId?, idempotencyKey}` → `{post, state}` (filters, creates
  the thread lazily, records participation).
- `POST .../actions/thread` — `{page, sort}` → `{prompt, posts, canSeePeers, total}` (enforces
  post-before-you-see, anonymity and moderation state server-side).
- `POST .../actions/report` and `POST .../actions/moderate` — delegate to CT.8's moderation routes.
- `POST .../actions/endorse` — instructor-only.

## 10. UI / UX

1. Prompt card with participation requirements ("Post once, then reply to two classmates").
2. Composer (always at the top when `postBeforeYouSee`, so it is the first thing you meet), with a note
   about anonymity mode and moderation.
3. Thread list: each post shows author (or "Anonymous classmate"), time, text, upvote, reply, report;
   endorsed posts pinned with a labelled badge; own posts show edit (within window) and delete.
4. One reply level, visually indented, collapsed beyond three replies with "show more".
5. Footer: pagination and participation progress.

**States** — *Locked (post first)*: the peer area shows an explanatory placeholder, not an empty list.
*Empty*: "Be the first to answer." *Filtered/blocked*: guidance with the draft preserved. *Removed
post*: tombstone visible to the author and instructors only. *Read-only*: archived notice.

**Mobile** — composer docked; thread scrolls; reply opens an inline composer under the parent.

**Accessibility** — thread is an ordered list with each post an article carrying an accessible name
("Post by Anonymous classmate, 2 hours ago"); new posts announced politely; report/reply/endorse are
labelled buttons; the locked state is conveyed in text, not by visual dimming alone.

**Copy & i18n** — `contentTools.tools.inlineDiscussion.*`; safeguarding-sensitive copy reviewed for a
K-12 register.

## 11. AI / ML Considerations

No generative AI in v1. The only ML involved is the shipped content filter and crisis classifier
(CT.8), which are not optional here. Reserved for later, each requiring its own disclosure: (a) a
discussion summary for the instructor ("three clusters of view emerged"); (b) a nudge that suggests an
unanswered classmate's post to reply to; (c) civility coaching on the composer before posting. (c) is
attractive for K-12 but needs careful design to avoid chilling participation, so it is explicitly not
in scope.

## 12. Integration Points

- **Internal** — `service/contenttools/tools/inlinediscussion/` (adapter),
  shipped discussion service and `server/migrations/138_discussion_forums.sql` tables,
  `service/contentfilter`, `service/notifications`, CT.8 moderation,
  `clients/web/src/components/content-tools/tools/inline-discussion/`.
- **Forums** — inline threads are tagged so the forum index can optionally list them; default hidden.
- **CT.7** — participation facets and completion.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1–CT.3 and **CT.8** (moderation is a hard prerequisite, not a follow-up).
- **Must ship before:** nothing.
- **Shared infra needed:** discussion tables, notifications, content filter.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Peer-visible text enables bullying | M | H | CT.8 filter + reporting + moderation + anonymity defaults + instructor visibility; K-12 launch gated on safeguarding review |
| Notification firehose from busy threads | H | M | Digest by default, reuse of existing preferences, per-thread mute |
| Anonymity misunderstood ("anonymous" ≠ from the teacher) | H | H | Explicit, unavoidable copy before posting and on every composer; documented in student help |
| Inline threads fragment discussion across a course | M | M | Optional forum index surfacing; guidance on when to use a forum instead |
| Post-before-you-see frustrates students who want to read first | M | L | Configurable; explanatory copy about why it is on |
| Reset ambiguity (does resetting delete my posts?) | M | M | Explicit choice in the CT.4 dialog with plain-language consequences |

## 15. Rollout Plan

- **Feature flag** — course tool allowlist + CT.8 `peer_visible` capability (org-deniable).
- **Sequencing** — adapter over the discussion service → post-before-you-see gate → renderer →
  moderation/report wiring → anonymity → endorsement → participation insights → K-12 safeguarding review
  → pilot in HE first, then K-12.
- **Dogfood** — one HE seminar course; K-12 only after the safeguarding review passes.
- **GA criteria** — moderation paths verified end-to-end; anonymity guarantees tested at the payload
  level; a11y audit passed; safeguarding sign-off for K-12.
- **Rollback** — deny `peer_visible` org-wide or remove from the allowlist; posts preserved and readable
  by instructors.

## 16. Test Plan

- **Unit** — post-before-you-see gate; anonymity projection; participation counting; edit-window
  enforcement; pagination and sort.
- **Integration** — filter block/flag paths; crisis escalation; moderation hides content across every
  read path; notification digest; reset with both post-handling choices.
- **End-to-end** — Playwright: locked → post → peers visible → reply → report → instructor removes →
  tombstone; anonymous mode from both student and instructor views.
- **Security** — direct post-id access after removal; payload inspection for identity under anonymity;
  cross-course thread access; report/post rate limits; XSS in post text.
- **Accessibility** — axe; screen-reader script for post/reply/report; announcement behaviour on new
  posts; focus after posting.
- **Performance** — 200-post thread pagination; posting latency with filtering.
- **Manual exploratory** — multilingual abusive content; long posts; rapid reply chains; RTL.

## 17. Documentation & Training

- **Instructor** — designing prompts worth answering; post-before-you-see; moderation duties; anonymity
  trade-offs; participation requirements.
- **Student** — who can see what (especially: your teacher always can), reporting, editing windows.
- **Admin** — the `peer_visible` capability; safeguarding posture; retention of posts.
- **Runbook** — escalation handling; bulk removal after an incident.

## 18. Open Questions

1. Should inline threads appear in the course forum index by default? Proposed: no — surfacing is an
   explicit course setting, to avoid drowning the forum.
2. Should K-12 default to `anonymous_to_peers`? Proposed: yes, confirmed with pilot instructors; HE
   defaults to named.
3. Should replies be limited to N per learner to prevent domination? Proposed: no hard cap; surface
   participation balance to the instructor instead.
4. Should upvotes be visible to students or instructor-only? Proposed: visible — they are a useful
   signal and low-risk compared with the post content itself.

## 19. References

- Existing files this work touches: `server/migrations/138_discussion_forums.sql`,
  the shipped discussion service and handlers, `server/internal/service/contentfilter/`,
  `server/internal/service/notifications/`, `clients/web/src/components/content-tools/`.
- External standards: WCAG 2.1 AA; K-12 safeguarding practice; COPPA constraints on peer-visible
  content for under-13 users (see CT.8 / S08).
- Related plans: [CT.8](CT.8-governance-safety-privacy-accessibility.md),
  [CT.21](CT.21-tool-class-pulse.md), [CT.13](CT.13-tool-highlight-and-annotate.md),
  [S08 children's privacy](../../plan/standards/S08-childrens-privacy-age-assurance-design-codes.md).
