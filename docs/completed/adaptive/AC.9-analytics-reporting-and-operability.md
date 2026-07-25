# AC.9 — Analytics, Reporting & Operability

> Implementation plan. Source: reporting + operability surface for ACE; extends analytics (9.x) & observability (17.7). Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AC.9 |
| **Section** | Adaptive Content Engine (ACE) |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Data/analytics + frontend + SRE |
| **Depends on** | AC.4 (cost/cache metrics), AC.7 (effectiveness aggregates) |
| **Unblocks** | Data-driven tuning & GA sign-off of ACE |

---

## 1. Problem Statement

ACE now runs end-to-end, but its value and health are invisible: an instructor can't see whether adaptation is helping their class or which units to fix; an admin can't see cost, coverage, or system health across the org; and SRE has no dashboards or alerts for the pipeline. Without this reporting + operability layer, ACE can't be tuned, justified to buyers, or safely operated at scale. This story surfaces the numbers AC.4/AC.7/AC.8 already compute into instructor, admin, and ops views, and wires the metrics, alerts, and runbooks that keep the engine reliable.

## 2. Goals

- Give instructors a clear, per-course "Adaptive Content" report: coverage, lift vs. control, effectiveness by unit/mode, cost, and what to fix.
- Give admins an org rollup: adoption, cost, effectiveness, disparity flags, incidents.
- Wire full observability (metrics, traces, alerts) and SRE runbooks for the pipeline (AC.4) and gates (AC.3/AC.8).
- Define GA dashboards and health SLOs for ACE.

## 3. Non-Goals

- Computing the underlying aggregates (AC.7 effectiveness, AC.4 cost/cache, AC.8 fairness) — this story visualizes and operationalizes them.
- New governance policy (AC.8) or serving logic (AC.6).
- A general BI/export platform beyond CSV + the existing analytics surfaces.

## 4. Personas & User Stories

- **As an instructor**, I want one screen that tells me if adaptive content is working in my course and which units need attention.
- **As a curriculum lead**, I want to compare unit effectiveness and see which emphasis modes drive the most lift.
- **As an admin**, I want org-wide adoption, spend, and health at a glance, with drill-down.
- **As an SRE**, I want dashboards + alerts for queue depth, generation latency, cache hit rate, gate rejections, and budget exhaustion, plus runbooks.
- **As a buyer/accreditor**, I want an exportable, methodology-backed effectiveness report.

## 5. Functional Requirements

- **FR-1.** The system MUST provide an instructor **course report**: coverage (% of eligible content adapted, % students profiled/served), lift vs. control (from AC.7), per-unit verdicts, effectiveness by emphasis mode, cost this period (from AC.4), and a ranked "units to review" list (regressing/insufficient/low-fidelity).
- **FR-2.** The system MUST provide an **admin org rollup**: courses using ACE, students impacted, total spend + budget headroom, aggregate lift, disparity flags (AC.8), open contests, regressing units, and incident/kill-switch state.
- **FR-3.** Reports MUST support CSV export and respect de-identification / small-cell suppression from AC.7/AC.8.
- **FR-4.** The system MUST register ACE metrics in the telemetry layer (17.7): generation latency, cache hit/miss, queue depth, tokens/cost, fidelity-score distribution, gate rejections (fidelity/safety/a11y), served-variant/base/holdout/fallback counts, disparity flags, contests.
- **FR-5.** The system MUST define alerts: queue depth > threshold, cache hit rate < target, generation error/retry storm, budget exhausted, fidelity-rejection spike, any `regressing` verdict (AC.7), any disparity flag (AC.8), kill-switch engaged.
- **FR-6.** The system MUST trace the generation and serving paths (spans for profile compute → gate check → cache/generate → serve) for latency debugging.
- **FR-7.** Effectiveness/report reads MUST be served from the AC.7 cache/materialized views (no heavy live aggregation on request).
- **FR-8.** The system SHOULD provide an ACE section in the platform "AI reports" surface (reusing `ai-reports-api.ts`) so ACE cost sits alongside other AI features.
- **FR-9.** The system MUST expose SLO dashboards (serve latency, availability of adapted serving with base-fallback, generation success rate) for GA sign-off.

