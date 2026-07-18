# Lextures Playwright E2E

Run via `make e2e` (or `bash e2e/scripts/e2e-local.sh`) from the repo root. See `AGENTS.md` for stack details.

## Course feature flag matrix (E2E.1)

Course tools are covered by a data-driven matrix so every API-persisted course flag has persistence, authz, and (where applicable) settings/nav coverage.

| Artifact | Purpose |
|---|---|
| `lib/course-feature-matrix.ts` | Single registry: JSON key, UI label, shard, nav locator, off-behavior |
| `lib/course-feature-matrix-helpers.ts` | UI toggle + nav gate helpers with snapshot/restore |
| `fixtures/api.ts` → `apiPatchCourseFeatures` | **Non-destructive** PATCH (GET+merge for non-pointer bools) |
| `tests/course-features-matrix-meta.spec.ts` | Uniqueness / shard partition unit checks |
| `tests/course-features-authz.spec.ts` | Learner 403, anon 401, omit preservation, error UI, keyboard |
| `tests/course-features-ui-matrix-{a,b,c}.spec.ts` | Settings toggle → API → reload for all 24 UI rows |
| `tests/course-features-nav-matrix.spec.ts` | Nav present/absent + direct-route gates |
| `tests/course-features-api-only.spec.ts` | `groupSpacesEnabled` (no settings row yet) |

### Fixture baseline

`seededCourse` enables: notebook, feed, calendar, question bank, discussions, collab docs, group spaces, and sections. Matrix cases snapshot flags at start and restore in `finally` so a failed assertion does not poison later tests on the same course.

### Registering a new course flag

1. Add the key to `CourseFeatureKey` and a row in `COURSE_FEATURE_MATRIX` (label must match `CourseFeaturesSection`).
2. Put UI-exposed flags in shard `a` / `b` / `c` (keep shards balanced for CI file sharding).
3. If the flag produces navigation, add a `nav` block (`linkName`, `route`, `audience`, `offBehavior`).
4. If the flag is API-only, set `uiLabel: null`, `uiShard: null`, and cover it in `course-features-api-only.spec.ts` (or extend that file).
5. Extend `apiPatchCourseFeatures` typing only if you add a new key to the matrix union — the helper already iterates `ALL_COURSE_FEATURE_KEYS`.
6. Do **not** send partial PATCHes that omit the six non-pointer bools (`notebookEnabled`, `feedEnabled`, `calendarEnabled`, `questionBankEnabled`, `lockdownModeEnabled`, `discussionsEnabled`) without going through `apiPatchCourseFeatures`.

### Canvas grade sync

Canvas grade sync is stored on the Canvas link, not the course-features endpoint. Cover it only when a fixture creates a linked Canvas course (conditional case; not part of the default matrix).

## Platform feature flag contract (E2E.2)

Global platform toggles (`PLATFORM_FEATURE_DEFINITIONS`) are covered by a data-driven contract so every displayed boolean has registry parity, API persistence, authz, and representative UI coverage.

| Artifact | Purpose |
|---|---|
| `lib/platform-feature-matrix.ts` | Manifest: key, label, category, ownership, runtime key / settings-only rationale, UI sample |
| `lib/platform-feature-matrix-helpers.ts` | Snapshot/restore (secret-safe), cross-worker lock, UI/API toggle helpers |
| `fixtures/api.ts` → `apiPutPlatformSettings` / snapshot helpers | Masked PUT + boolean-only restore (never SMTP/SAML secrets) |
| `tests/platform-features-matrix-meta.spec.ts` | Registry ↔ manifest parity (fails on missing classification) |
| `tests/platform-features-authz.spec.ts` | Anon/learner/instructor/org-admin deny, omit preservation, save failure, keyboard, env read-only |
| `tests/platform-features-api-contract-{a,b,c}.spec.ts` | Full database-owned registry via API (sharded) |
| `tests/platform-features-ui-sample.spec.ts` | One Global platform UI toggle per category |

### Registering a new platform flag

1. Add the server field + `PlatformSettingsPayload` key and (usually) a `PLATFORM_FEATURE_DEFINITIONS` row.
2. Add a matching row to `PLATFORM_FEATURE_MATRIX` (label must match the definition; set `runtimeKey` or a `settingsOnlyRationale`).
3. Put exactly one `uiSample: true` per category (move the sample if you introduce a new category).
4. If the flag is environment-owned, set `ownershipSource: 'environment'` — the Global platform switch stays read-only and shows its source badge.
5. Prefer `updateMask` single-field PUTs; never round-trip masked secrets.

## Flagged feature rollback & dependencies (E2E.3)

Representative off → on → off journeys for risk-bearing flagged product families, plus parent/child dependency truth tables. Does **not** duplicate happy-path product coverage — link those specs from the manifest instead.

| Artifact | Purpose |
|---|---|
| `lib/feature-lifecycle-manifest.ts` | Families, master/child flags, dependency edges, disabled HTTP contracts, linked happy-path specs |
| `lib/feature-lifecycle-helpers.ts` | Platform+course restore, probe assertions, truth tables, data-preservation helpers |
| `tests/feature-lifecycle-meta.spec.ts` | Manifest validation + parent-cycle detection (no stack required beyond Playwright) |
| `tests/feature-lifecycle-collaboration.spec.ts` | Boards + live quizzes (Priority 1) |
| `tests/feature-lifecycle-credentials.spec.ts` | Transcripts + parent portal (Priority 1) |
| `tests/feature-lifecycle-commerce-api.spec.ts` | Payments/tax/revenue + public API/tokens (Priority 1) |
| `tests/feature-lifecycle-ai.spec.ts` | Persistent tutor / study buddy / lesson generator (Priority 1) |
| `tests/feature-lifecycle-priority2.spec.ts` | Representative Priority 2 family samples |

### Registering a new lifecycle family

1. Add a `LifecycleFamily` row (shard, masters, children, edges, probes with **exact** disabled statuses).
2. If an edge is not parent-authoritative in product code, set `parentAuthoritative: false` and document `knownGap`.
3. Prefer one representative probe per kill switch; link existing happy-path specs in `linkedHappyPathSpecs`.
4. Put Priority 1 work in the four shards above; Priority 2 samples live in `feature-lifecycle-priority2.spec.ts`.
5. Global mutations must use `withFeatureLifecycleRestore` (platform lock + boolean restore).
