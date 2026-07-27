# CT.11 — Tool: Inline Questions (check for understanding)

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.11 |
| **Section** | Content Tools (CT) — tool shelf |
| **Severity** | BLOCKER (one of the two reference tools that prove the framework) |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Assessment team |
| **Depends on** | CT.1, CT.2, CT.3 |
| **Unblocks** | Reference implementation for every scored tool; CT.7 scoring paths |

---

## 1. Problem Statement

Reading comprehension is invisible until something tests it, and the platform's only test is a quiz —
a separate page, a formal attempt, a grade. So instructors either over-formalise (a graded quiz for a
30-second concept) or check nothing. **Inline Questions** is the low-stakes middle: one or two
questions embedded in the flow of a page, answered in place, scored immediately, with feedback the
instructor wrote for each option. The result is stored per enrollment, so the instructor learns which
distractor the class fell for — the single most useful piece of formative data in teaching — without
anyone sitting a quiz.

## 2. Goals

- Let an author add 1–3 questions inline, with correct answers and per-option feedback, in under a minute.
- Score automatically **server-side**, store the response per enrollment, and give immediate feedback.
- Support the question types that carry most formative value inline: single-choice, multi-select,
  true/false, short text with accepted answers, and numeric with tolerance.
- Give the instructor a distractor-level distribution so they can see *which* misunderstanding is common.
- Stay low-stakes by default (ungraded, retryable) while allowing an instructor to link it to the
  gradebook through CT.7 when they want it to count.

## 3. Non-Goals

- Replacing quizzes: no timers, no question banks, no proctoring, no attempt lifecycle, no partial
  credit models beyond simple per-question scoring.
- Rich question types (matching, ordering, hotspot) — those are CT.14 and CT.15, which have their own
  interaction and accessibility requirements.
- Item analysis / IRT calibration (the shipped `service/itemanalysis` and `service/irt` operate on the
  question bank; CT.7 facets are enough here).
- AI generation of questions in v1 (a fast follow, wired through the `ui.aiAssist` hook from CT.2).

## 4. Personas & User Stories

- **As an instructor**, I want to drop two questions after a hard paragraph so that students test
  themselves before moving on.
- **As an instructor**, I want to write feedback for each wrong option so that a mistake teaches instead
  of just being wrong.
- **As an instructor**, I want to see that 14 of 30 chose the same wrong answer so that I know exactly
  what to reteach.
- **As a student**, I want to know immediately whether I understood so that I do not build on a
  misunderstanding.
- **As a student**, I want a second try on a low-stakes check so that the point is learning, not judging.
- **As a student using a screen reader**, I want the question, options and result announced clearly so
  that I can do the check independently.
- **As an instructor**, I want to reset a student's answers after we discussed the topic so that they
  can re-test their understanding.

## 5. Functional Requirements

- **FR-1.** The tool MUST support 1–3 questions per instance, each of type `single`, `multi`,
  `true_false`, `short_text` or `numeric`.
- **FR-2.** Correct answers, per-option feedback and general explanations MUST be marked
  `x-lex-sensitive` so the framework never sends them to a student before their response is scored
  (CT.1 FR-10).
- **FR-3.** Scoring MUST happen in the server action `submit`; the client MUST NOT receive the answer
  key in advance and MUST NOT compute the score.
- **FR-4.** `short_text` MUST support a list of accepted answers with per-answer options:
  case-insensitive (default), trim whitespace (default), and optional normalisation of punctuation.
- **FR-5.** `numeric` MUST support an absolute or relative tolerance and an optional unit string that is
  displayed but not graded.
- **FR-6.** After scoring, the response MUST show: correctness, the option-specific feedback (if any),
  the general explanation (if configured), and — when configured — the correct answer.
- **FR-7.** The instructor MUST configure attempt behaviour: `attempts` (1..5 or unlimited),
  `revealCorrectAfter` (`first_attempt` | `last_attempt` | `never`), and `shuffleOptions`.
- **FR-8.** State MUST record every attempt (selected values, correctness, timestamp), not just the last
  one, so instructors can see whether a student self-corrected.
- **FR-9.** The tool MUST report a score to the framework (`score_raw`/`score_max` from the last
  attempt by default, configurable to best or first), enabling CT.7's optional gradebook bridge.
- **FR-10.** The tool MUST support alignment to a course learning outcome or standard per question, so
  CT.7 can record mastery evidence.