## 6. Non-Functional Requirements

- **Performance** — Report reads p95 ≤ 300 ms from cache/materialized views; CSV export streamed. Dashboards backed by pre-aggregated metrics, not ad-hoc scans.
- **Security** — Instructor reports scoped to their course; admin rollup admin-gated; exports carry the same suppression as on-screen; no per-student PII in cohort views.
- **Privacy & Compliance** — Reports use de-identified aggregates; individual data only in the instructor gradebook (existing authz). Cost/PII separation maintained. Aligns with analytics privacy posture (9.x) and evidence needs (S21).
- **Accessibility** — All charts have accessible table equivalents + text summaries; color-blind-safe palettes; keyboard-navigable; follows the `dataviz` skill conventions; WCAG 2.1 AA.
- **Scalability** — Metrics cardinality bounded (per-course labels capped; use exemplars/rollups for high-cardinality); materialized views refresh incrementally.
- **Reliability** — Dashboards degrade gracefully if a metric source is down; report shows "data as of <ts>" from the last refresh.
- **Observability** — This *is* the observability story; additionally self-monitors refresh-job freshness.
- **Maintainability** — Metrics registered via the existing telemetry package; report queries in `repos/adaptivecontent/reports.go`; web components under `components/lms/adaptive-content/reports/` reuse chart primitives.
- **Internationalization** — Report labels/units localized; numbers/dates locale-aware.
- **Backward compatibility** — Additive; no ACE data ⇒ empty states, never errors.

## 7. Acceptance Criteria

- **AC-1.** *Given* a course with servings + outcomes, *When* the instructor opens the Adaptive Content report, *Then* they see coverage, lift-vs-control, per-unit verdicts, mode breakdown, and cost, loading p95 ≤ 300 ms.
- **AC-2.** *Given* a regressing unit, *When* the report renders, *Then* it appears at the top of "units to review" with its verdict and a link to AC.5.
- **AC-3.** *Given* an admin, *When* they open the org rollup, *Then* they see adoption, spend vs. budget, aggregate lift, disparity flags, open contests, and incident state, with drill-down to a course.
- **AC-4.** *Given* any report, *When* exported to CSV, *Then* the file matches on-screen data and preserves small-cell suppression.
- **AC-5.** *Given* the pipeline, *When* queue depth exceeds threshold or cache hit rate drops below target, *Then* the corresponding alert fires to the on-call channel.
- **AC-6.** *Given* a generation request, *When* traced, *Then* spans cover profile→gate→cache/generate→serve with timings.
- **AC-7.** *Given* no ACE activity in a course, *When* the report opens, *Then* an informative empty state renders (no error).

## 8. Data Model

Shipped as `448_adaptive_content_reports.sql` (447 was taken by AC.8 governance). Mostly views over AC.4/AC.7/AC.8 tables.

