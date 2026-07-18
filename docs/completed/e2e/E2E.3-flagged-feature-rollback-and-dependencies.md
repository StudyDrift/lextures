# E2E.3 — Flagged Feature Rollback and Dependency Journeys

> Implementation plan. Source: disabled-state and nested-flag audit of completed features and Playwright specs (2026-07-17).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | E2E.3 |
| **Section** | End-to-End Coverage |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Owner (proposed)** | Feature teams / QA |
| **Depends on** | E2E.1, E2E.2 |
| **Unblocks** | Safe staged rollout of completed features |

---

## 1. Problem Statement

Many completed features have a happy-path E2E spec but no proof that their operational kill switch removes navigation, blocks direct routes and APIs, preserves data, and restores functionality when re-enabled. Nested products such as collaboration boards, interactive quizzes, parent portal v2, transcripts, and AI capabilities also require parent/child flag combinations that are currently easy to regress.

## 2. Goals

- Add representative off → on → off journeys for every risk-bearing flagged product family.
- Verify parent/child flag dependency truth tables.
- Standardize disabled responses and user-facing fallback behavior.
- Prove rollback is non-destructive and re-enable restores prior data.

## 3. Non-Goals

- Duplicate all happy-path assertions already owned by feature specs.
- Exercise destructive production rollback or real external-provider outages.
- Treat compile-time/mobile-only switches as web Playwright controls.

## 4. Personas & User Stories

- **As an operator**, I want a kill switch to stop a faulty feature immediately without deleting data.
- **As a learner**, I want disabled features to disappear cleanly rather than lead to broken pages.
- **As an instructor**, I want prior configuration restored when a feature returns.
- **As a compliance admin**, I want access-control checks to precede feature disclosure where required.

## 5. Functional Requirements

- **FR-1.** Each family MUST cover navigation, direct web route, authenticated API, unauthenticated API, persisted data, and re-enable behavior where applicable.
- **FR-2.** Nested families MUST test parent off/child off, parent on/child off, parent off/child on, and parent on/child on.
- **FR-3.** Priority 1 families MUST include boards (`ffVisualBoards`, course flag, realtime, external sharing), interactive quizzes (course flag plus hosting and eight child controls), transcripts/inbound, parent portal v1/v2, public API/API tokens, payments/billing/tax/revenue share, and AI tutor/study-buddy/lesson generation.
- **FR-4.** Priority 2 families MUST include identity/provisioning, grading workflows, learner profile/adaptivity, intro/onboarding, proctoring, feedback, SES, motion, gamification, evaluations, demographics, and report cards.
- **FR-5.** Disabled behavior MUST follow the documented contract (404, 403, 501, 503, redirect, or hidden UI) and MUST NOT be accepted as an arbitrary set of statuses.
- **FR-6.** Re-enabling a flag MUST reveal previously created non-destructive data.

## 6. Non-Functional Requirements

- **Performance** — one lifecycle journey per family, not per endpoint.
- **Security** — verify auth before feature-state disclosure where documented; do not weaken 401/403 expectations.
- **Privacy & Compliance** — explicitly cover parent/minor, research consent, transcript, and accommodations surfaces.
- **Accessibility** — disabled notices and redirects preserve focus and announce state.
- **Scalability** — family specs own isolated fixtures and can shard.
- **Reliability** — restore all global/course flags in `finally`-equivalent teardown.
- **Observability** — assert audit events for admin flag changes where supported.
- **Maintainability** — shared lifecycle helper with family-owned assertions.
- **Internationalization** — fallback/disabled copy uses translation keys.
- **Backward compatibility** — data remains readable after re-enable.

## 7. Acceptance Criteria

- **AC-1.** *Given* a populated flagged feature, *When* its platform master flag is disabled, *Then* navigation and direct access follow the documented disabled contract and stored data remains intact.
- **AC-2.** *Given* a parent flag is off and child flag is on, *When* a user visits the child surface, *Then* the parent remains authoritative.
- **AC-3.** *Given* a course flag is off while the platform flag is on, *When* a learner visits that course tool, *Then* it is unavailable only in that course.
- **AC-4.** *Given* a feature is disabled and then re-enabled, *When* the original user returns, *Then* prior data and permissions are restored.
- **AC-5.** *Given* an unauthenticated request, *When* a protected flag is off, *Then* the response does not leak more feature state than its documented auth-first contract allows.

