# MC.15 — Rollout, Cutover & Decommission of File-Based Content

> Implementation plan. Source: [docs/plan/marketing-content/README.md](README.md) §Architecture
> decision 5.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC.15 |
| **Section** | MC — Marketing Content Platform |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | MISSING — no cutover has been planned; `www/src/blog` and `www/src/docs` are the live source |
| **Estimated effort** | S (1w of execution, spread over a 3-week window) |
| **Owner (proposed)** | Web platform + Docs/Content (release manager accountable) |
| **Depends on** | MC.1–MC.14 (MC.14 optional) |
| **Unblocks** | — (program complete) |

---

## 1. Problem Statement

At the end of MC.14 the platform can author, validate, publish, render and govern content from the
database — but production still builds from files, the feature flag is off, nobody has permission,
and 75 markdown files remain the real source. A migration that stops here is worse than not starting:
two sources of truth, an unused workspace, and a growing chance someone edits the wrong one. This
plan performs the switch, deletes the old path, and leaves behind the documentation, permissions and
runbooks the content team needs to own the system.

## 2. Goals

- Flip production to database-sourced content with a rehearsed, reversible sequence and a content
  freeze that is measured in hours, not days.
- Verify the live site after cutover against an explicit checklist — URLs, structured data, sitemaps,
  feeds, search, redirects — not by vibe.
- Remove the file-based content path entirely: markdown directories, the globs that read them, and
  the scripts whose scope they were.
- Grant the content team the permissions and training to operate independently, and record who owns
  what.
- Leave CI guardrails that prevent regression to file-based content or silent loss of the DB path.

## 3. Non-Goals

- No new features. This plan ships zero user-visible capability that MC.1–MC.14 did not.
- No removal of file-based *non-article* pages (legal, VPAT, glossary, comparisons, templates,
  standards, integrations) — they stay in TypeScript modules and keep their own linting.
- No change to the marketing site's design or IA.
- No data migration beyond re-running MC.6's importer for the final freeze.

## 4. Personas & User Stories

- **As a release manager**, I want a written sequence with go/no-go gates and a rollback that takes
  minutes, so cutover is boring.
- **As a content expert**, I want to know exactly when to stop editing files and start editing in the
  workspace, and to have training before that moment.
- **As an engineer**, I want the old path removed so nobody wastes time wondering which source is
  live.
- **As an SEO owner**, I want a post-cutover verification pass and a two-week watch window on Search
  Console.
- **As a future maintainer**, I want CI to fail if someone re-adds a markdown article to `www/src`.

## 5. Functional Requirements

- **FR-1.** A **content freeze** MUST be declared for the cutover window: no merges touching
  `www/src/blog` or `www/src/docs`, announced 72 hours ahead, enforced by a CODEOWNERS/branch rule
  during the window.
- **FR-2.** The MC.6 importer MUST be re-run against production immediately before the flip, with
  `--fail-on-validation-error` off but the report reviewed, and the resulting `import-report.json`
  committed as the audit record.
- **FR-3.** The MC.6 parity harness MUST be green against production data within the freeze window;
  a non-allowlisted difference is a **no-go**.
- **FR-4.** `ff_marketing_content` MUST be enabled in production, and marketing-content permissions
  MUST be granted to the named content team members (view/author for writers, review for editors,
  publish for the content lead, admin for the platform owner) via the RBAC screen, recorded in the
  runbook.
- **FR-5.** `WWW_CONTENT_SOURCE=api` MUST be set in `.github/workflows/pages-www.yml` and deployed;
  the first DB-sourced production build MUST be verified against the post-cutover checklist (FR-6)
  before the freeze lifts.
- **FR-6.** A post-cutover verification checklist MUST be executed and recorded: every blog and docs
  URL returns 200 with the expected title/canonical/description; sitemap URL count matches the
  pre-cutover manifest; `llms.txt`/`llms-full.txt` non-empty and complete; feeds validate; a sample
  of 10 pages passes Rich Results; redirects resolve; docs search works; the in-app help widget
  returns DB-backed articles.
- **FR-7.** `marketing_build_provider` MUST be configured (MC.8) and an end-to-end publish MUST be
  performed by a content expert — a real article, published from the workspace, verified live — as
  the acceptance signal.
- **FR-8.** After **two weeks** of stable operation (no rollback, no content incident, Search Console
  showing no coverage or hreflang regressions), decommission MUST proceed:
  - delete `www/src/blog/**` and `www/src/docs/**` (except `_categories.ts` if still referenced —
    otherwise delete it too);
  - remove the `import.meta.glob` code paths and the `files` implementation of the content source;
  - remove `WWW_CONTENT_SOURCE` (or reduce it to a no-op documented as removed);
  - narrow `check-help-freshness.mjs` and `editorial*.mjs` to file-based pages only, or delete them
    where MC.11 fully supersedes them;
  - remove the legacy static mapping in `support_widget_http.go` (MC.13 FR-6 fallback);
  - remove `www/public/docs-*.png` once MC.5 assets serve every reference.
