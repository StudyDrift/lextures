# LP03 — Content & Modality Preferences (Facet)

> Implementation plan. A facet deriver on top of [LP01](LP01-foundation-derivation-engine.md).
> Primary signal: `analytics.engagement_events` (item_type/value), reading-level (204/211).
> Follows [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | LP03 |
| **Section** | Learner Profile |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | LP01, 9.7 (engagement events), 204/211 (reading level & prefs) |
| **Unblocks** | LP07, LP09 (content/format selection) |

---

## 1. Problem Statement

Different learners engage very differently with format: some watch every minute of a video, some
skim readings, some live in interactive activities. Lextures already captures video watch %, scroll
depth, time-on-task by `item_type`, and reading-level signals — but never tells the learner (or the
adaptive layer) which formats actually work for them. This facet derives **content & modality
preferences** so the profile can say, with evidence, "you finish videos but skim long readings," and
so LP09 can prefer formats that land.

## 2. Goals

- Derive **modality affinity**: relative engagement across video / reading / interactive-activity /
  quiz, from completion and depth signals (video watch %, scroll depth, time-on-task per item_type).
- Derive **content complexity comfort** from reading-level signals (204/211): the level at which the
  learner reads without slowing/abandoning.
- Derive **pacing through content** (fast skim vs thorough) from time-on-task vs content length.
- Attach evidence per LP01; degrade to `insufficient_data` under threshold.

## 3. Non-Goals

- Changing what content is shown (LP09 consumes this; this plan only derives).
- Study *timing* (LP02) and topic *interests* (LP05).
- Accessibility preferences (captions, dyslexia font) — those are explicit settings, not derived.

## 4. Personas & User Stories

- **As a student**, I want to see that I engage most with short videos and interactive activities,
  and skim long readings, so I understand my own patterns.
- **As the adaptive layer (LP09)**, I want a modality-affinity vector so I can prefer the format a
  learner completes when equivalent content exists.

## 5. Functional Requirements

- **FR-1.** The `content_modality` deriver MUST compute an **affinity score per modality**
  (`video`, `reading`/content_page, `interactive`/activity, `quiz`) from `analytics.engagement_events`:
  video → avg/max `percent_watched`; reading → scroll depth + time-on-task vs estimated read time;
  interactive/quiz → completion + time-on-task.
- **FR-2.** MUST compute a **complexity comfort** insight from reading-level signals (204/211): the
  reading level at which engagement stays high vs where it drops.
- **FR-3.** MUST compute a **pacing** insight (skimmer ↔ completer) from time-on-task relative to
  content length/expected duration.
- **FR-4.** MUST normalise across courses so a learner with more video content isn't spuriously
  "video-preferring" (affinity is relative engagement, not raw exposure).
- **FR-5.** Each insight MUST carry evidence (per-modality item counts, windows) per LP01.
- **FR-6.** MUST return `insufficient_data` when the learner has too few items across modalities
  (default: < 3 distinct items in ≥ 2 modalities).

## 6. Non-Functional Requirements

- **Performance** — Reads `analytics.engagement_events` by user + item_type; derive ≤ 500 ms.
- **Security / Privacy** — Self-only via LP01; FERPA/GDPR via LP01/LP08.
- **Accessibility** — Modality chart in LP07 needs text alternative.
- **Scalability** — Window-bounded, index-served; incremental on new engagement batches.
- **Observability** — LP01 metrics `facet="content_modality"`.
- **Internationalization** — Modality/level labels localised.

## 7. Acceptance Criteria

- **AC-1.** *Given* a learner who finishes videos (avg watch 90%) but scrolls only 30% of readings,
  *then* video affinity > reading affinity, each with supporting evidence.
- **AC-2.** *Given* exposure skew (10 videos, 1 reading), *then* affinity reflects *engagement*, not
  count (normalisation, FR-4).
- **AC-3.** *Given* reading-level signals show high engagement at grade 8 and drop-off at grade 12,
  *then* the complexity-comfort insight names ~grade 8–10 with evidence.
- **AC-4.** *Given* a learner with content in only one modality, *then* facet is `insufficient_data`.

## 8. Data Model

No new tables — writes LP01 facet tables with `facet_key='content_modality'`. Reads
`analytics.engagement_events` (`item_type`, `event_type`, `value`), reading-level tables (204/211),
and content length metadata (module content pages / video duration).

## 9. API Surface

Served by LP01 `GET /me/learner-profile/facets/content_modality`. Example value:
`{ "modalityAffinity":{"video":0.82,"interactive":0.7,"quiz":0.55,"reading":0.34},
"complexityComfort":{"low":"grade8","high":"grade10"}, "pacing":"thorough-on-video-skim-on-text" }`.

## 10. UI / UX

Rendered by LP07 as "How you like to learn" (modality bars + comfort band + pacing). Empty state =
`insufficient_data`. Never framed as "you're bad at reading" — neutral, format-preference framing.

## 11. AI / ML Considerations

Deterministic; reconstructable from evidence. No model.

## 12. Integration Points

- `server/internal/service/learnerprofile/derivers/content_modality.go` (new).
- Reads engagement events (9.7), reading level (`204_reading_level.sql`, `211_user_reading_preferences.sql`),
  content length (module content pages, video duration metadata).

## 13. Dependencies & Sequencing

- After LP01 + 9.7 + reading-level. Parallel with LP02/LP04–LP06. Feeds LP07, LP09.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Exposure skew mistaken for preference | M | H | Relative-engagement normalisation (FR-4) |
| Reading-level signal sparse | M | M | Fall back to modality/pacing only; mark comfort insight low-confidence |
| Modality read punitively | M | M | Neutral framing; profile is learner-owned, not an instructor score |

## 15. Rollout Plan

Behind `learner_profile_enabled`. Register deriver → pilot validate → GA with LP07. Rollback:
unregister deriver.

## 16. Test Plan

- **Unit** — affinity normalisation; complexity-comfort banding; pacing classification; sufficiency.
- **Integration** — seed mixed-modality events → derive → assert affinity ordering + evidence.
- **E2E** — seeded learner shows video-preferring profile via API.
- **Performance** — derive ≤ 500 ms on heavy learner.

## 17. Documentation & Training

- Student help: "How you like to learn — modality preferences and where they come from."

## 18. Open Questions

1. Source of "expected read time" for a content page (word count × WPM vs stored estimate)?
2. Do H5P/SCORM interactions count as a distinct "interactive" modality signal (h5p/scorm workers)?

## 19. References

- [LP01](LP01-foundation-derivation-engine.md), [9.7 engagement](../../completed/09-analytics-reporting/9.7-engagement-metrics.md).
- Existing: `204_reading_level.sql`, `211_user_reading_preferences.sql`, `analytics.engagement_events`.
