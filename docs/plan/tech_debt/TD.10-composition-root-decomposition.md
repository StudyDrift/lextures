# TD.10 — Composition Root: Decompose the `Deps` and `Config` God-Objects

> Implementation plan. Source: technical-debt static analysis, 2026-07-25. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | TD.10 |
| **Section** | Technical Debt Remediation |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS (internal) |
| **Status (today)** | MISSING |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | TD.6 |
| **Unblocks** | Honest per-domain testing; clear feature-dependency contracts |

---

## 1. Problem Statement

Every HTTP handler in the server is a method on one struct: `httpserver.Deps`, which carries **32 dependency fields** — the database pool, JWT signer, config, seven event hubs, four queues, object storage, Redis, telemetry, DRM, quotas, integrations, bots, and more. A handler that needs only the pool receives all 32, so nothing documents what any endpoint actually requires. Tests must construct the whole struct regardless of what they exercise. Roughly half the fields are optional (`nil` means "return 501"), so feature availability is expressed by nil-checks scattered across handlers rather than by types. Alongside it, `config.Config` carries **335 fields** in a single 1,529-line file spanning database, auth, storage, AI providers, billing, SMS, and observability settings. Together these are the reason a "small" backend change touches global structures, and the reason no domain can be reasoned about — or tested — in isolation.

## 2. Goals

- Replace the single 32-field `Deps` with **per-domain dependency structs** carrying only what each domain uses.
- Split `config.Config` into **domain-scoped configuration structs** composed into one root.
- Make optionality explicit in types rather than as nil-checks scattered through handlers.
- Let a domain's tests construct only that domain's dependencies.
- Preserve startup behaviour, environment-variable names, and platform-settings overrides exactly.

## 3. Non-Goals

- Introducing a dependency-injection framework or container. Explicit construction in a composition root is the target — Go idiom, not magic.
- Changing any environment-variable name, default, or precedence rule.
- Changing the platform-settings override mechanism (`platformstate.Platform`).
- Changing feature-flag semantics or which features are optional.
- Rewriting `internal/app/app.go`'s startup ordering.

## 4. Personas & User Stories

- **As a backend engineer**, I want a handler's dependencies visible in its signature, so that I know what it touches without reading its body.
- **As a test author**, I want to construct three dependencies instead of thirty-two, so that writing a test is not an exercise in stubbing.
- **As an SRE**, I want configuration grouped by concern, so that I can find every storage-related setting in one place.
- **As a domain owner**, I want a compile error when my domain gains a dependency, so that dependency growth is a deliberate decision.
- **As a new engineer**, I want `Config` to be navigable, so that adding a setting does not mean scrolling 1,529 lines.

## 5. Functional Requirements

- **FR-1.** The team MUST produce a **dependency-usage matrix** — which of the 32 `Deps` fields each domain package (post-TD.6) actually uses — derived mechanically from the symbol graph, not by inspection.
- **FR-2.** Each domain package MUST define its own dependency struct containing only the fields the matrix proves it uses.
- **FR-3.** A **composition root** MUST construct the full dependency set once and hand each domain its slice.
- **FR-4.** Optional dependencies MUST be modelled explicitly — an interface with a no-op or "unavailable" implementation, or an explicit `Optional[T]`-style wrapper — so a handler declares degraded behaviour rather than nil-checking inline.
- **FR-5.** Endpoints that currently return **501 when a dependency is nil** MUST continue to return 501 under exactly the same conditions. This behaviour is contractual and TD.1-verified.
- **FR-6.** `config.Config` MUST be split into domain-scoped structs (database, auth, storage, AI, billing, messaging, observability, …) composed into a root struct.
- **FR-7.** Environment-variable names, parsing, defaults, and precedence MUST be **unchanged**. The split is structural only.
- **FR-8.** The platform-settings override path (`Deps.effectiveConfig`, `platformstate.Platform`) MUST behave identically after the split.
- **FR-9.** Migration MUST be incremental — domain by domain — with `main` releasable throughout and `Deps` shrinking as domains migrate.
- **FR-10.** A test MUST assert that the full set of environment variables parsed before and after the config split is identical.
- **FR-11.** Startup ordering and failure behaviour MUST be preserved; a missing required setting must fail at the same point with the same message.

## 6. Non-Functional Requirements

