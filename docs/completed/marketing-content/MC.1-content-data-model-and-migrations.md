# MC.1 — Content Data Model, Feature Flag & RBAC

> Implementation plan. Source: [docs/plan/marketing-content/README.md](../../plan/marketing-content/README.md) §Architecture.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC.1 |
| **Section** | MC — Marketing Content Platform |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS (marketing surface serves all three) |
| **Status (today)** | COMPLETE — implemented in migrations 477–478 and `internal/repos/marketingcontent` |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Server platform |
| **Depends on** | — |
| **Unblocks** | MC.2, MC.3, MC.4, MC.5, MC.14 |

---

## 1. Problem Statement

Blog posts and help articles exist only as markdown files under `www/src/blog/` and `www/src/docs/`,
loaded at build time by `import.meta.glob` in [`www/src/utils/blog.ts`](../../../www/src/utils/blog.ts)
and [`www/src/utils/docs.ts`](../../../www/src/utils/docs.ts). There is nowhere in the database to
put an article, no identity for an author or a help category, no revision history, and no permission
that means "may edit marketing content". Every subsequent plan in this program — the authoring API,
the public read API, the admin workspace — needs that foundation first. Without it, content velocity
stays gated on engineers, which is the business problem this program exists to remove.

## 2. Goals

- Model blog posts and help articles in one `marketing` schema that captures **every** field the
  current front matter carries, with no lossy round-trip.
- Give every article an append-only revision history so an edit is recoverable and attributable.
- Ship the platform feature flag (`ff_marketing_content`) and the five RBAC permission strings the
  rest of the program gates on, defaulting to off/unheld.
- Make slug/locale/path uniqueness a database invariant, not application etiquette.
- Keep the schema additive: no existing table is altered except `settings.platform_app_settings`
  (one nullable column) and the RBAC seed tables.

## 3. Non-Goals

- No API, no handler, no UI — this plan ships migrations, Go types and repo functions only.
- No data backfill; the importer is [MC.6](MC.6-markdown-to-database-migration.md).
- No media/binary storage design beyond the FK stub; that is [MC.5](../../plan/marketing-content/MC.5-marketing-media-library.md).
- No translation rows; the `translation_group_id` column exists but is unused until
  [MC.14](MC.14-localization-and-translations.md).
- No change to course content, `course.content_pages`, or the Content Tools model.

## 4. Personas & User Stories

- **As a content expert**, I want my draft, my edits and my published version stored durably so that
  a browser crash or a colleague's concurrent edit never loses my work.
- **As an editor**, I want every save attributed to a person and a timestamp so that I can see who
  changed a published help article and revert it.
- **As a platform admin**, I want a single switch that hides the entire feature until we are ready,
  so that an unfinished workspace is never visible to customers.
- **As a security reviewer**, I want content editing to require an explicitly granted permission that
  no default role holds, so that enabling the flag does not widen anyone's access.
- **As a homeschool/K12/HE prospect** (indirect), I want help articles that stay accurate, which
  starts with the platform being able to record when an article was last reviewed and by whom.

## 5. Functional Requirements

- **FR-1.** The system MUST create a `marketing` schema containing `content_articles`,
  `content_revisions`, `content_categories`, `content_authors`, `content_tags`,
  `content_article_tags` and `content_redirects`.
- **FR-2.** `content_articles` MUST carry a `kind` discriminator with values `blog` and `doc`, and
  MUST store all fields currently expressed in front matter: `title`, `description`, `body_md`,
  `author_slug`, `reviewer_slug`, `published_at`, `content_updated_at`, `reviewed_at`,
  `review_due_on`, `primary_question`, `cluster`, `pillar`, `brief_ref`, `verified_against`,
  `keywords[]`, `related_to[]`, `roles[]`, `segments[]`, `citations[]`.
- **FR-3.** `content_articles.status` MUST be one of `draft`, `in_review`, `changes_requested`,
  `scheduled`, `published`, `archived`, and MUST default to `draft`.
- **FR-4.** The system MUST enforce uniqueness of `(kind, locale, slug)` among rows where
  `deleted_at IS NULL`, and MUST enforce uniqueness of the resolved public `path` under the same
  condition (a `doc` path includes its category, a `blog` path does not).
- **FR-5.** Every write to `content_articles` MUST insert a `content_revisions` row containing the
  full body and metadata snapshot, the acting user, and a monotonically increasing `revision_no`
  scoped to the article.
