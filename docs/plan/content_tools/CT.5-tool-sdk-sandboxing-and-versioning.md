# CT.5 — Content Tools: Tool SDK, Sandboxing, Versioning & State Migration

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.5 |
| **Section** | Content Tools (CT) |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Platform / ecosystem team |
| **Depends on** | CT.1, CT.3 |
| **Unblocks** | CT.9 (marketplace), safe long-term evolution of every tool |

---

## 1. Problem Statement

CT.1–CT.3 make one tool possible; they do not make **hundreds** of tools survivable. Three things
break at scale: (1) every tool author reinvents the same plumbing and drifts from the contract;
(2) a tool that changes its config or state shape silently corrupts thousands of stored documents,
because `state_json` is schemaless at the database level; (3) code we did not write — the entire
premise of a marketplace — would run on our origin with access to the learner's session. This story
ships the **SDK** that makes tools uniform, the **sandbox** that makes untrusted tools safe, and the
**versioning + migration** machinery that lets a tool evolve without breaking stored work.

## 2. Goals

- Publish `@lextures/tool-sdk`: the typed runtime contract, hooks, UI primitives and test harness a
  tool author builds against — one import, no framework archaeology.
- Define semver semantics for tools and enforce them: what is a patch, a minor, and a breaking change
  in terms of `configSchema` and `stateSchema`.
- Ship **state and config migrations**: a tool declares ordered migration functions; the framework
  applies them lazily on read and eagerly in a backfill job, never losing a document.
- Ship the **iframe sandbox** and `postMessage` bridge so a third-party tool can never touch the
  session, the DOM of the host, or any API beyond its own instance.
- Enforce per-tool budgets (bundle size, state size, action rate, AI spend) mechanically in CI and at
  runtime, so one bad tool cannot degrade the platform.

## 3. Non-Goals

- The marketplace itself — listing, review, install, revenue (CT.9).
- Server-side third-party code execution: tools never run our Go process. A third-party tool that
  needs a backend calls its own, from its own origin, under CSP allowlist (CT.9).
- Rewriting first-party tools to run sandboxed; first-party tools may mount in-process for performance
  and are held to the same contract by CI instead of by isolation.
- Visual design system work — the SDK re-exports existing primitives, it does not create a new one.

## 4. Personas & User Stories

- **As a tool author (internal)**, I want a typed SDK and a local harness so that I can build and test
  a tool without running the whole LMS.
- **As a tool author**, I want to change my state shape in v2 and have v1 documents migrate
  automatically so that shipping an improvement does not mean abandoning existing classes.
- **As a platform engineer**, I want a contract test suite that every tool must pass so that a
  malformed tool fails CI rather than a classroom.
- **As a security engineer**, I want third-party tools to run in an isolated origin with an explicit
  capability list so that installing a tool cannot exfiltrate a student's session.
- **As an SRE**, I want per-tool budgets and circuit breakers so that one popular, badly-written tool
  cannot take out the content page for everyone.
- **As an instructor**, I want a tool to keep working after the platform upgrades so that my course
  does not break between terms.

## 5. Functional Requirements

- **FR-1.** The system MUST publish `@lextures/tool-sdk` exporting: `defineTool(manifest, renderer)`,
  the `ToolProps` contract type, `useToolState`, `useToolAction`, `useToolAnnounce`, `useToolI18n`,
  and UI primitives (`ToolShell`, `ToolPrompt`, `ToolActions`, `ToolFeedback`, `ToolScore`).
- **FR-2.** The SDK MUST ship a **test harness** (`renderTool()`) that mounts a tool against an
  in-memory state store and a mock action dispatcher, so a tool's tests never need a server.
- **FR-3.** Tool versions MUST be semver. The framework MUST classify a version bump by diffing
  schemas: additive optional field = **minor**; new required field, removed field, narrowed type, or
  changed enum member = **major**; documentation/UI-only = **patch**.
- **FR-4.** CI MUST fail when a manifest's schema diff is inconsistent with its version bump.
- **FR-5.** A tool MUST be able to declare ordered `stateMigrations` and `configMigrations` keyed by
  source schema version, each a pure function `(doc) => doc`.
- **FR-6.** On read, a stored document whose `state_schema_version` is lower than the current version
  MUST be migrated in memory, served migrated, and persisted on the next write (lazy migration).
- **FR-7.** A backfill job MUST be able to eagerly migrate all documents for a tool, in batches, with
  progress, resumability and a dry-run that reports how many documents fail migration.
