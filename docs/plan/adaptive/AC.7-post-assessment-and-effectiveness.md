# AC.7 — Post-Assessment, Effectiveness Measurement & Holdout Experiments

> Implementation plan. Source: closes the ACE loop; extends outcomes reporting (9.5). Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AC.7 |
| **Section** | Adaptive Content Engine (ACE) |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Data/analytics + backend platform |
| **Depends on** | AC.2 (pre-score), AC.6 (serving + holdout) |
| **Unblocks** | AC.9 (dashboards consume these aggregates) |

---

## 1. Problem Statement

The tagline promises the environment *adapts* — but adaptation without measurement is just faith. This story closes the loop: bind a **post-assessment** (exit ticket) to each unit, compute per-student **lift** (post − pre, and mastery delta), attribute it to the exact variant that was served, and — crucially — compare adapted students against the **holdout/control** group so the effect can be claimed *causally*, not just observed. It turns "we rewrote the content" into "the rewrite raised mastery by X points versus the control," which is the number that justifies the whole feature and satisfies AI-Act efficacy expectations.

## 2. Goals

- Bind a post-assessment to a unit and capture per-student post-scores keyed to their serving record.
- Compute per-serving `lift` (post−pre) and `mastery_after − mastery_before` from the learner model.
- Aggregate effectiveness per unit / emphasis mode / variant, including a **treatment-vs-holdout** comparison with sample sizes and a simple significance signal.
- Feed per-outcome improvement into the existing outcomes report (9.5) so accreditation reporting reflects adaptive gains.
- Give instructors an honest verdict per unit: "helping", "no measurable effect", "needs data", or "hurting" (with guardrail alerting).

## 3. Non-Goals

- Rendering the dashboards (AC.9 owns visualization; AC.7 computes and stores the aggregates).
- Generating/serving content (AC.3/AC.6).
- A full experimentation platform (multi-arm bandits, sequential testing) — v1 is a fixed holdout with a straightforward comparison; advanced designs are future work.

## 4. Personas & User Stories

- **As an instructor**, I want to see whether my adaptive unit actually improved scores versus students who got the standard content.
- **As an instructor**, I want to know *which* adaptation (compress vs. remediate) worked so I can trust or tune it.
- **As a department lead / accreditor**, I want adaptive gains reflected in outcome achievement, with methodology I can defend.
- **As a student**, I want the exit ticket to feel like a fair check of what I just learned, not a gotcha.
- **As a safety owner**, I want an alert if an adaptation is *reducing* learning so we can pull it.

## 5. Functional Requirements

- **FR-1.** The system MUST let an instructor bind `post_assessment_item_id` (a `quiz`-kind item in the course) to a unit.
- **FR-2.** On post-assessment submission (hook into quiz-submit), the system MUST locate the student's `adaptation_servings` row for that unit and upsert `course.adaptation_outcomes` with `pre_score_pct`, `post_score_pct`, `mastery_before`, `mastery_after`, and `lift`.
- **FR-3.** `pre_score_pct` MUST come from the pre-assessment attempt referenced by the profile (AC.2); `mastery_before`/`mastery_after` MUST snapshot `learner_concept_states` for the unit's concepts at pre- and post-time.
- **FR-4.** The system MUST compute per-unit aggregates: mean lift and mean mastery delta for **treatment** (served a variant) vs. **holdout** (served base), with n per group and a confidence/uncertainty indicator (e.g., difference-in-means with standard error; flag "insufficient data" below a min-n).
- **FR-5.** The system MUST compute per-**emphasis-mode** and per-**variant** effectiveness so low-performing variants are identifiable.
- **FR-6.** The system MUST emit a per-unit **verdict**: `helping` (treatment lift > holdout by a margin with adequate n), `no_effect`, `insufficient_data`, or `regressing` (treatment underperforms holdout) — and MUST raise an alert/notification on `regressing`.
- **FR-7.** The system MUST feed adaptive outcome achievement into the outcomes report (9.5): where a unit targets a learning outcome, its post-assessment contributes to `analytics.outcomes_report_student`, tagged so adaptive vs. control can be separated.
- **FR-8.** Aggregates MUST refresh on a schedule (and on demand) into a cache table/materialized view; per-serving rows are the source of truth.
- **FR-9.** All effectiveness reads MUST be aggregate and de-identified for cohort views; only the instructor's per-student gradebook shows individual pre/post (respecting existing gradebook permissions).

