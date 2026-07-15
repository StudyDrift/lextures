# AI Multi-Provider Support (BYOK)

> Epic plan. Extends the thin [16.7 AI provider abstraction](../../completed/16-integrations-extensibility/16.7-ai-provider-abstraction.md) into full bring-your-own-key multi-provider support. **OpenRouter is a provider**, not the only path.

## Problem (today)

Lextures still behaves as **OpenRouter-first** for almost all production AI traffic:

| Area | Current state |
|---|---|
| Platform credentials | Single `openrouter_api_key` on `settings.platform_app_settings`; Intelligence UI is OpenRouter-only |
| Per-user feature models | `user.user_ai_settings` stores raw OpenRouter model IDs (`arcee-ai/…`, `black-forest-labs/…`) |
| Provider abstraction | `server/internal/service/aiprovider/` implements OpenRouter, Anthropic, OpenAI, Azure OpenAI, Bedrock, Vertex + dry-run — but flag **defaults off** |
| Tenant BYOK | `settings.tenant_ai_settings` + encrypted `tenant_ai_secrets` exist (plan 16.7) — one key per org, not multi-credential |
| Call sites | ~15+ HTTP/service paths call `openrouter.Client` directly; only notebook RAG uses the resolver path meaningfully |
| Streaming / vision | `ChatCompletionStream` + multimodal/vision only on OpenRouter client; not on `aiprovider.Provider` |
| Model catalog | `GET /api/v1/settings/ai/models` lists **only** OpenRouter |
| Feature gates | `OpenRouterConfigured`, tutor/study-buddy enablement check `openRouterClient() != nil` |
| Disclosure / trust / copy | Hardcoded “via OpenRouter” in UI, mobile locales, marketing, AI disclosure |

Related completed work that this epic **must not regress**: 16.7, 10.17 (disclosure/gateway), grading agent, tutor, study buddy, notebook RAG, usage logging.

## Goals of this epic

1. Any supported provider (including OpenRouter) can be the primary backend for a platform or tenant.
2. Operators bring their own keys (platform global and/or org BYOK) with secrets never returned in APIs/logs.
3. All AI features route through one resolver/gateway path — no feature-level OpenRouter coupling.
4. Model selection uses stable aliases or provider-scoped catalogs, not OpenRouter-only IDs.
5. Usage, cost, disclosure, and subprocessor surfaces reflect the **actual** provider used.

## Story index

| ID | Story | Effort | Depends on |
|---|---|---|---|
| [AP.1](AP.1-provider-capability-interface.md) | Complete provider interface (stream, vision, options, image, embed) | M | — |
| [AP.2](AP.2-credential-store-and-byok.md) | Multi-provider credential store (platform + tenant BYOK) | M | AP.1 (types only) |
| [AP.3](AP.3-model-registry-and-catalog.md) | Model registry, aliases, and per-provider catalogs | M | AP.1, AP.2 |
| [AP.4](AP.4-migrate-call-sites.md) | Migrate all AI call sites onto the resolver | L | AP.1–AP.3 |
| [AP.5](AP.5-admin-intelligence-ui.md) | Unify Intelligence + org provider admin UX | M | AP.2, AP.3 |
| [AP.6](AP.6-usage-disclosure-observability.md) | Usage, cost, disclosure, feature flags | S | AP.4 |
| [AP.7](AP.7-clients-docs-trust.md) | Web/mobile/CLI copy, trust center, docs | S | AP.5, AP.6 |
| [AP.8](AP.8-provider-auth-hardening.md) | Azure / Bedrock IAM / Vertex ADC hardening | M | AP.2, AP.4 |
| [AP.9](AP.9-test-rollout-deprecation.md) | Test matrix, GA rollout, OpenRouter peer deprecation | S | AP.4–AP.7 |

## Suggested implementation order

```
AP.1 ──┬──► AP.2 ──► AP.3 ──► AP.4 ──┬──► AP.6 ──► AP.7
       │                             │
       └─────────────────────────────┼──► AP.5 ──┘
                                     │
                                     └──► AP.8 (can parallel after AP.2+AP.4)
AP.9 last (gates GA)
```

## Codebase inventory (change map)

### Backend packages (core)

