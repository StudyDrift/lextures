# MC.8 — Publish Pipeline: Scheduling, Rebuild Dispatch & Cache Invalidation

> Completed implementation plan. Source: [marketing content plan index](../../plan/marketing-content/README.md) §Plans.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC.8 |
| **Section** | MC — Marketing Content Platform |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status** | COMPLETE — transactional publish events, scheduled publishing, coalesced GitHub dispatch, status APIs, and Pages deploy integration |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Server platform + Web platform |
| **Depends on** | MC.2, MC.7 |
| **Unblocks** | MC.11, MC.15 |

---

## 1. Problem Statement

After MC.7, the site *can* build from the database — but only when someone pushes code, because
`pages-www.yml` triggers on `www/**` paths. A content expert who clicks **Publish** would see
nothing change on `lextures.com` until the next unrelated code deploy, which is worse than today's
workflow because it looks like it worked. Scheduled publishing ("go live Monday 09:00") has no
mechanism at all, and the object cache in front of the public content API would keep serving the old
payload to the next build. Publishing needs to be an end-to-end pipeline with a visible status.

## 2. Goals

- Make **Publish** in the workspace result in a live page, automatically, within a predictable time
  window, with the state of that process visible to the person who clicked it.
- Support scheduled publish/unpublish with a worker that is correct across restarts and duplicate
  runs.
- Debounce rebuilds so publishing five articles in ten minutes triggers one build, not five.
- Invalidate the content API cache the instant content state changes, so a build never fetches stale
  data.
- Keep IndexNow/sitemap submission working for newly published URLs (SEO.2 FR-16–19).
- Fail safe: a broken dispatch must be loud (alert + visible status), never silent.

## 3. Non-Goals

- No incremental/partial site deploys — GitHub Pages replaces `dist/` wholesale.
- No preview deployments per draft (preview is in-app; see MC.10).
- No content CDN purge (the site is static on Pages; assets are content-addressed).
- No change to how code deploys work.
- No multi-environment publishing matrix (one production site, one staging site).

## 4. Personas & User Stories

- **As a content expert**, I want to click Publish and see "Publishing… live in ~6 minutes" and then
  "Live", so I know whether to tell the team it shipped.
- **As a content expert**, I want to schedule an announcement for Monday 09:00 and go home Friday.
- **As a marketing lead**, I want a batch of five articles to publish as one deploy, so the site does
  not thrash.
- **As an SRE**, I want a failed rebuild to page nobody at 3am but to be clearly visible in the
  workspace and in monitoring, since the public site is still serving the previous version.
- **As an SEO owner**, I want new URLs submitted to IndexNow on the deploy that first contains them.

## 5. Functional Requirements

- **FR-1.** A `content_publish_events` table MUST record every state change with actor, article,
  action, timestamp and the resulting build dispatch (if any).
- **FR-2.** Publishing, unpublishing, archiving, restoring or editing a **published** article MUST
  enqueue a rebuild request, coalesced into a single pending build.
- **FR-3.** Rebuild dispatch MUST debounce with a configurable quiet period (default 3 minutes,
  max wait 15 minutes): the build fires when no further publish has occurred for the quiet period, or
  when the max wait elapses — whichever comes first.
- **FR-4.** Dispatch MUST call the GitHub `repository_dispatch` API with
  `event_type: "marketing-content-publish"` and a payload listing the changed paths, using a
  fine-grained token stored in platform settings (encrypted) with **only** the "Contents: read &
  write" / workflow dispatch scope required.
- **FR-5.** `.github/workflows/pages-www.yml` MUST accept `repository_dispatch` (type
  `marketing-content-publish`) and `workflow_dispatch` in addition to its current triggers, building
  with `WWW_CONTENT_SOURCE=api`.
- **FR-6.** The dispatcher MUST record `build_id`, `dispatched_at`, `status ∈ {pending, dispatched,
  running, succeeded, failed, timed_out}` and MUST poll (or receive) the workflow run conclusion,
  updating status until terminal.
- **FR-7.** A scheduled job `marketing_content_publish_due` MUST run every minute, transition
  articles whose `scheduled_for <= now()` to `published` (through the same service path as a manual
  publish, so validation and revisions apply), and enqueue a rebuild.
