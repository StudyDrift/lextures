# SEO.7 — Help Center Expansion (6 → 60+ Articles)

> Implementation plan. Source: [docs/plan/seo/audit.md](audit.md) §S2 (F-14).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.7 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | THIN (6 articles, flat `/docs/:slug`, no categories, no search, no product coverage) |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Docs / Content |
| **Depends on** | SEO.1, SEO.5, SEO.6 |
| **Unblocks** | SEO.9 (integration pages reuse help content), SEO.13 (community answers link to help) |

---

## 1. Problem Statement

The public help center is six articles — create a course, find a course, navigate the course
interface, self-hosting, Zapier, Make — against a platform that ships adaptive content, IRT-based
quizzing, spaced review, standards alignment, rubrics, peer review, gradebook curving, what-if grades,
SIS roster sync, LTI, SSO, a parent portal, an accommodations engine, a course marketplace with
coupons, interactive quizzes, collaboration boards, and an AI stack (audit F-14). None of that is
documented on a crawlable public URL. Help content is the single highest-value AI-citable asset a SaaS
owns because it answers literal questions with literal answers — exactly the passage shape that gets
cited ([research §3](research.md#3-what-actually-earns-an-ai-citation)) — and it doubles as support
deflection and as evidence during evaluation. Today an assistant asked "how does Lextures handle IEP
accommodations?" has nothing to retrieve.

## 2. Goals

- Publish **60+ help articles** covering every shipped surface a user can touch, on categorised,
  crawlable URLs.
- Cover the **highest-intent, lowest-supply queries** first: the ones where a buyer is verifying that
  a capability exists (roster sync, SSO, accommodations, FERPA, self-hosting, LTI).
- Reach a state where an assistant asked any "how do I … in Lextures?" question can retrieve a
  correct, current answer.
- Give support a canonical URL to link for every recurring ticket theme, and measure deflection.
- Keep it current: an article that describes a shipped feature must be updated when that feature
  changes, enforced by process rather than goodwill.

## 3. Non-Goals

- Internal runbooks, API reference, or developer docs beyond what a customer needs (the OpenAPI spec
  and repo docs stay where they are; help articles may link to them).
- A ticketing/helpdesk product or in-app contextual help — separate work.
- Marketing content, guides, and comparisons (SEO.8, SEO.9).
- Localised help (SEO.17).

## 4. Personas & User Stories

- **As a district IT administrator evaluating Lextures**, I want to read exactly how roster sync and
  SSO work before a call, so that I can pre-qualify without a demo.
- **As a teacher mid-task**, I want a short article that answers my question in the first paragraph
  with a screenshot, so that I get unblocked in under a minute.
- **As a homeschool parent**, I want plain-language articles about enrolling multiple children and
  tracking progress, so that I do not need to ask support.
- **As an AI assistant**, I want a categorised, plain-text-accessible help corpus, so that I can
  answer product questions accurately and cite the source.
- **As a support engineer**, I want a canonical article for every recurring ticket theme, so that I
  answer once and link forever.
- **As a product engineer shipping a feature**, I want the docs requirement to be part of the PR, so
  that documentation does not lag the product.

## 5. Functional Requirements

**Structure**

- **FR-1.** Help URLs MUST become `/docs/:category/:slug` with category index pages at
  `/docs/:category` and a hub at `/docs`. Existing flat URLs MUST 301 to their new location
  (redirect map, SEO.5 FR-7).
- **FR-2.** Categories at launch:

  | Category | Slug | Scope |
  |---|---|---|
  | Getting started | `getting-started` | Accounts, first course, orientation, roles |
  | Courses & content | `courses` | Modules, pages, syllabus, files, content tools |
  | Assessment & quizzes | `assessment` | Question banks, quizzes, adaptive/IRT, interactive quizzes, integrity |
  | Grading & feedback | `grading` | Gradebook, rubrics, curving, what-if, peer review, submissions |
  | Adaptive learning | `adaptive` | Learner model, concept graph, paths, spaced review, misconceptions |
  | Outcomes & standards | `outcomes` | Standards alignment, mastery, reporting |
  | People & enrollment | `enrollment` | Roster sync, sections, invitations, roles, test student, waitlists |
  | Accounts & security | `accounts` | SSO, MFA, password policy, session management |
  | Accessibility & accommodations | `accessibility` | Accommodations engine, WCAG features, assistive tech |
  | Parents & guardians | `parents` | Parent portal, pairing, notifications, multi-child |
  | Integrations | `integrations` | LTI, SIS, Zapier, Make, webhooks, calendar, API basics |
  | Marketplace & payments | `marketplace` | Listing, pricing, coupons, payouts, refunds, tax |
  | Administration | `admin` | Org hierarchy, permissions, settings, audit logs, data export |
  | Mobile apps | `mobile` | iOS/Android, offline, notifications |
  | Self-hosting | `self-hosting` | Deployment, upgrades, backups, configuration |
  | Privacy & compliance | `compliance` | FERPA, COPPA, GDPR, data retention, DSARs, subprocessors |

- **FR-3.** Each category index MUST list every article in it with title + one-line description, and
  MUST carry ≥150 words of orienting copy (not a bare link list).
- **FR-4.** Every article MUST satisfy the [SEO.6](SEO.6-answer-first-content-system.md) content
  contract, scoring ≥8.0, with `<Steps>` for procedures and `<FAQ>` at the foot.

**Coverage**

- **FR-5.** The launch set MUST be **60 articles minimum**, allocated by buyer-verification value
  first. Tier 1 (weeks 1–4, 24 articles) covers the questions that gate a purchase:

  | # | Article | Category |
  |---|---|---|
  | 1 | How roster sync from your SIS works | enrollment |
  | 2 | Setting up SAML/OIDC single sign-on | accounts |
  | 3 | Connecting Lextures to your LMS with LTI 1.3 | integrations |
  | 4 | How accommodations are applied automatically | accessibility |
  | 5 | Which WCAG 2.2 AA features Lextures supports | accessibility |
  | 6 | How Lextures handles FERPA-protected records | compliance |
  | 7 | What student data Lextures stores, and for how long | compliance |
  | 8 | Exporting all of your institution's data | admin |
  | 9 | Roles and permissions reference | admin |
  | 10 | How adaptive content decides what a learner sees | adaptive |
  | 11 | How question difficulty and IRT calibration work | assessment |
  | 12 | Setting up spaced review for a course | adaptive |
  | 13 | Aligning assignments to standards and outcomes | outcomes |
  | 14 | Reading the mastery report | outcomes |
  | 15 | Building a rubric that grades consistently | grading |
  | 16 | Curving and scaling grades | grading |
  | 17 | Running a peer review assignment | grading |
  | 18 | Setting up the parent portal | parents |
  | 19 | Pairing a parent to a student account | parents |
  | 20 | Creating and hosting an interactive quiz | assessment |
  | 21 | Academic integrity settings and what they do | assessment |
  | 22 | Self-hosting Lextures: requirements and install | self-hosting |
  | 23 | Upgrading and backing up a self-hosted instance | self-hosting |
  | 24 | Course checklist: what "ready to launch" means | courses |

  Tier 2 (weeks 5–8, 20 articles) covers daily-use tasks; Tier 3 (weeks 9–12, 16+ articles) covers
  marketplace, mobile, and long-tail configuration. Full inventory lives in
  `www/docs/help-center-inventory.md`, maintained as the backlog.

- **FR-6.** Every article MUST map to a shipped capability. Documenting unshipped behaviour is
  prohibited; the inventory MUST cite the plan or code path that implements each article's subject.
- **FR-7.** Each article MUST state which **roles** can perform the action and which **segments**
  (K-12 / HE / HS) it applies to, rendered as a visible metadata strip and as front-matter used for
  filtering.

**Discovery**

- **FR-8.** `/docs` MUST provide client-side search over titles, descriptions and headings, built from
  a static index generated at build time (≤150 KB, lazily loaded so it does not affect the SEO.4
  content-page budget).
- **FR-9.** Every article MUST appear in `/sitemaps/docs.xml` and in `llms.txt` under a
  "Help center" section with its one-line description (SEO.2 FR-13).
- **FR-10.** Every article MUST have a plain-markdown sibling at `<url>.md` (SEO.2 FR-15).
- **FR-11.** Articles MUST cross-link: to their category index, to 3–6 related articles, and to the
  relevant `/platform/*` marketing page — and marketing pages MUST link back to the relevant category.

**Freshness**

- **FR-12.** Every article MUST carry `updated` and a `verifiedAgainst` field naming the product
  version or release date the screenshots and steps were verified against.
- **FR-13.** An article MUST be flagged stale after **180 days** without verification, reported in the
  SEO.16 lifecycle dashboard, and CI MUST fail if more than 10% of articles are stale.
- **FR-14.** PRs to `server/internal/httpserver` or `clients/web` that change a user-visible behaviour
  covered by an article SHOULD be flagged by a CODEOWNERS-style mapping so the docs owner is a
  required reviewer. (Advisory in phase 1; enforced for a named list of critical articles in phase 2.)

**Screenshots**

- **FR-15.** Screenshots MUST be generated by an automated Playwright script against a seeded demo
  tenant, not captured by hand, so they can be regenerated on every UI change.
- **FR-16.** Screenshots MUST contain no real learner data; the demo tenant uses synthetic names.
- **FR-17.** Every screenshot MUST have descriptive `alt` text that conveys the same information as
  the image, and MUST be accompanied by the equivalent instruction in text (an image may never be the
  only way to understand a step).

## 6. Non-Functional Requirements

- **Performance** — help pages are static, `interactive: false` (SEO.4 FR-4); the search index loads
  only on `/docs` and on user intent. Images per SEO.4 FR-12 (AVIF/WebP, sized, lazy).
- **Security** — screenshots must not leak tokens, internal hostnames, or tenant identifiers; the
  generation script runs against a dedicated demo tenant and a redaction check runs on output.
- **Privacy & Compliance** — compliance-category articles MUST be reviewed by the compliance owner
  (SEO.6 FR-6) and MUST match the shipped positions in the
  [standards plan set](../standards/README.md). No article may overstate a certification.
- **Accessibility** — WCAG 2.2 AA: `<Steps>` renders a real `<ol>`, images have text equivalents
  (FR-17), the metadata strip is not colour-coded only, and search results are keyboard-navigable with
  an announced result count.
- **Scalability** — structure supports 500+ articles; category indexes paginate above 60 items via
  sub-categories rather than paged lists (SEO.5 FR-17).
- **Reliability** — screenshot generation failure must not block the site build; it warns and reuses
  the previous images.
- **Observability** — track per-article: organic entrances, AI-citation appearances (SEO.15), search
  queries with no results (a direct backlog signal), and "was this helpful?" votes.
- **Maintainability** — one article per file under `www/src/docs/<category>/`, front-matter validated
  by SEO.6 FR-12 plus the help-specific fields.
- **Internationalization** — English at launch; front-matter and structure are locale-ready for
  SEO.17. Screenshot generation must accept a locale parameter.
- **Backward compatibility** — the six existing `/docs/:slug` URLs 301 to `/docs/:category/:slug`.

## 7. Acceptance Criteria

- **AC-1.** *Given* the launch set, *When* counted, *Then* ≥60 articles exist across ≥14 categories,
  and all 24 Tier-1 articles are published.
- **AC-2.** *Given* any legacy help URL (e.g. `/docs/creating-a-new-course`), *When* requested,
  *Then* it 301s to the new categorised URL.
- **AC-3.** *Given* every published article, *When* scored by SEO.6's checker, *Then* all score ≥8.0.
- **AC-4.** *Given* an article, *When* rendered with JS disabled, *Then* the full procedure including
  every step's text is readable, and each screenshot has non-empty descriptive `alt`.
- **AC-5.** *Given* `/docs` search, *When* I type "roster", *Then* results appear within 100 ms,
  are keyboard-navigable, and announce the result count to screen readers.
- **AC-6.** *Given* an article older than 180 days without re-verification, *When* the lifecycle
  report runs, *Then* it is listed as stale; *And* if >10% of articles are stale, CI fails.
- **AC-7.** *Given* an article in the `compliance` category, *When* published, *Then* front-matter
  contains `reviewedBy` set to the compliance owner and `reviewedAt` within 90 days.
- **AC-8.** *Given* the screenshot generation script, *When* run, *Then* it produces all images from
  the demo tenant with zero real learner names, verified by an assertion against the seed data.
- **AC-9.** *Given* `llms.txt`, *When* fetched, *Then* it contains a "Help center" section listing
  category hubs and the Tier-1 articles with descriptions.
- **AC-10.** *Given* six months after launch, *When* support ticket volume for documented themes is
  compared to baseline, *Then* it has decreased (target −25%), measured by ticket tagging.

## 8. Data Model

No database changes. Content layout:

```
www/src/docs/
  getting-started/  courses/  assessment/  grading/  adaptive/  outcomes/
  enrollment/  accounts/  accessibility/  parents/  integrations/
  marketplace/  admin/  mobile/  self-hosting/  compliance/
    <slug>.mdx
www/src/docs/_categories.ts      # id, title, description, order, icon
www/docs/help-center-inventory.md # backlog: article → capability → plan/code ref → tier → status
www/scripts/screenshots/         # Playwright capture specs
```

Help-specific front-matter (in addition to SEO.6 FR-12):

```yaml
category: enrollment
roles: [admin, teacher]              # who can do this
segments: [k12, higher-ed]           # where it applies
verifiedAgainst: "2026-11 release"
supportTicketThemes: [sis-sync-failure, roster-mismatch]   # for deflection measurement
```

## 9. API Surface

- No new public HTTP routes.
- Screenshot generation authenticates against a **demo tenant** on the staging environment using a
  dedicated service account with least privilege; credentials live in CI secrets, never in the repo.
- The static search index is a build artefact (`dist/docs-search-index.json`), not an API.

## 10. UI / UX

- **New pages:** `/docs` hub (search + categories + popular articles), `/docs/:category` indexes (16),
  60+ article pages.
- **Modified:** `docs-index.tsx`, `docs-post.tsx` rewritten for the categorised structure with
  breadcrumbs (SEO.5 FR-14), byline (SEO.3), metadata strip (roles/segments/updated), on-page table
  of contents for articles with >4 headings, and "Was this helpful?" feedback.
- **Flows**
  1. `/docs` → search "roster" → article → related articles → `/platform/…` → `/request-information`.
  2. Support agent pastes an article URL into a ticket reply → user reads → resolves.
  3. Buyer arrives from an AI citation on `/docs/compliance/ferpa` → `/trust` → VPAT.
- **States** — search: empty (show popular), no results (show "tell us what you were looking for" with
  the query logged), loading (skeleton matched to result height, CLS 0).
- **Responsive** — TOC becomes a collapsed disclosure above the content under 900 px; the metadata
  strip wraps rather than truncating.
- **Accessibility** — search is a labelled `role="searchbox"` with `aria-controls` on the results list
  and a polite live region for the count; TOC links move focus to the heading; screenshots have text
  equivalents (FR-17); no step depends on colour.
- **Copy & i18n** — `www.docs.*` keys for hub, category, search, feedback, metadata labels.

## 11. AI / ML Considerations

Not model-backed. Two notes:

- Help content is the corpus most likely to be *quoted verbatim* by assistants, which makes accuracy
  a support and trust obligation, not just an SEO one. FR-12/FR-13 freshness enforcement exists
  because a stale answer that gets cited is worse than no answer.
- Drafting with AI is permitted under the SEO.6 rules (human byline, human verification against the
  actual product, named reviewer for compliance). Every article MUST be verified by performing the
  steps in the product — this is stated as a hard rule in the contributor guide because it is the one
  place where speed most tempts us to skip.

## 12. Integration Points

- **External:** Playwright (screenshots), staging demo tenant.
- **Internal modules touched:** `www/src/docs/*` (restructured), `www/src/utils/docs.ts`,
  `www/src/pages/docs-index.tsx`, `www/src/pages/docs-post.tsx`, new
  `www/src/pages/docs-category.tsx`, `www/src/lib/route-manifest.ts`,
  `www/scripts/generate-site.mjs`, `www/scripts/screenshots/*`, `www/src/lib/redirects.ts`.
- **Events:** "Was this helpful?" votes → GA4 event; empty-search queries → GA4 event (backlog input).

## 13. Dependencies & Sequencing

- **Must ship after:** [SEO.1](SEO.1-static-rendering-and-crawlability.md) (help pages currently 404),
  [SEO.5](SEO.5-information-architecture-and-internal-linking.md) (categories, breadcrumbs, redirects),
  [SEO.6](SEO.6-answer-first-content-system.md) (the contract these articles are written to).
- **Must ship before:** [SEO.9](SEO.9-comparison-alternatives-and-integration-pages.md) (integration
  pages link to integration help), [SEO.13](SEO.13-offsite-entity-mentions-and-digital-pr.md)
  (community answers must have a canonical URL to point at).
- **Shared infra:** staging demo tenant with seeded synthetic data; CI secret for the service account.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| 60 articles is more writing capacity than exists | **H** | H | Tiered delivery (24 / 20 / 16) over 12 weeks; article template + screenshot automation cut per-article cost; SMEs draft, docs owner edits |
| Articles document behaviour that then changes | H | M | FR-12 `verifiedAgainst`, FR-13 staleness gate, FR-14 reviewer mapping on critical articles |
| Compliance articles overstate our position | M | **H** | FR-7/AC-7 compliance-owner review; claims must trace to a shipped standards plan |
| Screenshots leak real data | L | **H** | Dedicated demo tenant + synthetic seed + automated assertion (AC-8) |
| Docs become a de facto product spec that constrains engineering | M | L | Articles describe shipped behaviour only (FR-6); they follow releases, not lead them |
| Search index bloats content pages | M | L | Loaded only on `/docs`, lazily; size-budgeted at 150 KB |
| Thin articles published to hit the count | M | M | SEO.6 score ≥8.0 gate (AC-3); count is a floor, not a target — the inventory is capability-driven |

## 15. Rollout Plan

- **Feature flag:** none. Published article-by-article.
- **Sequencing**
  1. Restructure to `/docs/:category/:slug` with the six existing articles; ship redirects, hub,
     category indexes for populated categories only.
  2. Screenshot automation + demo tenant seeding.
  3. Tier 1 (24 articles) over weeks 1–4, publishing continuously — each article is submitted to
     IndexNow on deploy (SEO.2 FR-17).
  4. `/docs` search once ≥30 articles exist.
  5. Tier 2 (weeks 5–8), Tier 3 (weeks 9–12).
  6. Enable the staleness gate (FR-13) once the corpus is stable.
- **Dogfood:** support answers the next 20 tickets using only published articles and reports gaps.
- **GA criteria:** AC-1…AC-9 pass; ≥40 help URLs indexed in GSC and Bing; ≥10 help URLs appearing in
  AI citations (SEO.15).
- **Rollback:** articles are additive; the URL restructure is the only reversible-risk change and is
  covered by the redirect map.

## 16. Test Plan

- **Unit** — category/front-matter validation; roles/segments enumeration; search-index construction;
  staleness computation.
- **Integration** — build produces every category index and article; redirects from all six legacy
  URLs resolve (AC-2); `llms.txt` contains the help section (AC-9); `.md` siblings exist.
- **End-to-end (Playwright)** — JS-disabled article readability (AC-4); search interaction incl.
  keyboard and announcements (AC-5); "was this helpful" event fires; TOC focus behaviour.
- **Security** — screenshot redaction assertion (AC-8); verify the demo service account cannot read
  production data; scan generated images for text matching a denylist (tokens, `@lextures.com`
  internal addresses).
- **Accessibility** — axe on hub, a category index, and five representative articles; NVDA +
  VoiceOver on search and TOC; verify every screenshot has a text equivalent.
- **Performance / load** — content pages meet the SEO.4 static budget; `/docs` with search meets the
  interactive budget; search responds < 100 ms over a 500-article index.
- **Manual exploratory** — SME review of each Tier-1 article by performing the steps in the product;
  a new-hire dry run ("set up SSO using only the docs").

## 17. Documentation & Training

- `www/docs/help-center-inventory.md` — the living backlog: capability → article → tier → owner →
  status → verification date.
- `www/docs/writing-help-articles.md` — template, tone, screenshot workflow, verification rule,
  role/segment metadata.
- Support runbook: how to request an article, how to report a stale one.
- Engineering: add "does this change a documented behaviour?" to the PR template.

## 18. Open Questions

1. Who owns the help center — a dedicated docs owner, or rotating engineering ownership? (This is the
   single largest delivery risk; 60 articles needs a named owner.)
2. Do we host a public demo tenant that articles can link to for "try it yourself"?
3. Should compliance articles live under `/docs/compliance` or under `/trust` (SEO.5)? Recommendation:
   articles under `/docs/compliance`, linked prominently from `/trust`.
4. What is the ticket-tagging scheme needed to measure deflection (AC-10), and does the current
   support tool support it?
5. Do we publish articles for self-hosting at the same depth as SaaS, given the support burden?

## 19. References

- Existing files: `www/src/docs/*.md`, `www/src/utils/docs.ts`, `www/src/pages/docs-index.tsx`,
  `www/src/pages/docs-post.tsx`, `www/public/docs-*.png`
- Audit findings: [F-14](audit.md#f-14-the-help-center-is-6-articles-against-a-platform-with-hundreds-of-features)
- Research: [§3](research.md#3-what-actually-earns-an-ai-citation),
  [§7](research.md#7-content-strategy-concentration-beats-volume-utility-beats-pages)
- Related plans: [SEO.5](SEO.5-information-architecture-and-internal-linking.md),
  [SEO.6](SEO.6-answer-first-content-system.md),
  [SEO.9](SEO.9-comparison-alternatives-and-integration-pages.md),
  [SEO.16](SEO.16-seo-governance-and-ci-guardrails.md),
  [standards plan set](../standards/README.md),
  [CC — Course Checklist](../checklist/), [MKTC — coupons](../marketplace/README.md)
