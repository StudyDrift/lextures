# LP06 — Persistence & Help-Seeking / Learning Approach (Facet)

> Implementation plan. A facet deriver on top of [LP01](LP01-foundation-derivation-engine.md).
> Primary signal: quiz retakes (069), hints (095), revisions, notebooks/flashcards. Follows
> [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | LP06 |
| **Section** | Learner Profile |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | LP01; quiz attempts (069), hints (095), notebooks |
| **Unblocks** | LP07, LP09 (scaffolding/nudge tuning) |

---

## 1. Problem Statement

*How* a learner approaches difficulty is one of the most useful and least-surfaced things Lextures
knows. Do they retry quizzes until they master them, or bail after one try? Do they lean on hints
early or push through first? Do they consolidate with notebooks and flashcards? The signals — quiz
retakes (069), hint usage (095), assignment revisions, notebook/flashcard creation — all exist but are
never synthesised. This facet derives a **learning-approach** picture (persistence + help-seeking +
self-regulation) that is genuinely useful for self-reflection and for tuning how much scaffolding LP09
offers.

## 2. Goals

- Derive **persistence**: retake behaviour and whether retakes improve scores (productive persistence
  vs. thrashing), assignment revision after feedback.
- Derive **help-seeking style**: hint usage timing/frequency (early reliance vs. last-resort), and use
  of worked examples/scaffolds (095).
- Derive **self-regulation**: use of notebooks/flashcards to consolidate learning.
- Attach evidence per LP01; degrade to `insufficient_data` under threshold.

## 3. Non-Goals

- Judging learners (this is descriptive self-reflection, never a "grit score" shown to instructors).
- Changing scaffolding behaviour (LP09 consumes; this only derives).
- Mastery outcomes (LP04) — this is about *approach*, not achievement.

## 4. Personas & User Stories

- **As a student**, I want to see that I tend to retry until I get it and my scores improve when I do,
  so I recognise a strength in my own approach.
- **As a student**, I want to notice I reach for hints immediately, so I can try pushing further first.
- **As the adaptive layer (LP09)**, I want the learner's help-seeking style so I can calibrate how
  proactively to offer hints/worked examples.

## 5. Functional Requirements

- **FR-1.** The `learning_approach` deriver MUST compute **persistence** from `course.quiz_attempts`
  retake data (069): retake rate, and score improvement across attempts (productive vs unproductive
  persistence), plus assignment revision-after-feedback where available.
- **FR-2.** MUST compute **help-seeking style** from hint/scaffold usage (095): frequency and *timing*
  (hints requested early in an attempt vs. after genuine effort), and worked-example usage.
- **FR-3.** MUST compute **self-regulation** from notebook/flashcard creation and review activity
  (262/242/219) relative to coursework.
- **FR-4.** MUST classify into neutral, descriptive dimensions (e.g. persistence: low ↔ high;
  help-seeking: early ↔ independent; consolidation: light ↔ active) — never a single ranked score.
- **FR-5.** Each insight MUST carry evidence (attempt counts, hint counts, notebook counts, windows,
  score-delta samples) per LP01.
- **FR-6.** MUST return `insufficient_data` when the learner has too few attempts/interactions
  (default: < 5 quiz attempts and < 3 notebook actions).

## 6. Non-Functional Requirements

- **Performance** — Reads quiz attempts/hints/notebooks by user; derive ≤ 500 ms.
- **Security / Privacy** — Self-only via LP01. This facet is sensitive (behavioural); LP08 controls +
  neutral framing are mandatory. FERPA/GDPR via LP01/LP08.
- **Accessibility** — Approach dials/labels in LP07 use text + not-color-alone.
- **Scalability** — Bounded by learner's own attempts; incremental on new attempts/hints/notebooks.
- **Observability** — LP01 metrics `facet="learning_approach"`.
- **Internationalization** — Dimension labels localised.

## 7. Acceptance Criteria

- **AC-1.** *Given* a learner who retakes quizzes and improves scores on retake, *then* persistence is
  "high/productive" with evidence of the score deltas.
- **AC-2.** *Given* hints requested within the first few seconds of most attempts, *then* help-seeking
  is "early reliance" with hint-timing evidence.
- **AC-3.** *Given* frequent notebook/flashcard creation, *then* consolidation is "active" with counts.
- **AC-4.** *Given* a learner with 2 attempts and no notebooks, *then* facet is `insufficient_data`.
- **AC-5.** *Given* the facet, *then* it exposes dimensions, not a single grit score (FR-4).

## 8. Data Model

No new tables — writes LP01 facet tables with `facet_key='learning_approach'`. Reads
`course.quiz_attempts` (069) incl. retake/response data, hint/scaffold tables (095), notebooks
(262/242/219), and assignment submission/revision history where present.

## 9. API Surface

Served by LP01 `GET /me/learner-profile/facets/learning_approach`. Example value:
`{ "persistence":{"level":"high","productive":true,"retakeRate":0.6,"avgScoreDeltaOnRetake":0.18},
"helpSeeking":{"style":"early-reliance","hintsPerAttempt":1.4},
"consolidation":{"level":"active","notebookActions":22} }`.

## 10. UI / UX

Rendered by LP07 as "How you approach challenges" (three neutral dials + evidence). Empty state =
`insufficient_data`. Framing is strictly self-reflective; copy avoids virtue/vice language.

## 11. AI / ML Considerations

Deterministic; reconstructable from evidence. LP09 may use help-seeking style to calibrate scaffolding
proactiveness, but the facet itself is rule-based.

## 12. Integration Points

- `server/internal/service/learnerprofile/derivers/learning_approach.go` (new).
- Reads `quiz_attempts_http.go`/`069_quiz_attempts_responses.sql`, `095_hints_scaffolding.sql`,
  notebooks (`student_notebooks.go`, `notebook_tasks.go`), assignment submission history.

## 13. Dependencies & Sequencing

- After LP01. Parallel with other facets. Feeds LP07, LP09.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Behavioural read feels like surveillance/judgement | M | H | Neutral dimensions, learner-owned, LP08 pause/hide, never instructor-facing here |
| Hint-timing signal noisy | M | M | Require minimum attempts; confidence per insight; label approximate |
| "Early hints" penalised implicitly | M | M | Frame as style, not deficiency; pair with LP09 support, not a mark |

## 15. Rollout Plan

Behind `learner_profile_enabled`. Register deriver → pilot → GA with LP07 + LP08 (sensitive facet:
LP08 controls are a launch gate). Rollback: unregister.

## 16. Test Plan

- **Unit** — retake productivity; hint-timing classification; consolidation level; sufficiency.
- **Integration** — seed attempts/hints/notebooks → derive → assert dimensions + evidence.
- **E2E** — seeded learner shows learning-approach dials via profile API.
- **Performance** — derive ≤ 500 ms.

## 17. Documentation & Training

- Student help: "How you approach challenges — what these dials mean and where they come from."

## 18. Open Questions

1. Is per-attempt hint *timestamp* granularity available to judge "early vs late" (095 schema)?
2. Do assignment resubmissions carry enough structure to measure revision-after-feedback reliably?
3. Should this facet be default-collapsed in LP07 given its sensitivity?

## 19. References

- [LP01](LP01-foundation-derivation-engine.md).
- Existing: `069_quiz_attempts_responses.sql`, `095_hints_scaffolding.sql`, `quiz_attempts_http.go`,
  `student_notebooks.go`, `notebook_tasks.go`.
