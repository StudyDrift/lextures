# AP.2 — Multi-Provider Credential Store & BYOK

> Implementation plan. Source: multi-provider BYOK epic ([README](../../plan/ai-providers/README.md)).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AP.2 |
| **Section** | AI Providers |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE — platform + org multi-provider credentials (`settings.ai_provider_credentials` / `ai_provider_secrets`), dual-read legacy OpenRouter + BYOK, masked APIs, audit without secrets |
| **Estimated effort** | M (2–3w) |
| **Owner (proposed)** | Platform / Security |
| **Depends on** | AP.1 (provider names/settings shape); 17.17 secrets primitives (`appsecrets`) |
| **Unblocks** | AP.3, AP.4, AP.5, AP.8 |

---

## 1. Problem Statement

Operators configure AI today via a single platform `openrouter_api_key` (Settings → Intelligence → Models) and optionally a single org BYOK key when `AI_PROVIDER_ABSTRACTION_ENABLED` is on. Institutions need **multiple providers**, each with its own key and non-secret settings (Azure base URL/deployment, AWS region, GCP project/location), at **platform default** and **tenant override** scopes. Without a proper credential store, multi-provider is a toggle that still funnels most traffic through one OpenRouter secret.

## 2. Goals

- Support platform-scoped and org-scoped credentials for every provider in `aiprovider.ListProviders()` (OpenRouter included).
- Encrypt all secrets at rest; never return plaintext via API (mask/placeholder only).
- Let tenants override platform defaults (BYOK) or inherit platform keys when allowed.
- Reload runtime clients when credentials change (extend `platformstate`).

## 3. Non-Goals

- Per-user API keys (v1 is platform + org only).
- External KMS/HSM integration beyond existing `PlatformSecretsKey` AES-GCM (can evolve later).
- UI work beyond API shapes (AP.5).
- OAuth device-code flows for providers (static keys / AP.8 for IAM/ADC).

## 4. Personas & User Stories

- **As a self-hosted operator**, I want to paste my OpenAI and Anthropic keys at the platform level so that all orgs inherit them without OpenRouter.
- **As a university IT admin**, I want org-level Azure OpenAI credentials (key + endpoint + deployment) so that student data stays on our Azure agreement.
- **As a district admin**, I want to clear our BYOK and fall back to the platform default so that we can centralize billing.
- **As a security officer**, I want keys write-only with audit events so that keys never appear in logs or API GETs.

## 5. Functional Requirements

- **FR-1.** The system MUST store zero or more **provider credentials** per scope (`platform` | `org`), keyed by `provider` name (`openrouter`, `anthropic`, `openai`, `azure_openai`, `bedrock`, `vertex`, …).
- **FR-2.** Each credential MUST support: encrypted secret material, non-secret `settings` JSON (e.g. `azure_base_url`, `aws_region`, `gcp_project`, `gcp_location`, `vertex_base_url`, `bedrock_base_url`), `enabled` flag, and timestamps/actor.
- **FR-3.** Platform OpenRouter key currently in `platform_app_settings.openrouter_api_key` MUST migrate into the new store (or dual-write during transition) without downtime.
- **FR-4.** Tenant BYOK (`tenant_ai_secrets`) MUST be generalized to **per-provider** secrets (not a single `byok_api_key` only).
- **FR-5.** Resolution order MUST be: (1) org credential for selected provider if present and enabled, else (2) platform credential for that provider, else (3) error “AI not configured”.
- **FR-6.** APIs that accept secrets MUST treat placeholder/masked values as “unchanged”; support explicit clear.
- **FR-7.** Changing credentials MUST invalidate resolver caches and rebuild platform runtime clients.
- **FR-8.** Audit log MUST record create/update/clear of credentials without secret values (`EventAIConfigChange` or successor).
- **FR-9.** Platform policy MAY allow/deny tenant BYOK and MAY restrict which providers tenants may select.

