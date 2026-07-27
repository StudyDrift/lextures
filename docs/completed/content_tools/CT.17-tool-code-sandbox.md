# CT.17 — Tool: Code Sandbox (a runnable code cell with instructor tests)

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.17 |
| **Section** | Content Tools (CT) — tool shelf |
| **Severity** | MAJOR |
| **Markets** | HE / K12 (CS) / HS |
| **Status (today)** | SHIPPED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Assessment / platform team |
| **Depends on** | CT.1, CT.2, CT.3; shipped `service/codeexecution` |
| **Unblocks** | CS, data-science and quantitative-methods content taught inline |

---

## 1. Problem Statement

Programming is learned by running code, and Lextures can run code — `service/codeexecution` already
executes submissions against instructor test cases with time and memory limits. But that capability is
locked inside the assignment submission flow: a student reading a tutorial page about list
comprehensions cannot try one without leaving for an assignment. Every CS course therefore sends
students to an external playground, losing the state, the feedback and the evidence of practice. This
tool brings the shipped runner inline: an editable cell, a Run button, instructor tests, and per-learner
persistence of what they wrote and whether it passed.

## 2. Goals

- Put an editable, runnable code cell inside a content page, using the existing execution service.
- Support instructor-authored starter code, hidden and visible tests, and per-test feedback.
- Persist the learner's code, run history and test results per enrollment.
- Keep it low-stakes and formative by default, with an optional CT.7 grade bridge.
- Be honest about limits: this is practice, not a full IDE, and the constraints are visible up front.

## 3. Non-Goals

- Replacing coding assignments (multi-file projects, repos, autograders with rich rubrics).
- A terminal, package installation, network access, or long-running processes.
- Collaborative editing (the shipped collab-docs feature covers that need elsewhere).
- Languages beyond those the shipped runner supports; adding a language is a runner concern, not a tool
  concern.

## 4. Personas & User Stories

- **As a CS instructor**, I want a runnable example right under my explanation so that students try the
  idea in the same breath as reading it.
- **As a CS instructor**, I want a couple of tests so that "it works" is objective and immediate.
- **As a student**, I want to experiment with the example without breaking anything so that I can learn
  by poking at it.
- **As a student**, I want my code still there next session so that I can continue where I stopped.
- **As an instructor**, I want to see who is stuck on which failing test so that I can intervene early.
- **As a security engineer**, I want inline execution to inherit the same isolation as assignment
  execution so that this convenience adds no new attack surface.

## 5. Functional Requirements

- **FR-1.** The author MUST configure: language, starter code, an optional read-only prefix/suffix,
  and 0–10 test cases (input/expected output, hidden or visible, per-test feedback).
- **FR-2.** Execution MUST go through `service/codeexecution` via a CT.3 server action — never a
  client-side interpreter — inheriting its sandboxing, time limits and memory limits.
- **FR-3.** Hidden test inputs/expected outputs MUST be `x-lex-sensitive`; only pass/fail plus the
  author's feedback is returned for hidden tests.
- **FR-4.** The tool MUST support **Run** (execute with the visible sample input, show stdout/stderr)
  and **Check** (run all tests, show results).
- **FR-5.** Run and Check MUST be rate-limited per learner per instance (defaults: 30 runs/hour,
  20 checks/hour) on top of platform limits, with clear messaging on limit.
- **FR-6.** State MUST store the current code, the last N runs (default 10: timestamp, action, status,
  truncated output, tests passed) and the best result.
- **FR-7.** Output MUST be truncated server-side (default 8 KB) with an explicit "output truncated"
  marker; oversized output MUST NOT be stored.
- **FR-8.** The tool MUST report a score when tests exist (`passed/total`) for optional CT.7 bridging,
  and MUST support `scoring.mode = 'none'` when the author wants pure practice.
- **FR-9.** The editor MUST support syntax highlighting, auto-indent, bracket matching, and a
  **plain-textarea fallback** that is fully accessible and can be enabled by preference.