- **FR-9.** Git history MUST be preserved (no rewriting); a `docs/plan/marketing-content/ARCHIVE.md`
  MUST record the final commit SHA of the file-based content for provenance.
- **FR-10.** CI guardrails MUST be added: a check that fails if `www/src/blog/*.md` or
  `www/src/docs/**/*.md` exist; a check that the build's `contentSource` is `api`; the sitemap-count
  guard from MC.12; and the MC.4 golden-corpus jobs.
- **FR-11.** The e2e coverage manifest (`e2e/coverage/completed-feature-manifest.json`) MUST gain
  entries for MC.1–MC.15 with honest coverage classifications and spec references.
- **FR-12.** Documentation MUST be updated in one pass: `AGENTS.md` (marketing site content source),
  `www/docs/*` (site-generation, adding-a-page, contributor-guide, editorial-process,
  writing-help-articles, content-contract), `docs/ARCHITECTURE_CONVENTIONS.md`, and the plan README
  index in `docs/plan/README.md`.
- **FR-13.** Completed plans MUST be moved to `docs/completed/marketing-content/` following the
  repository convention, with the index links updated.
- **FR-14.** A rollback procedure MUST be documented and rehearsed once in staging: set
  `WWW_CONTENT_SOURCE=files`, redeploy, and (if files are already deleted) restore them from the
  archived commit — with the explicit note that content authored in the workspace after cutover is
  **not** in those files.

## 6. Non-Functional Requirements

- **Performance** — No regression: the post-cutover build must stay within the existing CI time and
  perf budgets; Lighthouse thresholds unchanged and passing.
- **Security** — The production GitHub dispatch token is created, scoped and stored during this plan;
  its rotation schedule (annual, or on team change) is recorded. Permission grants are minimal and
  named.
- **Privacy & Compliance** — No new processing. The content team is briefed on screenshot hygiene and
  the "no real learner data" rule.
- **Accessibility** — The post-cutover checklist includes an axe pass over one blog and one docs page
  and the workspace, ensuring the migration did not regress published-page accessibility.
- **Scalability** — Post-cutover, adding an article is a database row; the operational ceiling is the
  build time, tracked as content grows (alert at 15 min build).
- **Reliability** — The rollback path is one workflow variable and a redeploy. The freeze window
  bounds the period in which the two sources could diverge.
- **Observability** — Post-cutover dashboard: publish→live latency, build success rate, fallback
  usage, overdue-review ratio, zero-result help queries. Reviewed weekly for the first month.
- **Maintainability** — After decommission there is exactly one content path; the guardrails in FR-10
  keep it that way.
- **Internationalization** — If MC.14 shipped, the checklist includes hreflang verification;
  otherwise the flag stays off and English-only behaviour is confirmed unchanged.
- **Backward compatibility** — Every public URL is unchanged. That is the single most important
  property of this plan.

## 7. Acceptance Criteria

- **AC-1.** *Given* the freeze window opens, *when* a PR touching `www/src/docs` is opened, *then* it
  is blocked by the branch rule until the freeze lifts.
- **AC-2.** *Given* the production import, *when* it completes, *then* the report shows zero unmapped
  keys, zero unknown authors, and every article present, and the report is committed.
- **AC-3.** *Given* the parity harness against production data, *when* it runs, *then* it exits 0 with
  only allowlisted differences.
- **AC-4.** *Given* the flag and permissions are set, *when* a content expert signs in, *then* they
  see Marketing Content and can open, edit and preview articles.
- **AC-5.** *Given* `WWW_CONTENT_SOURCE=api` is deployed, *when* the production build completes,
  *then* `dist/.seo-manifest.json` reports `contentSource: "api"`, `fallbackUsed: false`, and a URL
  count equal to the pre-cutover manifest.
- **AC-6.** *Given* the verification checklist, *when* executed, *then* every item passes and the
  completed checklist is attached to the release record.
- **AC-7.** *Given* a content expert publishes a real article from the workspace, *when* the build
  completes, *then* the article is live at its URL, in the sitemap, in the feed and in the docs search
  index, with no engineer involvement.
- **AC-8.** *Given* two weeks of stability, *when* decommission runs, *then* the markdown directories
  and the `files` content-source implementation are deleted, and the full test suite plus the
  marketing build pass.
