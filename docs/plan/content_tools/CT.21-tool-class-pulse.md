# CT.21 — Tool: Class Pulse (vote, then see how the class answered)

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.21 |
| **Section** | Content Tools (CT) — tool shelf |
| **Severity** | MINOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Web platform team |
| **Depends on** | CT.1, CT.2, CT.3, CT.7 (aggregates) |
| **Unblocks** | Peer-instruction patterns; the first live-aggregate tool |

---

## 1. Problem Statement

"How many of you think it's A?" is free in a physical classroom and impossible in Lextures. Peer
instruction — vote, see the split, discuss, vote again — works precisely because the distribution is
visible, and asynchronous courses lose it entirely. The platform's live quiz product provides this only
in a synchronous, gamified session that an instructor must host; there is no way to place a simple
"where does the class stand?" moment inside a page a student reads on Tuesday night. This tool adds
one: an opinion or concept poll, answered once, followed by the anonymised class distribution.

## 2. Goals

- Let an author drop a poll into a page in under thirty seconds.
- Show the anonymised class distribution **after** the learner commits, never before.
- Support both opinion polls (no correct answer) and concept polls (a correct answer revealed after
  voting, per config).
- Support the two-vote peer-instruction cycle: vote → see split → discuss → revote.
- Protect small classes and individuals from re-identification.

## 3. Non-Goals

- Live/synchronous hosting, leaderboards, timers, points (the shipped interactive-quizzes product).
- Anonymous-to-the-instructor voting in v1 (the instructor can always see who voted, as with any
  activity — the anonymity is peer-facing). A truly anonymous mode is an open question below.
- Free-text polling (that is CT.20 or CT.22).
- Cross-course or cross-section aggregation.

## 4. Personas & User Stories

- **As an instructor**, I want a quick poll after a controversial claim so that students commit before
  reading my analysis.
- **As an instructor**, I want to see the split before class so that I know whether to spend ten minutes
  or two on the topic.
- **As a student**, I want to see how my classmates answered so that I calibrate my own thinking.
- **As a student**, I want to change my mind after seeing the split so that discussion actually matters.
- **As a student in a class of four**, I do not want the distribution to reveal what my classmate chose.

## 5. Functional Requirements

- **FR-1.** The author MUST configure a question and 2–8 options, optionally marking one correct
  (concept poll) and optionally supplying an explanation revealed after voting.
- **FR-2.** A learner MUST vote before seeing any distribution; the aggregate MUST NOT be present in the
  pre-vote payload.
- **FR-3.** The distribution MUST be suppressed until at least `minRespondents` (default 5, floor 3) have
  voted, with a clear explanatory message before that.
- **FR-4.** The author MUST be able to enable a **second vote**: after the distribution is shown, the
  learner may revote once; both votes MUST be stored and reported separately.
- **FR-5.** Correctness (concept polls) MUST be revealed per config: `after_vote`, `after_revote`, or
  `never`; the correct option MUST be `x-lex-sensitive` until revealed.
