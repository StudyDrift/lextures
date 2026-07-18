# E2E.1 — Course Feature Flag Matrix

> Implementation plan. Source: completed feature audit against `course_features.go`, `course-features-section.tsx`, and `e2e/tests/features-settings.spec.ts` (2026-07-17).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | E2E.1 |
| **Section** | End-to-End Coverage |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | DONE |
| **Owner (proposed)** | Web Platform / QA |
| **Depends on** | E2E.4 |
| **Unblocks** | E2E.3 |

---

## 1. Problem Statement

The course feature API persists 25 flags and Course Settings renders 24 of them (`groupSpacesEnabled` is currently API-only), but the shared settings spec toggles only Discussion forums and Course sections. Feature-specific tests sometimes seed another flag directly, which does not prove that an instructor can operate the switch, that the value survives reload, that navigation reacts, or that unauthorized users cannot mutate it.

## 2. Goals

- Cover every row rendered by Course Settings → Features from one data-driven matrix and every API flag from a companion contract matrix.
- Verify API persistence, reload state, navigation/surface gating, and restoration.
- Prove instructor authorization and student denial for the course feature endpoint.
- Leave each seeded course in a deterministic state, even after a failed assertion.

## 3. Non-Goals

- Re-test the complete business workflow of every course tool.
- Test platform master switches; those belong to E2E.2 and E2E.3.
- Exercise Canvas grade sync unless the fixture creates a linked Canvas course.

## 4. Personas & User Stories

- **As an instructor**, I want course tools to turn on and off predictably so that course navigation matches my design.
- **As a student**, I want disabled tools hidden or unavailable so that I do not enter dead-end screens.
- **As an administrator**, I want unauthorized course members unable to alter course capabilities.

## 5. Functional Requirements

- **FR-1.** The suite MUST enumerate notebook, feed, calendar, question bank, lockdown mode, standards alignment, adaptive paths, spaced repetition, diagnostic assessments, hint scaffolding, misconception detection, sections, discussions, collaborative documents, live sessions, group spaces, office hours, AI tutor, multilingual messaging, files, attendance, whiteboard, report cards, collaboration boards, and live quizzes.
- **FR-2.** Each UI matrix case MUST toggle through the visible UI, assert `aria-checked`, poll the course API, reload, and assert the persisted value; `groupSpacesEnabled` MUST receive API persistence and runtime-gate coverage until a settings row exists.
- **FR-3.** Each navigation-producing flag MUST assert its corresponding link or route is present when enabled and absent or safely redirected when disabled.
- **FR-4.** The endpoint MUST reject learner mutation with 403 and unauthenticated mutation with 401.
- **FR-5.** PATCH requests MUST preserve omitted nullable flags and MUST NOT reset unrelated features.
- **FR-6.** The helper SHOULD restore the original flag value in teardown.

## 6. Non-Functional Requirements

- **Performance** — shard the matrix so each Playwright project remains under the existing CI timeout.
- **Security** — include instructor, learner, and unauthenticated authorization cases.
- **Privacy & Compliance** — use generated fixture identities and no production data.
- **Accessibility** — locate toggles by label/role and assert accessible names and state.
- **Scalability** — parallel-safe courses; no shared mutable global course.
- **Reliability** — poll API state rather than use fixed sleeps; teardown is idempotent.
- **Observability** — assertion messages include course code and flag key.
- **Maintainability** — define the matrix once and derive tests/helpers from it.
- **Internationalization** — stable test identifiers SHOULD replace English row text where practical.
- **Backward compatibility** — additive tests only; preserve current fixture defaults.

## 7. Acceptance Criteria

