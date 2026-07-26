# CT.1 — Content Tools: Foundations, Tool Registry, Manifest Contract & Data Model

> Implementation plan. Source: new capability — interactive tools inside content sections. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | CT.1 |
| **Section** | Content Tools (CT) |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | — (builds on shipped course structure, enrollments, RBAC) |
| **Unblocks** | CT.2, CT.3, CT.4, CT.5, CT.6, CT.7, CT.8, CT.9 and every tool story CT.10–CT.23 |

---

## 1. Problem Statement

A Lextures section body is inert prose: a student can scroll a whole content page, assignment
description or syllabus section without producing one piece of evidence that they engaged with it.
Every interactive surface the platform has (quiz, assignment, H5P package, vibe activity, board) is
a *whole module item* the learner must navigate away to, which is far too heavy for a 20-second
comprehension check placed exactly where the confusion happens. There is no schema for "an
interactive widget placed inside a body of content", no contract describing what such a widget may
configure or store, and no per-enrollment place to keep its state — so no tool can be built. This
story lays the backbone: the per-course flag, the **tool manifest contract**, the registry, the
three core tables, and the configuration API, with nothing yet rendering or persisting learner work.

## 2. Goals

- Ship a **per-course** `content_tools_enabled` flag wired through the identical path as
  `adaptive_content_enabled`, with **no** required global platform on-switch.
- Define the **tool manifest** — the single contract every tool (first-party or third-party) obeys:
  identity, semver, capability declarations, JSON Schemas for config and state, scoring mode, AI
  usage, network needs, storage budget and accessibility statement.
- Define the core relational model — **instances** (a placed tool + its config), **states** (one
  JSONB document per `(instance, enrollment)`), and an append-only **event** log — additive
  migrations that change no existing table's semantics.
- Expose an instructor-gated instance CRUD + course-level tool-allowlist API.
- Guarantee that adding the 200th tool requires **zero** schema, route or `Deps` changes.
- Guarantee the feature is inert for every existing course and every course that leaves the flag off.

## 3. Non-Goals

- Any UI: the Tools dropdown and config forms are CT.2; the student renderer is CT.3.
- Persisting or validating *learner* state at runtime (CT.3 owns the write path; CT.1 only defines
  the table and the schema contract).
- Reset workflows (CT.4), the third-party SDK and iframe sandbox (CT.5), AI grounding (CT.6),
  analytics (CT.7), governance sign-off (CT.8), marketplace distribution (CT.9).
- Shipping any actual tool — CT.1 registers exactly one built-in `noop_probe` tool used only by tests.

## 4. Personas & User Stories

- **As an instructor**, I want to turn Content Tools on for one course so that I can experiment
  without my colleagues' courses changing.
- **As an instructor**, I want to restrict which tools are available in my course so that a
  code sandbox does not appear in a 4th-grade reading unit.
- **As a platform engineer**, I want a single declarative manifest per tool so that adding a tool is
  a pull request against one folder, not a migration plus five wiring points.
- **As an org admin**, I want tool availability governed by org policy so that a tool that sends text
  to a model is unavailable to districts that disabled AI features.
- **As a security engineer**, I want the answer key of a tool's config to be structurally
  distinguishable from its student-visible config so that redaction is enforced by the framework and
  not by each tool author remembering to do it.
- **As a homeschool parent-instructor**, I want tools to work in a course with a single learner and
  no org so that the feature is not gated behind tenant configuration I do not have.

## 5. Functional Requirements

- **FR-1.** The system MUST add `course.courses.content_tools_enabled BOOLEAN NOT NULL DEFAULT FALSE`,
  surfaced as `contentToolsEnabled` on the course features payload and patchable by users holding the
  course-settings permission, exactly as `adaptiveContentEnabled` is.
- **FR-2.** The system MUST provide a server-side **tool registry** keyed by a stable `tool_id`
  (`snake_case`, immutable for the life of the tool) that resolves to a `ToolManifest`.
- **FR-3.** A `ToolManifest` MUST declare: `id`, `version` (semver), `name`, `category`, `capabilities`,
  `configSchema` (JSON Schema draft 2020-12), `stateSchema`, `scoring` (`none|auto|manual|external`),
  `ai` (`{featureId, required}` or absent), `network` (`{allowedHosts}` or absent), `storage`
  (`{maxStateBytes}`), `roles`, `a11y`, `i18nNamespace` and `ui` (`{renderer, icon, group}`).
