# AP.5 — Unify Intelligence Admin UX (Platform + Org Providers)

> Implementation plan. Source: multi-provider BYOK epic ([README](README.md)).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AP.5 |
| **Section** | AI Providers |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | PARTIAL — Intelligence Models UI is OpenRouter-only; separate org BYOK panel exists behind flag |
| **Estimated effort** | M (2–3w) |
| **Owner (proposed)** | Frontend + Platform |
| **Depends on** | AP.2, AP.3 |
| **Unblocks** | AP.7 |

---

## 1. Problem Statement

Global admins configure AI under **Settings → Intelligence → Models** with a single OpenRouter API key and OpenRouter model pickers (`clients/web/src/pages/lms/settings.tsx`). Org admins have a separate **AI provider** panel (`ai-provider-settings-panel.tsx`) that only appears when the abstraction flag is on. The split confuses operators, still presents OpenRouter as the only platform path, and does not expose Azure/Bedrock/Vertex settings fields. Multi-provider BYOK needs one coherent Intelligence UX.

## 2. Goals

- Unified **platform** Intelligence settings: add/remove providers, credentials, defaults, feature model bindings.
- Unified **org** override: provider, fallback, BYOK, extra settings, test connection.
- Model pickers driven by AP.3 catalogs (provider-aware).
- Clear empty/error states when no provider is configured.

## 3. Non-Goals

- Mobile admin redesign beyond string/API alignment (AP.7).
- Cost dashboards redesign (AP.6 may tweak copy only).
- Prompt template editor changes.

## 4. Personas & User Stories

- **As a platform admin**, I want to add OpenAI and Anthropic keys and set which provider is default so that OpenRouter is optional.
- **As an org admin**, I want Azure endpoint + key fields and Test Connection so that I know grading will work before term start.
- **As a support engineer**, I want consistent labels for “configured / not configured” so that tickets are diagnosable.

## 5. Functional Requirements

- **FR-1.** Platform Intelligence UI MUST list supported providers (including OpenRouter) with status badges (configured, enabled, default).
- **FR-2.** For each provider, UI MUST collect required credential + settings fields (matrix below) with write-only secret inputs.
- **FR-3.** Feature model selectors (image, course setup, flashcards, vibe, grader) MUST load options from the active/default provider catalog; allow advanced raw model id.
- **FR-4.** Org AI settings panel MUST support provider, fallback, BYOK, provider settings JSON/fields, test connection (existing API).
- **FR-5.** UI MUST hide or disable providers the platform policy disallows for tenants.
- **FR-6.** Deprecate standalone “OpenRouter API key” primary field; migrate to provider card (keep temporary dual-field if needed for one release).
- **FR-7.** Accessibility: labels, focus order, error announcements, keyboard-operable selects.
- **FR-8.** i18n keys for all new strings (web); no hard-coded OpenRouter-only instructions in primary path.

### Provider field matrix (UI)

| Provider | Secret | Extra fields |
|---|---|---|
| OpenRouter | API key | (optional site URL headers already server-side) |
| Anthropic | API key | optional base URL |
| OpenAI | API key | optional base URL |
| Azure OpenAI | API key | `azure_base_url`, API version, default deployment |
| Bedrock | API key or access key material (AP.8 refines) | `aws_region`, optional base URL |
| Vertex | API key / access token (AP.8 refines) | `gcp_project`, `gcp_location` |

## 6. Non-Functional Requirements

- **Performance** — Settings page interactive < 2s with cached catalogs.
- **Security** — Secrets never re-displayed; placeholder when configured; no secrets in client logs.
- **Privacy** — N/A.
- **Accessibility** — WCAG 2.1 AA; axe clean on new panels.
- **Scalability** — N/A.
- **Reliability** — Test connection failures show provider error text sanitized.
- **Observability** — Client errors via existing toast/mutation reporting.
- **Maintainability** — Shared `ProviderCredentialForm` component for platform + org.
- **Internationalization** — All user strings in locale files.
- **Backward compatibility** — Existing saved OpenRouter key appears under OpenRouter card after load.

