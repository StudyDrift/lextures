# AP.9 — Test Matrix, GA Rollout & OpenRouter Peer Deprecation

> Implementation plan. Source: multi-provider BYOK epic ([README](README.md)).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AP.9 |
| **Section** | AI Providers |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | Flag `AI_PROVIDER_ABSTRACTION_ENABLED` default false; OpenRouter remains de-facto sole path |
| **Estimated effort** | S–M (1–2w) |
| **Owner (proposed)** | AI / QA / Platform |
| **Depends on** | AP.4, AP.5, AP.6, AP.7 (AP.8 optional for cloud-native GA) |
| **Unblocks** | Epic complete; remove dual paths |

---

## 1. Problem Statement

Without a deliberate GA plan, multi-provider work risks shipping half-migrated: flag off in production, dual code paths, deprecated OpenRouter columns forever, and incomplete tests. This story defines the **quality bar**, **default-on flip**, **deprecation of OpenRouter-only assumptions**, and **rollback**. OpenRouter remains a fully supported **peer provider**, not a special case hardcoded through the app.

## 2. Goals

- Comprehensive automated test matrix across providers (mocked) and critical features.
- Production default: abstraction on; OpenRouter is one configured provider when present.
- Remove or strictly allowlist residual OpenRouter-only APIs/fields after dual-read window.
- Document GA criteria and support playbooks.

## 3. Non-Goals

- Live paid API calls in CI.
- Sunset of OpenRouter as a product option.
- New provider implementations.

## 4. Personas & User Stories

- **As a release manager**, I want a clear checklist before flipping the default flag.
- **As a support engineer**, I want a rollback path if a provider outage coincides with the flip.
- **As a developer**, I want CI to fail if someone reintroduces direct OpenRouter coupling.

## 5. Functional Requirements

- **FR-1.** CI MUST include a grep/allowlist check: production code outside `service/openrouter` and `aiprovider/openrouter.go` MUST NOT call OpenRouter APIs directly.
- **FR-2.** Automated tests MUST cover resolver resolution order, BYOK mask, catalog without OpenRouter, and at least one stream + one vision path via mocks for ≥2 providers.
- **FR-3.** E2E MUST cover admin provider save + feature smoke with mocked backend (or stub provider).
- **FR-4.** Default configuration for new installs MUST not require OpenRouter; AI optional until any provider credential exists.
- **FR-5.** `AI_PROVIDER_ABSTRACTION_ENABLED` MUST default true (or be removed) after soak; document migration notes for operators who disabled it.
- **FR-6.** Deprecate: `openRouterApiKey` JSON fields, `OpenRouterConfigured` feature flag name, `platform_app_settings.openrouter_api_key` column (after backfill verified).
- **FR-7.** Deprecation MUST include API changelog and dual-read period ≥1 minor release.
- **FR-8.** Rollback procedure MUST restore prior behavior within one deploy (flag or config).

## 6. Non-Functional Requirements

- **Performance** — No regression SLOs vs pre-flip OpenRouter baseline.
- **Security** — Final secret scan; no keys in fixtures.
- **Privacy** — Disclosure pages reviewed post-flip.
- **Accessibility** — Admin UI axe gate in CI if not already.
- **Scalability** — N/A.
- **Reliability** — Soak period in staging with synthetic traffic.
- **Observability** — Alert on elevated `ai_provider_errors_total` post-flip.
- **Maintainability** — Delete dead code paths after deprecation window.
- **Internationalization** — All locales updated before GA (AP.7).
- **Backward compatibility** — Dual-read until FR-6 completes.

## 7. Acceptance Criteria

