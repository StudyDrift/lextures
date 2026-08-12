# MC.6 — Markdown → Database Migration & Parity Harness

> Implementation plan. Source: [docs/plan/marketing-content/README.md](README.md) §Architecture
> decision 5.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC.6 |
| **Section** | MC — Marketing Content Platform |
| **Severity** | BLOCKER (for the program) |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING — 5 blog posts, 70 help articles, 16 categories and 1 author exist only as files |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Server platform + Docs/Content |
| **Depends on** | MC.1, MC.2, MC.4, MC.5 |
| **Unblocks** | MC.7, MC.11, MC.15 |

---

## 1. Problem Statement

Everything in this program is theoretical until the existing content is in the database, byte-exact
and rendering identically. The corpus is small (75 articles) but the risk is not: these are pages
that currently return `200` with correct titles, canonicals, JSON-LD, and — after SEO.1–SEO.4 — real
crawlable HTML. A lossy import would silently degrade the entire organic surface. We need a
repeatable, idempotent importer plus a harness that *proves* the DB-sourced build is equivalent to
the file-sourced build before anyone flips a switch.

## 2. Goals

- Import every blog post, help article, help category, author profile and content screenshot into the
  MC.1 schema with zero field loss and no manual data entry.
- Make the import idempotent and re-runnable so it can be rehearsed on staging repeatedly and run
  once for real.
- Prove equivalence: generate the site from files and from the API and diff the outputs — HTML, SEO
  manifest, sitemaps, markdown siblings — with an explainable allowlist of intentional differences.
- Preserve honest `lastmod`: an article's `content_updated_at` comes from its git history, not from
  the import date, so the sitemap does not claim 75 pages changed on the same day (the exact
  regression the `pages-www.yml` workflow already guards against).
- Preserve authorship and review metadata so SEO.3 bylines and help-freshness reporting survive.

## 3. Non-Goals

- No content rewriting, reformatting or quality remediation during import. Articles that fail the
  MC.4 validator are imported with their findings recorded and marked `grandfathered`, exactly as
  today's content-lint report treats them.
- No deletion of the markdown files — that is [MC.15](MC.15-rollout-cutover-and-decommission.md),
  after the parity harness has been green for a full release cycle.
- No import of legal pages, VPAT, glossary/utility pages or comparison pages (they are TypeScript
  data modules, not markdown, and stay file-based).
- No import of course/marketplace content.
- No two-way sync. Once cutover happens, the database is the only source; files become read-only
  history.

## 4. Personas & User Stories

- **As an engineer**, I want to run one command against a database and get the current content
  loaded, so staging rehearsals are cheap and identical to production.
- **As a content expert**, I want my existing articles to show up in the workspace with their
  categories, authors and review dates intact, so I can start editing rather than re-creating.
- **As an SEO owner**, I want proof that no URL, title, canonical, structured-data node or lastmod
  changes as a result of the migration.
- **As a release manager**, I want the migration to be re-runnable and reversible until cutover, so a
  bad import is not a crisis.

## 5. Functional Requirements

- **FR-1.** A command `server/cmd/marketing-content-import` MUST read a content root (default
  `www/src`) and import blog posts (`blog/*.md`), help articles (`docs/**/*.md`), categories
  (`docs/_categories.ts`) and authors (`lib/authors.ts`).
- **FR-2.** The importer MUST parse front matter with the **same** semantics as
  `www/src/utils/blog.ts` and `www/src/utils/docs.ts`, including the array forms (`[a, b]` and
  pipe-joined `citations`), and MUST fail loudly on any key it does not recognise rather than
  dropping it.
