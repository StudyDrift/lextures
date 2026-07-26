# CT.12 — Tool: Predict & Reveal (commit a prediction, rate confidence, then see the answer)

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.12 |
| **Section** | Content Tools (CT) — tool shelf |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Assessment team |
| **Depends on** | CT.1, CT.2, CT.3 |
| **Unblocks** | Confidence-calibration reporting in CT.7 |

---

## 1. Problem Statement

The most wasteful sentence in instructional content is "as you can see, the result is…" — the answer
arrives before the learner has committed to a guess, so nothing is learned that was not already known.
Decades of learning-science evidence (the generation effect, hypercorrection of high-confidence errors)
say the opposite ordering works better: make the learner *predict*, make them *own* the prediction with
a confidence rating, and only then reveal. Lextures has no way to author that ordering; an author can
only hide the answer behind their own willpower. This tool makes commit-then-reveal a first-class,
stateful interaction.

## 2. Goals

- Force a commitment before revealing: the reveal is literally impossible until a prediction is stored.
- Capture **confidence** alongside the prediction so surprise — the strongest driver of correction — is
  visible to both learner and instructor.
- Store prediction, confidence, reveal time and (optionally) a post-reveal reflection per enrollment.
- Give instructors a calibration view: where the class was confidently wrong, which is exactly where
  reteaching pays.
- Work for both objective predictions (choose an outcome) and open ones (write what you expect).

## 3. Non-Goals

- Grading. `scoring.mode = 'none'` by default; a prediction being "wrong" is the pedagogical point.
  (An optional non-weighted correctness flag exists for analytics only.)
- Replacing inline questions (CT.11) — that tool tests knowledge; this one primes learning.
- Multi-step branching predictions (CT.24-class branching scenarios, not planned in this batch).
- Timed reveals or instructor-controlled class-wide reveal (a live-class feature, noted as future work).

## 4. Personas & User Stories

- **As an instructor**, I want students to guess what happens before I show the demo result so that the
  demo actually teaches.
- **As an instructor**, I want to see who was confidently wrong so that I address the misconception,
  not just the answer.
- **As a student**, I want to commit to a guess and then find out so that I remember the surprise.
- **As a student**, I want to see my own prediction next to the answer so that I notice how my thinking
  differed.
- **As a science teacher**, I want to use the classic predict-observe-explain cycle inside my page so
  that I do not need a separate worksheet.

## 5. Functional Requirements

