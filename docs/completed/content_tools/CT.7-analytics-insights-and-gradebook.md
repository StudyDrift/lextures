# CT.7 — Content Tools: Analytics, Instructor Insights & Gradebook Bridge

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.7 |
| **Section** | Content Tools (CT) |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Analytics team |
| **Depends on** | CT.3, CT.4 |
| **Unblocks** | Evidence-based iteration on the tool shelf; ACE effectiveness inputs |

---

## 1. Problem Statement

Tools generate the richest formative signal the platform has ever had — where a class stalls, which
misconception is common, who never engaged — but that signal dies inside `state_json` unless something
reads it. Instructors will not open 30 state documents by hand, the platform has no way to tell which
tools actually help learning, and any tool that scores work has no sanctioned path into the gradebook.
This story turns per-enrollment tool state into instructor insight, platform telemetry, standards-based
learning records, and (opt-in, carefully) grades.

## 2. Goals

- Give instructors a per-instance and per-page view that answers: *did they engage, where did they
  struggle, what do I need to reteach?*
- Define a **summary projection contract** so each tool contributes typed, aggregatable facts instead
  of the platform guessing at its JSON.
- Emit xAPI/Caliper statements for tool interactions so external LRS/analytics consumers see them.
- Provide an **opt-in gradebook bridge** for tools whose scoring is defensible, with explicit
  instructor consent per instance and clean interaction with CT.4 resets.
- Feed tool outcomes into the existing learner model, concept mastery and at-risk scoring.

## 3. Non-Goals

- Making tools graded by default — the shelf is formative first; grading is an explicit choice.
- Per-tool bespoke dashboards (each tool contributes to the shared shell; a tool may add one custom
  visual, no more).
- Replacing course-level analytics (`service/instructorinsights`) — CT.7 feeds it.
- Predictive/AI-driven interpretation of tool data (a later story; CT.7 ships descriptive facts).

## 4. Personas & User Stories

- **As an instructor**, I want a single view of who engaged with the tools on this page so that I can
  chase the three students who did not.
- **As an instructor**, I want to see the most-missed inline question so that tomorrow's warm-up
  targets the actual gap.
- **As an instructor**, I want to read the class's reflections in one scrollable list so that I can
  quote two of them in class.
- **As an instructor**, I want to decide, per tool, whether it counts for a grade so that low-stakes
  practice stays low-stakes.
- **As a department chair**, I want to know which tools correlate with better outcomes so that our
  course template uses what works.
- **As a data engineer at a district**, I want tool interactions in our LRS so that they join the rest
  of our learning-analytics pipeline.
- **As a student**, I want to see my own progress across the tools on a page so that I know what I
  have left to do.

## 5. Functional Requirements

- **FR-1.** Each tool MUST declare a **summary projection**: a pure function from `state_json` to a
  typed record `{engaged, completed, score?, durationMs?, facets: Record<string, string|number>}`,
  plus a `facetSchema` describing aggregatable dimensions (e.g. `chosenOption`, `confidence`).
- **FR-2.** The system MUST compute and persist per-state summaries on write (and on reset), so
  aggregation never re-parses raw documents at query time.
- **FR-3.** The system MUST provide instance-level aggregates: engagement rate, completion rate, mean
  and distribution of score, median time-to-complete, and facet distributions with counts.
- **FR-4.** The system MUST provide an activity-level rollup across every instance on an item, and a
  course-level rollup across items.
