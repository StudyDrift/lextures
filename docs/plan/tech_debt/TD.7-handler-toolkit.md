# TD.7 — Handler Toolkit: Typed I/O, Guards & Error Mapping

> Implementation plan. Source: technical-debt static analysis, 2026-07-25. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | TD.7 |
| **Section** | Technical Debt Remediation |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS (internal) |
| **Status (today)** | THIN |
| **Estimated effort** | M (2–4w) for the toolkit; adoption is continuous |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | TD.1, TD.5 |
| **Unblocks** | Faster, safer handler work across all domains; supports TD.3 coverage |

---

## 1. Problem Statement

Every handler in `internal/httpserver` hand-rolls the same five steps: decode JSON, validate, authorise, call a repo, encode JSON. The result is **7,280 `apierr.WriteJSON` call sites** — each an opportunity to return the wrong status code, an inconsistent error body, or a message that leaks internals. `requireCourseAccess` is invoked **329 times**, each followed by bespoke permission logic. Because the pattern is copied rather than shared, error semantics drift between endpoints, response envelopes are inconsistent, and there is no central place to enforce cross-cutting concerns like request-size limits, content-type checking, or structured validation errors. A handler that should be 15 lines of domain logic is 60 lines of ceremony — which is also why the package is 125K lines.

## 2. Goals

- Provide a small, idiomatic Go toolkit that collapses decode/validate/authorise/encode into declarative helpers.
- Make the **correct** thing the **easy** thing: consistent status codes, consistent error envelopes, no internal detail leakage.
- Centralise error mapping so a repo error (`pgx.ErrNoRows`, constraint violation, context cancellation) becomes the right HTTP status in one place instead of 7,280.
- Reduce handler length materially, making TD.6's packages genuinely readable.
- Improve the odds that OpenAPI documentation (TD.3) can be derived rather than hand-written.

## 3. Non-Goals

- Rewriting all 1,407 handlers in this story — the toolkit ships, adoption is incremental and continuous.
- Introducing a web framework or replacing `chi`.
- Adding runtime reflection-heavy magic. Go handlers should stay readable and debuggable; generics and small helpers, not a DI container.
- Changing any endpoint's observable contract during toolkit introduction.
- Replacing `apierr` — the toolkit builds on it.

## 4. Personas & User Stories

- **As a backend engineer**, I want to write a handler as "decode this type, check this permission, call this repo", so that I spend my time on domain logic.
- **As an API consumer**, I want every error to have the same envelope and sensible status, so that my client can handle failures generically.
- **As a security reviewer**, I want one place where errors become responses, so that I can verify no internal detail leaks.
- **As a reviewer**, I want handlers short enough that a missing authorisation check is visible.
- **As a technical writer**, I want request/response types to be real Go types, so that documentation can be generated from them.

## 5. Functional Requirements

- **FR-1.** The toolkit MUST provide typed request decoding with a bounded body size, strict content-type checking, and unknown-field policy consistent with current behaviour.
- **FR-2.** The toolkit MUST provide typed response encoding that sets `Content-Type` and status consistently.
- **FR-3.** The toolkit MUST provide a **central error mapper** translating domain and infrastructure errors to HTTP status + `apierr` code, covering at minimum: not-found, validation failure, permission denied, conflict/constraint violation, upstream timeout, and context cancellation.
- **FR-4.** The error mapper MUST NOT leak internal messages, SQL text, or stack traces to clients; it MUST log the internal detail with the request/trace ID while returning a safe message.
- **FR-5.** The toolkit MUST provide composable **authorisation guards** wrapping the existing `requireCourseAccess` family, so a handler declares its requirement rather than re-implementing the check.
- **FR-6.** Guards MUST **fail closed** — a handler that declares no guard MUST NOT silently become public. The design MUST make an unguarded route detectable (see FR-11).
- **FR-7.** The toolkit MUST preserve existing response envelopes exactly; adoption of an endpoint MUST NOT change its output (verified against TD.1 characterization goldens).
- **FR-8.** The toolkit MUST support the existing observability chain — trace IDs, access logs, Sentry — without handlers doing anything special.
- **FR-9.** Adoption MUST be **incremental and opt-in per handler**; the toolkit and hand-rolled handlers MUST coexist indefinitely without a flag day.
- **FR-10.** The toolkit MUST provide a validation entry point producing structured, field-level errors in the format the web client already expects.
- **FR-11.** A CI check SHOULD report the count of routes with no declared guard, as a shrink-only ratchet.
- **FR-12.** The toolkit SHOULD expose request/response types in a form usable for OpenAPI generation, to support TD.3's coverage goal.

## 6. Non-Functional Requirements