- **AC-1.** *Given* each of the 24 UI-exposed course flags is off, *When* an instructor enables its settings row, *Then* the switch, course API, and reloaded page all report on; the API-only group-spaces flag passes equivalent persistence and runtime-gate assertions.
- **AC-2.** *Given* a navigation-producing flag is on, *When* it is disabled, *Then* its learner navigation disappears and direct navigation does not expose the feature.
- **AC-3.** *Given* one flag changes, *When* the course is reloaded, *Then* all unrelated flag values remain unchanged.
- **AC-4.** *Given* a learner token, *When* it patches course features, *Then* the response is 403 and no values change.
- **AC-5.** *Given* a failed matrix case, *When* teardown runs, *Then* later specs receive the documented baseline flags.

## 8. Data Model

No production schema change. Test metadata should map JSON key, UI label, default, navigation locator, gated route, and any required platform parent flag.

## 9. API Surface

- `PATCH /api/v1/courses/{course_code}/features` — mutation and authz assertions.
- `GET /api/v1/courses/{course_code}` — persistence assertions.
- Extend `apiPatchCourseFeatures` so its type and payload support all course flags without silently supplying destructive defaults.

## 10. UI / UX

Exercise `/courses/{courseCode}/settings/features`, the course side navigation, empty/loading/error state for a failed PATCH, keyboard activation, and switch semantics. Canvas grade sync gets a separate conditional case because it is not stored by the course-features endpoint.

## 11. AI / ML Considerations

AI Tutor is tested only as a flag and gated shell; no live model request is made.

## 12. Integration Points

- `server/internal/httpserver/course_features.go`
- `server/internal/repos/course/features.go`
- `clients/web/src/pages/lms/course-features-section.tsx`
- `clients/web/src/context/course-nav-features-context.tsx`
- `e2e/fixtures/api.ts`, `e2e/fixtures/test.ts`, `e2e/tests/features-settings.spec.ts`

## 13. Dependencies & Sequencing

Create a non-destructive full-payload helper first, add API/authz tests second, then shard UI and navigation cases by fast shell-only versus feature-specific routes.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Current helper resets omitted flags | H | H | Fetch current course and merge, or send only fields with server-preserving semantics |
| Matrix makes CI slow | M | M | Reuse one course per worker and shard cases |
| Platform parent flag hides a course tool | M | H | Declare parent requirements in matrix and restore them |

## 15. Rollout Plan

No product flag. Land helper/authz coverage first, then enable matrix shards in CI. Quarantine only a single documented case, never the whole matrix; rollback is removal from the E2E project while retaining the manifest.

## 16. Test Plan

- **Unit** — validate matrix uniqueness and required metadata.
- **Integration** — existing Go repository/handler tests remain authoritative for DB writes.
- **End-to-end** — UI toggle, API persistence, reload, nav gate, direct-route gate, authz, preservation, and error state.
- **Security** — learner and anonymous PATCH denial.
- **Accessibility** — role, name, state, keyboard Space/Enter, focus retention after save.
- **Performance / load** — report per-case duration and keep shard below CI timeout.
- **Manual exploratory** — visually inspect filtered settings list and conditional Canvas row.

## 17. Documentation & Training

Update `e2e/README` or the suite checklist with the matrix, fixture baseline, and instructions for registering a new course flag.

## 18. Open Questions

1. Should disabled direct routes redirect to course home, render not-found, or return a dedicated unavailable screen consistently?
2. Should `groupSpacesEnabled` be added to the settings UI type/persist list if it remains an API-level course feature?

## 19. References

- `docs/completed/attendance.md`
- `docs/completed/01-adaptive-learning-core/1.4-adaptive-paths-across-modules.md`
- `docs/completed/01-adaptive-learning-core/1.5-spaced-repetition-retrieval-practice.md`
- `docs/completed/05-multi-tenancy-org-roles/5.4-sections.md`
- `docs/completed/06-communication-collaboration/6.6-group-spaces.md`
- `docs/completed/interactive-quizzes/IQ.1-foundation-and-feature-flag.md`
- `docs/completed/visual-collaboration/VC.1-foundation-and-feature-flag.md`
