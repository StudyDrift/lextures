# SEO.3 — Structured Data & Brand Entity Graph

> Implementation plan. Source: [docs/plan/seo/audit.md](../../plan/seo/audit.md) §S1 (F-8, F-9) and §S2 (F-11).
> **Shipped** 2026-08-11: composable `@graph` JSON-LD, Organization/WebSite/SoftwareApplication site-wide, `/about` entity home, author registry + `/authors/*`, Article/FAQ/Offer/Course merge, build-time graph validation. Wikidata + expanded `sameAs` wait on SEO.13 profile claims. See `www/docs/structured-data.md`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.3 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | SHIPPED — `@graph` entity graph, about/authors, bylines, build validation (2026-08-11); Wikidata/`sameAs` expansion is SEO.13 |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web platform + Marketing |
| **Depends on** | SEO.1 |
| **Unblocks** | SEO.9, SEO.10, SEO.11, SEO.12, SEO.13, SEO.14 |

---

## 1. Problem Statement

`document-head.ts` supports exactly one JSON-LD node keyed to the hard-coded element id
`course-json-ld`, so a page structurally **cannot** emit two schema nodes (audit F-8). There is no
`Organization`, `WebSite`, `BreadcrumbList`, `Article`, `SoftwareApplication`, `Offer`, or `Person`
anywhere, no `/about` entity home, no `sameAs` array, and no Wikidata item (F-9). Every blog post is
bylined "Lextures Team" — an entity no knowledge graph can resolve (F-11). This matters more than it
used to: branded mentions correlate **0.664** with AI-Overview citation versus **0.218** for
backlinks, and entity recognition now precedes everything else in citation selection
([research §4](../../plan/seo/research.md#4-entity-seo-is-the-highest-roieffort-ratio-available)). Entity work is
also the cheapest lever we have and the slowest to pay out — 60–180 days to a knowledge panel — so it
must start early.

## 2. Goals

- Make Lextures a **resolvable entity**: one canonical entity home, a complete `Organization` node,
  a verified `sameAs` set, and a Wikidata item.
- Replace the single-node JSON-LD plumbing with a composable graph so any page can emit multiple
  connected nodes with stable `@id`s.
- Attach real, credentialed **human authors** to every editorial page, with `Person` entities that
  themselves resolve.
- Emit the schema types that still pay in late 2026 — and deliberately emit the ones that no longer
  produce rich results but are still parsed by LLMs.
- Ship a machine-readable trust/compliance graph (VPAT, security, privacy) that answers the exact
  questions institutional buyers ask assistants.

## 3. Non-Goals

- Chasing deprecated rich results. `FAQPage` (rich result retired 7 May 2026) and `HowTo` (retired
  2023) are emitted for **machine comprehension only**; no plan milestone depends on a SERP feature
  from them.
- Review/rating schema on our own marketing pages (self-serving review markup is a spam vector).
  Course reviews on marketplace pages are in scope and are genuine first-party data — see SEO.11.
- Content authorship itself (SEO.8) or off-site profile creation (SEO.13) — this plan defines the
  schema they populate.

## 4. Personas & User Stories

- **As an AI assistant asked "who makes Lextures?"**, I want a resolvable organisation entity with
  founder, founding date, and verified profiles, so that I answer confidently instead of hedging.
- **As a district technology director**, I want the assistant's answer about Lextures' accessibility
  conformance to cite our VPAT, so that I can forward it to procurement.
- **As a reader of a Lextures article**, I want to see who wrote it and why they are qualified, so
  that I trust the claim.
- **As a search engine**, I want breadcrumb, article, and organisation nodes that reference each
  other by `@id`, so that I can place each page inside the site's structure.
- **As a marketing engineer**, I want to add a schema node by returning it from the route manifest,
  so that schema is never a hand-edited `<script>` tag.

## 5. Functional Requirements

**Plumbing**

- **FR-1.** `document-head.ts` MUST support an ordered **array** of JSON-LD nodes, emitted as a single
  `<script type="application/ld+json">` containing `{"@context":"https://schema.org","@graph":[…]}`.
  The `course-json-ld` single-id design MUST be removed.
- **FR-2.** Every node MUST carry a stable, absolute `@id` (e.g.
  `https://lextures.com/#organization`, `https://lextures.com/about#founder`,
  `https://lextures.com/blog/x#article`) so nodes reference each other rather than duplicating data.
- **FR-3.** Serialisation MUST escape `<`, `>`, `&`, and the `</script` sequence, and MUST be
  covered by a unit test using a hostile course title.
- **FR-4.** Schema MUST be emitted **at build time** into the served HTML (SEO.1 FR-5), and reapplied
  identically on client navigation.
- **FR-5.** The build MUST validate every emitted graph against a JSON Schema for the node types used,
  and MUST fail on: missing required property, non-absolute `@id`, or a dangling `@id` reference.

**Site-wide entity graph** (emitted on every page)

- **FR-6.** An `Organization` node MUST be emitted with: `name`, `alternateName`, `url`, `logo`
  (PNG ≥112×112), `description`, `foundingDate`, `founder` (→ `Person` `@id`), `email`,
  `contactPoint` (sales + support, `areaServed`, `availableLanguage`), `knowsAbout` (12–20 topic
  strings covering adaptive learning, IRT, LMS, standards alignment, WCAG, FERPA…), and `sameAs`.
- **FR-7.** `sameAs` MUST list only **verified, owned or claimed** profiles: GitHub
  (`StudyDrift/lextures`), LinkedIn company page, X, YouTube, Crunchbase, G2, Capterra, Wikidata item.
  Unclaimed third-party listings MUST NOT be included.
- **FR-8.** A `WebSite` node MUST be emitted with `publisher` → Organization `@id`, and a
  `SearchAction` `potentialAction` once site search exists (omit until then rather than pointing at a
  broken endpoint).
- **FR-9.** A `SoftwareApplication` node MUST describe the product: `applicationCategory:
  "EducationalApplication"`, `operatingSystem` (Web, iOS, Android), `featureList`, `offers` →
  `AggregateOffer` reflecting real published pricing, and `softwareHelp` → `/docs`.
- **FR-10.** Every page except the homepage MUST emit `BreadcrumbList` matching the visible
  breadcrumb trail introduced in [SEO.5](../../plan/seo/SEO.5-information-architecture-and-internal-linking.md).

**Page-type schema**

- **FR-11.** Blog posts MUST emit `Article` (or `TechArticle` where appropriate) with `headline`,
  `description`, `datePublished`, `dateModified`, `author` → `Person` `@id`, `publisher` →
  Organization `@id`, `image`, `mainEntityOfPage`, `wordCount`, `articleSection`, and `citation[]`
  listing the primary sources referenced in the body.
- **FR-12.** Help-center articles MUST emit `TechArticle` plus, where the article is procedural,
  `HowTo` with real `step` items — emitted for LLM comprehension despite producing no rich result.
- **FR-13.** Pages with a Q&A section MUST emit `FAQPage` with `mainEntity` `Question`/`Answer`
  pairs, for the same reason. Answers MUST match the visible on-page text verbatim.
- **FR-14.** `/pricing` MUST emit `Product`/`Offer` nodes with `priceCurrency`, `price` or
  `priceSpecification` (`UnitPriceSpecification` with `unitText: "student"`), `availability`, and
  `eligibleCustomerType` per segment. Prices MUST be generated from
  `www/src/lib/institution-pricing.ts` so schema cannot drift from the visible table.
- **FR-15.** `/courses` MUST emit an `ItemList` carousel with ≥3 `ListItem`s, each with a sequential
  `position` and a unique `url`; each `/courses/:slug` keeps its server-built `Course` node, extended
  with `provider` → Organization `@id`, `hasCourseInstance`, `educationalLevel`, and `teaches`.
- **FR-16.** Trust pages MUST emit machine-readable conformance: `/accessibility` and
  `/accessibility/vpat` emit `WebPage` + `CreativeWork` for the VPAT document with
  `accessibilityAPI`/`accessibilityFeature`/`conformsTo` (WCAG 2.2 AA URL); `/security` emits
  `WebPage` + `Organization.hasCredential` where certifications exist. No unearned claim may be
  encoded.
- **FR-17.** `/privacy` and `/terms` MUST emit `DigitalDocument` with `dateModified` matching the
  published version history.

**Author & entity home**

- **FR-18.** A new `/about` page MUST exist as the **entity home**: what Lextures is, who builds it,
  founding date, mission, funding/ownership status, contact, and links to every `sameAs` profile.
  This is the URL we point Wikidata and every directory at.
- **FR-19.** Author pages MUST exist at `/authors/:slug`, each emitting `Person` with `name`,
  `jobTitle`, `description`, `image`, `knowsAbout`, `alumniOf`/`hasCredential` where real, `sameAs`
  (LinkedIn, ORCID, Google Scholar, GitHub), and `worksFor` → Organization `@id`.
- **FR-20.** Blog and help front-matter MUST move from `author: "Lextures Team"` to
  `author: <author-slug>`, validated at build time against the author registry. A build MUST fail on
  an unknown slug. Existing five posts MUST be reattributed to real people.
- **FR-21.** Editorially reviewed pages SHOULD carry `reviewedBy` → `Person` and a visible
  "Reviewed by … on …" line, used for anything making a pedagogical or compliance claim.
- **FR-22.** A **Wikidata item** for Lextures MUST be created, referencing the `/about` page as
  official website, with properties for inception, founder, industry, and programming language, and
  sourced by independent references (press, directory listings) rather than self-published claims
  alone. Its QID MUST be added to `sameAs` and to `identifier`.

## 6. Non-Functional Requirements

- **Performance** — total JSON-LD payload ≤ 12 KB per page; it is inline in HTML so it counts against
  the SEO.4 LCP budget. Course pages with long descriptions MUST truncate `description` at 300 chars.
- **Security** — all schema values pass the escaping in FR-3. Course-derived fields are untrusted
  input. `sameAs` values are a fixed allowlist in code, never user input.
- **Privacy & Compliance** — `Person` schema publishes real names and credentials; every author must
  give written consent, recorded in the consent ledger
  ([S04](../../plan/standards/S04-unified-consent-preference-ledger.md)). No author's home location, personal
  email, or photo is published without consent. GDPR erasure of an author must be executable: the
  registry supports a `retired` state that removes the page, keeps the byline as text, and drops the
  `Person` node.
- **Accessibility** — schema is invisible; the *visible* byline, review line, and breadcrumb it
  mirrors must meet WCAG 2.2 AA (they are real content, per FR-13's "matches visible text" rule).
- **Scalability** — the graph builder is pure and O(nodes); course pages compose from cached data.
- **Reliability** — build-time validation (FR-5) means a malformed graph cannot deploy.
- **Observability** — `.seo-manifest.json` records `schemaTypes[]` per URL; SEO.15 tracks GSC
  "Unparsable structured data" and rich-result eligibility counts.
- **Maintainability** — one module `www/src/lib/schema/` with a file per node type, pure functions,
  `node --test` coverage. No schema string literals outside that directory.
- **Internationalization** — `inLanguage` on every node; `Organization.availableLanguage` reflects
  supported product locales; ready for SEO.17.
- **Backward compatibility** — the existing server-built course `jsonLd` payload keeps working; the
  builder merges it into the graph rather than replacing it. `JSON_LD_SCRIPT_ID` is removed — check
  `document-head.test.mjs` and the prerender test for references.

## 7. Acceptance Criteria

- **AC-1.** *Given* any deployed page, *When* I extract its JSON-LD, *Then* it parses as a single
  `@graph`, every node has an absolute `@id`, and every `@id` reference resolves within the graph or
  to a documented cross-page `@id`.
- **AC-2.** *Given* the homepage, *When* I run Google's Rich Results Test and Schema.org validator,
  *Then* zero errors are reported for `Organization`, `WebSite`, and `SoftwareApplication`.
- **AC-3.** *Given* `/courses`, *When* validated, *Then* an `ItemList` with ≥3 `ListItem`s is present,
  each with unique `url` and sequential `position` starting at 1.
- **AC-4.** *Given* a blog post, *When* validated, *Then* `Article.author` resolves to a `Person` node
  whose `@id` matches an existing `/authors/:slug` page, and `citation[]` is non-empty for any post
  making a statistical claim.
- **AC-5.** *Given* a build where a post's `author` slug is not in the registry, *When* CI runs,
  *Then* the build fails naming the post and the unknown slug.
- **AC-6.** *Given* `/pricing`, *When* I compare the `Offer` prices to the rendered pricing table,
  *Then* they are identical, because both derive from `institution-pricing.ts`.
- **AC-7.** *Given* a course whose title contains `</script><img src=x onerror=alert(1)>`, *When* the
  page is generated, *Then* the JSON-LD block is well-formed and the payload does not break out of
  the script element.
- **AC-8.** *Given* `/about`, *When* it is live, *Then* it is reachable in ≤1 click from the footer,
  emits the Organization node with ≥6 `sameAs` entries, and is the URL listed as official website on
  the Wikidata item.
- **AC-9.** *Given* GSC 30 days post-launch, *When* I open Enhancements, *Then* zero
  "Unparsable structured data" errors are present.
- **AC-10.** *Given* an author exercises erasure, *When* their registry entry is set to `retired`,
  *Then* `/authors/:slug` returns 410/404, their `Person` node disappears from all graphs, and no
  build fails.

## 8. Data Model

No database changes. New build-time registries:

```
www/src/content/authors/
  chase-willden.md          # front-matter: name, slug, jobTitle, bio, credentials,
                            # sameAs[], image, consentRecordedAt, status: active|retired
www/src/lib/schema/
  organization.ts  website.ts  software-application.ts  breadcrumb.ts
  article.ts  tech-article.ts  how-to.ts  faq.ts  offer.ts  course.ts
  person.ts  conformance.ts  graph.ts   # merge + validate + serialise
www/src/lib/schema/ids.ts   # canonical @id constants — the only place @ids are spelled
```

Front-matter change (breaking, migrated in one commit):

```diff
- author: "Lextures Team"
+ author: chase-willden
+ reviewedBy: <slug>        # optional
+ updated: 2026-09-14       # feeds dateModified and sitemap lastmod (SEO.2 FR-7)
```

## 9. API Surface

- No new Lextures HTTP routes.
- The existing `GET /api/v1/public/marketplace/courses/{slug}` already returns a server-built
  `jsonLd`. Requirement on the server side: that payload MUST include `@id` and MUST NOT include
  `@context` (the graph builder owns the context). If it currently emits `@context`, the builder
  strips it — tracked as a small server change in `server/internal/httpserver` marketplace handlers.
- Rate limits unchanged. OpenAPI: document the `jsonLd` field shape if not already described.

## 10. UI / UX

- **New pages:** `/about` (entity home), `/authors` (index), `/authors/:slug` (author profile).
- **Modified pages:** every editorial page gains a visible byline block — author photo, name, role,
  one-line bio, publish date, "Updated" date, and optional "Reviewed by". Blog index shows author.
- **Flows**
  1. Reader finishes an article → sees byline → clicks author → author page → other articles by them.
  2. Buyer lands on `/about` from an AI citation → sees who we are → routes to `/security`,
     `/accessibility`, `/request-information`.
- **States** — author with no photo falls back to initials avatar; retired author renders byline as
  plain text with no link.
- **Responsive** — byline collapses to a single line under 640 px with the avatar inline.
- **Accessibility** — author avatar `alt` is the author's name (or empty if the name is adjacent
  text); byline is a `<footer>` within `<article>`; the "Reviewed by" line is not a link-only row.
- **Copy & i18n** — keys `www.byline.writtenBy`, `www.byline.reviewedBy`, `www.byline.updated`,
  `www.authors.*`, `www.about.*`.

## 11. AI / ML Considerations

No model calls. Design notes specific to AI consumption:

- The `@graph` form is deliberate: assistants that extract JSON-LD do better with one connected graph
  than several disjoint script blocks.
- `knowsAbout` on Organization and Person is the cheapest topical-authority declaration available;
  it MUST reflect subjects we actually publish on (SEO.8 pillars), otherwise it is a false claim.
- `citation[]` on Article (FR-11) directly encodes the highest-impact factor from
  [research §3](../../plan/seo/research.md#3-what-actually-earns-an-ai-citation) (+132% from authoritative citations)
  in machine-readable form.
- **Anti-goal:** no schema property may assert something not visible and true on the page. That is
  both a Google structured-data policy violation and, under the May 2026 spam-policy extension, in
  scope for AI-surface enforcement.

## 12. Integration Points

- **External:** Wikidata, Crunchbase, G2, Capterra, LinkedIn, GitHub, YouTube (profiles must exist
  before being listed in `sameAs` — coordinate with [SEO.13](SEO.13-offsite-entity-mentions-and-digital-pr.md)).
- **Internal modules touched:** `www/src/lib/document-head.ts`, `www/src/lib/use-document-head.ts`,
  `www/src/lib/route-manifest.ts`, `www/src/utils/blog.ts`, `www/src/utils/docs.ts`,
  `www/src/lib/institution-pricing.ts`, `www/src/lib/vpat-data.ts`,
  `www/src/pages/*`, `www/scripts/generate-site.mjs`, marketplace course handler in
  `server/internal/httpserver`.
- **Events:** none.

## 13. Dependencies & Sequencing

- **Must ship after:** [SEO.1](SEO.1-static-rendering-and-crawlability.md) (multi-node JSON-LD needs
  the manifest and build-time emission).
- **Must ship before:** [SEO.9](../../plan/seo/SEO.9-comparison-alternatives-and-integration-pages.md),
  [SEO.10](../../plan/seo/SEO.10-programmatic-utility-pages.md), [SEO.11](../../plan/seo/SEO.11-marketplace-catalog-seo.md),
  [SEO.12](../../plan/seo/SEO.12-original-research-and-data-program.md) (Dataset/ScholarlyArticle schema),
  [SEO.14](../../plan/seo/SEO.14-multimodal-video-images-and-social-assets.md) (VideoObject).
- **Coordinates with:** [SEO.13](SEO.13-offsite-entity-mentions-and-digital-pr.md) — `sameAs` entries
  require the profiles to exist and be claimed first.
- **Shared infra:** consent records for author `Person` data (S04).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Schema asserts something untrue (e.g. a certification we do not hold) | M | **H** | FR-16 "no unearned claim"; conformance nodes generated from `vpat-data.ts`, not hand-written; legal review of trust-page schema |
| Wikidata item gets deleted for non-notability | M | M | Create only after ≥3 independent references exist (press/directory/review-site); SEO.13 sequences those first |
| Author named publicly then leaves / requests erasure | M | M | FR-20 registry `retired` state + AC-10; consent recorded up front |
| Pricing schema drifts from the visible table | M | M | FR-14 single source (`institution-pricing.ts`) + AC-6 assertion |
| `@id` collisions or dangling refs across pages | M | M | `ids.ts` is the only place `@id`s are written; FR-5 build validation |
| Effort spent on deprecated rich results | L | L | Explicit non-goal; FAQ/HowTo emitted for comprehension only, no milestone tied to SERP features |
| Bylining a small team makes us look small | L | L | Real authorship outperforms anonymous "Team" for E-E-A-T; supplement with named external contributors via SEO.8 |

## 15. Rollout Plan

- **Feature flag:** none (build output). Staged by node type.
- **Sequencing**
  1. Land `schema/` module + `@graph` plumbing + validation, emitting only `Organization` and
     `WebSite`. Verify in Rich Results Test.
  2. Ship `/about` entity home.
  3. Ship author registry, `/authors/*`, and reattribute the five existing posts (requires the
     consent step first).
  4. Add `BreadcrumbList` (after SEO.5's visible breadcrumbs), `Article`/`TechArticle`.
  5. Add `Offer`, `ItemList`, `Course` extensions, conformance nodes.
  6. Submit Wikidata item and complete `sameAs` once SEO.13 profiles are claimed.
- **Dogfood:** validate every page type in Rich Results Test + Schema Markup Validator on staging.
- **GA criteria:** AC-1…AC-10 pass; zero GSC structured-data errors at 30 days; brand query
  ("Lextures") returns a knowledge panel or Organization-derived sitelinks within 180 days.
- **Rollback:** revert commit; schema is additive and its removal has no user-visible effect beyond
  the byline/about/author pages, which can stay.

## 16. Test Plan

- **Unit** — one test file per node builder: required-property coverage, `@id` shape, escaping (FR-3),
  graph merge, dangling-reference detection, pricing→Offer mapping, author-slug validation.
- **Integration** — build a representative page of each type and assert the emitted graph against a
  golden fixture; assert the server's course `jsonLd` merges without duplicate `@context`.
- **End-to-end** — Playwright asserts JSON-LD is byte-identical between the prerendered HTML and the
  post-hydration DOM (no drift on client navigation).
- **Security** — hostile-input snapshot (AC-7); confirm no `sameAs` value can come from user data.
- **Accessibility** — axe on `/about`, `/authors`, `/authors/:slug` and on the byline component in
  both themes; verify avatar alt text and heading order.
- **Performance / load** — assert JSON-LD ≤ 12 KB per page in the build; Lighthouse on the three new
  pages against the SEO.4 budget.
- **Manual exploratory** — Rich Results Test, Schema Markup Validator, and a manual prompt set
  ("who founded Lextures?", "is Lextures WCAG compliant?") run against ChatGPT/Claude/Perplexity
  pre- and post-launch, recorded in the SEO.15 baseline.

## 17. Documentation & Training

- `www/docs/structured-data.md` — the node catalogue, `@id` conventions, how to add a node type, and
  the "never assert what is not visible and true" rule.
- `www/docs/authoring-bylines.md` — how to onboard an author (consent, bio, credentials, sameAs) and
  how to retire one.
- Update the content-publishing checklist (SEO.8) with author + citation requirements.
- Internal runbook: responding to a GSC structured-data error; updating the Wikidata item.

## 18. Open Questions

1. Who are the named authors at launch, and what credentials do we publish for each?
2. Do we have (or want) a claimed G2/Capterra profile before listing them in `sameAs`? (SEO.13 owns
   creation; this plan blocks on it.)
3. Does the founder want to be a public `Person` entity with `sameAs` to a personal LinkedIn?
4. Should `SoftwareApplication.offers` reflect the homeschool free tier as `price: "0"`, and does
   that risk a "free" label on paid segments?
5. Is there an existing legal-approved statement of accessibility conformance we can encode as
   `conformsTo`, or does that need counsel review first?

## 19. References

- Existing files: `www/src/lib/document-head.ts` (:11-12, :83-99), `www/src/lib/use-document-head.ts`,
  `www/src/lib/vpat-data.ts`, `www/src/lib/institution-pricing.ts`, `www/src/utils/blog.ts`,
  `www/src/blog/*.md`, `www/src/lib/document-head.test.mjs`
- Audit findings: [F-8, F-9](audit.md#s1--major-ai-search-readiness-is-absent),
  [F-11](audit.md#s2--major-content-depth-and-e-e-a-t)
- Research: [§3](../../plan/seo/research.md#3-what-actually-earns-an-ai-citation),
  [§4](../../plan/seo/research.md#4-entity-seo-is-the-highest-roieffort-ratio-available),
  [§5](../../plan/seo/research.md#5-structured-data-what-still-pays-what-is-dead)
- External: [Google — Course list structured data](https://developers.google.com/search/docs/appearance/structured-data/course),
  [Google — structured data general guidelines](https://developers.google.com/search/docs/appearance/structured-data/sd-policies),
  [schema.org/Organization](https://schema.org/Organization),
  [Wikidata notability policy](https://www.wikidata.org/wiki/Wikidata:Notability),
  [W3C — WCAG 2.2](https://www.w3.org/TR/WCAG22/)
- Related plans: [SEO.1](SEO.1-static-rendering-and-crawlability.md),
  [SEO.5](../../plan/seo/SEO.5-information-architecture-and-internal-linking.md),
  [SEO.11](../../plan/seo/SEO.11-marketplace-catalog-seo.md),
  [SEO.13](SEO.13-offsite-entity-mentions-and-digital-pr.md),
  [S04 — consent ledger](../../plan/standards/S04-unified-consent-preference-ledger.md)
