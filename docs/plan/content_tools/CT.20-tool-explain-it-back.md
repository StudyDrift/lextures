# CT.20 — Tool: Explain It Back (self-explanation with AI formative feedback)

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.20 |
| **Section** | Content Tools (CT) — tool shelf |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | AI platform team |
| **Depends on** | CT.1, CT.2, CT.3, CT.6, CT.8 |
| **Unblocks** | Writing-to-learn workflows; formative feedback at a scale instructors cannot staff |

---

## 1. Problem Statement

Explaining an idea in your own words is one of the strongest learning interventions known, and one of
the least assigned — because thirty explanations is thirty pieces of reading an instructor does not
have time for, so the feedback loop that makes the exercise work never closes. Lextures can already
grade essays with an agent and give rubric feedback on assignments, but those are heavyweight, graded,
submission-based flows. There is nothing that says: *in three sentences, explain why this works* —
right here, ungraded, with immediate, specific, non-judgemental feedback, and a class-level summary of
what people did and did not grasp.

## 2. Goals

- Let a learner write a short explanation inline and receive **formative** feedback in seconds.
- Ground the feedback in the actual content (CT.6 pack) and in the author's criteria — not in the
  model's general opinion.
- Keep it ungraded by default and framed as practice, so learners write honestly rather than performing.
- Give instructors a class-level view: which key ideas appeared, which were missing, and a few
  representative explanations to use in class.
- Provide a non-AI fallback path (instructor review) so the activity works under every org policy.

## 3. Non-Goals

- Grading writing (the shipped grading agent and rubrics own that).
- Long-form essays or multi-draft workflows — this is 2–6 sentences.
- Plagiarism/originality checking (shipped elsewhere; if an author needs it, the activity belongs in an
  assignment).
- Replacing peer feedback (CT.22 is the discussion tool).

## 4. Personas & User Stories

- **As an instructor**, I want students to explain the concept in their own words so that I find out who
  actually understands it.
- **As an instructor**, I want the feedback to point at what is missing rather than give the answer so
  that the next attempt is theirs.
- **As an instructor**, I want a summary of what my class said so that I can open the next lesson with
  their words.
- **As a student**, I want feedback right away so that I know whether my understanding holds.
- **As a student**, I want it to be low stakes so that I can be honest about what I do not know.
- **As a district that has disabled AI**, I want the activity to still work so that policy does not
  remove the pedagogy.

## 5. Functional Requirements

- **FR-1.** The author MUST configure a prompt, a length guide (min/max words), and 2–6 **key points**
  the explanation should contain (each with a short label and description).
- **FR-2.** Key points MUST be `x-lex-sensitive` — the learner MUST NOT see the checklist before writing
  (that would turn explanation into transcription), and MAY see it after submitting when configured.
- **FR-3.** On submit, a server action MUST call the model with the CT.6 context pack, the author's key
  points and the learner's text, returning: which key points were addressed, one strength, one concrete
  suggestion, and (optionally) one probing question.
- **FR-4.** Feedback MUST be formative in register: no grade, no score shown by default, never
  judgemental, and never simply the correct answer.
- **FR-5.** The author MUST configure attempts (default 3) so the learner can revise after feedback;
  every attempt and its feedback MUST be stored.
- **FR-6.** The tool MUST support `aiFeedback: false`, in which case submissions are stored for
  instructor review and the learner sees an acknowledgement — the same activity without a model call.
- **FR-7.** The tool MUST fall back to the non-AI path automatically when AI is denied by policy
  (CT.8), the learner has opted out, or the provider is unavailable — never blocking the learner.
- **FR-8.** Learner text MUST be PII-redacted before egress and content-filtered before storage (CT.8).
- **FR-9.** Key-point coverage MUST be recorded as a facet so CT.7 can show class-level coverage; the
  instructor MUST see coverage per key point and a small set of representative (anonymised)
  explanations, selected deterministically rather than by the model when possible.