- **Performance** — no runtime change; dependencies are wired once at startup.
- **Security** — the JWT signer, password checker, and auth services move between structs. A mis-wired security dependency (e.g. a domain silently receiving a nil signer and skipping verification) is the top risk; FR-4's explicit optionality and fail-closed defaults are the control. Security-relevant dependencies MUST be **required**, never optional.
- **Privacy & Compliance** — config carries secrets (JWT secret, provider keys). The split MUST NOT widen their visibility: a domain that does not need a secret MUST NOT receive it. This is a genuine improvement over the status quo where all 32 fields go everywhere.
- **Accessibility** — n/a.
- **Scalability** — n/a.
- **Reliability** — startup is the risk window. FR-11 and the §16 startup tests are the controls.
- **Observability** — telemetry currently reaches every handler via `Deps`; ensure domains that emit metrics still receive it.
- **Maintainability** — the goal.
- **Internationalization** — locale configuration moves with the rest; no behaviour change.
- **Backward compatibility** — operators must see **zero** change: same env vars, same defaults, same failure messages. This is the story's hardest external constraint.

## 7. Acceptance Criteria

- **AC-1.** *Given* the dependency matrix, *When* reviewed, *Then* every domain has a documented, minimal dependency set derived from the symbol graph.
- **AC-2.** *Given* a migrated domain, *When* its dependency struct is inspected, *Then* it contains strictly fewer fields than the 32-field `Deps`, and each field is used.
- **AC-3.** *Given* a migrated domain's tests, *When* they construct dependencies, *Then* they construct only that domain's struct.
- **AC-4.** *Given* an endpoint whose optional dependency is unconfigured, *When* it is called, *Then* it returns **501 exactly as before** (TD.1 golden verified).
- **AC-5.** *Given* the config split, *When* the FR-10 test runs, *Then* the set of environment variables read is identical before and after.
- **AC-6.** *Given* a deployment with the current production environment, *When* the server starts, *Then* it starts successfully with identical logged configuration.
- **AC-7.** *Given* a missing required setting, *When* the server starts, *Then* it fails at the same point with the same message as before.
- **AC-8.** *Given* the migration is complete, *When* `Deps` is inspected, *Then* it no longer exists as a 32-field god-struct.
- **AC-9.** *Given* a security-relevant dependency, *When* the type is inspected, *Then* it is required (non-optional), so no domain can operate without it.

## 8. Data Model

No schema change. Structural target:

```
server/internal/app/
  app.go            # startup orchestration (unchanged responsibilities)
  wire.go           # composition root: build everything once, hand out slices

server/internal/config/
  config.go         # root struct composing the below
  database.go  auth.go  storage.go  ai.go  billing.go
  messaging.go observability.go  ...

server/internal/httpserver/<domain>/
  deps.go           # that domain's minimal dependency struct
```

## 9. API Surface

**No HTTP API change**, with one contractual behaviour that must be preserved precisely: endpoints returning **501 Not Implemented** when an optional dependency is unconfigured (DRM, storage quota, integrations, bots, scheduler, and others). FR-5 and AC-4 protect this.

No environment-variable or operational surface change (FR-7).

## 10. UI / UX

No UI. Operator-facing surface — env vars, startup logs, failure messages — is explicitly unchanged (§6).

## 11. AI / ML Considerations

AI configuration is a significant slice of the 335 config fields (provider keys, model selection, budgets, BYOK/org credentials) and interacts with `platformstate.Platform` for runtime overrides. The AI config split MUST be coordinated with the AP plan owners, since AP.9 is actively changing this area. Consider sequencing AI config **last** to avoid conflicting with in-flight work.

## 12. Integration Points

- Internal: `internal/httpserver/server.go` — `Deps`, `NewHandler`, `effectiveConfig`, `openRouterClient`.
- Internal: `internal/app/app.go` (440 lines) — dependency construction and startup.
- Internal: `internal/config/config.go` (1,529 lines, 335 fields).
- Internal: `internal/platformstate` — runtime config overrides.
- Internal: every domain package created by TD.6.
- Deploy: `server/.env.example`, `docker-compose*.yml`, `deploy/`, `iac/` — must remain valid unchanged (FR-7).

## 13. Dependencies & Sequencing

