# AP.4 — Migrate AI Call Sites to Provider Resolver

> Implementation plan. Source: multi-provider BYOK epic ([README](README.md)).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AP.4 |
| **Section** | AI Providers |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | THIN — most features call `openRouterClient()` / `*openrouter.Client` directly |
| **Estimated effort** | L (3–5w) |
| **Owner (proposed)** | AI / Product eng |
| **Depends on** | AP.1, AP.2, AP.3 |
| **Unblocks** | AP.6, AP.9 |

---

## 1. Problem Statement

Even with `aiprovider` and tenant settings in place, nearly every product AI path still depends on OpenRouter: tutors stream via `ChatCompletionStream`, grading agent embeds `*openrouter.Client`, platform features gate on `openRouterClient() != nil`, and background jobs construct OpenRouter clients from `cfg.OpenRouterAPIKey`. Multi-provider remains theoretical until **all call sites** go through the resolver/gateway and record real provider metadata.

## 2. Goals

- Eliminate production imports of `openrouter` outside the OpenRouter provider adapter (and tests).
- Route every user-facing and background AI invocation through `aiprovider.Resolver` + `aigateway` policy checks.
- Gate “AI available” on **any** configured provider for the effective scope, not OpenRouter alone.
- Preserve feature behavior (prompts, JSON mode, streaming UX) for OpenRouter tenants during migration.

## 3. Non-Goals

- New AI product features.
- Provider-specific prompt tuning / eval harness.
- Full IAM/ADC for Bedrock/Vertex (AP.8) — use API key/static configs already supported.

## 4. Personas & User Stories

- **As a student**, I want the tutor to stream answers whether the school uses OpenRouter or Azure.
- **As an instructor**, I want quiz generation and grading agent to work on our Anthropic key.
- **As an operator**, I want AI feature flags to show enabled when any provider is configured.

## 5. Functional Requirements

- **FR-1.** The system MUST migrate the following call sites to the resolver (complete/stream/vision as appropriate):
  - Tutor + persistent tutor sessions (`tutor.go`, `tutor_sessions_http.go`)
  - Study buddy (`studybuddy_http.go`)
  - Grading agent (`grading_agent_http.go`, `gradingagent/`, dry-run WS)
  - Notebook RAG + flashcards (`me_notebook.go`, `notebookrag/`)
  - Course syllabus / structure module generation
  - Lesson generator (`lesson_generator_http.go`, `lessonplanai/`)
  - Translation + course translation
  - Reading-level simplification (`reading_level.go`, `contentsimplificationai/`)
  - Alt-text (`alt_text_http.go`, `alttextai/`)
  - Report card suggestions
  - Coaching tips background (`coachingtips/`, `background/`)
  - Plagiarism/originality internal AI path (`plagiarism/`, `originality_*`)
- **FR-2.** Services that currently take `*openrouter.Client` MUST take a small interface satisfied by the resolver (e.g. `Completer` already started in `notebookrag`).
- **FR-3.** `platform_features` / client gates MUST replace `OpenRouterConfigured` with `AIConfigured` (or keep field as deprecated alias of “any provider configured”).
- **FR-4.** Usage logging MUST use `recordAIProviderUsage` / `EntryFromProviderUsage` with the actual provider from `CallMeta`.
- **FR-5.** Gateway evaluation MUST run before provider calls with the resolved model id and provider name.
- **FR-6.** When abstraction flag is off, behavior MUST remain OpenRouter-only (compatibility), but implementation SHOULD still go through resolver with `ProviderOpenRouter` forced — avoid dual code paths long-term.
- **FR-7.** Streaming UIs MUST keep SSE/chunk behavior; if active provider lacks stream (AP.1 policy), either buffered complete with progressive flush or clear 503 with message — product decision documented in open questions.
- **FR-8.** Error copy MUST stop saying “Set an OpenRouter API key…”; use provider-agnostic “Configure AI under Settings → Intelligence”.

## 6. Non-Functional Requirements

- **Performance** — No regression > 5% p95 on tutor first token for OpenRouter path.
- **Security** — Gateway fail-closed preserved; secrets not passed into handlers.
- **Privacy** — Inference logs remain hashed per 10.17.
- **Accessibility** — Streaming live regions unchanged.
- **Scalability** — Shared resolver instance on `Deps` (not new-per-request factory that rebuilds caches poorly).
- **Reliability** — Fallback chain honored for non-stream calls; define stream policy.
- **Observability** — Every path emits provider+model metrics.
- **Maintainability** — Prefer one `Deps.aiCompleter(ctx, orgID)` helper.
- **Internationalization** — Update user-facing error strings in all locales when touched.
- **Backward compatibility** — OpenRouter-only deployments keep working without config changes.

## 7. Acceptance Criteria

