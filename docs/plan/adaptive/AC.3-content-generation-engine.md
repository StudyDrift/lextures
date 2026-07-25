# AC.3 — Adaptive Content Generation Engine

> Implementation plan. Source: extends the shipped AI stack (`aiprovider` / `aigateway` / `contentpagegeneration`). Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AC.3 |
| **Section** | Adaptive Content Engine (ACE) |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | AI/ML platform team |
| **Depends on** | AC.1, AC.2; reuses 10.17 (AI disclosure/gateway) |
| **Unblocks** | AC.4, AC.5, AC.6, AC.7 |

---

## 1. Problem Statement

AC.2 produces a structured adaptation profile per learner; nothing yet turns it into actual teaching content. This story is the engine: given a unit's base content page and a profile, produce a **content variant** that rewrites or re-emphasizes the material for that learner — introduce the topic from scratch, reinforce weak spots with worked examples, compress for a near-master, or directly confront a detected misconception — while **provably preserving the factual and standards-aligned meaning** of the original. Without a fidelity guarantee, "adaptive content" is just a hallucination risk bolted onto the syllabus; the engine's central job is to make generation *safe*, not merely possible.

## 2. Goals

- Generate a per-profile content variant from `(base content, adaptation profile, allowed axes)`, keyed by `profile_signature` so identical learners share one artifact.
- Implement the four emphasis modes (introduce / reinforce / compress / remediate) plus stackable axes (scaffolding, reading level, misconception-targeting, modality hint).
- Enforce a **content-fidelity check** (semantic equivalence + no new factual claims) and a **safety check** before a variant is eligible to serve.
- Route every call through `aigateway` for disclosure/consent/COPPA gating and log tokens/cost to `analytics.ai_usage_log` under feature `adaptive_content`.
- Produce variants in the same block/markdown shape as `module_content_pages` so the existing renderer displays them unchanged.

## 3. Non-Goals

- Async pre-warming, cache eviction, and per-course budget *enforcement* (AC.4 owns the pipeline; AC.3 defines the cache *key* and cost *accounting*).
- Instructor approval workflow UI (AC.5).
- Serving variants to students / holdout logic (AC.6).
- Choosing the profile (AC.2) or measuring lift (AC.7).

## 4. Personas & User Stories

- **As a struggling student**, I want the chapter rewritten to start from what I *do* know, with an extra worked example on the part I missed.
- **As a near-master student**, I want a tight summary that skips what I already proved I know so I can move on.
- **As a student with a specific misconception**, I want the content to name and correct that misconception, not restate the generic lesson.
- **As an instructor**, I want an ironclad guarantee the AI never changes the facts, the standard, or the required terminology of my lesson.
- **As a compliance owner**, I want every generation call disclosed, budgeted, logged, and reproducible.

## 5. Functional Requirements