- **FR-6.** `content_articles.revision_no` MUST act as an optimistic-concurrency token: a caller
  supplying a stale value MUST be rejected by the repo layer with a typed conflict error.
- **FR-7.** `content_categories` MUST reproduce the shape of
  [`www/src/docs/_categories.ts`](../../../www/src/docs/_categories.ts) (`slug`, `title`,
  `description`, `sort_order`, `platform_path`) and MUST be referenced by `doc` articles.
- **FR-8.** `content_authors` MUST reproduce the shape of
  [`www/src/lib/authors.ts`](../../../www/src/lib/authors.ts) (`slug`, `name`, `job_title`, `bio`,
  `knows_about[]`, `status ∈ {active, retired}`, `links jsonb`) and MUST be the referential target of
  `author_slug` and `reviewer_slug`.
- **FR-9.** `content_redirects` MUST store `from_path`, `to_path`, `status_code ∈ {301, 302}`,
  `source ∈ {manual, slug_change}` and MUST reject a `from_path` that equals any live article path.
- **FR-10.** A migration MUST add nullable `ff_marketing_content BOOLEAN` to
  `settings.platform_app_settings`, wired through `internal/config`, `platformconfig.Row`,
  `platformconfig.Merge` (default **false**) and the patch allowlist.
- **FR-11.** A migration MUST seed the permissions `global:app:marketing-content:view`,
  `:author`, `:review`, `:publish`, `:admin` with descriptions, and MUST grant all five to the
  Global Admin role only.
- **FR-12.** `content_articles` MUST maintain a generated `search_tsv tsvector` over title,
  description, primary question, keywords and body, with a GIN index.
- **FR-13.** Soft deletion MUST be supported via `deleted_at`; no plan in this program issues a hard
  `DELETE` on an article that has ever been published.
- **FR-14.** Every table MUST carry `created_at`, `updated_at` (trigger-maintained), `created_by` and
  `updated_by` where an acting user exists.
- **FR-15.** The migration MUST be reversible: a `.down.sql` sibling drops the schema, the flag
  column and the seeded permissions.
- **FR-16.** Repo functions MUST live in `server/internal/repos/marketingcontent/` and MUST be the
  only place SQL for these tables is written ([ARCHITECTURE_CONVENTIONS §2](../../ARCHITECTURE_CONVENTIONS.md)).

## 6. Non-Functional Requirements

- **Performance** — Article list by `(kind, status, updated_at DESC)` p95 < 25 ms at 5,000 rows;
  slug lookup p95 < 5 ms; body columns excluded from list queries (no `SELECT *`).
- **Security** — No PII beyond the author's display identity. Permission strings follow the existing
  four-segment `scope:area:function:action` grammar validated by
  [`internal/authz`](../../../server/internal/authz/authz.go). No table is org-scoped: marketing
  content is platform-global and readable only by holders of the new permissions until published.
- **Privacy & Compliance** — Author bios are public-facing profile data; the `content_authors` row is
  the consent surface, and retiring an author MUST NOT delete attribution history (SEO.3 byline
  policy). No learner data touches this schema, so FERPA/COPPA obligations are not engaged.
- **Accessibility** — N/A (no UI), but `content_media.alt_text` (MC.5) is declared NOT NULL here in
  spirit: the schema must never make an inaccessible image representable as "valid".
- **Scalability** — Designed for 10⁴ articles and 10⁶ revisions. Revisions are the growth table;
  it is partition-ready by `created_at` but not partitioned at this size.
- **Reliability** — All writes are single-statement or wrapped in a transaction with the revision
  insert; there is no state in which an article body advances without a revision row.
- **Observability** — Row counts by `(kind, status)` exported as a gauge
  (`marketing_content_articles`) via the existing `expvar`/Prometheus bridge in
  [`internal/telemetry`](../../../server/internal/telemetry).
- **Maintainability** — One repo package, files under the 400-line budget enforced by
  `make lint-structure`; no SQL outside `internal/repos/marketingcontent`.
- **Internationalization** — `locale` (BCP-47, default `en`) and `translation_group_id` exist from
  day one so MC.14 is additive, not a rewrite.
- **Backward compatibility** — Purely additive. With `ff_marketing_content` off, the platform behaves
  exactly as today; the tables are simply empty.

## 7. Acceptance Criteria

- **AC-1.** *Given* a clean database, *when* migrations run, *then* the `marketing` schema exists
  with all seven tables and `RUN_MIGRATIONS=true` completes without error, and `make migrate-lint`
  passes.
