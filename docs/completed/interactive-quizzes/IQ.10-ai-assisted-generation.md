# IQ.10 — AI-Assisted Quiz Generation

> Completed implementation plan. Source: net-new capability (interactive game-based quizzing). Landscape: [interactive-quizzes/README](../../plan/interactive-quizzes/README.md) (active plans) / [completed index](README.md). Reuses the multi-provider AI stack ([AP.1–AP.9](../../plan/ai-providers/)), `aiprovidercreds`/`aiusage`, `systemprompts`, and the async generation-job pattern (`lessongenerationjobs`). Writes through the [IQ.2](IQ.2-kit-authoring-and-question-types.md) question-insertion contract.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IQ.10 |
| **Section** | Interactive Quizzes |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | SHIPPED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Assessment squad + AI |
| **Depends on** | IQ.2 |
| **Unblocks** | — |

---

## 1. Problem Statement

Authoring a good 15-question game by hand is the single biggest barrier to a teacher actually using live
quizzing on a busy day. IQ.10 removes it: from a topic, a pasted passage, a lesson/reading already in the
course, or a learning objective, the platform **drafts a full quiz kit** — well-formed questions, plausible
distractors, timers, and explanations — that the teacher reviews and tweaks. It reuses the platform's
multi-provider AI layer (so it honours org BYOK credentials, cost budgets, and PII redaction) and writes
through the same insertion/validation contract as hand authoring, so generated content is indistinguishable
from — and as safe as — human-made questions.

## 2. Goals

- Generate a draft kit (or add questions to an existing kit) from: a **topic/prompt**, a **pasted passage**,
  or **existing course content** (a lesson, reading, or file already in the course).
- Produce well-formed questions of the supported IQ.2 types with correct answers, **plausible distractors**,
  optional explanations, and suggested timers/difficulty — all editable.
- Run generation **asynchronously** as a job (like `lessongenerationjobs`) with progress, cancel, and
  provider/cost accounting via `aiusage`.
- Route through the **AP multi-provider** layer so it uses the org's configured provider/credentials, respects
  budgets, and applies PII redaction; degrade gracefully when AI is unavailable.
- Keep a **human in the loop**: nothing is hosted until the teacher reviews; generated questions pass IQ.2
  validation and IQ.9 moderation like any other.

## 3. Non-Goals

- New question types or the authoring UI itself (IQ.2) — IQ.10 fills that UI's data.
- Auto-hosting or auto-grading without review — always teacher-reviewed.
- Building AI infrastructure — IQ.10 consumes the existing AP provider abstraction, credentials, usage
  metering, and system-prompt registry.
- Image/media generation — text questions only at launch (media is attached by the teacher via IQ.2).

## 4. Personas & User Stories

- **As an instructor**, I want to type "photosynthesis, grade 8, 10 questions" and get a draft kit, so I can
  run a game in minutes.
- **As an instructor**, I want to paste a reading and generate comprehension questions from it, so the quiz
  matches what we studied.
- **As an instructor**, I want to generate questions from a lesson already in my course, so it's aligned.
- **As an instructor**, I want to review and edit every generated question before hosting, so I stay
  accountable for accuracy.
- **As an admin**, I want AI generation to use our provider/credentials and stay within budget, so costs are
  controlled.

## 5. Functional Requirements

- **FR-1.** The system MUST offer "Generate with AI" from the kit editor (IQ.2) with inputs: source
  (`topic` | `passage` | `course_content_ref`), target question **count**, **types** allowed, **difficulty**/
  grade band, **language**, and whether to include explanations.
- **FR-2.** Generation MUST run as an **async job** (`quizgame.generation_jobs`) with states
  `queued → running → succeeded | failed | canceled`, progress, and cancelation, reusing the
  `lessongenerationjobs` pattern.
- **FR-3.** The generator MUST call the AI provider **only** through the AP abstraction
  (`ai-providers`/`aiprovidercreds`), selecting the org/platform-configured provider+model and honouring the
  resolved credential; direct provider SDK calls are prohibited.
- **FR-4.** All AI usage MUST be metered via `aiusage` (tokens + estimated cost), attributed to the course/org,
  and MUST respect configured **budgets/quotas** (refuse or queue when exceeded).
- **FR-5.** Inputs sent to the model MUST pass **PII redaction**/`aidisclosure` policy; course-content sources
  MUST respect content permissions (only content the actor may access).
