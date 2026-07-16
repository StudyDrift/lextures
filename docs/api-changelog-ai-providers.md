# API changelog — multi-provider AI (AP.9)

**OpenAPI bootstrap version:** `0.2.0` (`server/internal/openapi/openapi.go`)

## Summary

Lextures AI is multi-provider GA. OpenRouter remains a fully supported **peer** provider. Operators configure credentials under Settings → Intelligence → Models (or CLI). New installs do **not** require an OpenRouter key; AI features stay disabled until any provider credential exists.

## Deprecated (dual-read ≥1 minor release)

| Surface | Deprecated | Prefer |
| --- | --- | --- |
| `GET/PUT /api/v1/settings/ai` | `openRouterApiKey`, `clearOpenRouterApiKey` | `GET/PUT/DELETE /api/v1/settings/ai/providers/{provider}` |
| `GET /api/v1/platform/features` | `openRouterConfigured` | `aiConfigured` + `aiProvidersConfigured[]` |
| DB column `settings.platform_app_settings.openrouter_api_key` | dual-read / dual-write with credential store | `settings.ai_provider_credentials` + secrets |

During the dual-read window:

- Writes to the legacy OpenRouter key field still update the credential store (and the legacy column).
- Reads prefer the encrypted credential store, then fall back to the legacy column / env key.
- Clients should migrate to `aiConfigured` and provider credential APIs before the next minor after GA soak.

## Planned removal (after soak)

After dual-read metrics show **zero** legacy column reads for ≥14 days (and ≥1 minor release has elapsed):

1. Ship a migration dropping `platform_app_settings.openrouter_api_key`.
2. Remove dual-read/write helpers and stop returning `openRouterApiKey` / `openRouterConfigured`.
3. Delete `AI_PROVIDER_ABSTRACTION_ENABLED` once rollback is no longer needed (flag already defaults **on**).

## Operator notes

- Default: `AI_PROVIDER_ABSTRACTION_ENABLED` is **true** when unset.
- Rollback: set `AI_PROVIDER_ABSTRACTION_ENABLED=0` and redeploy — see [ai-provider-rollback.md](runbooks/ai-provider-rollback.md).
- Admin guide: [ai-providers-byok.md](ai-providers-byok.md).
