---
name: requirements
description: Turns a feature request into a full implementation-plan requirements document matching docs/plan/_TEMPLATE.md. Use when the user asks for requirements, a spec, an implementation plan, or wants to flesh out a feature before building. Also use for /requirements or "write requirements for X".
disable-model-invocation: true
---

# Requirements

Turn a user request into a complete Lextures implementation plan — the same structure and rigor as `docs/plan/`.

## Output

Produce a markdown document that follows `docs/plan/_TEMPLATE.md` exactly. Every section must be filled; no `…` or `TBD` placeholders unless listed explicitly in **Open Questions** (§18).

Default: paste the full document in chat. If the user asks to save it, write to `docs/plan/{section-folder}/{id}-{kebab-slug}.md` per [docs/plan/README.md](../../../docs/plan/README.md) conventions.

## Workflow

Copy this checklist and track progress:

```
Requirements progress:
- [ ] Step 1: Parse the request
- [ ] Step 2: Explore codebase & existing plans
- [ ] Step 3: Draft metadata
- [ ] Step 4: Write all 19 sections
- [ ] Step 5: Validate against checklist
```

### Step 1. Parse the request

Extract from the user's message:

- **Feature name** — short, descriptive title
- **Scope** — what they want; what they explicitly excluded
- **Audience** — K12 / HE / SL (self-learner) / platform-wide
- **Urgency signals** — blocker vs enhancement, pilot vs GA

If critical details are missing (audience, scope boundary, or whether this is net-new vs extending existing behavior), ask **one** focused question before proceeding. Otherwise state your interpretation briefly and continue.

### Step 2. Explore codebase & existing plans

Before writing requirements, ground the spec in what exists:

1. **Read** `docs/plan/_TEMPLATE.md` for the canonical section list.
2. **Search** `docs/plan/` for related plans — read 1–2 closest matches for tone, cross-references, and dependency patterns.
3. **Search** `docs/completed/` if the feature may partially exist.
4. **Explore the codebase** — Glob/Grep/Read for:
   - Existing routes, services, DB tables, UI pages that overlap
   - Patterns to extend (auth scopes, job queue, feature flags, migrations)
   - File paths to cite in §19 References

Record findings: what is **MISSING**, **PARTIAL**, or **THIN** today (for metadata **Status**).

### Step 3. Draft metadata

Fill the metadata table. Propose values when the user did not supply them:

| Field | Guidance |
|---|---|
| **Feature ID** | `{section}.{number}` — match an existing `docs/plan/` section when possible; otherwise propose next number in the closest section |
| **Section** | Top-level plan folder name (e.g. Platform, Performance & Operability) |
| **Severity** | BLOCKER / MAJOR / MINOR per [docs/plan/README.md](../../../docs/plan/README.md) legend |
| **Markets** | K12 / HE / SL — comma-separated |
| **Status (today)** | MISSING / PARTIAL / THIN based on codebase exploration |
| **Estimated effort** | XS / S / M / L / XL |
| **Depends on** / **Unblocks** | Feature IDs from related plans; use relative links |

Title line format:

```markdown
# {Feature ID} — {Feature Name}

> Implementation plan. Source: user request / {related doc if any}.
```

### Step 4. Write all 19 sections

Follow the template section-by-section. Quality bar matches completed plans like `17.14-feature-flags.md` and `19.3-ai-assisted-grading.md`.

**§1 Problem Statement** — 2–4 sentences. Gap today, who is hurt, business outcome.

**§2 Goals** — 3–5 outcome bullets. Verifiable, not implementation steps.

**§3 Non-Goals** — Explicit out-of-scope items. Reference other Feature IDs when deferring work.

**§4 Personas & User Stories** — `As a {role}, I want … so that …`. Cover student, instructor, admin, and self-learner where relevant.

**§5 Functional Requirements** — Numbered `FR-N`. RFC 2119: MUST / SHOULD / MAY. Testable, not vague.

**§6 Non-Functional Requirements** — Cover applicable sub-bullets from the template (performance, security, privacy, a11y, scalability, reliability, observability, maintainability, i18n, backward compatibility). Omit sub-bullets that genuinely do not apply; say "N/A" with one-line rationale.

**§7 Acceptance Criteria** — Numbered `AC-N`. Given/When/Then. Each AC should map to at least one test.

**§8 Data Model** — SQL or schema description. Migration naming: `server/migrations/NNN_*.sql`. Indexes, constraints, backfill strategy.

**§9 API Surface** — Table or list: method, path, auth scope, request/response shapes. Note rate limits and OpenAPI updates.

**§10 UI / UX** — Pages, components, flows, empty/loading/error states, responsive behavior, a11y notes, i18n keys.

**§11 AI / ML Considerations** — **Skip entirely** (write "N/A — not AI-touching.") only when the feature has zero AI/LLM/ML involvement. Otherwise: models, prompts, eval metrics, fallback, PII, cost budget.

**§12 Integration Points** — External services and internal modules with file paths.

**§13 Dependencies & Sequencing** — Must ship after/before; shared infra (Redis, job queue, object storage, etc.).

**§14 Risks & Mitigations** — Table: Risk | Likelihood | Impact | Mitigation.

**§15 Rollout Plan** — Feature flag name, phases, pilot cohort, GA criteria, rollback path.

**§16 Test Plan** — Unit, integration, e2e (Playwright), security, accessibility (axe), performance/load, manual QA.

**§17 Documentation & Training** — User docs, admin docs, API reference, runbooks.

**§18 Open Questions** — Numbered decisions still needing owners. **Only** place for unresolved `TBD`s.

**§19 References** — Existing code paths (`server/internal/...`, `clients/web/src/...`), external standards, related plans as relative links.

#### Lextures-specific defaults

When the spec touches these areas, align with repo conventions (see `AGENTS.md`):

- **Backend:** Go API, Chi router, pgx, JWT auth, `server/internal/`
- **Frontend:** React 19, Vite, TypeScript, Tailwind v4, `clients/web/src/`
- **Migrations:** `server/migrations/NNN_descriptive_name.sql`
- **Auth:** Note JWT scope and role checks on every new route
- **Jobs:** Background work via RabbitMQ queue (or in-process fallback locally)
- **Feature flags:** Reference OpenFeature-style flags and `platformstate` when rollout-sensitive
- **Compliance:** FERPA for student data; WCAG 2.1 AA for new UI; COPPA for K12 minors

### Step 5. Validate

Before delivering, verify every item in [references/checklist.md](references/checklist.md). Fix gaps before presenting.

## Delivery format

Present the document in this order:

1. **Summary** (3–5 bullets) — what the feature does, severity, effort, key dependencies
2. **Full requirements document** — complete markdown, all 19 sections
3. **Suggested next steps** — only if useful: save path, related plans to read, open questions needing user input

## Examples

**User:** "Write requirements for instructor bulk email to enrolled students"

**Agent:** Explores `server/internal/` for messaging/email modules and `docs/plan/06-communication-collaboration/`. Produces a full 19-section plan with FRs for opt-out compliance, rate limits, and audit logging.

**User:** "Requirements for adding dark mode to the student dashboard"

**Agent:** Finds existing theme/CSS variables in `clients/web/`. Status = PARTIAL if theme tokens exist. Non-goals reference system-preference-only scope. ACs cover persistence, contrast ratios, and no flash on load.

## Additional resources

- Canonical template: [docs/plan/_TEMPLATE.md](../../../docs/plan/_TEMPLATE.md)
- Plan conventions: [docs/plan/README.md](../../../docs/plan/README.md)
- Quality checklist: [references/checklist.md](references/checklist.md)
