# End-to-End Coverage Plans

This folder records the E2E gaps found by comparing the implemented feature registries, `docs/completed`, and the Playwright suite in `e2e/tests` on 2026-07-17.

## Audit summary

| Area | Implemented surface | Existing coverage | Planned work |
|---|---:|---|---|
| Course feature settings | 25 API-persisted course flags; 24 settings rows plus conditional Canvas grade sync (`groupSpacesEnabled` is currently API-only) | Matrix coverage in `e2e/tests/course-features-*.spec.ts` (see `e2e/README.md`) | [E2E.1](../completed/e2e/E2E.1-course-feature-flag-matrix.md) (completed) |
| Platform feature settings | 134 boolean definitions in `PLATFORM_FEATURE_DEFINITIONS` | Registry-wide UI/API/authz contract in `e2e/tests/platform-features-*.spec.ts` (see `e2e/README.md`) | [E2E.2](../completed/e2e/E2E.2-platform-feature-flag-contract.md) (completed) |
| Disabled and dependent behavior | Platform and course flags gate routes, navigation, APIs, and nested capabilities | Lifecycle + dependency truth tables in `e2e/tests/feature-lifecycle-*.spec.ts` (see `e2e/README.md`) | [E2E.3](../completed/e2e/E2E.3-flagged-feature-rollback-and-dependencies.md) (completed) |
| Completed-feature traceability | 482 documents under `docs/completed` | Spec names cover many shipped features, but there is no durable story-to-test manifest and new completed docs can silently ship without E2E review | [E2E.4](E2E.4-completed-feature-traceability.md) |

## Highest-confidence uncovered flag groups

The audit found no explicit E2E reference for these platform controls, or found a feature journey without a flag lifecycle assertion:

- Identity and grading: active sessions, annotations, blind grading, media feedback, grade posting policies, gradebook CSV, moderated grading, resubmissions, SCIM, OneRoster, MFA.
- Learner platform: learner profile and its four adaptivity flags, intro course, self-learner onboarding, gamification, course evaluations, demographics, parent portal v1/v2.
- Commerce and integrations: payment abstraction, Redis cache, ePortfolio, proctoring, feedback, SES, Zapier flag lifecycle, public API kill switch.
- New nested products: visual boards and its realtime/external-sharing children; interactive quizzes and its hosting, modes, homework, gradebook, catalog, guest, and AI children; transcript inbound; motion navigation.
- Course tools: notebook, feed, calendar, question bank, lockdown mode, standards alignment, adaptive paths, spaced repetition, diagnostics, hint scaffolding, misconception detection, collaborative documents, live sessions, group spaces, office hours, AI tutor, multilingual messaging, files, attendance, whiteboard, report cards, collaboration boards, and live quizzes. Only discussions and sections currently have direct settings-toggle coverage; group spaces has no settings row despite being accepted by the API.

“No explicit reference” is a prioritization signal, not proof that the feature has no happy-path test. For example, `parent-portal.spec.ts` and `public-api.spec.ts` exercise their products but do not establish a complete admin-toggle → runtime-off → runtime-on contract. E2E.2 and E2E.3 close that distinction.

## Sequencing

1. E2E.4 establishes the manifest and coverage vocabulary.
2. E2E.1 adds the course matrix and reusable course-flag helper.
3. E2E.2 adds registry-wide platform control coverage.
4. E2E.3 adds representative runtime rollback and dependency journeys, prioritized by risk (completed).
