# AP.7 — Clients, Documentation & Trust Surfaces

> Implementation plan. Source: multi-provider BYOK epic ([README](README.md)).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AP.7 |
| **Section** | AI Providers |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | OpenRouter-only copy across web marketing, mobile locales, CLI help, trust center, README |
| **Estimated effort** | S (≤1w) |
| **Owner (proposed)** | Frontend / Docs / Mobile |
| **Depends on** | AP.5, AP.6 |
| **Unblocks** | AP.9 (GA criteria) |

---

## 1. Problem Statement

User-facing and public materials still instruct operators to “add an OpenRouter API key” and describe AI as “via OpenRouter.” That is incorrect for BYOK multi-provider deployments and creates compliance risk on trust/subprocessor pages when institutions use Azure or Anthropic-direct. Clients (web residual strings, mobile admin, CLI, www marketing, repo README) and legal/trust surfaces need a consistent **provider-agnostic** story with OpenRouter listed as one option.

## 2. Goals

- Replace OpenRouter-only user instructions with multi-provider BYOK language.
- Keep OpenRouter accurately listed where it is a subprocessor when used.
- Align CLI, mobile, web residual copy, www, and developer docs.
- Ensure e2e trust-center tests reflect dynamic/provider-aware expectations.

## 3. Non-Goals

- Building the Intelligence UI (AP.5).
- Implementing provider backends (AP.1/AP.8).
- Full legal DPA redlines (legal owns final wording; eng updates product surfaces).

## 4. Personas & User Stories

- **As a prospective customer**, I want pricing/docs to say I can bring Azure/Anthropic keys so that I know we are not locked into OpenRouter.
- **As a mobile admin**, I want settings strings that match the web multi-provider UI.
- **As a security reviewer**, I want the trust center to describe AI subprocessors accurately for this deployment.

## 5. Functional Requirements

- **FR-1.** Web residual strings (`ai-disclosure-i18n`, banners, quiz page option labels, search hints) MUST be provider-agnostic or dynamic from disclosure API.
- **FR-2.** Mobile locales (`clients/mobile/locales/*.json`) MUST update admin AI models/reports strings for multi-provider.
- **FR-3.** CLI help and settings commands MUST document all providers; `openRouterApiKey` fields marked deprecated if still present.
- **FR-4.** `www/` pricing and product pages MUST describe optional AI via customer-chosen providers / OpenRouter.
- **FR-5.** Repo `README.md`, `docs/TECH_STACK.md`, and admin docs MUST describe multi-provider configuration.
- **FR-6.** Trust center sub-processor list MUST distinguish **platform default routing** vs **customer BYOK** (customer’s provider is not always Lextures’ subprocessor — legal-approved wording).
- **FR-7.** E2E tests that assert “OpenRouter” cells MUST be updated to the new trust model.
- **FR-8.** In-app help links MUST point to the new admin guide.

## 6. Non-Functional Requirements

- **Performance** — N/A.
- **Security** — Do not document secret values or example real keys.
- **Privacy & Compliance** — Trust copy legal-reviewed; align with S07.
- **Accessibility** — No regression on trust pages.
- **Scalability** — N/A.
- **Reliability** — N/A.
- **Observability** — N/A.
- **Maintainability** — Single glossary: “AI provider”, “BYOK”, “OpenRouter (provider)”.
- **Internationalization** — Update all mobile locale files together.
- **Backward compatibility** — Old bookmarks to OpenRouter docs can remain as secondary links.

## 7. Acceptance Criteria

- **AC-1.** *Given* English mobile locale, *When* searching for “OpenRouter API key” as the only key instruction, *Then* primary strings instead refer to AI provider credentials / BYOK.
- **AC-2.** *Given* www pricing FAQ, *When* reading AI answer, *Then* it mentions customer-provided provider keys, not OpenRouter alone.
- **AC-3.** *Given* trust center, *When* BYOK-only Azure deployment mode is documented, *Then* copy does not claim OpenRouter processes data if not configured (per legal template).
- **AC-4.** *Given* `README` AI section, *When* setup steps are followed, *Then* they match Intelligence multi-provider UI.
- **AC-5.** *Given* e2e trust-center test, *When* run, *Then* it passes under new expectations.

## 8. Data Model

- None.

## 9. API Surface

- Consumes disclosure JSON from AP.6; no new routes required.

## 10. UI / UX

- Copy-only and minor conditional rendering on disclosure/trust pages.
- CLI help text updates.
- Marketing pages copy updates.

## 11. AI / ML Considerations

- Model cards under `docs/ai/*` SHOULD note provider-agnostic routing where relevant.

## 12. Integration Points

| Path | Change |
|---|---|
| `clients/web/src/lib/ai-disclosure-i18n.ts` | Dynamic/provider-agnostic |
| `clients/web/src/components/ai-disclosure-banner.tsx` | Provider label |
| `clients/web/src/content/trust/sub-processors.ts` | BYOK-aware rows |
| `clients/web/src/pages/lms/course-module-quiz-page.tsx` | “AI-generated” label |
| `clients/mobile/locales/{en,es,fr,ar}.json` | Admin AI strings |
| `clients/cli/cmd/settings.go`, `platform_settings_logic.go` | Help + deprecations |
| `www/src/pages/pricing-page.tsx`, `self-learner-page.tsx`, … | Marketing |
| `README.md`, `docs/TECH_STACK.md` | Dev setup |
| `e2e/tests/trust-center.spec.ts` | Assertions |
| `docs/legal/review-checklist.md` | Checklist line items |

## 13. Dependencies & Sequencing

- After AP.5 (UI exists to document) and AP.6 (disclosure truth).
- Before AP.9 GA checklist.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Legal disagreement on BYOK subprocessor wording | M | H | Early legal review; template with “when configured” |
| Locale drift | M | L | en source of truth + sync checklist |
| Marketing overpromises providers | M | M | List GA providers only |

## 15. Rollout Plan

- Docs + copy can ship as soon as AP.5 is in dogfood.
- No flag; ensure docs don’t claim GA until AP.9.
- Rollback: revert copy commits.

## 16. Test Plan

- **E2E** — trust center, pricing smoke if covered.
- **Manual** — locale spot-check FR/ES/AR.
- **Docs** — link checker on new admin guide paths.

## 17. Documentation & Training

- New admin guide: `docs/` or help center “AI providers & BYOK”.
- Developer guide: adding a provider (pointer to AP.1).
- Support macros for “how do I use Azure OpenAI?”.

## 18. Open Questions

1. Hosted SaaS multi-tenant: do we still offer platform-managed OpenRouter by default?
2. Should trust center be instance-dynamic (from disclosure API) vs static marketing list?

## 19. References

- [README](README.md) inventory
- `e2e/tests/trust-center.spec.ts`
- Related: [AP.5](AP.5-admin-intelligence-ui.md), [AP.6](AP.6-usage-disclosure-observability.md), [S07](../standards/S07-cross-border-transfer-subprocessor-governance.md)
