# LP04 — Strengths & Growth Areas (Facet)

> Implementation plan. A facet deriver on top of [LP01](LP01-foundation-derivation-engine.md).
> Primary signal: `course.learner_concept_states` (plan 1.1) + misconceptions (096). Follows
> [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | LP04 |
| **Section** | Learner Profile |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | LP01, 1.1 (learner model — shipped), 1.10 (misconceptions — shipped) |
| **Unblocks** | LP07, LP09 (targets review/recommendations) |

---

## 1. Problem Statement

The learner model (1.1) already estimates per-concept mastery, and misconception detection (1.10)
already flags recurring errors — but both are per-course and mostly instructor-facing (the mastery
heatmap, 9.3). The learner has no single, cross-course view of *what they're strong at and what needs
work*, in their own words, with evidence. This facet synthesises mastery and misconceptions across
all of a learner's courses into a strengths/growth picture that is honest, encouraging, and traceable.

## 2. Goals

- Derive **top strengths** (high, stable mastery) and **growth areas** (low mastery, decayed, or
  needs-review) across all the learner's courses.
- Surface **recurring misconceptions** (from 1.10) as concrete, addressable growth items.
- Surface **needs-review** concepts (mastery decayed since last seen, from 1.1's decay model).
- Attach evidence (attempt counts, courses, last-seen windows, mastery deltas) per LP01.
- Degrade to `insufficient_data` when too few concepts have signal.

## 3. Non-Goals

- Changing the mastery algorithm (owned by 1.1/1.6) or misconception detection (1.10).
- Instructor-facing heatmaps (9.3) — this is the learner's own cross-course view.
- Scheduling reviews (spaced repetition 1.5) — LP09 may trigger it; here we only surface.

## 4. Personas & User Stories

- **As a student**, I want to see that I'm strong in "linear equations" but keep slipping on "unit
  conversions," across all my courses, so I know where to focus.
- **As a self-learner**, I want my growth areas ranked so I spend limited time well.
- **As the adaptive layer (LP09)**, I want the ranked growth list so recommendations and review
  prompts target real gaps.

## 5. Functional Requirements

- **FR-1.** The `strengths_growth` deriver MUST read `course.learner_concept_states` for the user
  across all enrolled courses and rank concepts into **strengths** (mastery ≥ strong threshold, with
  sufficient `attempt_count`) and **growth areas** (mastery ≤ weak threshold or `needs_review_at`
  elapsed).
- **FR-2.** MUST apply the same **time-decay** semantics as 1.1 so a decayed concept surfaces under
  needs-review rather than as a stale strength.
- **FR-3.** MUST include **recurring misconceptions** (096/1.10) as growth items with the concept and
  a plain-language description.
- **FR-4.** MUST cap and rank output by `salience` (e.g. top 5 strengths, top 5 growth) so the UI is
  focused, not a 200-concept dump.
- **FR-5.** Each insight MUST carry evidence: contributing attempts, course split, last-seen window,
  and mastery value/delta, per LP01.
- **FR-6.** MUST return `insufficient_data` when fewer than a threshold of concepts (default 3) have
  any mastery signal.

## 6. Non-Functional Requirements

- **Performance** — Reads `learner.concept_states` by user (`idx_lcs_user`); derive ≤ 300 ms even at
  hundreds of concepts.
- **Security / Privacy** — Self-only via LP01. Cross-course reads restricted to the user's own
  enrollments. FERPA/GDPR via LP01/LP08.
- **Accessibility** — Strength/growth lists in LP07 use text + not-color-alone.
- **Scalability** — Bounded by a learner's concept count; incremental recompute on new mastery events.
- **Observability** — LP01 metrics `facet="strengths_growth"`.
- **Internationalization** — Concept names already localised in `course.concepts` (1.2).

## 7. Acceptance Criteria

- **AC-1.** *Given* mastery 0.92 on concept X across two courses, *then* X appears as a top strength
  with evidence naming both courses and the attempt counts.
- **AC-2.** *Given* mastery 0.9 on Y but 40 days since last seen (decayed), *then* Y appears under
  needs-review, not strengths.
- **AC-3.** *Given* a recurring misconception flagged by 1.10, *then* it appears as a growth item with
  a plain-language description.
- **AC-4.** *Given* a learner with only 1 concept touched, *then* facet is `insufficient_data`.
- **AC-5.** *Given* 200 concepts, *then* the API returns at most the top N per category (FR-4).

## 8. Data Model

No new tables — writes LP01 facet tables with `facet_key='strengths_growth'`. Reads
`course.learner_concept_states`, `course.learner_concept_events`, `course.concepts`, and misconception
tables (096). Reuses 1.1's decay function (do not reimplement — call the shared learner-state service).

## 9. API Surface

Served by LP01 `GET /me/learner-profile/facets/strengths_growth`. Example value:
`{ "strengths":[{"concept":"Linear equations","mastery":0.92,"courses":2}],
"growth":[{"concept":"Unit conversions","mastery":0.41},{"misconception":"Treats % as additive"}],
"needsReview":[{"concept":"Factoring","lastSeenDays":40}] }`.

## 10. UI / UX

Rendered by LP07 as "Your strengths & growth areas" (two ranked lists + needs-review chips). Empty
state = `insufficient_data`. Encouraging, non-punitive tone; growth ≠ failure.

## 11. AI / ML Considerations

Reads existing mastery estimates; adds no model. LP09 may layer an LLM explanation, but the ranked
facet stays rule-based and evidence-backed.

## 12. Integration Points

- `server/internal/service/learnerprofile/derivers/strengths_growth.go` (new).
- Calls the shared learner-state service (1.1) for decayed mastery; reads misconceptions (1.10).
- `server/internal/service/learnerstate/`, `server/internal/service/misconception/`.

## 13. Dependencies & Sequencing

- After LP01 + 1.1 + 1.10. Parallel with other facets. Feeds LP07, LP09.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Growth areas feel demoralising | M | H | Encouraging framing; pair each growth item with a next step (LP09) |
| Concept graph coverage uneven across courses | M | M | Only surface concepts with real attempt evidence; confidence per insight |
| Duplicate concepts across courses | M | M | De-dupe by concept slug; evidence lists the courses |

## 15. Rollout Plan

Behind `learner_profile_enabled`. Register deriver → pilot → GA with LP07. Rollback: unregister.

## 16. Test Plan

- **Unit** — strength/growth thresholds; decay → needs-review reclassification; ranking/cap; sufficiency.
- **Integration** — seed concept states + misconceptions → derive → assert lists + evidence.
- **E2E** — seeded learner shows strengths/growth via profile API.
- **Performance** — derive ≤ 300 ms at hundreds of concepts.

## 17. Documentation & Training

- Student help: "Your strengths & growth areas — how mastery is estimated and where it comes from."

## 18. Open Questions

1. Strong/weak mastery thresholds: fixed vs per-difficulty-tier (concepts carry `difficulty_tier`)?
2. Should growth ranking weight *importance* (prerequisite concepts) higher, needing concept-graph edges (1.2)?

## 19. References

- [LP01](LP01-foundation-derivation-engine.md),
  [1.1 learner model](../../completed/01-adaptive-learning-core/1.1-learner-model-knowledge-state.md),
  [1.10 misconceptions](../../completed/01-adaptive-learning-core/1.10-misconception-detection-remediation.md).
- Existing: `course.learner_concept_states`, `server/internal/service/learnerstate/`, `.../misconception/`.

## 20. Implementation Notes

- **Deriver:** `server/internal/service/learnerprofile/derivers/strengths_growth.go` +
  `strengths_growth_math.go`, registered in `server/internal/app/app.go`.
- **Decay:** uses `learnermodel.DecayAdjustedMasteryAt` (shared 1.1 decay semantics with explicit
  reference time for batch derive and tests).
- **Cross-course dedup:** aggregates concepts by normalized name (slugs are globally unique in
  `course.concepts`); evidence lists contributing courses.
- **Tests:** unit tests in `strengths_growth_test.go`; Postgres integration in
  `strengths_growth_db_test.go`.
