# 09 — Features gated behind platform flags that have no admin UI toggle

- **Category:** Implemented but no UI/UX path to enable
- **Severity:** P3 (whole features unreachable on a default install)
- **Area:** Platform settings / admin experience (cross-cutting)

## Summary

A large set of features is fully built (backend handlers, React pages, nav entries,
background workers) but is gated behind **platform feature flags that default OFF and have
no toggle in the Settings → Global platform UI**. The settings screen only renders toggles
listed in `clients/web/src/components/settings/platform-feature-definitions.ts`; flags
missing from that array cannot be turned on through the product. The server's settings PUT
**does** accept these keys, so the only way to enable them today is a raw API call or direct
DB edit — i.e., not reachable by a Global Admin using the app.

This is squarely the "implemented but no UI/UX way to access the feature" bucket.

## Why these flags are unreachable

The toggle list is the *only* source of rendered toggles:

```ts
// clients/web/src/components/settings/platform-settings-panel.tsx:539
visiblePlatformFeatures.map((feature) => { /* renders one switch per definition */ })
// visiblePlatformFeatures is filtered from PLATFORM_FEATURE_DEFINITIONS (line 157-158)
```

The flags were migrated from env to DB **specifically to be UI-toggleable**, but the UI
rows were never added:

```sql
-- server/migrations/267_feature_flags_env_to_db.sql:1
-- Move previously env-only feature flags into platform settings so they are toggleable
-- in Settings → Global platform ...
ADD COLUMN IF NOT EXISTS ff_classroom_signals BOOLEAN,   -- nullable, no DEFAULT -> reads false
ADD COLUMN IF NOT EXISTS ff_library_integration BOOLEAN,
ADD COLUMN IF NOT EXISTS ff_reading_preferences BOOLEAN,
...
```

The server accepts all of them in the platform settings payload
(`server/internal/httpserver/settings_platform.go`, 53 `FF*` boolean fields), but
`platform-feature-definitions.ts` lists only ~47 of them.

## Confirmed: flag accepted by API + gates real UI, but no toggle

Each row below was verified to (a) be accepted by the settings PUT, (b) gate a real UI
surface, and (c) be **absent** from `PLATFORM_FEATURE_DEFINITIONS`.

| Flag (`json` key) | Gated UI surface(s) | Plan |
|---|---|---|
| `ffAcademicCalendar` | `pages/admin/academic-calendar`, dashboard upcoming-dates, iCal feed, side-nav | 14.6 |
| `ffClassroomSignals` | `components/classroom/classroom-signals-widget`, course & admin side-nav | 13.9 |
| `ffConferenceScheduling` | parent-teacher conference scheduling, main & admin side-nav | 13.12 |
| `ffBotSlack` / `ffBotTeams` / `ffBotDiscord` | classroom bots + `background/bot_reminders` worker | 16.6 |
| `ffBroadcasts` | `pages/admin/BroadcastComposer`, admin side-nav | — |
| `ffCourseEvaluations` | `pages/admin/EvaluationTemplates`, `EvaluationReport`, course nav | 14.7 |
| `ffSisIntegration` | `pages/admin/sis-integration`, admin side-nav | 16.x |
| `ffGamification` | `gamification-dashboard-card`, `LeaderboardWidget`, `MyProfile`, dashboard | — |
| `ffDemographics` | `pages/admin/title1-report`, `pages/admin/student-demographics` | 13.x |
| `ffEnrollmentStateMachine` | `pages/lms/course-enrollments`, dashboard | — |
| `ffIncompleteGradeWorkflow` | `pages/admin/incompletes`, gradebook | 14.x |
| `ffGradeSubmission` | `pages/lms/final-grade-submission`, `pages/admin/grade-submission-status` | 14.x |
| `ffCatalogIntegration` | `pages/lms/course-catalog`, main side-nav, dashboard | 16.x |
| `ffLibrary` | `reading-log-page`, `reading-dashboard-page`, `library-catalog-page` | 14.10 |
| `ffLibraryIntegration` | `pages/admin/LibraryIntegration` | 14.10 |
| `ffReadingPreferences` | reading-preferences control in `components/layout/top-bar` | 12.x |

> Verification used: `grep -rni "<flag>" clients/web/src` (real consumers, not just the
> read-only `platform-features-context.tsx`) cross-checked against the toggle definitions
> file.

### Candidates also accepted by the API but not in the toggle list (verify before acting)

`ffParentPortal`, `ffReportCards`, `ffPublicCatalog`, `ffSelfPacedMode`, `ffPublicApi`,
`ffUiMode`, `ffReadAloud`, `ffAltTextEnforcement`. These keys are accepted by the settings
PUT and are missing from the definitions array, but their UI is either wired to a
differently-named flag or reached another way — confirm whether each is enable-able through
some other surface before adding a toggle.

### Out of scope (these DO have their own settings UI — not affected)

`ffStudyReminders` (`components/settings/study-reminders-settings-panel.tsx`),
`ffPlagiarismChecks` (`pages/lms/course-settings.tsx`),
`ffContentFilterIntegration` (`pages/admin/content-filter-settings.tsx`).

## Impact

- On a fresh install every flag in the table is **off and un-toggleable from the UI**, so
  these features appear not to exist. Evaluators/self-hosters cannot discover or enable
  them without reading source and crafting an API/DB write.
- Hours of built functionality (K-12 classroom signals, conferences, academic calendar,
  course evaluations, library, gamification, SIS, grade submission, etc.) are effectively
  dark.

## Suggested fix

1. Add a `PlatformFeatureDefinition` entry (`key`, `label`, `description`, optional
   `sourceKey`) for each confirmed flag in
   `clients/web/src/components/settings/platform-feature-definitions.ts`. This is the whole
   fix for the confirmed list — the server already persists and honours them.
2. Add a lightweight test/lint that asserts **every boolean `FF*` key the server accepts in
   `settings_platform.go` is either present in `PLATFORM_FEATURE_DEFINITIONS` or explicitly
   allow-listed as "managed by a dedicated panel."** This prevents the list from drifting
   again.
3. Triage the "candidates" section: wire or remove each.

## Acceptance criteria

- A Global Admin can enable every feature in the confirmed table from Settings → Global
  platform, and the corresponding UI/nav appears.
- A test fails if a new server-side `FF*` flag is added without a toggle or an explicit
  exemption.
