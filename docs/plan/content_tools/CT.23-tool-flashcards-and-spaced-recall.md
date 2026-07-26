# CT.23 — Tool: Flashcards & Spaced Recall (an inline deck that feeds the shipped SRS)

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.23 |
| **Section** | Content Tools (CT) — tool shelf |
| **Severity** | MINOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Adaptive learning team |
| **Depends on** | CT.1, CT.2, CT.3; shipped SRS (`service/srs`, `course.srs_item_states`) |
| **Unblocks** | Retrieval practice at the point of first encounter, feeding long-term review |

---

## 1. Problem Statement

Lextures already ships spaced repetition — SM-2 scheduling, item states, review events, streaks — but a
learner can only use it if items already exist in the system, and there is no way for an author to say
"these six terms from this page belong in your review queue". So the platform's best retention mechanism
sits disconnected from the content that generates the material. Meanwhile the single most effective
thing a learner can do the moment they finish reading a page is *retrieve* what they just read — and
the page offers them nothing but a scroll bar.

## 2. Goals

- Let an author attach a small deck (3–20 cards) to a section, in seconds, from the terms on that page.
- Give the learner an immediate retrieval session inline: show prompt, self-rate recall, reveal.
- **Feed the shipped SRS** so those cards enter the learner's long-term review queue instead of dying
  in a one-off widget.
- Store per-enrollment progress (which cards seen, self-ratings, session history) as tool state, while
  the scheduling truth stays in the SRS tables.
- Make the deck useful without SRS enabled — degrade to a plain self-check.

## 3. Non-Goals

- Replacing or re-implementing the SRS scheduler (SM-2 is shipped; this tool submits reviews to it).
- Auto-grading recall (self-rating is the mechanism; it is not an assessment).
- Cloze/fill-in-the-blank generation (a separate tool in the backlog).
- Deck sharing/marketplace between courses (interesting, deferred).

## 4. Personas & User Stories

- **As a language teacher**, I want the ten new words on this page to become review cards so that
  vocabulary is spaced automatically.
- **As a biology teacher**, I want students to self-test on the terms immediately after reading so that
  the first retrieval happens while it is still encodable.
- **As a student**, I want the cards I struggled with to come back later so that I do not have to plan
  my own revision.
- **As a student**, I want a two-minute session, not a twenty-minute one, so that it fits between classes.
- **As an instructor**, I want to see which cards the class finds hardest so that I know which terms need
  more teaching.
- **As a homeschool parent**, I want review to continue across the course without me scheduling it.

## 5. Functional Requirements

- **FR-1.** The author MUST define 3–20 cards, each with a front (prompt) and back (answer), both
  supporting Markdown, inline math and an optional image with alt text.
- **FR-2.** The learner MUST be able to run an inline session: card front → self-recall → reveal back →
  self-rate (`again` / `hard` / `good` / `easy`, mapped to the shipped SM-2 quality scale).
- **FR-3.** When the course has SRS enabled, each rating MUST be submitted to the shipped SRS service,
  creating or updating the learner's `srs_item_states` row and appending an `srs_review_events` row.
- **FR-4.** When SRS is disabled for the course, the tool MUST still work as a self-check, storing
  ratings in tool state only, and MUST say so plainly ("these cards won't come back automatically").
- **FR-5.** Card identity MUST be stable across edits: a card keeps its id, so an author fixing a typo
  does not orphan a learner's scheduling history; changing the *meaning* of a card is an author action
  that creates a new card (documented, and warned about in the editor).
- **FR-6.** Session length MUST be configurable (default: all new cards plus any of this deck's cards
  currently due, capped at 20 per session).
- **FR-7.** State MUST record per card: times seen, last rating, last seen, and per session: started,
  completed, cards reviewed.
- **FR-8.** The tool MUST show the learner their own deck status: new / learning / due later, with the
  next due date when SRS is on.
- **FR-9.** The tool MUST support keyboard operation with documented shortcuts (`Space` reveal, `1–4`
  rate, `Esc` end session) and MUST announce card changes and ratings.
- **FR-10.** Completion MUST be defined as having rated every card at least once (first pass), reported
  to CT.7; ongoing SRS review is not a completion requirement.
- **FR-11.** The instructor MUST see per-card difficulty (share of `again`/`hard` first ratings) and
  deck-level completion.
- **FR-12.** CT.4 reset MUST clear tool state and, per an explicit choice in the reset dialog, either
  keep or clear the learner's SRS scheduling for this deck's cards — with the consequence stated.
- **FR-13.** The tool MUST honour `prefers-reduced-motion` (no flip animation) and MUST not rely on the
  flip metaphor for comprehension.
- **FR-14.** Cards MUST support reverse practice (back → front) when the author enables it, as separate
  SRS items so scheduling is independent.

## 6. Non-Functional Requirements

- **Performance** — Card transition ≤ 100 ms; rating round-trip p95 ≤ 200 ms (batched at session end
  where possible); renderer ≤ 22 KB gz.
- **Security** — Ratings are self-reported and non-scoring, so forgery is uninteresting; SRS writes are
  still server-side and enrollment-scoped.