- **FR-3.** Field mapping MUST be exhaustive and asserted by test:

  | File front matter | Column |
  |---|---|
  | `title`, `description` | `title`, `description` |
  | `date` | `published_at` (00:00Z on that date), `first_published_at` |
  | `updated` | `content_updated_at` |
  | `author`, `reviewedBy`, `reviewedAt` | `author_slug`, `reviewer_slug`, `reviewed_at` |
  | `pillar`, `briefRef`, `reviewDue` | `pillar`, `brief_ref`, `review_due_on` |
  | `citations` | `citations[]` |
  | `category` (or directory) | `category_id` |
  | `roles`, `segments`, `verifiedAgainst`, `relatedTo` | `roles[]`, `segments[]`, `verified_against`, `related_to[]` |
  | `primaryQuestion`, `cluster`, `keywords` | `primary_question`, `cluster`, `keywords[]` |
  | anything else | **error** (not `extra`) unless explicitly allowlisted |

- **FR-4.** Imported articles MUST be created with `status='published'` and their original
  `published_at`; nothing appears as a draft.
- **FR-5.** `content_updated_at` MUST be resolved by the same precedence `generate-site.mjs` uses for
  `lastmod`: front-matter `updated` → git `log -1 --format=%cI <file>` → `published_at`. The
  importer MUST run git itself and MUST refuse to run in a non-git checkout unless
  `--allow-missing-git` is passed.
- **FR-6.** The importer MUST be idempotent: re-running against the same source updates rows in place
  (matching on `kind`+`locale`+`slug`), creates no duplicates, and produces no revision when nothing
  changed.
- **FR-7.** The importer MUST support `--dry-run` (report only), `--only=blog|docs`, `--slug=<glob>`
  and `--fail-on-validation-error`.
- **FR-8.** Each imported article MUST be validated by MC.4; findings MUST be stored in
  `quality_report` with `grandfathered: true` when the article predates the gate, and the run MUST
  print a summary table by severity.
- **FR-9.** The importer MUST upload every image referenced by an imported article
  (`www/public/docs-*.png` and any other referenced local asset) through MC.5, preserving the known
  dimensions from `LOCAL_IMAGE_DIMENSIONS`, and MUST rewrite the markdown body's image URLs to the
  media URL form. The original markdown files MUST NOT be modified.
- **FR-10.** The importer MUST create the author registry rows from `www/src/lib/authors.ts`
  (including `status` and `knowsAbout`) and MUST fail if any article references an unknown author —
  the same invariant `requireAuthor()` enforces today.
- **FR-11.** The importer MUST create category rows from `_categories.ts` preserving `order` as
  `sort_order` and `platformPath`.
- **FR-12.** A parity harness (`www/scripts/parity-check.mjs`) MUST build the site twice —
  `WWW_CONTENT_SOURCE=files` and `=api` — and diff: every generated `dist/**/index.html`, the
  `.seo-manifest.json`, every `dist/sitemaps/*.xml`, `dist/llms.txt`, `dist/llms-full.txt` and every
  markdown sibling. It MUST exit non-zero on any difference not covered by an explicit allowlist.
- **FR-13.** The allowlist of intentional differences MUST be a checked-in file with a reason per
  entry (expected: image URLs when assets are localised under a checksum path, and nothing else).
- **FR-14.** The importer MUST write a machine-readable report
  (`docs/plan/marketing-content/import-report.json`) with per-article status, score, findings count
  and resolved lastmod, so the migration is auditable after the fact.
- **FR-15.** The importer MUST run inside a single transaction per article and MUST be safe to
  interrupt (partial runs leave the already-imported subset consistent).
- **FR-16.** The importer MUST NOT be reachable over HTTP; it is a command requiring `DATABASE_URL`,
  consistent with `server/cmd/provision-marketplace-courses`.

## 6. Non-Functional Requirements

- **Performance** — Full import of 75 articles + 8 images in < 60 s on a laptop; parity harness
  (two full builds + diff) in < 12 min in CI.
- **Security** — The command requires direct database credentials; it performs no network calls
  except to the media upload service (which it may call in-process). No secrets are written to the
  report.