- **AC-1.** *Given* CI main branch, *When* a PR adds `openRouterClient()` usage outside allowlist, *Then* check fails.
- **AC-2.** *Given* staging with Anthropic-only credentials, *When* tutor, notebook, and syllabus features run, *Then* success rate meets soak criteria (≥99% synthetic).
- **AC-3.** *Given* production config post-flip with OpenRouter key only, *When* traffic runs, *Then* behavior matches pre-flip (no user-visible regression).
- **AC-4.** *Given* deprecation window elapsed, *When* migration drops `openrouter_api_key` column, *Then* dual-read code removed and tests green.
- **AC-5.** *Given* runbook executed for rollback, *When* flag set false / prior deploy, *Then* AI features recover within RTO 30 minutes.

## 8. Data Model

- Final migration: drop `settings.platform_app_settings.openrouter_api_key` after credential store is sole source.
- Optionally drop legacy single-key BYOK rows after multi-provider secrets backfill verified.

## 9. API Surface

- Remove deprecated request/response fields after changelog notice.
- OpenAPI version bump.

## 10. UI / UX

- Remove legacy OpenRouter-only form if still behind codepath.
- In-app “What’s new” optional for admins.

## 11. AI / ML Considerations

- Optional qualitative eval: same prompts across OpenRouter vs direct providers; document known differences (not a ship blocker unless severity high).

## 12. Integration Points

- CI workflows (GitHub Actions / make targets).
- `server/test/ai_provider_e2e_test.go`, feature e2e specs.
- Config defaults in `config.go`.
- Docs plan README status → move stories to completed when done.

## 13. Dependencies & Sequencing

- Last story in epic.
- AP.8 can GA later as “enterprise cloud auth” without blocking OpenRouter/OpenAI/Anthropic GA.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Silent call-site miss | M | H | Grep gate + usage metric anomaly (all traffic still openrouter label) |
| Flag flip during provider outage | L | H | Flip during low traffic; watch error burn rate |
| Column drop too early | L | H | Dual-read metrics: old column reads == 0 for 14 days |

## 15. Rollout Plan

1. **Dogfood** — internal orgs, abstraction on.
2. **Pilot** — 2–3 external tenants (Azure, Anthropic, OpenRouter).
3. **Default on** for new deploys / hosted.
4. **Announce** deprecation timeline for old fields.
5. **Remove** dual paths and old column.
6. **GA** marketing (AP.7 already updated).

Feature flag: `AI_PROVIDER_ABSTRACTION_ENABLED` → default true → delete.

Rollback: set flag false or redeploy previous version; credentials remain in DB.

## 16. Test Plan

| Layer | Coverage |
|---|---|
| Unit | Registry, resolver, each provider mock, secrets mask |
| Integration | Credential CRUD, migration backfill, gateway+resolver |
| E2E | Admin configure provider; one AI feature; trust copy |
| Security | Secret redaction; authz on credential endpoints |
| Accessibility | Intelligence settings axe |
| Load | Optional soak on staging |
| Manual | Azure + Anthropic + OpenRouter matrix checklist |

### Manual GA checklist (excerpt)

- [ ] Platform OpenRouter only — all major AI features
- [ ] Platform Anthropic only — text features
- [ ] Org Azure BYOK override — generation + test connection
- [ ] No credentials — AI features disabled cleanly
- [ ] Disclosure + reports show correct provider
- [ ] Mobile admin strings acceptable
- [ ] CLI get/set/test provider

## 17. Documentation & Training

- Release notes.
- Support playbook: multi-provider incidents.
- Mark epic stories completed under `docs/completed/` when shipped.

## 18. Open Questions

1. Hosted multi-tenant: will Lextures continue offering a managed OpenRouter key as a paid add-on?
2. Minimum provider set for “GA” badge: OpenRouter + OpenAI + Anthropic only, with Azure/Bedrock/Vertex as preview?

## 19. References

- [README](README.md)
- `server/internal/config/config.go` (`AiProviderAbstractionEnabled`)
- `server/test/ai_provider_e2e_test.go`
- Related: [AP.4](AP.4-migrate-call-sites.md)–[AP.8](AP.8-provider-auth-hardening.md)