- **Privacy & Compliance** — Ratings and schedules are learning records (DSAR, retention). No AI, no
  egress.
- **Accessibility** — WCAG 2.1 AA: the card is a labelled region, not a decorative flip; reveal and
  rating are buttons with clear names; card changes announced politely; shortcuts documented in-tool and
  never the only path; images require alt text.
- **Scalability** — Small state; SRS writes are one row per review, matching the shipped feature's
  existing load profile.
- **Reliability** — Ratings queue offline (CT.3 outbox) and replay; a failed SRS write never blocks the
  session or loses the rating in tool state.
- **Observability** — `lextures_content_tool_card_reviews_total{rating}`, session completion rate,
  SRS-submission failure rate, per-card difficulty gauge.
- **Maintainability** — Thin adapter over `service/srs`; no scheduling logic in the tool.
- **Internationalization** — Card content is author language; UI localized; RTL supported; language pair
  hint available for vocabulary decks so screen readers announce the right language
  (`lang` attribute per side).
- **Backward compatibility** — Additive; existing SRS items unaffected.

## 7. Acceptance Criteria

- **AC-1.** *Given* a deck of six cards, *When* the learner runs a session, *Then* each card is shown,
  revealed on request, rated, and the state records six ratings.
- **AC-2.** *Given* SRS is enabled, *When* a card is rated `again`, *Then* an `srs_review_events` row is
  written and the item's next due date is computed by the shipped scheduler (not by this tool).
- **AC-3.** *Given* SRS is disabled, *Then* no SRS rows are written, the session still completes, and the
  UI states that cards will not return automatically.
- **AC-4.** *Given* the author fixes a typo on a card, *Then* the card id is unchanged and the learner's
  scheduling history for it is preserved.
- **AC-5.** *Given* reverse practice is enabled, *Then* forward and reverse are scheduled as independent
  SRS items.
- **AC-6.** *Given* the learner is offline, *When* they rate cards and reconnect, *Then* ratings replay
  in order and SRS state matches.
- **AC-7.** *Given* 25 learners have used the deck, *When* the instructor opens insights, *Then* per-card
  difficulty matches first-rating data.
- **AC-8.** *Given* keyboard-only operation, *Then* the full session is completable with the documented
  shortcuts and with visible buttons, and each card change is announced.
- **AC-9.** *Given* a CT.4 reset with "clear scheduling", *Then* both tool state and the learner's SRS
  items for this deck are cleared; with "keep scheduling", only tool state clears.
- **AC-10.** *Given* `prefers-reduced-motion`, *Then* no flip animation plays and the reveal is instant.

## 8. Data Model

**No migration.** Scheduling lives in the shipped `course.srs_item_states` / `srs_review_events`;
the tool contributes item identity and stores its own session record.

```ts
// configSchema
type FlashcardsConfig = {
  title?: string
  cards: Array<{
    id: string                          // stable; never regenerated on edit
    front: string
    back: string
    frontLang?: string
    backLang?: string
    imageUrl?: string
    imageAlt?: string
    hint?: string
  }>
  reversePractice: boolean              // default false
  sessionCap: number                    // default 20
  shuffle: boolean                      // default true
  requireFirstPass: boolean             // default true (defines completion)
}

// stateSchema — progress and sessions; scheduling truth is in the SRS tables
type FlashcardsState = {
  v: 1
  cards: Record<string, {
    seen: number
    lastRating?: 'again' | 'hard' | 'good' | 'easy'
    lastSeenAt?: string
    firstRating?: 'again' | 'hard' | 'good' | 'easy'
  }>
  sessions: Array<{ startedAt: string; endedAt?: string; reviewed: number }>
  firstPassCompletedAt?: string
}
```

`scoring.mode = 'none'`; `capabilities = ['state']`; `maxStateBytes = 16000`;
conflict policy `merge` (per-card counters merge by max/last-write).

## 9. API Surface

**No new routes.**

- `POST .../actions/rate` — `{cardId, rating, side, idempotencyKey}` → `{state, nextDueAt?}`; submits to
  `service/srs` when enabled.
- `POST .../actions/startSession` / `endSession` — session bookkeeping and card selection (due + new,
  capped).
- `PUT .../state` — offline-replayed batches.
- Insights via CT.7 facets `cardId`, `firstRating`.

## 10. UI / UX

1. Deck header: title, card count, status chips ("4 new · 2 due today"), **Start session**.
2. Card view: front content centred, **Show answer** button (and `Space`), optional hint link.
3. After reveal: back content plus four rating buttons with plain labels
   ("Again — I didn't know it", "Hard", "Good", "Easy") and their shortcut keys shown.
4. Session end: a summary ("6 reviewed · 2 to revisit soon") and, when SRS is on, the next due date.
5. Persistent footer note when SRS is off, explaining that cards will not return automatically.

**States** — *Not started*, *In session*, *Revealed*, *Session complete*, *All caught up (nothing due)*,
*Read-only*, *Offline (ratings queued)*, *Error*.

**Mobile** — full-width card, large rating buttons in a 2×2 grid, swipe optional but never required.