- **FR-4.** The registry MUST be validated at process start: duplicate ids, invalid semver, invalid
  JSON Schema, missing i18n bundle or a `maxStateBytes` above the platform ceiling MUST fail startup
  (fail fast, never silently drop a tool).
- **FR-5.** The system MUST persist a **tool instance** per placed tool: `(id, course_id,
  structure_item_id, host_kind, tool_id, tool_version, config_json, title, status)`.
- **FR-6.** The system MUST validate `config_json` against the manifest's `configSchema` on every
  write and reject non-conforming payloads with HTTP 422 and a field-level error list.
- **FR-7.** The system MUST persist **learner state** at most once per `(instance_id, enrollment_id)`
  in `state_json JSONB`, with `revision`, `status`, optional `score_raw`/`score_max`, interaction
  counters and reset bookkeeping.
- **FR-8.** State MUST be keyed by **enrollment**, not user — so the same human enrolled in two
  courses, or re-enrolled after a drop, gets independent state, and state is removed with the
  enrollment via `ON DELETE CASCADE`.
- **FR-9.** The system MUST support a per-course **tool allowlist**: `NULL`/empty = every
  org-permitted tool is available; a non-empty array restricts the palette to those `tool_id`s.
- **FR-10.** Config fields marked `"x-lex-sensitive": true` in a manifest's `configSchema` MUST be
  stripped from any response served to a principal without the instructor-grade permission on that
  course. The framework performs the stripping; tools MUST NOT be trusted to do it.
- **FR-11.** The system MUST expose instructor-gated instance CRUD under
  `/api/v1/courses/{course_code}/content-tools/instances`.
- **FR-12.** The system MUST expose a catalog endpoint listing the tools available in a given course
  after flag, allowlist, org-policy and role filtering.
- **FR-13.** The system MUST record an append-only event row for instance create / update / archive
  and (from CT.3 onward) for learner interactions.
- **FR-14.** All Content Tools endpoints MUST return HTTP 404 (not 403) when
  `content_tools_enabled` is false, so a disabled course leaks nothing about the feature.
- **FR-15.** The system SHOULD expose `GET .../content-tools/manifests/{tool_id}` returning the
  manifest (minus sensitive schema annotations) so clients can render config forms generically.
- **FR-16.** The system MAY support instance-level `tool_version` pinning ahead of CT.5's migration
  machinery; until CT.5, the pinned version is recorded but always resolved to the running build.

## 6. Non-Functional Requirements

- **Performance** — Catalog and instance-list reads p95 ≤ 80 ms server-side; a content page with 20
  instances MUST resolve all configs in **one** query (no N+1). Manifest lookup is in-memory O(1).
- **Security** — Every route authenticated and course-scoped through the existing `courseroles`
  middleware; config writes require the same permission as editing the host item. Sensitive config
  redaction (FR-10) is enforced in the serializer, covered by an explicit authz test matrix.
  `config_json` and `state_json` are size-capped (default 256 KB and 64 KB) to bound abuse.
- **Privacy & Compliance** — `state_json` is student work and therefore part of the education record:
  it is in scope for FERPA disclosure, DSAR export (S01) and retention (S02). CT.1 registers the
  tables with the existing DSAR/retention registries so no later story has to remember.
- **Accessibility** — No UI in this story. The manifest's `a11y` block is mandatory metadata: a tool
  cannot be registered without declaring keyboard operability and its screen-reader pattern; CT.8
  turns that declaration into a shipping gate.
- **Scalability** — Instances are indexed by `(structure_item_id, status)`; states by
  `(instance_id, enrollment_id)` unique and `(enrollment_id)`. Expect ≤ 50 instances per item, ≤ 10⁷
  state rows per large tenant. `state_json` uses `jsonb_path_ops` GIN only where CT.7 proves a need —
  not by default.
- **Reliability** — Instance writes are transactional with the host item's body save (CT.2). An
  orphaned instance (body no longer references it) is harmless and swept nightly. Availability target
  99.9%, inheriting the API SLO.
- **Observability** — `lextures_content_tool_instances_total{tool_id,action}`,
  `lextures_content_tool_config_validation_failures_total{tool_id}`,
  `lextures_content_tool_registry_size`, log field `tool_id` on every handler.
- **Maintainability** — One folder per tool; the registry index is generated, not hand-maintained. A
  `go test ./internal/service/contenttools -run TestRegistryContract` asserts every manifest against
  the contract so a malformed tool fails CI, not production.
