# UX.16 — Progress, Motivation and Learner Agency

> Implementation plan. Source: [audit.md](audit.md) §8 G-18.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.16 |
| **Section** | UI/UX — Learning Experience |
| **Severity** | MINOR (high strategic value) |
| **Markets** | K12 / HE / HS |
| **Status (today)** | PARTIAL — motivational features exist as unrelated siblings |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Product Design + Learning Design |
| **Depends on** | UX.9, UX.10 |
| **Unblocks** | Retention and completion outcomes; the "adapts" product thesis |

---

## 1. Problem Statement

Lextures ships gamification, badges, streaks, a leaderboard widget, study stats,
daily goals, credentials and learning paths — each built independently and
surfaced as a sibling banner on the dashboard. There is no motivational **model**
tying them together, and the evidence says the current mix is aimed at the wrong
need. A meta-analysis of 35 interventions (~2,500 participants) found gamification
significantly improves perceived **autonomy and relatedness** but has **minimal
effect on competence** (**R-5**) — yet competence ("I am getting better at this")
is precisely what a learner needs to persist, and it is the least-supported need
in the product today. Meanwhile the interface offers a learner almost no
**autonomy**: they cannot shape their dashboard, pace, or view. And a leaderboard
ships with no stated policy, despite evidence that ranking can read as
surveillance rather than motivation (**R-6**).

## 2. Goals

- Adopt **Self-Determination Theory** as the explicit design model for learner-
  facing surfaces, and make competence the primary signal.
- Give learners a **coherent, honest progress picture** — mastery and next step,
  not just points.
- Give learners **real agency**: pacing, goals, view, and what they see.
- Establish an evidence-based **policy** for competitive mechanics, defaulting to
  the safe option.
- Make motivational features *composable* rather than a set of independent banners.

## 3. Non-Goals

- Building new gamification mechanics. This plan **frames and, where evidence
  says so, demotes** what exists.
- Changing the learner model, concept graph, mastery calculation or the adaptive
  engine (`../adaptive/`) — consumed as-is.
