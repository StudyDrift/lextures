# SEO.11 — Marketplace Catalog SEO at Scale

> Implementation plan. Source: [docs/plan/seo/audit.md](../../plan/seo/audit.md) §S0 (F-2) and
> `www/docs/marketplace-seo.md` (plan MKT10, partially shipped).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.11 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | MAJOR |
| **Markets** | HS / HE (continuing ed) / K12 (supplemental) |
| **Status (today)** | COMPLETED (2026-08-11) — production SSG, incremental cache/recovery, hourly rebuilds, quality floor/reporting, catalog hubs, schema and UGC safeguards shipped |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform + Marketing |
| **Depends on** | SEO.1, SEO.2, SEO.3, SEO.5 |
| **Unblocks** | — |

---

## 1. Problem Statement

MKT10 built a competent course prerenderer — per-course title, description, canonical, OG tags and
`Course` JSON-LD, with unit tests and a loud-failure mode — and then the deploy workflow disabled it:
`.github/workflows/pages-www.yml` sets `SKIP_COURSE_PRERENDER: "1"`, so **no course page has ever been
prerendered in production** (audit F-2). Beyond turning it on, the catalog has no subject or level
hubs, no instructor pages, no pagination or facet canonical policy, and no quality floor — which
matters because a marketplace is the one part of our site that can generate thousands of URLs from
third-party content, and thin third-party pages at volume is precisely the shape the March 2026
scaled-content-abuse enforcement targeted
([research §7](../../plan/seo/research.md#7-content-strategy-concentration-beats-volume-utility-beats-pages)). Course
schema, meanwhile, is explicitly called out as still valuable for AI retrieval even after the
course-info rich result was retired
([research §5](../../plan/seo/research.md#5-structured-data-what-still-pays-what-is-dead)).

## 2. Goals

- Turn the existing prerenderer on in production and make it correct at catalog scale.
- Add the hub layer a catalog needs to be crawlable and browsable: subject, level, format, and
  instructor.
- Enforce a **quality floor** so thin or unfinished listings never enter the index — protecting the
  whole domain, not just the catalog.
- Keep the index fresh: a newly published course is indexable within hours, not on the next unrelated
  www deploy.
- Make courses the entities assistants retrieve for "where can I learn X" queries.

## 3. Non-Goals

- Marketplace product features (listing flow, coupons, payouts) — those are the
  [MKTC](../marketplace/README.md) plans and the completed MKT1–MKT10 set.
- Review collection mechanics (the review model already exists); this plan governs how reviews are
  *marked up and gated*.
- Paid course promotion or marketplace ads.
- Localised course pages (SEO.17).

## 4. Personas & User Stories

- **As a homeschool parent searching "online high school chemistry course"**, I want a Lextures course
  page in the results with price, length and level visible, so that I can evaluate it before clicking.
- **As a course creator**, I want my listing to be findable in Google and cited by assistants, so that
  publishing on Lextures is worth more than publishing elsewhere.
- **As someone asking an assistant "where can I learn AP statistics online"**, I want accurate,
  structured course data, so that the recommendation is real.
- **As a browsing learner**, I want subject and level hubs, so that I can explore without knowing a
  course's exact name.
- **As the platform owner**, I want unfinished or spammy listings excluded from the index, so that one
  bad creator cannot damage the domain.

## 5. Functional Requirements

**Turn it on and make it correct**

- **FR-1.** `SKIP_COURSE_PRERENDER` MUST be removed from the production workflow (SEO.1 FR-7). Course
  pages MUST be prerendered on every production deploy.
- **FR-2.** Prerendering MUST be **incremental**: only courses whose `updatedAt` changed since the last
  deploy are re-rendered; the rest are copied from the previous build. Full regeneration MUST be
  triggerable by a flag.
- **FR-3.** Course pages MUST be generated with bounded concurrency (default 8) and MUST complete for
  10,000 courses in ≤ 5 minutes.
- **FR-4.** On API failure, the build MUST reuse the previous deploy's course pages and warn (SEO.1
  FR-7) — never publish an empty catalog.

**Freshness**

- **FR-5.** A **scheduled rebuild** MUST run at least daily (workflow_dispatch + cron) so newly
  published courses appear without an unrelated www commit.
- **FR-6.** The server MUST emit a webhook (or the workflow MUST poll a lightweight
  `GET /api/v1/public/marketplace/courses/changes?since=`) so publish → deploy latency is ≤ 1 hour for
  new listings.
- **FR-7.** Every newly generated or materially changed course URL MUST be submitted to IndexNow on
  deploy (SEO.2 FR-17).

**Quality floor**

- **FR-8.** A course MUST meet **all** of these to be indexable; failing courses render for users but
  are emitted `noindex,follow` and excluded from sitemaps and `llms.txt`:

  | Check | Threshold |
  |---|---|
  | Description length | ≥ 300 characters of unique prose |
  | Description uniqueness | < 70% similarity to any other course by the same creator |
  | Structure | ≥ 3 modules or ≥ 5 content items |
  | Media | ≥ 1 course image meeting the SEO.14 dimensions |
  | Metadata | subject, level, language, and either a price or "free" set |
  | Creator | verified creator account |
  | Moderation | not flagged, not under review |

- **FR-9.** A course that becomes non-compliant after indexing MUST flip to `noindex` on the next
  build and be removed from sitemaps; if unpublished or deleted, the URL MUST return 410 and be
  removed from sitemaps (not left as a soft 404).
- **FR-10.** Creator-supplied HTML/markdown MUST be sanitised before rendering; outbound links in
  course descriptions MUST carry `rel="nofollow ugc noopener"`.
- **FR-11.** The quality report MUST be visible to creators — a listing-health panel in the creator UI
  showing exactly which checks fail and why the course is not indexable. (Search visibility is a
  creator benefit; hiding the criteria makes it feel arbitrary.)

**Hubs & facets**

- **FR-12.** Hub pages MUST exist at:
  - `/courses` — storefront (exists)
  - `/courses/subject/:subject` — ~20 subject hubs
  - `/courses/level/:level` — elementary, middle, high-school, college, adult
  - `/courses/format/:format` — self-paced, cohort, live (if applicable)
  - `/instructors/:slug` — instructor profile with their courses
- **FR-13.** Each hub MUST carry ≥150 words of unique, human-written orienting copy plus the course
  grid — a hub is a page, not a query result.
- **FR-14.** A hub MUST NOT be generated with fewer than **3 courses**; below that it is `noindex` and
  omitted from navigation.
- **FR-15.** Facet **combinations** (subject + level + price) MUST be client-side only, produce no
  distinct indexable URL, and MUST canonical to the primary hub. Only the single-dimension hubs in
  FR-12 are indexable.
- **FR-16.** Pagination MUST use real `/courses/subject/math?page=2` URLs with self-referential
  canonicals and `rel="prev"/"next"`-equivalent internal links; `page=2+` MUST NOT canonical to page 1
  (that hides deep courses from crawlers).
- **FR-17.** Sort-order parameters MUST NOT create indexable URLs.

**Schema**

- **FR-18.** `/courses` and every hub MUST emit `ItemList` with ≥3 `ListItem`s, sequential `position`,
  unique `url` (SEO.3 FR-15) — this is the surviving Course carousel requirement.
- **FR-19.** Course pages MUST emit `Course` with `provider` → Organization `@id`, `hasCourseInstance`
  (`courseMode`, `courseWorkload`), `educationalLevel`, `teaches`, `inLanguage`, `offers`
  (price + currency + availability), and `instructor` → `Person` when the instructor has a profile.
- **FR-20.** `AggregateRating` MAY be emitted **only** when the course has ≥5 genuine reviews from
  verified enrolled learners; below that threshold no rating markup is emitted. Individual `Review`
  nodes MUST include `author` and `datePublished`.
- **FR-21.** Instructor pages MUST emit `Person` with `worksFor`/`affiliation` and `hasCredential`
  only where verified.

**Copy quality**

- **FR-22.** Generated `<title>` and meta description MUST be templated but distinctive:
  `"{Course title} — {level} {subject} course | Lextures"`, description derived from the course's own
  summary via `truncateMetaDescription` (existing helper), never a generic template string.
- **FR-23.** Course pages MUST include platform-added context that no other site has: what's included,
  the enrolment path, refund/coupon terms, accessibility features, and links to the subject hub and
  related courses — so the page is not merely the creator's description restated.

## 6. Non-Functional Requirements

- **Performance** — course pages are static; the enrolment panel is the only interactive island.
  Course images must meet SEO.4's format and sizing rules.
- **Security** — creator content is untrusted: sanitise (FR-10), escape into JSON-LD (SEO.3 FR-3),
  `nofollow ugc` on outbound links. Prerendering must not expose draft or unlisted courses.
- **Privacy & Compliance** — instructor pages publish real names and bios; consent required and
  recorded (S04), with an opt-out that removes the profile and the `Person` node. No learner data
  appears on any public page; enrolment counts are only shown above a k-anonymity threshold.
- **Accessibility** — course cards need accessible names that are not just the image; the grid must be
  keyboard-navigable; filters must announce result counts; ratings must not be conveyed by stars alone
  (include the numeric value in text).
- **Scalability** — designed for 10k+ courses: sharded sitemaps (SEO.2 FR-9), incremental generation
  (FR-2), and hub pagination (FR-16).
- **Reliability** — API failure degrades to previous output (FR-4); a single malformed course must not
  fail the whole build (skip it, warn, and report).
- **Observability** — per-deploy: courses generated, skipped, `noindex`ed and why; per-course: index
  status, impressions, clicks, enrolments from organic. Creator-facing health panel (FR-11).
- **Maintainability** — extend `prerender-courses.mjs` into the SEO.1 generator rather than forking it;
  keep the pure-helper + `node --test` pattern already established.
- **Internationalization** — `inLanguage` per course; SEO.17 will add `hreflang` for localised
  listings.
- **Backward compatibility** — `/courses/:slug` URLs unchanged; `www/docs/marketplace-seo.md` is
  superseded and must be updated rather than left describing the disabled state.

## 7. Acceptance Criteria

- **AC-1.** *Given* a production deploy, *When* it completes, *Then* every listed+published course has
  an HTML file returning 200 with its own title, description, canonical and `Course` JSON-LD.
- **AC-2.** *Given* a course that fails any FR-8 check, *When* generated, *Then* it renders for users,
  carries `noindex,follow`, is absent from all sitemaps and `llms.txt`, and appears in the quality
  report with the failing check named.
- **AC-3.** *Given* a course is unpublished, *When* the next build runs, *Then* its URL returns 410 and
  is removed from the sitemap.
- **AC-4.** *Given* a newly published course, *When* one hour has passed, *Then* its URL exists, is in
  the sitemap, and has been submitted to IndexNow.
- **AC-5.** *Given* a subject hub with 2 courses, *When* generated, *Then* it is `noindex` and not
  linked from navigation; *Given* 3+, *Then* it is indexable with ≥150 words of unique copy.
- **AC-6.** *Given* `/courses/subject/math?page=2`, *When* inspected, *Then* its canonical is itself,
  not page 1, and its courses are linked from crawlable HTML.
- **AC-7.** *Given* a course with 3 reviews, *When* its schema is validated, *Then* no
  `AggregateRating` is emitted; *Given* 5+ verified reviews, *Then* it is emitted with a matching
  visible rating.
- **AC-8.** *Given* a course description containing `<script>` or a `javascript:` link, *When*
  rendered, *Then* it is sanitised and no script executes, and JSON-LD remains well-formed.
- **AC-9.** *Given* 10,000 courses, *When* an incremental build runs with 50 changed, *Then* it
  completes in ≤ 90 seconds and only 50 files change.
- **AC-10.** *Given* a creator viewing listing health, *When* their course fails a check, *Then* the
  panel names the check, the threshold, and the fix.

## 8. Data Model

No new tables required for www. Server-side additions:

| Change | Where | Purpose |
|---|---|---|
| `GET /api/v1/public/marketplace/courses/changes?since=` | marketplace handlers | FR-6 incremental discovery |
| `indexable` computed field on public course payload | marketplace handlers | FR-8 evaluated server-side so the creator UI and the generator agree |
| `qualityChecks[]` on the creator-facing course payload | creator API | FR-11 listing-health panel |
| Subject/level/format facet counts endpoint | marketplace handlers | FR-12 hub generation |

Build artefacts: `dist/.course-cache.json` (SEO.1), `dist/.catalog-quality.json` (per-course check
results, feeding SEO.16).

## 9. API Surface

```
GET /api/v1/public/marketplace/courses?page=&perPage=&updatedSince=
GET /api/v1/public/marketplace/courses/{slug}
GET /api/v1/public/marketplace/courses/changes?since={iso8601}      # new
GET /api/v1/public/marketplace/facets                               # new: subject/level/format + counts
GET /api/v1/public/marketplace/instructors/{slug}                   # new: profile + course list
```

- Auth: public, unauthenticated; only listed+published courses.
- Rate limits: build traffic identified by `User-Agent: lextures-www-prerender/<version>`, capped at
  20 req/s; `changes` endpoint cheap enough for hourly polling.
- Response shapes documented in OpenAPI; `jsonLd` field must omit `@context` (SEO.3 §9).

## 10. UI / UX

- **New pages:** ~20 subject hubs, 5 level hubs, format hubs, instructor profiles, hub pagination.
- **Modified:** `/courses` storefront gains hub links and orienting copy; course detail gains
  breadcrumbs (`Courses › Math › <course>`), related courses, and the platform-context sections from
  FR-23; creator UI gains the listing-health panel (FR-11).
- **Flows**
  1. Search "high school chemistry course" → course page → enrol.
  2. `/courses` → subject hub → level filter (client-side) → course → enrol.
  3. Creator publishes → health panel shows "not yet indexable: description too short" → fixes →
     indexable next build.
- **States** — hub with <3 courses is hidden from nav; course grid empty/loading/error states already
  exist and must keep zero CLS; a `noindex` course shows creators (not learners) a health notice.
- **Responsive** — grids reflow; filters become a bottom sheet on mobile.
- **Accessibility** — card accessible names, keyboard grid navigation, announced filter result counts,
  numeric rating text alongside stars.
- **Copy & i18n** — `www.courses.hubs.*`, `www.courses.health.*`; hub orienting copy is
  human-written per subject, not templated.

## 11. AI / ML Considerations

- **Course schema is the payload assistants use** for "where can I learn X" answers; FR-19's
  completeness (level, workload, language, price, instructor) is what makes our listings usable versus
  a bare title and description.
- **The quality floor is an AI-visibility decision as much as a spam-avoidance one.** Assistants
  penalise domains with a high proportion of thin pages; a marketplace can flip a domain's profile
  quickly, which is why FR-8 gates at the page level rather than trusting aggregate quality.
- No model is used to generate course copy. If AI-assisted listing copy is offered to creators later,
  it must not produce near-duplicate descriptions across courses — FR-8's similarity check is the
  guard, and it should be surfaced in that flow.

## 12. Integration Points

- **External:** none new.
- **Internal modules touched:** `www/scripts/prerender-courses.mjs` (merged into the SEO.1 generator),
  `www/src/pages/courses-page.tsx`, `www/src/pages/course-detail-page.tsx`,
  `www/src/components/courses/*`, `www/src/lib/marketplace-api.ts`, `www/src/lib/courses-copy.ts`,
  `.github/workflows/pages-www.yml`, marketplace handlers in `server/internal/httpserver`,
  creator course-settings UI in `clients/web`.
- **Events:** course publish/unpublish → rebuild trigger (FR-6); IndexNow submission (FR-7).

## 13. Dependencies & Sequencing

- **Must ship after:** [SEO.1](SEO.1-static-rendering-and-crawlability.md) (generator),
  [SEO.2](SEO.2-crawler-access-sitemaps-and-llms-txt.md) (sharded sitemaps, IndexNow),
  [SEO.3](SEO.3-structured-data-and-entity-graph.md) (`ItemList`, `Person`, graph),
  [SEO.5](SEO.5-information-architecture-and-internal-linking.md) (hub IA, breadcrumbs).
- **Must ship before:** nothing; [SEO.10](SEO.10-programmatic-utility-pages.md) standards pages link
  to course hubs and benefit from shipping after.
- **Shared infra:** scheduled workflow; creator consent records for instructor profiles.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Thin creator listings drag down the whole domain | **H** | **H** | FR-8 quality floor with `noindex` default-deny; FR-11 makes it actionable for creators; `.catalog-quality.json` monitored in SEO.16 |
| Facet URLs explode into an index bloat problem | M | H | FR-15 single-dimension hubs only; combinations client-side; sort params non-indexable; `robots.txt` disallow on parameter patterns (SEO.2 FR-3) |
| Prerendering 10k courses blows the build budget | M | M | FR-2 incremental + FR-3 bounded concurrency; AC-9 asserts the budget |
| Creator content injects script or spam links | M | H | FR-10 sanitisation + `nofollow ugc`; AC-8 hostile-input test |
| Unpublished courses leave soft 404s | M | M | FR-9 410 + sitemap removal; verified in the post-deploy smoke check |
| Instructor names published without consent | L | H | Consent recorded (S04) with an opt-out that removes the page and schema |
| Rating markup on 1–2 reviews looks manipulative | M | M | FR-20 ≥5 verified-enrolment reviews before any `AggregateRating` |

## 15. Rollout Plan

- **Feature flag:** server-side `marketplace_seo_quality_floor` (default on) so the floor can be
  loosened if it excludes too many legitimate listings; www itself has no runtime flags.
- **Sequencing**
  1. Remove `SKIP_COURSE_PRERENDER`; deploy with the existing per-course output. Verify in GSC.
  2. Add incremental generation + concurrency; verify build time.
  3. Add the quality floor in **report-only** mode; review how many live courses would be excluded and
     tune thresholds with the marketplace owner.
  4. Enable the floor; ship the creator health panel in the same release (never gate visibility
     without telling creators why).
  5. Ship subject/level hubs + breadcrumbs + `ItemList`.
  6. Ship instructor profiles (consent first).
  7. Enable scheduled + webhook rebuilds; wire IndexNow.
- **Dogfood:** run the floor against the current live catalog and review a sample of excluded courses
  by hand before step 4.
- **GA criteria:** AC-1…AC-10; ≥70% of eligible course URLs indexed within 60 days; zero index-bloat
  warnings in GSC.
- **Rollback:** flag off returns to indexing all published courses; prerendering can be reverted to
  the previous build's output.

## 16. Test Plan

- **Unit** (extending `prerender-courses.test.mjs`) — quality-check evaluation per rule; similarity
  detection; title/description templating and truncation; `ItemList` construction incl. position
  sequencing; 410 emission for unpublished; incremental change detection.
- **Integration** — build against a mock catalog of 10k courses (AC-9); hub generation thresholds
  (AC-5); pagination canonicals (AC-6); sitemap/`llms.txt` exclusion of `noindex` courses (AC-2).
- **End-to-end (Playwright)** — course page renders fully with JS disabled; enrolment island hydrates;
  breadcrumbs and related courses present; creator health panel shows the failing check (AC-10).
- **Security** — hostile course description/title (AC-8); assert `rel="nofollow ugc noopener"` on
  creator outbound links; assert unlisted/draft courses are never generated.
- **Accessibility** — axe on storefront, a hub, a course page and the creator health panel; keyboard
  grid navigation; announced filter counts; rating text equivalence.
- **Performance / load** — build-time budget (AC-9); course page meets the interactive budget;
  marketplace API stays under 20 req/s during a full regeneration.
- **Manual exploratory** — review 20 generated course pages for copy quality; confirm GSC coverage and
  the absence of "Duplicate without user-selected canonical" on hub pagination.

## 17. Documentation & Training

- Rewrite `www/docs/marketplace-seo.md` to describe the shipped state (it currently documents a
  disabled prerender and a flat sitemap).
- Creator-facing help article: "Why isn't my course showing up in Google?" — the quality checks, in
  plain language, in the `marketplace` help category (SEO.7).
- Runbook: forcing a full catalog regeneration; handling a spam listing; responding to index bloat.
- Update the marketplace team's launch checklist with the indexability criteria.

## 18. Open Questions

1. What proportion of the current live catalog would fail FR-8? (Must be measured in report-only mode
   before enabling — if it is most of them, thresholds or creator tooling need work first.)
2. Do we build the webhook (FR-6) or poll `changes`? Polling is simpler and hourly is within target.
3. Do instructor profiles require opt-in or opt-out? (Recommendation: opt-in, with the SEO benefit
   explained at listing time.)
4. Should `/courses` move under `/marketplace` (SEO.5 open question 4)? Decide before hubs are built.
5. What is the k-anonymity threshold for showing enrolment counts publicly?

## 19. References

- Existing files: `www/scripts/prerender-courses.mjs`, `www/scripts/prerender-courses.test.mjs`,
  `www/docs/marketplace-seo.md`, `www/src/pages/courses-page.tsx`,
  `www/src/pages/course-detail-page.tsx`, `www/src/components/courses/*`,
  `www/src/lib/marketplace-api.ts`, `.github/workflows/pages-www.yml`
- Audit findings: [F-2](../../plan/seo/audit.md#f-2-the-course-prerender-that-exists-is-disabled-in-production)
- Research: [§5](../../plan/seo/research.md#5-structured-data-what-still-pays-what-is-dead),
  [§7](../../plan/seo/research.md#7-content-strategy-concentration-beats-volume-utility-beats-pages)
- External: [Google — Course list structured data](https://developers.google.com/search/docs/appearance/structured-data/course),
  [Google — Faceted navigation best practices](https://developers.google.com/search/docs/crawling-indexing/crawling-managing-faceted-navigation),
  [Google — Review snippet guidelines](https://developers.google.com/search/docs/appearance/structured-data/review-snippet)
- Related plans: [SEO.1](SEO.1-static-rendering-and-crawlability.md),
  [SEO.2](SEO.2-crawler-access-sitemaps-and-llms-txt.md),
  [SEO.3](SEO.3-structured-data-and-entity-graph.md),
  [MKTC — course coupon codes](../marketplace/README.md),
  [MKT1–MKT10 (completed)](../../completed/marketplace/)
