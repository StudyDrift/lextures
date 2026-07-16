# AP.3 — Model Registry & Per-Provider Catalogs

> Implementation plan. Source: multi-provider BYOK epic ([README](../../plan/ai-providers/README.md)).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AP.3 |
| **Section** | AI Providers |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE — versioned alias registry (role + feature aliases), dual-read OpenRouter ids, per-provider curated/live catalogs, `GET /settings/ai/models?provider=&kind=`, gateway allow-list alias matching |
| **Estimated effort** | M (2w) |
| **Owner (proposed)** | AI / Platform |
| **Depends on** | AP.1, AP.2 |
| **Unblocks** | AP.4, AP.5 |

---

## 1. Problem Statement

Feature model defaults (`user.user_ai_settings`) and pickers assume OpenRouter model IDs (`arcee-ai/trinity-mini:free`, `black-forest-labs/flux.2-flex`). The alias registry maps only three models and invents poor cross-provider fallbacks (e.g. Gemini alias → Claude on Anthropic). Admins cannot list Anthropic/OpenAI/Azure models from the product UI. Multi-provider support requires a **stable registry**, **honest per-provider IDs**, and **catalog endpoints** that work when OpenRouter is not configured.

## 2. Goals

- Expand the model alias registry to cover all default feature models Lextures ships.
- Provide per-provider model listing (static curated + live list when the provider API supports it).
- Allow feature settings to store either a **stable alias** or a **provider-qualified model id**.
- Keep OpenRouter free-tier discovery for operators who use OpenRouter.

## 3. Non-Goals

- Training custom models or hosting weights.
- Guaranteeing live catalog parity with every cloud marketplace SKU.
- Per-student model choice beyond existing user preference rows.

## 4. Personas & User Stories

- **As a platform admin**, I want to pick “course setup” models from my Azure deployment list so that I am not forced into OpenRouter IDs.
- **As an instructor**, I want sensible defaults that still work when the org uses Anthropic-direct.
- **As a developer**, I want aliases like `grader-default` so that prompts do not hard-code vendor strings.

## 5. Functional Requirements

- **FR-1.** The system MUST maintain a versioned model registry mapping `alias → { provider → model_id }` for all providers in `ListProviders()`, including OpenRouter.
- **FR-2.** Aliases MUST cover at least: course setup, notebook flashcards, vibe activity, grader agent, image generation, translation, tutor, study buddy, syllabus, lesson plan, alt-text, simplification — either dedicated aliases or a small set of role aliases (`text-fast`, `text-strong`, `vision`, `image-gen`).
- **FR-3.** `GET /api/v1/settings/ai/models` MUST accept `provider` (default: active platform/org provider) and `kind=text|image|vision`; MUST NOT fail solely because OpenRouter is unconfigured when another provider is configured.
- **FR-4.** OpenRouter listing MUST continue to use `openrouter.ListModelsByOutputModality` when provider is OpenRouter.
- **FR-5.** For Anthropic/OpenAI/Azure/Bedrock/Vertex, the system MUST return a curated catalog and MAY enrich with live list APIs when credentials exist.
- **FR-6.** Unknown alias + raw model id: if the model string contains a provider-native id and the active provider matches, the system MUST pass it through; if alias is unknown, return a clear error.
- **FR-7.** User/feature model settings SHOULD migrate stored OpenRouter IDs to aliases where a mapping exists (backfill optional; dual-read required).
- **FR-8.** Tenant `allowedModels` in AI disclosure config MUST match against aliases and/or provider model ids consistently with gateway checks.

## 6. Non-Functional Requirements

- **Performance** — Curated catalog in-process; live lists cached ≥ 5 minutes.
- **Security** — Catalog calls use stored credentials server-side only.
- **Privacy** — No user content in catalog requests.
- **Accessibility** — N/A until AP.5 binds UI.
- **Scalability** — Cache per provider+kind.
- **Reliability** — Live catalog failure falls back to curated list (not 502 for settings page).
- **Observability** — `ai_model_catalog_fetch{provider,result}`.
- **Maintainability** — Registry as Go map + optional JSON seed under `server/internal/service/aiprovider/registry/`.
- **Internationalization** — Display names localizable later; ids stay English/vendor.
- **Backward compatibility** — Existing OpenRouter model id strings keep working when provider is OpenRouter.