## 7. Acceptance Criteria

- **AC-1.** *Given* platform admin opens Intelligence, *When* no keys exist, *Then* empty state explains multi-provider BYOK and links to docs.
- **AC-2.** *Given* admin saves Anthropic key as default, *When* page reloads, *Then* key shows as configured placeholder and models list is Anthropic catalog.
- **AC-3.** *Given* org admin configures Azure fields, *When* Test Connection succeeds, *Then* toast shows latency + provider name.
- **AC-4.** *Given* keyboard-only user, *When* completing provider form, *Then* all controls are reachable and errors announced.
- **AC-5.** *Given* abstraction flag off (if still present), *When* UI loads, *Then* either force OpenRouter-only simplified mode or show banner that multi-provider is disabled — no broken empty panel.

## 8. Data Model

- Uses AP.2 APIs; no direct DB access from client.

## 9. API Surface

- Consumes AP.2/AP.3 endpoints.
- May add `GET /api/v1/settings/ai/providers/schema` returning field descriptors for dynamic forms (SHOULD).

## 10. UI / UX

### Pages / components

- `clients/web/src/pages/lms/settings.tsx` — replace OpenRouter-centric Models section.
- New/extended: `components/settings/ai-providers-panel.tsx`, reuse pieces of `ai-provider-settings-panel.tsx`.
- `image-model-picker.tsx` / text model pickers — provider prop + catalog fetch.
- Side nav Intelligence entry unchanged.

### Flows

1. Platform admin → Intelligence → Providers → Add OpenRouter key → Set as default → Save.
2. Platform admin → Feature models → pick aliases → Save.
3. Org admin → AI provider → Select Azure → fill endpoint/key → Test → Save.
4. Clear key → confirm → status becomes not configured.

### States

- Loading skeletons for catalogs.
- Error: catalog failed → curated fallback + warning.
- Offline: existing app offline handling.

## 11. AI / ML Considerations

- Test connection uses minimal “Hello” prompt (existing); show token usage if returned.

## 12. Integration Points

- Web authorizedFetch to `/api/v1/settings/ai*`, `/api/v1/admin/ai-settings*`.
- Feature flag read from platform features payload.
- CLI already has partial AI provider commands — keep parity with field names (AP.7).

## 13. Dependencies & Sequencing

- After AP.2/AP.3 APIs.
- Parallel with late AP.4.
- Before AP.7 marketing/mobile polish.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Settings page already large/complex | H | M | Extract panels; avoid mega-PR |
| Users lose OpenRouter key on save bug | L | H | Placeholder semantics tests; e2e save/reload |
| Azure fields confusing | M | M | Inline help + docs link |

## 15. Rollout Plan

- Ship UI behind same abstraction flag or progressive enhancement once AP.2 live.
- Dogfood with internal admins.
- Rollback: feature flag to legacy OpenRouter form component.

## 16. Test Plan

- **Unit** — form validation, placeholder logic.
- **Component** — provider card render matrix.
- **E2E** — Playwright: save provider, reload, test connection (mocked API).
- **Accessibility** — axe on Intelligence panels.
- **Manual** — each provider field set.

## 17. Documentation & Training

- Admin help: “Configure AI providers (BYOK).”
- Screenshots for OpenRouter, Azure, Anthropic.

## 18. Open Questions

1. Single default provider vs per-feature provider in UI for v1? (Recommend single default + model overrides.)
2. Should platform admins see org-level BYOK status rollup?

## 19. References

- `clients/web/src/pages/lms/settings.tsx`
- `clients/web/src/components/settings/ai-provider-settings-panel.tsx`
- `clients/web/src/components/image-model-picker.tsx`
- Related: [AP.2](AP.2-credential-store-and-byok.md), [AP.3](AP.3-model-registry-and-catalog.md)