## 6. Non-Functional Requirements

- **Performance** — Credential decrypt + cache ≤ 5 ms p95 after warm; cache TTL ≤ 5 min (match current resolver).
- **Security** — AES-256-GCM via `appsecrets`; secrets never in metrics, traces, or JSON logs; DB access limited to service role.
- **Privacy & Compliance** — BYOK implies customer–provider DPA; document in admin help; FERPA-friendly (no prompt storage).
- **Accessibility** — N/A (API-only story).
- **Scalability** — Dozens of orgs × handful of providers; fine as row store.
- **Reliability** — Decrypt failure fails closed for that provider; other providers unaffected.
- **Observability** — Metric `ai_credentials_configured{scope,provider}`; alert if zero credentials and AI features enabled.
- **Maintainability** — Single repo package e.g. `repos/aiprovidercreds/` or extend `tenantaisettings` + `platformconfig`.
- **Internationalization** — Error strings via existing API error patterns.
- **Backward compatibility** — Read old `openrouter_api_key` and single BYOK until migration complete; dual-read window.

## 7. Acceptance Criteria

- **AC-1.** *Given* only an Anthropic platform key (no OpenRouter key), *When* resolver selects Anthropic, *Then* calls succeed and OpenRouter is never contacted.
- **AC-2.** *Given* org Azure credentials + settings, *When* GET admin settings, *Then* response shows `byokConfigured: true` and masked key, never plaintext.
- **AC-3.** *Given* existing deployments with `openrouter_api_key`, *When* migration runs, *Then* OpenRouter platform credential is populated and AI features keep working.
- **AC-4.** *Given* org clears BYOK, *When* next AI call runs, *Then* platform credential for that provider is used.
- **AC-5.** *Given* `PLATFORM_SECRETS_KEY` missing, *When* admin tries to store BYOK, *Then* API returns 503 with clear configuration guidance.
- **AC-6.** *Given* credential update, *When* audit log is read, *Then* event exists without secret fields.

## 8. Data Model

Proposed (names illustrative; follow repo migration numbering):

```sql
-- server/migrations/NNN_ai_provider_credentials.sql

CREATE TABLE IF NOT EXISTS settings.ai_provider_credentials (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope         TEXT NOT NULL CHECK (scope IN ('platform', 'org')),
    org_id        UUID REFERENCES tenant.organizations(id) ON DELETE CASCADE,
    provider      TEXT NOT NULL,
    enabled       BOOLEAN NOT NULL DEFAULT true,
    secret_ref    TEXT,                 -- key into secrets table
    settings      JSONB NOT NULL DEFAULT '{}',
    updated_by    UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (scope, org_id, provider),
    CHECK (
      (scope = 'platform' AND org_id IS NULL) OR
      (scope = 'org' AND org_id IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS settings.ai_provider_secrets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope       TEXT NOT NULL,
    org_id      UUID,
    provider    TEXT NOT NULL,
    secret_key  TEXT NOT NULL DEFAULT 'api_key',
    ciphertext  BYTEA NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (scope, org_id, provider, secret_key)
);
```

- **Backfill:** copy `platform_app_settings.openrouter_api_key` → platform/`openrouter`; copy `tenant_ai_secrets` → org/{provider from tenant_ai_settings}.
- Keep old columns until AP.9 deprecation.

Also extend/replace:

- `settings.tenant_ai_settings` — keep provider/model_alias/fallback; point `byok_secret_ref` at multi-provider store or drop in favor of credentials table.
- `config.Config.OpenRouterAPIKey` — derived view of platform OpenRouter secret during transition.

