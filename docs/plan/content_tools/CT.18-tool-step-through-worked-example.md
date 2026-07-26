# CT.18 — Tool: Step-Through Worked Example (one step at a time, checked, with faded scaffolding)

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.18 |
| **Section** | Content Tools (CT) — tool shelf |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Assessment team |
| **Depends on** | CT.1, CT.2, CT.3; shipped `service/hintservice` |
| **Unblocks** | Procedural fluency in maths, physics, chemistry, accounting, statistics |

---

## 1. Problem Statement

A worked example printed in a page is read, nodded at, and not learned — the learner never has to
produce a step, so the "I understood that" feeling is unearned. What works instead is *completion
practice*: show the first steps, make the learner produce the next one, check it, hint if needed, then
fade the scaffolding across examples. Lextures ships hints and worked examples
(`course.question_hints`, `course.question_worked_examples`) — but only attached to quiz questions,
revealed after a wrong answer. There is no way to place a step-by-step derivation inline and have the
learner drive it. This tool does that: the derivation becomes an activity, not a display.

## 2. Goals

- Let an author write a multi-step solution where any subset of steps is *blanked* for the learner.
- Check each step independently, with tolerance appropriate to the step type (numeric, algebraic,
  choice, text).
- Provide layered hints per step, reusing the shipped hint-scaffolding semantics (hint use is recorded,
  not punished).
- Support **fading**: across the instance's steps (or across instances via config), the proportion of
  blanked steps increases.
- Show instructors exactly which step the class breaks on — the single most actionable datum in
  procedural teaching.

## 3. Non-Goals

- A computer-algebra system. Algebraic equivalence checking is limited to a documented, safe subset
  (normalised polynomial/rational comparison over declared variables); anything beyond that is an
  author-supplied accepted-answers list.
- Free-form proof checking.
- Replacing quiz hints (they stay; this tool reuses the model, not the UI).
- AI-generated steps in v1.

## 4. Personas & User Stories

- **As a maths teacher**, I want students to fill in the missing algebra steps so that they practise the
  move, not just watch it.
- **As a chemistry teacher**, I want each stoichiometry step checked so that an early error does not
  silently poison the whole calculation.
- **As a student**, I want a hint when I am stuck rather than the answer so that I still do the thinking.
- **As a student**, I want to see the correct step after I have tried so that I can compare my reasoning.
- **As an instructor**, I want to know that 60% of the class failed step 3 so that I reteach step 3.
- **As a student who has mastered it**, I want the option to reveal the whole solution so that I am not
  forced through busywork.

## 5. Functional Requirements

- **FR-1.** The author MUST define an ordered list of steps, each with: displayed text (Markdown +
  math), an optional blank with an expected answer, a step type, and optional per-step hints.
- **FR-2.** Step types MUST include `numeric` (with tolerance), `expression` (algebraic equivalence over
  a declared variable set), `choice` (pick the correct next move), and `text` (accepted answers).
- **FR-3.** Expected answers, hints and explanations MUST be `x-lex-sensitive`; checking MUST be a server
  action.
- **FR-4.** Steps MUST be revealed sequentially: a step's content is available only after the previous
  blanked step is answered or skipped (configurable to "all visible").
- **FR-5.** Each step MUST support 1–3 hints, revealed on request in order; hint use MUST be recorded in
  state and MUST NOT reduce a reported score by default (configurable).
- **FR-6.** After a configurable number of attempts (default 3) on a step, the tool MUST offer
  **Show me this step** with the author's explanation, and MUST continue to the next step.
- **FR-7.** The author MUST configure **fading**: `blankPolicy` of `author` (explicit per step),
  `progressive` (blank an increasing share of later steps), or `all`.
- **FR-8.** State MUST record per step: attempts (with values), hints used, whether it was revealed, and
  timing.
- **FR-9.** The tool MUST report a score (steps correct without reveal / total blanked steps) for
  optional CT.7 bridging, with a `practiceOnly` mode that reports no score at all.