- **FR-8.** The scheduled publisher MUST be idempotent and safe under concurrent workers: it MUST
  claim rows with `SELECT … FOR UPDATE SKIP LOCKED` and MUST never double-publish.
- **FR-9.** Any content state change MUST invalidate the MC.3 object-cache entries for `/index`, the
  affected article, its category and author, within 5 seconds.
- **FR-10.** `GET /api/v1/admin/marketing/builds` MUST list recent build dispatches with status,
  trigger reason, changed paths and workflow run URL; `POST /api/v1/admin/marketing/builds` MUST
  allow a manual rebuild (`…:publish` permission, rate-limited to 6/hour).
- **FR-11.** The workspace MUST show live publish status per article (`Draft`, `Scheduled for …`,
  `Publishing…`, `Live`, `Publish failed`) derived from article status + latest build state.
- **FR-12.** When a build fails or times out (> 30 min), the system MUST mark the dispatch `failed`,
  surface it in the workspace with the run URL, and emit an alert metric; article status MUST remain
  `published` (the DB is correct; only the site is behind).
- **FR-13.** The deploy workflow MUST continue to run IndexNow submission for URLs new since the last
  manifest (existing `submit-indexnow.mjs` step), which now naturally covers DB-published articles.
- **FR-14.** Unpublishing MUST be treated as urgent: the quiet period MUST be bypassed
  (immediate dispatch) so a page can be pulled quickly, and the workspace MUST say how long removal
  will take.
- **FR-15.** All dispatch configuration (token, repo, workflow ref, quiet period, max wait) MUST live
  in platform settings and be editable by a platform admin, with the token write-only in the UI.
- **FR-16.** If dispatch is not configured, publishing MUST still succeed and the workspace MUST show
  "Published — site rebuild not configured" rather than failing the publish.

## 6. Non-Functional Requirements

- **Performance** — Publish API call returns in < 300 ms (dispatch is asynchronous). Time from
  publish to live page: p50 < 8 min, p95 < 15 min (dominated by the existing `www` build +
  Lighthouse job).
- **Security** — The GitHub token is a secret at rest (encrypted with the existing platform-settings
  crypto), never returned by any API, and scoped to a single repository. Dispatch payloads contain
  only paths, never content. `POST /builds` is permission-gated and rate-limited to prevent CI abuse.
- **Privacy & Compliance** — Publish events are audit records retained per the admin-audit policy.
  No learner data.
- **Accessibility** — Status changes in the workspace announce via `aria-live="polite"`; "Publishing…"
  is not conveyed by colour alone; the relative time ("live in ~6 min") has an accessible absolute
  timestamp.
- **Scalability** — At most a few dozen publishes per day; the queue is a single table with a small
  working set.
- **Reliability** — At-least-once dispatch with idempotency: a duplicate dispatch produces a
  redundant build, never a wrong site. Worker crash mid-claim leaves the row claimable after the lock
  is released. Scheduled publishing is exactly-once by row claim.
- **Observability** — `marketing_content_builds_total{status}`, `…_build_latency_seconds`
  (publish→live), `…_scheduled_publishes_total`, `…_dispatch_failures_total`; alert when
  `dispatch_failures_total` > 0 in 15 min or when a build stays `running` > 30 min; the `www` build
  summary's `fallbackUsed: true` also raises an alert (MC.7 FR-17).
- **Maintainability** — Dispatcher is one service (`internal/service/marketingpublish`) with a
  provider interface (`GitHubDispatcher`), so a future host (Cloudflare Pages, Netlify) is a new
  implementation rather than a rewrite.
- **Internationalization** — Status strings are i18n keys; relative time formatting uses the
  viewer's locale and timezone (`internal/l10n` + client `Intl`).
- **Backward compatibility** — The workflow keeps its `push` and `pull_request` triggers; code
  deploys behave exactly as today.

## 7. Acceptance Criteria

- **AC-1.** *Given* a draft article, *when* it is published, *then* a `content_publish_events` row
  exists, a build dispatch is created in `pending`, and the API responds in < 300 ms.
- **AC-2.** *Given* three publishes within 90 seconds and a 3-minute quiet period, *when* the
  dispatcher runs, *then* exactly one `repository_dispatch` call is made listing all three paths.
