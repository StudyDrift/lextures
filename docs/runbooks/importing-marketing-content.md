# Importing marketing content

Migration `485_marketing_content_seed.sql` now installs the initial 5 published blog posts, 65
categorized help articles, 16 help categories, author record, initial revisions, and contextual-help
hints during normal server startup. No production database shell or separate importer command is
required. MC.15 removed the source trees; they remain available only from the commit in
[`ARCHIVE.md`](../completed/marketing-content/ARCHIVE.md). These commands now apply only to a
recovery checkout or audit rehearsal, not normal publishing.

From `server/`, rehearse first:

```bash
go run ./cmd/marketing-content-import --dry-run
```

Then import into a local or staging database:

```bash
DATABASE_URL='postgres://…' go run ./cmd/marketing-content-import
```

Useful controls are `--only=blog|docs|media|taxonomy`, `--slug='glob'`,
`--fail-on-validation-error`, and `--force`. A non-local/non-staging database is rejected unless
`--confirm-production` is explicit. The default audit report is
`docs/plan/marketing-content/import-report.json`; choose a release-specific path with `--report`.

An unchanged rerun reports every article as `unchanged` and creates no revision. The importer skips
an article with revisions after its import revision unless `--force` is supplied. It resolves
`content_updated_at` from front matter, then git, then the publication date. Outside a git checkout,
the fallback requires `--allow-missing-git`.

After a staging import, run the parity harness from `www/`:

```bash
npm run content:parity -- --api-base https://staging.self.lextures.com
```

The original cutover could not proceed while the report had unexpected differences. Before human
editing, rollback was to delete rows whose `extra` contains `import`. After MC.15, restore a recovery
branch from the archived commit and reconcile workspace-authored revisions from the retained
database snapshot; the current build has no file-source switch.
