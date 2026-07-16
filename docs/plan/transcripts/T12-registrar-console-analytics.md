# T12 — Registrar Console & Transcript Analytics

> Implementation plan. The registrar's operational home + destination/volume/revenue/SLA analytics. Source landscape: [transcripts/README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | T12 |
| **Section** | Transcripts & Credentials Platform |
| **Severity** | MAJOR |
| **Markets** | HE · K12 |
| **Status (today)** | THIN — the only admin surface is a webhook-URL/secret form plus a list of *failed* requests (`handleGetAdminTranscriptRequests`). There is no operational console, no configuration hub, and no analytics on transcript activity. |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Registrar/SIS squad (Web + Backend) |
| **Depends on** | T02, T03, T05, T06 (data to manage/report) |
| **Unblocks** | Registrar adoption; institutional reporting |

---

## 1. Problem Statement

A registrar running transcripts on Lextures needs one place to configure the service, work the
order queue, manage holds and fees, and understand activity (volumes, destinations, revenue, SLAs).
Today the admin surface is a bare webhook form plus a failed-requests list — nothing an actual
registrar's office could operate on. This story delivers the registrar console (pulling together
T03 fulfillment, T04 consent visibility, T05 fees, T06 delivery config) and a transcript analytics
dashboard.

## 2. Goals

- A **registrar console** unifying: order queue (T03), holds, fee config (T05), delivery/adapter config (T06), recipient directory (T02), and consent visibility (T04).
- A **transcript analytics** dashboard: volumes, top destinations, delivery method mix, turnaround/SLA, revenue, hold/rejection rates.
- **Exportable reports** (CSV) for institutional reporting and finance reconciliation.
- **Role-scoped access** so registrar/bursar/admin see the right slices.
- **SLA monitoring** with queue-age and failure alerts.

## 3. Non-Goals

- The underlying workflow, fees, delivery, consent mechanics (owned by T03/T05/T06/T04) — this surfaces and configures them.
- Platform-wide analytics/BI (this is transcript-scoped; integrates with [09 analytics](../../completed/09-analytics-reporting/)).

## 4. Personas & User Stories

- **As a registrar**, I want one console to run the transcript operation so that I'm not jumping between screens.
- **As a registrar lead**, I want to see turnaround times and backlog so that I can staff appropriately.
- **As a bursar**, I want transcript revenue and hold-block counts so that I can reconcile and report.
- **As an admin**, I want to configure fees, delivery adapters, and the recipient directory in one place so that setup is coherent.
- **As leadership**, I want to see where our students send transcripts so that I understand outcomes.

## 5. Functional Requirements

- **FR-1.** The console MUST present the order **fulfillment queue** (T03) with filters (status, hold, urgency, age, recipient) and bulk actions.
- **FR-2.** The console MUST expose **configuration**: fee schedule + waivers (T05), delivery adapters/endpoints (T06), recipient directory (T02), consent text version (T04), auto-approval, letterhead/seal/signature assets (T01).
- **FR-3.** The analytics dashboard MUST report: order volume over time, delivery-method mix, top destinations, average/percentile **turnaround** (submit→delivered), on-hold/rejection/refund rates, and revenue.
- **FR-4.** Reports MUST be **exportable** as CSV, scoped by date range and org.
- **FR-5.** Access MUST be role-scoped (registrar vs. bursar vs. admin) via RBAC; users see only their org's data.
- **FR-6.** The console MUST show **SLA/queue health**: oldest pending order, backlog count, delivery failure rate, dead-letter count, with alert thresholds.
- **FR-7.** All metrics MUST be computed from authoritative order/item/event/delivery data (no double counting; refunds net revenue).
- **FR-8.** The dashboard MUST render accessibly and follow the [dataviz](../../) charting standards (color, legends, a11y).
- **FR-9.** Analytics queries MUST be performant on large histories (pre-aggregated views/materialization).
- **FR-10.** The console MUST link every summary figure to its underlying records (drill-down).

## 6. Non-Functional Requirements

