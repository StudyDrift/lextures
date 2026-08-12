# Architecture Conventions

> Short, opinionated charter for where code belongs and how structure is enforced.
> Enforcement: `make lint-structure` (local) and the **Structure (TD.2)** CI job.
> Programme: [tech debt remediation](plan/tech_debt/README.md).

## Prime directive

**No user-visible behaviour may change during structural remediation.**

Refactors, package splits, and dead-code deletions must leave the HTTP contract and
product UI behaviour byte-identical unless the story explicitly repairs a confirmed
defect. Proof mechanism: the [TD.1 safety net](completed/tech_debt/TD.1-refactoring-safety-net.md)
(`make route-inventory`, characterization goldens, web API-surface goldens).

If a change cannot be made behaviour-preserving, **stop and re-scope**.

Accessibility is not optional: any UI work remains bound to **WCAG 2.1 AA**.

---

## 1. Package layout

### Go (`server/`)

| Layer | Package | Responsibility |
|---|---|---|
| HTTP | `internal/httpserver` (→ domain packages per TD.6) | Authn/authz, decode/encode, call services/repos, map errors to status codes |
| Service | `internal/service/*` | Domain workflows, orchestration, side effects |
| Repo | `internal/repos/*` | SQL, `pgx`, transactions — the **only** place for database access |
| Model | `internal/models`, domain types | Shared types, no I/O |
| Composition | `cmd/server`, wiring in `httpserver`/`Deps` | Construct deps; no business logic |

New code goes in the thinnest package that already owns the concern. Prefer a new
file over growing a god-file; prefer a new package only when there is a real seam
(see budgets below).

`internal/repos/marketingcontent` owns all SQL for the platform-global
`marketing` schema, including articles, append-only revisions, authors,
categories, tags, and redirects (MC.1).

### TypeScript (`clients/web/src/`)

| Area | Path | Responsibility |
|---|---|---|
| Pages | `pages/` | Route-level orchestration; keep thin |
| Components | `components/` | Presentational and feature UI |
| API clients | `lib/*-api.ts` | HTTP calls and DTO types (consolidation: TD.11–TD.12) |
| Hooks / context | `hooks/`, `context/` | Shared client state (server-state library: TD.13) |
| Tests | colocated `__tests__/` or `*.test.ts(x)` next to source | Unit/component tests |

---

## 2. Layering rules

1. **`internal/httpserver` must not talk to the database.**
   - Forbidden: `d.Pool.Query`, `d.Pool.QueryRow`, `d.Pool.Exec`, and raw SQL string literals.
   - Allowed: call `internal/repos/*` (and services that use repos).
   - Burn-down: [TD.9](plan/tech_debt/TD.9-enforce-repo-layering.md); allowlist
     `scripts/allowlists/layering.txt`.
2. **Repos own SQL.** Handlers and services do not embed SQL.
3. **Web API modules own HTTP.** Pages/components call `lib/*-api` helpers; they do
   not reimplement `fetch`/`apiJson` (duplication burn-down: TD.11).
4. **No silent layering exceptions.** Legitimate exceptions are listed in an allowlist
   owned by a TD story and must shrink.

---

## 3. File-size budgets

| Language | Scope | Budget | Allowlist | Burn-down owner |
|---|---|---|---|---|
| Go | non-test `*.go` under `server/` | **≤ 600 LOC** | `scripts/allowlists/file-size.txt` | TD.6 (+ other domain splits) |
| TypeScript | `clients/web/src/**/*.{ts,tsx}` | **≤ 500 LOC** | same file | TD.12–TD.14 |

Splitting a file only to satisfy the budget without a real seam is a review fail.
Test files (`*_test.go`, `*.test.ts(x)`) are not budgeted by this rule.
Codegen output under `clients/web/src/lib/generated/` (e.g. OpenAPI types from TD.3) is
excluded from the TypeScript file-size budget.

---

## 4. Package-size budgets

| Scope | Budget | Allowlist | Burn-down owner |
|---|---|---|---|
| Go package directory under `server/` | **≤ 40** non-test `.go` files | `scripts/allowlists/package-size.txt` | TD.6 (`httpserver`), background cleanup |

Today `internal/httpserver` is far over budget (~396 files); new packages must not
cross 40 files without an allowlist entry (which is shrink-only).

---

## 5. Naming

### Go

- Files: `snake_case.go` (existing idiom).
- Packages: short, lower-case, no underscores (`httpserver`, `gradingagent`).
- Exported identifiers: `MixedCaps`.

### Web (`clients/web/src`)

- **File basenames: kebab-case** — `[a-z0-9.-]+.(ts|tsx)`.
  - Good: `course-modules.tsx`, `use-locale-format.ts`, `reading-preferences-panel.tsx`.
  - Bad: `CourseModules.tsx`, `useLocaleFormat.ts`, `ReadingPreferencesPanel.tsx`.
- Allowlist of existing violators: `scripts/allowlists/file-naming.txt` (TD.11/TD.14).
- React component **exports** may still be `PascalCase`; only the **filename** is constrained.

---

## 6. HTTP handlers (method dispatch)

- **Handlers do not check `r.Method`.** The chi router owns dispatch via
  `r.Get` / `r.Post` / `r.Put` / `r.Patch` / `r.Delete` (and `r.Method`).
- **`OPTIONS` preflight** is handled by `corsAll` (and the central
  `MethodNotAllowed` handler for unregistered verbs). Do not re-implement
  OPTIONS or 405 prologues in single-method handlers.