- **Privacy & Compliance** — Content is already public. The report contains no personal data beyond
  author slugs.
- **Accessibility** — The import must not drop image alt text present in markdown; missing alt text
  is reported as an `error` finding to be fixed by the content team before MC.15.
- **Scalability** — Linear in article count; the harness is the expensive part and runs on demand
  plus nightly, not on every PR.
- **Reliability** — Idempotent and interruptible (FR-6, FR-15). A failed article does not abort the
  run unless `--fail-on-validation-error` is set; failures are listed at the end with non-zero exit.
- **Observability** — Structured log per article (`slug`, `action=created|updated|unchanged`,
  `score`, `findings`, `lastmod`, `lastmodSource`), plus the JSON report.
- **Maintainability** — Front-matter parsing lives in one Go file with the mapping table expressed as
  data, so adding a field is one line plus one test.
- **Internationalization** — All imported content is `locale='en'` and shares a fresh
  `translation_group_id` per article; MC.14 links translations later.
- **Backward compatibility** — Files remain untouched and authoritative until MC.15; the database is
  a shadow copy during this plan.

## 7. Acceptance Criteria

- **AC-1.** *Given* a clean database, *when* the importer runs, *then* 5 blog posts, 70 help
  articles, 16 categories and the author registry exist, and the run reports zero unmapped
  front-matter keys.
- **AC-2.** *Given* an import has run, *when* it runs again unchanged, *then* every article reports
  `unchanged`, no new revisions are created, and the DB row count is identical.
- **AC-3.** *Given* an article whose front matter contains an unknown key `foo:`, *when* the importer
  runs, *then* it exits non-zero naming the file and key.
- **AC-4.** *Given* an article with no `updated` field, *when* imported, *then* `content_updated_at`
  equals its last git commit date and the report records `lastmodSource: "git"`.
- **AC-5.** *Given* the imported database, *when* the site is generated with `WWW_CONTENT_SOURCE=api`
  and compared to the `files` build, *then* every `dist/**/index.html` is byte-identical except for
  allowlisted image URL differences.
- **AC-6.** *Given* both builds, *when* `.seo-manifest.json` is compared, *then* the URL set, titles,
  descriptions, canonicals and `lastmod` values are identical.