- **FR-6.** The model MUST return **structured** output (JSON matching the IQ.2 question schema); the server
  MUST validate it against that schema and **reject/repair** malformed items, never inserting invalid
  questions.
- **FR-7.** Generated questions MUST be written through the **IQ.2 insertion API** (same validation, same
  `source` = `ai_generated` provenance), so they behave identically to authored questions and can be freely
  edited.
- **FR-8.** Each generated question SHOULD include a **confidence/needs-review** hint and a rationale for the
  correct answer, surfaced to the teacher; low-confidence items are flagged for review.
- **FR-9.** The system MUST support **"generate more like this"** and **"regenerate this question"** for
  targeted iteration without redoing the whole kit.
- **FR-10.** Generated content MUST pass IQ.9 **moderation** (no unsafe content) before it can be hosted; the
  teacher review step is mandatory (no silent auto-publish to a live game).
- **FR-11.** When AI is disabled/unavailable/over budget, the editor MUST **degrade gracefully** — the manual
  authoring path (IQ.2) is unaffected and the AI entry point explains why it's unavailable.
- **FR-12.** A prompt/template for generation MUST live in the `systemprompts` registry (versioned,
  overridable per org), not hard-coded.

## 6. Non-Functional Requirements

- **Performance** — a 10-question generation completes typically < 30 s; the UI is non-blocking (job + poll/
  push); partial results stream where the provider supports it.
- **Security** — provider credentials never exposed client-side; jobs authorized to the course; output
  sanitised before insert.
- **Privacy & Compliance** — PII redaction on inputs; AI-generated content disclosed per `aidisclosure`;
  course-content access enforced; student data never sent as a generation source.
- **Accessibility** — generation UI is AA; generated media prompts still require teacher-supplied alt text
  (IQ.2) before "ready".
- **Scalability** — jobs queued via the existing job runner; concurrency and rate bounded per org budget.
- **Reliability** — provider failures retried with backoff/fallback (AP fallback path); malformed output
  repaired or the item dropped, never a broken kit; jobs are idempotent/resumable.
- **Observability** — per-job metrics: provider, model, tokens, cost, latency, success/repair/failure rates;
  usage dashboards via `aiusage`.
- **Maintainability** — one generation service; prompt in `systemprompts`; output schema shared with IQ.2.
- **Internationalization** — generate in the requested language; prompts localizable.
- **Backward compatibility** — additive; purely opt-in.

## 7. Acceptance Criteria

- **AC-1.** *Given* a topic + count + types, *when* the instructor generates, *then* a job runs and produces
  that many well-formed, editable questions of the requested types in the kit as drafts.
- **AC-2.** *Given* a pasted passage, *when* generating comprehension questions, *then* the questions and
  correct answers are grounded in the passage.
- **AC-3.** *Given* an org with BYOK credentials, *when* generation runs, *then* it uses that provider/model
  and the usage is metered to the org in `aiusage`.
- **AC-4.** *Given* the org is over its AI budget, *when* generation is attempted, *then* it is refused/queued
  with a clear message and no provider call is billed beyond policy.
- **AC-5.** *Given* the model returns malformed JSON, *when* the server parses it, *then* invalid items are
  repaired or dropped and only valid questions are inserted (no broken kit).
- **AC-6.** *Given* generated questions, *when* they appear, *then* each is marked `ai_generated`, shows a
  needs-review hint where confidence is low, and passes IQ.2 validation + IQ.9 moderation before hosting.
- **AC-7.** *Given* AI is disabled, *when* the editor loads, *then* manual authoring works unchanged and the AI
  button explains it's unavailable.
- **AC-8.** *Given* one weak question, *when* the instructor clicks "regenerate this question", *then* only
  that item is replaced.

## 8. Data Model

Migration `404_interactive_quizzes_ai_generation.sql` (renumbered from planned `399` — that sequence was consumed by IQ.5 scoring):

