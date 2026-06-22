# 09 — Features gated behind platform flags that have no admin UI toggle

- **Category:** Implemented but no UI/UX path to enable
- **Severity:** P3 (whole features unreachable on a default install)
- **Area:** Platform settings / admin experience (cross-cutting)
- **Status:** Fixed (2026-06-22)

## Summary

Many platform feature flags were accepted by `PUT /api/v1/settings/platform` and gated real UI
surfaces, but were missing from `PLATFORM_FEATURE_DEFINITIONS` — so Global Admins could not
enable them from Settings → Global platform.

## Fix

- Added `PlatformFeatureDefinition` entries for all confirmed flags plus triaged candidates with
  server-side `ff*` keys (`platform-feature-definitions.ts`).
- Extended `PlatformSettingsPayload` and `emptyForm()` defaults so toggles persist correctly.
- Added `platform-feature-exemptions.ts` for flags managed in dedicated panels
  (`ffStudyReminders`, `ffPlagiarismChecks`, `ffContentFilterIntegration`).
- Added `clients/web/scripts/check-platform-feature-toggles.mjs` (run as part of `npm test`) that
  parses `settings_platform.go` and fails when a new `ff*` boolean lacks a toggle or exemption.

## Acceptance criteria

- A Global Admin can enable every feature in the confirmed table from Settings → Global
  platform, and the corresponding UI/nav appears.
- A test fails if a new server-side `FF*` flag is added without a toggle or an explicit
  exemption.