## 9. API Surface

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/api/v1/settings/ai/providers` | platform admin | List platform credentials (masked) |
| PUT | `/api/v1/settings/ai/providers/{provider}` | platform admin | Upsert platform credential + settings |
| DELETE | `/api/v1/settings/ai/providers/{provider}` | platform admin | Clear credential |
| GET | `/api/v1/admin/ai-settings` | org admin | Extend existing (16.7) with multi-credential summary |
| PUT | `/api/v1/admin/ai-settings` | org admin | Accept per-provider keys/settings |
| POST | `/api/v1/admin/ai-settings/test` | org admin | Keep; use resolved credentials |

Request shape (pseudo-TypeScript):

```ts
type ProviderCredentialUpsert = {
  enabled?: boolean
  apiKey?: string | null          // omit/placeholder = unchanged; null or clear flag = delete
  clearApiKey?: boolean
  settings?: {
    azure_base_url?: string
    azure_api_version?: string
    aws_region?: string
    gcp_project?: string
    gcp_location?: string
    // ...
  }
}
```

- Rate-limit test endpoint (e.g. 5/min/org).
- OpenAPI updates required.

## 10. UI / UX

- Deferred to [AP.5](AP.5-admin-intelligence-ui.md); this story ships API + migration only (CLI can dual-write for dogfood).

## 11. AI / ML Considerations

- No model calls except test endpoint (existing).
- Cost attribution remains per-provider once keys resolve.

## 12. Integration Points

- `server/internal/repos/platformconfig/` — dual-read OpenRouter key.
- `server/internal/repos/tenantaisettings/` — generalize or wrap.
- `server/internal/crypto/appsecrets/` — encrypt/decrypt.
- `server/internal/platformstate/` — reload multi-provider factory.
- `server/internal/httpserver/settings_ai.go`, `ai_provider_settings_http.go`.
- `server/internal/service/aiprovider/factory.go`, `resolver.go` — `apiKeyForProvider`.

## 13. Dependencies & Sequencing

- Must ship after: secrets key configuration documented; AP.1 provider name set stable.
- Must ship before: AP.3 (catalogs need which provider is configured), AP.4, AP.5.
- Shared infra: `PLATFORM_SECRETS_KEY` (32-byte).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Migration drops OpenRouter key | L | H | Dual-read old column; integration test; rollback migration keeps old column |
| Org overwrites platform with empty key | M | H | Placeholder semantics; require explicit clear |
| Multiple secrets for same provider confuse resolver | M | M | One primary `api_key` secret_key; extras only if provider requires (AP.8) |

## 15. Rollout Plan

- Feature flag: continue `AI_PROVIDER_ABSTRACTION_ENABLED` for multi-provider write paths; always dual-write OpenRouter into new store when set via Intelligence.
- Migration: schema → backfill → dual-read code → later drop (AP.9).
- Dogfood: internal tenants with Anthropic + OpenRouter.
- Rollback: dual-read falls back to `openrouter_api_key`.

## 16. Test Plan

- **Unit** — encrypt/decrypt, unique constraints, resolution order.
- **Integration** — migrate fixture DB with old OpenRouter key; PUT/GET mask; clear key.
- **E2E** — API-level in `server/test/ai_provider_e2e_test.go` extended.
- **Security** — assert response bodies and audit JSON never contain raw key patterns.
- **Manual** — rotate key mid-session; verify cache invalidation.

## 17. Documentation & Training

- Admin: “Platform vs organization AI credentials.”
- Runbook: key rotation, secrets key loss recovery.
- `.env.example`: document that OpenRouter env is no longer source of truth (already DB-only).

## 18. Open Questions

1. May tenants use a **different** provider than the platform default without platform listing that provider as allowed?
2. Should self-learner (no org) only use platform credentials?
3. Multi-key rotation grace period (5 min old key) — implement now or later?

## 19. References

- `server/migrations/317_tenant_ai_settings.sql`
- `server/migrations/118_platform_app_settings.sql`
- `server/internal/repos/tenantaisettings/repo.go`
- `server/internal/httpserver/settings_ai.go`
- Related: [AP.5](AP.5-admin-intelligence-ui.md), [AP.8](../../plan/ai-providers/AP.8-provider-auth-hardening.md)