```sql
CREATE TABLE quizgame.generation_jobs (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kit_id       UUID NOT NULL REFERENCES quizgame.kits (id) ON DELETE CASCADE,
  course_id    UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
  requested_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
  source_type  TEXT NOT NULL,                 -- topic | passage | course_content_ref
  source_ref   JSONB NOT NULL DEFAULT '{}'::jsonb, -- topic text / passage / content id (never student data)
  params       JSONB NOT NULL DEFAULT '{}'::jsonb, -- count, types, difficulty, language, explanations
  status       TEXT NOT NULL DEFAULT 'queued',     -- queued|running|succeeded|failed|canceled
  provider     TEXT,                          -- resolved via AP layer
  model        TEXT,
  usage_id     UUID,                          -- link to aiusage record
  error        TEXT,
  result_summary JSONB,                       -- {inserted, repaired, dropped}
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ
);
CREATE INDEX idx_quizgame_genjobs_kit ON quizgame.generation_jobs (kit_id, created_at DESC);

-- provenance on questions (IQ.2 table): mark AI origin + review state
ALTER TABLE quizgame.questions
  ADD COLUMN IF NOT EXISTS source        TEXT NOT NULL DEFAULT 'authored', -- authored | ai_generated | bank_import
  ADD COLUMN IF NOT EXISTS needs_review  BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS generation_job_id UUID REFERENCES quizgame.generation_jobs (id) ON DELETE SET NULL;
```

- No student data is ever a generation source; `source_ref` holds topic/passage/course-content ids only.
- `usage_id` links to the existing `aiusage` metering row.

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| POST | `/live-quizzes/kits/{kit_id}/generate` `{sourceType, sourceRef, params}` | `item:create` (+ AI enabled/budget) |
| GET | `/live-quizzes/kits/{kit_id}/generate/{job_id}` | requester/instructor |
| POST | `/live-quizzes/kits/{kit_id}/generate/{job_id}/cancel` | requester |
| POST | `/live-quizzes/kits/{kit_id}/questions/{qid}/regenerate` | `item:create` |

- Generation writes questions via the internal IQ.2 insertion path (not a separate schema).
- **OpenAPI:** document job lifecycle, params, and provenance fields.
- **Rate-limit:** per-org concurrency + budget checks before enqueue.

## 10. UI / UX

- **"Generate with AI" panel** in the kit editor: source picker (topic / paste passage / pick course content),
  count, types, difficulty/grade, language, include-explanations; a "Generate" button that starts a job.
- **Job progress:** non-blocking progress with cancel; results drop into the question list as **draft /
  needs-review** items with an AI badge and rationale.
- **Per-question actions:** "regenerate this", "generate more like this", accept/edit.
- **Guardrail copy:** clear "AI-drafted — review before hosting" banner; disclosure per `aidisclosure`.
- **States:** AI-unavailable/over-budget (button disabled + reason), generating, partial results, failed
  (retry), all-reviewed.
- **Accessibility:** panel + progress are AA; generated items still require alt text before "ready".
- **Copy & i18n:** `liveQuiz.ai.*`, plus disclosure strings.

## 11. AI / ML Considerations

- **Model(s):** whatever the org/platform selects via the **AP multi-provider** layer (default to the latest
  capable Claude model where the platform default applies); never a hard-coded provider.
- **Prompt:** a versioned template in `systemprompts` that instructs structured JSON output matching the IQ.2
  question schema, with per-type rules (e.g. exactly one correct for `mc_single`, plausible distractors,
  grounded answers for passage sources), difficulty calibration, and language.
- **Structured output & validation:** request JSON; validate against the shared schema; **repair** (one bounded
  re-ask) or drop malformed items; never insert invalid questions.
- **Eval metric:** offline eval set scoring well-formedness, answer correctness (grounded sources), distractor
  plausibility, and reading-level fit; track acceptance/edit rate in production as a quality signal.
- **Fallback path:** AP provider fallback on error; if all providers fail or budget is exhausted, the job fails
  cleanly and manual authoring is unaffected.
- **PII redaction:** inputs pass the redaction policy; student data is never a source.
- **Cost budget:** metered via `aiusage`; per-org budgets enforced pre-enqueue; cost surfaced to admins.
- **Disclosure & accountability:** generated items are labelled `ai_generated`, disclosed per `aidisclosure`,
  and require human review before hosting.

## 12. Integration Points

- **Reuse:** `ai-providers` (AP.1–AP.9) provider abstraction + `aiprovidercreds` (org BYOK), `aiusage`
  (metering/budget), `systemprompts` (prompt registry), `aidisclosure` (labelling), `lessongenerationjobs`
  (async job pattern), the job runner, IQ.2 insertion/validation, IQ.9 moderation.