- **FR-10.** Algebraic checking MUST be implemented by a safe normaliser (expand, collect, canonical
  ordering) over a declared variable set, with a documented list of supported constructs and an explicit
  failure mode: when it cannot decide, it falls back to the author's accepted-answer list and, failing
  that, marks the step *needs review* rather than wrong.
- **FR-11.** Math input MUST be enterable in plain text (`x^2 + 3x`) and rendered as KaTeX live, with a
  visible preview so learners can confirm what the system read.
- **FR-12.** The instructor MUST see a **step funnel**: attempts, success rate, hint usage and reveal
  rate per step, with the breaking step highlighted.
- **FR-13.** CT.4 reset MUST clear all step progress.

## 6. Non-Functional Requirements

- **Performance** — Step check p95 ≤ 200 ms including expression normalisation; renderer ≤ 36 KB gz
  (KaTeX is already loaded by the reader for math content).
- **Security** — Answers/hints server-side only; the expression normaliser is a bounded AST operation
  with depth/size limits, fuzz-tested, no `eval`.
- **Privacy & Compliance** — Step answers are student work; no AI, no egress in v1.
- **Accessibility** — WCAG 2.1 AA. Math input is a labelled text field with a live-rendered preview and
  an accessible text description of the rendered expression; step reveals are announced; hint requests
  do not steal focus; correctness never colour-only. Documented limitation: KaTeX output relies on MathML
  where available; the plain-text source is always exposed as the accessible value.
- **Scalability** — Small state; funnel aggregates from CT.7 facets.
- **Reliability** — Idempotent checks; draft input autosaved per step.
- **Observability** — `lextures_content_tool_step_checks_total{result}`, hint-usage and reveal-rate
  gauges per step, normaliser-undecidable rate (an authoring-quality signal).
- **Maintainability** — The normaliser is a standalone tested module shared with any future math tool.
- **Internationalization** — Locale decimal separators accepted in numeric steps; UI localized; math
  notation left as authored.
- **Backward compatibility** — Additive; quiz hints unchanged.

## 7. Acceptance Criteria

- **AC-1.** *Given* a blanked step, *When* the payload is inspected, *Then* neither the expected answer
  nor unrevealed hints are present.
- **AC-2.** *Given* a learner enters `3x + 6` where `3(x+2)` is expected, *Then* the expression step is
  marked correct by normalisation.
- **AC-3.** *Given* an expression the normaliser cannot decide, *Then* the step is marked *needs review*
  (not wrong), the learner may continue, and the instructor sees it flagged.
- **AC-4.** *Given* three failed attempts, *Then* **Show me this step** appears with the explanation and
  the learner may proceed; state records the reveal.
- **AC-5.** *Given* hints are requested, *Then* they appear in order, are recorded, and (by default) do
  not reduce the score.
- **AC-6.** *Given* `blankPolicy = 'progressive'`, *Then* later steps are blanked at an increasing rate
  per the configured curve, deterministically per enrollment.
- **AC-7.** *Given* 30 learners, *When* the instructor opens the funnel, *Then* per-step success, hint
  and reveal rates match raw state and the breaking step is highlighted.
- **AC-8.** *Given* a numeric step with `de-DE` locale input `3,14`, *Then* it parses as 3.14.
- **AC-9.** *Given* keyboard-only use with a screen reader, *Then* the learner can enter, preview, submit
  and hear the result of every step.
- **AC-10.** *Given* a CT.4 reset, *Then* all steps return to unanswered and prior work is snapshotted.

## 8. Data Model

**No migration.**