- **FR-8.** A document that fails migration MUST be quarantined (original preserved, tool renders a
  labelled recovery state) and MUST NOT be dropped or partially written.
- **FR-9.** Instances pin `tool_version`; the framework MUST resolve an instance to the newest
  compatible version within the pinned major, and MUST NOT auto-upgrade across a major.
- **FR-10.** The framework MUST support a `sandbox: 'iframe'` manifest mode rendering the tool in a
  cross-origin iframe with `sandbox="allow-scripts"` (no `allow-same-origin`) served from a distinct
  tool origin, with a strict CSP and no ambient credentials.
- **FR-11.** Sandboxed tools MUST communicate only over a versioned `postMessage` protocol:
  `init`, `stateChanged`, `save`, `runAction`, `resize`, `announce`, `error` — origin-checked,
  schema-validated, and rate-limited on the host side.
- **FR-12.** A sandboxed tool MUST NOT receive the session cookie, an API token, the user's email, or
  any identifier beyond an opaque per-instance participant id.
- **FR-13.** All network access from a sandboxed tool MUST be denied by CSP except hosts declared in
  `manifest.network.allowedHosts` and approved at install time (CT.9).
- **FR-14.** The framework MUST enforce budgets: renderer chunk ≤ 40 KB gz (CI), state ≤
  `maxStateBytes` (runtime), actions ≤ manifest rate (runtime), AI spend ≤ per-course tool budget
  (runtime, CT.6/CT.8).
- **FR-15.** The framework MUST implement a per-tool **circuit breaker**: when a tool's client render
  errors or action failures exceed a threshold, it is auto-disabled platform-wide with an alert, and
  renders a maintenance placeholder rather than failing.
- **FR-16.** A `deprecated` manifest flag MUST hide a tool from the palette while keeping existing
  instances rendering; a `sunset` date MUST warn authors in the editor.
- **FR-17.** The SDK MUST provide a `contract` version constant; the host MUST refuse to mount a tool
  declaring an unsupported contract range, rendering an "update required" placeholder.

## 6. Non-Functional Requirements

- **Performance** — Sandbox adds ≤ 60 ms to first paint per iframe tool; iframes are created lazily on
  viewport entry. Lazy migration adds ≤ 2 ms per document. Host `postMessage` handling stays off the
  main-thread critical path for bursts (batched per animation frame).
- **Security** — Threat model documented: hostile tool, compromised tool CDN, malicious course author,
  malicious student. Controls: separate origin, no `allow-same-origin`, CSP `frame-src`/`connect-src`
  allowlists, no credentialed requests, opaque ids, message schema validation, size and rate caps on
  the bridge, and Subresource Integrity for third-party bundles.
- **Privacy & Compliance** — A sandboxed tool sees only what its manifest declares and the instructor
  configured; the data-flow inventory per tool feeds the RoPA (S05) and sub-processor disclosure (S07)
  when the tool is third-party.
- **Accessibility** — The SDK primitives are WCAG 2.1 AA by construction (labelled controls, focus
  management, live-region wiring). Iframe tools MUST expose `title`, participate in the host's focus
  order, and be sized dynamically to avoid inner scrollbars. The SDK harness runs axe by default so a
  tool fails its own tests on a violation.
- **Scalability** — Registry, budgets and breakers are O(1) lookups. Migration backfill runs on the
  existing job queue at ≥ 2,000 docs/s per worker.
- **Reliability** — Breakers fail closed to a placeholder, never to a broken page. Migration is
  idempotent and resumable; quarantine guarantees no data loss.
- **Observability** — `lextures_content_tool_bridge_messages_total{tool_id,type,outcome}`,
  `…_migration_docs_total{tool_id,from,to,outcome}`, `…_breaker_state{tool_id}`,
  `…_bundle_bytes{tool_id}`. Alert on breaker open and on any quarantine.
- **Maintainability** — One SDK version supported at a time plus the previous major (12-month window).
  The contract test suite is the source of truth and runs against every registered tool in CI.
- **Internationalization** — SDK i18n helper resolves the tool namespace; sandboxed tools receive
  locale and direction at `init` and must not detect it themselves.
- **Backward compatibility** — Contract changes are additive within a major; deprecations warn for one
  minor before removal; instances never auto-cross a major.

## 7. Acceptance Criteria

- **AC-1.** *Given* a tool built only against `@lextures/tool-sdk`, *When* its tests run with the
  harness, *Then* they pass without a server or database.
- **AC-2.** *Given* a manifest that removes a required config field with only a minor bump, *When* CI
  runs, *Then* the schema-diff check fails with the offending path named.