- **Performance** — no more than one extra allocation per request versus hand-rolled code; benchmark decode/encode paths. Reject a design that adds measurable p95 latency.
- **Security** — this story concentrates authorisation and error handling into shared code. That is a large security win *and* a single point of failure: the toolkit's guard and error-mapping code requires security review and its own test suite. Body-size limits and content-type checks MUST match or tighten current behaviour, never loosen it.
- **Privacy & Compliance** — FR-4 is a privacy control: error responses must not echo learner data. Log redaction must continue to work (`internal/logging` redaction path).
- **Accessibility** — n/a directly; structured field-level errors (FR-10) enable accessible client-side error announcement.
- **Scalability** — n/a.
- **Reliability** — a toolkit bug affects every adopting endpoint. Mitigated by incremental adoption (FR-9) and per-endpoint golden verification (FR-7).
- **Observability** — errors mapped centrally SHOULD emit a metric labelled by code and route class, giving the first coherent view of API error rates.
- **Maintainability** — the story's purpose.
- **Internationalization** — user-facing error messages should route through the existing message layer; the toolkit must not hard-code English in a way that blocks future localisation. Note current messages are English sentences (`ST1005` is excluded in `.golangci.yml` for this reason).
- **Backward compatibility** — absolute. An adopted endpoint is byte-identical to its pre-adoption self.

## 7. Acceptance Criteria

- **AC-1.** *Given* a handler converted to the toolkit, *When* the TD.1 characterization suite runs, *Then* status, headers, and JSON key set are unchanged.
- **AC-2.** *Given* a repo returning `pgx.ErrNoRows`, *When* the error reaches the mapper, *Then* the client receives 404 with the standard envelope and the internal error is logged with the trace ID.
- **AC-3.** *Given* a handler whose repo returns a constraint-violation error, *When* mapped, *Then* the client receives 409 (not 500) and no SQL text appears in the response.
- **AC-4.** *Given* a request body exceeding the configured limit, *When* decoded, *Then* the client receives 413 and the connection is not exhausted reading the body.
- **AC-5.** *Given* a handler declaring a course-staff guard, *When* a learner calls it, *Then* the response is 403 with the same envelope the hand-rolled check produced.
- **AC-6.** *Given* a route registered with no guard declared, *When* the FR-11 check runs, *Then* it is reported in the unguarded-route count.
- **AC-7.** *Given* the toolkit's decode path, *When* benchmarked against the hand-rolled equivalent, *Then* latency and allocation overhead are within the §6 budget.
- **AC-8.** *Given* a converted handler, *When* its line count is compared to the original, *Then* it is materially shorter with no logic removed.
- **AC-9.** *Given* an invalid payload, *When* validation fails, *Then* the client receives field-level errors in the existing format the web client parses.

## 8. Data Model

No schema change. New code:

```
server/internal/httpserver/kernel/     # co-located with TD.6's kernel
  decode.go        # typed request decoding, size/content-type limits
  render.go        # typed response encoding
  errmap.go        # central error → status/code mapping
  guards.go        # composable authorisation guards
  validate.go      # structured field-level validation
```

If TD.6 has not yet created `kernel/`, this story creates it and TD.6 adopts it.

## 9. API Surface

**No API change on adoption** (FR-7). The toolkit changes how handlers are written, not what they return.

Sketch of the target shape:

```go
// before: ~60 lines — method dispatch, decode, permission check, lookup, encode, 6 error branches
// after:
var CreateOutcome = kernel.POST(
    kernel.RequireCoursePermission("item:create"),
    func(ctx kernel.Ctx, in CreateOutcomeRequest) (CreateOutcomeResponse, error) {
        row, err := courseoutcomes.InsertOutcome(ctx, ctx.DB(), ctx.CourseID(), in.Title, in.Description)
        if err != nil {
            return CreateOutcomeResponse{}, err   // mapped centrally
        }
        return toAPI(row), nil
    },
)
```

The exact signature is a design decision for the implementing team; the requirement is that decode, authorise, and error-mapping are declarative and that the wire contract is unchanged.

**Rate limiting** is unaffected — it stays in middleware.

## 10. UI / UX

No UI. The web client benefits indirectly from consistent error envelopes; TD.11's shared `readApiErrorMessage` becomes more reliable when every endpoint agrees on the format.

## 11. AI / ML Considerations

AI-backed endpoints (grading agent, adaptive content, tutor session) have distinctive failure modes — provider timeouts, rate limits, content-filter rejections, budget exhaustion. The error mapper SHOULD include a documented mapping for these so AI endpoints stop inventing their own status codes. Coordinate with the AP plan owners on the canonical mapping (e.g. provider 429 → 503 with `Retry-After`, budget exceeded → 402).

## 12. Integration Points