```ts
// configSchema
type WorkedExampleConfig = {
  title?: string
  problem: string                          // markdown + math
  variables?: string[]                     // declared symbols for expression checking
  steps: Array<{
    id: string
    label?: string                         // "Step 2 — distribute"
    text: string                           // shown before/around the blank
    blank?: {
      type: 'numeric' | 'expression' | 'choice' | 'text'
      expected?: string | number           // x-lex-sensitive
      tolerance?: { kind: 'absolute' | 'relative'; value: number }
      acceptedAnswers?: string[]           // x-lex-sensitive
      options?: Array<{ id: string; text: string }>
      correctOptionId?: string             // x-lex-sensitive
      unit?: string
    }
    hints?: string[]                       // x-lex-sensitive, revealed in order
    explanation?: string                   // x-lex-sensitive until reveal
  }>
  blankPolicy: 'author' | 'progressive' | 'all'   // default 'author'
  attemptsPerStep: number                  // default 3
  hintsAffectScore: boolean                // default false
  practiceOnly: boolean                    // default true (no score reported)
  showAllSteps: boolean                    // default false (sequential reveal)
}

// stateSchema
type WorkedExampleState = {
  v: 1
  steps: Record<string, {
    attempts: Array<{ value: string; result: 'correct' | 'incorrect' | 'needs_review'; at: string }>
    hintsUsed: number
    revealed: boolean
    completedAt?: string
  }>
  currentStepId?: string
  scoreRaw?: number
  scoreMax?: number
  completedAt?: string
}
```

`scoring.mode = 'auto'` unless `practiceOnly`; `capabilities = ['state','scoring']`;
`maxStateBytes = 32000`.

## 9. API Surface

**No new routes.**

- `POST .../actions/checkStep` — `{stepId, value, idempotencyKey}` →
  `{result, feedback?, attemptsRemaining, nextStep?, state}`.
- `POST .../actions/hint` — `{stepId}` → `{hint, hintsRemaining, state}`.
- `POST .../actions/revealStep` — `{stepId}` → `{explanation, expectedDisplay, state}`.
- `PUT .../state` — draft input per step.
- Funnel via CT.7 facets `stepId`, `result`, `hintsUsed`, `revealed`.

## 10. UI / UX

1. Problem statement at the top (Markdown + math), pinned while scrolling within the tool.
2. Steps as a vertical list: completed steps show the learner's answer and the correct form; the current
   step shows its text with an inline input; future steps are hidden or dimmed per config.
3. Current-step controls: input with live math preview, **Check**, **Hint** (with remaining count),
   and — after exhausted attempts — **Show me this step**.
4. Footer: progress ("Step 3 of 6"), hints used, and an optional **Reveal full solution** when the author
   allows it.

**States** — *Not started*, *In progress*, *Step correct* (advance animation, reduced-motion aware),
*Step incorrect (attempts left)*, *Step revealed*, *Needs review*, *Complete*, *Read-only*, *Error*.

**Mobile** — one step in view at a time with a compact stepper; math preview above the keyboard.

**Accessibility** — each step is a labelled group with the step number in its accessible name; the math
preview exposes the plain-text source as its accessible value; results and hint text announced politely;
focus moves to the next step's input on success (announced), never silently.

**Copy & i18n** — `contentTools.tools.workedExample.*`.

**Authoring** — custom editor: step list with drag reordering, per-step type and expected answer, hint
rows, an "auto-blank later steps" fading control, and a **verify** action that runs the author's own
expected answers through the checker to catch typos before publishing.

## 11. AI / ML Considerations

None in v1. Reserved, in priority order: (a) **generate hints** for a step the author wrote without any,
reviewed before saving; (b) **diagnose the error** — classify a wrong step against the shipped
misconception library and surface a targeted hint; (c) generate a parallel practice problem with the
same structure. All three would run through `aigateway`, be author-reviewed, and stay off by default,
because a wrong hint is worse than no hint.

## 12. Integration Points

- **Internal** — `service/contenttools/tools/workedexample/` (checking, normaliser),
  new shared `server/internal/service/mathnorm/` (expression normalisation, reusable),
  `service/hintservice` (hint semantics and recording conventions),
  `service/misconception` (reserved), `clients/web/src/lib/math.ts` (KaTeX),
  `clients/web/src/components/content-tools/tools/worked-example/`.