- **FR-10.** The instructor MUST be able to read individual explanations from the CT.4 roster and to
  leave a short instructor note visible to that learner.
- **FR-11.** Model output MUST be validated against a strict schema; a malformed response MUST be
  retried once and then degrade to the non-AI acknowledgement rather than showing raw output.
- **FR-12.** Rate limits: default 10 submissions/day/learner/instance, plus CT.6 course/org budgets.
- **FR-13.** The tool MUST report `status='completed'` after the first substantive submission (length
  guide met), regardless of feedback quality — completion is about writing, not about pleasing a model.
- **FR-14.** CT.4 reset MUST clear all attempts and feedback.

## 6. Non-Functional Requirements

- **Performance** — Feedback returned p95 ≤ 6 s; the writing surface never blocks on the model.
- **Security** — Model access server-side only; author key points never sent to the client pre-submit;
  prompt-injection defence on both content sources and learner text (learner text is data, not
  instruction).
- **Privacy & Compliance** — CT.8 governs: disclosure, COPPA gating, redaction, DSAR, retention,
  crisis-signal escalation on free text.
- **Accessibility** — WCAG 2.1 AA: labelled textarea with word count, feedback rendered as a labelled
  region announced politely on arrival, keyboard-complete, no timing pressure, feedback readable at
  200% zoom without horizontal scroll.
- **Scalability** — One row per learner; representative-explanation selection cached (CT.7).
- **Reliability** — Draft autosaved continuously; feedback failure never loses the learner's text;
  idempotent submissions prevent double spend.
- **Observability** — `lextures_content_tool_ai_calls_total{tool_id="explain_it_back",outcome}`,
  schema-validation failure rate, fallback rate, revision rate (did feedback prompt a second attempt?).
- **Maintainability** — One versioned prompt with a golden-set eval; the same structured-output pattern
  is reusable by future feedback tools.
- **Internationalization** — Feedback in the learner's locale; the prompt instructs the model to respond
  in the language of the learner's text when it differs from the UI locale.
- **Backward compatibility** — Additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* an unsubmitted learner, *Then* the payload contains the prompt and length guide but
  no key points.
- **AC-2.** *Given* a submitted explanation, *Then* feedback returns within the latency budget with
  per-key-point coverage, one strength and one suggestion, and both text and feedback are stored.
- **AC-3.** *Given* the model returns malformed JSON twice, *Then* the learner sees the non-AI
  acknowledgement, their text is stored, and an error metric increments.
- **AC-4.** *Given* org policy denies AI, *Then* the tool runs in review mode with no provider call and
  the learner sees an accurate description of what happens next.
- **AC-5.** *Given* a learner revises after feedback, *Then* both attempts and both feedback objects are
  stored and the instructor can see the progression.
- **AC-6.** *Given* learner text containing an email address, *Then* the provider payload is redacted and
  the stored log shows the redacted form.
- **AC-7.** *Given* learner text containing a crisis signal, *Then* the CT.8 escalation fires and the
  learner sees a supportive, non-alarming message rather than a normal feedback card.
- **AC-8.** *Given* 25 submissions, *When* the instructor opens insights, *Then* per-key-point coverage
  matches stored facets and representative explanations are anonymised.
- **AC-9.** *Given* a screen-reader user submits, *Then* the arrival of feedback is announced once and
  focus is not stolen from the text area.
- **AC-10.** *Given* a CT.4 reset, *Then* attempts and feedback are cleared and snapshotted.

## 8. Data Model

**No migration.**

```ts
// configSchema
type ExplainItBackConfig = {
  prompt: string                            // "In your own words, explain why…"
  minWords: number                          // default 25
  maxWords: number                          // default 150
  keyPoints: Array<{ id: string; label: string; description: string }>   // x-lex-sensitive
  revealKeyPointsAfterSubmit: boolean       // default true
  aiFeedback: boolean                       // default true
  feedbackStyle: 'encouraging' | 'neutral' | 'socratic'   // default 'encouraging'
  attempts: number                          // default 3
  includeProbeQuestion: boolean             // default true
  allowInstructorNote: boolean              // default true
}

// stateSchema
type ExplainItBackState = {
  v: 1
  attempts: Array<{
    at: string
    text: string
    feedback?: {
      covered: string[]                     // key point ids
      missing: string[]
      strength: string
      suggestion: string
      probe?: string
      mode: 'ai' | 'review'
    }
  }>
  instructorNote?: { text: string; at: string; by: string }
  completedAt?: string
}
```