- **AC-7.** *Given* both builds, *when* sitemaps are compared, *then* the URL sets and `lastmod`
  values match and no build emits a single-date sitemap (the workflow's stale-lastmod guard).
- **AC-8.** *Given* an article referencing `/docs-course-interface.png`, *when* imported, *then* the
  asset exists in `content_media` with dimensions 1280×720, the body references the media URL, and
  the generated page renders a `<picture>` with AVIF/WebP/PNG and explicit dimensions.
- **AC-9.** *Given* an article whose author is not in the registry, *when* imported, *then* the run
  fails with the file path and the offending slug.
- **AC-10.** *Given* the validation summary, *when* the run finishes, *then* the report lists every
  article's score and no article is silently marked passing.
- **AC-11.** *Given* the importer is interrupted after 30 articles, *when* re-run, *then* it
  completes the remaining articles and reports the first 30 as `unchanged`.
- **AC-12.** *Given* a `--dry-run`, *when* executed, *then* no rows are written and the report is
  still produced.

## 8. Data Model

No new tables. Writes to `marketing.content_articles`, `content_revisions` (one "imported" revision
per article with `change_note='imported from <path>@<git sha>'`), `content_categories`,
`content_authors`, `content_media`, `content_article_media`, `content_known_paths`.

Import provenance is stored in `content_articles.extra`:

```json
{ "import": { "sourcePath": "www/src/docs/courses/finding-your-course.md",
              "gitSha": "dd581282…", "importedAt": "2026-08-20T12:00:00Z",
              "lastmodSource": "git" } }
```

Backfill strategy: this **is** the backfill. Rollback is `DELETE FROM marketing.content_articles
WHERE extra ? 'import'` (safe only before any human edit — see MC.15 §Rollback for the post-edit
policy).

## 9. API Surface

No HTTP surface. Command interface:

```
marketing-content-import
  --content-root www/src          # default
  --database-url $DATABASE_URL    # or env
  --dry-run
  --only blog|docs|media|taxonomy
  --slug '<glob>'
  --fail-on-validation-error
  --allow-missing-git
  --report docs/plan/marketing-content/import-report.json
```

Parity harness:

```
node www/scripts/parity-check.mjs \
  --api-base https://staging.self.lextures.com \
  --allowlist www/scripts/parity-allowlist.json \
  --out www/parity-report.json
```

Exit codes: `0` identical (modulo allowlist), `1` differences found, `2` build failure.

## 10. UI / UX

No UI. Operator experience matters instead:

- The importer prints a progress line per article and a final table: created / updated / unchanged /
  failed, plus a score histogram.
- Failures print the file path, line number where derivable, and the exact fix.
- The parity harness prints the first 20 differing paths with a unified diff of the first difference
  in each, and writes the full report to JSON for CI artefact upload.

## 11. AI / ML Considerations

Not AI-touching. Explicitly: no LLM is used to "improve" or summarise content during migration.
Imported bytes equal source bytes.

## 12. Integration Points

- **Internal modules:** `server/cmd/marketing-content-import` (new),
  `internal/service/marketingcontent` (create/update via the service layer so validation and revision
  rules are identical to the API path), `internal/service/marketingmedia`,
  `internal/repos/marketingcontent`.
- **`www` modules:** `scripts/parity-check.mjs` (new), `scripts/generate-site.mjs`
  (`WWW_CONTENT_SOURCE` switch from MC.7), `src/utils/{blog,docs}.ts` (reference semantics),
  `src/lib/authors.ts`, `src/docs/_categories.ts`.
- **CI:** a new workflow job `content-parity` (nightly + on demand) that runs the harness against
  staging.
- **External:** `git` binary for lastmod resolution.

## 13. Dependencies & Sequencing

- Must ship after: MC.1 (schema), MC.2 (service layer for writes), MC.4 (validation), MC.5 (media).
  The parity harness additionally needs MC.7's `WWW_CONTENT_SOURCE=api` path to exist; land the
  importer first and the harness with MC.7.
- Must ship before: MC.15 (cutover), MC.11 (governance reports need real data), MC.14.
- Shared infra: a staging database and a staging API reachable from CI.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Silent field loss | M | **H** | Unknown keys are a hard error (FR-2/AC-3); mapping table asserted by test; round-trip test regenerates front matter from DB and diffs against the file |
| Rendering differs subtly (typographer, smart quotes) | M | H | Parity harness diffs generated HTML byte-for-byte (AC-5); MC.4's golden corpus covers the renderer itself |
| `lastmod` regresses to "all pages changed today" | M | H | Git-derived lastmod (FR-5, AC-4) and the existing workflow assertion that not all lastmods are identical |
| Import runs against production before rehearsal | L | H | Command refuses to run without `--confirm-production` when the database URL host is not localhost/staging |
| Images break after URL rewriting | M | M | AC-8 asserts the generated `<picture>`; files remain in `www/public` as a fallback until MC.15 |
| Parity harness too slow for CI | M | L | Nightly + on-demand, not per-PR; caches `node_modules` and the previous build |
| Content edited in the DB while files still change | M | M | MC.15 declares a freeze window; until then, re-running the importer overwrites DB edits — the command warns loudly when an article's `revision_no > 1` and skips it unless `--force` |

## 15. Rollout Plan

- **Feature flag:** `ff_marketing_content` (on in staging, off in production). The importer itself is
  not flag-gated — it is a command.
- **Sequencing:** dry-run on staging → real import on staging → parity harness green → content team
  reviews the workspace against the live site → production import during a content freeze (MC.15)
  → `WWW_CONTENT_SOURCE=api` flip (MC.15).
- **Dogfood:** the content team spends a week using the staging workspace on the imported corpus and
  reports anything that reads wrong.
- **GA criteria:** parity harness green three consecutive nightly runs; import report shows zero
  unmapped keys and zero unknown authors; every help screenshot present.
- **Rollback:** before any human edit, delete imported rows (§8) and re-import. After human edits,
  rollback means keeping the DB and flipping `WWW_CONTENT_SOURCE=files` — content edits made in the
  workspace are then unpublished until the flip returns.

## 16. Test Plan

- **Unit** — front-matter parser parity against the TS implementations (fixture-driven, using the
  real 75 files); mapping table exhaustiveness; git lastmod resolution incl. fallbacks; markdown
  image URL rewriting; idempotency hashing.
- **Integration (DB)** — full import into a test database; re-run idempotency; interruption/resume;
  `--dry-run` writes nothing; unknown author aborts; provenance recorded.
- **End-to-end** — the parity harness itself is the e2e: two builds, byte diff, allowlist. Recorded
  in `e2e/coverage/completed-feature-manifest.json` as `coverage: "full"` with the harness as the
  spec.
- **Security** — the command refuses production without confirmation; no secrets in the report; media
  upload path reuses MC.5's validation (a malicious file in `www/public` cannot bypass scanning).
- **Accessibility** — report lists every imported image lacking alt text; target is zero before
  MC.15.
- **Performance / load** — import timing assertion (< 60 s); harness runtime tracked over time.
- **Manual exploratory** — content team spot-checks 10 articles in the workspace against the live
  site: title, byline, dates, images, links, FAQ blocks.

## 17. Documentation & Training

- `docs/plan/marketing-content/import-report.json` committed after the production run as the audit
  record.
- Runbook: "Importing marketing content" — rehearsal steps, flags, expected output, what to do when
  parity fails.
- `www/docs/site-generation.md` — document `WWW_CONTENT_SOURCE` and the parity harness.

## 18. Open Questions

1. Do we import the three top-level docs files that live outside a category directory
   (`creating-a-new-course.md`, `finding-your-course.md`, `navigating-the-course-interface.md`,
   `self-hosting.md`, `connecting-lextures-to-zapier.md`, `using-lextures-with-make.md`) under their
   front-matter `category`, changing their URL? (Proposed: **no URL change** — keep their current
   paths exactly; the path column stores what `www` publishes today.)
2. Should the parity harness run per-PR on `www` changes rather than nightly? (Proposed: nightly plus
   a required run before the MC.15 flip; per-PR is too slow at 12 min.)
3. Do we keep the `import-report.json` in git long-term or attach it to the release? (Proposed:
   commit it once at production import for auditability.)
4. Do imported articles get `reviewer_slug` when only `reviewedBy` exists on some files? (Yes — and
   articles without it are listed for the content team to fill in during MC.11.)

## 19. References

- Files this work touches: `server/cmd/marketing-content-import/*`,
  `www/scripts/parity-check.mjs`, `www/scripts/parity-allowlist.json`,
  `docs/plan/marketing-content/import-report.json`.
- Source semantics: `www/src/utils/blog.ts`, `www/src/utils/docs.ts`, `www/src/lib/authors.ts`,
  `www/src/docs/_categories.ts`, `www/src/lib/markdown.ts` (`LOCAL_IMAGE_DIMENSIONS`),
  `www/scripts/seo-artifacts.mjs` (`resolveLastmod`).
- Precedents: `server/cmd/provision-marketplace-courses`, `server/cmd/hydrate-welcome-course`.
- Related plans: [MC.7](MC.7-www-build-time-content-integration.md),
  [MC.11](MC.11-editorial-workflow-and-governance.md),
  [MC.15](MC.15-rollout-cutover-and-decommission.md).