- **FR-5.** Aggregates MUST exclude instructor/TA self-interaction rows and preview-scope rows.
- **FR-6.** Aggregates over fewer than **5** learners MUST be suppressed in any surface where they
  could re-identify an individual (except the instructor's own roster, which is identified by design).
- **FR-7.** The system MUST emit xAPI statements for `interacted`, `answered`, `completed` and
  `scored`, mapped per tool, through `service/learningevents`, and MUST forward them to a configured LRS.
- **FR-8.** A tool whose manifest declares `scoring.mode = 'auto' | 'manual'` MUST support an opt-in
  **gradebook bridge**: the instructor links the instance to an assignment/column, sets weight and
  late policy, and scores flow on completion.
- **FR-9.** The gradebook bridge MUST be **off by default**, MUST require an explicit per-instance
  action, and MUST show a clear "this counts for a grade" badge to students before they interact.
- **FR-10.** A CT.4 reset on a bridged instance MUST revert the passed score in the same transaction
  or refuse (CT.4 FR-11), never leaving grade and tool state divergent.
- **FR-11.** Tool outcomes MUST optionally map to course learning outcomes / standards, contributing
  evidence to mastery calculations like quizzes do.
- **FR-12.** Tool engagement MUST feed `service/atriskscoring` and `service/learnerstate` as a
  formative signal, with weights configurable and defaulting to low.
- **FR-13.** Students MUST see their own progress summary for the tools on a page ("2 of 4 activities
  complete") and their own scores where scored.
- **FR-14.** The system MUST expose CSV/JSON export of instance aggregates and per-learner summaries.
- **FR-15.** Platform admins MUST see cross-course tool telemetry: adoption per tool, completion rates,
  error rates, AI cost per tool — the evidence base for growing or pruning the shelf.
- **FR-16.** Aggregate reads MUST be cached with a short TTL (default 60 s) and MUST be invalidated on
  reset so an instructor never sees stale numbers after clearing a class.

## 6. Non-Functional Requirements

- **Performance** — Instance aggregate p95 ≤ 150 ms for 300 learners (precomputed summaries + cache).
  Course rollup p95 ≤ 500 ms. Summary computation adds ≤ 5 ms to a state write.
- **Security** — Instructor-gated, course-scoped, section-narrowed for limited TAs. Student endpoints
  return only the caller's own data. Exports are audit-logged.
- **Privacy & Compliance** — Free-text student work appears in instructor surfaces only, never in
  cross-course admin analytics (which see counts, never content). Small-*n* suppression (FR-6). xAPI
  actor identity follows the existing LRS identity policy. Retention follows CT.4/CT.8 windows.
- **Accessibility** — Every chart has an accessible table alternative and a text summary; colour is
  never the only encoding; distributions are keyboard-explorable.
- **Scalability** — Summaries stored in a narrow table with covering indexes; rollups computed
  incrementally on write; heavy course/admin rollups run as nightly materialized aggregates.
- **Reliability** — Aggregation is derived data: a rebuild job can recompute every summary from raw
  state, so a bug is recoverable without data loss.
- **Observability** — `lextures_content_tool_summary_writes_total{tool_id,outcome}`,
  `…_aggregate_query_seconds{scope}`, `…_gradebook_pushes_total{tool_id,outcome}`,
  `…_xapi_statements_total{verb}`. Alert on summary-write failure rate > 1%.
- **Maintainability** — One aggregation pipeline; a tool contributes a projection function and a facet
  schema, never SQL.
- **Internationalization** — Facet labels are i18n keys; exports carry localized headers with a stable
  machine-readable key row.
- **Backward compatibility** — Summaries are recomputable; changing a projection triggers a rebuild
  rather than a migration.

## 7. Acceptance Criteria

- **AC-1.** *Given* 30 learners with 22 completions, *When* the instructor opens instance analytics,
  *Then* engagement and completion rates match a direct count and instructor/TA rows are excluded.
- **AC-2.** *Given* an inline-questions instance, *When* aggregates render, *Then* the per-option
  distribution matches raw state and identifies the most-chosen distractor.
- **AC-3.** *Given* 3 learners have engaged, *When* an aggregate is requested in a small-*n*-suppressed
  surface, *Then* counts are withheld with an explanatory message while the instructor roster still
  lists all three.
- **AC-4.** *Given* a tool interaction, *When* an LRS is configured, *Then* a conformant xAPI statement
  is stored and forwarded with the correct verb and object IRI.
- **AC-5.** *Given* an instructor links a scored instance to a gradebook column, *When* a learner
  completes it, *Then* the score appears in the gradebook with the tool named as its source.
- **AC-6.** *Given* a bridged instance is reset for one learner, *When* the reset commits, *Then* the
  gradebook entry is reverted in the same transaction and shows a reversal record.
- **AC-7.** *Given* an instance mapped to a learning outcome, *When* a learner completes it, *Then*
  mastery evidence is recorded with the same shape as quiz-derived evidence.
- **AC-8.** *Given* a student opens a page with four tools, *When* they have completed two, *Then*
  their progress summary reads "2 of 4" and matches their state rows.
- **AC-9.** *Given* a projection function is changed and a rebuild runs, *When* it completes, *Then*
  every summary matches a fresh computation and no raw state was modified.
- **AC-10.** *Given* a class reset, *When* aggregates are re-read within the cache TTL, *Then* the
  numbers reflect the reset (cache invalidated), not stale values.
- **AC-11.** *Given* an admin opens platform tool telemetry, *Then* adoption, completion and cost
  appear per tool with no student free-text anywhere in the payload.

## 8. Data Model

Migration `server/migrations/455_content_tool_analytics.sql` (+ `.down.sql`).

```sql
-- 455_content_tool_analytics.sql

-- One row per learner state, maintained on write; the aggregation substrate.
CREATE TABLE IF NOT EXISTS analytics.content_tool_state_summaries (
    state_id       UUID PRIMARY KEY REFERENCES course.content_tool_states (id) ON DELETE CASCADE,
    instance_id    UUID NOT NULL REFERENCES course.content_tool_instances (id) ON DELETE CASCADE,
    course_id      UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    enrollment_id  UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    tool_id        TEXT NOT NULL,
    role           TEXT NOT NULL,            -- enrollment role at write time (filters staff rows)
    engaged        BOOLEAN NOT NULL DEFAULT FALSE,
    completed      BOOLEAN NOT NULL DEFAULT FALSE,
    score_pct      NUMERIC(5,2),
    duration_ms    INTEGER,
    facets_json    JSONB NOT NULL DEFAULT '{}'::jsonb,
    projection_version INTEGER NOT NULL DEFAULT 1,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ctss_instance ON analytics.content_tool_state_summaries (instance_id, role);
CREATE INDEX IF NOT EXISTS idx_ctss_course_tool ON analytics.content_tool_state_summaries (course_id, tool_id);
CREATE INDEX IF NOT EXISTS idx_ctss_facets ON analytics.content_tool_state_summaries USING GIN (facets_json jsonb_path_ops);

-- Nightly cross-course rollup for platform/admin telemetry (no student content).
CREATE TABLE IF NOT EXISTS analytics.content_tool_daily_rollups (
    day            DATE NOT NULL,
    org_id         UUID REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    tool_id        TEXT NOT NULL,
    instances      INTEGER NOT NULL DEFAULT 0,
    learners       INTEGER NOT NULL DEFAULT 0,
    engagements    INTEGER NOT NULL DEFAULT 0,
    completions    INTEGER NOT NULL DEFAULT 0,
    mean_score_pct NUMERIC(5,2),
    ai_tokens      BIGINT NOT NULL DEFAULT 0,
    ai_cost_usd    NUMERIC(12,4) NOT NULL DEFAULT 0,
    render_errors  INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (day, org_id, tool_id)
);

-- Opt-in gradebook + outcome linkage, per instance.
CREATE TABLE IF NOT EXISTS course.content_tool_grade_links (
    instance_id    UUID PRIMARY KEY REFERENCES course.content_tool_instances (id) ON DELETE CASCADE,
    assignment_item_id UUID REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    outcome_id     UUID REFERENCES course.course_learning_outcomes (id) ON DELETE SET NULL,
    points_possible NUMERIC(10,2),
    counts_for_grade BOOLEAN NOT NULL DEFAULT FALSE,
    late_policy    TEXT NOT NULL DEFAULT 'accept'
                     CHECK (late_policy IN ('accept','accept_marked','reject')),
    enabled_by     UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    enabled_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Backfill** — a one-shot job computes summaries for any state rows written before this migration.
**Rebuild** — the same job is re-runnable per tool when a projection version changes.

## 9. API Surface

| Verb | Path | Auth scope |
|---|---|---|
| `GET` | `.../content-tools/instances/{instance_id}/analytics` | instructor |
| `GET` | `.../content-tools/analytics?itemId=` | instructor |
| `GET` | `.../content-tools/analytics/course` | instructor |
| `GET` | `.../content-tools/analytics/export?itemId=&format=csv\|json` | instructor |
| `GET` | `.../content-tools/my-progress?itemId=` | student |
| `PUT` | `.../content-tools/instances/{instance_id}/grade-link` | instructor |
| `DELETE` | `.../content-tools/instances/{instance_id}/grade-link` | instructor |
| `GET` | `/api/v1/admin/content-tools/telemetry?from=&to=` | platform admin |

```ts
type InstanceAnalytics = {
  instanceId: string; toolId: string; title: string | null
  learners: number; engaged: number; completed: number; suppressed: boolean
  score: { mean: number; median: number; distribution: Array<{ bucket: string; count: number }> } | null
  medianDurationMs: number | null
  facets: Array<{ key: string; label: string; values: Array<{ value: string; count: number; correct?: boolean }> }>
  needsAttention: Array<{ enrollmentId: string; displayName: string; reason: 'not_started' | 'stuck' | 'low_score' }>
}
```

- **Rate limits** — analytics 120/min/user; export 5/min/user.
- **OpenAPI** — all routes documented; admin telemetry marked internal.

## 10. UI / UX

**New components** under `clients/web/src/components/content-tools/analytics/`:
`instance-analytics-panel.tsx`, `facet-distribution-chart.tsx` (with table alternative),
`needs-attention-list.tsx`, `page-tools-overview.tsx`, `grade-link-dialog.tsx`,
`student-progress-strip.tsx`, `admin-tool-telemetry.tsx`.

**Flows**

1. *From the page* — instructor viewing a content page sees a **Responses / Insights** control on each
   `ToolFrame`; opening it shows engagement, distributions, needs-attention, and a link to the CT.4
   roster.
2. *Page overview* — a "Tools on this page" panel summarising every instance with a completion bar.
3. *Grade link* — from the tool's insights, **Count this for a grade** opens a dialog: target column
   (existing or new), points, late policy, and a preview of who would receive what today; students see
   a graded badge once enabled.
4. *Student* — a compact progress strip at the top of the page ("2 of 4 activities complete") linking
   to the first incomplete tool.
5. *Admin* — platform telemetry table: adoption, completion, errors, AI cost per tool, sortable, with
   date range.

**States** — *Empty*: "No one has engaged yet" with a nudge to share the page. *Suppressed*: explains
the 5-learner threshold. *Loading*: skeletons. *Error*: retry without losing panel context.

**Mobile / responsive** — charts become stacked bars with the table alternative always available.

**Accessibility** — every chart has `role="img"` with a text summary plus a toggle to a real table;
needs-attention lists are semantic lists with clear actions; no colour-only encoding of correctness.

**Copy & i18n** — `contentTools.analytics.*`, `contentTools.grading.*`.

## 11. AI / ML Considerations

CT.7 makes no model calls. It *reports* on AI usage (tokens, cost per tool, per course) sourced from
`analytics.ai_usage_log`, and it exports tool outcomes as features for existing models: at-risk
scoring, the learner model, and ACE's effectiveness measurement (AC.7). Any future AI *interpretation*
of tool responses ("summarise the class's reflections") is explicitly a separate story with its own
disclosure and evals.

## 12. Integration Points

- **Internal** — `service/instructorinsights`, `service/learningevents` (xAPI), `service/xapi`,
  `service/caliper`, `service/grading` + `service/sbgaggregation` (gradebook and standards),
  `service/outcomes`, `service/learnerstate`, `service/atriskscoring`,
  `service/adaptivecontent` (AC.7 effectiveness inputs), `internal/background` (rollups, rebuild).
- **Events** — outbound webhooks (`service/webhooks`) for `content_tool.completed` and
  `content_tool.scored` so integrators can react.
- **Exports** — reuses the shipped report-export pipeline for CSV/JSON.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.3 (state), CT.4 (roster projection and reset semantics).
- **Must ship before:** any decision to prune or promote tools; ACE effectiveness attribution.
- **Shared infra needed:** job queue, export pipeline, LRS forwarding.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Grading formative tools distorts their purpose | H | M | Off by default, explicit opt-in, graded badge to students, docs framing tools as formative-first |
| Small classes re-identified through aggregates | M | H | Small-*n* suppression everywhere except the instructor's own roster; admin surfaces never carry content |
| Summary drift from raw state after projection changes | M | M | `projection_version` + full rebuild job + parity test against fresh computation |
| Instructor dashboards become noise across 200 tools | H | M | One shared shell, at most one custom visual per tool, needs-attention framing over raw charts |
| Gradebook divergence after resets | M | H | Transactional reversal (CT.4 FR-11) with tests for every ordering |
| xAPI volume overwhelms an LRS | M | M | Verb allowlist, batching, per-org rate caps, sampling for `interacted` |

## 15. Rollout Plan

- **Feature flag** — inherits `content_tools_enabled`; gradebook bridge behind
  `content_tool_grade_links` existence (per-instance opt-in) plus an org policy switch for districts
  that forbid non-assignment grading.
- **Sequencing** — migration `455_*` → summary write path + backfill → instance analytics → page
  overview → xAPI verbs → grade link → admin telemetry.
- **Dogfood** — pilot instructors use insights for two weeks before the grade bridge is offered.
- **GA criteria** — aggregate parity tests green, suppression verified, zero gradebook divergence
  incidents, chart a11y audit passed.
- **Rollback** — hide analytics UI (flag) while summaries keep accruing; disable grade links per org.

## 16. Test Plan

- **Unit** — projection functions per tool (fixture states → expected summary); aggregate math;
  suppression thresholds; role filtering; facet schema validation; cache invalidation on reset.
- **Integration** — write state → summary row; reset → summary + cache updated + grade reverted;
  backfill/rebuild parity; xAPI statement shape and forwarding; export contents.
- **End-to-end** — Playwright: instructor reads distributions, opens needs-attention, enables the grade
  link, student sees the graded badge and their score after completion.
- **Security** — authz matrix; student access to only their own progress; export audit entries;
  admin telemetry payload asserted free of student content.
- **Accessibility** — axe on charts and tables; screen-reader script for a distribution chart and its
  table alternative.
- **Performance / load** — 300-learner instance aggregate p95 ≤ 150 ms; course rollup on a 40-instance
  course; rebuild throughput ≥ 5,000 summaries/s.
- **Manual exploratory** — mixed-role courses, cross-listed sections, courses with archived instances.

## 17. Documentation & Training

- **Instructor** — "Read your class's tool responses"; when to grade and when not to; what students see.
- **Admin** — platform tool telemetry; LRS verb map; suppression policy.
- **Developer** — writing a projection and facet schema; the rebuild procedure; adding a verb mapping.
- **API reference** — analytics, grade-link and telemetry routes.
- **Runbook** — rebuilding summaries; diagnosing gradebook divergence; LRS backpressure.

## 18. Open Questions

1. Is 5 the right small-*n* threshold, or should it follow the org's existing analytics policy?
   Proposed: follow org policy where set, default 5.
2. Should `interacted` xAPI statements be sampled by default (volume) or complete (fidelity)?
   Proposed: complete for scored tools, sampled at 10% for pure engagement pings.
3. Should tools contribute to at-risk scoring at GA or after a term of observed data? Proposed: ship
   the plumbing disabled, enable after one term of baselines.
4. Do we need a per-tool "instructor view" extension point (one custom visual), or is the shared shell
   enough for the first 20 tools? Proposed: allow one, budget-capped, reviewed at 20 tools.

## 19. References

- Existing files this work touches: `server/internal/service/instructorinsights/`,
  `server/internal/service/learningevents/emit.go`, `server/internal/service/grading/`,
  `server/internal/service/sbgaggregation/`, `server/internal/service/atriskscoring/`,
  `server/migrations/455_content_tool_analytics.sql`.
- External standards: xAPI 1.0.3, IMS Caliper 1.2, IMS OneRoster (gradebook semantics), WCAG 2.1 AA
  for data visualisation.
- Related plans: [CT.3](CT.3-student-runtime-and-state-persistence.md),
  [CT.4](CT.4-instructor-state-console-and-reset.md),
  [CT.8](CT.8-governance-safety-privacy-accessibility.md),
  [AC.7 effectiveness](../../completed/adaptive/AC.7-post-assessment-and-effectiveness.md),
  [AC.9 analytics](../../completed/adaptive/AC.9-analytics-reporting-and-operability.md).