- **FR-10.** The editor MUST be keyboard-accessible including a documented, discoverable way to move
  focus out of the editor (`Esc` then `Tab`), and MUST not trap focus.
- **FR-11.** A **Reset code** action MUST restore the starter code without clearing run history; CT.4
  reset clears everything.
- **FR-12.** Compilation/runtime errors MUST be displayed verbatim (they are the learning material),
  with a plain-language hint when the author supplied one for that error pattern.
- **FR-13.** The instructor MUST see: pass rate per test, most-failed test, distribution of attempts,
  and the ability to open a learner's current code from the CT.4 roster.
- **FR-14.** The tool MUST render read-only with the learner's last code when the activity is archived or
  the platform's execution service is unavailable.

## 6. Non-Functional Requirements

- **Performance** — Editor mount ≤ 80 ms; run round-trip p95 ≤ 3 s (dominated by the runner); renderer
  ≤ 40 KB gz **excluding** the editor component, which is lazily loaded on first focus and shared across
  instances on a page.
- **Security** — All execution server-side in the existing isolated runner; no network from executed
  code; hidden tests never sent to the client; per-user rate limits; output size caps; code stored as
  data and never rendered as HTML.
- **Privacy & Compliance** — Code is student work (DSAR, retention, reset snapshots). No AI, no external
  egress in v1.
- **Accessibility** — WCAG 2.1 AA. Declared limitation: rich code editors are historically poor with
  screen readers, so the plain-textarea fallback is a first-class, discoverable option (not hidden), and
  results/errors are announced politely. Line numbers and errors are associated textually
  ("Line 4: SyntaxError…").
- **Scalability** — Execution capacity is the constraint; the tool inherits the runner's queue and
  applies its own per-instance limits to protect it. A page with several cells shares one editor bundle.
- **Reliability** — Idempotent check actions; runner unavailability degrades to a clear message with the
  code preserved; run history is append-only and capped.
- **Observability** — `lextures_content_tool_code_runs_total{language,action,status}`, runner latency
  histogram, rate-limit hits, most-failed-test gauge per instance.
- **Maintainability** — Zero new execution code; the tool is a thin adapter over `service/codeexecution`.
- **Internationalization** — UI localized; compiler output is passed through untranslated (correct
  behaviour); RTL layout keeps code LTR.
- **Backward compatibility** — Additive; no change to coding assignments.

## 7. Acceptance Criteria

- **AC-1.** *Given* a learner presses Run, *Then* the code executes in the shipped runner and stdout,
  stderr and exit status are displayed and stored (truncated per cap).
- **AC-2.** *Given* hidden tests, *When* the learner checks, *Then* the payload contains pass/fail and
  feedback but no hidden inputs or expected outputs.
- **AC-3.** *Given* an infinite loop, *Then* the run terminates at the runner's time limit and the tool
  shows a clear "took too long" state without freezing the page.
- **AC-4.** *Given* the run rate limit is reached, *Then* the API refuses with a typed error naming the
  reset time and no execution occurs.
- **AC-5.** *Given* a learner returns the next day, *Then* their code and run history are restored.
- **AC-6.** *Given* Reset code, *Then* the editor returns to the starter code and run history is intact;
  *Given* a CT.4 reset, *Then* both are cleared and snapshotted.
- **AC-7.** *Given* 4 of 5 tests pass, *Then* the reported score is 4/5 and, when bridged, the gradebook
  reflects it.
- **AC-8.** *Given* the plain-textarea fallback is enabled, *Then* the full activity — edit, run, check,
  read results — is completable with a screen reader.
- **AC-9.** *Given* output exceeding the cap, *Then* it is truncated with a marker and the stored state
  stays within `maxStateBytes`.
- **AC-10.** *Given* the execution service is down, *Then* the tool shows a clear unavailable state, the
  code remains editable and saved, and no state corruption occurs.

## 8. Data Model

**No migration.** Execution reuses the shipped `service/codeexecution` types.