## 6. Non-Functional Requirements

- **Performance** — Post-submit outcome upsert p95 ≤ 150 ms (reuse the AC.2 batched mastery read). Aggregate refresh runs as a background job; cohort read p95 ≤ 200 ms from cache.
- **Security** — Individual pre/post visible only via existing gradebook authz; cohort/effectiveness views instructor+ only; holdout membership never exposed to students.
- **Privacy & Compliance** — Outcomes are education records (FERPA); the treatment/holdout comparison is a program evaluation, not per-student profiling for decisions. Methodology documented for DPIA (S06) and AI-Act efficacy monitoring (S13). Small-cell suppression (hide groups with n < k) to prevent re-identification.
- **Accessibility** — Effectiveness figures (rendered in AC.9) must have text/table equivalents; AC.7 exposes numeric data + CSV, not color-only.
- **Scalability** — Aggregates are grouped queries over `adaptation_servings ⋈ adaptation_outcomes`; indexed by unit; refreshed incrementally. Materialized view mirrors the 9.5 outcomes-report pattern.
- **Reliability** — Missing pre-score or serving row ⇒ the outcome row is still written with nulls where unknown and excluded from comparisons rather than corrupting aggregates. Idempotent upsert per serving.
- **Observability** — Gauges `adaptive_content.unit_mean_lift`, `.treatment_minus_holdout`; counters `.outcomes_recorded`, `.verdict_regressing`; alert on regressing verdict.
- **Maintainability** — Effectiveness computation in `service/adaptivecontent/effectiveness.go`; aggregates in `analytics` schema; verdict thresholds are named, tunable constants.
- **Internationalization** — Numeric/locale-independent; any verdict labels via i18n.
- **Backward compatibility** — Units without a post-assessment simply have no outcomes; additive tables; outcomes-report integration is opt-in per unit.

## 7. Acceptance Criteria

- **AC-1.** *Given* a student with pre-score 40% who was served a variant and scores 75% post, *When* they submit the exit ticket, *Then* an `adaptation_outcomes` row records `lift=35` linked to their serving + variant.
- **AC-2.** *Given* a unit with treatment and holdout groups, *When* aggregates refresh, *Then* the unit shows mean lift for each group, n per group, and a treatment−holdout difference with an uncertainty indicator.
- **AC-3.** *Given* fewer than the min-n students, *When* the verdict computes, *Then* it is `insufficient_data`, not a spurious "helping".
- **AC-4.** *Given* treatment mean lift is clearly below holdout with adequate n, *When* the verdict computes, *Then* it is `regressing` and an alert/notification fires to the instructor.
- **AC-5.** *Given* two emphasis modes on one unit, *When* effectiveness computes, *Then* per-mode mean lift is available and the weaker mode is identifiable.
- **AC-6.** *Given* a unit targeting a learning outcome, *When* the outcomes report (9.5) refreshes, *Then* the unit's post-assessment contributes to outcome achievement, separable by adaptive vs. control.
- **AC-7.** *Given* a cohort effectiveness read, *When* a group has n < k, *Then* that cell is suppressed (no re-identification) while the overall unit stats still render.

## 8. Data Model

Reserves `445_adaptive_content_effectiveness.sql`. Extends `adaptation_outcomes` (AC.1) + adds aggregate cache.