- **Internationalization** — Manifest carries `i18nNamespace`; names/descriptions are never stored in
  English in the DB, only as keys resolved client-side.
- **Backward compatibility** — Purely additive. Bodies that contain no ` ```lex-tool ` fence are
  byte-identical before and after. Courses with the flag off behave exactly as today.

## 7. Acceptance Criteria

- **AC-1.** *Given* a course with `contentToolsEnabled=false`, *When* any `/content-tools/*` endpoint
  is called, *Then* the response is 404 and no row is written.
- **AC-2.** *Given* an instructor patches course features with `contentToolsEnabled=true`, *When* the
  course is re-fetched, *Then* the flag persists and appears in the features payload.
- **AC-3.** *Given* a manifest declaring `configSchema.required = ["prompt"]`, *When* an instance is
  created without `prompt`, *Then* the API returns 422 with `errors[0].path = "/prompt"` and no row is
  created.
- **AC-4.** *Given* a manifest field marked `"x-lex-sensitive": true`, *When* a student fetches the
  instance, *Then* that field is absent from the response while the instructor's fetch contains it.
- **AC-5.** *Given* two enrollments of the same user in two courses, *When* each stores state for a
  tool, *Then* the rows are independent and neither read returns the other's document.
- **AC-6.** *Given* an enrollment is deleted, *When* the transaction commits, *Then* all
  `content_tool_states` rows for that enrollment are gone (FK cascade), verified by an integration test.
- **AC-7.** *Given* a course allowlist of `["inline_questions"]`, *When* the catalog is fetched,
  *Then* exactly one tool is returned regardless of how many are registered.
- **AC-8.** *Given* a registry containing a manifest with a duplicate `tool_id`, *When* the server
  starts, *Then* startup fails with a descriptive error and a non-zero exit code.
- **AC-9.** *Given* a content page with 20 instances, *When* the page's instances are listed, *Then*
  exactly one DB round-trip is issued (asserted by a query-count test).
- **AC-10.** *Given* a `state_json` payload above `maxStateBytes`, *When* it is written, *Then* the
  API returns 413 and the stored document is unchanged.

## 8. Data Model

Migration `server/migrations/449_content_tools_core.sql` (+ `.down.sql`).

```sql
-- 449_content_tools_core.sql

-- Per-course flag (mirrors adaptive_content_enabled / adaptive_paths_enabled).
ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS content_tools_enabled BOOLEAN NOT NULL DEFAULT FALSE;
COMMENT ON COLUMN course.courses.content_tools_enabled IS
    'When true, interactive Content Tools may be inserted into section bodies and may store per-enrollment state (plan CT.1).';

-- Per-course configuration, created lazily on first PUT.
CREATE TABLE IF NOT EXISTS course.content_tool_settings (
    course_id            UUID PRIMARY KEY REFERENCES course.courses (id) ON DELETE CASCADE,
    allowed_tool_ids     TEXT[] NOT NULL DEFAULT '{}',   -- empty = all org-permitted tools
    student_reset_allowed BOOLEAN NOT NULL DEFAULT FALSE, -- may a student clear their own state?
    max_instances_per_item SMALLINT NOT NULL DEFAULT 50 CHECK (max_instances_per_item BETWEEN 1 AND 200),
    updated_by           UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One placed tool. The body Markdown carries only `id`; everything else lives here.
CREATE TABLE IF NOT EXISTS course.content_tool_instances (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id             UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    structure_item_id     UUID REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    host_kind             TEXT NOT NULL CHECK (host_kind IN
                            ('content_page','assignment','quiz','syllabus','portfolio_artifact')),
    section_key           TEXT,            -- editor section this instance sits in (advisory only)
    tool_id               TEXT NOT NULL,
    tool_version          TEXT NOT NULL,   -- semver pinned at insert (CT.5 uses it to migrate)
    title                 TEXT,
    config_json           JSONB NOT NULL DEFAULT '{}'::jsonb,
    config_schema_version INTEGER NOT NULL DEFAULT 1,
    status                TEXT NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active','archived')),
    created_by            UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT content_tool_instances_config_size CHECK (pg_column_size(config_json) <= 262144),
    CONSTRAINT content_tool_instances_syllabus_shape CHECK (
        (host_kind = 'syllabus' AND structure_item_id IS NULL)
        OR (host_kind <> 'syllabus' AND structure_item_id IS NOT NULL)
    )
);
CREATE INDEX IF NOT EXISTS idx_cti_item      ON course.content_tool_instances (structure_item_id, status);
CREATE INDEX IF NOT EXISTS idx_cti_course_tool ON course.content_tool_instances (course_id, tool_id);

-- Per-enrollment learner state. THE dynamic surface: one JSONB document per (instance, enrollment).
CREATE TABLE IF NOT EXISTS course.content_tool_states (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id          UUID NOT NULL REFERENCES course.content_tool_instances (id) ON DELETE CASCADE,
    enrollment_id        UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    user_id              UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    state_json           JSONB NOT NULL DEFAULT '{}'::jsonb,
    state_schema_version INTEGER NOT NULL DEFAULT 1,
    revision             BIGINT NOT NULL DEFAULT 0,       -- optimistic concurrency token
    status               TEXT NOT NULL DEFAULT 'not_started'
                           CHECK (status IN ('not_started','in_progress','submitted','completed')),
    score_raw            NUMERIC(10,4),
    score_max            NUMERIC(10,4),
    interaction_count    INTEGER NOT NULL DEFAULT 0,
    first_interacted_at  TIMESTAMPTZ,
    last_interacted_at   TIMESTAMPTZ,
    completed_at         TIMESTAMPTZ,
    reset_count          INTEGER NOT NULL DEFAULT 0,
    last_reset_at        TIMESTAMPTZ,
    last_reset_by        UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (instance_id, enrollment_id),
    CONSTRAINT content_tool_states_size CHECK (pg_column_size(state_json) <= 65536),
    CONSTRAINT content_tool_states_score_shape CHECK (
        (score_raw IS NULL AND score_max IS NULL) OR (score_max IS NOT NULL AND score_max > 0)
    )
);
CREATE INDEX IF NOT EXISTS idx_cts_enrollment ON course.content_tool_states (enrollment_id);
CREATE INDEX IF NOT EXISTS idx_cts_instance_status ON course.content_tool_states (instance_id, status);

-- Append-only log: authoring changes now, learner interactions from CT.3, resets from CT.4.
CREATE TABLE IF NOT EXISTS course.content_tool_events (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    instance_id   UUID REFERENCES course.content_tool_instances (id) ON DELETE CASCADE,
    course_id     UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    enrollment_id UUID REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    actor_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    tool_id       TEXT NOT NULL,
    event_type    TEXT NOT NULL,   -- instance_created|instance_updated|instance_archived|…
    payload_json  JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_cte_instance_created ON course.content_tool_events (instance_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_cte_course_created   ON course.content_tool_events (course_id, created_at DESC);
```

**Indexes & constraints** — the unique `(instance_id, enrollment_id)` is the concurrency anchor:
every state write is an `INSERT … ON CONFLICT DO UPDATE … WHERE revision = $expected RETURNING`, so a
lost update is impossible without a client seeing 409.

**Backfill** — none. Every table starts empty; every existing course has the flag `FALSE`.

**Retention** — `content_tool_events` older than the org's `learning_event_retention_days` are purged
by the nightly sweeper CT.7 extends; state rows follow the enrollment.

## 9. API Surface

All routes are course-scoped and 404 when the course flag is off.

| Verb | Path | Auth scope |
|---|---|---|
| `GET` | `/api/v1/courses/{course_code}/content-tools/catalog` | any course member |
| `GET` | `/api/v1/courses/{course_code}/content-tools/manifests/{tool_id}` | any course member |
| `GET` | `/api/v1/courses/{course_code}/content-tools/settings` | instructor |
| `PUT` | `/api/v1/courses/{course_code}/content-tools/settings` | instructor |
| `GET` | `/api/v1/courses/{course_code}/content-tools/instances?itemId=&hostKind=` | any course member (config redacted for students) |
| `POST` | `/api/v1/courses/{course_code}/content-tools/instances` | instructor |
| `PATCH` | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}` | instructor |
| `DELETE` | `/api/v1/courses/{course_code}/content-tools/instances/{instance_id}` | instructor (soft → `archived`) |

```ts
type ToolManifest = {
  id: string                      // 'inline_questions' — immutable
  version: string                 // '1.2.0'
  name: string                    // i18n key
  category: 'assess' | 'explore' | 'reflect' | 'discuss' | 'practice' | 'read'
  capabilities: Array<'state' | 'scoring' | 'ai' | 'network' | 'media' | 'realtime' | 'aggregate'>
  configSchema: JSONSchema        // draft 2020-12; supports "x-lex-sensitive": true
  stateSchema: JSONSchema
  scoring: { mode: 'none' | 'auto' | 'manual' | 'external'; maxScore?: number }
  ai?: { featureId: string; required: boolean }
  network?: { allowedHosts: string[] }
  storage: { maxStateBytes: number }
  roles: { interact: Array<'student' | 'instructor' | 'observer'> }
  a11y: { keyboardOperable: true; srPattern: string; wcagNotes?: string }
  i18nNamespace: string
  ui: { renderer: string; icon: string; group: string }
}

type ToolInstance = {
  id: string
  toolId: string
  toolVersion: string
  hostKind: 'content_page' | 'assignment' | 'quiz' | 'syllabus' | 'portfolio_artifact'
  structureItemId: string | null
  sectionKey: string | null
  title: string | null
  config: Record<string, unknown>   // sensitive keys stripped for students
  status: 'active' | 'archived'
  updatedAt: string
}
```

- **Rate limits** — instance writes 60/min/user; catalog reads 600/min/user (cached 60 s).
- **OpenAPI** — every route added to the generated spec; CI fails if a `/content-tools/` route is
  missing from `openapi.json` (guardrail inherited from TD.3).

## 10. UI / UX

No user-facing UI ships in CT.1 beyond the existing course-settings surface:

1. **Course settings → Features** gains a *Content Tools* toggle with helper copy and a link to docs.
2. When toggled on, an *Available tools* multi-select appears (the allowlist, empty = all).
3. Toggling **off** shows a confirmation explaining that existing tool blocks will render as a
   static "This interactive element is turned off" placeholder and that **no state is deleted**.

Empty/loading/error states follow the existing features panel. Copy keys land in
`clients/web/public/locales/en/common.json` under `course.features.contentTools.*`.

## 11. AI / ML Considerations

None in CT.1 — zero prompts, zero provider calls. The manifest's `ai` block only *declares* that a
tool will call a model, which CT.6 honours and CT.8 audits. Registering `ai.featureId` values that
are unknown to `aigateway` fails startup validation (FR-4), so no tool can make an undisclosed call.

## 12. Integration Points

- **Internal** — `server/internal/httpserver/course_features.go` (flag), `models/course/types.go`,
  `clients/web/src/lib/courses-api-schemas.ts`, new `service/contenttools/`,
  `repos/contenttools/`, `models/contenttools/`, `httpserver/content_tools_*.go`.
- **Course copy / export** (`service/coursecopy`, `service/courseexportimport`) — copying a course
  MUST clone instances with **new** ids and rewrite the ` ```lex-tool ` fences in the copied bodies;
  learner state is never copied. CT.1 ships the clone hook and a test.
- **DSAR / retention** (`service/gdpr`, `service/ferpa`) — register `content_tool_states` and
  `content_tool_events` in the subject-data registry so exports and deletions include them.
- **Admin audit** (`service/adminaudit`) — flag flips and allowlist edits are audited.
- **Events** — no webhook emission in CT.1; CT.7 owns outbound events.

## 13. Dependencies & Sequencing

- **Must ship after:** nothing. Uses shipped course structure, enrollments and RBAC.
- **Must ship before:** CT.2 … CT.9 and every tool story.
- **Shared infra needed:** Postgres only. No queue, no object storage, no new external service.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `state_json` becomes an unqueryable dumping ground | M | M | Every tool must declare a `stateSchema`; CT.7 reads only through documented projections; size cap enforced by constraint |
| Instance rows orphaned when an author deletes the fence from the body | H | L | Orphans are inert; nightly sweep archives instances unreferenced for 30 days; state preserved until the instance is hard-deleted |
| Config schema churn breaks stored instances | M | H | `config_schema_version` + semver pinning now; CT.5 adds the actual migration runner before any tool ships a breaking change |
| Sensitive config (answer keys) leaks to students | M | H | Framework-level redaction (FR-10) with an authz test matrix; tools cannot opt out |
| Enrollment-keyed state surprises multi-section courses | L | M | Documented explicitly; sections share one enrollment by design, matching gradebook semantics |
| Registry grows to hundreds and slows startup | L | L | Manifests are compiled once; startup budget asserted at ≤ 50 ms for 500 tools in a benchmark test |

## 15. Rollout Plan

- **Feature flag** — per-course `content_tools_enabled` (default `FALSE`). Ops kill-switch
  `CONTENT_TOOLS_KILL_SWITCH` (default disengaged) short-circuits every route to 404 for incident
  response only.
- **Sequencing** — migration `449_*` → server code (registry, repo, handlers) → OpenAPI regen →
  features toggle in web → enable on the internal dogfood course.
- **Dogfood** — the Lextures intro course (`service/introcourse`) is the first course with the flag on.
- **GA criteria** — CT.1–CT.4 merged, CT.8 sign-off obtained, error rate < 0.1% over 7 dogfood days.
- **Rollback** — flip the flag off (content renders placeholders, state retained); if schema rollback
  is required, `449_content_tools_core.down.sql` drops the three tables and the column. No other table
  is touched, so rollback is total.

## 16. Test Plan

- **Unit** — manifest contract validation (valid, duplicate id, bad semver, bad schema, unknown
  `ai.featureId`, oversize `maxStateBytes`); config validation success/failure paths; sensitive-field
  redaction; registry lookup.
- **Integration** — instance CRUD against a real DB; flag-off 404 matrix; allowlist filtering; cascade
  delete on enrollment removal; `pg_column_size` constraints; single-query instance listing.
- **End-to-end** — Playwright: instructor enables the flag, the settings panel persists, the catalog
  endpoint reflects the allowlist.
- **Security** — authz matrix (student/TA/instructor/observer/other-course/anonymous × every route);
  redaction assertions; oversize payload rejection; SQL-injection fuzz on `tool_id` path params.
- **Accessibility** — axe on the features panel additions.
- **Performance** — benchmark: registry startup with 500 synthetic manifests ≤ 50 ms; instance list
  with 50 instances ≤ 80 ms p95.
- **Manual exploratory** — enable/disable cycling on a course with existing content; verify no body
  Markdown mutation.

## 17. Documentation & Training

- **End-user** — help-center article "Turn on Content Tools for your course".
- **Instructor** — what the allowlist does; what happens to student work when the flag is turned off.
- **Developer** — `docs/dev/content-tools-authoring.md`: how to add a tool (manifest → renderer →
  tests), the immutability rule for `tool_id`, and the "no migration" contract.
- **API reference** — OpenAPI regenerated; new tag `content-tools`.
- **Runbook** — kill-switch procedure; how to inspect a malformed `state_json` safely.

## 18. Open Questions

1. Should `content_tool_settings.student_reset_allowed` default to `TRUE` for homeschool-program
   courses (single learner, no instructor to ask)? Proposed: keep `FALSE`, revisit in CT.4.
2. Do we need instance-level scheduling (available-from/until) at the framework level, or is that a
   per-tool config concern? Proposed: framework level in CT.2 if two tools ask for it.
3. Should `host_kind = 'quiz'` be allowed at all, given quizzes have their own attempt lifecycle?
   Proposed: allow in the schema, block in the CT.2 palette until CT.7 defines the interaction with
   attempt state.
4. Ceiling for `maxStateBytes` — 64 KB covers every tool in CT.10–CT.23; the code sandbox (CT.17) may
   want 256 KB. Owner: CT.17 author, decide before CT.5 freezes the contract.

## 19. References

- Existing files this work touches: `server/internal/httpserver/course_features.go`,
  `server/internal/httpserver/courses_routes.go`, `server/internal/models/course/types.go`,
  `server/migrations/449_content_tools_core.sql`, `clients/web/src/lib/courses-api-schemas.ts`.
- Precedents followed: `server/migrations/439_adaptive_content_core.sql` (per-course flag + core
  model), `server/migrations/165_h5p_packages.sql` (`content.h5p_completions` per-user state),
  `clients/web/src/components/editor/extensions/board-tip-tap.ts` (fenced-block serialization).
- External standards: JSON Schema draft 2020-12; RFC 2119; SemVer 2.0.0.
- Related plans: [CT.2](CT.2-authoring-tools-dropdown-and-config.md),
  [CT.3](CT.3-student-runtime-and-state-persistence.md), [CT.5](CT.5-tool-sdk-sandboxing-and-versioning.md),
  [CT.8](CT.8-governance-safety-privacy-accessibility.md),
  [AC.1](../../completed/adaptive/AC.1-foundations-flag-and-data-model.md).