- Instructor-facing analytics (at-risk, what's-working).
- Grading policy.
- Behavioural nudging that manipulates rather than informs (see §14).

## 4. Personas & User Stories

- **As a student**, I want to see what I actually know and what is next, not a
  point total that tells me nothing about my understanding.
- **As a student**, I want to set my own goal and pace, and change it without
  penalty.
- **As a student who missed a week**, I want to recover without a broken streak
  shaming me.
- **As a student**, I want to turn off comparison with classmates.
- **As a K-12 parent**, I want progress framed as growth, not rank.
- **As an instructor**, I want to choose whether competitive mechanics are on in my
  course.
- **As an administrator**, I want an organisation-level policy for competitive and
  behavioural features.

## 5. Functional Requirements

### Competence — the primary signal

- **FR-1.** Every learner-facing progress surface MUST answer three questions:
  **what have I learned**, **what am I working on**, **what is next**.
- **FR-2.** A **mastery view** MUST be the primary progress representation,
  sourced from the existing learner model and concept graph — not a points total.
- **FR-3.** Progress MUST be shown at three grains: course, module, concept.
- **FR-4.** Progress MUST be **honest**: it MUST distinguish *completed* from
  *mastered*, and MUST NOT present activity as achievement.
- **FR-5.** Feedback on incorrect work MUST be framed as **actionable next steps**
  ("Review *cell division*, then retry"), never as a bare score. The shipped
  misconception data MUST feed this.
- **FR-6.** Growth over time MUST be visible — a learner should be able to see they
  know more than they did a month ago.

### Autonomy

- **FR-7.** Learners MUST be able to set and change a personal study goal (time or
  items per week) and MUST be able to lower or pause it **without any negative
  framing**.
- **FR-8.** Learners MUST be able to choose their next activity from a small set of
  reasonable options, not only accept the single recommended one — a recommendation
  the learner cannot decline is not autonomy-supportive.
- **FR-9.** Every recommendation MUST carry a **visible reason** (the existing
  `ProfileRationaleChip`), and adaptively-rewritten content MUST always offer the
  standard version.
- **FR-10.** Learners MUST be able to dismiss or hide any motivational widget
  permanently ([UX.9](UX.9-role-aware-dashboard.md) FR-8/FR-9).
- **FR-11.** Streaks MUST support **pauses/freezes** and MUST NOT use loss-framing
  language. A missed day is a fact, not a failure.

### Relatedness

- **FR-12.** Relatedness signals (instructor presence, cohort activity, peer help)
  MUST be **collaborative, not comparative** — "12 classmates are also working on
  this module", never "you are 14th".
- **FR-13.** Instructor feedback MUST be prominent in the learner's progress view;
  a human comment is the strongest relatedness signal available.
- **FR-14.** Any peer-visible signal MUST be **opt-in** by the learner.

### Competitive mechanics — policy

- **FR-15.** Leaderboards MUST be **off by default** at platform, organisation and
  course level.
- **FR-16.** Where enabled, leaderboards MUST be **scoped** (course or group, never
  organisation-wide), MUST be **opt-in per learner**, and MUST NOT rank by grade.
- **FR-17.** Organisation administrators and instructors MUST have explicit
  controls for competitive mechanics, with the evidence summarised in the setting's
  help text.
- **FR-18.** Under-13 accounts MUST NOT be shown public rankings, per
  `../standards/S08-childrens-privacy-age-assurance-design-codes.md`.

### Coherence

- **FR-19.** Motivational surfaces MUST be composed into **one progress home**, not
  scattered as sibling widgets.
- **FR-20.** Each motivational feature MUST declare which SDT need it serves, and
  the registry MUST be reviewed to ensure competence is not under-served.

## 6. Non-Functional Requirements

- **Performance** — Mastery views MUST come from precomputed learner-model data;
  no per-render computation. Progress home LCP ≤2.0 s p75.
- **Security** — Peer-visible signals MUST enforce opt-in server-side. A learner
  MUST NOT be able to infer another learner's grades from any aggregate.
- **Privacy & Compliance** — Progress and behavioural data are education records
  (FERPA) and, for minors, subject to
  `../standards/S08-childrens-privacy-age-assurance-design-codes.md`. Aggregates
  MUST have a minimum cohort size (suggested ≥5) before display. Any nudging
  feature MUST be assessed under `../standards/S06-dpia-pia-algorithmic-impact.md`.
- **Accessibility** — Progress MUST be conveyed by text and shape, never colour
  alone; mastery visualisations MUST have accessible table equivalents; celebration
  animations MUST respect `prefers-reduced-motion` (the shipped `delight-moment`
  component already does).
- **Scalability** — Adding a motivational feature means declaring its SDT need and
  registering it; no new dashboard slot.
- **Reliability** — Progress MUST degrade gracefully when the learner model is
  unavailable: show completion, clearly labelled as such, rather than nothing.
- **Observability** — Emit `progress_view`, `goal_set`, `goal_changed`,
  `goal_paused`, `recommendation_declined`, `alternative_chosen`,
  `widget_dismissed`, `leaderboard_opt_in`. **Correlate with completion and
  retention — the point is outcomes, not engagement.**
- **Maintainability** — One progress home; one registry.
- **Internationalization** — Motivational copy is highly culture-sensitive;
  translation MUST be reviewed by native speakers, not machine-drafted
  ([UX.15](UX.15-i18n-coverage-and-rtl-completion.md) §11).
- **Backward compatibility** — Turning leaderboards off by default changes
  behaviour for orgs currently using them; existing enablement MUST be preserved on
  migration and only *new* orgs get the safe default.

## 7. Acceptance Criteria

- **AC-1.** *Given* a learner opens their progress view, *When* it renders, *Then*
  it shows what they have learned, what they are working on, and what is next —
  each at course, module and concept grain.
- **AC-2.** *Given* a learner has completed a module without demonstrating mastery,
  *When* progress renders, *Then* "completed" and "mastered" are visibly distinct.
- **AC-3.** *Given* a learner answers incorrectly, *When* feedback renders, *Then*
  it names a specific next step, not only a score.
- **AC-4.** *Given* a learner has used the product for a month, *When* they view
  growth, *Then* change over time is visible.
- **AC-5.** *Given* a learner sets a weekly goal, *When* they later reduce or pause
  it, *Then* the interface uses neutral language with no loss framing.
- **AC-6.** *Given* a recommendation, *When* shown, *Then* a reason is visible and
  at least two reasonable alternatives are offered.
- **AC-7.** *Given* adaptively-rewritten content, *When* shown, *Then* the learner
  can switch to the standard version in one action.
- **AC-8.** *Given* a new organisation, *When* created, *Then* leaderboards are
  off; *And* an existing organisation's current setting is preserved by migration.
- **AC-9.** *Given* leaderboards are enabled, *When* a learner has not opted in,
  *Then* they neither appear on it nor see it.
- **AC-10.** *Given* an under-13 account, *When* any surface renders, *Then* no
  public ranking appears.
- **AC-11.** *Given* a cohort aggregate with fewer than 5 members, *When* it would
  render, *Then* it is suppressed.
- **AC-12.** *Given* any motivational widget, *When* the learner dismisses it,
  *Then* it stays dismissed and is recoverable.
- **AC-13.** *Given* the progress view, *When* axe runs and a screen reader is
  used, *Then* 0 violations and every visualisation has a text/table equivalent.
- **AC-14.** *Given* moderated testing with ≥10 students, *When* asked "how are you
  doing in this course?", *Then* ≥80% answer using mastery/next-step language
  rather than points.

## 8. Data Model

```sql
-- server/migrations/NNN_learner_goals_and_prefs.sql
CREATE TABLE learner_goals (
  user_id      uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  course_id    uuid        REFERENCES courses(id) ON DELETE CASCADE,  -- NULL = global
  kind         text        NOT NULL,          -- 'minutes_per_week' | 'items_per_week'
  target       int         NOT NULL,
  paused_until date,
  updated_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, COALESCE(course_id, '00000000-0000-0000-0000-000000000000'::uuid), kind),
  CONSTRAINT learner_goals_target_chk CHECK (target > 0)
);

-- Per-learner opt-in for peer-visible signals (FR-14, FR-16)
ALTER TABLE users
  ADD COLUMN peer_visibility_opt_in boolean NOT NULL DEFAULT false;

-- Org/course competitive-mechanics policy (FR-15, FR-17)
ALTER TABLE organizations
  ADD COLUMN competitive_mechanics_enabled boolean NOT NULL DEFAULT false;
ALTER TABLE courses
  ADD COLUMN competitive_mechanics_enabled boolean;   -- NULL = inherit org
```

- **Backfill** — organisations with the current gamification/leaderboard feature
  already enabled MUST be backfilled to `true` so no live behaviour changes on
  deploy (AC-8). New orgs get `false`.
- Cascade deletes satisfy `../standards/S02-data-retention-deletion-engine.md`.
- Streak pause state reuses the existing study-reminders storage.

## 9. API Surface

```ts
// GET /api/v1/me/progress?courseId=…                  (auth: self)
type LearnerProgress = {
  grain: 'course' | 'module' | 'concept'
  learned:   { id: string; label: string; mastery: number }[]
  inProgress:{ id: string; label: string; mastery: number }[]
  next:      { id: string; label: string; reason: string; href: string }[]
  growth:    { at: string; mastered: number }[]        // time series
  degraded:  boolean                                    // learner model unavailable
}

// GET  /api/v1/me/goals?courseId=…                    (auth: self)
// PUT  /api/v1/me/goals                               (auth: self)
// POST /api/v1/me/goals/pause                         (auth: self)
type LearnerGoal = { kind: 'minutes_per_week' | 'items_per_week'; target: number; pausedUntil: string | null }

// PUT /api/v1/me/peer-visibility                      (auth: self)
type PeerVisibility = { optIn: boolean }

// PUT /api/v1/orgs/{orgId}/competitive-mechanics      (auth: org admin)
// PUT /api/v1/courses/{code}/competitive-mechanics    (auth: course teacher)
type CompetitivePolicy = { enabled: boolean | null }    // null on course = inherit
```

- `GET /me/progress` MUST return `degraded: true` with completion-only data rather
  than failing when the learner model is unavailable (§6 Reliability).
- No WebSocket events. Standard per-user rate limits.
- **OpenAPI** — all routes documented; `make openapi-check` passes.

## 10. UI / UX

- **New pages** — a **Progress home** per course (and a global roll-up), composing
  mastery, growth, goals, feedback and next steps.
- **Modified pages** — dashboard motivational widgets consolidate into a single
  entry point to Progress home; `MyBadges`, `study-insights-page`,
  `LeaderboardWidget`, `GamificationDashboardCard`, `DailyGoalProgressCard`,
  `StudyStatsCard` are recomposed rather than removed.
- **Key user flows**
  1. Student opens Progress → sees concepts mastered, in progress, and next →
     clicks a next step → lands in the right activity.
  2. Student sets a 90-minutes-per-week goal → later pauses it for a week → neutral
     confirmation, no penalty.
  3. Student gets a recommendation with a reason → chooses an alternative instead.
  4. Instructor turns competitive mechanics off for their course.
- **States** — progress: loading (skeleton), empty (new learner: "Your progress
  appears here once you start" + first action), degraded (learner model
  unavailable → completion-only, clearly labelled), error (retry).
- **Mobile/responsive** — mastery visualisation collapses to a ranked list on small
  viewports; the accessible table equivalent is the same data.
- **Accessibility annotations** — mastery visualisations have an equivalent
  `DataTable`; progress values are text plus visual; celebrations honour
  `prefers-reduced-motion`; no information is conveyed by colour alone.
- **Copy & i18n** — a dedicated copy review. **Rules: describe, don't judge; name
  the next step; never use loss framing ("Don't lose your streak!"), scarcity, or
  guilt.** Motivational copy must be reviewed by native speakers per locale
  (§6 Internationalization).

## 11. AI / ML Considerations

Consumes the existing learner model, concept graph, misconception detection and
adaptive engine (`../adaptive/`).

- **Model(s)** — existing learner-model and recommendation systems; no new model
  introduced by this plan.
- **Prompts** — n/a directly; where AI generates next-step feedback text (FR-5), it
  MUST be grounded in the detected misconception and the course's own content, and
  MUST NOT invent material.
- **Eval metric** — the metric is **completion and mastery growth**, not
  engagement. A change that raises time-on-site but not learning is a regression.
  Compare against a holdout, consistent with the AC.* effectiveness methodology.
- **Fallback path** — when the learner model is unavailable, show completion-based
  progress labelled as such (`degraded: true`), never a fabricated mastery number.
- **Explainability** — every recommendation shows its reason (FR-9). Mastery
  estimates MUST be explainable in plain language ("based on 6 questions across 3
  attempts").
- **Fairness** — mastery estimation and recommendations affect learner opportunity
  and fall under `../standards/S13-eu-ai-act-high-risk.md`; the existing fairness
  monitoring from the AC.* programme applies unchanged.
- **PII redaction / cost** — no new inference at render; all values precomputed.

## 12. Integration Points

- **External** — none.
- **Internal**
  - `clients/web/src/components/{gamification,badges,study-stats,study-reminders,learner-profile,self-paced}/**`
  - `clients/web/src/pages/lms/{MyBadges,study-insights-page,LeaderboardWidget,my-paths}.tsx`
  - `clients/web/src/components/ui/delight-moment.tsx` — celebration moments
  - `clients/web/src/components/learner-profile/profile-rationale-chip.tsx`
  - `clients/web/src/components/lms/adaptive-content/**` — standard-version toggle
  - Learner model / concept graph / misconception services in `server/internal/`
  - `server/internal/httpserver` — progress, goals, policy routes
  - `clients/web/src/components/settings/**` — org/course policy controls
- **Events** — motivation telemetry into `server/internal/telemetry`, joined to
  completion outcomes.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.9](UX.9-role-aware-dashboard.md) (widget registry and
  dismissal), [UX.10](UX.10-course-home-and-learning-flow.md) (progress must be
  visible in the learning flow, not only on a separate page).
- **Coordinates with** — `../adaptive/` (AC.7–AC.9) and
  `../../completed/01-adaptive-learning-core/`.
- **Shared infra** — learner-model read APIs; student participants for AC-14.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Turning leaderboards off by default is read as removing a sold feature | M | **H** | Existing enablement is **preserved by backfill** (AC-8); the change affects new orgs only; the setting's help text carries the evidence so the decision is defensible |
| Honest mastery ("completed but not mastered") demotivates learners | M | **H** | Frame as growth and next step, never deficit (FR-5, copy rules); validate in moderated testing (AC-14); if it demotivates, the framing is wrong, not the honesty |
| Mastery estimates are wrong and learners lose trust | M | **H** | Explainability (§11); show the evidence behind an estimate; let learners flag disagreement; degrade to completion when confidence is low |
| Motivational design slides into dark patterns (streak guilt, scarcity) | M | **H** | Explicit copy rules (§10); a design-review checklist question: *does this inform or manipulate?*; DPIA assessment for any nudging feature |
| SDT framing becomes decoration rather than a real constraint | M | M | FR-20 requires each feature to declare its need, and the registry review must show competence is served — this is a reviewable artefact |
| Cohort aggregates leak individual performance in small classes | M | **H** | Minimum cohort size ≥5 (AC-11), enforced server-side |
| Engagement metrics improve while learning does not | M | **H** | §11 eval metric is explicitly completion/mastery against a holdout, not engagement |

## 15. Rollout Plan

- **Feature flag** — `ffProgressHome` for the new surface; `ffCompetitivePolicy`
  for the policy controls and defaults.
- **Sequencing**
  1. SDT audit: map every existing motivational feature to the need it serves;
     publish the registry (FR-20). **Deliverable: a reviewed artefact.**
  2. Competitive-mechanics policy + backfill preserving current behaviour (AC-8).
  3. Progress home behind `ffProgressHome`: mastery, growth, next steps.
  4. Actionable feedback framing (FR-5) using existing misconception data.
  5. Goals with neutral pause/reduce (FR-7, FR-11).
  6. Recommendation alternatives and reasons (FR-8, FR-9).
  7. Dashboard consolidation — motivational widgets link into Progress home.
  8. Internal → one volunteer pilot course → 10% → GA, **measuring completion, not
     engagement**.
- **Dogfood** — internal org plus a volunteer pilot course with instructor consent.
- **GA criteria** — AC-1…AC-14 green; AC-14 comprehension ≥80%; no decrease in
  module completion in the pilot cohort vs holdout.
- **Rollback** — `ffProgressHome` off restores the current widgets; the policy
  backfill means no org loses a feature it was using.

## 16. Test Plan

- **Unit** — mastery aggregation across grains; growth series; completed-vs-mastered
  distinction; goal set/reduce/pause state machine; cohort-size suppression;
  under-13 ranking suppression; policy inheritance (course → org → platform).
- **Integration** — progress endpoint authz (self only); `degraded` path when the
  learner model is down; peer-visibility opt-in enforced server-side; policy
  backfill correctness on a seeded snapshot.
- **End-to-end** — Playwright: view progress at all three grains; set, reduce and
  pause a goal; decline a recommendation and choose an alternative; switch
  adaptive content to the standard version; verify a non-opted-in learner is absent
  from a leaderboard; verify an under-13 account sees no ranking.
- **Security** — attempt to infer a peer's grade from aggregates; cross-user goal
  access; policy escalation by a course teacher beyond their course.
- **Accessibility** — axe on progress surfaces × 4 themes (AC-13); screen-reader
  script: understand current mastery and next step without seeing the
  visualisation; verify the table equivalent; reduced-motion celebrations.
- **Performance / load** — progress endpoint p95 with a 40-concept course;
  Progress home LCP.
- **User research** — 10+ moderated student sessions (AC-14); a 4-week pilot
  comparing completion against a holdout; qualitative interviews on whether the
  progress picture felt honest and encouraging.
- **Manual exploratory** — QA matrix: new learner, mid-course learner, struggling
  learner, returning-after-absence learner, under-13 account, opted-out learner.

## 17. Documentation & Training

- **End-user (student)** — help-centre: "Understanding your progress", "Setting a
  goal", "Controlling what classmates can see".
- **Instructor** — "Motivation settings for your course", including the evidence
  summary for competitive mechanics.
- **Admin** — organisation policy guidance, with the children's-privacy
  implications noted.
- **Engineer** — `docs/guides/learner-motivation.md`: the SDT model, the need each
  feature serves, the copy rules, the honest-progress rule, the cohort-size rule.
- **API reference** — OpenAPI for progress, goals and policy routes.
- **Runbook** — "A learner disputes their mastery estimate": how to read the
  evidence behind an estimate.
- **Design review** — add *"does this inform or manipulate?"* to the standard
  design-review checklist.

## 18. Open Questions

1. Should the leaderboard be **removed** rather than defaulted off? Evidence
   (**R-5/R-6**) is unfavourable, but it may be contractually promised to existing
   customers. Needs a product and commercial decision.
2. What is the right mastery visualisation — radial, bar, concept map, or a simple
   ranked list? The concept graph already exists; test comprehension, do not assume.
3. Does honest "completed but not mastered" reporting need instructor consent per
   course? Some instructors may not want mastery exposed to learners.
4. Minimum cohort size for aggregates — is 5 correct for small homeschool cohorts,
   where it may suppress everything? May need a per-segment value.
5. Should goals be shareable with parents/guardians by learner choice?
6. How do we reconcile K-12 (where extrinsic motivators are more culturally
   accepted) with higher-ed and homeschool? Possibly a per-segment default, but
   FR-15's safe default should hold everywhere.

## 19. References

- Existing files: `clients/web/src/components/gamification/**`,
  `components/badges/**`, `components/study-stats/study-stats-card.tsx`,
  `components/study-reminders/daily-goal-progress-card.tsx`,
  `components/learner-profile/profile-rationale-chip.tsx`,
  `components/ui/delight-moment.tsx`,
  `clients/web/src/pages/lms/{LeaderboardWidget,MyBadges,study-insights-page,my-paths}.tsx`,
  `clients/web/src/components/lms/adaptive-content/**`
- Research: [research.md](research.md) R-4, R-5, R-6, R-9, R-18, §2
- Audit: [audit.md](audit.md) G-18, G-6
- External: [NN/g — Autonomy, Relatedness, and Competence in UX Design](https://www.nngroup.com/articles/autonomy-relatedness-competence/),
  [Gamification meta-analysis (ETR&D)](https://link.springer.com/article/10.1007/s11423-023-10337-7),
  [SDT in Behaviour Change Technologies (OUP)](https://academic.oup.com/iwc/advance-article/doi/10.1093/iwc/iwae040/7760010)
- Related plans: [UX.9](UX.9-role-aware-dashboard.md),
  [UX.10](UX.10-course-home-and-learning-flow.md),
  `../adaptive/`, `../../completed/01-adaptive-learning-core/`,
  `../standards/S08-childrens-privacy-age-assurance-design-codes.md`,
  `../standards/S13-eu-ai-act-high-risk.md`
