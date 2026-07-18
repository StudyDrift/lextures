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