- **FR-11.** Answers MUST be submitted per question (not one submit for the whole block) so a student
  gets feedback on question 1 before answering question 2 when `sequential` is enabled.
- **FR-12.** When all attempts are exhausted, the tool MUST enter a stable reviewed state that still
  shows the student's answers and the feedback.
- **FR-13.** The instructor MUST see per-question distributions with correct/incorrect marking and a
  most-chosen-distractor callout, and MUST be able to open the CT.4 roster from there.
- **FR-14.** Reset (CT.4) MUST clear all attempts and return the tool to its initial state, reverting
  any bridged grade.
- **FR-15.** `short_text` responses MUST pass through the CT.8 content filter before storage.

## 6. Non-Functional Requirements

- **Performance** — Submit round-trip p95 ≤ 200 ms (pure server logic, no external calls). Renderer
  chunk ≤ 20 KB gz.
- **Security** — The answer key never reaches a student's browser before scoring; scoring is
  server-only; attempt limits enforced server-side; a student cannot submit for another enrollment.
- **Privacy & Compliance** — Responses are education records (DSAR export, retention per CT.8). No AI,
  no egress in v1, so this tool is available under every org policy.
- **Accessibility** — Options are real radio/checkbox groups inside a `fieldset` with a `legend`
  carrying the question; correctness is conveyed by text and icon, never colour alone; result is
  announced politely; error states are associated via `aria-describedby`; full keyboard operation.
  WCAG 2.1 AA with no known limitations.
- **Scalability** — One row per learner per instance; submit is a single upsert plus an event append.
- **Reliability** — Idempotent submits (CT.3 FR-10) so a double-tap does not consume two attempts.
- **Observability** — `lextures_content_tool_submits_total{tool_id="inline_questions",outcome}`,
  `…_inline_questions_correct_total`, attempt-count histogram.
- **Maintainability** — Grading logic is one pure function per question type with a table-driven test
  suite; adding a type is a case plus tests, not a redesign.
- **Internationalization** — Question text and feedback are instructor content (any language);
  UI chrome localized; numeric parsing respects locale decimal separators.
- **Backward compatibility** — Additive; no interaction with quiz attempts.

## 7. Acceptance Criteria

- **AC-1.** *Given* a student loads a page with an unanswered check, *When* the instance payload is
  inspected, *Then* it contains no correct-answer or feedback fields.
- **AC-2.** *Given* a student selects an option and submits, *Then* the server returns correctness and
  the option-specific feedback, and the state records the attempt.
- **AC-3.** *Given* `attempts = 2`, *When* a student submits a third time, *Then* the API refuses with a
  typed error and the state is unchanged.
- **AC-4.** *Given* a `numeric` question with tolerance ±0.05, *When* the student answers 3.14 for
  a correct value of 3.14159, *Then* it is marked correct; 3.0 is marked incorrect.
- **AC-5.** *Given* a `short_text` question accepting "photosynthesis", *When* a student types
  " Photosynthesis ", *Then* it is correct under default normalisation.
- **AC-6.** *Given* `revealCorrectAfter = 'never'`, *When* attempts are exhausted, *Then* the correct
  answer is still absent from every client payload.
- **AC-7.** *Given* 30 students answered, *When* the instructor opens insights, *Then* the per-option
  distribution matches raw state and the most-chosen distractor is highlighted.
- **AC-8.** *Given* an instructor resets one student, *Then* that student sees a fresh, unanswered check
  and their prior attempts exist in the CT.4 snapshot.
- **AC-9.** *Given* the instance is bridged to the gradebook and a student improves on a second attempt
  with `scorePolicy = 'best'`, *Then* the gradebook shows the better score.
- **AC-10.** *Given* keyboard-only navigation, *When* the student answers and submits, *Then* every step
  is reachable, the result is announced once, and focus lands on the feedback.
- **AC-11.** *Given* a student submits the same request twice with one `idempotencyKey`, *Then* one
  attempt is recorded.
- **AC-12.** *Given* `sequential = true`, *When* question 1 is unanswered, *Then* question 2 is not
  interactive and this is conveyed to assistive technology.

## 8. Data Model

**No migration.** Manifest schemas only.

