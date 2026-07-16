# IQ.11 — Admin Governance, Quotas, Analytics & Lifecycle

> Implementation plan. Source: net-new capability (interactive game-based quizzing). Landscape: [interactive-quizzes/README](README.md). Mirrors the admin/analytics/quotas/lifecycle pattern from Collaboration Boards ([VC.10](../visual-collaboration/VC.10-admin-analytics-quotas-lifecycle.md)) and plugs into the admin console, platform settings, and retention engine already in the platform.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IQ.11 |
| **Section** | Interactive Quizzes |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Platform / Admin squad |
| **Depends on** | IQ.1 |
| **Unblocks** | (governs IQ.8 public catalog, IQ.3 concurrency) |

---

## 1. Problem Statement

Once Live Quizzes is live across an institution, admins need the controls that make it operable and safe at
scale: the **platform master switch** and defaults, **quotas** (how many concurrent live games, players per
game, AI generation budget), **analytics** (adoption, usage, cost), a **moderation/catalog review** queue, and
a **data lifecycle** (retention, anonymisation, DSAR/export) for the student data these games produce. IQ.11
delivers that admin surface so the feature is governable, not just usable.

## 2. Goals

- Expose the **platform master flag** and section defaults in platform settings (default game mode, guest-join
  policy, leaderboard privacy, retention window).
- Enforce **quotas/limits**: concurrent live games per tenant, players per game, kits per course, AI-generation
  budget (via `aiusage`), and rate limits — all configurable with safe defaults.
- Provide **admin analytics**: adoption (courses/instructors using it), usage (games hosted, players, answers),
  outcomes signal (participation), and **AI cost**, in the admin console.
- Provide a **moderation/catalog review** queue for public-catalog submissions (IQ.8) and reported content
  (IQ.9).
- Own the **data lifecycle**: retention/anonymisation of sessions/responses/guest data, DSAR export/deletion
  integration, and archival/cleanup jobs — honouring the shipped compliance engines.

## 3. Non-Goals

- The per-course toggle and gameplay (IQ.1–IQ.6) — IQ.11 governs, it doesn't play.
- The moderation *policy/enforcement* at game time (IQ.9) — IQ.11 provides the admin **review queue** and
  configuration surface on top.
- Rebuilding analytics/retention infra — IQ.11 plugs Live Quizzes into `adminconsole`, `reports`, and the
  retention/DSAR engines.

## 4. Personas & User Stories

- **As a platform admin**, I want to turn Live Quizzes on/off tenant-wide and set safe defaults, so rollout is
  controlled.
- **As an admin**, I want to cap concurrent live games and players per game, so we don't overload the realtime
  tier.
- **As an admin**, I want to see adoption and AI cost, so I can justify and manage the feature.
- **As an admin**, I want a queue to approve/reject kits submitted to the public catalog, so only vetted
  content is listed.
- **As a DPO/compliance officer**, I want game data to age out and be exportable/deletable on request, so we
  meet FERPA/GDPR obligations.

## 5. Functional Requirements

- **FR-1.** Platform settings MUST expose `FFInteractiveQuizzes` (master) plus section defaults: default mode,
  `allowGuests` policy, default `leaderboardPrivacy`, default retention window, and AI-generation enablement.
- **FR-2.** The system MUST enforce **quotas** with tenant-configurable limits and safe defaults: max
  concurrent live games/tenant, max players/game, max kits/course, max AI generations per period (budget via
  `aiusage`), and per-endpoint rate limits; exceeding a limit yields a clear error and an admin-visible signal.
- **FR-3.** The **admin console** MUST show Live-Quizzes analytics: number of games (by mode), unique
  hosts/players, answers submitted, average participation, guest vs enrolled split, and **AI cost** — filtered
  by org unit and time range, reusing `adminconsole`/`reports`.
- **FR-4.** A **moderation review queue** MUST list public-catalog submissions (IQ.8) and reported content
  (IQ.9) with approve/reject/takedown actions and audit; rejections notify the submitter with a reason.
- **FR-5.** The system MUST implement **retention/anonymisation**: after a configurable window, ended sessions'
  responses are anonymised or deleted (guest data sooner); a scheduled job performs this, honouring
  [S02](../standards/S02-data-retention-deletion-engine.md).
- **FR-6.** **DSAR** integration ([S01](../standards/S01-unified-data-subject-rights-orchestration.md)): a
  student's data export MUST include their game responses/scores/results, and a deletion request MUST remove/
  anonymise them across `quizgame.*` (guest rows purged, enrolled rows anonymised per policy).