- **AC-3.** *Given* continuous publishes for 20 minutes, *when* the max wait is 15 minutes, *then* a
  build fires at 15 minutes despite ongoing activity.
- **AC-4.** *Given* an article scheduled for a past minute, *when* the job runs, *then* the article is
  `published` exactly once even with two worker instances running (concurrency test).
- **AC-5.** *Given* a scheduled article that fails validation at publish time, *when* the job runs,
  *then* the article stays `scheduled`, the failure is recorded on the publish event, and the
  workspace shows "Scheduled publish failed — fix findings".
- **AC-6.** *Given* a publish, *when* the next `/api/v1/public/content/index` request is made, *then*
  it reflects the change (cache invalidated) within 5 seconds.
- **AC-7.** *Given* a `repository_dispatch` is sent, *when* the workflow runs, *then* it builds with
  `WWW_CONTENT_SOURCE=api`, deploys, and the dispatch row reaches `succeeded` with the run URL
  recorded.
- **AC-8.** *Given* a workflow failure, *when* the conclusion is observed, *then* the dispatch is
  `failed`, the workspace shows the failure with a link, and the metric increments; the article
  remains `published`.
- **AC-9.** *Given* an unpublish, *when* it happens, *then* a dispatch fires immediately (no quiet
  period) and the workspace states the expected removal time.
- **AC-10.** *Given* dispatch is unconfigured, *when* an article is published, *then* publish
  succeeds and the workspace shows "site rebuild not configured".
- **AC-11.** *Given* seven manual rebuild requests in an hour with a limit of six, *when* the seventh
  is sent, *then* it is rejected with `429`.
- **AC-12.** *Given* a newly published URL, *when* the deploy completes, *then* the existing IndexNow
  step submits it (manifest diff contains it).

## 8. Data Model

Migration `481_marketing_content_publish_pipeline.sql` (indicative number):

```sql
CREATE TABLE marketing.content_publish_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id   UUID REFERENCES marketing.content_articles (id) ON DELETE SET NULL,
    path         TEXT NOT NULL,
    action       TEXT NOT NULL CHECK (action IN ('publish','unpublish','archive','update','schedule','scheduled_publish')),
    actor_id     UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    build_id     UUID,
    error        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_mc_publish_events_recent ON marketing.content_publish_events (created_at DESC);

CREATE TABLE marketing.content_builds (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status         TEXT NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending','dispatched','running','succeeded','failed','timed_out')),
    reason         TEXT NOT NULL,                       -- 'publish' | 'unpublish' | 'manual' | 'scheduled'
    paths          TEXT[] NOT NULL DEFAULT '{}',
    urgent         BOOLEAN NOT NULL DEFAULT FALSE,
    not_before     TIMESTAMPTZ NOT NULL DEFAULT now(),  -- quiet-period deadline
    deadline       TIMESTAMPTZ NOT NULL,                -- max-wait deadline
    dispatched_at  TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    provider_run_id TEXT,
    provider_run_url TEXT,
    error          TEXT,
    requested_by   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_mc_builds_one_pending ON marketing.content_builds ((status))
    WHERE status = 'pending';                            -- coalescing invariant
CREATE INDEX idx_mc_builds_recent ON marketing.content_builds (created_at DESC);
```

Platform settings additions (same migration): `marketing_build_provider` (`github` | `none`),
`marketing_build_repo`, `marketing_build_workflow_ref`, `marketing_build_token_encrypted`,
`marketing_build_quiet_seconds` (default 180), `marketing_build_max_wait_seconds` (default 900).

Backfill: none.

## 9. API Surface

| Verb | Path | Permission | Notes |
|---|---|---|---|
| GET | `/api/v1/admin/marketing/builds` | `…:view` | recent builds + status |
| POST | `/api/v1/admin/marketing/builds` | `…:publish` | manual rebuild, 6/hour |
| GET | `/api/v1/admin/marketing/publish-events` | `…:view` | filterable by article |
| GET | `/api/v1/admin/marketing/articles/{id}` | `…:view` | gains `liveStatus` + `latestBuild` |

