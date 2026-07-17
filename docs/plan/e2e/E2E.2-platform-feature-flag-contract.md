# E2E.2 — Platform Feature Flag Contract

> Implementation plan. Source: completed feature audit against `PLATFORM_FEATURE_DEFINITIONS`, platform settings handlers, runtime feature payload, and Playwright specs (2026-07-17).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | E2E.2 |
| **Section** | End-to-End Coverage |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | THIN |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Platform / QA |
| **Depends on** | E2E.4 |
| **Unblocks** | E2E.3 |

---

## 1. Problem Statement

The Global platform page is the operational control plane for 132 boolean definitions at audit time. Individual E2E specs frequently force one flag through an API helper, but no registry-wide test proves that every displayed control can be read, changed by a global admin, persisted, reflected by `/api/v1/platform/features`, protected from lower roles, and restored.

## 2. Goals

- Establish a data-driven contract for every platform boolean definition.
- Detect settings/UI/runtime payload drift automatically.
- Verify global-admin-only mutation and environment-owned read-only behavior.
- Cover save failure, reload, and safe state restoration.

## 3. Non-Goals

- Run the full user journey for every platform feature.
- Override environment-owned fields in CI.
- Assert external providers such as Stripe, SES, Redis, or AI vendors are operational.

## 4. Personas & User Stories

- **As a global admin**, I want each feature control to persist and take effect so that I can roll out or kill functionality safely.
- **As an org admin**, I want platform-only controls protected from me.
- **As an operator**, I want registry drift to fail CI before an unmanageable flag ships.

## 5. Functional Requirements

- **FR-1.** The suite MUST derive or validate a manifest containing every key in `PLATFORM_FEATURE_DEFINITIONS`.
- **FR-2.** Each database-owned flag MUST be toggled through Global platform UI, saved, reloaded, and compared with settings and runtime feature APIs.
- **FR-3.** Environment-owned controls MUST be visibly read-only and MUST retain their effective value after a save attempt.
- **FR-4.** Anonymous, learner, instructor, and org-admin callers MUST NOT mutate platform settings.
- **FR-5.** A single-field update MUST preserve all unrelated settings and secrets.
- **FR-6.** The suite MUST restore original settings in teardown with secret-safe patch behavior.
- **FR-7.** CI MUST fail when a new registry definition lacks manifest classification.

## 6. Non-Functional Requirements

- **Performance** — use contract batches and targeted UI sampling; avoid one independent login per definition.
- **Security** — never log or round-trip masked SMTP, SAML, SES, or provider secrets.
- **Privacy & Compliance** — fixture-only settings and identities.
- **Accessibility** — controls must have label, description, switch semantics, and disabled explanation.
- **Scalability** — partition manifest by feature category and ownership source.
- **Reliability** — serialize global-setting mutations or isolate the stack per worker.
- **Observability** — failures name key, label, source, settings value, and runtime value.
- **Maintainability** — registry/manifest parity test prevents silent additions.
- **Internationalization** — prefer keys/test IDs for selection; separately verify translated labels render.
- **Backward compatibility** — accept intentionally settings-only flags when the manifest documents why no runtime field exists.

## 7. Acceptance Criteria

- **AC-1.** *Given* a database-owned platform flag, *When* a global admin toggles and saves it, *Then* settings, reload, and runtime payload agree.
- **AC-2.** *Given* an environment-owned flag, *When* the settings page loads, *Then* its control is read-only and identifies its source.
- **AC-3.** *Given* an org admin or lower role, *When* it attempts a platform mutation, *Then* access is denied and state is unchanged.
- **AC-4.** *Given* a simulated save failure, *When* a toggle is changed, *Then* the UI reports the failure and does not claim persistence.
- **AC-5.** *Given* a new platform definition without manifest metadata, *When* E2E validation runs, *Then* it fails with the missing key.

## 8. Data Model

No production schema change. Add a test manifest with key, label, category, source ownership, runtime payload key, default, prerequisites, secret sensitivity, and representative gated surface.

## 9. API Surface

- `GET/PUT /api/v1/settings/platform`
- `GET /api/v1/platform/features`
- Authentication endpoints/fixtures for the role matrix.
- Patch helpers MUST merge safely and MUST omit masked secrets.

## 10. UI / UX

Exercise Settings → Global platform search/filter, control labels, source badges, save status, loading and error states, keyboard operation, and reload. Use a small representative UI sample per category plus API-level iteration for the full registry if full UI iteration exceeds the CI budget.

## 11. AI / ML Considerations

AI-related flags assert configuration truth only. Tests must stub providers and make no billable inference calls.

## 12. Integration Points

- `clients/web/src/components/settings/platform-feature-definitions.ts`
- `clients/web/src/components/settings/platform-settings-panel.tsx`
- `clients/web/src/components/settings/platform-settings-types.ts`
- `clients/web/src/context/platform-features-context.tsx`
- `server/internal/httpserver/settings_platform.go`
- `server/internal/httpserver/platform_features.go`
- `server/internal/repos/platformconfig/`
- `e2e/fixtures/platform-features.ts`

## 13. Dependencies & Sequencing

First establish registry parity and safe snapshot/restore. Then add API contract/authz batches, followed by representative UI cases for identity, grading, learner, admin, commerce, integration, accessibility, AI, boards, quizzes, and transcripts.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Parallel specs race on global state | H | H | Dedicated serial project or isolated stack |
| PUT clears masked secrets | M | H | Snapshot only non-secret booleans and use server-supported merge semantics |
| Some flags intentionally lack runtime fields | H | M | Explicit manifest classification with rationale |
| External dependencies make “on” unusable | M | M | Contract-test flag state; stub provider readiness separately |

## 15. Rollout Plan

No product flag. Start as a non-blocking scheduled job to measure stability, fix state isolation, then make registry parity and authz blocking before enabling the sharded toggle contract in required CI.

## 16. Test Plan

- **Unit** — registry-to-manifest parity and duplicate-label/key validation.
- **Integration** — rely on Go merge/source precedence tests and add gaps discovered by E2E.
- **End-to-end** — UI save/reload, API agreement, source ownership, role matrix, failure state, restoration.
- **Security** — authorization and secret non-disclosure/non-erasure.
- **Accessibility** — switch semantics, descriptions, disabled reason, focus after save.
- **Performance / load** — total contract duration budget and per-category timing report.
- **Manual exploratory** — one pass over search, category grouping, and environment-source presentation.

## 17. Documentation & Training

Document how to add a platform flag: storage, settings type, definition, runtime payload, manifest entry, representative gate, and E2E ownership.

## 18. Open Questions

1. Should the platform settings API support PATCH to eliminate full-payload and secret-preservation risk?
2. Which settings-only booleans intentionally do not belong in `/platform/features`?
3. Should global-setting E2E run against a dedicated serial database?

## 19. References

- `docs/completed/09-platform-feature-flags-without-admin-toggle.md`
- `docs/completed/18-admin-experience/18.1-admin-console.md`
- `docs/completed/intro-course/IC01-foundation-provisioning-flag.md`
- `docs/completed/marketplace/MKT1-marketplace-platform-foundation.md`
- `docs/completed/interactive-quizzes/README.md`
- `docs/completed/visual-collaboration/README.md`