- **AC-2.** *Given* an article with `kind='doc'`, `locale='en'`, `slug='finding-your-course'`,
  *when* a second row with the same triple is inserted, *then* the unique index rejects it.
- **AC-3.** *Given* a published article whose slug changes, *when* the repo update runs, *then* a
  `content_redirects` row (`source='slug_change'`, `status_code=301`) is created in the same
  transaction.
- **AC-4.** *Given* an update carrying `expectedRevisionNo=3` while the stored value is `4`, *when*
  the repo update runs, *then* it returns `ErrRevisionConflict` and no row changes.
- **AC-5.** *Given* any successful article write, *when* the transaction commits, *then*
  `content_revisions` contains a new row whose `body_md` equals the persisted body and whose
  `revision_no` equals the article's new `revision_no`.
- **AC-6.** *Given* the platform settings row is unset, *when* `platformconfig.Merge` runs, *then*
  `FFMarketingContent` is `false` (unit test mirrors `merge_test.go`).
- **AC-7.** *Given* a fresh install, *when* the permission seed runs, *then* exactly the Global Admin
  role holds the five `global:app:marketing-content:*` grants, and re-running the migration is a
  no-op (`ON CONFLICT DO NOTHING`).
- **AC-8.** *Given* an article referencing `author_slug='ghost'`, *when* insert runs, *then* the FK
  to `content_authors` rejects it.
- **AC-9.** *Given* 5,000 seeded articles, *when* the list query runs with the `(kind, status,
  updated_at DESC)` index, *then* `EXPLAIN` shows an index scan and p95 < 25 ms.
- **AC-10.** *Given* the down migration, *when* it runs, *then* the schema, flag column and seeded
  permissions are removed and the rest of the database is unchanged.

## 8. Data Model

New schema `marketing`. Migration files (indicative numbers — take the next free number at merge):
`server/migrations/475_marketing_content_core.sql` (+ `.down.sql`) and
`server/migrations/476_marketing_content_rbac_and_flag.sql` (+ `.down.sql`).