```ts
type ContentBuild = {
  id: string
  status: 'pending'|'dispatched'|'running'|'succeeded'|'failed'|'timed_out'
  reason: 'publish'|'unpublish'|'manual'|'scheduled'
  urgent: boolean
  paths: string[]
  dispatchedAt: string | null; completedAt: string | null
  providerRunUrl: string | null; error: string | null
  estimatedLiveAt: string | null
}
type LiveStatus = 'draft'|'scheduled'|'publishing'|'live'|'publish_failed'|'rebuild_not_configured'
```

Outbound (server → GitHub):

```
POST https://api.github.com/repos/{owner}/{repo}/dispatches
{ "event_type": "marketing-content-publish",
  "client_payload": { "buildId": "…", "paths": ["/blog/x", "/docs/courses/y"], "reason": "publish" } }
```

Scheduled job: `marketing_content_publish_due`, spec `* * * * *`, `DefaultEnabled: true`, registered
in `internal/scheduler/config.go` alongside the existing builtins. A second job
`marketing_content_build_dispatch`, spec `* * * * *`, drives the debounce/dispatch/poll state machine.

## 10. UI / UX

Surfaced in MC.9/MC.10; specified here:

1. **Publish dialog** — confirms target ("This will appear on lextures.com"), shows validation
   findings if any, and offers **Publish now** / **Schedule**.
2. **Status pill** on every row and in the editor header: `Draft`, `Scheduled · Mon 9:00`,
   `Publishing…` (with a subtle indeterminate indicator and "usually 5–10 minutes"), `Live`,
   `Publish failed` (destructive token + link to the run).
3. **Site status strip** at the top of the workspace: "Last site build succeeded 12 minutes ago" or
   "Site build failed — content is saved but lextures.com is behind" with a **Retry build** action.
4. Empty/loading/error states: builds list empty ("No site builds yet"), loading skeleton rows,
   error retry.
5. Mobile: status pills wrap; the strip collapses to a single line with a details disclosure.
6. Accessibility: status polls update an `aria-live="polite"` region at most once per 30 s to avoid
   chatter; "Publishing…" has a text label, not just a spinner; retry is a real button.
7. Copy/i18n: `marketingContent.publish.*` (`status.publishing`, `status.live`, `status.failed`,
   `estimate`, `notConfigured`, `retry`, `scheduleFor`).

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **Internal modules:** `internal/service/marketingpublish` (new: debounce, dispatch, poll),
  `internal/scheduler/config.go` (two new builtin jobs), `internal/workers` (job handlers),
  `internal/service/marketingcontent` (transition hooks), `internal/objectcache` (invalidation),
  `internal/repos/platformconfig` (settings + encrypted token), `internal/telemetry`.
- **External services:** GitHub REST `POST /repos/{owner}/{repo}/dispatches` and
  `GET /repos/{owner}/{repo}/actions/runs` (poll). Rate limits: 5,000 req/h authenticated — polling
  every 30 s for at most 30 min per build is well inside budget.
- **CI:** `.github/workflows/pages-www.yml` gains `repository_dispatch` + `workflow_dispatch`
  triggers and passes `WWW_CONTENT_SOURCE=api`.
- **Events:** consumes `marketingcontent.published/unpublished/updated` from MC.2.

## 13. Dependencies & Sequencing

- Must ship after: MC.2 (transitions emit events), MC.7 (a DB-sourced build exists to trigger).
- Must ship before: MC.15 (cutover needs a working publish loop), MC.11 (editorial calendar shows
  scheduled items).
- Shared infra: the existing scheduler + job queue; outbound HTTPS to api.github.com; encrypted
  settings storage.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| GitHub token leak | L | **H** | Fine-grained, single-repo, dispatch-only scope; encrypted at rest; write-only in the UI; rotation runbook; never logged |
| Build storms from rapid editing | M | M | Debounce + single-pending-build unique index (AC-2); manual rebuild rate-limited |
| Publish→live latency confuses authors | **H** | M | Explicit status pill with an estimate and a site-status strip (UI §10); documented expectation in training |
| Workflow trigger drift (renamed workflow/branch) | M | M | Configurable workflow ref; a self-test button in settings dispatches a no-op build and reports the result |
| Scheduled publish fires while validation fails | M | M | AC-5: stay scheduled, record the error, notify the author (MC.11 notification) |
| Polling misses the run (rate limit or transient error) | M | L | Terminal timeout at 30 min marks `timed_out`; status is advisory, the site is unaffected |
| GitHub Actions outage | L | M | Manual rebuild button; documented fallback of pushing an empty commit |

