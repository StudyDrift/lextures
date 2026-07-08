# LP01 — Learner Profile Foundation: Store, Provenance Model & Derivation Engine

> Implementation plan. Source: homepage promise *"Set a course for learning that adapts to
> every student."* Aggregates existing signal plans (1.1, 9.7). Follows
> [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | LP01 |
| **Section** | Learner Profile |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | L (1–2 mo) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | 1.1 (learner model), 9.7 (engagement events) — both shipped |
| **Unblocks** | LP02, LP03, LP04, LP05, LP06, LP07, LP08, LP09, LP10 |

---

## 1. Problem Statement

Lextures records a rich, autonomous stream of learning behaviour (engagement events, per-concept
mastery, quiz attempts, notebooks, reading level, feed activity) but it is siloed per course and
per feature, and almost all of it is instructor-facing. There is no single, learner-owned,
cross-course representation of *who a learner is and how they learn*, and no shared substrate the
adaptive engines or a learner-facing page could read. This plan builds that substrate: a profile
store, a **provenance model** that ties every derived insight back to the evidence it came from,
and a **derivation engine** that recomputes the profile autonomously from existing signals. It
ships nothing user-visible on its own except an (initially empty) real profile; facets LP02–LP06
fill it in.

## 2. Goals

- Persist one **learner profile per user**, cross-course, composed of pluggable **facets**.
- Define a **provenance/evidence model** so every insight records what signals produced it, how
  many, from where, and when — the transparency guarantee for the whole epic.
- Provide a **derivation engine**: a registry of per-facet "derivers" run both incrementally
  (reacting to new signals) and on a **nightly full recompute**, idempotently and versioned.
- Expose a stable internal Go interface (`LearnerProfileService`) and a read-only HTTP surface
  (`/api/v1/me/learner-profile`) so LP07/LP09/LP10 never touch source tables directly.
- Be **autonomous and zero-config**: no onboarding, no instructor setup, no per-assignment flag.
- Be FERPA/GDPR-correct from day one (erasable, exportable, pausable — hooks for LP08).

## 3. Non-Goals

- The facet derivation logic itself (LP02–LP06 own each facet's algorithm).
- The learner-facing UI (LP07) and mobile UI (LP10).
- Privacy controls UI and DSAR wiring (LP08 — this plan only leaves the hooks).
- Feeding the profile into recommendations/tutor/adaptive paths (LP09).
- Any new client-side event instrumentation — signals already exist (9.7).

## 4. Personas & User Stories

- **As a student**, I want the platform to hold one coherent picture of how I learn across all
  my courses, so what it shows me and adapts to is consistent everywhere.
- **As a platform engineer**, I want a single `LearnerProfileService` to read/derive profiles so
  new features don't re-query engagement/mastery tables and re-implement authorization.
- **As a privacy officer**, I want every profile insight to carry machine-readable provenance so
  a DSAR export or an "explain this" request is answerable without a data-science investigation.
- **As a self-learner**, I want my profile to build itself from what I do, with no setup.

## 5. Functional Requirements

- **FR-1.** The system MUST maintain exactly one `learner.profiles` row per user, created lazily
  on first signal or first read, storing status, `computed_version`, `last_computed_at`, and a
  `paused` flag (LP08 sets it).
- **FR-2.** A profile MUST be decomposed into **facets** (`learner.profile_facets`), keyed by a
  stable `facet_key` from a registry (`study_rhythm`, `content_modality`, `strengths_growth`,
  `interests`, `learning_approach`; extensible). Each facet stores a JSONB `summary`, a
  `confidence` (0–1), a `data_sufficiency` flag, and `updated_at`.
- **FR-3.** A facet MUST decompose into discrete **insights** (`learner.profile_insights`), each
  with an `insight_key`, human `label`, JSONB `value`, `confidence`, and `salience` (for ordering
  in the UI).
- **FR-4.** Every insight MUST carry ≥1 **evidence** row (`learner.profile_evidence`) recording
  `source_kind` (e.g. `engagement_event`, `quiz_attempt`, `learner_concept_state`, `notebook`),
  `source_table`, an aggregate `observation_count`, a `contribution` weight, the observation time
  window, and optional `course_id`/`sample_refs`. Insights with no evidence MUST NOT be surfaced.
- **FR-5.** The **derivation engine** MUST expose a `FacetDeriver` interface; each facet
  registers one. A run MUST be idempotent (re-running over the same signals yields the same facet)
  and MUST write facet + insights + evidence atomically per facet.
- **FR-6.** The engine MUST run in two modes: (a) **incremental** — triggered when relevant new
  signals arrive (debounced), recomputing only affected facets; (b) **full recompute** — a
  nightly scheduled job over all active learners. Both use the existing job queue (`338`) /
  scheduler (`340`).
- **FR-7.** The system MUST expose `GET /api/v1/me/learner-profile` returning the caller's full
  profile (facets + insights + top evidence) and `GET /api/v1/me/learner-profile/facets/{key}`
  and `.../facets/{key}/evidence` for drill-down. Callers MAY read **only their own** profile
  (LP08 adds any staff/guardian scopes).
- **FR-8.** Each facet MUST report **data sufficiency**: below a minimum signal threshold the
  facet MUST return a `insufficient_data` state rather than a low-confidence guess.
- **FR-9.** The system SHOULD record a `computed_version` per deriver so a deriver upgrade can
  force a recompute and old insights are never mixed with new-algorithm insights.
- **FR-10.** All derived numeric outputs MUST be reproducible from stored evidence (no hidden
  state), so the profile can be fully rebuilt from source signals on demand.

## 6. Non-Functional Requirements

- **Performance** — `GET /me/learner-profile` p95 ≤ 80 ms for a fully populated profile (read from
  materialised `learner.*` tables, no on-read signal scans). Nightly full recompute of 100 k
  learners MUST finish inside a 2-hour window; incremental recompute of one learner's affected
  facets ≤ 2 s.
- **Security** — Profile is PII. All endpoints require a valid JWT; a learner reads only their own
  profile. Deriver jobs run as system with row-scoped queries. No cross-tenant reads.
- **Privacy & Compliance** — Profile is a FERPA education record **and** GDPR Art. 4(4) profiling
  output. MUST be erasable (LP08) via `ON DELETE CASCADE` on `user_id`, exportable via
  `report_export.go`, and pausable. No raw profile text sent to external LLMs without redaction.
- **Accessibility** — No UI in this plan; LP07 carries WCAG 2.1 AA.
- **Scalability** — `learner.profile_evidence` is the high-row table; store **aggregated** evidence
  (counts + windows + a small sample of refs), never one row per source event. Index by
  `(profile_id, facet_key)`.
- **Reliability** — Deriver runs MUST be at-least-once with per-facet idempotency keys so a retried
  job doesn't double-write. A failing deriver MUST NOT block other facets (isolate per facet).
- **Observability** — Emit `learner_profile_recompute_total{facet,mode,result}`,
  `learner_profile_recompute_duration_seconds{facet}`, and `learner_profile_facets_populated`
  gauges via `telemetry` (prefix `lextures_`; never put `user_id` in labels). Log each recompute
  at DEBUG with hashed user id, facet, signal counts, old/new confidence.
- **Maintainability** — Engine in `server/internal/service/learnerprofile/`; derivers in
  `.../derivers/<facet>.go`; repo in `server/internal/repos/learnerprofile/`. No business logic in
  handlers.
- **Internationalization** — Insight `label`s and any human strings are i18n keys resolved at read
  time; stored `value` is language-neutral data. Time-of-day/rhythm math is tz-aware per user.
- **Backward compatibility** — Purely additive: new `learner` schema, no existing table altered
  except one feature-flag column on `settings.platform_app_settings`.

## 7. Acceptance Criteria

- **AC-1.** *Given* a user with sufficient signals in one facet, *when* the incremental deriver
  runs, *then* a `learner.profiles` row, a `learner.profile_facets` row for that facet, ≥1
  `profile_insights`, and ≥1 `profile_evidence` row exist, written in one transaction.
- **AC-2.** *Given* an insight, *when* fetched via the API, *then* it includes an `evidence` array
  where each item names `source_kind`, `source_table`, `observation_count`, and a time window.
- **AC-3.** *Given* the same signals, *when* a deriver runs twice, *then* the resulting facet
  summary and insight values are byte-identical (idempotency).
- **AC-4.** *Given* a facet below its minimum signal threshold, *when* derived, *then* the facet
  state is `insufficient_data` and no fabricated insights are returned.
- **AC-5.** *Given* user A, *when* they request `GET /api/v1/me/learner-profile`, *then* they
  receive only their own profile; there is no endpoint by which A can read B's profile (403/absent).
- **AC-6.** *Given* a user erasure (LP08), *when* it runs, *then* all `learner.*` rows for that
  user are deleted and a subsequent read returns an empty/absent profile.
- **AC-7.** *Given* one deriver throws, *when* a recompute runs, *then* the other facets still
  update and the failure is recorded in metrics/logs (no all-or-nothing).

## 8. Data Model

```sql
-- server/migrations/358_learner_profile_core.sql
CREATE SCHEMA IF NOT EXISTS learner;

CREATE TABLE learner.profiles (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL UNIQUE REFERENCES "user".users (id) ON DELETE CASCADE,
    status            TEXT NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'paused')),   -- LP08 flips to paused
    last_computed_at  TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE learner.profile_facets (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id        UUID NOT NULL REFERENCES learner.profiles (id) ON DELETE CASCADE,
    facet_key         TEXT NOT NULL,                 -- registry-controlled
    state             TEXT NOT NULL DEFAULT 'ok'
                        CHECK (state IN ('ok', 'insufficient_data')),
    summary           JSONB NOT NULL DEFAULT '{}',
    confidence        NUMERIC(4,3) NOT NULL DEFAULT 0 CHECK (confidence BETWEEN 0 AND 1),
    computed_version  INTEGER NOT NULL DEFAULT 1,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (profile_id, facet_key)
);

CREATE TABLE learner.profile_insights (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    facet_id          UUID NOT NULL REFERENCES learner.profile_facets (id) ON DELETE CASCADE,
    insight_key       TEXT NOT NULL,
    label_i18n_key    TEXT NOT NULL,
    value             JSONB NOT NULL DEFAULT '{}',
    confidence        NUMERIC(4,3) NOT NULL DEFAULT 0 CHECK (confidence BETWEEN 0 AND 1),
    salience          INTEGER NOT NULL DEFAULT 0,     -- display order, higher = more prominent
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (facet_id, insight_key)
);

CREATE TABLE learner.profile_evidence (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    insight_id        UUID NOT NULL REFERENCES learner.profile_insights (id) ON DELETE CASCADE,
    source_kind       TEXT NOT NULL,   -- 'engagement_event'|'quiz_attempt'|'learner_concept_state'|'notebook'|'enrollment'|'feed'
    source_table      TEXT NOT NULL,   -- fully-qualified table the count came from
    course_id         UUID REFERENCES course.courses (id) ON DELETE SET NULL,
    observation_count INTEGER NOT NULL DEFAULT 0,
    window_start      TIMESTAMPTZ,
    window_end        TIMESTAMPTZ,
    contribution      NUMERIC(4,3),    -- relative weight in the insight
    sample_refs       JSONB,           -- small illustrative sample (e.g. up to 5 source ids)
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_lp_facets_profile     ON learner.profile_facets (profile_id, facet_key);
CREATE INDEX idx_lp_insights_facet     ON learner.profile_insights (facet_id, salience DESC);
CREATE INDEX idx_lp_evidence_insight   ON learner.profile_evidence (insight_id);

-- Feature flag (default off).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS learner_profile_enabled BOOLEAN;
COMMENT ON COLUMN settings.platform_app_settings.learner_profile_enabled IS
    'Enables the autonomous learner profile (LP epic).';
```

Backfill: none required. On flag enable, the nightly full recompute populates profiles from
existing signals; profiles otherwise materialise lazily.

## 9. API Surface

```
GET /api/v1/me/learner-profile
  Auth: Bearer JWT (self only)
  200: { profile: { status, lastComputedAt, facets: FacetSummary[] } }
      | 200 with facets:[] and status:"insufficient_data" when nothing derived yet

GET /api/v1/me/learner-profile/facets/{facetKey}
  200: { facet: Facet, insights: Insight[] }   404 if facet unknown

GET /api/v1/me/learner-profile/facets/{facetKey}/evidence
  200: { insightKey: Evidence[] ... }           -- provenance drill-down

interface Insight {
  insightKey: string; label: string; value: unknown;
  confidence: number; salience: number;
  evidence: Evidence[];
}
interface Evidence {
  sourceKind: string; sourceTable: string; courseId?: string;
  observationCount: number; windowStart?: string; windowEnd?: string; contribution?: number;
}
```

Internal Go interface:

```go
// server/internal/service/learnerprofile/service.go
type FacetDeriver interface {
    Key() string
    Derive(ctx context.Context, userID uuid.UUID) (FacetResult, error) // reads signals, returns facet+insights+evidence
    MinSignals() int
    Version() int
}
type LearnerProfileService interface {
    Get(ctx, userID) (Profile, error)
    RecomputeIncremental(ctx, userID, changedFacets ...string) error
    RecomputeAll(ctx) error   // nightly
    Pause(ctx, userID) error; Resume(ctx, userID) error; Erase(ctx, userID) error // LP08 hooks
}
```

All routes documented in OpenAPI consistent with `server/internal/httpserver/` + `openapi/`.

## 10. UI / UX

No UI in this plan. It exposes the read model and empty/insufficient states that LP07 renders:
`status:"insufficient_data"` → LP07 shows a "your profile is still building" empty state. This
plan defines the JSON contract LP07/LP10 consume.

## 11. AI / ML Considerations

The engine itself is deterministic (no LLM). Derivers use explainable statistics, not opaque
models, so every value is reconstructable from evidence (FR-10). LP09 may later add an LLM
*summary* on top, but the underlying facets must remain rule-based and auditable. Any future LLM
summarisation MUST redact `user_id` and send only aggregated facet values, never raw event rows.

## 12. Integration Points

- Reads (via repos, never bypassing authz): `analytics.engagement_events`,
  `course.learner_concept_states`, `course.quiz_attempts`, notebooks (`student_notebooks`),
  `course.course_enrollments`, feed tables. Actual columns owned by each facet plan (LP02–LP06).
- Job queue `server/migrations/338_job_queue.sql`; scheduler `340_scheduler.sql` for the nightly
  run; `server/internal/workers/` pattern for the recompute worker.
- Telemetry: `server/internal/telemetry` (`RecordBusinessEvent`, metrics registry).
- Feature flags: `settings.platform_app_settings` + `clients/web/src/lib/platform-features.ts`.

## 13. Dependencies & Sequencing

- **After:** 1.1 (mastery) and 9.7 (engagement) — both shipped.
- **Before:** LP02–LP10 (all depend on this substrate).
- **Shared infra:** Postgres, job queue, scheduler, telemetry — all present.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Nightly recompute too slow at scale | M | H | Incremental-first; only full-recompute stale profiles (`last_computed_at` cutoff); shard by user-id hash |
| Evidence table row explosion | M | H | Store aggregated evidence (counts + windows + tiny sample), never per-event rows |
| Deriver upgrade mixes old/new insights | M | M | `computed_version` per facet; recompute forces re-derive; never merge versions |
| "Profiling" flagged by privacy review | M | H | Provenance-by-construction + LP08 controls + DSAR/erase hooks from day one |
| Low-signal learners get misleading facets | M | M | `insufficient_data` state + `MinSignals()` gate (FR-8) |

## 15. Rollout Plan

- **Flag:** `learner_profile_enabled` (default `true`).
- **Sequencing:** migration → engine + repo + one no-op deriver behind flag → enable for internal
  org → nightly recompute validated → enable for pilot cohort → GA after LP07 + LP08 ship.
- **Pilot:** 2–3 self-learner accounts + one instructor test course.
- **GA criteria:** recompute p95 within targets; read endpoint p95 ≤ 80 ms; LP08 controls live.
- **Rollback:** disable flag (reads return empty; tables retained, no data loss).

## 16. Test Plan

- **Unit** — deriver registry; idempotency of a fake deriver; evidence aggregation; sufficiency
  gate; version bump forces recompute.
- **Integration** — signals present → recompute writes facet/insights/evidence atomically; one
  deriver panics → others still commit; erase removes all `learner.*` rows.
- **End-to-end** — Playwright stub: seed signals, run recompute, `GET /me/learner-profile` returns
  populated facets with evidence.
- **Security** — authz: self-only read; no cross-user path; job runs row-scoped.
- **Performance / load** — recompute 100 k synthetic learners within window; read p95 under load.
- **Manual exploratory** — force `insufficient_data`; corrupt one facet, verify isolation.

## 17. Documentation & Training

- Internal runbook `docs/runbooks/learner-profile.md`: force recompute, inspect a profile, clear a
  user, read metrics.
- Engineering doc: "How to add a facet deriver" (the extension contract).
- OpenAPI schemas for the read endpoints.

## 18. Open Questions

1. Nightly full recompute vs. rolling incremental-only with a staleness sweep — start with both,
   measure, possibly drop nightly.
2. Should evidence retain `sample_refs` (source ids) or only counts, given privacy minimisation?
   (Leaning: tiny sample, redactable.)
3. Is the profile strictly single-tenant per user, or must it span orgs for users in multiple
   tenants? (Assume single active tenant for v1.)
4. Where does the tz for rhythm math come from — user setting, org, or last-seen locale?

## 19. References

- Existing files: `server/migrations/175_engagement_events.sql`, `.../087_learner_model.sql`,
  `.../338_job_queue.sql`, `.../340_scheduler.sql`, `server/internal/telemetry/`,
  `server/internal/workers/`.
- Related plans: [1.1 learner model](../../completed/01-adaptive-learning-core/1.1-learner-model-knowledge-state.md),
  [9.7 engagement](../../completed/09-analytics-reporting/9.7-engagement-metrics.md),
  [LP07](LP07-settings-page-transparency-ui.md), [LP08](LP08-privacy-consent-controls.md).
- External: FERPA 34 CFR Part 99; GDPR Art. 4(4) & Art. 22 (profiling / automated decisions).
