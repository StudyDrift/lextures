# Content Tools authoring (developers)

How to add a first-party Content Tool after CT.1 foundations.

## Contract

> **Adding a tool = a manifest + a renderer bundle + (optionally) an action handler.
> No migration. No new table. No new route. No `Deps` change.**

The server registry loads every tool under
`server/internal/service/contenttools/tools/`. The process fails fast at startup
if any manifest violates the contract (duplicate `tool_id`, bad semver, invalid
JSON Schema, missing i18n bundle, `maxStateBytes` above the platform ceiling, or
unknown `ai.featureId`).

## Steps

1. Create `server/internal/service/contenttools/tools/<tool_id>/` with:
   - `manifest.json` — identity, semver, `configSchema` / `stateSchema` (draft 2020-12),
     scoring, storage, roles, a11y, `i18nNamespace`, `ui`
   - `i18n/en.json` — at least `name` (and usually `description`)
   - `tool.go` — `//go:embed` the manifest + i18n; export `ID`, `ManifestJSON`, `I18nEN`
2. Register the tool in `tools/index.go` `All()` (keep sorted by id).
3. Mark answer keys / secrets in `configSchema` with `"x-lex-sensitive": true`.
   The framework strips those fields for non-instructors — do not rely on the
   renderer to hide them.
4. Add a client renderer under `clients/web/src/components/content-tools/tools/`
   (CT.2 authoring UI / CT.3 student runtime). CT.2 ships the Tools dropdown and
   generic config form; student rendering lands in CT.3.
5. Run `go test ./internal/service/contenttools -run TestRegistryContract`.

## Immutability

`tool_id` is **immutable for the life of the tool**. Prefer a new id over
repurposing an existing one. Version bumps use semver in the manifest;
`tool_version` is pinned on each instance row (migration machinery arrives in CT.5).

## Course flag

Tools are inert until an instructor sets `contentToolsEnabled` on the course.
Ops may engage `CONTENT_TOOLS_KILL_SWITCH=on` to force every `/content-tools/*`
route to HTTP 404 without flipping individual courses.
