# TD.3 — Repair and Verify the OpenAPI Contract

> Implementation plan — **completed 2026-07-25**. Source: technical-debt static analysis. Programme overview: [tech_debt README](../../plan/tech_debt/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | TD.3 |
| **Section** | Technical Debt Remediation |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE (2026-07-25) |
| **Estimated effort** | S (1w) for the repair + guard; M to close documentation coverage |
| **Owner (proposed)** | Backend platform team |
| **Depends on** | TD.1 (route inventory supplies the coverage denominator) |
| **Unblocks** | Typed web client generation; TD.12 |

---

## 1. Problem Statement

`GET /api/openapi.json` **serves syntactically invalid JSON**, and has done since before the current working tree — confirmed by parsing the committed spec at `HEAD`. The document's root object closes prematurely: the first JSON value ends at byte **187,559** of **206,683**, leaving the entire `components` block (security schemes and all reusable schemas, ~19 KB / 9% of the document) as unparseable trailing data. Every consumer is affected: `npm run openapi:types` cannot generate types, `/api/docs` cannot render, and any partner integrating against the published spec fails at parse time. Separately, the spec is a hand-maintained **5,390-line Go string literal** documenting **226 paths against 1,407 registered routes (16%)** — a format that makes this class of defect invisible and inevitable.

This is the one story in the folder that intentionally changes behaviour, because the current behaviour is a defect.

## 2. Goals

- Restore `/api/openapi.json` to a **valid, parseable** OpenAPI 3.0.3 document.
- Add a CI guard making it **structurally impossible** to ship an invalid or non-conformant spec again.
- Restore `npm run openapi:types` and `/api/docs` to working order.
- Measure and publish documentation coverage (documented paths ÷ registered routes) and add a shrink-only ratchet.
- Move the spec off a hand-edited Go string literal onto a format where a syntax error cannot compile or cannot merge.

## 3. Non-Goals

- Documenting all 1,407 routes in this story — the ratchet drives that over time.
- Changing any actual API behaviour, route, or response shape.
- Adopting code-first annotation generation (e.g. `swaggo`) repo-wide — evaluated in Open Questions, not committed here.
- Publishing a public developer portal.

## 4. Personas & User Stories

- **As a frontend engineer**, I want `npm run openapi:types` to succeed, so that I get generated types instead of hand-writing 202 interfaces in `courses-api.ts`.
- **As an integration partner**, I want a parseable spec, so that I can generate a client instead of reverse-engineering endpoints.
- **As a backend engineer**, I want a broken spec to fail my PR, so that I learn about it in seconds rather than after release.
- **As a technical writer**, I want `/api/docs` to render, so that I can review what we publish.
- **As a security reviewer**, I want the spec's `securitySchemes` to actually load, so that auth requirements are visible per endpoint.

## 5. Functional Requirements

- **FR-1.** The system MUST serve a spec at `/api/openapi.json` that parses as valid JSON with **no trailing data**.
- **FR-2.** The served spec MUST validate against the **OpenAPI 3.0.3** schema (structural conformance, not merely valid JSON).
- **FR-3.** A Go test MUST assert FR-1 and FR-2 and MUST run on every PR; this test is the permanent regression guard.
- **FR-4.** The repair MUST restore the orphaned `components` block — including `securitySchemes.bearerAuth` and all reusable schemas — into the document root, with no loss of previously authored content.
- **FR-5.** The spec MUST be relocated out of the Go string literal into a checked-in `.json` (or `.yaml`) file embedded via `go:embed`, so that editors, formatters, and diff tools can validate it and reviewers can read a real diff.
- **FR-6.** A CI check MUST verify every `$ref` in the document resolves.
- **FR-7.** A CI check MUST compute documentation coverage against the TD.1 route inventory and MUST fail if coverage decreases below the checked-in baseline.
- **FR-8.** `npm run openapi:types` MUST succeed against a locally running server and produce compiling TypeScript.
- **FR-9.** `/api/docs` MUST render the spec without console errors.
- **FR-10.** The spec's `info.version` SHOULD be bumped, and the repair noted in `docs/api-changelog-ai-providers.md` or a new general API changelog.
- **FR-11.** Documented paths MUST NOT reference routes that do not exist — a reverse check catching spec drift in the other direction.

## 6. Non-Functional Requirements

- **Performance** — spec served from memory; p95 < 20 ms. Embedding must not slow startup measurably.
- **Security** — the spec is public: it MUST NOT leak internal-only routes, admin-only paths not intended for publication, or example values containing real data. Audit the restored `components` block for this before shipping.
- **Privacy & Compliance** — example payloads MUST use synthetic data only (FERPA: no real learner records in published examples).
- **Accessibility** — `/api/docs` must meet WCAG 2.1 AA to the extent the embedded viewer allows; record any gaps.
- **Scalability** — n/a.
- **Reliability** — the validity test makes an invalid spec unshippable; this is the story's core reliability guarantee.
- **Observability** — CI prints coverage (documented / total) each run.
- **Maintainability** — the spec must be editable without touching Go source; a JSON syntax error must fail CI with a line number.
- **Internationalization** — `description` fields are English-only; note as accepted.
- **Backward compatibility** — the repair is strictly additive from a consumer's perspective (previously unparseable → parseable). No consumer can depend on the broken state.

## 7. Acceptance Criteria

- **AC-1.** *Given* the server is running, *When* `GET /api/openapi.json` is fetched and parsed with a strict JSON parser, *Then* it parses with zero trailing bytes.
- **AC-2.** *Given* the served spec, *When* validated against the OpenAPI 3.0.3 meta-schema, *Then* validation passes with no errors.
- **AC-3.** *Given* the repaired spec, *When* the `components.securitySchemes.bearerAuth` path is read, *Then* it is present and correctly typed — proving the orphaned block was reattached, not discarded.
- **AC-4.** *Given* an engineer introduces a JSON syntax error into the spec file, *When* CI runs, *Then* the build fails naming the line.
- **AC-5.** *Given* a `$ref` pointing at a non-existent schema, *When* CI runs, *Then* the build fails naming the ref.
- **AC-6.** *Given* a running server, *When* `npm run openapi:types` runs, *Then* it exits 0 and the generated file passes `tsc`.
- **AC-7.** *Given* the coverage check, *When* CI runs at baseline, *Then* it reports **226 / 1,407 (16%)** and passes; *When* a PR lowers coverage, *Then* it fails.
- **AC-8.** *Given* the repaired spec, *When* a diff against the pre-repair document's parseable prefix is taken, *Then* no previously documented path or schema is missing (**nothing lost in repair**).
- **AC-9.** *Given* `/api/docs` is loaded in a browser, *When* the page settles, *Then* the spec renders with no console errors.

## 8. Data Model

No database change. File changes:

- `server/internal/openapi/openapi.json` — extracted spec (new).
- `server/internal/openapi/openapi.go` — reduced to `go:embed` + handlers (from 5,390 lines to well under 100).
- `server/internal/openapi/openapi_test.go` — validity, conformance, `$ref`, and coverage tests.
- `scripts/check-openapi-coverage.sh` + `scripts/allowlists/openapi-coverage.txt` — baseline ratchet.

## 9. API Surface

| Route | Verb | Auth | Change |
|---|---|---|---|
| `/api/openapi.json` | GET | public | **Fixed** — now serves valid JSON. Content-Type unchanged (`application/json; charset=utf-8`). |
| `/api/docs` | GET | public | **Fixed** — now renders. |

No other route changes. Response *content* for `/api/openapi.json` changes only by becoming parseable and regaining `components`.

**Rate limiting** — the spec endpoint is public and now genuinely useful; confirm it sits behind the existing rate limiter and consider caching headers (`ETag`, `Cache-Control: public, max-age=300`).

## 10. UI / UX

- `/api/docs` — the existing embedded viewer begins working. Verify: empty state (n/a), loading state (spinner while spec fetches), error state (viewer must show a readable message if the spec 5xxs rather than a blank page).
- Mobile: the docs viewer must not horizontally overflow; wide schema tables scroll within their own container.
- Accessibility: verify heading order and focus handling in the viewer; record deviations in `docs/vpat/` if the third-party viewer cannot meet AA.
- Copy/i18n: spec descriptions are English-only (accepted; noted in §6).

## 11. AI / ML Considerations

Not applicable, except that the spec's `info.description` currently carries AI-provider deprecation notes (AP.9). Preserve that text verbatim during extraction — it is a published deprecation contract.

## 12. Integration Points

- Internal: `server/internal/openapi/openapi.go`, `server/internal/publicapi/openapi_serve.go` (note: `SpecBytes` there is currently **dead code** per `deadcode` — reconcile with TD.4 rather than deleting blindly; it may be the intended public-API serving path).
- Internal: `server/internal/httpserver/server.go` route registration for `/api/openapi.json` and `/api/docs`.
- Web: `clients/web/package.json` → `openapi:types`; `clients/web/src/lib/generated/openapi-types.ts`.
- CI: lint/test workflow.
- Docs: `docs/api-changelog-ai-providers.md`.

## 13. Dependencies & Sequencing

- Must ship after: **TD.1** (route inventory provides the denominator for FR-7).
- Must ship before: any story that wants generated types — notably [TD.12](TD.12-split-courses-api-module.md), which could replace hand-written interfaces with generated ones once this works.
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Repair silently drops previously documented content | M | H | AC-8 diffs the repaired spec against the pre-repair parseable prefix; reviewer checks the extraction commit is a pure move |
| Restored `components` block exposes internal-only endpoints or real example data | M | H | Explicit security/privacy audit of the restored 19 KB before merge (§6); this content has never been publicly parseable, so treat it as unreviewed |
| Consumers built workarounds against the broken endpoint | L | M | Search client code and partner integrations for truncation hacks before shipping; announce in the API changelog |
| Extraction to `go:embed` changes the served bytes (whitespace) | M | L | Test asserts semantic equality (parsed deep-equal), not byte equality |
| Coverage ratchet blocks unrelated PRs that add routes | M | M | Ratchet is on *absolute documented count*, not percentage, so adding an undocumented route does not fail the build — only removing documentation does. Revisit once coverage is materially higher. |
| `publicapi.SpecBytes` is the real intended path and repair happens in the wrong place | M | M | Resolve the two-spec question (Open Question 1) **before** writing the fix |

## 15. Rollout Plan

- **Feature flag** — none; a broken endpoint has no users to protect.
- **Sequencing** — (1) add the failing validity test (red, proving the defect); (2) extract the literal to `openapi.json` as a **pure move**, test still red; (3) repair brace structure, test green; (4) security/privacy audit of restored content; (5) add conformance, `$ref`, and coverage checks; (6) bump `info.version`, changelog entry.
- **Dogfood** — run `npm run openapi:types` and commit the generated types; confirm `tsc -b` passes.
- **GA criteria** — validity test green in CI for one week; `/api/docs` verified manually in Chrome, Firefox, Safari.
- **Rollback** — revert the commit; the endpoint returns to being broken, which is the status quo ante. No data risk.

## 16. Test Plan

- **Unit** — spec parses; no trailing data; validates against OpenAPI 3.0.3 meta-schema; all `$ref`s resolve; `components.securitySchemes.bearerAuth` present.
- **Integration** — `GET /api/openapi.json` through the full middleware chain returns 200, correct `Content-Type`, parseable body; `GET /api/docs` returns 200 HTML.
- **End-to-end** — Playwright spec loading `/api/docs` and asserting the viewer rendered at least one operation and produced no console errors.
- **Security** — manual audit of the restored `components` block for internal-only paths and non-synthetic examples; confirm the endpoint is rate-limited.
- **Accessibility** — axe scan of `/api/docs`; record gaps.
- **Performance / load** — assert p95 < 20 ms for the spec endpoint; confirm `ETag`/caching behaviour if added.
- **Manual exploratory** — generate a client with `openapi-generator` and confirm it builds.

Reproduce the defect today:

```go
// server/internal/openapi/openapi_test.go
dec := json.NewDecoder(strings.NewReader(spec))
var v any
_ = dec.Decode(&v)
// currently: dec.InputOffset() == 187559, len(spec) == 206683  → trailing data
```

## 17. Documentation & Training

- API changelog entry describing the repair and the restored `components` block.
- `AGENTS.md` — document that the spec now lives in `openapi.json` and is embedded, so engineers stop editing Go.
- `docs/ARCHITECTURE_CONVENTIONS.md` (TD.2) — add the rule: "API changes update the spec in the same PR."
- Runbook: how to regenerate web types after an API change.

## 18. Open Questions

1. **Which spec is canonical?** `internal/openapi` serves `/api/openapi.json`, while `internal/publicapi/openapi_serve.go` has an unreachable `SpecBytes`. Are these two specs (internal vs public API) intended to coexist? Resolve before repairing — this determines where the fix lands and whether TD.4 should delete `SpecBytes`.
2. Should the spec move to YAML for reviewability, accepting a build-time conversion, or stay JSON for zero tooling?
3. Is code-first generation (`swaggo` annotations on handlers) worth adopting after [TD.6](../../plan/tech_debt/TD.6-decompose-httpserver-package.md) splits the package into domains? Doing it before the split would embed annotations in files that are about to move.
4. What is the target coverage and by when? 16% today; a credible ratchet needs a destination.
5. Should the coverage ratchet be absolute count or percentage? (Proposed: absolute, per §14.)

## 19. References

- `server/internal/openapi/openapi.json` — embedded LMS OpenAPI 3.0.3 document
- `server/internal/publicapi/openapi_serve.go` — partner Public API OpenAPI 3.1 (separate product surface)
- `clients/web/package.json` — `openapi:types` / `openapi:types:file`
- `docs/api-changelog-ai-providers.md` — deprecation notes + TD.3 repair entry
- OpenAPI 3.0.3 specification — <https://spec.openapis.org/oas/v3.0.3>
- Related plans: [TD.1](TD.1-refactoring-safety-net.md), [TD.4](../../plan/tech_debt/TD.4-delete-confirmed-dead-code.md), [TD.12](../../plan/tech_debt/TD.12-split-courses-api-module.md)

---

## 20. Implementation notes (completed 2026-07-25)

### Root cause

The document was not merely “closed early before `components`.” A **missing path key** for
`/api/v1/settings/permissions` left orphaned `"get"` / `"post"` operations after
`/oneroster/v1p2/users`. The extra closing brace that followed collapsed the `paths` object,
so subsequent path entries became **root-level keys**. The first JSON value therefore ended
before `components`, leaving ~32 KB of trailing data unparseable by strict consumers.

### Repair

1. Restored `"/api/v1/settings/permissions": { ... }` around the orphaned methods.
2. Extracted the literal into `server/internal/openapi/openapi.json` and rewrote
   `openapi.go` to `go:embed` + thin handlers (`Cache-Control: public, max-age=300`).
3. Bumped `info.version` to **0.2.1** and noted the repair in `info.description` and
   `docs/api-changelog-ai-providers.md`.
4. Security/privacy skim of restored `components`: no emails or real PII in examples.

### Guards

| Deliverable | Location |
|---|---|
| Spec file | `server/internal/openapi/openapi.json` |
| Embed + handlers | `server/internal/openapi/openapi.go` |
| Go tests (JSON, structure, bearerAuth, `$ref`, coverage, reverse path check) | `server/internal/openapi/openapi_test.go` |
| Coverage baseline | `scripts/allowlists/openapi-coverage.txt` (`min_documented_paths=252`) |
| Shell + Python check | `scripts/check-openapi-coverage.sh`, `scripts/lib/openapi_check.py` |
| Make target | `make openapi-check` |
| CI | `.github/workflows/ci.yml` step *OpenAPI contract (TD.3)* |
| Generated TS types | `clients/web/src/lib/generated/openapi-types.ts` |
| E2E smoke | `e2e/tests/public-api.spec.ts` (`/api/openapi.json`, `/api/docs`) |

Live coverage at ship: **252 / 1260** unique route patterns (**20.0%**); **302** documented
operations vs **1559** inventory rows. Ratchet is on **absolute documented path count**.

### Open-question resolutions for this ship

1. **Two specs coexist intentionally.** `internal/openapi` = full LMS surface at
   `/api/openapi.json` (3.0.3). `internal/publicapi` = partner Public API at
   `/api/v1/openapi.json` (3.1). `publicapi.SpecBytes` remains for that package’s tests/serving
   path; TD.4 should not delete it as dead without re-checking call graph after this work.
2. **Stay JSON** for zero tooling and `go:embed` simplicity.
3. **Code-first / swaggo deferred** until after TD.6 package splits.
4. **Target coverage:** not fixed in this story; ratchet floor is 252 paths; raise as docs land.
5. **Absolute count** ratchet confirmed (adding undocumented routes does not fail CI).