```sql
CREATE SCHEMA IF NOT EXISTS marketing;

CREATE TABLE marketing.content_authors (
    slug          TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    job_title     TEXT NOT NULL DEFAULT '',
    bio           TEXT NOT NULL DEFAULT '',
    knows_about   TEXT[] NOT NULL DEFAULT '{}',
    image_media_id UUID,                         -- FK added in MC.5
    links         JSONB NOT NULL DEFAULT '{}'::jsonb,
    user_id       UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    status        TEXT NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active', 'retired')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE marketing.content_categories (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug          TEXT NOT NULL,
    locale        TEXT NOT NULL DEFAULT 'en',
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    sort_order    INTEGER NOT NULL DEFAULT 100,
    platform_path TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (locale, slug)
);

CREATE TABLE marketing.content_articles (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind                 TEXT NOT NULL CHECK (kind IN ('blog', 'doc')),
    slug                 TEXT NOT NULL,
    locale               TEXT NOT NULL DEFAULT 'en',
    translation_group_id UUID NOT NULL DEFAULT gen_random_uuid(),   -- MC.14
    category_id          UUID REFERENCES marketing.content_categories (id),
    path                 TEXT NOT NULL,                             -- '/blog/x' | '/docs/cat/x'
    title                TEXT NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    body_md              TEXT NOT NULL DEFAULT '',
    status               TEXT NOT NULL DEFAULT 'draft'
                         CHECK (status IN ('draft','in_review','changes_requested',
                                           'scheduled','published','archived')),
    author_slug          TEXT NOT NULL REFERENCES marketing.content_authors (slug),
    reviewer_slug        TEXT REFERENCES marketing.content_authors (slug),
    published_at         TIMESTAMPTZ,
    first_published_at   TIMESTAMPTZ,
    scheduled_for        TIMESTAMPTZ,
    content_updated_at   TIMESTAMPTZ,          -- front-matter `updated`, drives sitemap lastmod
    reviewed_at          DATE,
    review_due_on        DATE,
    primary_question     TEXT NOT NULL DEFAULT '',
    cluster              TEXT NOT NULL DEFAULT '',
    pillar               TEXT NOT NULL DEFAULT '',
    brief_ref            TEXT NOT NULL DEFAULT '',
    verified_against     TEXT NOT NULL DEFAULT '',
    keywords             TEXT[] NOT NULL DEFAULT '{}',
    related_to           TEXT[] NOT NULL DEFAULT '{}',
    roles                TEXT[] NOT NULL DEFAULT '{}',
    segments             TEXT[] NOT NULL DEFAULT '{}',
    citations            TEXT[] NOT NULL DEFAULT '{}',
    hero_media_id        UUID,                  -- FK added in MC.5
    quality_score        NUMERIC(3,1),          -- MC.4
    quality_report       JSONB,                 -- MC.4 findings
    noindex              BOOLEAN NOT NULL DEFAULT FALSE,
    canonical_override   TEXT,
    extra                JSONB NOT NULL DEFAULT '{}'::jsonb,
    revision_no          INTEGER NOT NULL DEFAULT 1,
    created_by           UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    updated_by           UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at           TIMESTAMPTZ,
    search_tsv           TSVECTOR GENERATED ALWAYS AS (
                            setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
                            setweight(to_tsvector('english', coalesce(description, '')), 'B') ||
                            setweight(to_tsvector('english', coalesce(primary_question, '')), 'B') ||
                            setweight(to_tsvector('english', array_to_string(keywords, ' ')), 'C') ||
                            setweight(to_tsvector('english', coalesce(body_md, '')), 'D')
                         ) STORED,
    CONSTRAINT doc_requires_category CHECK (kind <> 'doc' OR category_id IS NOT NULL),
    CONSTRAINT published_has_timestamp CHECK (status <> 'published' OR published_at IS NOT NULL),
    CONSTRAINT scheduled_has_timestamp CHECK (status <> 'scheduled' OR scheduled_for IS NOT NULL)
);

CREATE UNIQUE INDEX idx_mc_articles_slug_live
    ON marketing.content_articles (kind, locale, slug) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_mc_articles_path_live
    ON marketing.content_articles (path) WHERE deleted_at IS NULL;
CREATE INDEX idx_mc_articles_admin_list
    ON marketing.content_articles (kind, status, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_mc_articles_published
    ON marketing.content_articles (kind, locale, published_at DESC)
    WHERE status = 'published' AND deleted_at IS NULL;
CREATE INDEX idx_mc_articles_due_publish
    ON marketing.content_articles (scheduled_for) WHERE status = 'scheduled';
CREATE INDEX idx_mc_articles_search ON marketing.content_articles USING GIN (search_tsv);
CREATE INDEX idx_mc_articles_group ON marketing.content_articles (translation_group_id);

CREATE TABLE marketing.content_revisions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id   UUID NOT NULL REFERENCES marketing.content_articles (id) ON DELETE CASCADE,
    revision_no  INTEGER NOT NULL,
    body_md      TEXT NOT NULL,
    metadata     JSONB NOT NULL,        -- full column snapshot minus body
    change_note  TEXT NOT NULL DEFAULT '',
    status_after TEXT NOT NULL,
    actor_id     UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (article_id, revision_no)
);
CREATE INDEX idx_mc_revisions_article ON marketing.content_revisions (article_id, revision_no DESC);

CREATE TABLE marketing.content_tags (
    id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug   TEXT NOT NULL UNIQUE,
    label  TEXT NOT NULL
);
CREATE TABLE marketing.content_article_tags (
    article_id UUID NOT NULL REFERENCES marketing.content_articles (id) ON DELETE CASCADE,
    tag_id     UUID NOT NULL REFERENCES marketing.content_tags (id) ON DELETE CASCADE,
    PRIMARY KEY (article_id, tag_id)
);

CREATE TABLE marketing.content_redirects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_path   TEXT NOT NULL UNIQUE,
    to_path     TEXT NOT NULL,
    status_code INTEGER NOT NULL DEFAULT 301 CHECK (status_code IN (301, 302)),
    source      TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'slug_change')),
    article_id  UUID REFERENCES marketing.content_articles (id) ON DELETE SET NULL,
    created_by  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Flag + RBAC migration:

```sql
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_marketing_content BOOLEAN;
COMMENT ON COLUMN settings.platform_app_settings.ff_marketing_content IS
    'Enables the database-backed Marketing Content workspace and APIs (plan MC). Default OFF.';

