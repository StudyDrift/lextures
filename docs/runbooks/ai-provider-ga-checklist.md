# Manual GA checklist — multi-provider AI (AP.9)

Complete before declaring hosted / self-host GA for multi-provider BYOK. Automated coverage: unit/integration in `server/internal/service/aiprovider`, HTTP e2e in `server/test/ai_provider_e2e_test.go`, Playwright `e2e/tests/ai-providers-settings.spec.ts`, CI OpenRouter coupling script.

## Staging soak

- [ ] Abstraction on (`AI_PROVIDER_ABSTRACTION_ENABLED` unset or `1`)
- [ ] Anthropic-only credentials — tutor, notebook, syllabus synthetic success ≥99%
- [ ] OpenRouter-only credentials — no user-visible regression vs pre-flip baseline
- [ ] Alert **AIProviderElevatedErrors** wired; burn rate watched for 24–72h

## Manual matrix

- [ ] Platform OpenRouter only — major AI features
- [ ] Platform Anthropic only — text features
- [ ] Org Azure BYOK override — generation + Test connection
- [ ] No credentials — AI features disabled cleanly (`aiConfigured: false`)
- [ ] Disclosure + reports show correct provider labels
- [ ] Mobile admin strings acceptable (provider / BYOK wording)
- [ ] CLI `settings ai-provider get|set|test`

## Deprecation gate (before column drop)

- [ ] Dual-read metrics: legacy `openrouter_api_key` reads == 0 for 14 days
- [ ] Clients no longer depend on `openRouterConfigured` / `openRouterApiKey`
- [ ] Changelog notice published ≥1 minor release prior ([api-changelog-ai-providers.md](../api-changelog-ai-providers.md))

## Rollback drill

- [ ] Set `AI_PROVIDER_ABSTRACTION_ENABLED=0`, redeploy, confirm AI recovers within 30 minutes — [ai-provider-rollback.md](ai-provider-rollback.md)