- Must ship after: **TD.6** — domain boundaries must exist before per-domain dependency structs can.
- Should coordinate with: **TD.8** (repos take `Querier`, reducing pool plumbing), AP plans (§11).
- Must ship before: nothing hard.
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| A domain receives a nil security dependency and fails open | L | **H** | FR-4 explicit optionality; AC-9 requires security deps to be non-optional; security review of the composition root |
| Config split changes an env-var name or default, breaking deployments | M | **H** | FR-7 + AC-5 mechanical env-var set comparison; AC-6 start against a production-like environment |
| A 501 endpoint starts returning 500 or 200 | M | H | FR-5 + AC-4 TD.1 golden verification for every optional-dependency endpoint |
| Startup ordering changes, causing a subtle race | M | H | FR-11; startup integration tests; no reordering in the same PR as restructuring |
| Conflicts with in-flight AP work on AI config | **H** | M | §11 sequencing — AI config last, coordinated with AP owners |
| Migration stalls half-done, leaving two patterns | M | M | FR-9 incremental with `Deps` shrinking measurably each PR; track remaining field count |
| Secrets accidentally widened rather than narrowed | L | H | §6 review requirement; assert per-domain structs exclude unneeded secrets |

## 15. Rollout Plan

- **Feature flag** — none (compile-time structure).
- **Sequencing** — (1) generate and review the dependency matrix; (2) build the composition root while `Deps` still exists, so both work; (3) migrate domains one at a time, removing fields from `Deps` as they are no longer referenced; (4) split config by area, **excluding AI**; (5) split AI config once AP work settles; (6) delete `Deps`.
- **Dogfood** — deploy each stage to staging with a production-shaped environment; verify startup logs are identical.
- **GA criteria** — `Deps` deleted, config split, two weeks in production with no startup or configuration incident.
- **Rollback** — per-domain revert; the composition root can hand out the old full struct as a fallback during transition.

## 16. Test Plan

- **Unit** — config parsing per domain struct; defaults; precedence with platform overrides; optional-dependency wrappers behave as no-ops.
- **Integration** — server starts with (a) the full production-shaped environment, (b) the minimum viable environment, (c) each optional dependency unconfigured — asserting 501 behaviour per AC-4.
- **End-to-end** — `make e2e` green after each domain.
- **Security** — verify no domain receives a secret it does not need; verify auth dependencies are non-optional; re-run the authz matrix.
- **Accessibility** — n/a.
- **Performance / load** — measure startup time before and after (should be unchanged).
- **Manual exploratory** — deploy to staging with several optional features disabled and confirm the expected 501s and no crashes.

Baseline:

```bash
cd server
awk '/^type Deps struct/,/^}/' internal/httpserver/server.go | grep -cE '^\s+[A-Z]'   # 32
awk '/^type Config struct/,/^}/' internal/config/config.go | grep -cE '^\s+[A-Z]'     # 335
wc -l internal/config/config.go internal/app/app.go                                    # 1529, 440
```

## 17. Documentation & Training

- `docs/ARCHITECTURE_CONVENTIONS.md` (TD.2) — the composition-root pattern; domains declare minimal dependencies; no DI framework.
- `docs/ARCH.md` — updated dependency and configuration diagrams.
- `server/.env.example` — unchanged, but re-verified and annotated by domain grouping.
- Runbook — where each configuration area now lives.

## 18. Open Questions

1. What is the right optionality mechanism — interface + no-op implementation, or an explicit `Optional[T]` wrapper? Prototype both on one domain and pick by readability.
2. Should the composition root live in `internal/app` or a dedicated `internal/wire`?
3. Do event hubs and queues (7 hubs, 4 queues) belong per-domain, or in a shared messaging dependency several domains receive?
4. Can the config split be verified by generating the env-var set from reflection in both versions and diffing? (Would make AC-5 fully mechanical — preferred.)
5. Should `platformstate.Platform` also be decomposed per domain, or stay a single runtime-override service?
6. What is the AP plans' timeline, so AI config can be sequenced without conflict?

## 19. References

- `server/internal/httpserver/server.go:46` — `Deps` (32 fields)
- `server/internal/config/config.go` — 1,529 lines, 335 fields
- `server/internal/app/app.go` — 440 lines, dependency construction
- `server/internal/platformstate` — runtime overrides
- `server/.env.example` — operator contract that must not change
- Related plans: [TD.6](TD.6-decompose-httpserver-package.md), [TD.8](TD.8-querier-abstraction-for-repos.md), [AP — AI providers](../../completed/ai-providers/)
