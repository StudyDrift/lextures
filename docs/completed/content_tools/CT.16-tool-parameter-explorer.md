# CT.16 — Tool: Parameter Explorer (sliders drive a live model, with guided noticing)

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](../plan/content_tools/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.16 |
| **Section** | Content Tools (CT) — tool shelf |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform team |
| **Depends on** | CT.1, CT.2, CT.3 |
| **Unblocks** | Inquiry-style STEM, economics and statistics content; first CT.5 sandbox canary |

---

## 1. Problem Statement

"Notice what happens to the graph as you increase *r*" is one of the highest-leverage sentences in
maths and science teaching, and Lextures cannot render it. Authors have three bad options: a static
image (no manipulation), a linked external simulation (leaves the platform, no state, unknown privacy),
or a vibe-coded HTML activity (no typed state, not resettable, not analysable). What is missing is a
declarative way to say "here are the variables, here is the relationship, here is what I want them to
notice" — and to capture what the learner actually explored and concluded.

## 2. Goals

- Let an author declare parameters (sliders/toggles/choices), a model (formula or preset), and an
  output view (plot, value readout, table) with no code.
- Capture the learner's exploration as state: parameter values visited, key observations, and answers
  to the author's noticing prompts.
- Ask *guided noticing* questions tied to the exploration ("set r above 3.5 — what changes?"), with
  optional checkpoints that verify the learner actually reached a configuration.
- Make the whole thing keyboard- and screen-reader-operable, including a data-table view of the output.
- Be the first tool converted to the CT.5 iframe sandbox, proving the path for richer third-party tools.

## 3. Non-Goals

- A general programming environment (CT.17 is the code sandbox).
- Arbitrary physics engines or 3D — v1 covers 1–2 dimensional relationships and small discrete models.
- Importing external simulation formats (PhET, GeoGebra) — a linked-embed pattern is a separate story
  with its own privacy review.
- AI-generated models in v1.

## 4. Personas & User Stories

- **As a maths teacher**, I want students to manipulate the coefficients of a quadratic and see the
  curve change so that "coefficient" stops being a word and becomes a behaviour.
- **As a biology teacher**, I want a simple population model with a growth-rate slider so that students
  find the tipping point themselves.
- **As an economics instructor**, I want supply/demand curves that students shift so that equilibrium is
  discovered, not asserted.
- **As a student**, I want my exploration and my answers saved so that I can return to the reasoning.
- **As an instructor**, I want to see whether students actually explored the interesting region or just
  answered the questions.
- **As a blind student**, I want the output as a table and as announced values so that I can reason about
  the relationship too.

## 5. Functional Requirements

- **FR-1.** The author MUST be able to declare 1–6 **parameters**: numeric (min/max/step/default),
  boolean, or enumerated choice, each with a label, unit and description.
- **FR-2.** The author MUST be able to declare **outputs** as either (a) an expression over the
  parameters evaluated by a safe expression evaluator, or (b) a preset model from a small built-in
  library (linear, quadratic, exponential, logistic growth, projectile, supply/demand, normal
  distribution, compound interest).
- **FR-3.** Expressions MUST be evaluated in a **sandboxed, non-`eval`** evaluator supporting arithmetic,
  common maths functions and the declared parameters only — no host access, no network, no loops.
- **FR-4.** The output view MUST support: line/scatter plot over a swept variable, a scalar readout with
  units, and a data table — with the table always available (accessibility, FR-9).
- **FR-5.** The author MUST be able to add **noticing prompts**: free-text or choice questions rendered
  beside the model, whose answers are stored in state.
- **FR-6.** The author MAY add **checkpoints**: a predicate over parameters (e.g. `r > 3.5`) that must be
  reached before a prompt unlocks; reaching it MUST be recorded with a timestamp.
- **FR-7.** State MUST record the current parameter values, a bounded exploration trace (distinct
  configurations visited, capped and downsampled), checkpoint hits, and prompt answers.
- **FR-8.** Every parameter control MUST be operable by keyboard with arrow-key stepping, `Home`/`End`
  for bounds, and a numeric text input as an alternative to the slider.
- **FR-9.** The plot MUST have an equivalent accessible table and a text summary of the trend; values
  MUST be announced (throttled) as parameters change.
- **FR-10.** The tool MUST honour `prefers-reduced-motion` (no animated transitions between states).
- **FR-11.** Completion MUST be defined as answering all required prompts (and hitting all required
  checkpoints when configured).
- **FR-12.** The instructor MUST see: parameter-space coverage across the class (which regions were
  explored), checkpoint hit rate, and prompt answers.
- **FR-13.** The tool MUST render at `sandbox: 'iframe'` when the platform's sandbox mode requires it
  (CT.5), and identically in-process otherwise.
- **FR-14.** CT.4 reset MUST restore default parameters and clear trace, checkpoints and answers.

## 6. Non-Functional Requirements

- **Performance** — Parameter change → re-render ≤ 16 ms for ≤ 500 plotted points; expression evaluation
  ≤ 1 ms per point; renderer ≤ 40 KB gz (at the CT.5 budget ceiling; charting is hand-rolled SVG rather
  than a charting library for exactly this reason).
- **Security** — No `eval`; expression parser is an allowlisted AST interpreter with depth and step
  limits, fuzz-tested. No network from the tool.
- **Privacy & Compliance** — Exploration traces are student work (DSAR, retention). Free-text prompt
  answers pass the CT.8 filter.
- **Accessibility** — WCAG 2.1 AA: table equivalent for every plot, keyboard-operable controls, throttled
  live announcements, ≥ 3:1 contrast for plot lines and axes, no colour-only series distinction
  (patterns/markers), text trend summary.
- **Scalability** — Trace capped (default 200 distinct configurations, downsampled) inside the state cap;
  coverage aggregates computed in CT.7 as a coarse grid.
- **Reliability** — Parameter state autosaves as a draft; a failed save never blocks manipulation.
- **Observability** — `lextures_content_tool_explorer_checkpoints_total{outcome}`, time-in-tool
  histogram, coverage entropy (an engagement-quality signal).
- **Maintainability** — The preset model library is data, not code branches; adding a preset is a config
  entry plus tests.
- **Internationalization** — Number formatting per locale (including decimal separators in the numeric
  inputs), RTL layout, localized axis labels from author strings.
- **Backward compatibility** — Additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a declared quadratic model, *When* the learner moves the *a* slider, *Then* the plot
  updates within one frame and the readout matches an independent computation.
- **AC-2.** *Given* an expression containing a disallowed construct (function call to an unknown symbol,
  property access, loop), *When* the author saves, *Then* the editor rejects it with a clear message.
- **AC-3.** *Given* a checkpoint `r > 3.5`, *When* the learner reaches it, *Then* the dependent prompt
  unlocks, the hit is recorded with a timestamp, and it stays unlocked afterwards.
- **AC-4.** *Given* keyboard-only operation, *When* the learner adjusts every parameter and answers the
  prompts, *Then* the activity completes without pointer input.
- **AC-5.** *Given* a screen-reader user, *When* they open the table view, *Then* the plotted data is
  available as a semantic table with a text trend summary.
- **AC-6.** *Given* rapid slider movement, *Then* announcements are throttled (≤ 1 per 500 ms) and the
  trace is downsampled rather than storing every intermediate value.
- **AC-7.** *Given* 25 learners, *When* the instructor opens coverage, *Then* the explored-region map
  matches the aggregated traces and checkpoint hit rate is correct.
- **AC-8.** *Given* the platform requires sandbox mode, *Then* the tool renders inside the iframe with
  identical behaviour and its state saves through the bridge.
- **AC-9.** *Given* a CT.4 reset, *Then* parameters return to defaults and trace/answers are cleared.

## 8. Data Model

**No migration.**

```ts
// configSchema
type ParameterExplorerConfig = {
  prompt: string
  parameters: Array<
    | { id: string; kind: 'number'; label: string; unit?: string; min: number; max: number; step: number; default: number; description?: string }
    | { id: string; kind: 'boolean'; label: string; default: boolean; description?: string }
    | { id: string; kind: 'choice'; label: string; options: Array<{ value: string; label: string }>; default: string }
  >
  model:
    | { kind: 'preset'; preset: 'linear' | 'quadratic' | 'exponential' | 'logistic' | 'projectile' | 'supply_demand' | 'normal' | 'compound_interest'; bind: Record<string, string> }
    | { kind: 'expression'; expression: string; sweep: { paramId: string; from: number; to: number; points: number } }
  outputs: Array<{ kind: 'plot' | 'readout' | 'table'; label: string; yLabel?: string; xLabel?: string }>
  noticingPrompts?: Array<{
    id: string
    text: string
    kind: 'text' | 'choice'
    options?: Array<{ id: string; text: string }>
    required?: boolean
    unlockWhen?: string            // predicate over parameters, same safe evaluator
  }>
  requireAllCheckpoints?: boolean
}

// stateSchema
type ParameterExplorerState = {
  v: 1
  params: Record<string, number | boolean | string>
  trace: Array<{ at: string; params: Record<string, number | boolean | string> }>   // downsampled, capped
  checkpoints: Record<string, string>        // promptId → first-hit ISO timestamp
  answers: Record<string, string>
  completedAt?: string
}
```

`scoring.mode = 'none'` (optionally `manual` on prompts); `capabilities = ['state','aggregate']`;
`maxStateBytes = 48000`.

## 9. API Surface

**No new routes.**

- `PUT .../state` — parameter values, trace (client-computed, server-capped), answers.
- `POST .../actions/checkpoint` — records a checkpoint hit server-side (so unlocks cannot be forged).
- Coverage via CT.7 facets (binned parameter values, checkpoint ids).

The model is evaluated **client-side** by design: interaction must be instant, and the evaluator is
sandboxed and side-effect-free. Checkpoints are re-validated server-side against the submitted params,
so nothing that gates content depends on client honesty.

## 10. UI / UX

1. Prompt and a short "what to try" hint.
2. Controls column: labelled sliders with numeric inputs, toggles, choice selects, units, and a
   **Reset to defaults** action.
3. Output area: plot (SVG) with axes and gridlines, a readout strip of key values, and a **Table** tab.
4. Noticing prompts beneath, greyed with a lock icon and explanatory text until their checkpoint is hit.
5. Progress line: "2 of 3 questions answered."

**States** — *Default*, *Exploring (draft saved)*, *Checkpoint reached (announced)*, *Complete*,
*Read-only*, *Error*.

**Mobile** — controls above the plot; sliders full width with numeric inputs; table tab prominent.

**Accessibility** — every control has a visible label and value text; slider uses the native range
semantics with `aria-valuetext` including units; plot has `role="img"` with a text summary plus the
table tab; announcements throttled; focus never moves on parameter change.

**Copy & i18n** — `contentTools.tools.parameterExplorer.*`.

**Authoring** — custom editor: parameter builder, preset picker with live preview, expression editor
with validation and an inline preview plot, checkpoint predicate builder with a "test with these values"
helper.

## 11. AI / ML Considerations

None in v1. Reserved: (a) "generate a model from this paragraph" via `ui.aiAssist`, which would draft
parameters and an expression for author review; (b) AI feedback on free-text noticing answers, which
would reuse CT.20's rubric-feedback action rather than adding a second implementation. Neither ships in
v1 because the tool's value does not depend on them.

## 12. Integration Points

- **Internal** — `service/contenttools/tools/parameterexplorer/` (checkpoint validation, preset defs),
  shared `clients/web/src/lib/safe-expression/` evaluator (new, reusable),
  `clients/web/src/components/content-tools/tools/parameter-explorer/`,
  `service/contentfilter` (free-text answers).
- **CT.5** — first sandbox canary; its `resize` and `announce` bridge usage validates the protocol.
- **CT.7** — coverage and checkpoint facets.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1–CT.3.
- **Must ship before:** it is the recommended canary for CT.5's sandbox mode.
- **Shared infra needed:** none beyond the framework.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Expression evaluator becomes an injection vector | M | H | AST allowlist, no property access, no function definitions, depth/step limits, fuzz tests, no `eval`/`Function` |
| Plot accessibility is asserted but not real | M | H | Table equivalent is a hard FR with tests; trend summary; screen-reader validation before GA |
| Renderer exceeds the 40 KB budget | H | M | Hand-rolled SVG plotting, no charting dependency, budget enforced in CI |
| Trace storage grows unbounded | M | M | Downsampling + cap + server-side truncation |
| Authors write models that are subtly wrong | M | M | Live preview in the editor, preset library for common cases, "test values" helper |
| Sliders frustrating for fine values on touch | M | M | Paired numeric input always present; step tuning guidance |

## 15. Rollout Plan

- **Feature flag** — course tool allowlist; sandbox mode per CT.5.
- **Sequencing** — safe evaluator + tests → preset library → renderer + table view → checkpoints →
  authoring editor → coverage insights → sandbox conversion (CT.5 canary).
- **Dogfood** — an algebra unit (quadratics) and a biology unit (population growth).
- **GA criteria** — evaluator fuzz clean; a11y audit passed including table equivalence; bundle budget met.
- **Rollback** — remove from the allowlist.

## 16. Test Plan

- **Unit** — expression parsing/evaluation (valid, invalid, adversarial, precision edge cases); preset
  model correctness against reference values; checkpoint predicate evaluation; trace downsampling;
  locale number parsing.
- **Integration** — server-side checkpoint validation rejecting forged params; state caps; reset;
  filter on free text.
- **End-to-end** — Playwright: manipulate → checkpoint unlock → answer → reload persists; keyboard-only
  completion; table view equivalence; sandbox-mode run.
- **Security** — evaluator fuzzing (100k random and adversarial inputs); attempts at prototype access,
  infinite loops, huge exponents; forged checkpoint action calls.
- **Accessibility** — axe; screen-reader script for controls, announcements and table; contrast of plot
  elements; reduced motion; 400% zoom.
- **Performance** — 500-point sweep at 60 fps on a mid-range device; bundle size.
- **Manual exploratory** — extreme parameter ranges, NaN/Infinity handling, RTL, tiny screens.

## 17. Documentation & Training

- **Instructor** — designing a good exploration (what to vary, where the interesting region is);
  writing noticing prompts; using checkpoints without turning exploration into a maze.
- **Student** — keyboard controls and the table view.
- **Developer** — the safe expression evaluator contract; adding a preset model.

## 18. Open Questions

1. Should presets be extensible by third parties (a preset is close to a mini-tool)? Proposed: no —
   third parties should ship their own tool through CT.9 rather than injecting models here.
2. Should the exploration trace be visible to the student ("here's where you looked")? Proposed: yes as
   a small coverage strip; it is good metacognition and costs little.
3. Do we need two-parameter (surface/heat) plots in v1? Proposed: no; one swept variable plus discrete
   series covers the common cases.

## 19. References

- Existing files this work touches: `clients/web/src/components/content-tools/`,
  `server/internal/service/contenttools/`, math rendering (`clients/web/src/lib/math.ts`).
- Precedents: vibe activities (`course.module_vibe_activities`) as the unstructured predecessor this
  tool replaces for parameterised models.
- External standards: WCAG 2.1 AA (1.4.11 non-text contrast, 4.1.3 status messages); WAI-ARIA slider
  pattern.
- Related plans: [CT.5](CT.5-tool-sdk-sandboxing-and-versioning.md),
  [CT.17](../plan/content_tools/CT.17-tool-code-sandbox.md), [CT.7](CT.7-analytics-insights-and-gradebook.md).