- **AC-3.** *Given* a stored v1 state document and a tool now at v2 with a migration, *When* the
  learner loads the page, *Then* the migrated document renders and the next write persists
  `state_schema_version = 2`.
- **AC-4.** *Given* a document that throws during migration, *When* it is read, *Then* the original is
  untouched, the tool shows a recovery placeholder, and a quarantine metric increments.
- **AC-5.** *Given* an eager backfill dry-run, *When* it completes, *Then* it reports counts of
  migratable and failing documents and mutates nothing.
- **AC-6.** *Given* a sandboxed tool, *When* it attempts `document.cookie`, a fetch to an unapproved
  host, or access to `window.parent`, *Then* each attempt fails and the page is unaffected.
- **AC-7.** *Given* a sandboxed tool sends a malformed bridge message, *Then* the host drops it,
  increments an error counter, and the tool keeps running.
- **AC-8.** *Given* a tool's render errors exceed the breaker threshold, *When* the next page loads,
  *Then* the tool renders the maintenance placeholder and an alert has fired.
- **AC-9.** *Given* a tool bundle exceeds 40 KB gz, *When* CI runs, *Then* the build fails naming the
  tool and its size.
- **AC-10.** *Given* an instance pinned to `1.4.0` and a published `2.0.0`, *When* it renders, *Then*
  it resolves to the newest `1.x` and never to `2.0.0`.
- **AC-11.** *Given* a deprecated tool, *When* an author opens the palette, *Then* it is absent, while
  an existing instance still renders and is still resettable.

## 8. Data Model

Migration `server/migrations/453_content_tool_versions.sql` (+ `.down.sql`).

```sql
-- 453_content_tool_versions.sql

-- Registry mirror: one row per (tool_id, version) actually seen by this deployment.
CREATE TABLE IF NOT EXISTS course.content_tool_versions (
    tool_id           TEXT NOT NULL,
    version           TEXT NOT NULL,
    manifest_json     JSONB NOT NULL,
    config_schema_version INTEGER NOT NULL DEFAULT 1,
    state_schema_version  INTEGER NOT NULL DEFAULT 1,
    sandbox_mode      TEXT NOT NULL DEFAULT 'inprocess'
                        CHECK (sandbox_mode IN ('inprocess','iframe')),
    status            TEXT NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active','deprecated','sunset','disabled')),
    breaker_open_at   TIMESTAMPTZ,
    sunset_at         TIMESTAMPTZ,
    first_seen_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tool_id, version)
);
CREATE INDEX IF NOT EXISTS idx_ctv_tool_status ON course.content_tool_versions (tool_id, status);

-- Documents that failed migration; the original is preserved verbatim.
CREATE TABLE IF NOT EXISTS course.content_tool_state_quarantine (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    state_id       UUID NOT NULL REFERENCES course.content_tool_states (id) ON DELETE CASCADE,
    tool_id        TEXT NOT NULL,
    from_version   INTEGER NOT NULL,
    to_version     INTEGER NOT NULL,
    error          TEXT NOT NULL,
    original_json  JSONB NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at    TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_ctsq_tool ON course.content_tool_state_quarantine (tool_id, resolved_at);

-- Eager migration jobs.
CREATE TABLE IF NOT EXISTS course.content_tool_migration_jobs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tool_id        TEXT NOT NULL,
    from_version   INTEGER NOT NULL,
    to_version     INTEGER NOT NULL,
    dry_run        BOOLEAN NOT NULL DEFAULT TRUE,
    status         TEXT NOT NULL DEFAULT 'queued'
                     CHECK (status IN ('queued','running','succeeded','failed','cancelled')),
    total_docs     INTEGER NOT NULL DEFAULT 0,
    migrated_docs  INTEGER NOT NULL DEFAULT 0,
    failed_docs    INTEGER NOT NULL DEFAULT 0,
    cursor_state_id UUID,
    error          TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at    TIMESTAMPTZ
);
```

**Backfill** — on first boot after deploy, the registry mirror is populated from the in-process
registry; no learner data is touched.

## 9. API Surface

Platform-admin scoped (not course scoped), under `/api/v1/admin/content-tools`:

| Verb | Path | Auth scope |
|---|---|---|
| `GET` | `/api/v1/admin/content-tools/versions` | platform admin |
| `PATCH` | `/api/v1/admin/content-tools/versions/{tool_id}/{version}` (status, breaker reset) | platform admin |
| `POST` | `/api/v1/admin/content-tools/migrations` (dry-run or execute) | platform admin |
| `GET` | `/api/v1/admin/content-tools/migrations/{job_id}` | platform admin |
| `GET` | `/api/v1/admin/content-tools/quarantine?toolId=` | platform admin |

