# Build a Content Tool

CT.5 developer guide for first-party tools using `@lextures/tool-sdk`.

## Contract

1. Add a folder under `server/internal/service/contenttools/tools/{tool_id}/` with `manifest.json` and `i18n/en.json`.
2. Register it in `tools/index.go`.
3. Implement a renderer against `@lextures/tool-sdk` (`defineTool`, `ToolProps`, hooks, UI primitives).
4. Optionally register server actions with `RegisterActionHandler`.

## Versioning

- Semver on `manifest.version`.
- Additive optional schema fields → **minor**.
- Removed / required / narrowed fields → **major**.
- Docs/UI-only → **patch**.
- CI: `npm run tool:schema-diff` (from `clients/web`) fails when the bump is too weak.

## Migrations

Declare ordered pure migrations in the Go `MigrationRegistry` (keyed by source schema version). Lazy migration runs on read; eager backfill via `POST /api/v1/admin/content-tools/migrations` (dry-run first). Failures quarantine the original document.

## Sandbox

- `sandbox: "inprocess"` (default) mounts in the LMS SPA.
- `sandbox: "iframe"` mounts under `sandbox="allow-scripts"` (no `allow-same-origin`) and speaks the versioned `postMessage` bridge.
- Platform flag `CONTENT_TOOLS_SANDBOX_MODE=off|optin|required`.

## Budgets

- Renderer chunk ≤ 40 KB gzip (`npm run bundle:check`).
- `storage.maxStateBytes` enforced at runtime.
- Action rate limits from the manifest; circuit breaker auto-disables a failing tool.
