# Runbook: AI provider / multi-provider rollback (AP.9)

Use when AI features fail or error rates spike after enabling multi-provider abstraction (default on at GA), or when a single upstream provider outage coincides with a deploy.

**RTO target:** restore prior AI behavior within **30 minutes**.

## Fast path — disable abstraction (one deploy)

1. Set process env:
   ```bash
   AI_PROVIDER_ABSTRACTION_ENABLED=0
   ```
2. Redeploy / restart API instances (same image is fine — flag only).
3. Confirm:
   - `GET /api/v1/platform/features` → `aiProviderAbstractionEnabled: false`
   - Multi-provider admin routes (`/api/v1/settings/ai/providers`, `/api/v1/admin/ai-settings`) return **404**
   - Legacy Intelligence OpenRouter key path still works when a platform OpenRouter key exists
4. Credentials in `settings.ai_provider_credentials` are **not** deleted; re-enable the flag later to restore multi-provider UI and org BYOK.

## Alternate — redeploy previous API version

If the regression is code (not config), follow [emergency-rollback.md](emergency-rollback.md) to the last known-good image tag. Do **not** run down-migrations unless the failed deploy applied a destructive schema change (AP.9 does not drop `openrouter_api_key` in the GA cut).

## Provider outage (keep abstraction on)

1. Open Grafana → **AI Provider** dashboard; filter by `provider`.
2. Watch `lextures_ai_provider_calls_total{outcome="error"}` and alert **AIProviderElevatedErrors**.
3. Temporarily clear or disable the failing provider credential; ensure another configured provider (or OpenRouter) can serve traffic.
4. Org BYOK overrides: check tenant settings if only one org is affected.

## Verification checklist

- [ ] Tutor / notebook / syllabus smoke (or synthetic probe) succeeds
- [ ] Disclosure page still lists only configured providers
- [ ] No plaintext keys in API responses or logs
- [ ] Incident note: whether rollback was flag-off vs image rollback

## Related

- [ai-providers-byok.md](../ai-providers-byok.md)
- [api-changelog-ai-providers.md](../api-changelog-ai-providers.md)
- [observability-oncall.md](observability-oncall.md#aiproviderelevatederrors)
- [emergency-rollback.md](emergency-rollback.md)