**Bridge protocol** (not HTTP — `postMessage`, versioned):

```ts
type BridgeToTool =
  | { t: 'init'; v: 1; instanceId: string; config: unknown; state: unknown; revision: number
      locale: string; dir: 'ltr' | 'rtl'; readOnly: boolean; participantId: string }
  | { t: 'stateAccepted'; v: 1; revision: number }
  | { t: 'actionResult'; v: 1; id: string; result: unknown }
  | { t: 'error'; v: 1; id?: string; code: string; message: string }

type BridgeFromTool =
  | { t: 'ready'; v: 1; contract: string }
  | { t: 'save'; v: 1; state: unknown; revision: number }
  | { t: 'runAction'; v: 1; id: string; action: string; input: unknown }
  | { t: 'resize'; v: 1; height: number }
  | { t: 'announce'; v: 1; message: string; assertive?: boolean }
```

Host-side caps: ≤ 20 messages/s per iframe, ≤ 64 KB per message, unknown `t` dropped, origin pinned.

## 10. UI / UX

- **Developer-facing** — the SDK's local harness (`npm run tool:dev`) renders a tool against fixture
  config/state with panels for state inspection, action mocking, locale/RTL switching and an axe report.
- **Admin-facing** — *Admin → Platform → Content Tools*: table of tool versions with status, instance
  count, breaker state, bundle size and a11y declaration; actions to deprecate, disable, reset a
  breaker, and launch a migration (dry-run first, results shown inline).