```sql
-- 445_adaptive_content_effectiveness.sql
ALTER TABLE course.adaptation_outcomes
    ADD COLUMN IF NOT EXISTS emphasis_mode TEXT,          -- denormalized from profile for grouping
    ADD COLUMN IF NOT EXISTS was_holdout BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS post_attempt_id UUID REFERENCES course.quiz_attempts (id) ON DELETE SET NULL;

CREATE INDEX idx_ac_outcomes_measured ON course.adaptation_outcomes USING BRIN (measured_at);

-- Per-unit effectiveness cache (refreshed by a job; source of truth = per-serving rows).
CREATE TABLE analytics.adaptive_content_effectiveness (
    unit_id UUID PRIMARY KEY REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    n_treatment INTEGER NOT NULL DEFAULT 0,
    n_holdout INTEGER NOT NULL DEFAULT 0,
    mean_lift_treatment REAL,
    mean_lift_holdout REAL,
    treatment_minus_holdout REAL,
    diff_std_error REAL,
    mean_mastery_delta_treatment REAL,
    mean_mastery_delta_holdout REAL,
    verdict TEXT NOT NULL DEFAULT 'insufficient_data'
        CHECK (verdict IN ('helping','no_effect','insufficient_data','regressing')),
    refreshed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ac_eff_course ON analytics.adaptive_content_effectiveness (course_id);

-- Per (unit, emphasis_mode) and per-variant breakdowns.
CREATE TABLE analytics.adaptive_content_effectiveness_by_mode (
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    emphasis_mode TEXT NOT NULL,
    n INTEGER NOT NULL DEFAULT 0,
    mean_lift REAL,
    PRIMARY KEY (unit_id, emphasis_mode)
);
```

**Backfill:** none. Aggregates populate on first refresh after servings/outcomes exist.

## 9. API Surface

```
PATCH /api/v1/courses/{course_code}/adaptive-content/units/{id}   instructor  (set postAssessmentItemId)
GET   /api/v1/courses/{course_code}/adaptive-content/units/{id}/effectiveness   instructor
   -> { nTreatment, nHoldout, meanLiftTreatment, meanLiftHoldout, treatmentMinusHoldout,
   --    diffStdError, verdict, byMode:[{emphasisMode,n,meanLift}], byVariant:[...] }
GET   /api/v1/courses/{course_code}/adaptive-content/effectiveness              instructor (all units)
POST  /api/v1/courses/{course_code}/adaptive-content/effectiveness/refresh      instructor (on-demand)
```

Internal: quiz-submit hook `adaptivecontent.OnPostAssessmentSubmitted(ctx, attempt)`; scheduled `effectiveness.RefreshCourse(courseID)`.

## 10. UI / UX

AC.9 builds the rich dashboard; AC.7 provides the data + a compact inline summary in the AC.5 workspace:
- **Per-unit effectiveness chip:** "▲ +12 pts vs. control (n=40/10)" or "Needs more data" or a red "▼ regressing — review".
- **Post-assessment picker** in the unit editor (mirrors the pre-assessment picker).
- **Verdict banner** on the unit with a link to the full breakdown (AC.9).
- **States:** insufficient data (neutral), helping (green), no effect (neutral), regressing (red + "review this unit").
- **Accessibility:** all figures have table/text equivalents; verdict conveyed by icon + text, not color alone.
- **Mobile:** chips stack; breakdown is a table.

## 11. AI / ML Considerations

Measurement is statistics, not AI: difference-in-means between treatment and holdout with a standard error and a min-n gate. Deliberately simple and explainable for accreditation and AI-Act efficacy evidence. The verdict thresholds (margin, min-n) are documented constants. Future work (bandits/sequential testing) is out of scope to keep the causal claim defensible. The **holdout is the causal backbone** — without it, we could only report pre→post change, which confounds adaptation with normal learning; the control group is what lets us attribute the delta to the adaptation.

## 12. Integration Points

- `server/internal/httpserver/quiz_delivery_http.go`, `module_quiz.go` — post-submit hook (mirrors AC.2 pre-hook).
- `server/internal/service/adaptivecontent/effectiveness.go` (new).
- `server/internal/repos/adaptivecontent/outcomes.go` (new).
- `analytics.outcomes_report_student` / `analytics.outcomes_report` (9.5) — contribute adaptive outcome scores.
- `server/internal/telemetry/` — metrics + regressing alert.
- notifications service — instructor alert on `regressing`.
- `server/migrations/445_adaptive_content_effectiveness.sql` (+ down).
- Related: [AC.2](../../completed/adaptive/AC.2-pre-assessment-and-adaptation-profile.md), [AC.6](../../completed/adaptive/AC.6-student-runtime-and-transparency.md), [AC.9](AC.9-analytics-reporting-and-operability.md).

## 13. Dependencies & Sequencing