**Accessibility** — the card is a labelled region whose content changes are announced ("Card 3 of 6");
reveal and rating are real buttons with descriptive names (not just "1"); `lang` set per side for
vocabulary decks; shortcuts documented in an in-tool help affordance; no flip-dependent meaning.

**Copy & i18n** — `contentTools.tools.flashcards.*`.

**Authoring** — custom editor with a **bulk paste** mode (`front — back` per line, or TSV/CSV paste from
a spreadsheet), because deck authoring is entirely about speed; plus per-card editing, reverse toggle,
and an explicit warning when editing a card that learners already have scheduled.

## 11. AI / ML Considerations

None in v1. Reserved: **generate a deck from this section** via `ui.aiAssist` — extracting key terms and
definitions from the section text for author review. This is the highest-value assist for this tool
(deck authoring is pure typing), and it is deliberately deferred to keep v1 available under every AI
policy; the shipped SRS scheduling itself is algorithmic (SM-2), not ML, and stays that way.

## 12. Integration Points

- **Internal** — `service/contenttools/tools/flashcards/` (adapter), `service/srs` and
  `service/srsscheduler` (unchanged), `server/migrations/091_spaced_repetition.sql` tables,
  course SRS feature flag (`srsEnabled` in course features),
  `clients/web/src/components/content-tools/tools/flashcards/`.
- **Existing SRS surfaces** — cards created here appear in the learner's normal review queue alongside
  other SRS items; the queue shows their source page.
- **CT.7** — per-card difficulty facets and first-pass completion.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1–CT.3.
- **Must ship before:** nothing.
- **Shared infra needed:** shipped SRS.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Card edits orphan scheduling history | M | M | Stable ids, editor warning when learners have history, documented "new meaning = new card" rule |
| Learners rate everything "easy" to finish fast | H | L | It is self-directed practice; instructor sees first-rating difficulty; framing copy explains the cost of lying to yourself |
| Decks proliferate and flood the review queue | M | M | Session caps, queue shows source, instructor guidance on deck size (3–20 cards) |
| SRS disabled makes the tool feel pointless | M | M | Clear messaging, still useful as a self-check, prompt to enable SRS shown to instructors |
| Flip animation confuses or nauseates | L | M | Reduced-motion honoured; flip is decorative only |
| Bulk paste mangles content with dashes in it | M | L | Configurable delimiter, TSV/CSV paths, preview before import |

## 15. Rollout Plan

- **Feature flag** — course tool allowlist; behaviour adapts to the existing course `srsEnabled` flag.
- **Sequencing** — SRS adapter → renderer + session logic → bulk-paste authoring → insights → pilot in a
  language course and a biology course.
- **GA criteria** — SRS integration verified against the shipped scheduler's expectations; offline replay
  correct; a11y audit passed.
- **Rollback** — remove from the allowlist; SRS items already created remain valid and reviewable.

## 16. Test Plan

- **Unit** — rating → SM-2 quality mapping; session card selection (new + due, capped, shuffled
  deterministically); state merge across tabs/offline; first-pass completion.
- **Integration** — SRS row creation/update against a real DB; SRS-disabled path writes nothing; reset
  with both scheduling choices; reverse items independent.
- **End-to-end** — Playwright: full session → reload shows updated status → next due date correct;
  offline session replays; keyboard-only session.
- **Security** — cross-enrollment rating attempts; forged card ids; SRS writes scoped to the caller.
- **Accessibility** — axe; screen-reader script for a full session; `lang` attributes on vocabulary
  cards; reduced motion; button naming.
- **Performance** — 20-card session transitions; chunk budget.
- **Manual exploratory** — math on cards, images, long definitions, RTL/CJK decks, bulk paste from
  Google Sheets and Excel.

## 17. Documentation & Training

- **Instructor** — deck size guidance; bulk paste; why editing meaning needs a new card; enabling SRS to
  get spacing; reading per-card difficulty.
- **Student** — how spacing works, why honest self-rating matters, keyboard shortcuts.
- **Developer** — the SRS adapter contract and card-identity rule.

## 18. Open Questions

1. Should decks be shareable across courses (a deck library)? Proposed: valuable, but it belongs with
   the course-content sharing model rather than this tool; revisit after usage data.
2. Should typed-recall (type the answer, then self-rate) be a mode? Proposed: yes as a follow-up — it is
   strictly better retrieval practice for some material, and reuses CT.11's text matching.
3. Should the review queue link back to the source page? Proposed: yes; it is a small change in the
   existing SRS surface and makes decks feel connected rather than orphaned.

## 19. References

- Existing files this work touches: `server/migrations/091_spaced_repetition.sql`,
  `server/internal/service/srs/` (`sm2.go`, `submit.go`), `server/internal/service/srsscheduler/`,
  `clients/web/src/components/content-tools/`.
- External standards: WCAG 2.1 AA; SM-2 algorithm (as already implemented); learning-science basis —
  spacing effect, testing effect.
- Related plans: [CT.11](CT.11-tool-inline-questions.md),
  [CT.7](CT.7-analytics-insights-and-gradebook.md),
  [CT.3](CT.3-student-runtime-and-state-persistence.md).