## 7. Acceptance Criteria

- **AC-1.** *Given* platform provider Anthropic, *When* GET models `kind=text`, *Then* response is non-empty without OpenRouter key.
- **AC-2.** *Given* alias `text-fast` and provider OpenAI, *When* `ResolveModelID` runs, *Then* a valid OpenAI model id is returned.
- **AC-3.** *Given* stored user setting `arcee-ai/trinity-mini:free` and OpenRouter provider, *When* course setup runs, *Then* behavior matches pre-migration.
- **AC-4.** *Given* stored OpenRouter id while active provider is Anthropic, *When* resolve runs, *Then* system maps via alias table or returns actionable error (not a silent bad request to Anthropic).
- **AC-5.** *Given* OpenRouter provider, *When* list models, *Then* free-tier models still appear as today.

## 8. Data Model

- Optional: `settings.ai_model_registry_overrides` JSONB for platform-editable alias overrides (MAY defer; code registry sufficient for v1).
- `user.user_ai_settings` columns remain TEXT; semantics become “alias or provider model id”.
- Document defaults in `user/ai_settings.go` changing from pure OpenRouter ids to aliases once AP.4 lands.

No hard requirement for new tables if registry stays in code.

## 9. API Surface

Extend:

```
GET /api/v1/settings/ai/models?provider=openai&kind=text
→ { configured: bool, provider: string, models: [{ id, name, pricing?, modalities[] }] }

GET /api/v1/settings/ai
→ feature model fields + activeProvider + openRouterApiKey (deprecated alias of providers.openrouter)
```

- Admin AI settings (org) already returns `modelAliases`; expand to richer alias metadata (label, capabilities).

## 10. UI / UX

- AP.5 consumes this; ensure response shape supports:
  - Provider selector then model picker
  - Capability badges (text / vision / image)
  - Empty curated list messaging

## 11. AI / ML Considerations

- Alias quality: prefer models with similar capability, not forced wrong-family remaps.
- Image aliases only resolve for providers that support image generation.
- Cost estimates for non-OpenRouter catalogs may be null; AP.6 handles estimation.

## 12. Integration Points

- `aiprovider/models.go` — expand registry.
- `openrouter/list_models.go` — OpenRouter catalog only.
- `httpserver/settings_ai.go` — query params + multi-provider.
- `aidisclosure/disclosure.go` — stop assuming OpenRouter names for every model.
- `clients/web/src/lib/ai-models.ts`, `image-model-picker*.tsx` — later AP.5/AP.7.

## 13. Dependencies & Sequencing

- After AP.1/AP.2.
- Before AP.4 (call sites need resolve rules) and AP.5 (UI).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Stale curated IDs | H | M | Version registry; admin free-text model override |
| Alias maps to weaker model | M | M | Document alias intent; allow raw id override |
| Azure deployment names ≠ model names | H | H | Store deployment name in credential settings; registry maps alias → deployment |

## 15. Rollout Plan

- Ship registry expansion first (code-only).
- Dual-accept old OpenRouter ids.
- Flip default feature models to aliases in a later migration once AP.4 is green.

## 16. Test Plan

- **Unit** — Resolve every alias × every provider; unknown alias error; pass-through OpenRouter ids.
- **Integration** — Catalog endpoint with mock provider list servers.
- **E2E** — Settings page load without OpenRouter key when Anthropic configured (AP.5/AP.9).

## 17. Documentation & Training

- Developer: “Registering a new alias.”
- Admin: “Model aliases vs provider model IDs.”

## 18. Open Questions

1. Should model registry be admin-editable in UI for enterprise, or code-only for v1?
2. Per-feature provider override (tutor on Anthropic, images on OpenRouter) — v1 or later?
3. How do we name Azure “deployments” in the catalog UX?

## 19. References

- `server/internal/service/aiprovider/models.go`
- `server/internal/repos/user/ai_settings.go`
- `server/internal/httpserver/settings_ai.go`
- Related: [AP.5](AP.5-admin-intelligence-ui.md)
