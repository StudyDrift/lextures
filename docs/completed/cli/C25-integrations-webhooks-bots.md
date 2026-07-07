# C25 — Integrations, cloud providers, webhooks & bots

> CLI parity plan. Source: `integrations_http.go` (`integrations`, `admin-console/integrations`), `registerCloudProviderRoutes` (`cloud-providers`, `admin/cloud-providers`), `registerWebhookRoutes` (`webhooks`), `bots_http.go` (`bots`, `me/bot-link`), `admin_tokens.go` / `api_tokens_http.go` (`admin/tokens`, `me/access-keys`). Baseline: `clients/cli/cmd/tokens.go`, `access_keys.go`, `webhooks.go`, `cloud_providers.go`, `integrations.go`, `bots.go`, `integrations_webhooks_bots_logic.go`, `integrations_webhooks_bots_test.go`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | C25 |
| **Section** | Integrations & interoperability |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Integrations / CLI |
| **Depends on** | C40 |
| **Unblocks** | C09, C26 |

---

## 1. Problem Statement

Third-party integrations, cloud storage providers, outbound webhooks, chat bots, and API tokens/access keys are UI-only. Platform teams cannot register webhooks for event-driven automation, configure cloud storage, manage bot integrations, or programmatically mint API tokens for CI — the connective tissue of an automatable platform.

## 2. Goals

- Register/list/rotate API tokens and personal access keys for CI/service accounts.
- Configure outbound webhooks (events → external endpoints) and test delivery.
- Manage cloud storage providers and third-party integrations.
- Register/link bots.

## 3. Non-Goals

- Receiving inbound webhooks (server-side; e.g. originality/proctoring callbacks).
- Building specific integration connectors.

## 4. Personas & User Stories

- **As a devops engineer**, I want `tokens create --name ci --scope ...` to mint a CI token.
- **As an integrator**, I want `webhooks create --event grade.posted --url ...` and `webhooks test`.
- **As an admin**, I want `cloud-providers set --file s3.json` to configure storage.
- **As a comms admin**, I want `bots register` / `bots link` for a Slack/Teams bot.

## 5. Functional Requirements

- **FR-1.** MUST add `tokens list|create|revoke` (`admin/tokens`) and `access-keys list|create|revoke` (`me/access-keys`).
- **FR-2.** MUST add `webhooks list|create|update|delete|test|deliveries` (`registerWebhookRoutes`) with event filters.
- **FR-3.** MUST add `cloud-providers list|get|set|test` (`registerCloudProviderRoutes`).
- **FR-4.** SHOULD add `integrations list|get|enable|disable` (`integrations_http.go`).
- **FR-5.** SHOULD add `bots list|register|link|unlink` (`bots_http.go`, `me/bot-link`).

## 6. Non-Functional Requirements

- **Performance** — webhook `deliveries` paginated.
- **Security** — integration-admin scope; token/key secrets shown once then redacted; webhook signing secret managed server-side; provider creds via file/stdin.
- **Privacy & Compliance** — webhook payloads may carry PII; command warns which events include student data.
- **Reliability** — token/webhook create idempotent by name/id; `webhooks test` sends a signed sample.
- **Observability** — `deliveries` shows attempt/status/response for debugging.
- **Backward compatibility** — additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a scope, *When* `tokens create`, *Then* a one-time token prints; re-list redacts it.
- **AC-2.** *Given* a webhook, *When* `webhooks test`, *Then* a signed sample is delivered and status prints.
- **AC-3.** *Given* a cloud provider config, *When* `cloud-providers test`, *Then* connectivity validates without exposing creds.

## 8. Data Model

- None client-side. Document provider/webhook config JSON.

## 9. API Surface

- `admin/tokens` + `me/access-keys`; `registerWebhookRoutes`; `registerCloudProviderRoutes`; `integrations_http.go`; `bots_http.go` + `me/bot-link`.

## 10. UI / UX

- `lextures tokens ...`, `lextures access-keys ...`, `lextures webhooks ...`, `lextures cloud-providers ...`, `lextures integrations ...`, `lextures bots ...`.

## 11. AI / ML Considerations

- Bots may be AI-backed; CLI only registers/links, no model calls. AI provider config lives in C21.

## 12. Integration Points

- Server integration/webhook/cloud/bot/token handlers; feeds C09 (AI), C26 (events).

## 13. Dependencies & Sequencing

- After: C40.
- Before: C26 (event streaming), C09 (bot/AI adjacency).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Token/secret leakage | M | H | One-time display + redaction + `--secret-out` |
| Webhook to malicious URL | L | M | Server SSRF protections; CLI just configures |

## 15. Rollout Plan

- Ship tokens/access-keys + webhooks first (highest automation value), then cloud/integrations/bots.
- Rollback: additive.

## 16. Test Plan

- **Unit** — secret one-time display; event-filter parsing.
- **Integration** — webhook create/test/deliveries; token revoke.
- **Security** — secrets never re-shown.
- **E2E** — create token → use it for a subsequent CLI call.

## 17. Documentation & Training

- "Mint a CI token" and "Wire an outbound webhook" recipes.

## 18. Open Questions

1. What is the webhook event catalog?
2. Are access-keys (`me/access-keys`) equivalent to admin tokens or personal-scope only?

## 19. References

- `integrations_http.go`, `registerWebhookRoutes`, `registerCloudProviderRoutes`, `bots_http.go`, `admin_tokens.go`.
- Related: [C09](C09-ai-grading-agents.md), [C21](C21-platform-settings.md), [C26](C26-xapi-lrs-engagement.md).
