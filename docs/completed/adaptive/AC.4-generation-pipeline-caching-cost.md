# AC.4 — Generation Pipeline, Caching & Cost Controls

> Implementation plan. Source: operability layer for the AC.3 engine. Folder overview: [README](../../plan/adaptive/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AC.4 |
| **Section** | Adaptive Content Engine (ACE) |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | AC.3; job queue (17.3) if present, else Postgres-backed worker |
| **Unblocks** | AC.6 (fast serving), AC.9 (cost/cache metrics) |

---

## 1. Problem Statement

AC.3 can generate a variant synchronously, but making a student wait 5–8 seconds for a model call at page-load is unacceptable, and calling the model once per student would be ruinously expensive and slow. This story wraps the engine in an **asynchronous pipeline**: pre-generate the likely variants for a unit ahead of demand, cache them by `profile_signature`, deduplicate concurrent generation of the same signature, and enforce per-course token/cost **budgets** so a course cannot run away. It turns AC.3 from "possible" into "fast, cheap, and safe under load."

## 2. Goals

- Serve variants from cache in the common case; generate on-demand only for a cache miss, and never twice for the same signature.
- Pre-warm the top-N expected signatures for a unit when it is activated or its pre-assessment cohort grows.
- Enforce per-course `monthly_token_budget` and a global rate limit; degrade gracefully (serve base) when exhausted.
- Provide retry-with-backoff for transient model failures without blocking learners.
- Emit the cost/cache/queue metrics AC.9 reports on.

## 3. Non-Goals

- The generation logic, prompt, or fidelity gate (AC.3).
- The serving decision and student UI (AC.6).
- Dashboards (AC.9 renders the metrics this story emits).
- Cross-course/global budget policy beyond a platform rate ceiling (admin policy is AC.9/AC.8).

## 4. Personas & User Stories

- **As a student**, I want the adapted page to load as fast as a normal page — no visible AI wait.
- **As an instructor**, I want variants ready before my class opens the unit, and I want to see how much of my budget I've used.
- **As a platform admin**, I want a hard ceiling so one course can't consume the whole AI bill or rate-limit the provider for everyone.
- **As an SRE**, I want failed generations retried and observable, and a way to pause the pipeline instantly.

## 5. Functional Requirements

- **FR-1.** The system MUST enqueue a generation job on: unit activation, pre-assessment submission producing a *new* signature with no cached variant, base-content edit (regenerate needed signatures), and instructor "pre-warm now".
- **FR-2.** The system MUST dedupe: a signature already `pending`/`generating`/`approved` MUST NOT enqueue a second job (advisory lock or unique partial index on `(unit_id, profile_signature)` for in-flight jobs).
- **FR-3.** On a serve request (AC.6) with no ready variant, the system MUST return base content immediately and enqueue generation for next time — never block on the model.
- **FR-4.** The system MUST enforce `monthly_token_budget` per course: before a call, check accumulated `ai_usage_log` tokens for `feature='adaptive_content'` in the current period; if the projected call would exceed budget, skip generation and serve base, recording a `budget_exhausted` event.
- **FR-5.** The system MUST enforce a global concurrency/rate limit for adaptive-content model calls (token bucket) independent of per-course budgets, to protect the shared provider quota.
- **FR-6.** The system MUST retry transient failures (timeouts, 5xx, rate-limit) with exponential backoff and a max attempt count; permanent failures (fidelity/safety rejection) MUST NOT retry.
- **FR-7.** The system MUST expose pipeline controls to admins: pause/resume adaptive-content generation platform-wide (distinct from the AC.1 kill-switch, which also blocks *serving*), and per-course pause.
- **FR-8.** Pre-warming MUST be bounded: at most `max_prewarm_variants` (default = distinct-signature cap from AC.2, ~12) per unit, prioritized by observed signature frequency in the cohort.
- **FR-9.** The system MUST record queue depth, generation latency, cache hit/miss, retries, and budget state as metrics.

## 6. Non-Functional Requirements

- **Performance** — Cache-hit serve p95 ≤ 30 ms (indexed lookup on `(unit_id, profile_signature, status='approved'|'auto_served')`). Job pickup latency p95 ≤ 2 s at normal load. Pre-warm a unit's top signatures within 2 minutes of activation.
- **Security** — Job payloads carry ids only (no PII). Admin pause controls require platform-admin permission. Budget counters are server-authoritative.
- **Privacy & Compliance** — No new data classes; jobs reference existing rows. Budget/period accounting reuses `ai_usage_log`.
- **Accessibility** — N/A (backend); the student sees either cached variant or base, both already accessible.
- **Scalability** — Worker pool horizontally scalable; dedupe prevents thundering-herd on a hot unit; global token bucket caps provider pressure. Queue is Postgres-backed (`SKIP LOCKED`) if the shared job queue (17.3) is unavailable.
- **Reliability** — At-least-once job semantics with idempotent generation (signature dedupe makes re-runs safe). A crashed worker's job is reclaimed after a visibility timeout. Pause is immediate and durable.
- **Observability** — Gauges `adaptive_content.queue_depth`, `.inflight`, `.tokens_used_period{course}`, `.budget_remaining{course}`; counters `.cache_hit`, `.cache_miss`, `.job_retry`, `.budget_exhausted`; histogram `.job_latency_ms`. Alert on queue depth > threshold, budget_exhausted spikes, retry storms.
- **Maintainability** — Worker in `service/adaptivecontent/pipeline.go`; reuses the platform job-queue abstraction if present, else a thin `repos/adaptivecontent/jobs.go` with `SKIP LOCKED`.
- **Internationalization** — N/A.
- **Backward compatibility** — Purely additive; if the pipeline is paused or absent, AC.6 still works (serves base + logs miss). No change to base content behavior.

## 7. Acceptance Criteria

- **AC-1.** *Given* a ready approved variant for a signature, *When* a student is served, *Then* the response uses the cached variant with no model call and p95 ≤ 30 ms.
- **AC-2.** *Given* 50 students submit pre-checks yielding the same signature simultaneously, *When* processed, *Then* exactly one generation job runs and all 50 later serve the same variant.
- **AC-3.** *Given* a cache miss at serve time, *When* the student loads the unit, *Then* they immediately get base content and a generation job is enqueued.
- **AC-4.** *Given* a course at 100% of its `monthly_token_budget`, *When* a new signature needs generation, *Then* no model call occurs, base is served, and a `budget_exhausted` event is recorded.
- **AC-5.** *Given* a transient provider 503, *When* a job runs, *Then* it retries with backoff up to the max and succeeds or dead-letters — without any learner waiting.
- **AC-6.** *Given* an admin pauses the pipeline, *When* jobs are pending, *Then* no new generation starts until resume, and serving falls back to base/existing cache.
- **AC-7.** *Given* a unit's base content is edited, *When* saved, *Then* affected signatures are re-enqueued and old variants are `superseded` (not served) until regenerated.

## 8. Data Model

Reserves `442_adaptive_content_jobs.sql`.

```sql
-- 442_adaptive_content_jobs.sql
CREATE TABLE course.adaptive_content_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    profile_signature TEXT NOT NULL,
    content_version INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','generating','done','failed','dead_letter','canceled')),
    attempts SMALLINT NOT NULL DEFAULT 0,
    priority SMALLINT NOT NULL DEFAULT 0,
    run_after TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_by TEXT,
    locked_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- One in-flight/complete job per (unit, signature, content_version) → dedupe.
CREATE UNIQUE INDEX ux_ac_jobs_dedupe
    ON course.adaptive_content_jobs (unit_id, profile_signature, content_version)
    WHERE status IN ('pending','generating','done');
CREATE INDEX idx_ac_jobs_pickup ON course.adaptive_content_jobs (status, run_after) WHERE status = 'pending';

-- Per-course generation controls + period accounting cache.
ALTER TABLE course.adaptive_content_settings
    ADD COLUMN IF NOT EXISTS generation_paused BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS max_prewarm_variants SMALLINT NOT NULL DEFAULT 12,
    ADD COLUMN IF NOT EXISTS budget_period_start DATE,
    ADD COLUMN IF NOT EXISTS tokens_used_period BIGINT NOT NULL DEFAULT 0;

-- Platform-wide pipeline switch (distinct from AC.1 kill-switch).
-- Stored in settings.platform_app_settings for admin control.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS adaptive_content_generation_paused BOOLEAN;
```

**Backfill:** none. `tokens_used_period` is a cache derivable from `ai_usage_log`; a nightly job reconciles it.

## 9. API Surface

```
POST /api/v1/courses/{course_code}/adaptive-content/units/{id}/prewarm     instructor
   -> enqueues generation for the unit's top expected signatures
GET  /api/v1/courses/{course_code}/adaptive-content/budget                  instructor
   -> { monthlyTokenBudget, tokensUsedPeriod, budgetRemaining, periodStart, generationPaused }
PATCH /api/v1/courses/{course_code}/adaptive-content/settings              instructor
   -> can set generationPaused, maxPrewarmVariants
-- Admin platform control (existing admin console surface):
PATCH /api/v1/admin/adaptive-content   admin  -> { generationPaused: bool }
```

Internal: `pipeline.Enqueue(unitID, signature, version, priority)`, `pipeline.Worker.RunOnce()` (SKIP LOCKED pickup), `budget.CheckAndReserve(courseID, estTokens)`.

## 10. UI / UX

- **Instructor budget widget** (in the AC.5 authoring workspace and course settings): a meter showing tokens used / budget this period, "pre-warm now" button, and a "generation paused" toggle with explanation.
- **States:** budget healthy (green), > 80% (amber), exhausted (red + "students see the original until next period or budget increase"). Pre-warm shows a progress toast ("Generating 9 variants…").
- **Admin console:** a global "Adaptive Content generation" pause switch with current queue depth and cost-this-month, alongside the existing AI governance panel.
- **Mobile:** budget meter collapses to a compact chip.
- **Accessibility:** meter has text value + ARIA; pause toggles are labeled switches.

## 11. AI / ML Considerations

- No new model calls — this story *schedules* AC.3's calls. It owns **cost governance**: token estimation before a call, budget reservation, global rate limiting, and reconciliation against `ai_usage_log` (the source of truth for actual spend).
- Pre-warm prioritization can later use signature-frequency prediction, but v1 uses observed cohort frequency (simple counts), keeping it explainable.

## 12. Integration Points

- Platform job queue (`17.3` if shipped) or Postgres `SKIP LOCKED` worker in `repos/adaptivecontent/jobs.go`.
- `service/adaptivecontent/pipeline.go`, `budget.go` (new); calls `generate.GenerateVariant` (AC.3).
- `analytics.ai_usage_log` — spend source of truth; nightly reconcile job.
- `server/internal/httpserver/admin.go` — platform pause control (near AI governance).
- `server/internal/telemetry/` — metrics registration (per the observability layer).
- `server/migrations/442_adaptive_content_jobs.sql` (+ down).
- Related: [AC.3](AC.3-content-generation-engine.md), [AC.6](AC.6-student-runtime-and-transparency.md), [AC.9](../../plan/adaptive/AC.9-analytics-reporting-and-operability.md).

## 13. Dependencies & Sequencing

- **Must ship after:** AC.3.
- **Must ship before:** AC.6 relies on cache-first serving for acceptable latency; AC.9 charts these metrics.
- **Shared infra:** job queue (or Postgres worker), metrics pipeline, `ai_usage_log`.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Thundering herd on a hot unit at class start | M | M | Dedupe unique index + advisory locks; pre-warm on activation |
| Budget accounting drift vs. real spend | M | M | `ai_usage_log` is source of truth; nightly reconcile; conservative pre-call estimate |
| Stuck `generating` jobs after worker crash | M | M | Visibility timeout reclaim; `locked_at` staleness sweep |
| Pre-warm wastes budget on rare signatures | M | M | Cap `max_prewarm_variants`; prioritize by frequency; on-demand for the long tail |
| Provider global rate-limit hit | L | M | Global token bucket; backoff; admin pause |

## 15. Rollout Plan

- **Feature flag:** course `adaptive_content_enabled` (AC.1) gates whether jobs enqueue at all; `generation_paused` (course) and the platform pause give layered off-switches.
- **Sequencing:** deploy migration → ship worker + budget in "shadow" (generate + cache, but AC.6 not yet serving) → validate cache hit rates and cost on pilot → enable serving (AC.6).
- **Pilot cohort:** the AC.3 pilot courses under real class load.
- **GA criteria:** cache hit ≥ 90% after warm-up; zero learner-visible model waits; budget enforcement verified; retry/backoff proven under injected failures.
- **Rollback:** platform pause (stop generating) and/or AC.1 kill-switch (stop serving); cache persists; base content always available.

## 16. Test Plan

- **Unit** — dedupe index behavior; budget check-and-reserve math across period boundary; backoff schedule; SKIP LOCKED pickup fairness.
- **Integration** — concurrent identical signatures → one job; cache miss → base + enqueue; budget exhausted → base + event; content edit → supersede + re-enqueue; worker crash → job reclaimed.
- **End-to-end** — Playwright: instructor clicks pre-warm, watches variants become ready; opens unit as a warmed student → instant adapted page.
- **Security** — admin-only platform pause; instructor-only per-course pause/pre-warm; job payloads PII-free.
- **Performance / load** — k6/soak: 5k students hitting a unit; cache hit p95 ≤ 30 ms; queue drains within SLA; provider rate stays under ceiling.
- **Manual exploratory** — inject provider 503s and timeouts; confirm retries and no learner blocking; exhaust a budget and confirm graceful base fallback.

## 17. Documentation & Training

- Instructor guide: "Pre-warming, budgets, and what happens when I run out."
- Admin runbook: pausing generation, reading queue/cost metrics, budget increases.
- SRE runbook: dead-letter triage, stuck-job sweep, provider rate-limit response.

## 18. Open Questions

1. Reuse the platform job queue (17.3) or a dedicated Postgres worker? (Prefer reuse if available; the plan supports either.)
2. Should budgets be per-course, per-org, or both? (v1 per-course to match the course-level flag; org rollup in AC.9.)
3. Is a Redis cache layer worth it over the Postgres `content_variants` table for serve latency? (Measure first; add only if p95 target missed.)
4. How aggressively should we pre-warm before any student has taken the pre-check (cold start)? (v1: pre-warm the neutral + a few canonical profiles; refine with data.)

## 19. References

- Existing files: `analytics.ai_usage_log` (`server/migrations/281_ai_usage_logs.sql`), `server/internal/telemetry/`, `server/internal/httpserver/admin.go`.
- Related plans: [AC.3](AC.3-content-generation-engine.md), [AC.6](AC.6-student-runtime-and-transparency.md), [AC.9](../../plan/adaptive/AC.9-analytics-reporting-and-operability.md); platform job queue plan `17.3`.
- External: token-bucket rate limiting; Postgres `SELECT … FOR UPDATE SKIP LOCKED` queue pattern.