- **CT.7** — step funnel facets, optional grade bridge.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1–CT.3.
- **Must ship before:** nothing.
- **Shared infra needed:** none beyond the framework.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Equivalence checking marks correct work wrong | H | H | Conservative normaliser, accepted-answers fallback, *needs review* instead of wrong when undecidable, instructor flag, author verify tool |
| Authors write steps that are ambiguous to answer | H | M | Verify action, preview, guidance to prefer numeric/choice steps where phrasing is loose |
| Math text input frustrates learners | H | M | Live preview showing what was parsed, forgiving syntax (`^`, `*` optional), symbol palette, plain-text always accepted |
| Sequential reveal feels like a maze | M | M | Configurable all-visible mode; reveal-full-solution escape hatch |
| Hint usage stigmatised | M | M | Hints do not reduce score by default; framing copy; instructor sees usage as information, not misconduct |
| Normaliser as an attack surface | L | H | Bounded AST, depth/size limits, fuzzing, no `eval` |

## 15. Rollout Plan

- **Feature flag** — course tool allowlist.
- **Sequencing** — normaliser + fuzz tests → checking actions → renderer with math preview → hints and
  reveal → fading → authoring editor with verify → funnel insights → pilot.
- **Dogfood** — an algebra unit and a stoichiometry unit (different step shapes).
- **GA criteria** — false-negative rate on a 200-item expression corpus < 1%; a11y audit passed;
  authoring verify catches seeded typos.
- **Rollback** — remove from the allowlist.

## 16. Test Plan

- **Unit** — normaliser corpus (equivalent/non-equivalent pairs, edge cases, undecidable cases);
  numeric tolerance and locale parsing; attempt/hint/reveal accounting; fading determinism; scoring.
- **Integration** — key/hint redaction; sequential gating enforced server-side; reset; grade bridge.
- **End-to-end** — Playwright: full derivation with a wrong step, hint, retry, reveal; all-visible mode;
  keyboard-only completion.
- **Security** — payload inspection; forged step ids; normaliser fuzzing; oversized input.
- **Accessibility** — axe; screen-reader script covering input, preview, result and advance; focus
  behaviour on step change; reduced motion.
- **Performance** — check latency including normalisation; chunk budget.
- **Manual exploratory** — chemistry units with units-of-measure, long derivations on mobile, RTL.

## 17. Documentation & Training

- **Instructor** — designing completion problems; which step types to choose; using fading across a
  unit; interpreting the funnel; the verify action.
- **Student** — how to type maths; that hints are free.
- **Developer** — the normaliser's supported subset and failure semantics.

## 18. Open Questions

1. Should *needs review* steps queue for instructor adjudication, or just be flagged? Proposed: flagged
   in the funnel with a one-click "accept as correct" that updates the learner's state.
2. Should fading persist **across** instances in a unit (problem 1 mostly worked, problem 4 mostly
   blank)? Proposed: yes as a follow-up requiring a unit-level grouping concept.
3. How much of a CAS is worth building? Proposed: none — stop at polynomial/rational normalisation and
   lean on accepted answers; revisit only with evidence.

## 19. References

- Existing files this work touches: `server/migrations/095_hints_scaffolding.sql` (hint model),
  `server/internal/service/hintservice/`, `clients/web/src/lib/math.ts`,
  `clients/web/src/components/content-tools/`.
- External standards: WCAG 2.1 AA; MathML accessibility guidance; learning-science basis — worked-example
  effect, completion problems, expertise-reversal (Sweller, Renkl).
- Related plans: [CT.11](CT.11-tool-inline-questions.md), [CT.17](CT.17-tool-code-sandbox.md),
  [CT.7](CT.7-analytics-insights-and-gradebook.md).