- **AC-9.** *Given* the CI guardrails, *when* someone adds `www/src/blog/new-post.md`, *then* the
  build fails with a message pointing at the workspace.
- **AC-10.** *Given* the rollback rehearsal in staging, *when* `WWW_CONTENT_SOURCE=files` is set and
  redeployed, *then* the site builds from files successfully within one deploy cycle.
- **AC-11.** *Given* the documentation pass, *when* a new engineer reads `AGENTS.md` and
  `www/docs/adding-a-page.md`, *then* nothing instructs them to add a markdown article to `www/src`.
- **AC-12.** *Given* the plan set is complete, *when* the program closes, *then* MC.1–MC.15 are moved
  to `docs/completed/marketing-content/` and `docs/plan/README.md` reflects it.

## 8. Data Model

No schema changes. Two data operations:

1. **Final import** — MC.6 importer against production (idempotent; articles already imported during
   rehearsal are updated in place, and any article edited in the workspace since is **skipped**
   unless `--force`, with the skip listed loudly in the report).
2. **Post-cutover snapshot** — a `pg_dump` of the `marketing` schema taken immediately before and
   after the flip, retained with the release record. This is the real rollback asset once files are
   deleted (FR-14's caveat).

## 9. API Surface

No new endpoints. Configuration changes only:

- Platform settings: `ff_marketing_content = true`, `marketing_build_provider = github`,
  `marketing_build_repo`, `marketing_build_workflow_ref`, `marketing_build_token_encrypted`,
  `marketing_content_lint_enforce = true`.
- Workflow: `WWW_CONTENT_SOURCE: api`, `CONTENT_API_BASE: https://self.lextures.com`,
  `CONTENT_KNOWN_PATHS_TOKEN` secret.
- RBAC grants per named user (recorded in the runbook, not in code).

## 10. UI / UX

No new UI. User-facing communication is the deliverable:

- **Before:** an announcement to the content team with the freeze window, training session date and
  what changes for them.
- **During:** a workspace banner (existing banner system, `internal/repos/banners`) stating the freeze
  and that publishing resumes at a stated time.
- **After:** a short "you now publish from here" note pinned in the workspace, plus the help articles
  from MC.10/MC.11.
- Accessibility: banners use the existing accessible banner component; announcements are text, not
  images.

## 11. AI / ML Considerations

Not AI-touching. One closing note for the record: the program deliberately shipped no AI content
generation, and the governance surfaces (MC.11) are where any future AI-assisted authoring would have
to declare itself.

## 12. Integration Points

- **Repos/files:** `www/src/blog/**`, `www/src/docs/**`, `www/src/utils/{blog,docs}.ts`,
  `www/src/lib/content-source.ts`, `www/scripts/{check-help-freshness,editorial,editorial-core}.mjs`,
  `www/public/docs-*.png`, `.github/workflows/pages-www.yml`, `scripts/check-*.sh` (new guardrail),
  `e2e/coverage/completed-feature-manifest.json`, `docs/plan/README.md`, `AGENTS.md`.
- **Server:** `support_widget_http.go` legacy mapping removal; platform settings configuration.
- **Process:** release management, content team training, Search Console monitoring.

## 13. Dependencies & Sequencing

- Must ship after: MC.1–MC.13 (MC.14 optional — cutover does not require translations).
- Must ship before: nothing; this closes the program.
- Shared infra: production database access for the import; GitHub admin for the token and branch
  rule.

**Cutover sequence (single day)**

```
T-72h  Announce freeze; confirm training complete; verify staging parity green
T-24h  Dry-run import against a production clone; review report
T-0    Freeze starts; final import; parity harness against production data   ── GO/NO-GO
T+1h   Enable flag; grant permissions; verify workspace access
T+2h   Deploy WWW_CONTENT_SOURCE=api; watch build; run verification checklist ── GO/NO-GO
T+3h   Configure build dispatch; content expert publishes a real article end-to-end
T+4h   Freeze lifts; announce
T+2w   Decommission (FR-8) if stable
```

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Divergence between files and DB during the freeze | M | H | Short freeze; branch protection on the two directories; importer skips workspace-edited articles and reports them |
| Cutover build regresses SEO silently | M | **H** | Parity harness gate, manifest URL-count guard, post-cutover checklist, two-week Search Console watch |
| Rollback after decommission is not really possible | M | H | Pre/post `pg_dump` snapshots retained; archived commit SHA recorded; decommission deferred a full two weeks; explicit statement that workspace-authored content is not in the files |
| Content team not ready → nobody publishes → the old way persists | M | H | Training before the freeze; the first real publish is an acceptance criterion (AC-7), not an afterthought |
| Permission grants too broad in the rush | M | M | Named grants recorded in the runbook; review the RBAC report a week after cutover |
| Build dispatch misconfigured in production only | M | M | Self-test dispatch (MC.8 risk table) executed at T+3h before the freeze lifts |
| Guardrails block a legitimate file-based page | L | L | The check targets only `www/src/blog` and `www/src/docs`, not other content modules |

## 15. Rollout Plan

- **Feature flag:** `ff_marketing_content` flips to **true** in production in this plan; it remains
  the master kill switch afterwards.
- **Migration sequencing:** schema (already applied) → final backfill → code already deployed →
  flag flip → build-source flip → dispatch configuration.
- **Dogfood / pilot cohort:** the content team has been on staging since MC.9; production cutover is
  the same people on the same workflow.
- **GA criteria:** AC-5, AC-6 and AC-7 all satisfied, plus zero P1/P2 content incidents in the first
  week.
- **Rollback path:** (a) within minutes — `WWW_CONTENT_SOURCE=files` + redeploy; (b) flag off —
  workspace hidden, APIs 404, site unaffected; (c) post-decommission — restore files from the
  archived SHA **and** re-import any workspace-authored articles from the DB snapshot.

## 16. Test Plan

- **Unit** — CI guardrail script (fails on a planted markdown article, passes otherwise); manifest
  `contentSource` assertion.
- **Integration** — full production-clone rehearsal: import → build → verify; rollback rehearsal in
  staging (AC-10).
- **End-to-end** — the real publish (AC-7) executed by a content expert, plus the existing e2e suite
  green; `make e2e` and the marketing build both pass post-decommission.
- **Security** — verify the dispatch token scope in production; verify permission grants match the
  runbook; confirm no draft content is reachable on the public site (spot-check with a draft article's
  path).
- **Accessibility** — axe over one published blog page, one docs page and the workspace after
  cutover.
- **Performance / load** — production build time recorded before and after; Lighthouse CI green;
  publish→live latency measured on the first three real publishes.
- **Manual exploratory** — the post-cutover checklist itself (FR-6), executed by two people
  independently.

## 17. Documentation & Training

- **Content team training** (before freeze): writing in the workspace, the content contract, review
  flow, publishing and expected latency, media and alt text, what to do when a build fails.
- **Runbooks:** cutover runbook (this plan's sequence), publishing runbook, "site is behind" runbook,
  token rotation, permission grants register.
- **Docs pass:** `AGENTS.md`, `www/docs/{site-generation,adding-a-page,contributor-guide,
  editorial-process,writing-help-articles,content-contract}.md`,
  `docs/ARCHITECTURE_CONVENTIONS.md`, `docs/plan/README.md`, plus moving MC.1–MC.15 to
  `docs/completed/marketing-content/`.
- **Public help articles** (authored in the new system, closing the loop): "Publishing marketing
  content", "Reviewing marketing content", "Adding images to help articles".

## 18. Open Questions

1. Who is the accountable owner of the content system after handover — Docs/Content or Web platform?
   (Must be named before cutover; proposed: Docs/Content owns content and workflow, Web platform owns
   the pipeline.)
2. How long do we retain the pre/post `pg_dump` snapshots? (Proposed: 12 months with the release
   record.)
3. Do we delete `www/src/docs/_categories.ts` or keep it as the seed for category creation?
   (Proposed: delete after confirming no import path depends on it; the DB is the source.)
4. Should the freeze also cover `www/src/lib/authors.ts`? (Yes — it is content data; add it to the
   protected paths.)
5. Do we announce the new publishing capability externally (a blog post about how we publish)?
   (Optional; a nice dogfood artefact if the team wants it.)

## 19. References

- Files this work touches: `.github/workflows/pages-www.yml`, `www/src/blog/**`, `www/src/docs/**`,
  `www/src/utils/{blog,docs}.ts`, `www/src/lib/content-source.ts`, `www/scripts/*`,
  `server/internal/httpserver/support_widget_http.go`, `scripts/check-no-file-articles.sh` (new),
  `e2e/coverage/completed-feature-manifest.json`, `docs/plan/README.md`, `AGENTS.md`.
- Related plans: every plan in this folder; in particular
  [MC.6](MC.6-markdown-to-database-migration.md) (parity harness),
  [MC.7](MC.7-www-build-time-content-integration.md) (`WWW_CONTENT_SOURCE`),
  [MC.8](MC.8-publish-pipeline-and-scheduling.md) (dispatch configuration),
  [MC.12](MC.12-seo-parity-from-database.md) (verification surface).
- Precedent: the SEO.1 cutover pattern (build-time generation adopted with CI assertions rather than
  a flag day).