- **Server new:** `repos/quizgame/generation.go`, a generation worker in `background/`,
  `httpserver/quizgame_generation.go`.
- **Web new:** the "Generate with AI" editor panel + job progress.

## 13. Dependencies & Sequencing

- Must ship after: IQ.2 (insertion/validation contract to write through).
- Must ship before: nothing hard-depends.
- Shared infra: AP provider layer, AI credentials/usage/budget, system prompts, job runner.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Hallucinated / wrong "correct" answers | H | H | Mandatory human review; rationale + needs-review flags; grounded prompts for passage sources; eval set |
| Malformed model output breaks the kit | M | M | Schema validation + bounded repair; drop invalid items; never insert broken questions |
| Runaway AI cost | M | H | `aiusage` metering + per-org budgets enforced pre-enqueue; concurrency caps |
| Provider lock-in / outage | M | M | Route only through AP abstraction with fallback; degrade to manual authoring |
| Sensitive data sent to a provider | L | H | PII redaction; student data barred as source; content-permission checks |
| Over-reliance erodes item quality | M | M | Review-required UX; track acceptance/edit rate; teacher accountability messaging |

## 15. Rollout Plan

- **Flag:** `interactive_quizzes_enabled` + an `iq_ai_generation` sub-flag, and only where the platform/org AI
  is enabled and in budget.
- **Sequencing:** migration `404` → generation worker + AP wiring → editor panel → internal eval → pilot.
- **Dogfood:** generate kits across topics/passages/course content; measure well-formedness and edit rate;
  verify metering/budget.
- **GA criteria:** AC-1..AC-8 pass; eval thresholds met; cost controls verified; graceful degradation proven.
- **Rollback:** disable `iq_ai_generation`; manual authoring unaffected; jobs table retained.

## 16. Test Plan

- **Unit** — output-schema validation + repair/drop; budget/permission gating; provenance flags; prompt
  assembly.
- **Integration** — end-to-end job via a stubbed AP provider; malformed-output handling; over-budget refusal;
  regenerate-single.
- **End-to-end** — Playwright: generate from topic → review → edit → validate → host; AI-disabled degradation.
- **Security** — credentials never client-side; course-content permission enforcement; redaction on inputs.
- **Accessibility** — generation panel + progress axe/keyboard; generated items enforce alt text.
- **Quality (eval)** — offline eval set for well-formedness/correctness/distractors/reading level; track
  acceptance rate.
- **Manual** — spot-check factual accuracy across subjects; grounded-vs-topic comparison.

## 17. Documentation & Training

- Instructor: "Generate a quiz with AI" + "always review before hosting" guidance; grounding tips (paste a
  passage for on-topic questions).
- Admin: enabling AI generation; provider/credential/budget setup; disclosure policy.
- API reference: generation job endpoints + provenance fields.
- Runbook: prompt location/versioning, eval set, budget alerts, provider fallback behaviour.

## 18. Open Questions

1. Ground-truth policy: allow topic-only generation (ungrounded) at GA, or require a source passage/content for
   factual subjects? (Recommendation: allow topic-only with a stronger "review" nudge; encourage grounding.)
2. Auto-tagging subject/grade on generated kits (feeds IQ.8 discovery) — include now? (Recommendation: yes,
   cheap add via the same call.)
3. Media generation (images for prompts/answers) — in scope later? (Recommendation: defer; teacher-attached
   media via IQ.2 for now.)

## 19. References

- Existing files: `server/internal/repos/aiprovidercreds/`, `server/internal/repos/aiusage/`,
  `server/internal/repos/systemprompts/`, `server/internal/repos/aidisclosure/`,
  `server/internal/repos/lessongenerationjobs/`, `server/internal/repos/jobqueue/`,
  `server/internal/repos/quizgame/generation.go`, `server/internal/service/quizgameai/`,
  `server/internal/httpserver/quizgame_generation.go`,
  `clients/web/src/components/live-quiz/generate-with-ai-panel.tsx`.
- Related plans: [IQ.2](IQ.2-kit-authoring-and-question-types.md), [IQ.9](IQ.9-moderation-safety-accessibility.md),
  [AP.1–AP.9 (AI multi-provider)](../../plan/ai-providers/).