- **Method-agnostic handlers** are registered with `Handle` / `HandleFunc` /
  `Mount`, or the same handler is bound to more than one verb. Those handlers
  may keep in-handler method dispatch and **must document why** (comment at the
  registration site or on the handler). See TD.5.
- Guard: `scripts/check-handler-method-dispatch.sh` (via `make lint-structure`).
- Analysis map: `python3 scripts/analyze-handler-methods.py`.

---

## 6b. Error handling

### Go

- Return `error`; do not panic in request paths.
- Map domain errors to HTTP in the handler layer (toolkit: TD.7).
- Log with `log/slog` at the boundary; include request/correlation ids where available.
- Never swallow errors with `_ =` without a comment justifying why.

### TypeScript

- Surface API failures to the user; do not empty-catch.
- Prefer typed error helpers from the shared HTTP foundation (TD.11) over ad-hoc `throw new Error(await res.text())`.

---

## 7. Test placement

| Kind | Location |
|---|---|
| Go unit / table tests | `foo_test.go` next to `foo.go` |
| Go integration (DB) | same package or `test/`; gated by `DATABASE_URL` / `-short` |
| Web unit / component | `*.test.ts(x)` or `__tests__/` next to source |
| E2E journeys | `e2e/tests/*.spec.ts` |
| Structural / contract guards | TD.1 goldens under `server/internal/httpserver/testdata/`; web `api-surface` tests; OpenAPI guards in `server/internal/openapi/` |

A PR that only moves files must leave TD.1 inventories unchanged.

---

## 7b. OpenAPI contract (TD.3)

- The LMS HTTP OpenAPI 3.0.3 document lives at **`server/internal/openapi/openapi.json`**
  and is embedded via `go:embed` (served at `GET /api/openapi.json`, UI at `GET /api/docs`).
- **API changes update the spec in the same PR** when the change is intentional and
  client-visible. Do not hand-edit a Go string literal for the document.
- A separate partner Public API spec lives at `server/internal/publicapi/openapi.json`
  (OpenAPI 3.1, `GET /api/v1/openapi.json`) — do not conflate the two.
- Guards: `make openapi-check`, `go test ./internal/openapi/`. Coverage baseline:
  `scripts/allowlists/openapi-coverage.txt` (absolute documented path count, shrink-only).
- After backend contract changes, regenerate web types when needed:
  `cd clients/web && npm run openapi:types:file`.

---

## 8. Dead code

- Confirmed-unreachable Go functions are tracked by `scripts/check-deadcode-baseline.sh`.
- Baseline list: `scripts/allowlists/deadcode-baseline.txt` (owner: **TD.4**).
- The count of unreachable functions **must not grow**. Deletions lower the baseline.
- **Unreachable code is deleted, not commented out.** The ratchet enforces this: new unreachable
  functions fail CI. Prefer wiring a real call site or removing the symbol over leaving scaffolding.
- Triage classes and decision dates for retained findings: [`docs/tech-debt/deadcode-triage.md`](tech-debt/deadcode-triage.md).
- Symbols removed under TD.4 are recorded in [`docs/tech-debt/removed-symbols.md`](tech-debt/removed-symbols.md).

---

## 9. Enforcement & ratchets

| Check | Command | Allowlist | Blocks |
|---|---|---|---|
| File size | `scripts/check-file-budgets.sh` | `file-size.txt` | new oversized files |
| Package size | `scripts/check-package-budgets.sh` | `package-size.txt` | new oversized packages |
| Layering | `scripts/check-layering.sh` | `layering.txt` | new SQL/pool use in HTTP |
| Web naming | `scripts/check-file-naming.mjs` | `file-naming.txt` | new non-kebab files |
| Deadcode | `scripts/check-deadcode-baseline.sh` | `deadcode-baseline.txt` | new unreachable funcs |
| Handler method dispatch (TD.5) | `scripts/check-handler-method-dispatch.sh` | multi-method allowlist in script | new single-method `r.Method` checks |
| OpenAPI | `scripts/check-openapi-coverage.sh` | `openapi-coverage.txt` | invalid spec / coverage drop |
| Shrink-only | `scripts/check-allowlist-shrink.sh` | all of the above | allowlist growth |

```bash
make lint-structure          # all structural checks (blocking)
make lint-structure-report   # remaining counts only
make lint                    # app linters + structural checks
```

**Allowlists are shrink-only.** Adding a line fails CI unless the PR is labelled
`structure-allowlist-override` (GitHub) or you set `STRUCTURE_ALLOWLIST_GROW=1`
locally with a documented reason in the PR body. Prefer fixing the violation.

**Warn-only mode** (rollback / dogfood): `STRUCTURE_CHECKS_WARN=1 make lint-structure`
prints violations but exits 0.

---

## 10. How to proceed when a check fails

1. **Read the message** — it names the file, rule, measured value, budget, and owning TD story.
2. **Fix the code** (preferred): split on a real seam, move SQL into a repo, rename to kebab-case, delete dead code.
3. **If the violation is pre-existing and already allowlisted** — no action.
4. **If you need a temporary exception** — request the `structure-allowlist-override` label from a platform lead, add the entry, and open a follow-up to remove it. Never raise a numeric budget to silence CI.

---

## 11. Related docs

- [Tech debt programme](plan/tech_debt/README.md)
- [TD.1 safety net](completed/tech_debt/TD.1-refactoring-safety-net.md)
- [AGENTS.md](../AGENTS.md) — commands reference
- [docs/ARCH.md](ARCH.md) — longer-horizon architecture roadmap
- [clients/web/CONTRIBUTING.md](../clients/web/CONTRIBUTING.md) — web client conventions