## 15. Rollout Plan

- **Feature flag:** `ff_marketing_content` plus the `marketing_build_provider` setting
  (`none` by default → publishing works, rebuild does not fire).
- **Sequencing:** events + tables → dispatcher with provider `none` (records builds, dispatches
  nothing) → workflow triggers → staging repo dispatch → production configuration at MC.15.
- **Dogfood:** staging publishes drive real staging deploys for a week; measure publish→live latency
  and record it in the runbook.
- **GA criteria:** all ACs; p95 publish→live < 15 min measured over ≥ 20 staging publishes; alerting
  verified by a deliberately failed build.
- **Rollback:** set `marketing_build_provider=none` — publishing continues, rebuilds stop, and the
  site can be rebuilt by a normal code push.

## 16. Test Plan

- **Unit** — debounce state machine (quiet period, max wait, urgent bypass); single-pending
  coalescing; status transitions; dispatch payload construction; token redaction in logs; estimate
  computation.
- **Integration (DB)** — scheduled publish claim under concurrency (`SKIP LOCKED`, two workers, no
  double publish); cache invalidation after transition; build row lifecycle with a stubbed provider;
  timeout path.
- **End-to-end** — Playwright against staging: publish an article → status becomes `Publishing…` →
  (stubbed or real) build completes → status `Live`; schedule an article one minute out and observe
  it publish.
- **Security** — token never present in API responses, logs or the dispatch payload; `POST /builds`
  authz + rate limit; verify the GitHub token's scope in a manual pre-production check.
- **Accessibility** — axe on the status strip and publish dialog; verify polite announcements are not
  excessive; keyboard path through publish/schedule/retry.
- **Performance / load** — 50 publishes in 5 minutes produce ≤ 2 builds; dispatcher CPU negligible;
  polling budget check against GitHub rate limits.
- **Manual exploratory** — kill the API mid-build; fail the workflow deliberately; unconfigure the
  provider; publish with an expired token and confirm the error surfaces clearly.

## 17. Documentation & Training

- Runbook: "Marketing content publishing" — how dispatch is configured, how to rotate the token, how
  to force a rebuild, what to do when a build fails or the site shows fallback content.
- Admin docs: settings fields and their effects.
- Content-team training: what "Publishing…" means, expected latency, and that unpublish is fast but
  not instant.

## 18. Open Questions

1. Poll vs webhook for run completion — a GitHub App webhook would be push-based but adds an inbound
   endpoint and secret. (Proposed: poll for v1; revisit if latency reporting matters more.)
2. Should an urgent unpublish also write a temporary `_redirects`/`noindex` entry so the page stops
   being served before the build completes? (Worth doing if legal-removal scenarios are real —
   decide with the content team.)
3. Do we want a "publish window" (e.g. no automatic builds during a code freeze)? (Proposed: yes,
   a simple settings toggle deferred to MC.11 governance.)
4. Should scheduled publishing respect an editorial timezone rather than UTC? (Proposed: store UTC,
   display in the author's timezone; confirm with the content team.)

## 19. References

- Files this work touches: `server/internal/service/marketingpublish/*`,
  `server/internal/scheduler/config.go`, `server/internal/workers/*`,
  `server/internal/httpserver/marketing_builds_http.go`, `server/migrations/481_*`,
  `.github/workflows/pages-www.yml`.
- Precedents: `internal/scheduler/config.go` builtin jobs, `www/scripts/submit-indexnow.mjs`,
  `internal/repos/platformconfig` encrypted settings.
- External: GitHub REST API — repository dispatch events, Actions runs.
- Related plans: [MC.2](MC.2-authoring-api-and-revisions.md),
  [MC.7](MC.7-www-build-time-content-integration.md), [MC.9](../../plan/marketing-content/MC.9-marketing-content-workspace-shell.md),
  [MC.11](../../plan/marketing-content/MC.11-editorial-workflow-and-governance.md).