- **AC-1.** *Given* Anthropic org BYOK and flag on, *When* quiz/module generation runs, *Then* outbound HTTP hits Anthropic, not OpenRouter.
- **AC-2.** *Given* no OpenRouter key but OpenAI platform key, *When* GET platform features, *Then* AI-capable flags report configured.
- **AC-3.** *Given* OpenRouter-only legacy config, *When* full e2e tutor + flashcards + grading dry-run, *Then* all pass.
- **AC-4.** *Given* `rg` over `server/internal` excluding `aiprovider/openrouter.go` and `service/openrouter/`, *When* AP.4 done, *Then* no production references to `openRouterClient` or `OpenRouterAPIKey` for invocation (config dual-read may remain until AP.9).
- **AC-5.** *Given* a blocked gateway decision, *When* any migrated feature is called, *Then* the same block reasons/messages as today apply.
- **AC-6.** *Given* successful call, *When* `analytics.ai_usage_log` is inspected, *Then* `provider` is not defaulted incorrectly to openrouter when another backend served the call.

## 8. Data Model

- No new tables; relies on AP.2/AP.3.
- Possibly add helper columns only if feature-model binding needs provider scope (prefer not).

## 9. API Surface

- No new routes required.
- Response fields: deprecate `openRouterConfigured` in favor of `aiConfigured` + `aiProvidersConfigured[]` (keep old field one release).
- Error messages updated on existing AI endpoints.

## 10. UI / UX

- Minimal: toast/error string updates where they mention OpenRouter.
- Feature availability badges follow new flags (AP.7 polish).

## 11. AI / ML Considerations

- Prompt templates stay in `systemprompts` / service packages.
- Model resolution via AP.3; per-feature user model preferences still applied as model override to resolver.
- Vision grading only when provider supports vision (else clear error).

## 12. Integration Points

Inventory of primary files (non-exhaustive):

```
server/internal/httpserver/tutor.go
server/internal/httpserver/tutor_sessions_http.go
server/internal/httpserver/studybuddy_http.go
server/internal/httpserver/grading_agent_http.go
server/internal/httpserver/grading_agent_ai_build.go
server/internal/httpserver/grading_agent_dry_run_ws.go
server/internal/httpserver/me_notebook.go
server/internal/httpserver/course_syllabus.go
server/internal/httpserver/structure_module_http.go
server/internal/httpserver/lesson_generator_http.go
server/internal/httpserver/translation.go
server/internal/httpserver/course_translation.go
server/internal/httpserver/reading_level.go
server/internal/httpserver/alt_text_http.go
server/internal/httpserver/report_cards_http.go
server/internal/httpserver/platform_features.go
server/internal/httpserver/originality_http.go
server/internal/httpserver/server.go
server/internal/service/gradingagent/
server/internal/service/notebookrag/
server/internal/service/lessonplanai/
server/internal/service/alttextai/
server/internal/service/contentsimplificationai/
server/internal/service/coachingtips/
server/internal/service/plagiarism/
server/internal/background/periodic.go
server/internal/background/coaching_tips.go
server/internal/background/originality_sweep.go
server/internal/platformstate/platformstate.go
```

## 13. Dependencies & Sequencing

- After AP.1–AP.3.
- Before AP.6 (usage correctness depends on migration), AP.9.
- Can partially migrate in PR slices (recommended order below).

### Suggested PR slices

1. Shared `Deps` resolver wiring + `AIConfigured` flag
2. Non-stream text features (translation, syllabus, structure, reading level, report cards, coaching)
3. Notebook RAG/flashcards
4. Streaming tutor + study buddy
5. Grading agent + vision + originality
6. Delete dead OpenRouter injection paths

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Stream gap on non-OR providers | H | H | Buffered fallback or feature gate per capability |
| JSON mode differences break grader parsing | M | H | Keep JSON-mode tests per provider; soft fallback retry without JSON mode (existing grader behavior) |
| Missed call site | M | H | CI grep gate forbidding new `openrouter.Client` deps outside allowlist |
| Org context missing on some handlers | M | M | Standardize org resolution helper before complete |

## 15. Rollout Plan

- Keep `AI_PROVIDER_ABSTRACTION_ENABLED` default false until slices 1–3 green; enable for dogfood orgs.
- Prefer single code path with forced OpenRouter when flag off.
- Rollback: flag off + platform OpenRouter credential.

## 16. Test Plan

- **Unit** — Each service with mock `Completer`.
- **Integration** — Provider e2e extended per feature smoke.
- **E2E** — Existing Playwright AI tests against OpenRouter mock/stub; add one Anthropic mock path.
- **Security** — Gateway matrix unchanged.
- **Performance** — Tutor stream smoke.

## 17. Documentation & Training

- Changelog: “AI features respect org/platform provider settings.”
- Update internal architecture notes that OpenRouter is optional.

## 18. Open Questions

1. Buffered stream fallback vs hard fail for non-streaming providers?
2. Background jobs without user/org — always platform credentials?
3. Should plagiarism internal AI be multi-provider or remain optional/disabled without platform text provider?

## 19. References

- [README inventory](README.md)
- `server/internal/httpserver/ai_provider_settings_http.go` (resolver construction)
- `server/internal/httpserver/me_notebook.go` (partial migration pattern)
- Related: [AP.1](AP.1-provider-capability-interface.md), [AP.6](AP.6-usage-disclosure-observability.md)