```ts
// configSchema
type InlineQuestionsConfig = {
  questions: Array<{
    id: string
    type: 'single' | 'multi' | 'true_false' | 'short_text' | 'numeric'
    prompt: string                       // markdown, may contain KaTeX
    options?: Array<{                    // single | multi | true_false
      id: string
      text: string
      correct: boolean                   // x-lex-sensitive
      feedback?: string                  // x-lex-sensitive
    }>
    acceptedAnswers?: string[]           // short_text — x-lex-sensitive
    caseSensitive?: boolean
    correctValue?: number                // numeric — x-lex-sensitive
    tolerance?: { kind: 'absolute' | 'relative'; value: number }
    unit?: string
    explanation?: string                 // x-lex-sensitive until reveal
    outcomeId?: string
    points?: number                      // default 1
  }>
  attempts: number | 'unlimited'         // default 2
  revealCorrectAfter: 'first_attempt' | 'last_attempt' | 'never'   // default 'last_attempt'
  shuffleOptions: boolean                // default false
  sequential: boolean                    // default false
  scorePolicy: 'last' | 'best' | 'first' // default 'last'
}

// stateSchema
type InlineQuestionsState = {
  v: 1
  answers: Record<string, {              // keyed by question id
    attempts: Array<{
      value: string | string[] | number
      correct: boolean
      at: string
    }>
    revealed: boolean
  }>
  scoreRaw?: number                      // server-written
  scoreMax?: number                      // server-written
  completedAt?: string
}
```

`scoring.mode = 'auto'`; `capabilities = ['state','scoring']`; `storage.maxStateBytes = 16000`;
conflict policy `server_wins` (the server is the authority on attempts).

## 9. API Surface

**No new routes.** Uses the CT.3 contract:

- `POST .../instances/{id}/actions/submit` — `{questionId, value, idempotencyKey}` →
  `{correct, feedback?, explanation?, correctAnswer?, attemptsRemaining, state}`.
- `POST .../instances/{id}/actions/reveal` — when `revealCorrectAfter` allows an explicit reveal.
- `PUT .../instances/{id}/state` — draft values only (typed-but-unsubmitted text), never correctness.
- Instructor distributions via `GET .../instances/{id}/analytics` (CT.7) with facets
  `questionId`, `optionId`, `correct`.

## 10. UI / UX

Rendered inside the CT.3 `ToolFrame` as a compact card:

1. Optional label ("Check your understanding").
2. Question prompt (Markdown + math), then the input appropriate to the type.
3. **Check** button; after submit, an inline result row: ✓/✗ with text, option feedback, explanation
   when revealed, and attempts remaining.
4. When multiple questions: a small progress dot row; `sequential` mode dims later questions.

**States** — *Unanswered*: neutral. *Correct*: success styling + text + icon. *Incorrect with attempts
left*: "Not quite — try again" + feedback + **Try again**. *Exhausted*: reviewed state with the
student's answer preserved. *Read-only*: answers and feedback, no controls. *Error*: retry, selection
preserved.

**Mobile** — full-width options with ≥ 44 px targets; feedback appears below without layout jump.

**Accessibility** — `fieldset`/`legend` per question; `aria-describedby` links feedback to the group;
result announced once politely; `aria-disabled` plus explanatory text for sequential locking; visible
focus; correctness never colour-only.

**Copy & i18n** — `contentTools.tools.inlineQuestions.*`.

**Authoring** — generic schema form is adequate for one question, but arrays of options with a
"correct" toggle are fiddly, so this tool ships a **custom editor** (CT.2 FR-8): question type picker,
option rows with radio-style correct marking, inline feedback fields, and a live student preview. This
is deliberate: it exercises the custom-editor path that later tools will need.

## 11. AI / ML Considerations

None in v1 — no model call, so the tool is available under every org AI policy. Two AI touchpoints are
reserved and explicitly deferred: (a) **generate questions from this section** via the CT.2 `ui.aiAssist`
hook, reusing `service/quizgenerationai` prompts, and (b) **misconception tagging** of common wrong
answers via the shipped `service/misconception`, which would let CT.7 name the misconception rather
than just the distractor. Both are fast follows with their own disclosure requirements.

## 12. Integration Points

- **Internal** — `service/contenttools/tools/inlinequestions/` (grading functions),
  `service/outcomes` + `service/sbgaggregation` (outcome evidence), `service/contentfilter`
  (short-text), `service/misconception` (reserved),
  `clients/web/src/components/content-tools/tools/inline-questions/`.