- **FR-6.** Aggregates MUST be computed server-side, cached (default 30 s TTL), scoped to the course
  (and to the learner's section when the course uses sections), and MUST exclude instructor/TA votes.
- **FR-7.** The instructor MUST see the full distribution at any time, the shift between first and
  second votes, and the CT.4 roster of who voted.
- **FR-8.** Votes MUST be one per enrollment per round; changing a vote within a round MUST be
  disallowed once submitted (the commitment is the point).
- **FR-9.** The tool MUST support `readOnly` and CT.4 reset (clearing votes and recomputing aggregates).
- **FR-10.** The distribution MUST be rendered as both a bar chart and an accessible table, with counts
  and percentages.
- **FR-11.** The tool MUST poll for updated aggregates while the learner has the tool in view (default
  every 30 s, stopping when the tab is hidden), so a class voting together sees movement without a
  WebSocket.
- **FR-12.** `scoring.mode` MUST be `'none'` for opinion polls; concept polls MAY report a score but
  MUST default to not doing so.

## 6. Non-Functional Requirements

- **Performance** — Vote round-trip p95 ≤ 150 ms; aggregate read served from cache p95 ≤ 60 ms;
  renderer ≤ 16 KB gz (bars are CSS/SVG, no charting library).
- **Security** — Aggregates never expose individual votes to peers; server-side vote-once enforcement;
  the correct option withheld until revealed.
- **Privacy & Compliance** — Peer-facing anonymity is a stated guarantee; small-*n* suppression enforced
  server-side (not just hidden in the UI); votes are education records for DSAR/retention purposes.
- **Accessibility** — WCAG 2.1 AA: options as a radio group in a `fieldset`; results as a table plus
  bars with text labels (never colour alone); result arrival announced politely; percentages and counts
  both stated.
- **Scalability** — Aggregates are a single grouped count per instance, cached; polling is throttled and
  visibility-aware; a 500-learner cohort voting in five minutes stays within one cached query per TTL.
- **Reliability** — Idempotent votes; a failed vote leaves the selection editable; cache invalidated on
  vote so the voter immediately sees their own contribution reflected.
- **Observability** — `lextures_content_tool_votes_total{round}`, aggregate cache hit rate, revote
  shift magnitude, suppression-hit rate.
- **Maintainability** — The aggregate-with-suppression helper is shared with CT.12's peer results.
- **Internationalization** — Percentages formatted per locale; RTL bar direction mirrored.
- **Backward compatibility** — Additive; unrelated to live quizzes.

## 7. Acceptance Criteria

- **AC-1.** *Given* an unvoted learner, *Then* the payload contains options but no aggregate and no
  correct-option marker.
- **AC-2.** *Given* the learner votes, *Then* the vote is stored once and the response includes the
  aggregate (or a suppression message when respondents < `minRespondents`).
- **AC-3.** *Given* 3 respondents and `minRespondents = 5`, *Then* the aggregate is withheld server-side
  and the UI explains why.
- **AC-4.** *Given* the learner tries to vote twice in the same round, *Then* the API refuses and the
  stored vote is unchanged.
- **AC-5.** *Given* a second vote is enabled and the learner revotes, *Then* both rounds are stored and
  the instructor sees the shift.
- **AC-6.** *Given* an instructor votes, *Then* their vote is excluded from the class distribution.
- **AC-7.** *Given* `revealCorrect = 'after_revote'`, *Then* the correct option is absent from the
  payload until the second vote is cast.
- **AC-8.** *Given* the tab is hidden, *Then* aggregate polling stops and resumes on focus.
- **AC-9.** *Given* a screen-reader user votes, *Then* the result is announced once and the table
  presents the same data as the bars.
- **AC-10.** *Given* a CT.4 reset for one learner, *Then* their vote is removed and the aggregate
  recomputes without it.

## 8. Data Model

**No migration.** Aggregates are computed from `analytics.content_tool_state_summaries` facets (CT.7),
so no new table is required even for a "live" tool — a useful proof of the framework's headroom.

```ts
// configSchema
type ClassPulseConfig = {
  question: string
  options: Array<{ id: string; text: string }>
  correctOptionId?: string              // x-lex-sensitive until reveal
  explanation?: string                  // x-lex-sensitive until reveal
  allowSecondVote: boolean              // default false
  revealCorrect: 'after_vote' | 'after_revote' | 'never'   // default 'never'
  minRespondents: number                // default 5, min 3
  scopeToSection: boolean               // default true when the course uses sections
  showPercentages: boolean              // default true
}

// stateSchema
type ClassPulseState = {
  v: 1
  votes: Array<{ round: 1 | 2; optionId: string; at: string }>
  sawAggregateAt?: string
  completedAt?: string
}
```

`scoring.mode = 'none'` (concept polls MAY set `'auto'`); `capabilities = ['state','aggregate']`;
`maxStateBytes = 4000`.

## 9. API Surface

**No new routes.**

- `POST .../actions/vote` — `{optionId, round, idempotencyKey}` → `{state, aggregate|suppressed}`.
- `GET .../instances/{id}/analytics` (CT.7) — instructor distribution and shift.
- Aggregate for students is returned by the vote action and refreshed by a lightweight
  `POST .../actions/aggregate` (no side effects, cached), so no new GET route is needed.

## 10. UI / UX

1. Question with a small "Class poll" label.
2. Options as radio cards; **Submit vote** button.
3. After voting: horizontal bars with option text, count and percentage; the learner's own choice marked
   with a "Your answer" tag; total respondents shown.
4. Concept polls: correctness marker appears at the configured reveal point, with the explanation.
5. Second-vote mode: after the distribution, a "Discuss with a classmate, then vote again" prompt and a
   fresh option set; afterwards both distributions are shown side by side.

**States** — *Not voted*, *Voting*, *Voted (aggregate)*, *Voted (suppressed, waiting for more)*,
*Revote available*, *Revealed*, *Read-only*, *Error (selection preserved)*.

**Mobile** — full-width option cards; bars stack with labels above.

**Accessibility** — `fieldset`/`legend`; bars have a companion table (always present, visually compact,
not hidden behind a toggle for a display this simple); result announced once; "Your answer" conveyed in
text; percentages and raw counts both given.

**Copy & i18n** — `contentTools.tools.classPulse.*`.

**Authoring** — the generic CT.2 schema form is sufficient (question + option rows + a correct toggle),
so this tool deliberately ships **no custom editor** — the second proof, alongside CT.12, that the
generic form carries simple tools.

## 11. AI / ML Considerations

None, in any form. This tool is deliberately AI-free so it remains available in every org policy state,
including districts that deny the `ai` capability wholesale — which also makes it the safest tool to
recommend first to a cautious district.

## 12. Integration Points

- **Internal** — `service/contenttools/tools/classpulse/` (vote enforcement, aggregate with
  suppression), CT.7 summaries as the aggregate source, `courseroles` (role exclusion, section scoping),
  `clients/web/src/components/content-tools/tools/class-pulse/`.
- **Shared helper** — `aggregateWithSuppression` reused by CT.12 peer results and any future
  distribution-showing tool.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1–CT.3, CT.7 (aggregate substrate).
- **Must ship before:** nothing.
- **Shared infra needed:** none beyond the framework.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Small-class re-identification | M | H | Server-side suppression below `minRespondents`, floor of 3, section scoping decided carefully |
| Conformity — students vote with the crowd | M | M | Aggregate only after commitment; vote-once per round; second vote is explicitly framed as post-discussion |
| Polling load in a synchronous class | M | M | 30 s TTL cache, visibility-aware polling, single grouped query |
| Instructors expect live real-time movement | M | L | Documented refresh cadence; WebSocket deferred until a second live tool justifies it |
| Poll used for sensitive topics (identity, opinions) | M | H | Peer anonymity guarantee, instructor guidance, CT.8 policy applies; note that instructors *can* see individual votes |

## 15. Rollout Plan

- **Feature flag** — course tool allowlist.
- **Sequencing** — vote action + suppression helper → renderer → aggregate polling → second-vote cycle →
  pilot.
- **Dogfood** — one large lecture course using peer instruction.
- **GA criteria** — suppression verified server-side; a11y audit passed; cache behaviour verified under a
  simulated class-sized burst.
- **Rollback** — remove from the allowlist; votes preserved.

## 16. Test Plan

- **Unit** — vote-once enforcement; round handling; suppression thresholds; role exclusion; section
  scoping; percentage formatting per locale.
- **Integration** — aggregate correctness against raw state; cache invalidation on vote; reveal gating;
  reset recomputation.
- **End-to-end** — Playwright: vote → distribution → revote → both shown; suppressed state with 3
  respondents; instructor view.
- **Security** — payload inspection pre-vote; double-vote attempts; forged round numbers; attempts to
  read aggregates without voting.
- **Accessibility** — axe; screen-reader script for vote and results; table/bar parity; contrast.
- **Performance** — 500 votes in 60 s with one cached query per TTL; chunk budget.
- **Manual exploratory** — ties, unanimous results, 8-option polls on mobile, RTL bars.

## 17. Documentation & Training

- **Instructor** — peer instruction in two votes; writing options that expose reasoning; what students
  can and cannot see; the small-class suppression rule.
- **Student** — your vote is anonymous to classmates, visible to your instructor.
- **Developer** — the aggregate-with-suppression helper; how a "live" tool avoids new tables.

## 18. Open Questions

1. Should a fully instructor-anonymous mode exist for climate/wellbeing polls? Proposed: yes as a
   distinct, clearly-labelled mode — but it needs a different storage shape (no enrollment linkage),
   which is a separate story rather than a config flag on this one.
2. Is 30 s the right refresh cadence, or should it adapt to observed vote velocity? Proposed: adaptive
   (5–60 s) based on recent vote rate, capped, if dogfood shows staleness complaints.
3. Should section scoping default on or off? Proposed: on when the course uses sections — a section's
   distribution is more useful than a 400-student aggregate.

## 19. References

- Existing files this work touches: `server/internal/service/contenttools/`,
  `server/internal/courseroles/`, `clients/web/src/components/content-tools/`.
- Precedents: interactive quizzes / live game aggregates (`quizgame` schema) — deliberately *not*
  reused, since this tool is asynchronous and stateless by comparison.
- External standards: WCAG 2.1 AA; learning-science basis — peer instruction (Mazur).
- Related plans: [CT.12](CT.12-tool-predict-and-reveal.md),
  [CT.7](CT.7-analytics-insights-and-gradebook.md),
  [CT.22](CT.22-tool-inline-discussion.md).
