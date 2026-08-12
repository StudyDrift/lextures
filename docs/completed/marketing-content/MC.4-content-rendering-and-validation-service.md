# MC.4 — Rendering, Sanitization & Content-Contract Validation

> Implementation plan. Source: [docs/plan/marketing-content/README.md](README.md) §Architecture
> decision 3–4; [www/docs/content-contract.md](../../../www/docs/content-contract.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC.4 |
| **Section** | MC — Marketing Content Platform |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | PARTIAL — a renderer (`www/src/lib/markdown.ts`) and a linter (`www/scripts/content-lint/`) exist, but only at build time on files; nothing validates content written through an API |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform + Docs/Content |
| **Depends on** | MC.1 |
| **Unblocks** | MC.2 (publish gate), MC.6, MC.10, MC.13 |

---

## 1. Problem Statement

The answer-first content contract is currently enforced by a Node linter that runs in `www`'s build
over files in the repo. Once content lives in the database and is written by non-engineers through a
web form, that gate disappears at exactly the moment it matters most: an article can be published to
the public site without a key-takeaways block, with an unknown directive, with a broken internal
link, or with raw HTML. Separately, the editor needs to render a preview, and the in-app help widget
needs sanitized HTML — neither of which can run `www`'s build-time renderer. We need one specified
rendering + validation contract with implementations that provably agree.

## 2. Goals

- Define the markdown/directive dialect once, as a written spec plus a shared golden corpus, and pin
  both implementations to it with tests that fail on divergence.
- Port `www/scripts/content-lint/core.mjs` rules into a server-side validator that runs on every save
  and blocks publish on `error` severity.
- Produce a sanitized HTML rendering path in Go for preview and the in-app help widget, with a strict
  allowlist (no raw HTML, no scripts, no unknown directives).
- Return findings with line numbers and rule ids so the editor can show inline, actionable markers.
- Keep `www/src/lib/markdown.ts` as the renderer of record for the public site — no visual change to
  a single published page.

## 3. Non-Goals

- Not replacing the `www` renderer with server-side HTML for the public site. The static build keeps
  rendering markdown itself (README decision 3).
- Not redesigning the content contract or the scoring formula — this is a port, not a rewrite. Rule
  changes are [MC.11](MC.11-editorial-workflow-and-governance.md) territory.
- Not implementing a WYSIWYG editor or a markdown-to-rich-text bridge (MC.10 uses a markdown editor
  with live preview).
- Not link-checking external URLs on every save (expensive, flaky); external link health is a
  scheduled job in MC.11.
- No AI-based quality scoring.

## 4. Personas & User Stories

- **As a content expert**, I want to see, while writing, that my article is missing an answer block
  or a required FAQ, so I can fix it before asking for review.
- **As a content expert**, I want a live preview that looks exactly like the published page, so I am
  not surprised after publishing.
- **As an editor**, I want publish to be blocked when the contract is violated, so quality does not
  depend on whether anyone remembered to check.
- **As a security engineer**, I want stored markdown to be rendered through a strict allowlist, so a
  compromised author account cannot inject script into `lextures.com` or into the app.
- **As an engineer**, I want one corpus proving the Go and JS renderers agree, so "it looked right in
  the editor but broke on the site" cannot happen silently.

## 5. Functional Requirements

- **FR-1.** A written dialect spec MUST exist at `docs/guides/marketing-content-dialect.md`
  enumerating supported markdown features, the nine `:::` directives currently implemented
  (`key-takeaways`, `answer`, `definition`, `comparison-table`, `steps`, `faq`, `callout`, `stat`,
  `sources`), footnote citations (`[^n]`), image handling and heading-id rules.
- **FR-2.** A shared golden corpus MUST live at `tests/fixtures/content-render/` as pairs of
  `NNN-name.md` + `NNN-name.expected.html`, covering every directive, nesting, malformed input and
  the five existing blog posts.
- **FR-3.** `www` MUST gain a test that renders each corpus input with `renderMarkdown()` and asserts
  byte equality with the expected HTML; the corpus becomes the definition of "correct".
- **FR-4.** The Go renderer (`internal/service/marketingcontent/render`) MUST render the same corpus
  to the same normalized HTML (whitespace-insensitive comparison via a canonicalizer shared by both
  tests), and its test MUST fail if either side drifts.
- **FR-5.** The Go renderer MUST sanitize output with a strict allowlist: no `<script>`, `<style>`,
  `<iframe>`, `on*` attributes, `javascript:`/`data:` URLs (except `data:image/*` in already-stored
  media), and MUST strip any raw HTML present in the source (matching `markdown-it` `html: false`).
- **FR-6.** The validator MUST implement these rule classes, each with a stable id, severity and
  message: front-matter completeness (`fm.*`), structure (`struct.answer-block`,
  `struct.key-takeaways`, `struct.faq-count`, `struct.heading-questions`), passage quality
  (`passage.length`, `passage.self-contained`), citation policy (`cite.numeric-claim`,
  `cite.source-resolvable`), link policy (`link.internal-resolves`, `link.descriptive-anchor`),
  directive policy (`directive.unknown`, `directive.malformed`), and safety (`safety.raw-html`,
  `safety.script-url`).
- **FR-7.** The validator MUST compute the extractability score on the same formula as
  `www/scripts/content-lint/core.mjs`, and a shared score-fixture set MUST prove the two
  implementations produce identical scores (±0.05) for at least 20 documents.
- **FR-8.** Severity mapping MUST be: score `< 6.0` → `error`; `6.0–7.9` → `warn`; `≥ 8.0` → pass.
  `safety.*`, `directive.unknown`, `fm.*` (required fields) and `link.internal-resolves` are always
  `error`.
- **FR-9.** `link.internal-resolves` MUST validate internal links against the union of (a) published
  article paths in the database, (b) the static route list published by `www`
  (`dist/.seo-manifest.json` mirrored into a `marketing.content_known_paths` table refreshed by the
  build), and MUST flag unknown paths as `error`.
- **FR-10.** The validator MUST be callable as `POST /api/v1/admin/marketing/lint` (MC.2 FR-2) with
  `{ kind, bodyMd, metadata }`, returning `{ score, findings[], stats }` without persisting anything.
- **FR-11.** Every article write MUST store the resulting `quality_score` and `quality_report` on the
  row so the workspace can filter and report without recomputation.
- **FR-12.** The renderer MUST expose `PlainText(bodyMd)` producing a normalized text extraction used
  by search snippets (MC.13) and `llms-full.txt` (MC.12).
- **FR-13.** Rendering MUST be bounded: 40 KB input must render in < 50 ms p95, and inputs above the
  1 MB cap are rejected before rendering.
- **FR-14.** Findings MUST carry `line` and `column` where derivable, and `path` (front-matter field
  name) otherwise.
- **FR-15.** The corpus test MUST run in both CI jobs (`www` unit tests and `go test`), so neither
  language can merge a drift-inducing change alone.

## 6. Non-Functional Requirements

- **Performance** — Validate + render a 40 KB article in < 80 ms combined p95; no per-save network
  calls. Link resolution uses an in-memory path set refreshed at most every 60 s.
- **Security** — This is the primary XSS boundary for content authored by humans and rendered in two
  origins. Output sanitization is allowlist-based and unit-tested against an injection corpus
  (`javascript:` URLs, `onerror` attributes, HTML comments, nested code fences, unicode escapes).
  Any sanitizer change requires a security review per `SECURITY.md`.
- **Privacy & Compliance** — No PII processing. Rendered output must preserve author attribution
  markup required by SEO.3 byline policy.
- **Accessibility** — Rendered HTML must keep the accessibility affordances the current renderer
  provides: `id` + `tabindex="-1"` on `h2`–`h4`, `aria-labelledby` on takeaways/FAQ, `figcaption` on
  stats, `alt` preserved on images, and external links marked `rel="noopener noreferrer"`. A rule
  (`a11y.image-alt`) MUST flag images without alt text as `error`.
- **Scalability** — Pure functions; no shared mutable state except the cached path set.
- **Reliability** — A validator panic must never take down a save: the service recovers, records
  `validator_error`, and treats the article as `warn` (never silently "pass") — publish then requires
  an explicit override.
- **Observability** — `marketing_content_lint_total{severity}`,
  `marketing_content_lint_duration_seconds`, `marketing_content_render_errors_total`; log the rule
  ids that blocked a publish.
- **Maintainability** — Rules are table-driven (`[]Rule{ID, Severity, Check}`) so adding a rule is
  one entry plus one test. The Go package must not import anything from `www`.
- **Internationalization** — Rule messages are English; the client maps `rule` ids to i18n keys.
  Word-count-based rules must use a grapheme-aware counter so CJK locales (MC.14) are not
  systematically failed; locale-specific thresholds are a documented MC.14 follow-up.
- **Backward compatibility** — The corpus is generated from today's renderer output, so the first
  commit proves parity with production HTML. Any intentional rendering change must update the corpus
  in the same commit and note it in MC.12's SEO regression checklist.

## 7. Acceptance Criteria

- **AC-1.** *Given* the golden corpus, *when* the `www` test runs, *then* every input renders
  byte-identically to its expected HTML.
- **AC-2.** *Given* the same corpus, *when* the Go test runs, *then* canonicalized output matches the
  same expected HTML for every case.
- **AC-3.** *Given* a change to `www/src/lib/markdown.ts` that alters output, *when* CI runs, *then*
  both the `www` and Go corpus tests fail until the corpus is updated.
- **AC-4.** *Given* markdown containing `<script>alert(1)</script>` and `[x](javascript:alert(1))`,
  *when* rendered by the Go renderer, *then* the output contains neither, and the validator reports
  `safety.raw-html` and `safety.script-url` as `error`.
- **AC-5.** *Given* the five existing blog posts, *when* validated, *then* their computed scores
  match `npm run content:score` for the same files within ±0.05.
- **AC-6.** *Given* an article missing the `:::answer` block, *when* validated as `kind='blog'`,
  *then* `struct.answer-block` is reported and the score is below the publish floor.
- **AC-7.** *Given* an internal link to `/docs/does-not-exist`, *when* validated, *then*
  `link.internal-resolves` is `error` with the offending line number.
- **AC-8.** *Given* an image without alt text, *when* validated, *then* `a11y.image-alt` is `error`.
- **AC-9.** *Given* a 40 KB article, *when* validate + render runs, *then* p95 < 80 ms over 100 runs.
- **AC-10.** *Given* a validator panic injected in a test, *when* a save occurs, *then* the save
  succeeds, `quality_report.validatorError` is true, and a subsequent publish requires override.
- **AC-11.** *Given* `POST /lint` with a body and metadata, *when* called, *then* nothing is
  persisted and the response contains score, findings and stats (word count, passage lengths,
  heading count, FAQ count).
- **AC-12.** *Given* an unknown directive `:::wat`, *when* validated, *then* `directive.unknown` is
  `error` and the renderer emits the raw text escaped, never HTML.

## 8. Data Model

- No new article columns (MC.1 already declares `quality_score`, `quality_report`).
- New table for link resolution:

```sql
CREATE TABLE marketing.content_known_paths (
    path        TEXT PRIMARY KEY,
    source      TEXT NOT NULL CHECK (source IN ('article', 'static_route')),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Populated for `article` rows by triggers/service on publish, and for `static_route` rows by a small
`POST /api/v1/admin/marketing/known-paths` call the `www` deploy job makes with its
`.seo-manifest.json` route list (permission `…:admin`, service token). Migration
`479_marketing_content_known_paths.sql` (indicative number). Backfill: the first `www` deploy after
the endpoint exists populates static routes; article paths are backfilled by MC.6.

## 9. API Surface

- `POST /api/v1/admin/marketing/lint` — defined in MC.2 FR-2; body `{ kind, bodyMd, metadata }`,
  response `{ score: number, findings: Finding[], stats: RenderStats }`.
- `POST /api/v1/admin/marketing/known-paths` — `{ paths: string[] }`, permission `…:admin`, replaces
  the `static_route` set atomically; rate-limited to 10/min.
- Internal Go surface:

```go
package render // internal/service/marketingcontent/render
func HTML(md string) (string, error)          // sanitized
func PlainText(md string) string
func Stats(md string) RenderStats

package validate // internal/service/marketingcontent/validate
func Article(in Input) Report                 // Input{Kind, BodyMD, Metadata, KnownPaths}
type Report struct { Score float64; Findings []Finding; Stats render.RenderStats; ValidatorError bool }
```

- Rate limits: `/lint` 120/min/user (it runs on save and on demand).
- OpenAPI: both routes documented.

## 10. UI / UX

Consumed by MC.10; this plan specifies the contract the UI relies on:

- Findings render as inline gutter markers (error = destructive token, warn = warning token) plus a
  right-rail list grouped by severity, each with rule id, message and a "learn more" link into the
  dialect guide.
- The score renders as a labelled meter with the floor marked; colour is never the only signal
  (numeric value + text label), satisfying WCAG 1.4.1.
- Preview pane renders the sanitized HTML with the site's content styles so authors see the true
  result; a "preview differs from site" banner appears if the corpus version stored with the article
  is older than the current renderer version.
- Copy/i18n keys: `marketingContent.lint.rule.<id>`, `.severity.<level>`, `.score.label`,
  `.score.floor`.

## 11. AI / ML Considerations

No model is used. Deliberate: an LLM scorer would be non-deterministic and unauditable at a publish
gate. If AI-assisted *suggestions* are added later they must be advisory only — never able to change
a score or unblock a publish — and must be disclosed through `internal/aidisclosure`.

## 12. Integration Points

- **Internal modules:** `internal/service/marketingcontent/render`, `…/validate`,
  `internal/repos/marketingcontent` (known paths, quality columns), `internal/httpserver`
  (lint route, known-paths route).
- **`www` modules:** `www/src/lib/markdown.ts` (unchanged behaviour, new corpus test),
  `www/scripts/content-lint/core.mjs` (rule ids aligned; remains the file-based linter until MC.15),
  `www/scripts/generate-site.mjs` (posts the route list to `/known-paths` after a successful deploy).
- **Shared fixtures:** `tests/fixtures/content-render/` consumed by both test suites.
- **External standards:** CommonMark, GFM tables, WCAG 2.1 AA, OWASP XSS prevention cheat sheet.

## 13. Dependencies & Sequencing

- Must ship after: MC.1 (columns to store reports).
- Must ship before: MC.2's publish gate becomes real, MC.6 (importer validates every imported
  article), MC.10 (editor UX), MC.13 (plain-text extraction for search).
- Shared infra: none. Go markdown library choice (`goldmark` + custom directive extension) is an
  implementation decision made in this plan.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Two renderers drift over time | **H** | H | Golden corpus enforced in both CI jobs (FR-15, AC-3); the corpus is the spec, and PRs that change either renderer must update it |
| `goldmark` and `markdown-it` differ on edge cases (typographer, linkify, nested emphasis) | H | M | Corpus is generated *from* `markdown-it` output; Go implementation configures goldmark to match (smartypants, linkify, no raw HTML) and any irreconcilable case is documented as a corpus exclusion with a lint rule banning that syntax |
| Ported scoring formula subtly differs | M | M | 20-document score-fixture parity test (FR-7, AC-5) with tolerance ±0.05 |
| Publish gate blocks legitimate urgent content | M | M | Documented override with justification, recorded in the audit log and surfaced in MC.11's governance report |
| Sanitizer over-strips legitimate content (e.g. tables) | M | M | Injection corpus paired with a "must survive" corpus; both run on every change |
| Link resolution false-positives after a `www` route change | M | M | Static route set refreshed on every deploy; unknown-path findings include "route list last refreshed at" so authors can tell |

## 15. Rollout Plan

- **Feature flag:** covered by `ff_marketing_content`; additionally, the publish gate honours
  `marketing_content_lint_enforce` (platform setting, default **on** once MC.6's imported corpus
  passes) so the gate can be relaxed without disabling the feature.
- **Sequencing:** dialect spec → corpus generation from current output → `www` corpus test (proves
  no change) → Go renderer + sanitizer → validator rules → lint route → wire MC.2 publish gate.
- **Dogfood:** run the validator over all 75 imported articles and publish the report; fix or
  grandfather each failure before the gate is enforced (`grandfathered` flag in `quality_report`,
  same concept as today's content-lint report).
- **GA criteria:** corpus green in both CI jobs; ≥ 95% of imported articles at or above the warn
  threshold; zero `safety.*` findings on imported content.
- **Rollback:** set `marketing_content_lint_enforce=false` (gate becomes advisory). Renderer rollback
  is a code revert; the corpus prevents silent output changes either way.

## 16. Test Plan

- **Unit (Go)** — each rule: positive, negative and boundary; sanitizer injection corpus; plain-text
  extraction; stats; panic recovery path.
- **Unit (JS)** — corpus render equality; existing `content-lint/core.test.mjs` extended with the
  shared score fixtures.
- **Integration** — `POST /lint` end-to-end with auth and rate limits; known-paths refresh and its
  effect on `link.internal-resolves`; article save persisting `quality_report`.
- **End-to-end** — deferred to MC.10 (editor shows findings); this plan's e2e contribution is the
  CI-level corpus job.
- **Security** — XSS corpus (≥ 40 vectors) asserting no executable output; fuzzing the directive
  parser with `go-fuzz`/`testing.F` for panics and quadratic blowup; verify no `data:` URL except
  stored images.
- **Accessibility** — automated axe run over rendered corpus HTML in a headless page; assert heading
  ids, alt text and `rel` attributes survive rendering.
- **Performance / load** — benchmark render+validate across the corpus; assert p95 budgets (AC-9);
  regression threshold in CI at +25%.
- **Manual exploratory** — content team writes one new article in staging and confirms every finding
  is understandable and actionable.

## 17. Documentation & Training

- New: `docs/guides/marketing-content-dialect.md` (the spec) and a rule reference table with examples
  of pass/fail for each rule id.
- Update: `www/docs/content-contract.md` to point at the shared spec and note that the gate now runs
  server-side.
- Training: a 30-minute walkthrough for the content team on the score, the findings list and the
  override policy.

## 18. Open Questions

1. Which Go markdown library — `goldmark` (extensible, well-maintained) vs a hand-rolled parser for
   exact `markdown-it` parity? (Proposed: goldmark with a custom directive extension; the corpus
   arbitrates.)
2. Should the corpus live at `tests/fixtures/content-render/` or inside `www/scripts/__fixtures__/`?
   (Proposed: repo root `tests/` so neither language "owns" it.)
3. Do we keep `www/scripts/content-lint` after MC.15, for content still authored as files
   (e.g. legal pages)? (Proposed: yes — legal/utility pages stay file-based; the linter keeps
   covering them.)
4. Should the score floor differ for `doc` vs `blog`? (Today's contract applies one floor; help
   articles are shorter and may need a separate rubric — decide with the content team in MC.11.)

## 19. References

- Files this work touches: `www/src/lib/markdown.ts`, `www/scripts/content-lint/core.mjs`,
  `www/scripts/generate-site.mjs`, `server/internal/service/marketingcontent/render/*`,
  `…/validate/*`, `tests/fixtures/content-render/*`.
- Standards: CommonMark 0.31, GFM tables, WCAG 2.1 AA (1.4.1, 2.4.4, 4.1.2), OWASP XSS prevention.
- Related plans: [MC.2](MC.2-authoring-api-and-revisions.md),
  [MC.6](MC.6-markdown-to-database-migration.md), [MC.10](MC.10-article-editor.md),
  [MC.11](MC.11-editorial-workflow-and-governance.md), [MC.12](MC.12-seo-parity-from-database.md).
