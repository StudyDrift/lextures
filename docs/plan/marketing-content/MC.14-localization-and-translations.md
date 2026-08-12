# MC.14 — Localization & Translated Content

> Implementation plan. Source: [docs/plan/marketing-content/README.md](README.md) §Plans; extends the
> `locale` / `translation_group_id` columns reserved in [MC.1](MC.1-content-data-model-and-migrations.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC.14 |
| **Section** | MC — Marketing Content Platform |
| **Severity** | MINOR (MAJOR once a non-English market is committed) |
| **Markets** | HE / K12 (international), HS |
| **Status (today)** | MISSING — every marketing page is English-only; the app ships localized UI and an RTL locale, but the public site does not |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform + Docs/Content |
| **Depends on** | MC.1, MC.3, MC.10 |
| **Unblocks** | SEO.17 (international SEO & hreflang) for content pages |

---

## 1. Problem Statement

The product is localized — the app ships translated UI and at least one RTL locale — but every blog
post and help article is English. Today that gap is structural: translating a markdown file means
inventing a directory convention, duplicating front matter and teaching the build about it, so it has
never been attempted. Moving content into the database makes translation a data problem instead of a
build problem, and this is the moment to model it correctly: the same article in several languages,
linked, individually reviewable, individually publishable, and correctly declared to search engines.
Doing it later means a migration of live content instead of a column that is already there.

## 2. Goals

- Let a content expert create a translation of an article, see the source alongside it, and publish
  it independently.
- Serve locale-scoped content through the same public API and build, at URLs that follow one explicit
  policy.
- Emit correct `hreflang` / `x-default` annotations and locale-correct metadata so translations help
  rather than compete.
- Keep the English site byte-identical when only one locale exists — enabling this plan must cost
  nothing until a translation is actually published.
- Make translation status visible: what is translated, what is stale relative to its source, what is
  missing.

## 3. Non-Goals

- No machine translation, and no AI translation workflow (a human owns every published translation).
- No localized *product* strings work — that is the existing i18n system in `clients/web` and
  `internal/l10n`.
- No locale-specific pricing, legal or compliance content (those pages are file-based and have their
  own owners).
- No regional domain strategy (ccTLDs) or geo-routing.
- No translation-vendor integration (TMS/XLIFF export) in v1 — noted as a follow-on.

## 4. Personas & User Stories

- **As a content expert**, I want to create the Spanish version of a help article and see the English
  source next to it while I write.
- **As a content lead**, I want to know which translations are behind their English source, so I can
  prioritise updates.
- **As an international visitor**, I want to land on the page in my language and be able to switch.
- **As an SEO owner**, I want reciprocal `hreflang` on every translated page so Google serves the
  right one and does not treat them as duplicates.
- **As an engineer**, I want the single-locale case to remain exactly as it is today.

## 5. Functional Requirements

- **FR-1.** An article's `locale` MUST be immutable after creation; translations are separate rows
  sharing a `translation_group_id`.
- **FR-2.** The API MUST support creating a translation from a source article
  (`POST /articles/{id}/translations` with `{locale}`), copying structure and metadata, leaving body
  and title to be written, and linking the group.
- **FR-3.** Each translation MUST have its own status, reviewer, review due date, quality report and
  publish lifecycle — publishing English MUST NOT publish anything else.
- **FR-4.** The URL policy MUST be path-prefixed for non-default locales: `/{locale}/blog/{slug}` and
  `/{locale}/docs/{category}/{slug}`; the default locale (`en`) keeps today's unprefixed URLs.
  Slugs MAY be localized (`/es/docs/cursos/encontrar-tu-curso`) and MUST be unique per
  `(kind, locale)`.
- **FR-5.** `content_categories` MUST support per-locale titles/descriptions and localized slugs,
  keyed by `(locale, slug)` (already the MC.1 unique key), with a `category_group_id` linking
  translations of the same category.
- **FR-6.** The public API MUST accept `locale` on every list/detail route, MUST default to `en`, and
  MUST return `availableLocales[]` (path + locale) for every article so the build can emit `hreflang`
  and a language switcher.
- **FR-7.** `/index` MUST group by `translationGroupId` and expose each group's members.
- **FR-8.** The build MUST generate pages for every published translation, add
  `<link rel="alternate" hreflang="…">` for every group member plus `x-default` pointing at the
  default locale, set `<html lang>` and `Content-Language`, and include translations in sitemaps with
  `xhtml:link` alternates.
- **FR-9.** A language switcher MUST appear on translated pages, listing only locales where the
  article is published, and MUST be keyboard accessible and announced correctly.
- **FR-10.** A translation MUST be marked **stale** when its source article's `content_updated_at` is
  newer than the translation's `source_synced_at`; stale translations MUST be visible in the
  workspace and in MC.11's health view.
- **FR-11.** The editor MUST offer a side-by-side source view (read-only English pane) when editing a
  translation, and MUST show the source's revision the translation was last synced to.
- **FR-12.** Content validation (MC.4) MUST use locale-aware text metrics: word/character counts must
  not systematically fail CJK or agglutinative languages; per-locale thresholds MUST be configurable,
  defaulting to the English values for Latin-script locales.
- **FR-13.** Full-text search MUST use a per-locale Postgres text-search configuration where one
  exists (`spanish`, `french`, `german`, …), falling back to `simple`, and the generated `search_tsv`
  MUST be rebuilt accordingly.
- **FR-14.** The docs search index and in-app help (MC.13) MUST be locale-scoped: a viewer in `es`
  gets Spanish articles, falling back to English when a translation does not exist, with the fallback
  labelled.
- **FR-15.** With only `en` content present, every generated artefact MUST be byte-identical to the
  pre-MC.14 build (no empty `hreflang` tags, no `/en/` duplicates, no switcher).
- **FR-16.** Redirects MUST be locale-aware: changing a translated slug creates a redirect under that
  locale's prefix only.

## 6. Non-Functional Requirements

- **Performance** — Locale filtering is indexed (`(locale, kind, published_at)` from MC.1); no
  additional per-request cost. Build time grows linearly with published translations.
- **Security** — No new surface; locale is a validated enum (BCP-47 subset from an allowlist), never
  interpolated into paths without validation (path traversal guard on `/{locale}/…`).
- **Privacy & Compliance** — None specific. Note: translated help content that describes
  jurisdiction-specific behaviour must not imply legal advice; MC.11's review checklist covers it.
- **Accessibility** — `<html lang>` must be correct per page (a wrong `lang` breaks screen-reader
  pronunciation); RTL locales must render with `dir="rtl"` and mirrored layout on both the site and
  the editor preview; the language switcher must expose language names in their own language
  (`Español`, not "Spanish") with `lang` attributes on each option.
- **Scalability** — Article count multiplies by locale count; the index endpoint and sitemaps must
  shard correctly at 500 × N. Sitemap alternates increase payload — measured and budgeted.
- **Reliability** — A missing translation is never an error; the site falls back to the default
  locale.
- **Observability** — `marketing_content_articles{locale}` gauge; `…_translations_stale` gauge;
  per-locale page counts in the build summary.
- **Maintainability** — Locale is a first-class parameter throughout the source interface (MC.7 FR-1
  already requires it), not a special case bolted on.
- **Internationalization** — This plan *is* the i18n plan for marketing content; it must align with
  SEO.17's hreflang guidance and with `internal/l10n`'s locale list so the app and site agree on
  locale codes.
- **Backward compatibility** — FR-15 is the compatibility contract: single-locale output is
  unchanged, proven by the MC.6 parity harness after this plan ships.

## 7. Acceptance Criteria

- **AC-1.** *Given* only English content, *when* the site is built, *then* output is byte-identical to
  the pre-MC.14 build (no `hreflang`, no `/en/` paths, no switcher).
- **AC-2.** *Given* a published English article, *when* a translator creates an `es` translation,
  *then* a new row shares the `translation_group_id`, starts as `draft`, and inherits structural
  metadata.
- **AC-3.** *Given* a published `es` translation, *when* the site is built, *then*
  `/es/docs/{cat}/{slug}` exists with `<html lang="es">`, reciprocal `hreflang` on both pages, and an
  `x-default` pointing at the English URL.
- **AC-4.** *Given* both pages, *when* sitemaps are generated, *then* each entry carries
  `xhtml:link rel="alternate"` for every locale in the group.
- **AC-5.** *Given* an English article edited after its translation was synced, *when* the workspace
  loads, *then* the translation shows **Stale** and appears in the health view.
- **AC-6.** *Given* the editor on a translation, *when* it opens, *then* the English source is shown
  read-only alongside, with the synced revision number displayed.
- **AC-7.** *Given* an `ar` (RTL) translation, *when* rendered, *then* the page uses `dir="rtl"`,
  layout mirrors correctly, and the editor preview matches.
- **AC-8.** *Given* a viewer whose app locale is `es`, *when* the help widget opens, *then* Spanish
  articles are returned where they exist and English ones are returned labelled as fallback.
- **AC-9.** *Given* a Spanish article, *when* it is searched with a Spanish stem
  (e.g. "curso"/"cursos"), *then* FTS matches using the `spanish` configuration.
- **AC-10.** *Given* a CJK article of typical length, *when* validated, *then* passage-length rules do
  not fail it purely on word count (locale-aware metrics).
- **AC-11.** *Given* a translated slug change, *when* saved, *then* a redirect is created under that
  locale prefix only and the English URL is untouched.
- **AC-12.** *Given* an unsupported locale code in a URL, *when* requested/built, *then* it is
  rejected (not generated) and no path-traversal is possible.

## 8. Data Model

Migration `484_marketing_content_i18n.sql` (indicative number):

```sql
ALTER TABLE marketing.content_articles
    ADD COLUMN IF NOT EXISTS source_article_id UUID REFERENCES marketing.content_articles (id),
    ADD COLUMN IF NOT EXISTS source_synced_revision INTEGER,
    ADD COLUMN IF NOT EXISTS source_synced_at TIMESTAMPTZ;

ALTER TABLE marketing.content_categories
    ADD COLUMN IF NOT EXISTS category_group_id UUID NOT NULL DEFAULT gen_random_uuid();

CREATE TABLE marketing.content_locales (
    code        TEXT PRIMARY KEY,              -- BCP-47, e.g. 'en', 'es', 'ar'
    label       TEXT NOT NULL,                 -- endonym, e.g. 'Español'
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    rtl         BOOLEAN NOT NULL DEFAULT FALSE,
    ts_config   TEXT NOT NULL DEFAULT 'simple',-- Postgres regconfig name
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order  INTEGER NOT NULL DEFAULT 100
);
CREATE UNIQUE INDEX idx_mc_locales_one_default ON marketing.content_locales ((is_default)) WHERE is_default;

INSERT INTO marketing.content_locales (code, label, is_default, ts_config)
VALUES ('en', 'English', TRUE, 'english') ON CONFLICT DO NOTHING;
```

`search_tsv` becomes locale-aware: replace the generated column with a trigger-maintained column that
looks up `content_locales.ts_config` for the row's locale (generated columns cannot reference other
tables). The migration MUST rebuild the index and MUST be tested for behaviour parity on English rows.