## 8. Data Model

No production schema change. Fixtures need stable records for boards/posts, quiz kits/games, transcript orders, parent links, payment entitlements, AI sessions, and representative grading/configuration data before rollback.

## 9. API Surface

Use settings and course feature mutation APIs plus each family's representative list/create/read endpoint. Record the expected disabled status per route in the E2E manifest; inconsistent existing contracts become explicit product bugs rather than permissive assertions.

## 10. UI / UX

Assert side-nav and admin-nav removal, direct URL behavior, a comprehensible unavailable state where applicable, no stale controls after runtime refresh, and restored screens after re-enable without requiring logout unless the product explicitly caches by session.

## 11. AI / ML Considerations

AI family tests stub inference and verify that disabled flags prevent calls. Re-enable uses deterministic fake responses and asserts no PII enters recordings.

## 12. Integration Points

- Platform and course feature contexts and settings panels.
- Router/navigation registries in `clients/web/src`.
- Feature-gated handlers in `server/internal/httpserver`.
- Existing family specs such as `parent-portal.spec.ts`, `public-api.spec.ts`, `billing.spec.ts`, `lesson-generator.spec.ts`, `course-marketplace-*.spec.ts`, `transcripts*.spec.ts`, `visual-collaboration`, and `interactive-quizzes` implementations.

## 13. Dependencies & Sequencing

Implement Priority 1 in four shards: collaboration, credentials/transcripts, commerce/API, and AI. Then implement Priority 2 by product ownership. Resolve disabled-contract inconsistencies before making each shard required.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Existing endpoints intentionally use different statuses | H | M | Document per-route contract and file normalization follow-ups |
| Global flag mutation contaminates parallel tests | H | H | Serial global lifecycle project and robust restore |
| External services unavailable | M | H | Stub at provider boundary and assert no call while off |
| Very large scope duplicates feature specs | M | M | One representative lifecycle per family; link existing happy paths |

## 15. Rollout Plan

No product flag. Add family shards incrementally, initially scheduled, then required once isolation is proven. Roll back a flaky family shard independently while retaining manifest coverage and an issue owner.

## 16. Test Plan

- **Unit** — dependency manifest validation and parent-cycle detection.
- **Integration** — handler-level disabled status and data-preservation checks.
- **End-to-end** — representative family lifecycle and dependency truth tables.
- **Security** — auth-first ordering, direct-route denial, lower-role settings denial.
- **Accessibility** — focus and announcements for removed/disabled surfaces.
- **Performance / load** — confirm disabling expensive features prevents provider/job invocation.
- **Manual exploratory** — operator kill-switch drill for one family per release.

## 17. Documentation & Training

Add expected off-state and rollback behavior to each feature runbook. Document the family-owner rule: a flagged feature is not E2E-complete until one lifecycle test is linked.

## 18. Open Questions

1. Which disabled HTTP status should be the platform standard for authenticated feature gates?
2. Which flag changes must propagate live versus on reload or next login?
3. Should flag mutation always emit a dedicated audit event?

## 19. References

- `docs/completed/marketplace/MKT1-marketplace-platform-foundation.md`
- `docs/completed/interactive-quizzes/README.md`
- `docs/completed/visual-collaboration/README.md`
- `docs/completed/transcripts/README.md`
- `docs/completed/ai-providers/AP.4-migrate-call-sites.md`
- `docs/completed/13-k12-specific/`
- `docs/completed/14-higher-ed-specific/`
- `docs/completed/15-self-learner-specific/`

## 20. Implementation notes

Delivered under `e2e/`:

- Manifest + cycle detection: `lib/feature-lifecycle-manifest.ts`, `tests/feature-lifecycle-meta.spec.ts`
- Lifecycle helpers (platform lock/restore + course restore): `lib/feature-lifecycle-helpers.ts`
- Priority 1 shards: `feature-lifecycle-collaboration|credentials|commerce-api|ai.spec.ts`
- Priority 2 samples: `feature-lifecycle-priority2.spec.ts`
- Operator guide: `e2e/README.md` (E2E.3 section)

Known product gaps are recorded on dependency edges (`knownGap`) rather than asserted permissively — notably parent portal API is account-type gated (not `ffParentPortal`), and `ffVisualBoards` is always-on in merge (course flag is the kill switch).