`scoring.mode = 'none'` (optionally `manual`); `capabilities = ['state','ai']`;
`maxStateBytes = 48000`.

## 9. API Surface

**No new routes.**

- `PUT .../state` — draft text (debounced, never sent to a model).
- `POST .../actions/submit` — `{text, idempotencyKey}` → `{feedback, state}`; server builds context,
  gates via `aigateway`, calls the model with structured output, validates, filters, stores.
- `POST .../actions/instructorNote` — instructor-only note on a learner's explanation.
- Insights via CT.7 facets `keyPointId`, `covered`, `attemptCount`.

## 10. UI / UX

1. Prompt (Markdown) with the length guide ("about 3–5 sentences") and a live word count.
2. Textarea with autosave indicator; **Submit for feedback** (or **Submit** in review mode).
3. Feedback card: *What you got* (covered key points as labelled chips), *What's missing* (only after
   the first attempt, phrased as an invitation), *One strength*, *One suggestion*, optional *Think about*
   probe question, and **Revise** (attempts remaining).
4. Optional instructor note appears above the feedback when present.

**States** — *Empty (draft)*, *Too short (submit disabled with guidance)*, *Submitting*, *Feedback
shown*, *Revising*, *Review mode (acknowledgement)*, *AI unavailable (auto-fallback message)*,
*Read-only*, *Error (text preserved, retry)*.

**Mobile** — full-width textarea, feedback card below, sticky word count.

**Accessibility** — textarea labelled with the prompt; word count is `aria-live="polite"` but throttled;
feedback region is `role="region"` with a heading and announced once on arrival; chips carry text labels
not colour alone; no auto-focus theft.

**Copy & i18n** — `contentTools.tools.explainItBack.*`; feedback framing strings are reviewed for tone
with a K-12 register.

**Authoring** — custom editor: prompt field, key-point rows (label + description), length guide,
feedback style, and a **test with a sample answer** action so the author sees the feedback their students
will get before publishing.

## 11. AI / ML Considerations

- **Feature id** — `content_tool_explain_back` in `aigateway`; disclosed, budgeted, logged.
- **Prompt** — versioned; inputs are the CT.6 pack, the author's key points, the learner's redacted text,
  and the feedback style. Structured output (strict JSON schema) with `covered`, `missing`, `strength`,
  `suggestion`, `probe`. Explicit instructions: do not reveal missing content verbatim, do not grade, do
  not moralise, respond in the learner's language.
- **Eval** — golden set of 80 (explanation, key points) pairs measuring key-point detection precision and
  recall, tone compliance, non-revelation (does the feedback leak the answer?), and language matching.
  Gates prompt/model changes.
- **Bias** — feedback tone and coverage detection audited across writing quality bands and non-native
  English samples; disparate strictness is a release blocker (CT.8 fairness commitment).
- **Fallback** — malformed output → one retry → review mode. Provider down → review mode.
- **Cost** — ~1.5k context + 300 completion tokens per submission; per-learner daily cap; course budget.

## 12. Integration Points

- **Internal** — `service/contenttools/tools/explainitback/`, CT.6 context, `service/aigateway`,
  `service/aiprovider`, `service/contentfilter` (+ crisis escalation),
  `clients/web/src/components/content-tools/tools/explain-it-back/`.
- **Adjacent** — the shipped grading agent and rubric feedback remain for graded writing; documentation
  draws the line clearly.