**Backfill:** all existing rows are `en` with `ts_config='english'` — identical to today's behaviour.

## 9. API Surface

| Verb | Path | Auth | Notes |
|---|---|---|---|
| POST | `/api/v1/admin/marketing/articles/{id}/translations` | `…:author` | `{locale, slug?}` → new draft |
| GET | `/api/v1/admin/marketing/articles/{id}/translations` | `…:view` | group members + staleness |
| POST | `/api/v1/admin/marketing/articles/{id}/mark-synced` | `…:author` | records source revision |
| GET/POST | `/api/v1/admin/marketing/locales` · PATCH `/{code}` | `…:view` / `…:admin` | enable/disable, order |
| GET | `/api/v1/public/content/*?locale=` | anonymous | all read routes gain `locale` |

```ts
type TranslationLink = {
  locale: string; path: string; status: Status; stale: boolean
  sourceSyncedRevision: number | null; publishedAt: string | null
}
// PublicArticle gains:
//   availableLocales: Array<{ locale: string; path: string }>
//   isFallback?: boolean          // set by help/search when falling back to default locale
```

## 10. UI / UX

**Workspace**

- A **Locale** column and filter in the content list; a group indicator ("EN · ES · stale").
- Article header shows a locale chip and a **Translations** menu: existing translations with status,
  plus "Add translation ▾".