- **FR-1.** The system MUST expose `adaptivecontent.GenerateVariant(ctx, client, unit, baseContent, profile, allowedAxes) (Variant, CallMeta, error)` producing block/markdown content in the `module_content_pages` shape.
- **FR-2.** The generation prompt MUST encode: the base content (authoritative source), the emphasis mode, the concept gaps and misconceptions (by name/description from the concept + misconception libraries), the target Bloom level, the reading-level/modality preferences, and the allowed axes — and MUST instruct the model to **preserve all facts, definitions, standards references, and required terminology** from the base content and add nothing not entailed by it.
- **FR-3.** The system MUST run a **fidelity gate** after generation: (a) an automated semantic-fidelity score (embedding similarity of claims + a rubric-scored LLM-judge "does the variant introduce any claim not supported by the base?") producing `fidelity_score` ∈ [0,1]; (b) hard checks that all base **key terms** (instructor-marked glossary/standard terms) and any LaTeX/math and code blocks survive intact.
- **FR-4.** A variant MUST NOT reach `approved`/`auto_served` status if `fidelity_score < course.min_fidelity` (default 0.85) or any hard check fails; such variants are stored with `status='rejected'` and a `safety_flags` reason for audit, and the unit falls back to base content downstream.
- **FR-5.** The system MUST run a **safety check** (reuse existing moderation path) for age-appropriateness and prohibited content; failures set `status='rejected'` with a flag.
- **FR-6.** All calls MUST pass through `aigateway.Evaluate` (feature `adaptive_content`) and MUST be blocked/fallback when the gateway denies (opt-out, COPPA, tenant); every call (success or failure) MUST log to `analytics.ai_usage_log` with `feature='adaptive_content'`, `course_id`, and token counts.
- **FR-7.** The system MUST be deterministic-enough for caching: given the same `(unit content version, profile_signature, model, prompt version)` it MUST reuse the stored `content_variants` row rather than regenerate (one variant per `UNIQUE(unit_id, profile_signature)`); a base-content edit bumps a `content_version` that invalidates variants.
- **FR-8.** The system MUST record `axes_applied`, `model`, `prompt_version`, `fidelity_score`, `safety_flags`, and token/cost on each `content_variants` row.
- **FR-9.** The system SHOULD support a **max output size** and MUST truncate/normalize to the same limits as `contentpagegeneration` (headings/sections/markdown rune caps).
- **FR-10.** The system MUST support a `neutral`/`base` signature short-circuit: never call the model for a neutral profile — return the base content directly.

## 6. Non-Functional Requirements

- **Performance** — Generation is async-first (AC.4 queues it); synchronous instructor "preview" p95 ≤ 8 s per variant. Cache hit (existing variant) p95 ≤ 30 ms.
- **Security** — Prompts include only course content + non-PII profile enums/ids (never student name/email). Model output is sanitized (block editor allow-list) before storage; no script/HTML injection into the renderer.
- **Privacy & Compliance** — Feature is disclosed via `aidisclosure`; COPPA/GDPR gating via `aigateway`; content variants are education records (retention S02, DSAR S01). PII redaction: profile inputs carry no PII by construction (AC.2). AI-Act high-risk transparency: `prompt_version` + inputs stored for reproducibility (S13/S06).
- **Accessibility** — Generated content MUST conform to WCAG structure rules (proper heading nesting, alt-text placeholders required for any suggested image, no color-only meaning); a post-gen a11y lint runs and flags violations (AC.8 owns enforcement; AC.3 emits the signal).
- **Scalability** — Cost is bounded by distinct signatures per unit (AC.2 caps this) × units × courses. Cache-first design means N students share ≤ ~12 variants per unit.
- **Reliability** — On any model error, timeout, or gate denial, the caller receives a typed fallback signal ⇒ base content is served; a failed generation is logged and retried by AC.4 with backoff, never blocking the learner.
- **Observability** — Histograms `adaptive_content.generate_ms`, `adaptive_content.fidelity_score`; counters `adaptive_content.generated`, `.rejected_fidelity`, `.rejected_safety`, `.cache_hit`; per-course token/cost via `ai_usage_log`.
- **Maintainability** — Prompt lives in `settings.system_prompts` (key `adaptive_content_variant`) so it is versioned/editable like `adaptive_quiz`; `prompt_version` stamped on each variant. Engine is in `service/adaptivecontent/generate.go`; JSON parsing mirrors `contentpagegeneration`.
- **Internationalization** — Variant is generated in the base content's language; reading-level adaptation respects locale; RTL preserved.
- **Backward compatibility** — Base content is never mutated; variants are separate rows. Units with generation disabled or fidelity failures degrade to base transparently.

## 7. Acceptance Criteria