- **Author-facing** — a deprecated tool shows an inline notice in the editor card ("This tool will be
  retired on {date}; existing student work is unaffected"); an iframe tool shows a "runs in a secure
  sandbox" badge with a link to what that means.
- **Student-facing** — maintenance placeholder ("This activity is temporarily unavailable") and
  recovery placeholder ("We couldn't load your previous work — your teacher can restore it") for
  quarantined documents.

**States** — loading skeleton inside the iframe frame until `ready`; timeout at 10 s → error card with
retry. **Mobile** — iframes sized by `resize` messages; no nested scrolling. **A11y** — iframe carries
`title`, is reachable in tab order, and the host relays `announce` into its shared live region.
**Copy & i18n** — `contentTools.sdk.*`, `contentTools.admin.*`.

## 11. AI / ML Considerations

The SDK exposes no model access directly: an AI tool calls `runAction`, the server calls `aigateway`,
and the sandbox never holds a key. Third-party tools MAY call their own models from their own backend
(disclosed at install, CT.9), in which case the platform's disclosure surface labels the tool as using
an external AI service and the org's AI policy applies to *installing* it, not to each call. Per-tool
AI budgets are enforced in the action dispatcher (CT.6/CT.8), not in tool code.

## 12. Integration Points

- **Internal** — `clients/web/src/components/content-tools/host/*` (mount modes), new package
  `clients/packages/tool-sdk/`, `server/internal/service/contenttools/registry.go` and
  `migrations.go`, `httpserver/admin_content_tools.go`, `internal/background` (migration jobs),
  `internal/telemetry`.
- **Build** — Vite config for the tool origin bundle; CI job for bundle budgets and schema-diff checks;
  SRI hashes for third-party bundles.
- **Infra** — a separate tool-serving origin (e.g. `tools.<domain>`) with its own CSP; documented in
  `deploy/` and `iac/`.
- **CT.9** — install-time host approval writes the CSP allowlist this story enforces.

## 13. Dependencies & Sequencing

- **Must ship after:** CT.1 (registry/manifest), CT.3 (runtime contract).
- **Must ship before:** CT.9 (marketplace cannot ship without isolation), and before any tool ships a
  breaking schema change.
- **Shared infra needed:** second origin/host, CDN for tool bundles, job queue.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Sandbox escape via bridge message handling | L | H | Schema-validated, origin-pinned, size/rate-capped messages; no `allow-same-origin`; pen-test before CT.9 |
| Migration corrupts stored work | M | H | Pure functions, dry-run, quarantine-on-failure, lazy-then-eager order, snapshot before eager backfill |
| SDK becomes a second design system that drifts | M | M | SDK re-exports existing primitives; visual regression tests shared with the web app |
| Iframe tools feel slow or janky compared to first-party | M | M | Lazy iframe creation, `resize` protocol, skeletons, budget on time-to-`ready` |
| Version pinning strands courses on old majors | M | M | Deprecation notices with sunset dates, author-facing upgrade prompt, admin migration tooling |
| Breaker auto-disables a tool mid-class | L | M | Threshold tuned high, alert to on-call, one-click reset, placeholder explains rather than blanks |

## 15. Rollout Plan

- **Feature flag** — `CONTENT_TOOLS_SANDBOX_MODE` (`off|optin|required`) plus per-tool manifest mode.
  Ships `optin`; CT.9 requires `required` for third-party tools.
- **Sequencing** — SDK package → contract tests over existing tools → migration engine (lazy → eager)
  → registry mirror + admin UI → tool origin + CSP → first tool converted to iframe mode as a canary.
- **Dogfood** — convert CT.16 (Parameter Explorer) to sandbox mode first: rich, self-contained, low
  data sensitivity.
- **GA criteria** — pen-test findings closed; migration engine proven on ≥ 10,000 documents; bundle
  budgets green for every registered tool.
- **Rollback** — set sandbox mode `off` (first-party tools unaffected); disable a specific version via
  the registry mirror without a deploy.

## 16. Test Plan

- **Unit** — semver classification from schema diffs; migration chaining and idempotency; quarantine
  path; bridge message validation (malformed, oversized, wrong origin, unknown type); breaker state
  machine; version resolution within a major.
- **Integration** — lazy migration on read + persistence on write; eager backfill with resume after a
  simulated crash; registry mirror sync; admin routes authz.
- **End-to-end** — Playwright: iframe tool completes a full interact→save→action→score loop; hostile
  fixture tool attempting cookie access, storage access, parent DOM access, disallowed fetch, message
  flooding — all contained.
- **Security** — dedicated sandbox threat-model test suite plus an external pen-test before CT.9;
  CSP report-only monitoring in staging; SRI mismatch handling.
- **Accessibility** — axe inside the iframe via the harness; focus traversal into and out of iframe
  tools; announcement relay verified with a screen reader.
- **Performance / load** — 10 iframe tools on one page within the page budget; migration throughput
  benchmark; bridge burst handling at 200 msg/s (dropped politely, no jank).
- **Manual exploratory** — offline behaviour of iframe tools; browser extension interference; Safari
  and Firefox iframe storage partitioning.

## 17. Documentation & Training

- **Developer** — "Build a Content Tool" guide: manifest, renderer, tests, budgets, a11y checklist,
  versioning rules, migration authoring, and the sandbox capability list. This is the document CT.9's
  external developers read.
- **Admin** — deprecating a tool, running a migration, resetting a breaker.
- **API reference** — admin routes + the bridge protocol specification.
- **Runbook** — breaker response, quarantine triage, rolling back a tool version.

## 18. Open Questions

1. Should first-party tools eventually be *required* to run sandboxed for uniformity, accepting the
   performance cost? Proposed: no — keep both modes, hold both to the same contract tests.
2. Is 40 KB gz the right bundle budget, or should it scale by category (a simulation legitimately
   needs more)? Proposed: 40 KB default with a documented exception process recorded in the manifest.
3. Should migrations be allowed to call the network (e.g. re-grade with AI)? Proposed: no — migrations
   are pure; anything else is a job.
4. Where do sandboxed tool bundles live — our CDN with SRI, or the developer's origin? Proposed: our
   CDN, because revocation and integrity are ours to guarantee. Confirm with CT.9.

## 19. References

- Existing files this work touches: `clients/web/src/components/content-tools/host/*`,
  `server/internal/service/contenttools/`, `server/internal/httpserver/admin_*.go`,
  `deploy/`, `iac/`, `server/migrations/453_content_tool_versions.sql`.
- Precedents followed: sandboxed vibe-activity iframes (`course.module_vibe_activities`, `srcDoc` +
  sandbox attributes); H5P asset isolation (`content.h5p_packages.assets_prefix`).
- External standards: HTML `iframe` sandbox semantics; CSP Level 3; Subresource Integrity;
  SemVer 2.0.0; JSON Schema draft 2020-12; OWASP ASVS V14 (configuration).
- Related plans: [CT.1](CT.1-foundations-registry-and-data-model.md),
  [CT.3](CT.3-student-runtime-and-state-persistence.md),
  [CT.9](CT.9-tool-marketplace-and-third-party-tools.md),
  [16.9 marketplace / plugin system](../../completed/16.9-marketplace-plugin-system.md).