- **CT.7** — facet schema (`questionId`, `optionId`, `correct`), score reporting, optional grade link.
- **Question bank** — deliberately *not* coupled in v1; an "import from question bank" affordance is a
  noted enhancement once demand is proven.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1–CT.3.
- **Must ship before:** CT.7's scoring paths are exercised end-to-end (this is their first consumer).
- **Shared infra needed:** none beyond the framework.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Answer key leaking to the client | M | H | Framework-level `x-lex-sensitive` redaction, server-only scoring, explicit test asserting payload contents |
| Instructors treat it as a quiz and grade everything | M | M | Ungraded default, explicit CT.7 opt-in, documentation framing it as formative |
| Short-text grading frustrates students with near-miss answers | H | M | Normalisation defaults, multiple accepted answers, generous default attempts, instructor-visible near-miss list in insights |
| Shuffling breaks stored option references | M | M | Options carry stable ids; shuffle is display-only |
| Overuse makes pages feel like worksheets | M | L | `max_instances_per_item` (CT.1) and authoring guidance |
| Numeric locale parsing errors (comma decimals) | M | M | Locale-aware parsing with tests for `de-DE`, `fr-FR`, `en-US` |

## 15. Rollout Plan

- **Feature flag** — course tool allowlist only.
- **Sequencing** — grading functions + tests → manifest → renderer → custom editor → insights facets →
  pilot.
- **Dogfood** — the intro course adds two checks per page; instructors review distributions weekly.
- **GA criteria** — grading test matrix green for every type; a11y audit passed; zero answer-key
  leakage findings; p95 submit ≤ 200 ms.
- **Rollback** — remove from the allowlist; responses preserved.

## 16. Test Plan

- **Unit** — grading table per type (exact, tolerance boundaries, multi-select partial vs strict,
  normalisation cases, locale decimals); attempt accounting; reveal policy; score policy (last/best/first).
- **Integration** — submit → state → score; attempt exhaustion; idempotent double submit; reset clears
  and reverts a bridged grade; outcome evidence written.
- **End-to-end** — Playwright: answer wrong → feedback → retry → correct → reviewed state persists
  across reload; sequential mode; instructor sees the distribution.
- **Security** — payload inspection for answer keys at every state; submitting for another enrollment;
  tampered `questionId`/`optionId`; oversized short-text.
- **Accessibility** — axe; screen-reader script for each question type; keyboard-only completion;
  verification that correctness is not colour-only (contrast + icon + text).
- **Performance** — submit latency; renderer chunk size budget.
- **Manual exploratory** — math-heavy prompts, RTL locales, very long option text, 3-question sequential
  blocks on mobile.

## 17. Documentation & Training

- **Instructor** — writing good distractors and feedback; when to use inline checks vs a quiz;
  reading distributions; resetting after reteaching.
- **Student** — low-stakes framing: "this does not count unless your teacher says so".
- **Developer** — this tool as the reference **scored** tool: server-side grading, sensitive config,
  custom editor, CT.7 facets.
- **Runbook** — investigating a disputed auto-grade.

## 18. Open Questions

1. Should multi-select default to strict (all-or-nothing) or partial credit? Proposed: strict by
   default with a partial-credit option, matching quiz semantics teachers already know.
2. Should the tool offer "import a question from the question bank"? Proposed: defer until instructors
   ask; the value of inline checks is that they are quick to write in place.
3. Should exhausted-attempt state auto-reveal the correct answer for K-12 program types? Proposed: yes
   as a default (`revealCorrectAfter='last_attempt'`), configurable.
4. Do we want a "confidence" prompt attached to each answer, or is that CT.12's job? Proposed: CT.12's
   job — keep this tool simple.

## 19. References

- Existing files this work touches: `server/internal/service/outcomes/`,
  `server/internal/service/sbgaggregation/`, `server/internal/service/contentfilter/`,
  `clients/web/src/components/content-tools/`.
- Precedents followed: quiz question types (`server/migrations/075_question_bank.sql`), quiz response
  grading (`service/quizattemptgrading`).
- External standards: WCAG 2.1 AA (form and error patterns); QTI question-type vocabulary (informative).
- Related plans: [CT.7](CT.7-analytics-insights-and-gradebook.md),
  [CT.12](CT.12-tool-predict-and-reveal.md), [CT.14](CT.14-tool-sort-and-sequence.md),
  [CT.18](CT.18-tool-step-through-worked-example.md).