- **Performance** — dashboard p95 < 1s using pre-aggregated views; queue list < 400ms.
- **Security** — RBAC per role; org isolation; export authorization; no cross-tenant leakage.
- **Privacy & Compliance** — analytics aggregate/anonymized where possible; drill-down access-logged; FERPA-aware.
- **Accessibility** — console + charts WCAG 2.1 AA; charts have data-table equivalents (dataviz skill).
- **Scalability** — materialized aggregates refreshed on a schedule; queries indexed.
- **Reliability** — figures reconcile with source; refresh failures visible, not silently stale.
- **Observability** — reuse platform metrics ([17.7](../17-platform-performance-operability/)); dashboard load metrics.
- **Maintainability** — analytics as SQL views/materialized views; one console shell composing feature panels.
- **Internationalization** — labels, dates, currency localized.
- **Backward compatibility** — replaces the thin admin form; existing `ff_transcripts` config migrates in.

## 7. Acceptance Criteria

- **AC-1.** *Given* orders across states, *When* the registrar opens the queue, *Then* filters and bulk actions work and figures match the underlying records.
- **AC-2.** *Given* a date range, *When* the dashboard loads, *Then* volume, method mix, top destinations, turnaround percentiles, and net revenue render and reconcile with source data.
- **AC-3.** *Given* a bursar role, *When* they open the console, *Then* they see revenue/holds but not registrar-only config (RBAC).
- **AC-4.** *Given* a report export, *When* generated, *Then* the CSV matches the on-screen figures for the same range/org.
- **AC-5.** *Given* a growing backlog, *When* it crosses the SLA threshold, *Then* the health panel flags it (and alerts fire).
- **AC-6.** *Given* a summary figure, *When* clicked, *Then* it drills down to the contributing orders.

## 8. Data Model

Migration `389_transcript_analytics_views.sql` (indicative) — read-model views/materialized views
over T02/T03/T05/T06 tables (no new source-of-truth tables):

```sql
-- Daily order/delivery/revenue rollup per org (materialized; refreshed on schedule).
CREATE MATERIALIZED VIEW transcripts.mv_daily_stats AS
SELECT o.org_id,
       date_trunc('day', o.created_at)          AS day,
       count(DISTINCT o.id)                      AS orders,
       count(oi.id)                              AS items,
       count(oi.id) FILTER (WHERE oi.status='delivered') AS delivered,
       count(*) FILTER (WHERE o.status='on_hold')        AS on_hold,
       count(*) FILTER (WHERE o.status='rejected')       AS rejected,
       coalesce(sum(o.total_amount),0)
         - coalesce(sum(o.amount_refunded),0)     AS net_revenue_minor
FROM transcripts.orders o
LEFT JOIN transcripts.order_items oi ON oi.order_id = o.id
GROUP BY o.org_id, day;
CREATE UNIQUE INDEX ux_mv_daily_stats ON transcripts.mv_daily_stats (org_id, day);

-- Turnaround (submit → delivered) sourced from order_events (T03) + delivery_attempts (T06).
CREATE VIEW transcripts.v_turnaround AS
SELECT oi.order_id, oi.id AS item_id,
       o.submitted_at,
       min(da.created_at) FILTER (WHERE da.status='delivered') AS delivered_at
FROM transcripts.order_items oi
JOIN transcripts.orders o ON o.id = oi.order_id
LEFT JOIN transcripts.delivery_attempts da ON da.order_item_id = oi.id
GROUP BY oi.order_id, oi.id, o.submitted_at;
```

## 9. API Surface

- `GET /api/v1/admin/transcripts/dashboard?from=&to=` — aggregated metrics (RBAC, org-scoped).
- `GET /api/v1/admin/transcripts/reports/export?type=&from=&to=` — CSV export.
- `GET /api/v1/admin/transcripts/health` — SLA/queue-health panel data.
- Reuses admin endpoints from T02 (recipients), T03 (queue/holds/transitions), T05 (fees/waivers/refunds), T06 (delivery config).
- OpenAPI updated; RBAC scopes documented.

## 10. UI / UX

