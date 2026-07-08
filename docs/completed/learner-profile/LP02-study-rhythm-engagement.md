# LP02 — Study Rhythm & Engagement Patterns (Facet)

> Implementation plan. A facet deriver on top of [LP01](LP01-foundation-derivation-engine.md).
> Primary signal: `analytics.engagement_events` (plan 9.7) + login audit. Follows
> [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | LP02 |
| **Section** | Learner Profile |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | LP01 (engine), 9.7 (engagement events — shipped) |
| **Unblocks** | LP07 (renders this facet), LP09 (adaptive timing) |

---

## 1. Problem Statement

Lextures records when and how long learners are active (heartbeats, logins, session activity in
`analytics.engagement_events`) but never tells the learner anything about their own study rhythm.
When do they actually study? Are they consistent or do they cram? How long are their sessions? This
facet turns that raw event stream into an honest, self-reflective picture of the learner's rhythm —
autonomously, with provenance — and gives the adaptive layer a signal for *when* to reach a learner.

## 2. Goals

- Derive the learner's **preferred study times** (time-of-day / day-of-week concentration), tz-aware.
- Derive **consistency**: active-days cadence, current/longest streak, cram-vs-steady pattern.
- Derive **session shape**: typical session length and number of sessions per active week.
- Attach evidence (event counts, windows) to every insight, per LP01.
- Degrade to `insufficient_data` for learners with too few active days.

## 3. Non-Goals

- Content/modality preferences (LP03) and topic interests (LP05).
- Study reminders / nudges UI (existing `study-reminders-settings-panel`); LP09 may consume timing.
- New client instrumentation — reuse 9.7 heartbeats and login events.

## 4. Personas & User Stories

- **As a student**, I want to see that I study best on weekday evenings in ~35-minute sessions, so
  I can plan around my real habits.
- **As a self-learner**, I want to see my study streak so I stay accountable.
- **As the adaptive layer (LP09)**, I want the learner's peak active window so reminders and
  spaced-repetition prompts land when they're actually studying.

## 5. Functional Requirements

- **FR-1.** The `study_rhythm` deriver MUST compute, from `analytics.engagement_events`
  (`event_type` in heartbeat/time-on-task) and login events over a rolling window (default 90 days):
  a **time-of-day distribution** (bucketed, tz-adjusted to the user's timezone) and a
  **day-of-week distribution**, and emit the dominant window(s) as insights.
- **FR-2.** MUST compute **active-days-per-week** cadence and a **consistency score** (steady vs
  bursty), plus **current** and **longest** streak of consecutive active days.
- **FR-3.** MUST compute **median session length** and **sessions per active week**, where a session
  is a run of heartbeats with gaps < 30 min (aligned with 9.7's heartbeat cadence).
- **FR-4.** Each insight MUST record evidence: event `observation_count`, window, and per-course
  split where relevant, per LP01 FR-4.
- **FR-5.** MUST return `insufficient_data` when the learner has fewer than a threshold of active
  days (default 5) in the window.
- **FR-6.** MUST be tz-aware: all time-of-day math uses the learner's timezone (LP01 Open Q4), not
  server time.

## 6. Non-Functional Requirements

- **Performance** — Deriver reads pre-aggregated `analytics.engagement_summaries` where possible;
  a single learner's derive ≤ 500 ms. No full-table scans (uses `idx_engagement_events_user_occurred`).
- **Security / Privacy** — Self-only via LP01 read model; FERPA/GDPR via LP01/LP08. No new PII.
- **Accessibility** — Rhythm charts in LP07 need text alternatives (owned by LP07).
- **Scalability** — Window-bounded queries; incremental recompute on new engagement batches.
- **Observability** — LP01 metrics, `facet="study_rhythm"`.
- **Internationalization** — Day/time labels localised; week-start locale-aware.

## 7. Acceptance Criteria

- **AC-1.** *Given* a learner active mostly 7–10pm on weekdays, *when* the deriver runs, *then* the
  facet's top insight names an evening/weekday window with supporting event evidence.
- **AC-2.** *Given* 12 consecutive active days then a gap, *then* longest streak ≥ 12 and current
  streak resets after the gap.
- **AC-3.** *Given* sessions averaging ~35 min, *then* median session length ≈ 35 min (±5) with
  evidence counting the sessions used.
- **AC-4.** *Given* a learner active only 2 days in 90, *then* the facet is `insufficient_data`.
- **AC-5.** *Given* a learner in tz `America/Denver`, *then* time-of-day buckets reflect local time.

## 8. Data Model

No new tables — writes `learner.profile_facets`/`_insights`/`_evidence` (LP01) with
`facet_key='study_rhythm'`. Reads `analytics.engagement_events` + `engagement_summaries`. May add a
lightweight per-user materialised cache only if the window query proves slow (defer; measure first).

## 9. API Surface

Served by LP01's `GET /me/learner-profile/facets/study_rhythm`. Example insight values:
`{ "peakWindows":[{"dow":"weekday","hourBucket":"19-22","share":0.41}], "consistencyScore":0.72,
"currentStreakDays":6, "longestStreakDays":12, "medianSessionMin":34, "sessionsPerActiveWeek":4 }`.

## 10. UI / UX

Rendered by LP07 as the "How you study" section (rhythm heatmap + streak + session stats). This plan
only defines the data; empty state = `insufficient_data` copy. Text-table alternative required (LP07).

## 11. AI / ML Considerations

Deterministic statistics only; no model. Fully reconstructable from evidence.

## 12. Integration Points

- `server/internal/service/learnerprofile/derivers/study_rhythm.go` (new).
- Reads `analytics.engagement_events` and `user.user_audit` (9.7 + login audit).
- Timezone from user/org setting (LP01 Open Q4).

## 13. Dependencies & Sequencing

- After LP01 + 9.7. Parallel with LP03–LP06. Feeds LP07, LP09.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Session segmentation off (mobile visibility gaps) | M | M | 30-min gap heuristic (matches 9.7 accuracy note); label as approximate |
| Rhythm read as judgement ("you cram") | M | M | Neutral, self-reflective framing in LP07; never a score shown to instructors here |
| Sparse learners get noisy windows | M | M | `insufficient_data` gate + confidence on insights |

## 15. Rollout Plan

Behind `learner_profile_enabled` (LP01). Register deriver → validate on pilot → GA with LP07.
Rollback: unregister deriver; facet disappears; other facets unaffected (LP01 isolation).

## 16. Test Plan

- **Unit** — session segmentation; streak math; tz bucketing; sufficiency gate.
- **Integration** — seed events → derive → assert facet/insights/evidence.
- **E2E** — seeded learner shows evening-weekday rhythm in profile API.
- **Performance** — derive over 90-day heavy learner ≤ 500 ms.

## 17. Documentation & Training

- Student help: "How you study — what the rhythm section means and where it comes from."

## 18. Open Questions

1. Rolling window length: 90 days vs term-aware?
2. Should logins alone (no content activity) count toward streaks, or only real study activity?

## 19. References

- [LP01](LP01-foundation-derivation-engine.md), [9.7 engagement](../../completed/09-analytics-reporting/9.7-engagement-metrics.md).
- Existing: `analytics.engagement_events`, `study-reminders-settings-panel.tsx`.
