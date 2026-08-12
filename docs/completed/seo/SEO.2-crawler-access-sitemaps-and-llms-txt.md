# SEO.2 — Crawler Access, Sitemaps, llms.txt & Index Submission

> Implementation plan. Source: [docs/plan/seo/audit.md](../../plan/seo/audit.md) §S1 (F-4, F-5, F-6, F-7).
> **Shipped** 2026-08-11: typed crawler policy → robots.txt, sitemap index + real lastmod, llms.txt / llms-full.txt / `.md` siblings, IndexNow post-deploy. Manual GSC/Bing DNS verification remains an ops step. See `www/docs/crawler-policy.md`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.2 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | SHIPPED — crawler policy, sitemap index, llms.txt, IndexNow (2026-08-11); GSC/Bing account verification is ops |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Web platform |
| **Depends on** | SEO.1 |
| **Unblocks** | SEO.11, SEO.15, SEO.16 |

---

## 1. Problem Statement

Our `robots.txt` is six lines with no posture on any AI crawler (audit F-4), there is no `llms.txt`
(F-5), every sitemap URL claims the same build-date `lastmod` so the freshness signal is worthless
(F-6), and we have never submitted the site to Bing — which is the retrieval index behind ChatGPT
Search and Copilot (F-7). A Q1 2026 audit found each blocked AI bot costs 18–34% of potential
citations on that engine ([research §2](../../plan/seo/research.md#2-ai-crawlers-do-not-run-javascript)); we are not
blocking anyone today, but we have also declared nothing, which means one defensive edit silently
costs us the channel. Once SEO.1 makes pages real, this plan is what tells every engine they exist.

## 2. Goals

- Declare an explicit, documented posture for all three crawler jobs (training, retrieval, live fetch)
  so access is a deliberate decision rather than an accident of a wildcard.
- Ship a sitemap **index** with per-section sitemaps and honest per-document `lastmod`, so recrawl
  budget follows real change.
- Ship `llms.txt` + `llms-full.txt` with descriptions, not bare URL lists.
- Get verified and indexed in Google Search Console **and** Bing Webmaster Tools, with IndexNow
  pinging on every deploy.
- Make new-URL discovery automatic: publish → indexed, without a human remembering to submit.

## 3. Non-Goals

- Blocking any crawler. Our position is maximum retrievability; if a licensing/monetisation posture
  changes later, this plan's structure makes the change a one-line edit.
- Paid indexing services or link-submission tools.
- Page-level content changes (SEO.6) or schema (SEO.3).
- Log-based crawl analytics dashboards — that is [SEO.15](SEO.15-measurement-search-console-and-ai-share-of-voice.md).

## 4. Personas & User Stories

- **As a prospective buyer using ChatGPT**, I want Lextures pages to be in Bing's index, so that the
  assistant can retrieve and cite them.
- **As an AI agent answering "how do I sync rosters in Lextures?"**, I want a curated map of our help
  content with descriptions, so that I fetch the right page instead of guessing.
- **As a content marketer**, I want a post to be submitted for indexing within minutes of publishing,
  so that timely posts are not stale before they are crawled.
- **As the site owner**, I want an auditable record of which bots we allow and why, so that a future
  policy change is a decision, not a regression.
- **As an SRE**, I want crawler traffic to be identifiable and bounded, so that a crawl surge does not
  look like an incident.

## 5. Functional Requirements

**robots.txt**

- **FR-1.** `robots.txt` MUST be generated from a typed source (`www/src/lib/crawler-policy.ts`)
  rather than hand-maintained, and MUST group directives by crawler job with inline comments
  explaining intent.
- **FR-2.** The policy MUST explicitly `Allow: /` for, at minimum:
  `Googlebot`, `Googlebot-Image`, `Google-Extended`, `Bingbot`, `GPTBot`, `OAI-SearchBot`,
  `ChatGPT-User`, `ClaudeBot`, `Claude-SearchBot`, `Claude-User`, `PerplexityBot`, `Perplexity-User`,
  `Applebot`, `Applebot-Extended`, `Amazonbot`, `Bytespider`, `CCBot`, `meta-externalagent`,
  `DuckDuckBot`, `YandexBot`, `cohere-ai`, `Diffbot`, `Timpibot`.
- **FR-3.** The policy MUST `Disallow` only genuinely non-indexable paths: `/404`, `/*?*` query-string
  variants that duplicate canonical pages (e.g. `?coupon=`), and any thank-you/confirmation route.
  It MUST NOT disallow `/assets/` — AI systems fetch images and CSS for multimodal context.
- **FR-4.** `robots.txt` MUST reference the sitemap **index**, not the flat sitemap.
- **FR-5.** The redundant `Allow: /courses` lines MUST be removed (they are no-ops under `Allow: /`).

**Sitemaps**

- **FR-6.** The build MUST emit a sitemap index at `/sitemap.xml` referencing section sitemaps:
  `/sitemaps/pages.xml`, `/sitemaps/blog.xml`, `/sitemaps/docs.xml`, `/sitemaps/courses.xml`,
  `/sitemaps/compare.xml`, `/sitemaps/glossary.xml`, `/sitemaps/research.xml`. Section files are
  created only when non-empty.
- **FR-7.** Each `<url>` MUST carry a **real** `lastmod`: git commit date of the source markdown for
  content pages, `updatedAt` for courses, and the last commit touching the component/manifest entry
  for static pages. Build date MUST NOT be used as a fallback that changes every deploy — if no real
  date is known, `lastmod` MUST be omitted.
- **FR-8.** `<priority>` and `<changefreq>` MAY be retained for other engines but MUST NOT be relied
  on; Google ignores both.
- **FR-9.** No sitemap file may exceed 50,000 URLs or 50 MB uncompressed; the courses sitemap MUST
  shard automatically (`courses-1.xml`, `courses-2.xml`, …).
- **FR-10.** The build MUST fail if any sitemap URL is absent from `dist/.seo-manifest.json`, or if
  any indexable manifest URL is absent from the sitemaps (bidirectional parity — audit F-1's root
  cause was a sitemap that outran reality).
- **FR-11.** URLs marked `robots: noindex` MUST NOT appear in any sitemap.

**llms.txt**

- **FR-12.** The build MUST emit `/llms.txt` (200 OK, `text/plain`, no redirect) in the documented
  format: an `# Lextures` H1, a `>` blockquote summary, then `##` sections of
  `- [Title](absolute-url): one-sentence description of the question this page answers`.
- **FR-13.** `llms.txt` MUST be curated, not exhaustive: ≤ 200 links, ordered
  Product → Segments → Pricing → Help center → Guides & research → Trust & compliance → Courses (hub
  only). Descriptions are required and MUST be written for a model deciding whether to fetch.
- **FR-14.** The build MUST also emit `/llms-full.txt` — the concatenated plain-text body of the help
  center and blog (markdown source, front-matter stripped, absolute links) — capped at 5 MB with a
  clear truncation notice and a pointer back to `llms.txt`.
- **FR-15.** Every content page SHOULD have a `.md` sibling served as `text/plain`
  (`/docs/roster-sync.md` alongside `/docs/roster-sync`) so agents can fetch clean source. The HTML
  page MUST advertise it via `<link rel="alternate" type="text/markdown" href="…">`.

**Index submission**

- **FR-16.** The site MUST be verified in **Google Search Console** (DNS TXT, so it survives host
  changes) and **Bing Webmaster Tools**, with sitemap index submitted in both.
- **FR-17.** The deploy workflow MUST generate/maintain an **IndexNow** key file at
  `/{key}.txt` and POST changed URLs to `https://api.indexnow.org/indexnow` after every successful
  production deploy, computing the changed set by diffing `.seo-manifest.json` against the previous
  deploy's copy.
- **FR-18.** The workflow MUST also ping Google's sitemap endpoint and MUST NOT submit more than
  10,000 URLs per IndexNow call (batching required).
- **FR-19.** IndexNow submission failures MUST warn, not fail the deploy.

## 6. Non-Functional Requirements

- **Performance** — `robots.txt`, `llms.txt` and sitemaps are static files served from the CDN edge;
  `llms-full.txt` MUST be gzip-compressible and ≤ 5 MB.
- **Security** — the IndexNow key is a public file by design but MUST be generated once and stored as
  a repo constant (not a secret) to avoid rotation breaking submissions. The submission step MUST NOT
  echo any environment secrets. Verification tokens for GSC/Bing use DNS records, not committed files.
- **Privacy & Compliance** — `llms-full.txt` contains only already-public marketing/help content; the
  generator MUST refuse to include any file under `src/content/legal/` history or any page marked
  `noindex`.
- **Accessibility** — n/a (non-visual artefacts).
- **Scalability** — sitemap sharding (FR-9) and IndexNow batching (FR-18) handle catalog growth to
  100k+ course URLs.
- **Reliability** — all four artefacts are produced by the same generator run as SEO.1; if generation
  fails, the deploy fails before publishing an inconsistent set.
- **Observability** — deploy logs record: URLs per sitemap, count submitted to IndexNow, HTTP status
  of each submission. SEO.15 consumes these.
- **Maintainability** — one policy module; adding a bot is one array entry with a required `job` and
  `rationale` field.
- **Internationalization** — sitemap entries carry `xhtml:link rel="alternate" hreflang` once SEO.17
  ships; the builder MUST accept the field now and emit nothing when unset.
- **Backward compatibility** — `/sitemap.xml` stays the canonical entry point (it becomes an index),
  so existing GSC submissions keep working.

## 7. Acceptance Criteria

- **AC-1.** *Given* the deployed site, *When* I `curl https://lextures.com/robots.txt`, *Then* it
  returns 200 `text/plain`, contains an explicit `Allow` block for each of the 23 named agents, and
  references `https://lextures.com/sitemap.xml`.
- **AC-2.** *Given* the deployed site, *When* I fetch `/sitemap.xml`, *Then* it is a valid
  `<sitemapindex>` and every child sitemap returns 200 and validates against the sitemap XSD.
- **AC-3.** *Given* two consecutive deploys with no content change, *When* I diff the sitemaps,
  *Then* no `lastmod` value changed.
- **AC-4.** *Given* a blog post edited and deployed, *When* I read `/sitemaps/blog.xml`, *Then* only
  that post's `lastmod` advanced, and it equals the commit date of the edit.
- **AC-5.** *Given* the deployed site, *When* I fetch `/llms.txt`, *Then* it returns 200 (no redirect
  hop), is ≤ 200 links, and every link has a non-empty description and resolves to a 200 URL.
- **AC-6.** *Given* any indexable page, *When* I fetch `<path>.md`, *Then* I receive `text/plain`
  markdown of that page's body, and the HTML page links to it via `rel="alternate"`.
- **AC-7.** *Given* a production deploy that adds one new URL, *When* the workflow completes, *Then*
  the logs show exactly that URL submitted to IndexNow with an HTTP 200/202.
- **AC-8.** *Given* the build, *When* a URL exists in a sitemap but not in `.seo-manifest.json` (or
  vice versa for indexable pages), *Then* the build fails naming the offending URL.
- **AC-9.** *Given* Bing Webmaster Tools, *When* checked 14 days post-launch, *Then* ≥ 90% of
  submitted URLs are in the "Discovered/Indexed" state.

## 8. Data Model

No database changes. New build artefacts:

| Artefact | Path | Notes |
|---|---|---|
| Crawler policy | `www/src/lib/crawler-policy.ts` | `{ agent, job: 'training'\|'retrieval'\|'user-fetch', allow: boolean, rationale: string }[]` |
| Sitemap index | `dist/sitemap.xml` | `<sitemapindex>` |
| Section sitemaps | `dist/sitemaps/*.xml` | sharded at 50k |
| llms.txt | `dist/llms.txt` | curated |
| llms-full.txt | `dist/llms-full.txt` | concatenated corpus |
| Markdown siblings | `dist/**/*.md` | `text/plain` |
| IndexNow key | `dist/{32-hex}.txt` | public by design |
| Previous manifest | CI cache `seo-manifest-prev.json` | for URL diffing |

`lastmod` resolution order (FR-7): content front-matter `updated:` → git `log -1 --format=%cI <file>`
→ course `updatedAt` → **omit**.

## 9. API Surface

No Lextures API changes. Outbound calls from CI only:

| Call | Purpose | Notes |
|---|---|---|
| `POST https://api.indexnow.org/indexnow` | Submit changed URLs | JSON body `{host, key, keyLocation, urlList}`; ≤10k URLs; expect 200/202 |
| `GET https://www.google.com/ping?sitemap=…` | Legacy sitemap ping | Best-effort; ignore failure |
| `GET {API_BASE}/api/v1/public/marketplace/courses` | Course `updatedAt` for `lastmod` | Already used by SEO.1 |

## 10. UI / UX

- **New visible surface:** none required. Optionally a small footer link to `/llms.txt` under
  "For developers" (also serves as a discoverability signal and a mild differentiator).
- **Flows:** entirely machine-facing.
- **States:** n/a.
- **Accessibility:** if the footer link ships, it must have descriptive link text
  ("AI crawler index (llms.txt)"), not a bare URL.
- **Copy & i18n:** one optional footer string.

## 11. AI / ML Considerations

No models are invoked, but the artefacts are consumed by them:

- `llms.txt` descriptions are effectively **prompts to a retrieval decision**. They must state the
  question the page answers, not the page's title again. Style rule enforced in review:
  *"Pricing for all three segments, including the per-student bands and the homeschool free tier."*
  not *"Our pricing page."*
- `llms-full.txt` is the fallback when an agent cannot or will not crawl; keeping it current is a
  content-freshness obligation, checked by SEO.16's staleness job.

## 12. Integration Points

- **External:** Google Search Console, Bing Webmaster Tools, IndexNow (Microsoft), Cloudflare/GitHub
  Pages static hosting.
- **Internal modules touched:** `www/scripts/generate-site.mjs` (from SEO.1),
  `www/src/lib/crawler-policy.ts` (new), `www/src/lib/route-manifest.ts` (reads `robots` + `sitemap`
  fields), `.github/workflows/pages-www.yml` (submission step), `www/public/robots.txt` (deleted —
  now generated).
- **Events:** post-deploy IndexNow submission; SEO.11 may later add a publish-webhook trigger.

## 13. Dependencies & Sequencing

- **Must ship after:** [SEO.1](./SEO.1-static-rendering-and-crawlability.md) — sitemaps must describe
  URLs that actually return 200, and `lastmod`/parity checks consume `.seo-manifest.json`.
- **Must ship before:** [SEO.11](../../plan/seo/SEO.11-marketplace-catalog-seo.md) (catalog scale needs sharding),
  [SEO.15](SEO.15-measurement-search-console-and-ai-share-of-voice.md) (GSC/Bing are its data
  sources), [SEO.16](../../plan/seo/SEO.16-seo-governance-and-ci-guardrails.md) (parity check is a CI gate).
- **Shared infra:** DNS access for GSC/Bing TXT verification; CI cache for the previous manifest.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Over-permissive robots lets a crawler hammer the marketplace API through course pages | L | M | Course pages are static HTML — no API call per crawl; add `Crawl-delay` for non-major bots and Cloudflare rate rules |
| `llms-full.txt` becomes stale and misrepresents the product | M | M | Regenerated every deploy; SEO.16 staleness check compares its hash to source content |
| IndexNow key leaks/rotates and submissions silently fail | L | L | Key is public by design; workflow asserts 200/202 and warns on anything else; weekly synthetic check |
| Sitemap parity check becomes a build-blocker for legitimate work | M | M | Clear failure message naming the URL and the fix; `sitemap: false` is an explicit, reviewable opt-out |
| Blanket `lastmod` omission reduces recrawl frequency short-term | M | L | Real dates are available for ~all content; omission only affects a handful of pages |
| A future business decision to block training crawlers gets made ad hoc | M | M | `rationale` field is required per agent; changes require a PR touching one reviewed file |

## 15. Rollout Plan

- **Feature flag:** none (build artefacts).
- **Sequencing**
  1. Land `crawler-policy.ts` + generated `robots.txt`; diff against current file in review.
  2. Land sitemap index + real `lastmod` + parity check.
  3. Verify GSC (DNS) and Bing; submit sitemap index in both.
  4. Land `llms.txt` / `llms-full.txt` / `.md` siblings.
  5. Land IndexNow key + post-deploy submission.
  6. Request indexing manually for the 12 highest-value URLs to prime discovery.
- **Dogfood:** validate all artefacts on the staging origin first (staging MUST serve
  `robots.txt: Disallow: /` and `noindex` — a staging leak is a real risk here).
- **GA criteria:** AC-1…AC-9 pass; GSC "Sitemaps" shows all children as Success; Bing shows ≥90%
  discovered at 14 days.
- **Rollback:** revert commit; previous static artefacts restore. IndexNow submissions cannot be
  un-sent, which is acceptable (they only accelerate crawling of URLs we control).

## 16. Test Plan

- **Unit** — policy → `robots.txt` rendering; sitemap XML escaping and sharding; `lastmod` resolution
  order incl. the "omit" branch; `llms.txt` formatting and the ≤200-link cap; IndexNow batching.
- **Integration** — full build produces every artefact; parity check fails on an injected mismatch
  (AC-8); `llms-full.txt` excludes `noindex` and legal-history content.
- **End-to-end** — post-deploy smoke job fetches `/robots.txt`, `/sitemap.xml`, every child sitemap,
  `/llms.txt`, the IndexNow key file, and 10 random `.md` siblings; asserts 200 + content-type.
- **Security** — assert staging never serves an indexable `robots.txt`; assert no secret appears in
  submission logs; validate that `llms-full.txt` cannot include a path outside the allowed roots.
- **Accessibility** — n/a beyond the optional footer link (axe on footer).
- **Performance / load** — assert `llms-full.txt` ≤ 5 MB and that sitemap generation adds < 10 s to
  the build.
- **Manual exploratory** — run each sitemap through Google's Rich Results/sitemap validators; fetch
  the homepage and one help article with `curl -A "OAI-SearchBot"` and `-A "PerplexityBot"` and
  confirm full HTML.

## 17. Documentation & Training

- `www/docs/crawler-policy.md` — the three crawler jobs, our stance, how to add/remove an agent, and
  the business rationale for staying open.
- Update `www/docs/site-generation.md` with the sitemap/llms.txt artefacts.
- Runbook: verifying GSC + Bing after a host migration; re-submitting sitemaps; reading IndexNow logs.
- Add "check `llms.txt` description" to the content-publishing checklist (SEO.8).

## 18. Open Questions

1. Do we want `Crawl-delay` for second-tier bots, or rely on CDN rate limiting? (Google ignores
   `Crawl-delay`; Bing honours it.)
2. Should `llms.txt` include course URLs individually once the catalog is large, or only the hub?
   (Interacts with SEO.11 and the 200-link cap.)
3. Who owns the GSC and Bing accounts, and are they on shared credentials or delegated access?
4. Does serving `.md` siblings create a duplicate-content risk with Google? (Mitigate with
   `X-Robots-Tag: noindex` on `.md` responses — requires FR-12 hosting from SEO.1.)
5. Do we adopt `ai.txt` / `Content-Signal` style headers as they stabilise, or wait for consolidation?

## 19. References

- Existing files: `www/public/robots.txt`, `www/scripts/prerender-courses.mjs`,
  `.github/workflows/pages-www.yml`, `www/docs/marketplace-seo.md`
- Audit findings: [F-4, F-5, F-6, F-7](../../plan/seo/audit.md#s1--major-ai-search-readiness-is-absent)
- Research: [§2](../../plan/seo/research.md#2-ai-crawlers-do-not-run-javascript), [§9](../../plan/seo/research.md#9-measurement-in-2026)
- External: [sitemaps.org protocol](https://www.sitemaps.org/protocol.html),
  [IndexNow documentation](https://www.indexnow.org/documentation),
  [llms.txt specification](https://llmstxt.org/),
  [Google — robots.txt specification](https://developers.google.com/search/docs/crawling-indexing/robots/robots_txt),
  [Anagram — AI crawlers explained (2026)](https://www.anagram.ai/blog/ai-crawlers-explained-gptbot-claudebot-perplexitybot-and-how-to-let-them-in-2026)
- Related plans: [SEO.1](./SEO.1-static-rendering-and-crawlability.md),
  [SEO.11](../../plan/seo/SEO.11-marketplace-catalog-seo.md),
  [SEO.15](SEO.15-measurement-search-console-and-ai-share-of-voice.md),
  [SEO.16](../../plan/seo/SEO.16-seo-governance-and-ci-guardrails.md)
- Implementation docs: [www/docs/crawler-policy.md](../../../www/docs/crawler-policy.md),
  [www/docs/site-generation.md](../../../www/docs/site-generation.md)