- **FR-1.** The tool MUST support two prediction modes: `choice` (pick one of the author's outcomes) and
  `open` (short free text), configurable per instance.
- **FR-2.** The tool MUST require a prediction before the reveal control becomes enabled; the reveal
  content MUST be `x-lex-sensitive` and MUST NOT be present in the client payload before commitment.
- **FR-3.** The tool MUST capture confidence on a configurable scale (default 3-point: guessing /
  fairly sure / certain; optional 5-point or 0–100%), required by default.
- **FR-4.** On reveal, the tool MUST display the author's reveal content (Markdown, math, image) and the
  student's own prediction side by side.
- **FR-5.** The tool MAY collect a post-reveal reflection ("what surprised you?") when configured, and
  MUST store it in state.
- **FR-6.** Commitment MUST be irreversible for the student (no edit after reveal); only a CT.4 reset
  restores the pre-commit state.
- **FR-7.** In `choice` mode the author MAY mark one or more outcomes as correct; correctness MUST be
  recorded for analytics but MUST NOT produce a score by default.
- **FR-8.** The instructor MUST see: prediction distribution, confidence distribution, and a
  **calibration matrix** (confidence × correctness) with the confidently-wrong cell highlighted.
- **FR-9.** In `open` mode the instructor MUST be able to read predictions in a list, and free text MUST
  pass the CT.8 content filter.
- **FR-10.** The tool MUST support an optional "class results" reveal: after committing, the student may
  see the anonymised distribution of peers' predictions (off by default; requires ≥ 5 respondents).
- **FR-11.** Reveal MUST be recorded with a timestamp so CT.7 can report time-to-reveal.
- **FR-12.** The tool MUST support `readOnly`, showing prediction and reveal without controls.

## 6. Non-Functional Requirements

- **Performance** — Commit round-trip p95 ≤ 150 ms; renderer ≤ 18 KB gz.
- **Security** — Reveal content withheld server-side until a commit row exists for the caller; commit is
  an action, not a state write, so the client cannot forge the gate.
- **Privacy & Compliance** — Predictions are student work (DSAR, retention). Peer distribution respects
  the CT.7 small-*n* threshold and is always anonymous.
- **Accessibility** — Confidence scale is a labelled radio group (not a bare slider); the reveal is
  announced politely and focus moves to it; the calibration matrix has a table alternative; no
  colour-only encoding of correctness. WCAG 2.1 AA.
- **Scalability** — One row per learner; peer distribution served from the CT.7 aggregate cache.
- **Reliability** — Idempotent commit; a failed commit leaves the prediction editable.
- **Observability** — `lextures_content_tool_commits_total{tool_id="predict_reveal"}`, confidence and
  time-to-reveal histograms, confidently-wrong rate per instance.
- **Maintainability** — No bespoke server logic beyond the commit gate and correctness tagging.
- **Internationalization** — Confidence labels localized and configurable; RTL verified.
- **Backward compatibility** — Additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* an uncommitted student, *When* the instance payload is inspected, *Then* reveal
  content is absent and the reveal control is disabled.
- **AC-2.** *Given* a student commits a prediction and confidence, *Then* the reveal content is returned
  by the action and stored state contains prediction, confidence and `committedAt`.
- **AC-3.** *Given* a committed student, *When* they attempt to change their prediction, *Then* the API
  refuses and the UI offers no edit path.
- **AC-4.** *Given* confidence is required, *When* a student tries to commit without it, *Then* an inline
  error appears and no commit occurs.
- **AC-5.** *Given* 20 students committed, *When* the instructor opens insights, *Then* the calibration
  matrix matches raw state and the confidently-wrong cell is highlighted.
- **AC-6.** *Given* peer results are enabled and only 3 students responded, *Then* the distribution is
  suppressed with an explanation.
- **AC-7.** *Given* a CT.4 reset, *Then* the student returns to the uncommitted state and the reveal is
  again withheld from the payload.
- **AC-8.** *Given* keyboard-only use, *Then* prediction, confidence, commit and reveal are all
  reachable, and the reveal is announced once.
- **AC-9.** *Given* `open` mode, *When* a prediction contains filtered content, *Then* the configured
  CT.8 action applies before storage.

## 8. Data Model

**No migration.**

```ts
// configSchema
type PredictRevealConfig = {
  question: string                        // markdown
  mode: 'choice' | 'open'                 // default 'choice'
  outcomes?: Array<{ id: string; text: string; correct?: boolean }>   // choice mode
  openPlaceholder?: string
  confidenceScale: 'none' | 'three' | 'five' | 'percent'   // default 'three'
  confidenceRequired: boolean             // default true
  reveal: { markdown: string; imageUrl?: string }          // x-lex-sensitive until commit
  reflectionPrompt?: string
  showPeerResults: boolean                // default false
}

// stateSchema
type PredictRevealState = {
  v: 1
  prediction?: { outcomeId?: string; text?: string }
  confidence?: number                     // normalized 0..1
  committedAt?: string
  revealedAt?: string
  correct?: boolean                       // analytics only, choice mode
  reflection?: string
}
```

`scoring.mode = 'none'`; `capabilities = ['state','aggregate']`; `maxStateBytes = 8000`;
conflict policy `server_wins`.

## 9. API Surface

**No new routes.**

- `POST .../actions/commit` — `{prediction, confidence, idempotencyKey}` → `{reveal, state, peerResults?}`.
- `POST .../actions/reflect` — `{text}` → `{state}`.
- `PUT .../state` — draft text before commit only.
- Insights via CT.7 with facets `outcomeId`, `confidenceBucket`, `correct`.

## 10. UI / UX

1. Question (Markdown/math), then the prediction control (radio list or textarea).
2. Confidence row: "How sure are you?" as a labelled radio group.
3. **Commit prediction** button (disabled until valid), with helper text: "You can't change this after."
4. After commit: a two-column layout — *Your prediction* / *What actually happens* — with the reveal
   content, optional peer distribution, and the optional reflection prompt.

**States** — *Uncommitted*, *Committing*, *Revealed*, *Read-only*, *Error (retry, input preserved)*,
*Peer results suppressed (small n)*.

**Mobile** — columns stack; reveal scrolls into view with focus moved to its heading.

**Accessibility** — `fieldset`/`legend` for prediction and confidence; reveal container has a heading
and receives focus; `aria-live="polite"` announcement "Answer revealed"; calibration matrix has a table
alternative and text summary.

**Copy & i18n** — `contentTools.tools.predictReveal.*`; confidence labels overridable by the instructor.

## 11. AI / ML Considerations

None in v1. Two reserved enhancements: (a) AI clustering of `open` predictions into themes for the
instructor view (reuse CT.10's clustering path, disclosed and budgeted); (b) mapping predictions to the
shipped misconception library so a confidently-wrong pattern names the misconception. Both are opt-in
and deferred.

## 12. Integration Points

- **Internal** — `service/contenttools/tools/predictreveal/`, `service/contentfilter` (open mode),
  `service/misconception` (reserved), `clients/web/src/components/content-tools/tools/predict-reveal/`.
- **CT.7** — calibration facets; time-to-reveal metric.
- **ACE** — confidently-wrong signals are a useful adaptation input (AC.2 profile); exposed via CT.7,
  not wired directly.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1–CT.3.
- **Must ship before:** nothing.
- **Shared infra needed:** none beyond the framework.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Students game the gate by guessing randomly to see the answer | H | M | Confidence capture makes low-effort guessing visible; instructor insight shows guess-rate; framing copy explains the point |
| Irreversible commit frustrates mis-taps | M | M | Explicit confirm-style helper text, large targets, instructor reset available, self-reset when permitted |
| Peer results create conformity pressure | M | M | Off by default, shown only after commit, always anonymous, small-*n* suppressed |
| Open predictions are unreadable at scale | M | M | Instructor list view with filters; AI clustering reserved |
| Authors misuse it as a quiz | M | L | Ungraded by design, documentation, correctness optional |

## 15. Rollout Plan

- **Feature flag** — course tool allowlist.
- **Sequencing** — manifest + commit gate → renderer → calibration insights → pilot in a science course.
- **Dogfood** — one physics/chemistry unit; compare recall on predicted vs non-predicted concepts.
- **GA criteria** — gate never leaks reveal content (test asserted); a11y audit passed.
- **Rollback** — remove from the allowlist; state preserved.

## 16. Test Plan

- **Unit** — commit gate; confidence normalisation across scales; correctness tagging; calibration
  bucketing; reveal redaction.
- **Integration** — payload contains no reveal before commit; commit idempotency; reset restores the
  gate; peer aggregate suppression at n<5.
- **End-to-end** — Playwright: predict → commit → reveal → reflect → reload persists; attempt to edit
  after commit; keyboard-only path.
- **Security** — direct action calls attempting reveal without commit; tampered outcome ids; filter on
  open text.
- **Accessibility** — axe; screen-reader script for commit and reveal; matrix table alternative.
- **Performance** — commit latency; chunk size.
- **Manual exploratory** — image-heavy reveals, math reveals, long open predictions, RTL.

## 17. Documentation & Training

- **Instructor** — the predict-observe-explain pattern; writing outcomes that expose misconceptions;
  reading the calibration matrix.
- **Student** — why committing first helps you remember.
- **Developer** — the commit-gate pattern as the reference for any "withhold until action" tool.

## 18. Open Questions

1. Should a confidently-wrong result trigger an automatic follow-up (e.g. surface a CT.11 check)?
   Proposed: not automatically; expose the signal and let ACE/instructors act.
2. Is a 3-point confidence scale enough for calibration research value, or should percent be default?
   Proposed: 3-point default for K-12 usability, percent available for HE.
3. Should instructors be able to trigger a class-wide reveal during a live lesson? Proposed: valuable,
   but it needs the live-session channel — defer to a live-class story.

## 19. References

- Existing files this work touches: `server/internal/service/contentfilter/`,
  `server/internal/service/misconception/`, `clients/web/src/components/content-tools/`.
- External standards: WCAG 2.1 AA; learning-science basis — generation effect, hypercorrection effect,
  predict-observe-explain (White & Gunstone).
- Related plans: [CT.11](CT.11-tool-inline-questions.md), [CT.21](CT.21-tool-class-pulse.md),
  [CT.7](CT.7-analytics-insights-and-gradebook.md),
  [1.10 misconception detection](../../completed/01-adaptive-learning-core/).
