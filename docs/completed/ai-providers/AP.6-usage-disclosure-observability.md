# AP.6 — Usage, Cost, Disclosure & Observability

> Implementation plan. Source: multi-provider BYOK epic ([README](../../plan/ai-providers/README.md)).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AP.6 |
| **Section** | AI Providers |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE — usage logs record real provider + optional cost estimates; disclosure lists configured providers without forcing OpenRouter; reports filter by provider; Prometheus/Grafana multi-provider labels |
| **Estimated effort** | S–M (1–2w) |
| **Owner (proposed)** | AI / Compliance / Observability |
| **Depends on** | AP.4 (call sites emit CallMeta) |
| **Unblocks** | AP.7, AP.9 |

---

## 1. Problem Statement

Intelligence reports, Prometheus metrics, and public AI disclosure still read as OpenRouter-centric. When tenants use direct providers, cost fields may be zero, disclosure wrongly attributes models to OpenRouter, and dashboards mislead finance and DPOs. Multi-provider support is incomplete without honest **usage**, **estimated cost**, **disclosure**, and **metrics**.

## 2. Goals

- Every successful/failed AI call records accurate `provider`, model id/alias, tokens, estimated USD.
- Public/in-app disclosure reflects the providers actually configured/used.
- Grafana/Prometheus dashboards remain correct with multi-provider labels.
- Gateway inference logs store the real provider string.

## 3. Non-Goals

- Full chargeback billing product (beyond estimates).
- Replacing 10.17 consent/opt-out flows.
- Subprocessor legal register automation (S07) — only product disclosure surfaces here.

## 4. Personas & User Stories

- **As a finance officer**, I want usage broken down by provider so that I can reconcile Anthropic vs OpenAI invoices.
- **As a DPO**, I want the AI disclosure page to list the real providers our instance calls.
- **As an SRE**, I want error rates per provider so that I can page on Azure outages only.

## 5. Functional Requirements

- **FR-1.** `analytics.ai_usage_log` inserts MUST set `provider` from `CallMeta.Provider` (never silently default when meta is present).
- **FR-2.** When a provider omits cost, the system SHOULD estimate USD from a maintainable price table keyed by provider+model (best-effort); store as estimate and optionally flag `cost_estimated=true` if column added.
- **FR-3.** Intelligence AI reports UI MUST group/filter by provider; copy MUST NOT say “OpenRouter spend” exclusively.
- **FR-4.** `aidisclosure.BuildPublicDisclosure` MUST list configured providers and model display names without forcing “(via OpenRouter)” when not routed through OpenRouter.
- **FR-5.** In-app disclosure banner MUST use active provider label.
- **FR-6.** Prometheus metrics (`lextures_ai_provider_*`, estimated cost) MUST use real provider labels for all paths.
- **FR-7.** `aigateway.LogInference` default provider MUST not assume OpenRouter when caller passes empty — prefer `"unknown"` or require caller (post-AP.4 callers always pass).
- **FR-8.** Export/report APIs used by admins MUST include provider dimension.

## 6. Non-Functional Requirements

- **Performance** — Price table lookup O(1); no extra provider HTTP for cost.
- **Security** — No prompt content in usage rows (existing).
- **Privacy & Compliance** — Disclosure accuracy for EU AI Act / FERPA transparency; align with S07 narrative.
- **Accessibility** — Reports tables remain accessible.
- **Scalability** — Indexes on `(provider, created_at)` if missing for report queries.
- **Reliability** — Usage insert best-effort must not fail the user request (existing pattern).
- **Observability** — This story *is* the observability hardening.
- **Maintainability** — Price table in `aiprovider/pricing.go` with update comments.
- **Internationalization** — Disclosure strings via i18n (AP.7 may finish locales).
- **Backward compatibility** — Old rows with provider=`openrouter` remain valid.

## 7. Acceptance Criteria

- **AC-1.** *Given* a completion via Anthropic, *When* usage log is written, *Then* `provider=anthropic` and tokens > 0 when API returns usage.
- **AC-2.** *Given* OpenAI without cost field, *When* estimate table has rates, *Then* `cost_usd` > 0 estimate.
- **AC-3.** *Given* instance with only Azure configured, *When* public disclosure is fetched, *Then* body does not claim OpenRouter as the router for those models.
- **AC-4.** *Given* metrics scrape after mixed providers, *When* querying `lextures_ai_provider_latency_seconds`, *Then* both provider labels appear.
- **AC-5.** *Given* Intelligence reports panel, *When* rendered, *Then* title/description are provider-agnostic.

## 8. Data Model

```sql
-- optional
ALTER TABLE analytics.ai_usage_log
  ADD COLUMN IF NOT EXISTS cost_estimated BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS model_alias TEXT;

CREATE INDEX IF NOT EXISTS idx_ai_usage_log_provider_created
  ON analytics.ai_usage_log (provider, created_at DESC);
```

Backfill not required for historical accuracy.

## 9. API Surface

- Extend reports endpoints (if present) with `?provider=` filter.
- Disclosure endpoints return `providers: string[]` and per-model `provider` fields (update consumers).
- OpenAPI updates.

## 10. UI / UX

- `clients/web/src/components/settings/ai-reports-panel.tsx` — provider filter + copy.
- `ai-disclosure-banner.tsx`, disclosure page — dynamic provider.
- Grafana dashboard JSON under `deploy/observability/` — ensure variables include all providers.

## 11. AI / ML Considerations

- Estimation table will drift; document update cadence.
- Do not invent costs for image models without rates (leave 0 + estimated false).

## 12. Integration Points

- `repos/aiusage/aiusage.go`
- `httpserver/ai_provider_usage.go`
- `aidisclosure/disclosure.go`
- `aigateway/service.go`
- `aiprovider/metrics.go`, `telemetry/metrics.go`
- `deploy/observability/README.md` / Grafana AI Provider dashboard
- Trust/legal copy coordination with AP.7

## 13. Dependencies & Sequencing

- After AP.4 (otherwise most rows still OpenRouter-shaped).
- Before AP.7 (copy depends on accurate disclosure API).
- Related: S07, 10.17.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Wrong cost estimates | H | M | Label as estimate; link to provider invoices |
| Disclosure overclaims providers not used | M | M | Prefer “configured” vs “used in last 30d” sections |
| Metric cardinality explosion | L | M | Bound model label to alias or low-cardinality id |

## 15. Rollout Plan

- Additive columns; deploy code; update dashboards.
- No feature flag required if AP.4 already emitting meta.
- Rollback: ignore new columns; old queries still work.

## 16. Test Plan

- **Unit** — cost estimator; disclosure assembly with multi-provider fixtures.
- **Integration** — usage insert after mock Complete.
- **E2E** — trust/disclosure pages (AP.7).
- **Observability** — metric registration tests exist; extend labels.

## 17. Documentation & Training

- Admin: “Understanding AI usage reports.”
- On-call: AI provider dashboard runbook updates.

## 18. Open Questions

1. Show configured providers, historically used, or both on disclosure?
2. Should estimated cost be hidden by default until rates reviewed by finance?

## 19. References

- `server/internal/repos/aiusage/aiusage.go`
- `server/internal/aidisclosure/disclosure.go`
- `deploy/observability/README.md`
- Related: [AP.4](AP.4-migrate-call-sites.md), [AP.7](AP.7-clients-docs-trust.md)
