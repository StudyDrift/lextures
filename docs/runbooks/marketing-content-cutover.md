# Marketing content cutover and rollback

Docs/Content owns article accuracy and workflow. Web Platform owns the public API, build pipeline,
and incident response. The release manager owns each cutover record. Never put names, tokens, or
database dumps in git; attach them to the restricted release record.

## Before the freeze

1. Announce the four-hour freeze at least 72 hours ahead. Temporarily require Content/Web Platform
   CODEOWNER approval for the archived `www/src/blog`, `www/src/docs`, and `www/src/lib/authors.ts`
   paths. Record the branch-rule URL and the start/end times.
2. Train writers and reviewers in the Marketing Content workspace. Reconfirm the no-real-learner-
   data screenshot rule.
3. Take a `pg_dump --schema=marketing` snapshot and retain it with the release for 12 months.
4. On a production clone, run the importer with validation errors reported, review every skip, then
   run `npm run content:parity -- --api-base <clone-origin>`. Any non-allowlisted difference is no-go.

## Cut over

1. Start the freeze and deploy the server with migration `485_marketing_content_seed.sql`. Normal
   startup migrations seed 5 blog and 65 help articles without direct production database access.
   Confirm those counts, 16 categories, one author, 70 initial revisions, and zero migration errors
   in the release record. The migration uses insert-only conflict handling and never overwrites an
   existing or human-edited article.
2. Run parity against production. Record the command, commit, API origin, result, and approver.
3. Enable `ff_marketing_content`. In RBAC grant named writers view/author, editors review, the content
   lead publish, and the platform owner admin. Export the grant report to the restricted release
   record and review it again after one week.
4. Configure `marketing_build_provider=github`, repository, workflow ref, and encrypted token. The
   fine-grained token is repository-scoped; rotate annually and whenever the owning team changes.
5. Dispatch the API-backed Pages build. It must report `contentSource: "api"`,
   `fallbackUsed: false`, and the approved pre-cutover sitemap count in `dist/.seo-manifest.json`.
6. Have a content expert publish one real approved article. Confirm it live in its canonical URL,
   sitemap, feed (for blog), docs search (for help), and contextual help where applicable.

## Verification record

Two reviewers independently record pass/fail and evidence for: every migrated URL returning 200
with expected title, canonical, and description; sitemap count; complete non-empty `llms.txt` and
`llms-full.txt`; RSS and JSON Feed validation; ten Rich Results samples; redirects; docs search;
DB-backed contextual help; axe on one blog page, one docs page, and the workspace; Lighthouse/build
budgets; and hreflang when translations are enabled. The freeze lifts only when all required items
pass.

For two weeks, monitor Search Console coverage/hreflang, publish-to-live latency, build success,
cache fallback use, overdue-review ratio, and zero-result help queries. A content incident, rollback,
or unexplained coverage regression resets the stability window.

## Rollback rehearsal and recovery

The pre-decommission staging rehearsal used `WWW_CONTENT_SOURCE=files` and one redeploy. That switch
is intentionally gone after MC.15: production now has exactly one code path. Post-decommission,
recover from the archived commit documented in
[`ARCHIVE.md`](../completed/marketing-content/ARCHIVE.md), deploy that recovery branch, then reconcile
all workspace-authored revisions from the retained database snapshot. The archived files alone are
not a current backup. Disable `ff_marketing_content` only to hide authoring during an incident; it
does not replace published static pages.