**Editor**

- Editing a translation shows a read-only source pane (collapsible) with a header stating "Source:
  English, revision 7" and a **Mark synced** action after updating.
- A **Stale** badge with the source diff since last sync (reuses MC.10's diff view).
- Metadata panel shows the locale (immutable) and locale-specific slug with URL preview.

**Public site**

- Language switcher in the page header of translated pages: a menu button labelled with the current
  language endonym, listing available languages with `lang` and `hreflang` attributes; keyboard
  accessible; no auto-redirect based on browser language (an explicit user choice, respecting SEO
  guidance and avoiding cloaking).
- RTL pages mirror layout via the existing RTL support.

**States** — no translations ("This article is only available in English — Add translation");
stale ("English changed 4 days ago — review this translation"); partial group (switcher lists only
published locales).

**Copy & i18n keys** — `marketingContent.translations.*` (`add`, `stale`, `sourcePane.title`,
`markSynced`, `switcher.label`, `fallbackNotice`).

## 11. AI / ML Considerations

Explicitly excluded: no machine translation in the publish path. If MT is ever introduced it must
produce a **draft** that a human reviews and approves, must be labelled in the workspace, and must be
disclosed under the AI disclosure framework — a machine-translated help article published without
review is a support-quality and trust risk, not a cost saving.

## 12. Integration Points

- **Server:** `internal/repos/marketingcontent` (locale-aware queries, tsvector trigger),
  `internal/service/marketingcontent` (translation creation, staleness), `internal/l10n` (locale list
  agreement), MC.13's contextual help (locale + fallback).
- **Client:** workspace list/editor changes; `lib/marketing-content-api.ts`.
- **`www`:** `src/lib/content-source.ts` (locale param), `route-manifest.tsx` (locale-prefixed route
  families), `entry-server.tsx` (`lang`/`dir`), `scripts/seo-artifacts.mjs` (hreflang + sitemap
  alternates), header component (switcher).
- **Related plan:** SEO.17 defines the site-wide hreflang and international strategy; this plan
  implements it for content pages and must not contradict it.

## 13. Dependencies & Sequencing

- Must ship after: MC.1 (columns), MC.3 (locale params), MC.10 (editor to extend). MC.12 should land
  first so hreflang is added to a stable artefact pipeline.
- Must ship before: any non-English content launch.
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Building i18n before any translation exists wastes effort | M | M | Cost is bounded and the alternative (retrofitting live URLs later) is worse; FR-15 guarantees zero impact until used. If no locale is committed within two quarters, this plan can be deferred without blocking others |
| Wrong or missing `hreflang` harms rankings | M | H | Reciprocity asserted in tests (AC-3); `x-default` required; SEO.17 alignment; Search Console international report checked after launch |
| Generated `search_tsv` → trigger conversion regresses search | M | M | Behaviour-parity test on English rows before/after; index rebuild verified |
| Stale translations mislead users worse than no translation | **H** | H | Staleness surfaced in the workspace, health view and (optionally) an on-page "last reviewed" note; policy: unpublish rather than let a translation drift beyond one major product change |
| RTL layout breaks on content pages | M | M | RTL locale added to the visual test matrix; the editor preview shares the site styles |
| Locale code mismatch between app and site | M | L | Single allowlist derived from `internal/l10n`; a test asserts agreement |

## 15. Rollout Plan

- **Feature flag:** `ff_marketing_content` plus `marketing_content_locales_enabled` (default off).
  With it off, the API ignores locale parameters beyond `en` and no locale UI appears.
- **Sequencing:** schema + locale table → API locale params → build locale routing + hreflang (still
  no non-English content) → workspace/editor UI → pilot translation of 5 help articles → enable.
- **Dogfood:** translate the five most-read help articles into one target locale; measure traffic and
  staleness handling for a month before expanding.
- **GA criteria:** all ACs; parity build unchanged for English; Search Console shows no hreflang
  errors after two weeks.
- **Rollback:** disable the locales setting → non-English pages stop being generated on the next
  build (and, if already live, are redirected to their English equivalents rather than 404ing —
  handled by an automatic redirect rule when a locale is disabled).

## 16. Test Plan

- **Unit** — locale validation and allowlist; path derivation per locale; hreflang set construction
  (reciprocity, `x-default`); staleness computation; locale-aware text metrics; ts_config resolution.
- **Integration (DB)** — translation creation and group linkage; per-locale slug uniqueness; tsvector
  trigger across locales; search with Spanish stemming; redirect scoping per locale.
- **End-to-end** — Playwright: create a translation, publish it, verify the built page, the switcher,
  the `lang`/`dir` attributes, and the sitemap alternates; verify English output unchanged when no
  translation exists.
- **Security** — locale path traversal attempts; unsupported locale rejection; no locale-based
  content leakage of unpublished translations.
- **Accessibility** — `lang` correctness per page; switcher keyboard/screen-reader script; RTL visual
  and axe checks on the site and the editor preview.
- **Performance** — build time and sitemap size with 2 and 5 locales simulated; index endpoint
  payload growth.
- **Manual exploratory** — a native speaker reviews one translated page end-to-end including the
  switcher, metadata and search behaviour.

## 17. Documentation & Training

- Help article: "Translating a help article" (workflow, staleness, when to unpublish).
- `www/docs/url-policy.md` — locale prefix policy and slug localization rules.
- `www/docs/structured-data.md` / `site-generation.md` — hreflang and sitemap alternates.
- Internal: locale onboarding checklist (add locale row, ts_config, endonym, RTL flag, review
  intervals).

## 18. Open Questions

1. Path prefix (`/es/…`) vs subdomain vs ccTLD? (Proposed: path prefix — cheapest, keeps one
   deployment and one authority; confirm against SEO.17.)
2. Do we localize slugs or keep English slugs under a locale prefix? (Proposed: localize — better for
   local search; the redirect machinery already handles slug changes.)
3. Which locale ships first, and is there a committed market driving it? (Open — the answer decides
   whether this plan runs now or is deferred; see the first risk row.)
4. Do we want a TMS/XLIFF export for agency translation? (Proposed: v2, once volume justifies it.)
5. Should the site auto-detect browser language and suggest (not redirect) a translation? (Proposed:
   a dismissible banner, never an automatic redirect.)

## 19. References

- Files this work touches: `server/migrations/484_*`, `server/internal/repos/marketingcontent/*`,
  `server/internal/service/marketingcontent/*`, `clients/web/src/pages/admin/marketing-content/*`,
  `www/src/lib/content-source.ts`, `www/src/lib/route-manifest.tsx`, `www/src/entry-server.tsx`,
  `www/scripts/seo-artifacts.mjs`, `www/src/components/header.tsx`.
- Related plans: [MC.1](MC.1-content-data-model-and-migrations.md),
  [MC.3](MC.3-public-content-read-api.md), [MC.10](MC.10-article-editor.md),
  [MC.12](MC.12-seo-parity-from-database.md), [MC.13](MC.13-docs-search-and-in-app-help.md);
  SEO.17 (international SEO & hreflang).
- Standards: BCP-47, Google hreflang guidance, sitemaps.org `xhtml:link` alternates, WCAG 2.1 AA
  (3.1.1 Language of Page, 3.1.2 Language of Parts).