- **AC-1.** *Given* a `compress` profile, *When* a variant is generated, *Then* it is shorter than the base, preserves every instructor-marked key term, and scores `fidelity ≥ 0.85`.
- **AC-2.** *Given* a `remediate` profile carrying misconception M, *When* generated, *Then* the variant explicitly addresses M (verified by the LLM-judge rubric) and passes fidelity.
- **AC-3.** *Given* a model that injects a fabricated statistic not in the base, *When* the fidelity gate runs, *Then* `fidelity_score` drops below threshold and the variant is stored `rejected`; the unit falls back to base.
- **AC-4.** *Given* a base content page containing a LaTeX formula and a code block, *When* a variant is generated, *Then* the formula and code survive byte-verifiably (hard check) or the variant is rejected.
- **AC-5.** *Given* two students with identical `profile_signature`, *When* both trigger generation, *Then* the model is called once and both resolve to the same `content_variants` row.
- **AC-6.** *Given* `aigateway` denies (student opted out of AI), *When* generation is requested, *Then* no model call is made and the caller gets a fallback signal (base content).
- **AC-7.** *Given* the base content page is edited, *When* its `content_version` bumps, *Then* previously generated variants are marked `superseded` and are not served.
- **AC-8.** *Given* any generation call, *When* it completes or fails, *Then* an `analytics.ai_usage_log` row exists with `feature='adaptive_content'`, correct `course_id`, and token counts.

## 8. Data Model

Reserves `441_adaptive_content_generation.sql`. Extends `content_variants` (AC.1) and adds content versioning + key terms.

```sql
-- 441_adaptive_content_generation.sql
ALTER TABLE course.content_variants
    ADD COLUMN IF NOT EXISTS prompt_version TEXT NOT NULL DEFAULT 'v1',
    ADD COLUMN IF NOT EXISTS content_version INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS prompt_tokens INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS completion_tokens INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS a11y_flags JSONB NOT NULL DEFAULT '[]'::jsonb;

-- Monotonic version bumped whenever the base content page changes, to invalidate variants.
ALTER TABLE course.adaptive_content_units
    ADD COLUMN IF NOT EXISTS content_version INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS min_fidelity REAL NOT NULL DEFAULT 0.85 CHECK (min_fidelity BETWEEN 0 AND 1);

-- Instructor-marked terms/facts that MUST survive any rewrite (glossary/standards anchors).
CREATE TABLE course.adaptive_content_key_terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    term TEXT NOT NULL,
    must_appear BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ac_key_terms_unit ON course.adaptive_content_key_terms (unit_id);

-- Versioned prompt (mirrors adaptive_quiz seeding).
INSERT INTO settings.system_prompts (key, label, content)
VALUES ('adaptive_content_variant', 'Adaptive content variant (per-learner rewrite)',
  '<<see §11 prompt sketch>>')
ON CONFLICT (key) DO NOTHING;
```

**Backfill:** none — new/extended empty tables.

## 9. API Surface

AC.3 is mostly an internal service. It exposes one instructor-facing synchronous preview route; bulk/async generation is AC.4.

```
POST /api/v1/courses/{course_code}/adaptive-content/units/{id}/variants/preview   instructor
   body: { profileSignature?: string, syntheticProfile?: {...} }   // preview for a real or hypothetical learner
   resp: { variant: {...}, fidelityScore, a11yFlags, safetyFlags, tokens }        // NOT persisted as servable unless approved
GET  /api/v1/courses/{course_code}/adaptive-content/units/{id}/variants           instructor (list w/ status)
```

Internal contract:
```go
type Variant struct {
    Sections      []DraftSection // reuse contentpagegeneration.DraftSection shape
    AxesApplied   []string
    FidelityScore float64
    SafetyFlags   []string
    A11yFlags     []string
    Model         string
    PromptVersion string
}
func GenerateVariant(ctx context.Context, client aiprovider.ScopedCompleter, in GenerateInput) (Variant, aiprovider.CallMeta, error)
```

## 10. UI / UX

- **Instructor preview panel (surfaced fully in AC.5):** "Preview as a learner who…" control lets the instructor pick an emphasis mode / synthetic profile and see the generated variant side-by-side with the base, with badges for `fidelity 0.92`, a11y warnings, and token cost. AC.3 provides the endpoint + rendering payload; AC.5 builds the workspace around it.
- **Diff affordance:** the preview highlights added/removed passages vs. base so the instructor can trust what changed.
- **States:** generating (spinner + "checking fidelity…"), rejected (red badge + reason), passed (green + score). Errors show "couldn't generate — students will see the original."
- **Mobile:** preview stacks base-over-variant; diff collapses to a toggle.
- **Accessibility:** preview panels are landmarked; fidelity/a11y badges have text equivalents, not color-only.

