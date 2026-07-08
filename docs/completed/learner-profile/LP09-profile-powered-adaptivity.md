# LP09 — Profile-Powered Adaptivity (Capstone)

> Implementation plan. Wires the learner profile (LP01–LP06) into Lextures' existing adaptive
> engines so the platform demonstrably *"adapts to every student."* Follows
> [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | LP09 |
| **Section** | Learner Profile |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | L (1–2 mo) |
| **Owner (proposed)** | Adaptive/AI team |
| **Depends on** | LP01–LP06 (profile), LP08 (consent/pause posture); consumes 1.4, 1.5, 1.8, 355 |
| **Unblocks** | The credible delivery of the homepage promise |

---

## 1. Problem Statement

The learner profile is only half the promise. "Set a course for learning that adapts to every student"
requires the profile to actually *change what the platform does*. Lextures already has adaptive engines
— recommendations (1.8), adaptive paths (1.4), spaced repetition (1.5), adaptive quiz selection
(`adaptivequizai`), and a persistent tutor (355) — but they run on narrow, per-course signals and don't
know the learner holistically. This capstone lets those engines consume the profile so recommendations,
sequencing, review timing, scaffolding, and tutor tone all reflect *this* learner — advisory and
transparent, never a consequential automated decision (LP08 Art. 22 posture).

## 2. Goals

- Expose the profile to adaptive consumers via a stable, read-optimised internal contract (extend
  LP01's `LearnerProfileService`), not by each engine re-querying signals.
- Feed concrete profile facets into concrete adaptive behaviours: interests/growth → recommendations
  (1.8); growth/needs-review → review timing (1.5) and adaptive sequencing (1.4); modality (LP03) →
  content-format selection; help-seeking (LP06) → tutor scaffolding proactiveness (355).
- Keep every adaptation **assistive, transparent, and reversible**: honour pause (LP08); surface a
  "personalised because …" rationale tied to the profile insight that drove it.
- Prove the loop with an eval: personalised vs. control on engagement/mastery outcomes.

## 3. Non-Goals

- Building new adaptive engines — this integrates the ones that exist.
- Consequential automated decisions (grades, placement, discipline) — explicitly excluded (Art. 22).
- Changing the profile derivation (LP01–LP06 own that).

## 4. Personas & User Stories

- **As a student**, I want my recommended next steps to reflect what I'm strong at, what I'm drawn to,
  and how I learn — and to see *why* something was recommended.
- **As a student who learns better from video**, I want the platform to prefer the video version of
  equivalent content when it exists.
- **As a learner who reaches for hints early**, I want the tutor to offer a nudge before a full hint,
  matching my style — and I want to be able to turn personalisation off.
- **As a self-learner**, I want review prompts timed to my real study window and aimed at my decayed
  concepts.

## 5. Functional Requirements

- **FR-1.** Extend `LearnerProfileService` (LP01) with a read-optimised
  `GetAdaptiveContext(userID) -> AdaptiveContext` returning the facets adaptive engines need
  (interests, growth/needs-review concepts, modality-affinity, peak study window, help-seeking style),
  cached and cheap.
- **FR-2.** **Recommendations (1.8)** MUST incorporate interests (LP05) and growth areas (LP04) when
  ranking suggested content/courses, and MUST attach a profile-derived rationale to each suggestion.
- **FR-3.** **Review timing (1.5) / sequencing (1.4)** SHOULD use needs-review concepts (LP04) and the
  learner's peak study window (LP02) to schedule/spot review prompts.
- **FR-4.** **Content selection** SHOULD prefer the modality (LP03) the learner engages with when
  equivalent items exist, without hiding other formats.
- **FR-5.** **Persistent tutor (355)** SHOULD adjust scaffolding proactiveness to help-seeking style
  (LP06) — e.g., offer a nudge-before-hint for early-reliance learners.
- **FR-6.** Every profile-driven adaptation MUST be **explainable**: expose the driving insight so the
  UI can show "personalised because {insight}", and MUST be **suppressed when the profile is paused**
  (LP08) or below sufficiency (fall back to today's per-course behaviour).
- **FR-7.** Adaptations MUST be **advisory only** — never gate grades, placement, or access (Art. 22).

## 6. Non-Functional Requirements

- **Performance** — `GetAdaptiveContext` p95 ≤ 20 ms (cached read); adaptive consumers add ≤ 10 ms to
  their existing latency budget.
- **Security / Privacy** — Consumers read the profile via the service (authz-enforced), never raw
  tables. Paused/insufficient → no personalisation. LLM tutor prompts (355) redact PII, send only
  aggregated facet values (LP01 AI note).
- **Accessibility** — "Personalised because …" rationales are text, screen-reader friendly (with LP07/UI).
- **Scalability** — Adaptive context cached (Redis, existing `redisclient`), invalidated on recompute.
- **Reliability** — If the profile service is unavailable, consumers fall back to current behaviour
  (graceful degradation, never a hard dependency).
- **Observability** — Metrics `learner_profile_adaptation_total{consumer,applied|suppressed}`; log the
  driving insight (hashed user id). A/B assignment recorded for eval.
- **Internationalization** — Rationale copy localised.
- **Backward compatibility** — Each consumer keeps its pre-profile path as the fallback (flagged).

## 7. Acceptance Criteria

- **AC-1.** *Given* a learner with a strong interest (LP05) and a growth area (LP04), *when*
  recommendations load, *then* suggestions reflect both and each carries a profile rationale.
- **AC-2.** *Given* a video-preferring learner (LP03) and equivalent content in two formats, *then* the
  video is preferred (other formats still available).
- **AC-3.** *Given* an early-reliance learner (LP06), *then* the tutor offers a nudge before a full hint.
- **AC-4.** *Given* the profile is **paused** (LP08), *then* all consumers revert to non-personalised
  behaviour and show no "personalised because" rationale.
- **AC-5.** *Given* an `insufficient_data` profile, *then* consumers use today's per-course behaviour.
- **AC-6.** *Given* any adaptation, *then* it is advisory — no grade/placement/access is gated by it.

## 8. Data Model

- No new profile tables. Adds a cache entry (`redisclient`) for `AdaptiveContext` keyed by user,
  invalidated on LP01 recompute. Optional `learner.adaptation_events` (append-only) to log applied
  adaptations for the eval (or reuse engagement/telemetry).

## 9. API Surface

- Internal: `LearnerProfileService.GetAdaptiveContext(userID) -> AdaptiveContext`.
- Consumers already have endpoints (recommendations, tutor, adaptive quiz); this adds a
  `rationale` field to their responses where an adaptation was profile-driven. No new public routes.

## 10. UI / UX

- "Personalised because {insight}" rationale chips on recommendation cards, review prompts, and (where
  natural) tutor suggestions, each linking back to the LP07 facet that drove it — closing the
  transparency loop end-to-end.
- A global "personalise using my profile" is implicitly the LP08 pause control (no separate toggle).

## 11. AI / ML Considerations

- Tutor (355) prompt augmentation uses aggregated facet values only, PII-redacted; label AI output per
  `ai_disclosure_http.go`.
- **Eval:** measure personalised vs. control cohorts on engagement (LP02 signals) and mastery gain
  (1.1) over a pilot; define success thresholds before GA. Guard against feedback loops (profile →
  adaptation → profile) by deriving from behaviour, not from prior adaptations.

## 12. Integration Points

- `server/internal/service/learnerprofile/` (GetAdaptiveContext), `redisclient`.
- Consumers: recommendations (1.8), adaptive paths (1.4), spaced repetition (1.5),
  `server/internal/service/adaptivequizai/`, persistent tutor (`355_persistent_tutor.sql`).
- Telemetry for adaptation metrics + A/B assignment.

## 13. Dependencies & Sequencing

- After LP01–LP06 + LP08. Ships last in the epic. Can land consumer-by-consumer (recommendations first,
  tutor last) behind sub-flags.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Feedback loop (profile ↔ adaptation) skews the profile | M | H | Derive only from raw behaviour; never feed adaptations back as signal |
| Over-personalisation narrows exposure (filter bubble) | M | M | Prefer, don't restrict; keep exploration; cap modality bias |
| Adaptation perceived as opaque | M | H | Mandatory "personalised because" rationale linking to LP07 |
| Consequential-decision creep | L | H | Hard rule (FR-7); review any new consumer against Art. 22 + DPIA (S06) |
| Profile service becomes a hard dependency | M | M | Graceful fallback to current behaviour on unavailability |

## 15. Rollout Plan

Behind `learner_profile_enabled` + per-consumer sub-flags (`lp_adapt_recommendations`,
`lp_adapt_tutor`, …). Ship one consumer → A/B eval → expand → GA on positive eval. Rollback: sub-flags
revert each consumer independently.

## 16. Test Plan

- **Unit** — `GetAdaptiveContext` shape/caching; suppression when paused/insufficient; rationale build.
- **Integration** — each consumer with/without profile; pause → fallback; cache invalidation on recompute.
- **E2E** — recommendation shows profile rationale; video-preferred selection; tutor nudge-before-hint;
  paused → no personalisation.
- **Security** — consumers read via service only; PII redaction in tutor prompts.
- **Eval/experiment** — A/B harness; engagement & mastery deltas; guardrails vs. filter bubble.

## 17. Documentation & Training

- Student help: "How your profile personalises Lextures (and how to turn it off)."
- Instructor/admin note: adaptations are assistive, not grading.
- Eval writeup + DPIA update (S06) for the adaptive use.

## 18. Open Questions

1. Order of consumer integration (recommendations → review timing → modality → tutor)?
2. Success metric + threshold for the personalised-vs-control eval before GA?
3. Do we log applied adaptations in a dedicated table or reuse telemetry/engagement events?

## 19. References

- [LP01](LP01-foundation-derivation-engine.md)–[LP06](LP06-persistence-help-seeking.md), [LP08](LP08-privacy-consent-controls.md).
- Existing: `server/internal/service/adaptivequizai/`, `355_persistent_tutor.sql`; plans
  [1.4](../../completed/01-adaptive-learning-core/1.4-adaptive-paths-across-modules.md),
  [1.5](../../completed/01-adaptive-learning-core/1.5-spaced-repetition-retrieval-practice.md),
  [1.8](../../completed/01-adaptive-learning-core/1.8-recommendations-engine.md).
- External: GDPR Art. 22; DPIA (S06).
