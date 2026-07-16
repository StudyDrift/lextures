# AP.1 — Complete Provider Capability Interface

> Implementation plan. Source: multi-provider BYOK epic ([README](../../plan/ai-providers/README.md)). Extends [16.7](../16-integrations-extensibility/16.7-ai-provider-abstraction.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AP.1 |
| **Section** | AI Providers |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE — `Provider` supports Complete, CompleteStream, CompleteVision, Embed; optional `ImageProvider`; ChatOptions include JSONMode/MaxTokens/Temperature/Timeout |
| **Estimated effort** | M (2–3w) |
| **Owner (proposed)** | AI / Platform |
| **Depends on** | — |
| **Unblocks** | AP.3, AP.4, AP.8 |

---

## 1. Problem Statement

The abstraction in `server/internal/service/aiprovider/` only models a text chat completion. Production features require **streaming** (tutor, study buddy), **vision** (grader agent, alt-text), **JSON mode + max tokens** (grading workflows), and eventually **image generation** and **embeddings**. Because those capabilities live only on `openrouter.Client`, features cannot switch providers without rewriting each call site. Completing the interface is the foundation for true multi-provider support.

## 2. Goals

- Expand `Provider` (and resolver) so every AI capability Lextures uses is expressible without importing OpenRouter.
- Preserve OpenRouter as a first-class implementation that reuses existing client code.
- Normalize options, errors, and usage metadata across backends.
- Document capability matrix so UI and callers can degrade gracefully when a provider lacks a feature.

## 3. Non-Goals

- Migrating call sites (AP.4).
- Credential storage redesign (AP.2).
- Full production-grade Bedrock IAM / Vertex ADC (AP.8).
- Embedding-powered product features (only the interface + best-effort impls).

## 4. Personas & User Stories

- **As a platform engineer**, I want one interface for chat, stream, and vision so that I do not couple features to OpenRouter.
- **As a feature author**, I want capability discovery so that I can disable image generation when the active provider cannot produce images.
- **As a QA engineer**, I want a dry-run provider that implements every method so that CI never needs live keys.

## 5. Functional Requirements

- **FR-1.** The system MUST extend `aiprovider.Provider` (or a composed set of interfaces) to support at minimum: `Complete`, `CompleteStream`, `CompleteVision` (or multimodal messages), and `Embed`. Image generation MAY be a separate `ImageProvider` interface implemented where available.
- **FR-2.** `ChatOptions` MUST include at least: `JSONMode`, `MaxTokens`, and optional `Temperature` / timeout override (parity with `openrouter.ChatOptions` + `WithTimeout`).
- **FR-3.** Streaming MUST invoke a chunk callback with the same semantics as `openrouter.ChatCompletionStream` and return final `UsageInfo` when the provider supplies it.
- **FR-4.** Multimodal messages MUST support text + image URL (and data-URL) parts sufficient for alt-text and vision grading.
- **FR-5.** Each concrete provider MUST return `ErrNotSupported` (typed) for unimplemented capabilities rather than panicking or silently no-oping.
- **FR-6.** `Resolver` MUST expose stream and vision entry points with the same tenant/fallback rules as `Complete`.
- **FR-7.** OpenRouter adapter MUST wrap existing stream, vision, and chat paths without behavior change for OpenRouter-only tenants.
- **FR-8.** Anthropic, OpenAI/Azure, Bedrock, and Vertex MUST implement non-streaming `Complete` with normalized errors (`ProviderError` + `IsRetryable`); stream/vision SHOULD be implemented where the public API supports it, otherwise `ErrNotSupported`.
- **FR-9.** Dry-run provider MUST implement all interface methods with deterministic synthetic output for tests.

## 6. Non-Functional Requirements

- **Performance** — Abstraction overhead ≤ 5 ms vs direct OpenRouter client; stream first-token latency not increased beyond measurement noise.
- **Security** — No API keys in error messages or metrics labels; redact provider response bodies in logs.
- **Privacy & Compliance** — Prompt content not logged; usage metadata only.
- **Accessibility** — N/A (no UI in this story).
- **Scalability** — Providers remain stateless; shared HTTP client pool OK.
- **Reliability** — Hard timeout default 120s (configurable); stream cancels with context.
- **Observability** — Reuse `recordLatency` / `recordError` / `recordCostUSD` for all methods; add `operation` label (`complete|stream|vision|embed|image`).
- **Maintainability** — One file per provider; capability matrix table in package docs.
- **Internationalization** — N/A.
- **Backward compatibility** — Existing `Complete` signatures remain; new methods additive.

## 7. Acceptance Criteria

- **AC-1.** *Given* a mock OpenRouter backend, *When* `Resolver.CompleteStream` is called, *Then* chunks match the existing OpenRouter stream parser behavior.
- **AC-2.** *Given* Anthropic provider with a key, *When* `Complete` is called with `JSONMode`, *Then* the request uses Anthropic’s structured-output / system+messages mapping correctly.
- **AC-3.** *Given* a provider without streaming, *When* `CompleteStream` is called, *Then* `errors.Is(err, ErrNotSupported)` and fallback rules apply only for retryable transport errors (not capability gaps).
- **AC-4.** *Given* vision alt-text inputs, *When* `CompleteVision` runs on OpenRouter adapter, *Then* output parity tests pass against the current `VisionMessage` path.
- **AC-5.** *Given* dry-run mode, *When* any interface method is invoked in unit tests, *Then* no network I/O occurs.
- **AC-6.** *Given* a 503 from primary, *When* fallback is configured for `Complete`, *Then* one retry occurs (existing FR-7 from 16.7 preserved).

## 8. Data Model

- No DB migrations required.
- Internal types only: extend `Message` to support multimodal parts, or add `Content []ContentPart`.

## 9. API Surface

- No new public HTTP routes.
- Internal Go API changes in `server/internal/service/aiprovider/{provider,types,resolver,*.go}`.
- Callers continue using HTTP feature routes unchanged until AP.4.

## 10. UI / UX

- None in this story (capability matrix consumed later by AP.5).

## 11. AI / ML Considerations

- Model IDs still provider-specific at the adapter boundary; aliases remain in AP.3.
- Prompt templates unchanged.
- Eval of output quality across providers is out of scope (future eval harness).

## 12. Integration Points

- `server/internal/service/openrouter/{openrouter,stream}.go` — adapt, do not delete.
- `server/internal/service/aiprovider/{anthropic,openai,bedrock,vertex,openrouter,dryrun}.go`.
- Metrics in `aiprovider/metrics.go` and `telemetry.ObserveAIProvider`.

## 13. Dependencies & Sequencing

- Must ship after: —
- Must ship before: AP.4 (migration), AP.3 (catalog can start in parallel after types land).
- Shared infra: none new.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Provider stream formats differ (SSE vs Anthropic events) | H | M | Per-provider parsers; shared chunk callback contract |
| Vision part schemas differ | H | M | Normalize to internal `ContentPart`; map per provider |
| Incomplete non-OR stream support blocks AP.4 | M | H | Allow stream fallback: buffer `Complete` when stream unsupported (document latency tradeoff) |

## 15. Rollout Plan

- Ship behind existing `AI_PROVIDER_ABSTRACTION_ENABLED` for new methods; OpenRouter path unchanged for legacy callers.
- No user-facing flag flip until AP.4.
- Rollback: revert package; no schema.

## 16. Test Plan

- **Unit** — Each provider `Complete` with httptest; stream parsers; vision payload mapping; dry-run; `IsRetryable`.
- **Integration** — Optional live-key smoke behind build tags (not CI-required).
- **End-to-end** — Deferred to AP.4/AP.9.
- **Security** — Error strings never include `Authorization` / API key material.
- **Performance** — Benchmark resolver dispatch vs direct client.

## 17. Documentation & Training

- Package godoc capability matrix table.
- Developer note: “Adding a provider implements Provider + optional ImageProvider.”

## 18. Open Questions

1. Should stream-unsupported providers auto-fallback to buffered `Complete` for tutor UX, or fail closed until implemented?
2. Is image generation in-scope for GA providers, or OpenRouter-only until AP.8?
3. Do we need tool/function-calling in the interface for future agents?

## 19. References

- `server/internal/service/aiprovider/provider.go`, `types.go`, `resolver.go`
- `server/internal/service/openrouter/openrouter.go`, `stream.go`
- Related: [AP.4](AP.4-migrate-call-sites.md), [16.7](../16-integrations-extensibility/16.7-ai-provider-abstraction.md)