## 11. AI / ML Considerations

- **Model:** default to the platform's configured content model via `aiprovider.ScopedCompleter` (same selection logic as `contentpagegeneration`); JSON mode for the section array; a smaller/cheaper model for the LLM-judge fidelity rubric.
- **Prompt sketch** (`adaptive_content_variant`, versioned):
  > *You adapt an existing LMS content page for one learner. You are given the AUTHORITATIVE base content, an emphasis mode, the learner's concept gaps and misconceptions (by name), a target cognitive level, and allowed adaptation axes. Rewrite/re-emphasize the base to fit the learner. **Absolute rules:** preserve every fact, definition, standard reference, formula, code block, and listed key term from the base exactly; introduce no claim, statistic, or example not entailed by the base; never remove required terminology. Output ONLY JSON: {"sections":[{"heading","markdown"}]}.* Plus per-mode guidance (compress ⇒ condense; introduce ⇒ build from prerequisites; reinforce ⇒ add a worked example on gap concepts; remediate ⇒ name and correct the misconception).
- **Fidelity eval:** hybrid — (1) embed base vs. variant claim sets, flag low-support claims; (2) LLM-judge with a fixed rubric returning a 0–1 support score and a list of unsupported claims; (3) deterministic key-term / math / code presence checks. Composite `fidelity_score` = min(judge, term-check-pass?1:0-weighted).
- **Fallback path:** gate-deny, model error, low fidelity, or safety fail ⇒ base content. Never serve an unverified variant.
- **PII redaction:** inputs are concept ids/enums only; a guard asserts no PII fields are present before the call.
- **Cost budget:** per-course `monthly_token_budget` (AC.1) enforced by AC.4; AC.3 accounts tokens per call and refuses when AC.4 signals budget exhausted (returns fallback).

## 12. Integration Points

- `server/internal/service/aiprovider/` (ScopedCompleter), `service/aigateway/service.go` (add nothing — `FeatureAdaptiveContent` reserved in AC.1), `service/contentpagegeneration/` (reuse `DraftSection`, JSON parse/normalize helpers).
- `server/internal/service/adaptivecontent/generate.go`, `fidelity.go` (new).
- `server/internal/repos/adaptivecontent/variants.go` (new).
- `analytics.ai_usage_log` (existing) — cost logging.
- `settings.system_prompts` — `adaptive_content_variant` key.
- `server/internal/aidisclosure/` — feature disclosure copy.
- `server/migrations/441_adaptive_content_generation.sql` (+ down).
- Related: [AC.2](AC.2-pre-assessment-and-adaptation-profile.md) (profile input), [AC.4](AC.4-generation-pipeline-caching-cost.md) (queue/budget), [AC.8](AC.8-governance-safety-fairness-privacy.md) (safety/fairness enforcement).

## 13. Dependencies & Sequencing

- **Must ship after:** AC.1, AC.2; requires the shipped `aigateway`/`aiprovider`.
- **Must ship before:** AC.4 (pipeline wraps this engine), AC.5 (preview/approval uses it), AC.6 (serves its output), AC.7 (measures its effect).
- **Shared infra:** AI provider access; `settings.system_prompts`; `ai_usage_log`.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Hallucinated facts reach a learner | M | **H** | Multi-layer fidelity gate + hard term/math/code checks + human approval option (AC.5); reject-to-base default |
| LLM-judge is itself unreliable | M | H | Combine judge with deterministic checks; sample audits; instructor approval required by default (`require_instructor_approval=true`) |
| Reading-level simplification distorts meaning | M | M | Reading-level is an axis under the same fidelity gate; key-term preservation enforced |
| Cost runs away | M | H | Signature caching (AC.2) + per-course budgets (AC.4) + cache-first + neutral short-circuit |
| Prompt drift across versions changes behavior silently | L | M | `prompt_version` stamped; variants invalidated on prompt bump; changes reviewed |

