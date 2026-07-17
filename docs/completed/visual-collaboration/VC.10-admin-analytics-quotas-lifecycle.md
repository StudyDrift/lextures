# VC.10 — Admin Governance, Analytics, Quotas & Lifecycle

> Implementation plan. Source: net-new capability (real-time visual collaboration boards). Landscape: [visual-collaboration/README](../../plan/visual-collaboration/README.md). Integrates the platform flag layer (`server/internal/repos/platformconfig/features.go`), storage quotas (`server/internal/service/storagequota`), and the observability layer (`server/internal/telemetry`).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | VC.10 |
| **Section** | Visual Collaboration Boards |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Collaboration squad + Platform |
| **Depends on** | VC.1 |
| **Unblocks** | — |

---

## 1. Problem Statement

Once boards are live, the organisation needs the controls to run them responsibly: a **platform master
switch** and org policies, **analytics** on adoption and engagement, **storage quotas** so media doesn't
blow up costs, and **lifecycle** rules (retention, deletion, export) so student content is governed. VC.10
adds the admin, analytics, quota, and lifecycle layer that makes Boards enterprise- and compliance-ready.

## 2. Goals

- Ship the **platform master flag** (`VisualBoardsEnabled`) and org-level policies (external sharing on/off,
  minors moderation floor, default attribution).
- Provide **analytics**: per-board and per-course adoption, contribution counts, active participants, and an
  admin overview of usage.
- Enforce **storage quotas** for board attachments, integrated with the existing quota service.
- Wire **retention, deletion, and export** of board content into the shipped compliance engines (DSAR,
  retention, deletion).
- Ensure **observability** (metrics/traces/logs) and **i18n/accessibility** across the whole feature.

## 3. Non-Goals

- The board/post/engagement features themselves (VC.1–VC.9).
- A new analytics warehouse — reuse existing analytics/telemetry infrastructure.
- Billing/pricing changes for storage (surface usage; pricing owned by billing).
- Cross-course/org boards runtime (only the policy hooks and Open Questions here).

## 4. Personas & User Stories

- **As an admin**, I want a single switch to enable/disable Boards org-wide.
- **As an admin**, I want to forbid external sharing and force moderation for minors org-wide.
- **As an admin**, I want to see how many courses use Boards and how active they are.
- **As an instructor**, I want to see participation on my board (who contributed, how much).
- **As a DPO/admin**, I want a student's board contributions included in their data-subject export and
  deleted on erasure.
- **As an admin**, I want board attachments to count against storage quota with alerts before limits.

## 5. Functional Requirements

- **FR-1.** The system MUST add `VisualBoardsEnabled` (default `FALSE`) to `platformconfig` as a DB-managed
  flag, surfaced in the platform settings admin UI; when off, all board routes/nav are inert (per VC.1
  FR-2).
- **FR-2.** The system MUST add org policies: `boards_external_sharing` (default off — gates VC.6 link/public
  and VC.9 QR external links), `boards_minor_moderation_floor` (default on — forces VC.7 approval + blocking
  filter for age-gated courses), and `boards_default_attribution`.
- **FR-3.** The system MUST provide **per-board analytics**: total cards, unique contributors, contributions
  per participant, reactions/comments totals, and last-activity; visible to the board's managers.
- **FR-4.** The system MUST provide an **admin overview**: number of boards, active boards (activity in last
  N days), courses with Boards enabled, storage consumed by board attachments, and top content types.
- **FR-5.** Board attachment uploads MUST consult the **storage-quota** service and MUST be rejected (with a
  clear error) when the course/org quota is exceeded; usage MUST be attributed to the course.
- **FR-6.** A student's board **posts, comments, reactions, and attachments** MUST be included in their
  data-subject **export** and removed/anonymised on **erasure**, via the DSAR/deletion engines.
- **FR-7.** Boards MUST honour **retention schedules**: archived/abandoned boards and their attachments are
  purged per the retention engine; export files (VC.9) expire per policy.
- **FR-8.** The system MUST emit **metrics/traces/logs** for board operations (create, post, ws-connect,
  export, moderation) through the telemetry layer, with dashboards and alerts for error rates and abuse
  spikes.
- **FR-9.** All board UI strings MUST be in the i18n catalog and the whole feature MUST meet WCAG 2.1 AA
  (aggregate accessibility acceptance across VC.1–VC.9).
- **FR-10.** An optional per-course/org **board cap** MAY be configurable to bound proliferation
  (enforced at create).

## 6. Non-Functional Requirements

- **Performance** — analytics queries are precomputed/aggregated (not per-request scans); admin overview
  p95 < 500 ms.
- **Security** — analytics respect FERPA (managers see only their course; org admins see aggregates, not
  peer-visible student content); policy changes are audited.
- **Privacy & Compliance** — full coverage by DSAR ([S01](../../plan/standards/S01-unified-data-subject-rights-orchestration.md)),
  retention/deletion ([S02](../../plan/standards/S02-data-retention-deletion-engine.md)), and children's privacy
  ([S08](../../plan/standards/S08-childrens-privacy-age-assurance-design-codes.md)); board content registered in the
  data inventory/RoPA ([S05](../../plan/standards/S05-ropa-data-inventory-mapping.md)).