- Internal: `internal/apierr/apierr.go` — the toolkit builds on it; may need extension for structured field errors.
- Internal: `internal/httpserver/kernel/` (new or TD.6's).
- Internal: `internal/courseroles` — `UserHasPermission`, wrapped by guards.
- Internal: `internal/logging` — redaction and access logging.
- Internal: `internal/telemetry` — error metrics (§6).
- Web: `clients/web/src/lib/errors.ts` (`readApiErrorMessage`) — must keep parsing every envelope.

## 13. Dependencies & Sequencing

- Must ship after: **TD.1** (goldens verify each conversion), **TD.5** (do not build a toolkit around dispatch code that is being deleted).
- Runs alongside: **TD.6** — the toolkit lives in the kernel; conversions happen per domain after each domain moves.
- Must ship before: nothing hard, but earlier adoption compounds — every new handler written without it is future conversion work.
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Central guard bug exposes data across many endpoints at once | L | **H** | Dedicated security review and test suite for `guards.go`; fail-closed design (FR-6); incremental adoption limits initial blast radius |
| Error mapper changes a status code an existing client depends on | M | H | FR-7 + AC-1 per-endpoint golden verification; convert one endpoint at a time |
| Toolkit becomes over-abstracted and less readable than what it replaced | M | H | Keep it small (5 files); reject reflection-heavy designs; require the sketch in §9 to stay legible to a Go engineer with no toolkit knowledge |
| Two styles coexist forever, adding a *second* pattern instead of replacing the first | **H** | M | Convention (TD.2) mandates the toolkit for new handlers; FR-11 ratchet tracks adoption; accept long coexistence but never allow new hand-rolled handlers |
| Error mapping hides a real 500 as a 4xx, masking incidents | M | M | Mapper emits metrics by code (§6); unmapped errors default to 500 and are logged loudly, never silently downgraded |
| Performance regression from generics/allocation | L | M | AC-7 benchmark gate |

## 15. Rollout Plan

- **Feature flag** — none; adoption is per-handler at compile time.
- **Sequencing** — (1) build `errmap.go` and adopt it *alone* across a handful of handlers, verifying goldens (this is the highest-value, lowest-risk piece); (2) add decode/render; (3) add guards, with security review; (4) add validation; (5) convert handlers opportunistically, domain by domain, as TD.6 lands each package.
- **Dogfood** — first conversions target a low-traffic, well-tested domain; hold a review before broad adoption.
- **GA criteria** — toolkit stable for four weeks; at least one full domain converted; no golden diffs attributable to the toolkit.
- **Rollback** — revert individual conversions; the toolkit itself is additive and can sit unused.

## 16. Test Plan

- **Unit** — error mapper table tests for every error class; decode limits (size, content-type, malformed JSON); render envelope; validation output format.
- **Integration** — converted handlers exercised end-to-end against a live database; compare against TD.1 goldens.
- **End-to-end** — `make e2e` green after each conversion batch.
- **Security** — dedicated guard test suite: every guard denies by default, denies the wrong role, allows the right role; fuzz the decoder with malformed and oversized bodies; assert no internal error text ever appears in a response body (property test over the mapper).
- **Accessibility** — verify field-level errors reach the client in the shape the SPA needs for accessible error announcement.
- **Performance / load** — benchmark decode/render/map against hand-rolled equivalents; enforce the §6 budget in CI.
- **Manual exploratory** — deliberately misuse converted endpoints (wrong role, malformed body, oversized payload) and confirm responses are sane and leak-free.

Baseline:

```bash
cd server
grep -rho 'apierr\.WriteJSON(' internal/httpserver/*.go | wc -l    # 7280
grep -rho 'requireCourseAccess' internal/httpserver/*.go | wc -l   # 329
```

## 17. Documentation & Training

- `docs/ARCHITECTURE_CONVENTIONS.md` (TD.2) — "new handlers use the toolkit" with a worked before/after example.
- Toolkit package doc comment as the primary reference — Go engineers read `godoc`, not wikis.
- Error-mapping table published for API consumers and for TD.3's spec.
- Team session with a live conversion of a real handler.

## 18. Open Questions

1. What exactly is the current error envelope? `apierr.WriteJSON` is used 7,280 times but the envelope must be confirmed uniform before centralising — sample widely first.
2. Does the web client depend on any endpoint-specific error shape that a uniform mapper would break? Audit `clients/web/src/lib/errors.ts` and its callers.
3. Should guards be generic over resource type (course, org, module) or a small closed set of concrete guards? (Leaning concrete — fewer type gymnastics, clearer failures.)
4. Is `unknown field` currently rejected or ignored on decode? Behaviour must be preserved exactly per endpoint; this may not be uniform today.
5. Should the toolkit own OpenAPI generation (FR-12) or merely enable it? Coupling them risks scope creep.
6. What is the canonical AI-provider error mapping (§11)? Needs AP plan owner input.

## 19. References

- `server/internal/apierr/apierr.go` — existing error writer
- `server/internal/httpserver/course_outcomes.go:334` — representative hand-rolled handler
- `server/internal/courseroles` — permission checks wrapped by guards
- `clients/web/src/lib/errors.ts` — client-side envelope parsing
- Related plans: [TD.1](../../completed/tech_debt/TD.1-refactoring-safety-net.md), [TD.5](TD.5-remove-unreachable-method-dispatch.md), [TD.6](TD.6-decompose-httpserver-package.md), [TD.3](../../completed/tech_debt/TD.3-repair-and-verify-openapi-contract.md)