- **FR-7.** Admins MUST be able to **force-end** or archive a runaway/abandoned game and to **bulk-archive**
  old kits.
- **FR-8.** All admin actions (flag/quota changes, approvals, takedowns, force-ends) MUST be **audited**
  (`adminaudit`).
- **FR-9.** The system MUST emit **operational metrics/alerts**: live-game and connected-player gauges,
  realtime-tier saturation, moderation-queue depth, retention-job success, and AI-budget breaches (via
  `telemetry`).
- **FR-10.** Quotas and defaults MUST be settable at **platform** and (where supported) **org-unit** scope,
  with org overriding platform within allowed bounds.

## 6. Non-Functional Requirements

- **Performance** — analytics queries backed by aggregates/materialized views; admin pages p95 < 500 ms;
  quota checks O(1) on the hot path (start-game/join/generate).
- **Security** — admin surfaces require platform/org admin roles; quota bypass impossible from clients;
  cross-tenant isolation on analytics.
- **Privacy & Compliance** — retention/DSAR/anonymisation honour FERPA/GDPR/COPPA and the standards engines;
  guest data minimised and aged fastest; analytics use aggregates, not raw PII, where possible.
- **Accessibility** — admin console additions meet AA (tables, charts with data-table equivalents).
- **Scalability** — analytics precomputed; retention job batched/chunked; quotas cheap to evaluate at scale.
- **Reliability** — retention/cleanup jobs idempotent and resumable; force-end finalises cleanly (reuses IQ.3
  finaliser); quota state consistent under concurrency.
- **Observability** — the section's operational dashboard lives here; alerts wired to the standard channels.
- **Maintainability** — quotas/defaults are config-driven (one settings source of truth), not scattered.
- **Internationalization** — admin copy localised; time ranges timezone-aware.
- **Backward compatibility** — additive; defaults chosen so existing behaviour is unchanged until admins tune.

## 7. Acceptance Criteria

- **AC-1.** *Given* the platform flag is off, *when* any course tries to use Live Quizzes, *then* it's
  unavailable tenant-wide (consistent with IQ.1 AC-7).
- **AC-2.** *Given* a tenant at its concurrent-games cap, *when* another host starts a game, *then* it's
  refused with a clear message and the event is visible to admins.
- **AC-3.** *Given* a game exceeding max players, *when* another player joins, *then* the join is refused per
  the configured cap.
- **AC-4.** *Given* the admin console, *when* an admin opens Live-Quizzes analytics, *then* adoption, usage,
  participation, guest/enrolled split, and AI cost render, filterable by org unit and time.
- **AC-5.** *Given* a catalog submission, *when* an admin rejects it, *then* it isn't listed and the submitter
  is notified with the reason; the action is audited.
- **AC-6.** *Given* the retention window passes, *when* the job runs, *then* old responses are anonymised/
  deleted and guest data is purged, verifiably.
- **AC-7.** *Given* a student DSAR export, *when* generated, *then* it includes their game responses/results;
  a deletion request removes/anonymises them across `quizgame.*`.
- **AC-8.** *Given* an abandoned live game, *when* an admin force-ends it, *then* it finalises and frees its
  concurrency slot; the action is audited.

## 8. Data Model

Migration `400_interactive_quizzes_governance.sql`:

```sql
-- platform + org defaults/quotas
ALTER TABLE settings.platform_app_settings
  ADD COLUMN IF NOT EXISTS iq_max_concurrent_games INTEGER,      -- NULL = unlimited
  ADD COLUMN IF NOT EXISTS iq_max_players_per_game INTEGER NOT NULL DEFAULT 300,
  ADD COLUMN IF NOT EXISTS iq_max_kits_per_course  INTEGER,
  ADD COLUMN IF NOT EXISTS iq_retention_days       INTEGER NOT NULL DEFAULT 365,
  ADD COLUMN IF NOT EXISTS iq_guest_join_policy     TEXT NOT NULL DEFAULT 'disabled', -- disabled|teacher_mediated|open
  ADD COLUMN IF NOT EXISTS iq_default_mode          TEXT NOT NULL DEFAULT 'live_classic',
  ADD COLUMN IF NOT EXISTS iq_default_leaderboard_privacy TEXT NOT NULL DEFAULT 'names',
  ADD COLUMN IF NOT EXISTS iq_ai_generation_enabled BOOLEAN NOT NULL DEFAULT FALSE;

-- org-unit overrides (bounded by platform)
CREATE TABLE quizgame.org_settings (
  org_unit_id   UUID PRIMARY KEY REFERENCES org.org_units (id) ON DELETE CASCADE,
  overrides     JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_by    UUID REFERENCES "user".users (id) ON DELETE SET NULL,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- moderation / catalog review queue (IQ.8/IQ.9 feed this)
CREATE TABLE quizgame.review_queue (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kind         TEXT NOT NULL,                 -- catalog_submission | reported_content
  kit_id       UUID REFERENCES quizgame.kits (id) ON DELETE CASCADE,
  session_id   UUID REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
  detail       JSONB NOT NULL DEFAULT '{}'::jsonb,
  status       TEXT NOT NULL DEFAULT 'pending', -- pending | approved | rejected | actioned
  reviewer_id  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
  reason       TEXT,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at  TIMESTAMPTZ
);
CREATE INDEX idx_quizgame_review_pending ON quizgame.review_queue (status) WHERE status = 'pending';

-- lightweight usage rollup for analytics (populated by a job)
CREATE TABLE quizgame.usage_daily (
  day          DATE NOT NULL,
  org_unit_id  UUID,
  course_id    UUID,
  games        INTEGER NOT NULL DEFAULT 0,
  players      INTEGER NOT NULL DEFAULT 0,
  answers      INTEGER NOT NULL DEFAULT 0,
  ai_cost_cents INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (day, org_unit_id, course_id)
);
```

- Quota checks read platform/org settings on the hot path (cached).
- Retention job uses `iq_retention_days`; guest rows aged faster per policy.

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| GET/PATCH | `/api/v1/admin/settings/interactive-quizzes` | platform admin |
| GET/PATCH | `/api/v1/admin/org-units/{id}/interactive-quizzes` | org admin (bounded) |
| GET | `/api/v1/admin/interactive-quizzes/analytics?orgUnit=&from=&to=` | admin |
| GET | `/api/v1/admin/interactive-quizzes/review-queue` | admin/moderator |
| POST | `/api/v1/admin/interactive-quizzes/review-queue/{id}/{approve\|reject}` | admin/moderator |
| POST | `/api/v1/admin/interactive-quizzes/games/{game_id}/force-end` | admin |
| POST | `/api/v1/admin/interactive-quizzes/kits/bulk-archive` | admin |

- DSAR/retention hooks are invoked by the existing S01/S02 orchestrators, not new public endpoints.
- **OpenAPI:** document admin settings, analytics, and review-queue schemas.

## 10. UI / UX

- **Platform settings panel:** a "Live Quizzes" section in `platform-settings-panel` (mirroring existing
  feature toggles) with the master flag, defaults, and quotas; feature definition added to
  `platform-feature-definitions.ts`.
- **Admin console page:** `Live Quizzes` analytics (adoption/usage/participation/AI cost with chart + data
  tables), a **review queue** with approve/reject, and a **live games** operational view (force-end).
- **Org-unit settings:** bounded overrides where org admin scope applies.
- **Flows:** enable feature → set defaults/quotas → monitor analytics → review catalog submissions → force-end
  a stuck game → confirm retention running.
- **States:** feature-off, no-data, quota-breach banner, empty review queue, retention-job status.
- **Accessibility:** admin tables/charts AA with data-table equivalents; keyboard-operable queue actions.
- **Copy & i18n:** `admin.liveQuiz.*` keys.

## 11. AI / ML Considerations

Not AI-touching itself, but IQ.11 **governs** IQ.10: the AI-generation budget, enablement, and cost analytics
live here (reading `aiusage`).

## 12. Integration Points

- **Reuse:** `adminconsole` + `platform-settings-panel` / `platform-feature-definitions.ts` (UI),
  `platformconfig` (flags/defaults), `reports` (analytics), `aiusage` (AI cost/budget), the retention/DSAR
  engines ([S01](../standards/S01-unified-data-subject-rights-orchestration.md)/[S02](../standards/S02-data-retention-deletion-engine.md)),
  `adminaudit` (audit), `telemetry` (metrics/alerts), the IQ.3 finaliser (force-end).
- **Server new:** `repos/quizgame/{governance,review,usage}.go`, admin handlers
  `httpserver/quizgame_admin.go`, a retention job + a usage-rollup job in `background/`.
- **Web new:** platform-settings section, admin analytics + review-queue + live-games pages.