- **ACE** — key-point coverage is a strong signal for adaptation profiles; exposed through CT.7.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1–CT.3, CT.6, CT.8.
- **Must ship before:** nothing.
- **Shared infra needed:** AI provider, content filter.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Feedback reveals the answer, defeating the exercise | H | H | Explicit non-revelation instruction, eval metric for leakage, `missing` phrased as invitations not answers |
| Model is harsher on non-native writing | M | H | Fairness audit across writing bands, tone constraints, instructor visibility, review-mode escape |
| Students write to please the model | M | M | Ungraded framing, key points hidden pre-submit, instructor sees real text |
| Cost scales with class size | M | M | Per-learner caps, course budget, short outputs, cheap-model default option |
| Crisis content in a free-text field | M | H | CT.8 escalation path with a supportive UI state |
| Instructors expect it to grade | M | M | Documentation and UI copy; `scoring.mode = 'none'` by default |

## 15. Rollout Plan

- **Feature flag** — course tool allowlist + org AI policy.
- **Sequencing** — action + structured-output validation → prompt + evals → renderer → review-mode
  fallback → insights → authoring test-with-sample → pilot.
- **Dogfood** — two courses across different subjects; instructors compare feedback to their own.
- **GA criteria** — eval targets met (leakage < 2%, coverage F1 at target), fairness audit clean,
  fallback verified, a11y audit passed.
- **Rollback** — set `aiFeedback: false` platform-wide via the CT.8 AI kill path (the tool keeps working
  in review mode), or remove from the allowlist.

## 16. Test Plan

- **Unit** — structured-output validation (valid, malformed, extra fields, wrong types); key-point
  redaction; word-count gating; attempt accounting; fallback selection logic.
- **Integration** — end-to-end with a stubbed provider; `aigateway` denial → review mode; budget denial;
  filter block; crisis escalation; reset.
- **End-to-end** — Playwright: write → submit → feedback → revise → second feedback → reload persists;
  AI-denied org runs in review mode; opted-out learner path.
- **Security** — injection corpus in both content sources and learner text; payload inspection for key
  points; cross-enrollment submissions.
- **Accessibility** — axe; screen-reader script for submit and feedback arrival; word-count announcement
  throttling; 200% zoom.
- **Performance / load** — 60 concurrent submissions; token spend per submission within budget.
- **Manual exploratory** — very short and very long answers, non-English answers, deliberately wrong
  explanations, sarcastic/off-topic answers.

## 17. Documentation & Training

- **Instructor** — writing prompts and key points that produce useful feedback; the difference between
  this and a graded writing assignment; reading class coverage; using review mode.
- **Student** — feedback is practice, not a grade; how to revise.
- **Admin** — AI policy, budgets, retention of explanations and feedback.
- **Runbook** — provider outage (auto review mode), eval regression response.

## 18. Open Questions

1. Should the instructor be able to promote an anonymised explanation to the class as an exemplar?
   Proposed: yes with explicit consent from the author, as a fast follow.
2. Should coverage feedback be shown as chips (fast) or prose (gentler)? Proposed: chips plus one prose
   sentence; A/B during dogfood.
3. Should repeated near-identical submissions be detected (gaming for a "covered" verdict)? Proposed:
   flag to the instructor, do not block.
4. Should review mode notify the instructor per submission or in a digest? Proposed: digest, to avoid
   notification fatigue.

## 19. References

- Existing files this work touches: `server/internal/service/aigateway/service.go`,
  `server/internal/service/contentfilter/`, `server/internal/service/aitutor/aitutor.go` (redaction),
  `clients/web/src/components/content-tools/`.
- External standards: OWASP LLM Top 10 (LLM01, LLM02); WCAG 2.1 AA; learning-science basis —
  self-explanation effect (Chi), elaborative interrogation.
- Related plans: [CT.6](CT.6-grounded-context-and-link-ingestion.md),
  [CT.8](CT.8-governance-safety-privacy-accessibility.md), [CT.10](CT.10-tool-ask-questions.md),
  [CT.7](CT.7-analytics-insights-and-gradebook.md).