```sql
-- 448_adaptive_content_reports.sql
-- Course-level rollup (refreshed with AC.7 effectiveness).
CREATE MATERIALIZED VIEW analytics.adaptive_content_course_report AS
SELECT
    u.course_id,
    COUNT(*)::INTEGER                                            AS n_units,
    COUNT(*) FILTER (WHERE u.status = 'active')::INTEGER         AS n_active_units,
    AVG(e.treatment_minus_holdout)::REAL                         AS mean_lift_vs_control,
    COUNT(*) FILTER (WHERE e.verdict = 'helping')::INTEGER       AS n_helping,
    COUNT(*) FILTER (WHERE e.verdict = 'regressing')::INTEGER    AS n_regressing,
    COUNT(*) FILTER (WHERE e.verdict = 'insufficient_data')::INTEGER AS n_insufficient,
    COUNT(*) FILTER (WHERE e.verdict = 'no_effect')::INTEGER     AS n_no_effect,
    COALESCE(s.tokens_used_period, 0)::BIGINT                    AS tokens_used_period,
    COALESCE(s.monthly_token_budget, 0)::BIGINT                  AS monthly_token_budget,
    NOW()                                                        AS refreshed_at
FROM course.adaptive_content_units u
LEFT JOIN analytics.adaptive_content_effectiveness e ON e.unit_id = u.id
LEFT JOIN course.adaptive_content_settings s ON s.course_id = u.course_id
GROUP BY u.course_id, s.tokens_used_period, s.monthly_token_budget;

CREATE UNIQUE INDEX idx_ac_course_report ON analytics.adaptive_content_course_report (course_id);

-- Coverage snapshot: eligible content items vs. adapted, students profiled/served (refreshed by job).
CREATE TABLE analytics.adaptive_content_coverage (
    course_id UUID PRIMARY KEY REFERENCES course.courses (id) ON DELETE CASCADE,
    eligible_content_items INTEGER NOT NULL DEFAULT 0,
    adapted_units INTEGER NOT NULL DEFAULT 0,
    students_profiled INTEGER NOT NULL DEFAULT 0,
    students_served_variant INTEGER NOT NULL DEFAULT 0,
    students_holdout INTEGER NOT NULL DEFAULT 0,
    refreshed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Backfill:** none. Views/tables populate on first refresh after activity exists.

## 9. API Surface

```
GET /api/v1/courses/{course_code}/adaptive-content/report            instructor
GET /api/v1/courses/{course_code}/adaptive-content/report/export     instructor  (CSV)
GET /api/v1/admin/adaptive-content/report                            admin  (org rollup)
GET /api/v1/admin/adaptive-content/report/export                     admin  (CSV)
-- ACE cost surfaces via the existing AI reports API (ai-reports-api.ts) filtered to feature=adaptive_content.
```

Metrics/traces exported through the existing telemetry pipeline (Prometheus/OTel per 17.7); alert rules defined in the monitoring config.

## 10. UI / UX

- **Instructor "Adaptive Content" report** (new tab beside the AC.5 workspace, or a section within it):
  1. **Header KPIs:** coverage %, students impacted, mean lift vs. control, spend this period.
  2. **Units-to-review** list (regressing → low-fidelity → insufficient data), each linking to the AC.5 unit.
  3. **Effectiveness by unit** (bar/interval chart with control comparison) and **by emphasis mode**.
  4. **Cost & budget** meter (from AC.4).
- **Admin org rollup:** adoption + spend + aggregate lift + disparity/contest/incident tiles; drill to course.
- **SRE dashboards** (Grafana/monitoring, not in-app): pipeline health, gate rejections, SLOs.
- **States:** rich empty states ("No adaptive data yet — set up a unit"); "data as of <ts>"; loading skeletons; export progress.
- **Accessibility:** every chart has a toggleable data table + text summary; color-blind-safe; keyboard-navigable; per the `dataviz` skill.
- **Mobile:** KPIs stack; charts become scrollable/tabular.

## 11. AI / ML Considerations

No model calls. This story reports on the AI system's behavior — including **fidelity-score distributions** and **gate-rejection rates** (from AC.3) and **fairness disparities** (from AC.8) — so operators can see model quality and drift over time. Surfacing fidelity/rejection trends is part of AI-Act post-market monitoring (S13) and continuous evidence (S21).

## 12. Integration Points

- `analytics.adaptive_content_effectiveness*` (AC.7), `adaptive_content_fairness` (AC.8), `adaptive_content_jobs` + budget (AC.4) — data sources.
- `server/internal/telemetry/` (17.7) — metric/trace registration + alert rules.
- `clients/web/src/lib/ai-reports-api.ts` (existing) — ACE cost slice.
- `server/internal/httpserver/adaptive_content_reports.go` (new); `repos/adaptivecontent/reports.go`.
- `clients/web/src/components/lms/adaptive-content/reports/*` — reuse chart primitives + `dataviz` conventions.
- `server/migrations/448_adaptive_content_reports.sql` (+ down).
- Related: [AC.4](AC.4-generation-pipeline-caching-cost.md), [AC.7](AC.7-post-assessment-and-effectiveness.md), [AC.8](AC.8-governance-safety-fairness-privacy.md).

## 13. Dependencies & Sequencing

- **Must ship after:** AC.4 (cost/cache/queue metrics), AC.7 (effectiveness aggregates); consumes AC.8 fairness.
- **Must ship before:** ACE GA sign-off (needs SLO dashboards + alerts).
- **Shared infra:** telemetry/monitoring stack (17.7), analytics schema, AI reports surface.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Metric cardinality blows up monitoring costs | M | M | Cap per-course labels; use rollups/exemplars; aggregate high-cardinality dims |
| Instructors misread lift as proof without caveats | M | M | Show n + uncertainty + "vs. control" framing; link methodology (AC.7/AC.8) |
| Report queries slow at scale | M | M | Materialized views + incremental refresh; p95 budget enforced |
| Alert fatigue | M | M | Tuned thresholds, dedupe, severity tiers; route regressing/disparity to owners |
| Cost double-counting vs. platform AI reports | L | M | Single source (`ai_usage_log`, feature=adaptive_content); reconcile |

## 15. Rollout Plan

- **Feature flag:** course `adaptive_content_enabled` (AC.1) gates the in-app report; SRE dashboards always on for ops.
- **Sequencing:** register metrics + traces (early, alongside AC.3/AC.4) → build SRE dashboards + alerts → ship instructor report → ship admin rollup → GA sign-off against SLOs.
- **Pilot cohort:** pilot instructors + admins reviewing their real ACE data.
- **GA criteria:** report reads p95 ≤ 300 ms; all alerts firing correctly in staging drills; charts pass a11y; cost reconciles with AI reports; SLO dashboards green.
- **Rollback:** hide in-app reports via flag; SRE dashboards/alerts remain (harmless, read-only).

## 16. Test Plan

- **Unit** — report aggregation queries; suppression in exports; KPI math; empty-state handling.
- **Integration** — materialized-view refresh correctness; CSV matches screen; admin rollup drill-down scoping; AI-reports cost slice matches `ai_usage_log`.
- **End-to-end** — Playwright: instructor opens report, sees KPIs + units-to-review, exports CSV; admin opens rollup, drills to a course.
- **Security** — instructor scoped to course; admin rollup gated; exports suppressed; no PII in cohort views.
- **Accessibility** — axe on all report views; chart data-table toggles; color-blind-safe; keyboard nav; per `dataviz` skill.
- **Observability tests** — synthetic load triggers each alert; traces present with expected spans; SLO panels compute.
- **Performance** — report p95 ≤ 300 ms; export streams large courses; metric cardinality within budget.
- **Manual exploratory** — regressing unit surfaces top; disparity flag appears in admin rollup; "data as of" reflects refresh.

## 17. Documentation & Training

- Instructor guide: "Reading your Adaptive Content report and acting on it."
- Admin guide: org rollup, spend, disparities, incidents.
- SRE runbook: ACE dashboards, alert response, SLOs, refresh-job health.
- Buyer-facing: exportable effectiveness report with methodology (ties to AC.7/AC.8).

## 18. Open Questions

1. In-app charts vs. embedding the existing analytics/BI surface? (Lean in-app for instructor immediacy; admin rollup may reuse the analytics dashboard.)
2. Which SLOs gate GA precisely (serve latency, generation success, adapted-availability-with-fallback)? (Define targets with SRE.)
3. Do curriculum leads need cross-course unit benchmarking (same standard across sections)? (Likely; phase 2.)
4. Real-time vs. scheduled refresh cadence for effectiveness? (Start scheduled; add on-demand refresh, already in AC.7.)

## 19. References

- Existing files: `server/internal/telemetry/` (17.7 observability), `clients/web/src/lib/ai-reports-api.ts`, `server/migrations/173_outcomes_report.sql`, `281_ai_usage_logs.sql`.
- Related plans: [AC.4](../../completed/adaptive/AC.4-generation-pipeline-caching-cost.md), [AC.7](../../completed/adaptive/AC.7-post-assessment-and-effectiveness.md), [AC.8](AC.8-governance-safety-fairness-privacy.md); analytics section `../completed/09-analytics-reporting/`.
- External: `dataviz` skill (chart/accessibility conventions); Prometheus/OpenTelemetry; NIST AI RMF (measure/monitor); WCAG 2.1 AA.