| Path | Role today | Change needed |
|---|---|---|
| `server/internal/service/aiprovider/` | Interface + multi-backend impls | Expand interface; real stream/vision/image; cost estimation; auth extras |
| `server/internal/service/openrouter/` | Full OpenRouter client (chat, stream, vision, list models) | Become one backend; keep as OpenRouter provider implementation detail |
| `server/internal/service/aigateway/` | Policy gate (opt-out, COPPA, GDPR, tenant allow-lists) | Provider-agnostic already; default provider string + allow-list semantics |
| `server/internal/platformstate/` | Holds only `*openrouter.Client` | Hold resolver / multi-client registry; reload on credential change |
| `server/internal/repos/tenantaisettings/` | One provider + one BYOK key per org | Multi-credential support (or new table); extra settings validation |
| `server/internal/repos/platformconfig/` | `openrouter_api_key` only | Platform multi-provider credentials (encrypted) |
| `server/internal/repos/user/ai_settings.go` | OpenRouter model ID columns | Provider-scoped or alias-based feature models |
| `server/internal/repos/aiusage/` | Logs with `provider` column (default openrouter) | Always record real provider; cost without OpenRouter usage payload |
| `server/internal/aidisclosure/` | Names models “via OpenRouter” | Provider-aware disclosure document |
| `server/internal/config/config.go` | `OpenRouterAPIKey`, `AiProviderAbstractionEnabled` | Platform provider config; promote abstraction to default-on path |

### HTTP / features still on OpenRouter client

| File / area | Capability |
|---|---|
| `httpserver/tutor.go`, `tutor_sessions_http.go` | Streaming tutor |
| `httpserver/studybuddy_http.go` | Streaming study buddy |
| `httpserver/grading_agent_http.go`, `gradingagent/` | Grader agent + JSON mode + vision |
| `httpserver/me_notebook.go` | Flashcards still OpenRouter; RAG partially migrated |
| `httpserver/course_syllabus.go`, `structure_module_http.go` | Course setup / module generation |
| `httpserver/lesson_generator_http.go` | Lesson plans |
| `httpserver/translation.go`, `course_translation.go` | Translation |
| `httpserver/reading_level.go`, `contentsimplificationai/` | Reading-level simplify |
| `httpserver/alt_text_http.go`, `alttextai/` | Alt-text suggestion (vision) |
| `httpserver/report_cards_http.go` | Comment suggestions |
| `httpserver/settings_ai.go` | Platform AI models + OpenRouter key |
| `httpserver/platform_features.go` | `OpenRouterConfigured` gates |
| `background/periodic.go`, `coachingtips/` | Background coaching tips |
| `plagiarism/`, `originality_*` | Internal AI originality path |

### Clients & surfaces

| Path | Notes |
|---|---|
| `clients/web/src/pages/lms/settings.tsx` | Intelligence → Models: OpenRouter key + model pickers |
| `clients/web/src/components/settings/ai-provider-settings-panel.tsx` | Org provider BYOK UI (flag-gated) |
| `clients/web/src/components/image-model-picker*.ts(x)`, `lib/ai-models.ts` | OpenRouter catalogs / free-tier heuristics |
| `clients/web/src/components/ai-disclosure-banner.tsx`, `lib/ai-disclosure-i18n.ts` | “via OpenRouter” copy |
| `clients/web/src/content/trust/sub-processors.ts` | Static OpenRouter/Anthropic/OpenAI rows |
| `clients/cli/cmd/settings.go`, `platform_settings_logic.go` | AI provider CLI already partially present |
| `clients/mobile/locales/*.json` | Admin AI strings hardcode OpenRouter |
| `www/` pricing & product pages | Marketing assumes OpenRouter key |

### Tests / e2e

- `server/test/ai_provider_e2e_test.go`, `httpserver/ai_provider_settings_*`
- `e2e/tests/notebook-flashcards.spec.ts`, `ai-tutor.spec.ts`, `multilingual-messaging.spec.ts`, `grader-agent.spec.ts`, `trust-center.spec.ts`

## Non-goals (epic-wide)

- Training / fine-tuning models.
- Shipping a local Ollama runtime as GA (optional later story).
- Replacing the policy gateway (10.17) — only make it provider-agnostic.
- Per-student provider selection (org/platform scope only for v1).

## Success criteria (epic)

- A tenant can complete a full AI feature path (tutor message, quiz gen, grading agent) using Anthropic-direct or Azure OpenAI BYOK with **no** OpenRouter key configured.
- OpenRouter remains fully supported as one provider with its model catalog and optional platform key.
- No production call site imports `openrouter` except the OpenRouter provider adapter.
- Feature flag can be removed or flipped default-on after AP.9.

## Related plans

- [16.7 AI provider abstraction](../../completed/16-integrations-extensibility/16.7-ai-provider-abstraction.md) (thin base)
- [10.17 AI usage disclosure](../../completed/10-compliance-privacy-security/10.17-ai-usage-disclosure.md)
- [S07 subprocessor governance](../standards/S07-cross-border-transfer-subprocessor-governance.md)
- [S13 EU AI Act](../standards/S13-eu-ai-act-high-risk.md)
