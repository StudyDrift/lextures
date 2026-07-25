# AC.2 — Pre-Assessment Binding & Adaptation Profile

> Implementation plan. Source: extends the shipped adaptive-learning core. Folder overview: [README](../../plan/adaptive/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AC.2 |
| **Section** | Adaptive Content Engine (ACE) |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | AC.1; reuses 1.1 (learner model), 1.6 (IRT theta), 1.7 (diagnostic), 1.10 (misconceptions) |
| **Unblocks** | AC.3, AC.6, AC.7 |

---

## 1. Problem Statement

AC.1 gave us empty adaptation tables. To adapt content we first need to know *how* a given learner should have their content adapted — and that decision must be deterministic, auditable, and cheap to recompute, not an opaque LLM guess. This story binds a pre-assessment ("entry ticket") to an adaptive unit, ingests the learner's attempt, and computes a structured **adaptation profile**: which concepts are gaps, which misconceptions are present, which Bloom level to target, and which emphasis mode (introduce / reinforce / compress / remediate) the content should take. The profile is the *input contract* for the generation engine (AC.3) and the anchor for measuring lift (AC.7). No content is rewritten here.

## 2. Goals

- Let an instructor designate an existing quiz (or a lightweight generated pre-check) as the **pre-assessment** for a unit.
- On pre-assessment submission, deterministically compute an **adaptation profile** from the learner model, concept-question tags, and misconception events — pure Go, no LLM.
- Produce a stable **profile signature** (hash of the adaptation inputs) so identical learners can share a cached variant in AC.3.
- Support three trigger modes: explicit pre-quiz submit, first-visit diagnostic, and "use existing mastery state if fresh enough."
- Fall back safely (empty/neutral profile ⇒ base content) whenever mastery data is missing.

## 3. Non-Goals

- Generating or serving any content variant (AC.3 / AC.6).
- Authoring the pre-assessment questions (reuses the existing quiz authoring + adaptive-quiz generation).
- Post-assessment / lift measurement (AC.7).
- New psychometrics — reuses shipped mastery (1.1) and IRT theta (1.6); does not invent a new model.

## 4. Personas & User Stories

- **As an instructor**, I want to point a unit at my existing "Chapter 3 pre-check" quiz so the system knows what each student already understands before they read the chapter.
- **As an instructor**, I want a one-click "generate a 5-question pre-check for this unit" so I don't have to author one (reuses adaptive-quiz generation).
- **As a student**, I want a short entry ticket that visibly personalizes what comes next, not busywork.
- **As a struggling student**, I want the system to detect the *specific* misconception I hold, not just "you got 40%".
- **As a researcher/DPO**, I want the exact inputs behind each adaptation decision recorded so decisions are explainable and contestable (S06, S13).

## 5. Functional Requirements

- **FR-1.** The system MUST let an instructor set/clear `pre_assessment_item_id` on a unit; it MUST reference a `quiz`-kind structure item in the same course.
- **FR-2.** The system MUST provide "generate pre-check" that creates an adaptive quiz (`module_quizzes.is_adaptive`) seeded from the unit's `base_content_item_id` and tagged to the unit's concepts, reusing the shipped `adaptive_quiz` generation path.
- **FR-3.** On pre-assessment submission (hook into the existing quiz-submit flow), the system MUST compute an adaptation profile for `(unit, enrollment)` and upsert `course.adaptation_profiles`.
- **FR-4.** Profile computation MUST derive: per-concept `gap = 1 - mastery` for concepts linked to the unit (via `concept_question_tags` on the pre-assessment's questions and/or the unit's outcome links); detected misconceptions from `misconception_events` on that attempt; a target Bloom level; and an `emphasis_mode`.
- **FR-5.** `emphasis_mode` MUST be selected by a documented deterministic rule: `compress` if all gaps ≤ `low_gap` (≈0.2); `remediate` if any misconception is present; `reinforce` if mean gap ∈ (low, high); `introduce` if mean gap ≥ `high_gap` (≈0.6) or no prior mastery record.
- **FR-6.** The system MUST compute `profile_signature = hash(unit_id, sorted(concept_id:bucketed_gap), sorted(misconception_ids), emphasis_mode, target_bloom, reading_level_pref, modality_pref, axis_set)` where gaps are bucketed (e.g., to 0.1) so near-identical learners collide into one cache key.
- **FR-7.** The system MUST support trigger modes on a unit: `pre_quiz` (default), `diagnostic_first_visit` (reuse 1.7 placement), and `mastery_snapshot` (skip the quiz, read current `learner_concept_states` if `last_seen_at` within `freshness_days`).
- **FR-8.** If required mastery/concept data is unavailable, the system MUST write a neutral profile (`emphasis_mode='introduce'`, empty gaps, signature `base`) so AC.6 serves base content — never blocking the learner.
- **FR-9.** The system MUST expose `GET .../units/{id}/profile` for the current student (their own profile) and, for instructors, `GET .../units/{id}/profiles` (cohort distribution — counts per emphasis_mode/signature, no free-text PII).
- **FR-10.** Profiles MUST be recomputed on a new pre-assessment attempt (latest attempt wins) and the recompute MUST write an `adaptive_content_events` row (`event_type='profile_computed'`) with the input snapshot.

## 6. Non-Functional Requirements

- **Performance** — Profile computation p95 ≤ 150 ms: one batched read of the learner's concept states + the attempt's misconception events, then in-memory rules. No per-concept N+1.
- **Security** — A student may read only their own profile; instructors read aggregate/cohort views. Profile writes only from the trusted server-side submit hook, never a client-supplied profile.
- **Privacy & Compliance** — The profile is a FERPA education record and an *automated-decision input* under GDPR Art. 22 / EU AI Act (S13); `payload_json` stores only concept ids, bucketed gaps, misconception ids, and enum decisions — no free text, no demographic attributes. Full input snapshot retained for explainability (S06) with retention per S02.
- **Accessibility** — The entry-ticket quiz reuses the accessible quiz runner; the "we're personalizing your content" interstitial has an ARIA live announcement.
- **Scalability** — O(concepts_in_unit) per submission; typically < 30 concepts. Signature bucketing bounds distinct variants per unit (target ≤ ~12 in practice), which caps AC.3 generation cost.
- **Reliability** — Computation is idempotent per attempt (`UNIQUE(unit, enrollment)` upsert keyed on `source_attempt_id`); a failure leaves the prior profile intact and logs an error, never a partial profile.
- **Observability** — Histogram `adaptive_content.profile_compute_ms`; counter by `emphasis_mode`; gauge `adaptive_content.distinct_signatures_per_unit`.
- **Maintainability** — All thresholds (`low_gap`, `high_gap`, `freshness_days`, bucket size) are named constants in `service/adaptivecontent/profile.go`, overridable per course via settings; the rule function is pure and unit-tested with synthetic mastery maps (mirrors the adaptivepath service style).
- **Internationalization** — Reading-level and modality preferences are locale-aware; thresholds are numeric/locale-independent.
- **Backward compatibility** — Units without a pre-assessment simply never produce profiles (AC.6 serves base). Additive columns only.

## 7. Acceptance Criteria

- **AC-1.** *Given* a unit bound to a pre-check and a student with mastery 0.9 on all unit concepts, *When* they submit the pre-check, *Then* a profile with `emphasis_mode='compress'` and a non-`base` signature is written.
- **AC-2.** *Given* the same unit and a student whose attempt triggers a tagged misconception, *When* they submit, *Then* the profile has `emphasis_mode='remediate'` and the misconception id appears in `payload_json`.
- **AC-3.** *Given* a student with no prior mastery record and mean gap 0.8, *When* they submit, *Then* `emphasis_mode='introduce'`.
- **AC-4.** *Given* the learner-state service errors, *When* a pre-check is submitted, *Then* a neutral `base` profile is written and an error is logged; the learner is not blocked.
- **AC-5.** *Given* two students whose bucketed gaps and misconceptions are identical, *When* both submit, *Then* their `profile_signature` values are equal (cache-shareable).
- **AC-6.** *Given* a student submits a *second* pre-check attempt, *When* processed, *Then* the profile is replaced by the newer attempt's computation and a `profile_computed` audit event references the new attempt.
- **AC-7.** *Given* an instructor, *When* they call `GET .../units/{id}/profiles`, *Then* they receive counts per emphasis_mode and per signature with **no** individual free-text or demographic data.

## 8. Data Model

Reserves `440_adaptive_content_profiles.sql`. Extends the `adaptation_profiles` table from AC.1.

```sql
-- 440_adaptive_content_profiles.sql
ALTER TABLE course.adaptation_profiles
    ADD COLUMN IF NOT EXISTS target_bloom course.bloom_level,
    ADD COLUMN IF NOT EXISTS reading_level_pref TEXT,       -- e.g. grade band or 'default'
    ADD COLUMN IF NOT EXISTS modality_pref TEXT,            -- 'text'|'worked_example'|'visual'|'default'
    ADD COLUMN IF NOT EXISTS axis_set TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS is_neutral BOOLEAN NOT NULL DEFAULT FALSE;

-- Per-unit trigger config.
ALTER TABLE course.adaptive_content_units
    ADD COLUMN IF NOT EXISTS trigger_mode TEXT NOT NULL DEFAULT 'pre_quiz'
        CHECK (trigger_mode IN ('pre_quiz','diagnostic_first_visit','mastery_snapshot')),
    ADD COLUMN IF NOT EXISTS mastery_freshness_days SMALLINT NOT NULL DEFAULT 30 CHECK (mastery_freshness_days >= 0);

-- The concepts a unit adapts around (explicit, in addition to those inferred from outcome links).
CREATE TABLE course.adaptive_content_unit_concepts (
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    concept_id UUID NOT NULL REFERENCES course.concepts (id) ON DELETE CASCADE,
    PRIMARY KEY (unit_id, concept_id)
);

COMMENT ON COLUMN course.adaptation_profiles.payload_json IS
    'Explainability snapshot: {conceptGaps:[{conceptId,gap}], misconceptions:[id], meanGap, priorRecord:bool}. No PII/free-text.';
```

**Backfill:** none — new/extended empty tables.

## 9. API Surface

```
PATCH /api/v1/courses/{course_code}/adaptive-content/units/{id}      instructor
   body may set: preAssessmentItemId, triggerMode, masteryFreshnessDays, conceptIds[]
POST  /api/v1/courses/{course_code}/adaptive-content/units/{id}/pre-check/generate   instructor
   -> creates an adaptive quiz item seeded from base content + concepts; returns its structure item
GET   /api/v1/courses/{course_code}/adaptive-content/units/{id}/profile              student (own)
GET   /api/v1/courses/{course_code}/adaptive-content/units/{id}/profiles             instructor (cohort)
```

Internal (not a public route): the existing quiz-submit path in `httpserver/quiz_delivery_http.go` / `module_quiz.go` gains a post-submit hook `adaptivecontent.OnPreAssessmentSubmitted(ctx, attempt)` that computes and upserts the profile transactionally with the attempt grade.

```ts
type AdaptationProfile = {
  unitId: string; emphasisMode: 'introduce'|'reinforce'|'compress'|'remediate';
  targetBloom?: string; profileSignature: string; isNeutral: boolean;
  conceptGaps: { conceptId: string; gap: number }[];
  misconceptions: string[];
};
```

## 10. UI / UX

- **Student entry-ticket flow:** the unit's pre-assessment renders in the existing quiz runner; on submit, a brief interstitial ("Personalizing this section for you…") precedes the content page (AC.6 renders the result). If `trigger_mode='mastery_snapshot'`, no quiz is shown — the interstitial notes "Using your recent progress."
- **Instructor unit editor (AC.5 hosts the full editor; AC.2 adds the pre-assessment picker):** dropdown to select an existing quiz or "Generate pre-check"; concept multi-select (reuses the `concepts-for-path` picker component); trigger-mode selector with inline help.
- **Cohort view:** a compact bar showing how the class distributes across emphasis modes (e.g., "12 introduce · 8 reinforce · 3 remediate · 5 compress"), linking to AC.9 for depth.
- **Empty/error states:** no pre-assessment set ⇒ "Add a pre-check to start adapting"; computation error ⇒ silent base fallback for the student, surfaced only in the instructor cohort view as "unprofiled".
- **Mobile/a11y:** entry-ticket interstitial uses `role="status"`; concept picker is keyboard-navigable.

## 11. AI / ML Considerations

- **Profile computation itself is not AI** — it is deterministic rules over the learner model, which is the deliberate design (explainable, contestable, cheap, cache-stable). This is important for S13 (high-risk AI transparency): the *decision* is rule-based; only the later *content rendering* (AC.3) is generative.
- **The optional pre-check generator** reuses the shipped `adaptive_quiz` system prompt and `aigateway` `quiz_generation` feature — no new model surface, no new prompt in this story.

## 12. Integration Points

- `server/internal/service/learnerstate/mastery.go` — read mastery/theta; misconception events already flow here.
- `server/internal/repos/concepts/` — concept ↔ question tags, concepts-for-course.
- `server/internal/httpserver/quiz_delivery_http.go`, `module_quiz.go` — post-submit hook.
- `server/internal/service/adaptivecontent/profile.go` (new) — pure rule engine + signature.
- `server/internal/repos/adaptivecontent/profiles.go` (new).
- `server/migrations/440_adaptive_content_profiles.sql` (+ down).
- Related: `../completed/01-adaptive-learning-core/1.7-diagnostic-placement-assessments.md`, `../completed/01-adaptive-learning-core/1.10-misconception-detection-remediation.md`.

## 13. Dependencies & Sequencing

- **Must ship after:** AC.1; reuses shipped 1.1 / 1.6 / 1.7 / 1.10.
- **Must ship before:** AC.3 (consumes the profile), AC.6 (serves per profile), AC.7 (anchors lift on the profile).
- **Shared infra:** none new; the pre-check generator reuses the AI stack only if the instructor opts in.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Signature too granular → cache explosion & cost | M | H | Bucket gaps to 0.1, cap axes, monitor `distinct_signatures_per_unit`, alert > threshold |
| Signature too coarse → learners under-served | M | M | Bucket size + thresholds per-course tunable; validate via AC.7 lift |
| Misconception detection misses ⇒ wrong emphasis | M | M | `remediate` requires an explicit tagged event; otherwise degrade to `reinforce`, not silent wrong content |
| Pre-check adds friction / test fatigue | M | M | Keep pre-checks ≤ 5 items; `mastery_snapshot` mode skips the quiz entirely |
| Automated-decision compliance (Art. 22 / AI Act) | M | H | Rule-based + full input snapshot + human oversight (AC.5) + student contest path (AC.8) |

## 15. Rollout Plan

- **Feature flag:** gated by the course `adaptive_content_enabled` (AC.1) — no separate flag. Profile computation only runs for units whose course flag is on and whose status is `active`.
- **Sequencing:** deploy migration → ship profile service + submit hook (writes profiles but nothing consumes them yet — safe) → verify signatures/aggregates on the pilot course → hand off to AC.3.
- **Pilot cohort:** the AC.1 internal course plus one volunteer instructor course with a real pre-check.
- **GA criteria:** deterministic profiles reproducible in tests; p95 ≤ 150 ms; neutral-fallback verified; cohort view shows sane distributions.
- **Rollback:** disable at unit level (`status='paused'`) or course flag off; existing profile rows are harmless (nothing serves them without AC.6).

## 16. Test Plan

- **Unit** — rule engine truth table across synthetic mastery/misconception inputs for all four emphasis modes; signature stability & collision tests; bucketing edge cases; neutral fallback on service error.
- **Integration** — submit pre-check → profile upserted transactionally with grade; second attempt replaces profile; `mastery_snapshot` path reads states without a quiz.
- **End-to-end** — Playwright: student takes a pre-check, sees the personalizing interstitial; instructor sees cohort distribution update.
- **Security** — student cannot read another student's profile (403); client-supplied profile body ignored; instructor cohort view exposes no PII.
- **Accessibility** — interstitial announced; concept picker keyboard-operable.
- **Performance / load** — k6: 1k concurrent submissions, profile compute p95 ≤ 150 ms; signature cardinality bounded.
- **Manual exploratory** — deliberately mistag a misconception, confirm `remediate` fires; remove pre-assessment, confirm base fallback.

## 17. Documentation & Training

- Instructor guide: "Setting a pre-check and choosing how the system profiles your students."
- Student help: "What happens when I take an entry ticket?"
- Compliance appendix: "How an adaptation decision is made" (rule table) for DPIA/AI-Act evidence (S06/S13).
- API reference: profile + pre-check-generate routes.

## 18. Open Questions

1. Should `mastery_snapshot` be the default for returning students to reduce quiz fatigue, with `pre_quiz` only on first exposure? (Lean yes; validate with AC.7 lift.)
2. How should conflicting signals (high mastery *and* a misconception) resolve? (v1: misconception wins → `remediate`; revisit with data.)
3. Should reading-level/modality preferences come from the LP09 learner profile when present, and if so, is that gated by the LP09 platform flag? (Yes when available; degrade to `default` otherwise.)
4. Do we expose the profile to the student in plain language ("we noticed you're strong on X, so we'll go faster")? (Deferred to AC.6 transparency.)

## 19. References

- Existing files: `server/internal/service/learnerstate/mastery.go`, `server/migrations/087_learner_model.sql`, `088_concept_graph.sql`, `096_misconceptions.sql`, `040_quiz_adaptive.sql`.
- Related plans: [AC.1](AC.1-foundations-flag-and-data-model.md), [AC.3](AC.3-content-generation-engine.md), [AC.7](AC.7-post-assessment-and-effectiveness.md), [AC.8](../../plan/adaptive/AC.8-governance-safety-fairness-privacy.md).
- External: GDPR Art. 22 (automated decisions); EU AI Act Annex III (education); Bloom's revised taxonomy.