## 13. Dependencies & Sequencing

- Must ship after: IQ.1 (flag/schema exist); quota enforcement hooks into IQ.3 (start/join) and IQ.10 (budget).
- Must ship before: opening the IQ.8 **public catalog** (needs the review queue) and any high-scale rollout
  (needs quotas + ops dashboard).
- Shared infra: admin console, platform settings, reports, retention/DSAR engines, job runner, telemetry.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| No quotas → realtime tier overload at scale | M | H | Concurrent-games + players caps enforced on the hot path; ops alerts on saturation |
| Retention job deletes too much / too little | M | H | Idempotent, dry-run mode, batched; policy from S02; audited; restore window before hard delete |
| DSAR misses `quizgame.*` data | M | H | Explicit S01 adapters for kits/sessions/responses; tested export/delete |
| Catalog review bottleneck | M | M | Clear queue UI, reason templates, SLA metrics; auto-flag via `contentfilter` pre-screen |
| Cross-tenant analytics leakage | L | H | Org scoping on every query; tenant isolation tests |
| Quota state races under concurrency | M | M | Atomic counters/advisory locks for concurrent-game slots; force-end frees slots |

## 15. Rollout Plan

- **Flag:** `FFInteractiveQuizzes` master + section defaults; quotas ship with safe defaults on.
- **Sequencing:** migration `400` → settings + quotas → analytics → review queue → retention/DSAR jobs.
- **Dogfood:** set quotas low and verify refusals; run analytics on dogfood data; approve/reject a catalog
  submission; run the retention job in dry-run then live.
- **GA criteria:** AC-1..AC-8 pass; retention + DSAR verified; ops dashboard/alerts live; quotas enforced.
- **Rollback:** master flag off disables the section; jobs/settings retained; no data loss.

## 16. Test Plan

- **Unit** — quota evaluation; org-override bounding; retention selection logic; usage-rollup math.
- **Integration** — concurrent-games cap refusal + slot free on force-end; players cap; review approve/reject +
  audit + notify; retention anonymise/delete; DSAR export/delete across `quizgame.*`.
- **End-to-end** — Playwright (admin): toggle flag, set quotas, view analytics, process review queue,
  force-end a game.
- **Security** — admin-role gating; cross-tenant isolation; client cannot bypass quotas.
- **Accessibility** — admin pages axe + data-table equivalents.
- **Performance** — analytics query latency on large datasets; retention job on a big backlog.
- **Compliance** — retention window correctness; guest-data purge; DSAR completeness.

## 17. Documentation & Training

- Admin: "Configure & govern Live Quizzes" — flags, defaults, quotas, guest policy, retention.
- Admin: reading analytics; the catalog review workflow; force-ending games.
- Compliance: retention/anonymisation policy; DSAR coverage for game data.
- Runbook: retention + usage-rollup jobs, quota tuning, ops alerts, catalog moderation SLA.

## 18. Open Questions

1. Default `iq_max_concurrent_games` — unlimited or a conservative cap until the multi-instance fan-out
   (IQ.3) lands? (Recommendation: a conservative per-instance cap until horizontal fan-out ships.)
2. Retention default — 365 days, or align per-course to the assessment retention policy? (Recommendation:
   365-day platform default, org/course-overridable within S02 bounds.)
3. Should org-unit admins tune quotas, or platform-only? (Recommendation: platform sets bounds; org-unit admins
   tune within them where the org model supports it.)

## 19. References

- Existing files: `server/internal/repos/adminconsole/`, `server/internal/httpserver/settings_platform.go`,
  `clients/web/src/components/settings/platform-settings-panel.tsx`,
  `clients/web/src/components/settings/platform-feature-definitions.ts`,
  `server/internal/repos/aiusage/`, `server/internal/repos/adminaudit/`, `server/internal/telemetry`.
- Related plans: [IQ.1](IQ.1-foundation-and-feature-flag.md), [IQ.3](../../completed/interactive-quizzes/IQ.3-live-game-hosting-engine.md),
  [IQ.8](IQ.8-library-templates-sharing.md), [IQ.9](IQ.9-moderation-safety-accessibility.md),
  [IQ.10](IQ.10-ai-assisted-generation.md), [VC.10](../visual-collaboration/VC.10-admin-analytics-quotas-lifecycle.md);
  [S01 DSAR](../standards/S01-unified-data-subject-rights-orchestration.md), [S02 retention](../standards/S02-data-retention-deletion-engine.md).