## 15. Rollout Plan

- **Feature flag:** course `adaptive_content_enabled` (AC.1). Default serving mode is **auto-serve after gates pass**: a variant that clears fidelity + safety + a11y becomes `auto_served` without waiting for a human, because the automated gates are the primary trust mechanism and the fidelity guarantee is the whole point of this engine. Instructors/orgs may set `require_instructor_approval=true` per unit/course to require sign-off first, and AC.8 forces approval on for high-risk/minor contexts regardless. Oversight for auto-served units is retained *post-hoc* via AC.5 review, revoke, and the AC.8 contest path.
- **Sequencing:** deploy migration + prompt → ship engine + preview route → generate previews only (nothing serves) → instructor eyeballs fidelity on pilot units → enable AC.4 pre-warming.
- **Pilot cohort:** 2–3 instructor courses spanning K12 and HE with math/code content (stress the hard checks).
- **GA criteria:** fidelity gate blocks ≥ 95% of seeded hallucination test cases; key-term/math/code checks 100% on the fixture set; cost per unit within budget; a11y lint wired.
- **Rollback:** stop generation (AC.4 pause / unit `paused` / course flag off); already-approved variants can be revoked to base via AC.5; base content unaffected.

## 16. Test Plan

- **Unit** — prompt builder includes required guardrail clauses; JSON parse/normalize; key-term/math/code presence checks; fidelity composite math; neutral short-circuit skips the model.
- **Integration** — end-to-end generate for each emphasis mode against fixture pages; cache hit on identical signature; content-version bump supersedes; gateway-deny ⇒ fallback; `ai_usage_log` row written.
- **AI evals (offline)** — a golden set of (base, profile) pairs with human fidelity labels; measure gate precision/recall; a "hallucination gauntlet" of adversarial bases; track judge agreement vs. human.
- **End-to-end** — Playwright: instructor previews a variant, sees fidelity badge and diff.
- **Security** — no PII in outbound prompt (assertion + fixture); output sanitized against injection; only instructors can preview.
- **Accessibility** — a11y lint on generated variants (heading nesting, alt-text placeholders, no color-only).
- **Performance / load** — preview p95 ≤ 8 s; cache-hit path ≤ 30 ms; judge cost tracked.
- **Manual exploratory** — try to make the model drop a key term or change a definition; confirm rejection.

## 17. Documentation & Training

- Instructor guide: "How adaptive rewriting works and the fidelity guarantee."
- Trust/marketing page: "We never let AI change your facts" (fidelity gate explainer).
- Compliance appendix: prompt version + eval methodology for the DPIA/AI-Act file (S06/S13).
- Runbook: rotating the `adaptive_content_variant` prompt and invalidating variants.

## 18. Open Questions

1. Do we let instructors edit the per-course prompt, or only pick from vetted templates? (Lean: templates + optional appended instructions, mirroring `adaptive_system_prompt` on quizzes.)
2. Should math-heavy courses require a stricter `min_fidelity` default? (Likely yes — per-subject defaults.)
3. Is embedding-based claim comparison worth the extra model call vs. judge-only? (Validate in offline evals; make it configurable.)
4. How do we version and re-eval the prompt without regenerating every variant at once? (Lazy regeneration on next serve after a prompt bump — coordinate with AC.4.)

## 19. References

- Existing files: `server/internal/service/contentpagegeneration/service.go`, `service/aigateway/service.go`, `service/aiprovider/`, `server/migrations/040_quiz_adaptive.sql`, `281_ai_usage_logs.sql`.
- Related plans: [AC.2](AC.2-pre-assessment-and-adaptation-profile.md), [AC.4](AC.4-generation-pipeline-caching-cost.md), [AC.5](AC.5-instructor-authoring-and-approval.md), [AC.8](AC.8-governance-safety-fairness-privacy.md), `../standards/S13-eu-ai-act-high-risk.md`, `../standards/S06-dpia-pia-algorithmic-impact.md`.
- External: EU AI Act Annex III; NIST AI RMF (generative content controls); WCAG 2.1 AA.