```ts
// configSchema
type CodeSandboxConfig = {
  language: string                         // constrained to runner-supported languages
  prompt: string
  starterCode: string
  prefixCode?: string                      // read-only, prepended at run time
  suffixCode?: string                      // read-only, appended at run time
  sampleInput?: string
  tests?: Array<{
    id: string
    name: string
    input: string                          // x-lex-sensitive when hidden
    expectedOutput: string                 // x-lex-sensitive when hidden
    hidden: boolean
    feedback?: string                      // x-lex-sensitive
  }>
  runLimitPerHour: number                  // default 30
  checkLimitPerHour: number                // default 20
  editorMode: 'rich' | 'plain' | 'user_choice'   // default 'user_choice'
  errorHints?: Array<{ match: string; hint: string }>
}

// stateSchema
type CodeSandboxState = {
  v: 1
  code: string
  runs: Array<{
    at: string
    action: 'run' | 'check'
    status: 'ok' | 'compile_error' | 'runtime_error' | 'timeout' | 'memory' | 'error'
    stdout?: string                        // truncated
    stderr?: string                        // truncated
    tests?: Array<{ id: string; passed: boolean }>
  }>
  best?: { passed: number; total: number; at: string }
  completedAt?: string
}
```

`scoring.mode = 'auto'` when tests exist, else `'none'`; `capabilities = ['state','scoring','code_execution']`;
`storage.maxStateBytes = 128000` (the documented exception to the 64 KB default — CT.1 open question 4
is resolved here, and CT.5 records the exception in the manifest).

## 9. API Surface

**No new routes.**

- `PUT .../state` — code drafts (debounced).
- `POST .../actions/run` — `{code, stdin?, idempotencyKey}` → `{status, stdout, stderr, state}`.
- `POST .../actions/check` — `{code, idempotencyKey}` → `{tests: [{id, name, passed, feedback?}], passed, total, state}`.
- Instructor test-level stats via CT.7 facets `testId`, `passed`.

## 10. UI / UX

1. Prompt (Markdown), then the editor with language badge and a line-count indicator.
2. Toolbar: **Run**, **Check** (when tests exist), **Reset code**, editor-mode toggle, and remaining-runs
   counter.
3. Output panel below with tabs: *Output* (stdout/stderr) and *Tests* (per-test rows with ✓/✗, name and
   feedback; hidden tests show name and result only).
4. Footer: last run time, best result, and a note on limits ("no network, 5 s limit").

**States** — *Idle*, *Running* (spinner + cancel where the runner supports it), *Success*, *Failed tests*,
*Compile/runtime error*, *Timeout*, *Rate limited*, *Runner unavailable*, *Read-only*.

**Mobile** — editor with a horizontal scroll and a compact toolbar; a "code keyboard" row of common
symbols; output panel collapsible. Documented as usable but not the recommended surface for long code.

**Accessibility** — editor-mode toggle is prominent; `Esc`-then-`Tab` escape documented in an in-tool
help affordance and announced on first focus; results announced politely ("3 of 5 tests passed");
errors associated with line numbers in text; no colour-only pass/fail.

**Copy & i18n** — `contentTools.tools.codeSandbox.*`.

**Authoring** — custom editor: language picker, starter/prefix/suffix fields, test-case table with
hidden toggles and feedback, plus **Try it** that runs the author's own reference solution against the
tests to prove they pass before publishing.

## 11. AI / ML Considerations

None in v1 — deliberately, because AI code help inside a practice cell is exactly where "AI does the
learning" risk is highest. Reserved for a follow-up with instructor control: an **explain this error**
action (error text + code, no solution) and a **hint, don't solve** action following CT.10's
`hint_only` stance, both disclosed and per-course toggleable, and both off by default. The
shipped `service/hintservice` scaffolding model is the intended pattern.

## 12. Integration Points

- **Internal** — `service/contenttools/tools/codesandbox/` (adapter), `service/codeexecution`
  (runner, unchanged), `internal/ratelimit`,
  `clients/web/src/components/content-tools/tools/code-sandbox/`, shared lazy editor component.