- **Accessibility** — admin/analytics dashboards are accessible; charts have table alternatives.
- **Scalability** — analytics aggregation batched; quota checks O(1) on the hot upload path.
- **Reliability** — deletion/erasure is verifiable (no orphaned attachments after erasure); idempotent
  retention jobs.
- **Observability** — this plan *is* the observability wiring; define the metric names, log fields, and
  alerts.
- **Maintainability** — reuse platformconfig, storagequota, telemetry, DSAR/retention engines; no parallel
  implementations.
- **Internationalization** — admin copy localised.
- **Backward compatibility** — additive flags/policies default to safe values.

## 7. Acceptance Criteria

- **AC-1.** *Given* `VisualBoardsEnabled = false`, *when* any board route or nav is accessed, *then* it is
  inert (404 / hidden) regardless of per-course flags.
- **AC-2.** *Given* `boards_external_sharing = false`, *when* an instructor opens sharing, *then* link/public
  and external QR options are unavailable (ties to VC.6/VC.9).
- **AC-3.** *Given* an age-gated course and the minor floor on, *when* a board is created, *then* approval
  mode + blocking filter are enforced (ties to VC.7).
- **AC-4.** *Given* a board with activity, *when* a manager opens analytics, *then* contributor counts and
  engagement totals are shown accurately.
- **AC-5.** *Given* a course over its storage quota, *when* a student uploads a board attachment, *then* the
  upload is rejected with a quota message.
- **AC-6.** *Given* a data-subject export request, *when* processed, *then* the user's board posts, comments,
  reactions, and attachments are included.
- **AC-7.** *Given* an erasure request, *when* processed, *then* the user's board content is removed/
  anonymised and no attachment objects remain orphaned.
- **AC-8.** *Given* the retention schedule, *when* a board passes its retention window, *then* it and its
  attachments/export files are purged.
- **AC-9.** *Given* board operations, *when* they run, *then* metrics/traces/logs appear in the telemetry
  dashboards with error/abuse alerts configured.

## 8. Data Model

Migration `398_board_admin_analytics.sql`:

```sql
-- Platform master flag lives on the platform settings table (DB-managed, default FALSE),
-- alongside the other flags in platformconfig.applyPlatformBools (VisualBoardsEnabled).
-- Org policies stored in the existing org/tenant settings JSON or a small policy table:
CREATE TABLE board.org_policies (
    org_id                 UUID PRIMARY KEY,
    external_sharing       BOOLEAN NOT NULL DEFAULT FALSE,
    minor_moderation_floor BOOLEAN NOT NULL DEFAULT TRUE,
    default_attribution    TEXT NOT NULL DEFAULT 'named',
    board_cap_per_course   INTEGER,                       -- null = unlimited
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Precomputed analytics rollups (refreshed by a job):
CREATE TABLE board.analytics_daily (
    board_id        UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    day             DATE NOT NULL,
    card_count      INTEGER NOT NULL DEFAULT 0,
    contributor_count INTEGER NOT NULL DEFAULT 0,
    reaction_count  INTEGER NOT NULL DEFAULT 0,
    comment_count   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (board_id, day)
);
```

- Storage attribution: board attachments reuse the existing storage-object accounting so
  `storagequota` sees them; no separate ledger.

## 9. API Surface

| Verb | Path | Auth |
|---|---|---|
| GET/PATCH | `/api/v1/admin/boards/policies` | org admin |
| GET | `/api/v1/admin/boards/overview` | org admin |
| GET | `/boards/{id}/analytics` | `item:create` |
| (existing) | DSAR export & erasure include board content | DSAR engine |
| (existing) | retention jobs purge boards/attachments/exports | retention engine |

- Platform master flag toggled via the existing platform settings admin surface
  (`settings_platform`/`platform-settings-panel`).
- **OpenAPI**: policies, overview, analytics endpoints.

## 10. UI / UX

- **Platform settings**: a "Collaboration boards" master toggle in the platform settings panel
  (`clients/web/src/components/settings/platform-settings-panel.tsx`), plus the org policy controls.
- **Admin overview**: a dashboard card/page with boards adoption, active boards, storage used, top content
  types (charts + table alternative).
- **Board analytics** (`components/boards/board-analytics.tsx`): contributor list with counts, engagement
  totals, activity sparkline — visible to managers.
- **Quota errors**: upload rejection surfaces a clear "storage limit reached" message with admin guidance.
- **States**: no-data analytics empty state; policy-locked controls (org floor) shown disabled with a
  tooltip.
- **Accessibility**: dashboards have data tables behind charts; all controls labelled.
- **Copy & i18n**: `boards.admin.*`, `boards.analytics.*` keys.

## 11. AI / ML Considerations