- **Registrar console shell** (new admin area) with tabs: **Queue** (T03), **Holds**, **Fees** (T05), **Delivery** (T06), **Recipients** (T02), **Settings** (consent text, letterhead/seal, auto-approval), **Analytics**.
- **Analytics dashboard**: KPI tiles (orders, delivered, avg turnaround, net revenue), time-series volume chart, method-mix + top-destinations charts, hold/rejection/refund rates, SLA health panel; date-range picker; drill-down; CSV export. Follow the **dataviz** skill for all charts (color, legend, a11y, data-table equivalents).
- States: empty (no orders yet), loading, stale-data warning, permission-scoped panels, export generating.
- Accessibility: charts have accessible tables; console keyboard/SR navigable; WCAG 2.1 AA.
- i18n + currency/locale formatting.

## 11. AI / ML Considerations

Optional: natural-language summary of the dashboard ("orders up 20% MoM; top destination …") using existing AI provider layer (AP.*), advisory only, cost-capped, no PII beyond aggregates. Off by default.

## 12. Integration Points

- **Internal:** T02/T03/T04/T05/T06 admin surfaces and data, RBAC, platform metrics ([17.7](../17-platform-performance-operability/)), [09 analytics](../../completed/09-analytics-reporting/), dataviz standards, existing platform settings panels (`settings/platform-*`).
- **External:** none required.
- **Emissions:** none (read-model); consumes existing events.

## 13. Dependencies & Sequencing

- After: T02, T03, T05, T06 (needs their data + admin surfaces). Ships last in Phase 4.
- Shared infra: materialized-view refresh job, RBAC, charting.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Metrics don't reconcile with finance | M | H | Single source (orders/items), net-of-refund revenue, reconciliation tests vs. Stripe/T05 |
| Stale materialized views mislead | M | M | Scheduled refresh + last-refreshed indicator + stale warning |
| Slow analytics on large history | M | M | Pre-aggregation, indexes, date-range bounds |
| RBAC leakage across roles/tenants | L | H | Org-scoped queries + role checks + tests for each panel |

## 15. Rollout Plan

- Flag `ff_transcripts`; console behind `transcripts.registrar_console` (shared with T03).
- Sequence: console shell + config panels → queue integration → analytics views + dashboard → exports → SLA/health + alerts.
- Pilot: one registrar office runs daily operations from the console for a term.
- Rollback: revert to the thin admin form; data/views retained.

## 16. Test Plan

- **Unit** — metric computations; net-revenue math; turnaround percentiles; RBAC scoping.
- **Integration** — dashboard reconciles with seeded orders; export matches on-screen; drill-down correctness.
- **E2E** — registrar runs queue actions + reads dashboard + exports CSV.
- **Security** — RBAC per role/panel; org isolation; export authz.
- **Accessibility** — console + charts axe; chart data-table equivalents; keyboard/SR.
- **Performance** — dashboard p95 with large history; view refresh timing.

## 17. Documentation & Training

- Registrar/admin runbook: configuring the service, working the queue, reading analytics, exports.
- Finance: revenue reconciliation guide.
- Metric definitions (turnaround, net revenue, SLA) glossary.

## 18. Open Questions

1. Materialized-view refresh cadence and near-real-time expectations for the queue vs. analytics?
2. Which reports finance requires for reconciliation, and in what schema?
3. Alerting channel for SLA breaches (email/Slack/in-app) and thresholds per org.

## 19. References

- Existing: `handleGetAdminTranscriptRequests` / admin config in `server/internal/httpserver/transcripts_http.go`, platform settings panels (`clients/web/src/components/settings/platform-settings-panel.tsx`, `transcripts-settings-panel.tsx`), observability ([17.7](../17-platform-performance-operability/)), [09 analytics](../../completed/09-analytics-reporting/).
- Related plans: [T02](../../completed/transcripts/T02-recipient-directory-and-orders.md), [T03](../../completed/transcripts/T03-order-lifecycle-fulfillment-holds.md), [T05](T05-fees-payments-waivers.md), [T06](T06-electronic-delivery-standards.md); **dataviz** skill for charts.