- **CT.7** — test-level facets and optional grade bridge.
- **Coding assignments** — shares the runner; documentation explains when to use which.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1–CT.3; requires `service/codeexecution` capacity review.
- **Must ship before:** nothing.
- **Shared infra needed:** code execution runner capacity.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Inline runs exhaust execution capacity | H | H | Per-instance and per-user rate limits, capacity review before GA, queue backpressure surfaced as a friendly wait state, per-course caps |
| Rich editor is inaccessible | H | H | Plain-textarea mode as a first-class, discoverable option; declared limitation; screen-reader validation |
| Students paste solutions from elsewhere | H | L | It is formative practice; instructor visibility of code; not a graded integrity surface by default |
| Hidden test data leaking | M | H | Sensitive-field redaction with an explicit payload test |
| Large outputs bloat state | M | M | Server-side truncation before storage; capped run history |
| Bundle weight from the editor | H | M | Lazy load on first focus, shared across instances, plain mode needs no bundle at all |

## 15. Rollout Plan

- **Feature flag** — course tool allowlist, plus an org capability switch (`code_execution`, CT.8) so
  districts without CS needs can deny it wholesale.
- **Sequencing** — adapter + rate limits → renderer (plain first, rich second) → tests panel → authoring
  editor with reference-solution check → capacity review → pilot in one CS course.
- **Dogfood** — an intro-programming unit with 8 cells across 4 pages.
- **GA criteria** — capacity headroom verified under class-sized load; a11y validated in plain mode;
  zero hidden-test leakage.
- **Rollback** — remove from the allowlist or deny the `code_execution` capability org-wide.

## 16. Test Plan

- **Unit** — adapter mapping to runner types; truncation; rate-limit accounting; score computation;
  error-hint matching.
- **Integration** — run/check against the real runner in CI (per supported language); timeout and memory
  paths; hidden-test redaction; state caps; reset semantics (code reset vs CT.4 reset).
- **End-to-end** — Playwright: edit → run → fail tests → fix → pass → reload persists; rate-limit
  message; runner-down state.
- **Security** — payload inspection for hidden tests; attempts to execute for another enrollment;
  oversized code and output; verification that executed code has no network access.
- **Accessibility** — axe in both editor modes; screen-reader script for the full loop in plain mode;
  focus-escape verification in rich mode.
- **Performance / load** — 40 concurrent learners checking in one class; editor mount time; bundle budget.
- **Manual exploratory** — very long output, unicode output, non-terminating programs, mobile editing.

## 17. Documentation & Training

- **Instructor** — authoring cells and tests; the reference-solution check; when to use a cell vs a
  coding assignment; rate limits.
- **Student** — what the sandbox can and cannot do (no network, time limits), how to use plain mode.
- **Admin** — the `code_execution` capability and its capacity implications.
- **Runbook** — runner saturation, disabling the tool org-wide, investigating a disputed test result.

## 18. Open Questions

1. Should learners be able to add their own scratch cells on a page (not author-placed)? Proposed: no in
   v1 — it turns a content page into a notebook, which is the shipped notebook's job.
2. Should `stdin` be author-configurable per test only, or learner-editable for exploration? Proposed:
   both, with a sample input the learner may edit for Run but not for Check.
3. Is 128 KB the right state cap for code plus history? Proposed: yes with history capped at 10 runs;
   revisit if data-science cells with long outputs become common.

## 19. References

- Existing files this work touches: `server/internal/service/codeexecution/runner.go` and `types.go`,
  `server/internal/ratelimit/`, `clients/web/src/components/content-tools/`.
- Precedents: coding-assignment execution flow; `service/hintservice` for the reserved hint pattern.
- External standards: WCAG 2.1 AA (2.1.2 No Keyboard Trap, 4.1.3 Status Messages).
- Related plans: [CT.7](CT.7-analytics-insights-and-gradebook.md),
  [CT.8](CT.8-governance-safety-privacy-accessibility.md),
  [CT.18](CT.18-tool-step-through-worked-example.md).