Optional/future (flagged off): AI "board insights" for instructors (theme clustering of cards, participation
summaries) using the AI provider path with cost budget, PII handling, and a manual fallback. Explicitly out
of scope for GA; listed here as the home for future AI-on-boards features.

## 12. Integration Points

- **Reuse**: `server/internal/repos/platformconfig/features.go` (master flag),
  `server/internal/service/storagequota` (quota enforcement),
  `server/internal/service/storageobjects`/`filestorage` (attachment accounting),
  `server/internal/telemetry` (metrics/traces — see the observability memory note),
  DSAR/retention/deletion engines from [`../../plan/standards/`](../../plan/standards/),
  `clients/web/src/components/settings/platform-settings-panel.tsx`, existing analytics dashboards.
- **New**: `server/internal/repos/board/policies.go`, `board/analytics.go`,
  `server/internal/httpserver/board_admin_http.go`, analytics rollup job (`server/internal/background/`),
  `clients/web/src/components/boards/board-analytics.tsx`.

## 13. Dependencies & Sequencing

- Must ship after: VC.1 (adds the master flag alongside foundation; policies referenced by VC.6/VC.7/VC.9).
- Ideally the **master flag + external-sharing + minor-floor** policy hooks land early (with or right after
  VC.1) because VC.6/VC.7/VC.9 read them; analytics/quotas/lifecycle can follow.
- Shared infra: platform settings, storage quota, telemetry, compliance engines, job queue.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Orphaned attachments after erasure | M | H | Erasure walks `board.post_attachments` → object store; reconciliation job verifies |
| Analytics scans hurt DB | M | M | Daily rollup job + indexed reads; no per-request full scans |
| Policy hooks not honoured by feature stories | M | H | Central policy resolver read by VC.6/VC.7/VC.9; integration tests assert enforcement |
| Quota check on hot upload path adds latency | L | M | O(1) cached quota read; fail-open only for transient quota-service errors with alert |

## 15. Rollout Plan

- **Flag**: `VisualBoardsEnabled` master (default off) + org policies (safe defaults). GA flips the master
  flag once VC.1–VC.7 are green.
- **Sequencing**: land master flag + policy hooks with VC.1 → add quota enforcement with VC.2 → add analytics
  + admin overview → confirm DSAR/retention coverage before GA.
- **Rollback**: master flag off disables the feature org-wide; compliance/retention jobs remain safe (inert
  when no boards).

## 16. Test Plan

- **Unit** — policy resolver; quota decision; analytics rollup math; DSAR/erasure collectors for board tables.
- **Integration** — master-flag inertness; external-sharing/minor-floor enforcement end-to-end (with VC.6/
  VC.7); quota rejection on upload; DSAR export includes board content; erasure removes it with no orphans;
  retention purge.
- **End-to-end** — Playwright: admin toggles master flag + policies; manager views analytics; over-quota
  upload blocked.
- **Security** — analytics FERPA scoping; policy-change audit; admin-only endpoints.
- **Accessibility** — dashboards axe + table alternatives.
- **Performance / load** — analytics rollup on a large tenant; quota hot-path latency.
- **Manual** — erasure verification (no orphaned objects); retention dry-run.

## 17. Documentation & Training

- Admin: enabling Boards; org policies (sharing, minors, attribution, caps); reading the overview.
- Instructor: board analytics.
- DPO/compliance: how board content is covered by DSAR/retention/erasure; RoPA entry.
- Runbook: master flag, quota alerts, retention/erasure jobs, telemetry dashboards.
- API reference: policies/overview/analytics endpoints.

## 18. Open Questions

1. Do we expose board analytics to students (their own contribution stats) or managers only? (Recommendation:
   managers only for v1; opt-in self-stats later.)
2. Should org-level boards (outside a course) exist, and how are they governed here? (Deferred from VC.1;
   decide before HE GA.)
3. Storage pricing/packaging for board media — bundle with course files quota or separate? (Owned by
   billing; surface usage now.)
4. Ship AI board insights in a later VC.11, or fold into the AI roadmap? (Recommendation: separate AI plan.)

## 19. References

- Existing files: `server/internal/repos/platformconfig/features.go`,
  `server/internal/service/storagequota/*`, `server/internal/service/storageobjects/*`,
  `server/internal/telemetry/*` (per the observability memory), `server/internal/httpserver/settings_platform.go`,
  `clients/web/src/components/settings/platform-settings-panel.tsx`.
- Related plans: [VC.1](VC.1-foundation-and-feature-flag.md), [VC.6](VC.6-sharing-access-contributors.md),
  [VC.7](VC.7-moderation-safety-governance.md), [VC.9](VC.9-embedding-export-presentation.md),
  and the compliance engines in [`../../plan/standards/`](../../plan/standards/)
  ([S01](../../plan/standards/S01-unified-data-subject-rights-orchestration.md),
  [S02](../../plan/standards/S02-data-retention-deletion-engine.md),
  [S05](../../plan/standards/S05-ropa-data-inventory-mapping.md),
  [S08](../../plan/standards/S08-childrens-privacy-age-assurance-design-codes.md)).