INSERT INTO "user".permissions (permission_string, description) VALUES
  ('global:app:marketing-content:view',    'View the Marketing Content workspace and article list.'),
  ('global:app:marketing-content:author',  'Create and edit marketing content drafts.'),
  ('global:app:marketing-content:review',  'Approve or request changes on marketing content.'),
  ('global:app:marketing-content:publish', 'Publish, schedule, unpublish and archive marketing content.'),
  ('global:app:marketing-content:admin',   'Manage marketing content categories, authors, redirects and media.')
ON CONFLICT (permission_string) DO NOTHING;
-- Grant all five to Global Admin only (pattern: migrations/431_parent_link_assign.sql).
```

**Backfill strategy.** None here. The tables ship empty; [MC.6](MC.6-markdown-to-database-migration.md)
imports the 5 blog posts, 70 help articles, 16 categories and the author registry. `content_authors`
is seeded with `chase-willden` by MC.6, not by this migration, so the schema stays data-free.

## 9. API Surface

No HTTP surface in this plan. The Go surface it establishes:

```go
// server/internal/repos/marketingcontent/
type Article struct { /* one field per column above */ }
type ArticleFilter struct { Kind, Status, Locale, CategorySlug, Q string; Cursor string; Limit int }

func ListArticles(ctx, db, ArticleFilter) ([]ArticleSummary, string, error)  // no body_md
func GetArticleByID(ctx, db, uuid.UUID) (*Article, error)
func GetArticleByPath(ctx, db, path string) (*Article, error)
func InsertArticle(ctx, tx, NewArticle) (*Article, error)      // writes revision 1
func UpdateArticle(ctx, tx, ArticleUpdate) (*Article, error)   // ErrRevisionConflict on stale token
func SoftDeleteArticle(ctx, tx, uuid.UUID, actor uuid.UUID) error
func ListRevisions(ctx, db, articleID uuid.UUID, limit int) ([]Revision, error)
func GetRevision(ctx, db, articleID uuid.UUID, no int) (*Revision, error)
func UpsertCategory / ListCategories / UpsertAuthor / ListAuthors / ...
func InsertRedirect / ListRedirects / DeleteRedirect
var ErrRevisionConflict = errors.New("marketingcontent: revision conflict")
var ErrDuplicateSlug    = errors.New("marketingcontent: duplicate slug")
```

Config surface: `config.Config.FFMarketingContent bool`; `platformconfig.Row.FFMarketingContent
*bool`; `Merge` default `false`; patch key `ff_marketing_content`; exposed later as
`ffMarketingContent` by `GET /api/v1/platform/features` (wired in MC.9).

Rate limits, OpenAPI: N/A here; introduced with the routes in MC.2/MC.3.

## 10. UI / UX

No UI. One indirect surface: **Settings → Global platform** gains a "Marketing Content" toggle row,
which is generated from the platform settings schema (`clients/web/src/components/settings/platform-settings-panel.tsx`)
and therefore needs only the flag key, label and help text:

- Label: "Marketing content workspace"
- Help: "Lets permitted users write and publish the public blog and help center from inside
  Lextures. Requires a marketing-content permission to see."
- i18n keys: `settings.platform.ffMarketingContent.label`, `.help`.

Empty/loading/error states, responsive behaviour and focus order are inherited from the existing
settings panel; no new interaction patterns are introduced.

## 11. AI / ML Considerations

Not AI-touching. The schema deliberately reserves `extra JSONB` rather than modelling AI provenance
now; if AI-assisted drafting is added later it will record disclosure through the existing
`internal/aidisclosure` package rather than a bespoke column.

## 12. Integration Points

- **Internal modules touched:** `server/migrations/`, new `server/internal/repos/marketingcontent/`,
  `server/internal/config/config.go`, `server/internal/repos/platformconfig/{platformconfig,features,patch}.go`,
  `server/internal/telemetry` (gauge registration).
- **External services:** none.
- **Events:** none emitted in this plan; MC.8 adds publish events.
- **RBAC:** `"user".permissions` / `"user".rbac_role_permissions` seed rows, matching the pattern in
  `server/migrations/431_parent_link_assign.sql`.

## 13. Dependencies & Sequencing

- Must ship after: nothing.
- Must ship before: MC.2 (authoring API), MC.3 (public API), MC.4 (validation), MC.5 (media),
  MC.14 (translations).
- Shared infra needed: PostgreSQL 16 with `pgcrypto`/`gen_random_uuid()` (already in use);
  no queue, storage or mail dependency.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Front-matter field missed → lossy import | M | H | MC.6 importer asserts round-trip equality of every parsed key; unknown keys land in `extra` and fail the importer loudly rather than being dropped |
| Generated `search_tsv` over `body_md` bloats rows / slows writes | M | M | `body_md` is weight D and articles are small (<40 KB); measure write latency in AC-9 load seed; fall back to a trigger-maintained column if p95 write > 50 ms |
| Path uniqueness fights category moves | M | M | Path is derived and rewritten in the same transaction as the category change, with an automatic `slug_change` redirect (AC-3) |
| Permission strings drift from the four-segment grammar | L | M | Unit test asserts each seeded string passes `authz.PermissionMatches(s, s)` |
| Flag column added to an already-wide settings table | L | L | Column is nullable and merged with an explicit default; `platformconfig` tests cover unset/true/false |

## 15. Rollout Plan

- **Feature flag:** `ff_marketing_content`, default **OFF** (`mergeBool(db.FFMarketingContent, false)`).
- **Sequencing:** schema migration → config/plumbing → repo package → (no backfill) → flag stays off.
- **Dogfood:** none at this stage; the schema is invisible.
- **GA criteria:** migrations green on staging and production, `make migrate-lint` and
  `go test ./internal/repos/marketingcontent/...` passing, gauge visible in Prometheus.
- **Rollback:** run the `.down.sql` pair. Because nothing writes to these tables yet, rollback is
  data-lossless in this plan (it stops being so after MC.6 — noted in MC.15).

## 16. Test Plan

- **Unit** — column/enum round-trip; `ErrRevisionConflict`; slug-change redirect creation; path
  derivation for `blog` vs `doc`; permission-string grammar; `platformconfig.Merge` default off.
- **Integration (DB)** — migration up/down/up idempotency; unique index behaviour under concurrent
  insert of the same slug; revision insert atomicity under a forced transaction abort; FK behaviour
  when an author is retired (attribution preserved).
- **End-to-end** — none (no UI). The e2e coverage manifest entry for MC.1 is `coverage: "unit"` with
  a rationale pointing at the DB tests.
- **Security** — assert no default role other than Global Admin holds any
  `global:app:marketing-content:*` grant after migration (query-based test).
- **Accessibility** — N/A.
- **Performance / load** — seed 5,000 articles + 50,000 revisions; assert list p95 < 25 ms and
  `EXPLAIN` index usage (AC-9).
- **Manual exploratory** — verify the settings toggle appears and persists; verify the flag defaults
  off on a database with no `platform_app_settings` row.

## 17. Documentation & Training

- `server/migrations/README.md` — note the new schema and its ownership.
- `docs/ARCHITECTURE_CONVENTIONS.md` — add `internal/repos/marketingcontent` to the repo table.
- Internal runbook: "Enabling the Marketing Content workspace" stub (completed in MC.15).
- No end-user docs yet — the feature is invisible until MC.9.

## 18. Open Questions

1. Should `content_authors.user_id` be required for authors who log in, so the byline and the acting
   account cannot diverge? (Proposed: optional now, enforced for new authors in MC.11.)
2. Do we want per-org marketing content eventually (white-label help centers), and if so should
   `org_id` be nullable-from-day-one rather than added later? (Proposed: out of scope; the marketing
   site is single-tenant, and adding a nullable column later is cheap.)
3. Is `english` the right FTS configuration given MC.14 adds locales? (Proposed: store a
   `regconfig`-per-locale mapping in MC.14 and rebuild the generated column then.)
4. Should `body_md` have a hard size cap (e.g. 512 KB) enforced by CHECK rather than by the API?

## 19. References

- Files this work touches: `server/migrations/475_*`, `476_*`,
  `server/internal/repos/marketingcontent/*`, `server/internal/config/config.go`,
  `server/internal/repos/platformconfig/{platformconfig,features,patch}.go`.
- Shape sources: `www/src/utils/blog.ts`, `www/src/utils/docs.ts`, `www/src/docs/_categories.ts`,
  `www/src/lib/authors.ts`.
- Precedents: `server/migrations/472_course_coupons.sql` (feature table + flag),
  `server/migrations/431_parent_link_assign.sql` (permission seed),
  `server/internal/repos/platformconfig/merge_test.go` (flag default tests).
- Related plans: [MC.2](../../plan/marketing-content/MC.2-authoring-api-and-revisions.md), [MC.3](../../plan/marketing-content/MC.3-public-content-read-api.md),
  [MC.6](MC.6-markdown-to-database-migration.md), [MC.14](MC.14-localization-and-translations.md).