- **Must ship after:** AC.2 (pre-score/profile), AC.6 (serving + holdout).
- **Must ship before:** AC.9 (dashboards visualize these aggregates); accreditation-grade outcome reporting.
- **Shared infra:** scheduled-job runner, notifications, `analytics` schema, outcomes report (9.5).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Over-claiming causal effect from small/biased samples | M | H | Min-n gate, difference-with-holdout, uncertainty shown, "insufficient_data" default, small-cell suppression |
| Holdout ethics (control gets less help) | M | H | Small/time-boxed holdout, instructor can set 0, equity monitoring (AC.8), everyone still gets valid content |
| Post-assessment gaming / different difficulty than pre | M | M | Encourage parallel forms; note limitation; support mastery-delta as a second signal |
| Confounders (motivation, time-on-task) | M | M | Randomized holdout controls for average confounders; document caveats |
| Regressing units linger unnoticed | M | M | Automated `regressing` alert + red verdict + AC.9 surfacing |

## 15. Rollout Plan

- **Feature flag:** course `adaptive_content_enabled` (AC.1); effectiveness only computes for units with a post-assessment and existing servings.
- **Sequencing:** deploy migration → ship post-submit hook + per-serving outcome write → ship aggregate refresh job + verdict → wire outcomes-report contribution → surface chips (AC.5) and dashboard (AC.9).
- **Pilot cohort:** AC.6 pilot courses once they've run a full pre→content→post cycle with a holdout.
- **GA criteria:** lift computed correctly on fixtures; treatment/holdout comparison + min-n gating verified; regressing alert fires; outcomes-report integration validated; small-cell suppression enforced.
- **Rollback:** stop the refresh job / hide effectiveness UI; per-serving outcome rows are harmless raw data.

## 16. Test Plan

- **Unit** — lift + mastery-delta math; difference-in-means + std error; verdict thresholds incl. min-n and regressing; small-cell suppression.
- **Integration** — post-submit writes outcome linked to serving+variant; missing pre-score handled; aggregate refresh produces expected treatment/holdout stats; outcomes-report gains adaptive contribution.
- **End-to-end** — Playwright: run pre→adapted content→post for treatment and holdout students; instructor sees a verdict chip and breakdown.
- **Security** — individual pre/post gated by gradebook authz; cohort de-identified; holdout hidden from students.
- **Accessibility** — figures have table equivalents; verdict icon+text.
- **Performance** — post-submit ≤ 150 ms; refresh job within SLA on a large course; cohort read ≤ 200 ms.
- **Manual exploratory** — seed a regressing unit → confirm alert; tiny sample → confirm "insufficient_data"; n<k group → confirm suppression.

## 17. Documentation & Training

- Instructor guide: "Reading effectiveness: lift, control groups, and verdicts."
- Methodology whitepaper: how ACE measures impact (for accreditors + DPIA/AI-Act, S06/S13).
- Help center: "What is a holdout group and why does it exist?"
- Runbook: responding to a `regressing` alert (pause unit, review variants).

## 18. Open Questions

1. Default holdout %? (Lean 10–20% for measurement, time-boxed, instructor-adjustable to 0 once a unit is proven.)
2. Should we require parallel pre/post forms to make lift trustworthy, or accept any pre/post pair with caveats? (Recommend parallel; allow either with a warning.)
3. Use mastery-delta or score-lift as the primary metric? (Report both; pick per subject.)
4. When is a unit "proven" enough to retire its holdout automatically? (Threshold on n + sustained positive verdict; define with data.)

## 19. References

- Existing files: `server/migrations/173_outcomes_report.sql` (9.5 outcomes report), `087_learner_model.sql`, `069_quiz_attempts_responses.sql`, `server/internal/service/learnerstate/`.
- Related plans: [AC.2](../../completed/adaptive/AC.2-pre-assessment-and-adaptation-profile.md), [AC.6](../../completed/adaptive/AC.6-student-runtime-and-transparency.md), [AC.9](AC.9-analytics-reporting-and-operability.md), `../standards/S06-dpia-pia-algorithmic-impact.md`, `../standards/S13-eu-ai-act-high-risk.md`.
- External: difference-in-means / A-B testing basics; NIST AI RMF (measure function); education program-evaluation methodology.
