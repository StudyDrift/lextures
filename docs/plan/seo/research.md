# SEO & AI-Search Research — Late 2026 Landscape

> Evidence base for the [`seo/`](README.md) plan set. Compiled 2026-08-10. Every claim below is
> sourced; the plans cite this file rather than re-arguing the strategy.

---

## 1. The structural shift: retrieval replaced ranking

The channel we are optimising for is no longer "ten blue links". Roughly **70% of Google searches
now end without a click** — the answer is delivered in the SERP or synthesised by an AI system
([Evergreen Media](https://www.evergreen.media/en/guide/seo-this-year/)). Three consequences drive
the whole plan set:

1. **Being *retrievable* beats being *ranked*.** 47% of AI citations come from pages ranking
   **below position #5** ([Stridec](https://www.stridec.com/blog/google-ai-overview-ranking-factors/)).
   A page that ranks #12 but is cleanly extractable can out-earn a page that ranks #3 and is not.
2. **The unit of authority moved from the link to the mention.** An Ahrefs study of 75,000 brands
   found web mentions correlate **0.664** with AI-citation rate versus **0.218** for backlinks —
   roughly 3× stronger ([Subscribe PR](https://subscribepr.com/blog/backlinks-vs-citations-for-aeo/),
   [AirOps](https://www.airops.com/blog/reddit-quora-ai-citation-potential)).
3. **Traffic and visibility have decoupled.** Only 12–18% of Perplexity citations produce a click
   ([SparkToro via AuthorityTech](https://authoritytech.io/curated/ai-referral-traffic-brand-citation-measurement-2026)).
   Optimising for sessions alone will under-report the channel by ~5×; we must measure citations and
   share-of-voice as first-class KPIs (see [SEO.15](SEO.15-measurement-search-console-and-ai-share-of-voice.md)).

**Implication for Lextures.** Buyers of a learning platform (district curriculum directors, HE
instructional designers, homeschool parents) are exactly the population now asking an assistant
"what's a good adaptive LMS for a small district?" instead of typing "best LMS" into Google. If we
are not in the retrieval corpus, we are not in the consideration set — and unlike a SERP, there is
no page two to be discovered on.

---

## 2. AI crawlers do not run JavaScript

This is the single most consequential technical fact for our stack.

- **OpenAI's crawlers are HTML-only parsers.** A March 2026 experiment confirmed that if pricing,
  product names, or descriptions only appear after JavaScript executes, `OAI-SearchBot` cannot see
  them ([Subscribe PR](https://subscribepr.com/blog/how-to-get-indexed-on-bing/)).
- **Server-side rendering is now listed as a core 2026 success factor** precisely "because many LLM
  bots cannot render JavaScript" ([ZUMO](https://www.zumoseo.ch/en/blog/seo-trends-2026)).
- **Bing's index is the retrieval layer for ChatGPT Search and Microsoft Copilot.** A page missing
  from Bing is invisible to two of the largest answer engines regardless of Google rank
  ([Subscribe PR](https://subscribepr.com/blog/how-to-get-indexed-on-bing/)).

The three crawler *jobs* are now distinct and must be addressed separately in `robots.txt`:

| Job | Bots | Our posture |
|---|---|---|
| Model training | `GPTBot`, `Google-Extended`, `CCBot`, `anthropic-ai` | Allow (we want to be in the weights) |
| Search/answer retrieval | `OAI-SearchBot`, `Claude-SearchBot`, `PerplexityBot`, `Bingbot`, `Googlebot` | **Allow — non-negotiable** |
| Live user fetch | `ChatGPT-User`, `Claude-User`, `Perplexity-User` | Allow |

A single `User-agent: *` block no longer expresses intent
([Anagram](https://www.anagram.ai/blog/ai-crawlers-explained-gptbot-claudebot-perplexitybot-and-how-to-let-them-in-2026)).
A Q1 2026 audit found **41% of B2B sites still block at least one major AI bot**, each blocked bot
costing an estimated **18–34% of potential AI citations** on that engine
([DigitalApplied](https://www.digitalapplied.com/blog/ai-crawler-access-control-2026-robots-llms-txt-decision-matrix)).

**`llms.txt`** must be served at the domain root with a `200 OK` — crawlers do not follow a redirect
to find it — and **descriptions beat bare URL lists**, because the description is what tells the
model what question each page answers
([AI Rank Lab](https://www.airanklab.com/blog/llms-txt-best-practices-ai-crawlers-index-content)).

---

## 3. What actually earns an AI citation

Ranked by measured effect size:

| Factor | Measured effect | Source |
|---|---|---|
| **Authoritative outbound citations** | **+132% visibility** — the single highest factor | [Wellows](https://wellows.com/blog/google-ai-overviews-ranking-factors/) |
| **Semantic completeness** ≥8.5/10 | **4.2× more likely to be cited** | [Stridec](https://www.stridec.com/blog/google-ai-overview-ranking-factors/) |
| **Multimodal + schema** (text + image + video + structured data) | **156% higher selection**, up to **317% more citations** | [SEOcrawl](https://seocrawl.ai/blog/ai-overview-ranking-factors) |
| **Entity authority** (clear brand positioning) | Primary determinant, ahead of domain authority | [Digivate](https://digivate.com/blog/ai/how-to-rank-in-google-ai-overviews-2026) |
| **Brand mentions** vs backlinks | 0.664 vs 0.218 correlation | [Ahrefs via Subscribe PR](https://subscribepr.com/blog/backlinks-vs-citations-for-aeo/) |

**The passage is the unit, not the page.** AI systems prioritise passages that fully answer a query
in **self-contained 134–167 word units**
([Stridec](https://www.stridec.com/blog/google-ai-overview-ranking-factors/)). Content extracts more
effectively when it uses explicit question→answer structure, bullets, and concrete examples
([AirOps](https://www.airops.com/blog/reddit-quora-ai-citation-potential)). This is a *writing
format* requirement, and it is enforceable — see
[SEO.6](SEO.6-answer-first-content-system.md).

**E-E-A-T is the substrate.** Named authors with real credentials, an organisation with established
reputation, citations to primary sources, and a clean link profile all raise the probability of
being treated as reliable ([Wellows](https://wellows.com/blog/google-ai-overviews-ranking-factors/)).
Our current blog bylines every post to "Lextures Team" — an entity that does not exist in any
knowledge graph.

---

## 4. Entity SEO is the highest ROI/effort ratio available

Entity SEO — getting engines to recognise the brand as a *thing* with verified relationships rather
than a string — is described as "the substrate underneath everything Google does in 2026, and the
substrate every LLM-based search interface reads from"
([DigitalApplied](https://www.digitalapplied.com/blog/entity-seo-knowledge-graph-optimization-guide-2026)).

- **Entity home + Wikidata QID + `sameAs`** are called out as the highest-value starting point
  specifically because implementation cost is low and impact on entity resolution is high
  ([DigitalApplied](https://www.digitalapplied.com/blog/entity-seo-knowledge-graph-optimization-guide-2026)).
- Google officially confirms it uses the `sameAs` property
  ([Stackmatix](https://www.stackmatix.com/blog/organization-schema-knowledge-graph)).
- `founder`, `parentOrganization`, `sameAs` each add an edge to the brand's entity graph.
- Expected timeline: **knowledge panel in 60–180 days, AI-citation lift in 90–120 days**
  ([DigitalApplied](https://www.digitalapplied.com/blog/entity-seo-knowledge-graph-optimization-guide-2026)).

This sets our earliest realistic date for measurable AI-visibility movement: **Q1 2027** if entity
work lands in Q3 2026.

---

## 5. Structured data: what still pays, what is dead

| Type | Status late 2026 | Do we use it? |
|---|---|---|
| `Organization`, `WebSite`, `BreadcrumbList` | **Alive and load-bearing** for entity resolution | Not yet |
| `Course` + `ItemList` carousel | **Alive.** Needs ≥3 courses, `ItemList` with sequential `position` + unique `url` ([Google](https://developers.google.com/search/docs/appearance/structured-data/course)) | Course JSON-LD partially built (MKT10); carousel missing |
| `Course` *info* rich result (price/date/instructor card) | **Retired June 2025**, dropped from Search Console Sept 2025 ([SEJ](https://www.searchenginejournal.com/google-clarifies-course-structured-data-requirements/456806/)) | n/a |
| `FAQPage` | **Rich result deprecated 7 May 2026**; report dropped June 2026; Search Console API support ended Aug 2026. **The schema type itself is still valid and still parsed** ([SEO Strategy](https://www.seostrategy.co.uk/learn/faq-schema-deprecation-2026-rich-result-vs-schema/)) | Not used |
| `HowTo` | Rich result retired 2023; zero SERP lift, still valid vocabulary | Not used |
| `Article` / `TechArticle` + `author` → `Person` | **Alive**, primary E-E-A-T carrier | Not used |
| `SoftwareApplication` / `Offer` | Alive; feeds pricing extraction | Not used |

**Design rule this implies:** emit FAQ/HowTo markup for *machine comprehension*, never for rich
results — and never let a rich-result deprecation be a reason to remove semantics an LLM reads.
Google's own position: unused structured data "does not cause problems for Search"
([Relevant Audience](https://www.relevantaudience.com/seo/google-removes-structured-data-2025-guide-for-websites/)).
Course schema specifically "remains highly valuable for AI-driven search… making your course
significantly more likely to be cited than prose-only content"
([JSON Schema App](https://jsonschemaapp.com/education-schema-markup/)).

---

## 6. Page experience: the bar moved up in March 2026

- Thresholds: **LCP < 2.0s** (lowered from 2.5s in the March 2026 update), **INP < 200ms**,
  **CLS < 0.1** ([DigitalApplied](https://www.digitalapplied.com/blog/core-web-vitals-2026-inp-lcp-cls-optimization-guide)).
- **INP was promoted to an equal ranking signal** alongside LCP and CLS, confirmed by Google on
  2026-03-18. **43% of sites fail the 200ms bar** — it is the most-failed vital
  ([NitroPack](https://nitropack.io/blog/most-important-core-web-vitals-metrics/)).
- Post-March-2026, sites with poor LCP/INP saw **0.8–4 position drops** on competitive queries
  ([DigitalApplied](https://www.digitalapplied.com/blog/core-web-vitals-2026-inp-lcp-cls-optimization-guide)).

Relevance still outranks page experience — but for queries where many relevant pages exist (every
query we care about), page experience is the differentiator.

---

## 7. Content strategy: concentration beats volume, utility beats pages

**Topical authority.** "A site with 20 interconnected articles on a specific subject will
consistently outrank a site with one 5,000-word guide" and "25 well-connected articles on a topic
ranks better than a generalist outlet with 250 scattered articles"
([Posicionament-Web](https://posicionament-web.cat/en/blog/topical-authority-seo-clusters-2026),
[ClickRank](https://www.clickrank.ai/topical-authority/)). The architecture is a
**2,500–4,000-word pillar** with internally-linked clusters. Strategic pillar selection correlates
with **~55% organic lift within six months**.

**Bottom-of-funnel is the resilient segment.** Top-of-funnel traffic is in industry-wide decline,
but comparison and "alternatives" queries held up — and convert at **10–20%**
([Optiseon](https://optiseon.com/blog/b2b-saas-seo-playbook-2026/),
[GenesysGrowth](https://genesysgrowth.com/blog/designing-competitive-comparison-pages)). The
recommended cadence is **one comparison + one alternatives + one integration page per month for 12
months** = 36 compounding high-intent pages.

**Programmatic SEO survived, but only as "programmatic utility."** The 2026 quality floor: *every*
page must let the user perform an action or solve a micro-problem — calculators, live comparisons,
API-driven data. Otherwise "it's just digital litter"
([Averi](https://www.averi.ai/blog/programmatic-seo-for-b2b-saas-startups-the-complete-2026-playbook),
[NicoDigital](https://www.nicodigital.com/digital-marketing/programmatic-seo-2026-playbook/)).

**The enforcement risk is real.** The **March 2026 core update explicitly named scaled content
abuse**; sites publishing hundreds/thousands of AI-generated pages without editorial oversight saw
**50–80% traffic drops** ([DigitalApplied](https://www.digitalapplied.com/blog/scaled-content-abuse-google-march-update-ai-pages-decimated)).
On **15 May 2026 Google formally extended all spam policies to AI Overviews and AI Mode**, including
scaled content abuse, **inauthentic mentions**, cloaking, link spam, site reputation abuse and
doorway abuse ([PPC Land](https://ppc.land/google-spam-policies-now-officially-cover-ai-overviews-and-ai-mode-in-search/)).
Google does **not** penalise content for being AI-generated — it penalises low-value content
regardless of origin ([Google](https://developers.google.com/search/docs/fundamentals/using-gen-ai-content)).

> **The "inauthentic mentions" clause is why our off-site program
> ([SEO.13](SEO.13-offsite-entity-mentions-and-digital-pr.md)) is built around disclosed, genuine
> participation.** Astroturfing Reddit is now an explicitly named spam vector *and* the community
> platforms enforce it harder than Google does.

---

## 8. Off-site: where AI citations actually come from

- **~48% of AI citations come from community platforms** (Reddit, YouTube).
- **85% of brand mentions in AI answers originate on third-party pages**, not the owned domain
  ([AirOps](https://www.airops.com/report/the-2026-state-of-ai-search)).
- Google's **$60M Reddit data-licensing deal** puts Reddit content directly in the training and
  retrieval path; models treat it as a credible reflection of peer insight.
- In 2026, organic mentions on **Reddit and LinkedIn** are the top drivers of AI citations
  ([AirOps](https://www.airops.com/blog/reddit-quora-ai-citation-potential)).

**Original research is the highest-yield linkable/citable asset**: data stories convert into
backlinks more reliably than opinion, because "data needs a source, creating a natural incentive for
citations" ([OutReachFrog](https://outreachfrog.com/blog/digital-pr-for-backlinks-2026-data-stories)).
Proprietary data wins citations "at multiples of ordinary content."

We sit on a genuinely rare dataset — anonymised, aggregated adaptive-learning outcomes. That is the
raw material for [SEO.12](SEO.12-original-research-and-data-program.md).

---

## 9. Measurement in 2026

- **AI Share of Voice is the headline KPI**: % of AI answers mentioning the brand across a fixed
  prompt set. Top performers hit **≥15%**; category leaders **25–30%** in specialised verticals
  ([ClickRank](https://www.clickrank.ai/ai-share-of-voice-sov/)).
- Companion metrics: **Sentiment Score, Citation CTR, Entity Accuracy**
  ([AuthorityTech](https://authoritytech.io/blog/ai-share-of-voice-measure-grow-llm-brand-presence-2026)).
- **GA4 shipped a native "AI Assistant" channel in May 2026** — but it excludes Perplexity and any
  session arriving without a referrer, so a **custom channel group + UTM + hidden form field** is
  required for end-to-end attribution
  ([AuthorityTech](https://authoritytech.io/blog/ai-traffic-attribution-how-to-track-chatgpt-perplexity-gemini)).
- AI referral share (of AI-sourced traffic): **ChatGPT 74.8%, Gemini 11.6%, Perplexity 7.2%,
  Copilot 3.5%, Claude 2.6%** ([SE Ranking](https://seranking.com/blog/ai-traffic-research-study/)).
  But **Google's own AI Overviews/AI Mode produce more AI-influenced traffic than all standalone
  assistants combined** — so Google remains the primary target, with ChatGPT (via Bing) second.
- Traffic from AI search engines grew **16× from 2024 to 2026**
  ([Trakkr](https://trakkr.ai/ai-search-traffic)).

---

## 10. Category context: the LMS/edtech search landscape

The comparison surface is well-defined and heavily contested by incumbents publishing their own
"alternatives" pages (D2L ranks for *Moodle alternatives*, Jotform and Teachfloor for *Canvas LMS
alternatives*). Recurring entities in the consideration set:

- **K-12:** Google Classroom, Schoology (PowerSchool), Canvas, D2L Brightspace, Edsby
- **Higher-ed:** Canvas (Instructure), Blackboard/Anthology, Moodle, D2L Brightspace, Open edX
- **Course marketplace / creator:** Teachable, Thinkific, LearnWorlds, Kajabi, Podia
- **Corporate-adjacent:** Docebo, TalentLMS, 360Learning, Absorb, LearnUpon

A **May 2026 breach affecting ~9,000 institutions and 275M user records** measurably increased
"Canvas alternatives" interest ([Teachfloor](https://www.teachfloor.com/blog/canvas-lms-alternatives)).
Our shipped trust surfaces (VPAT, security page, privacy history, self-hosting docs) are a
differentiated answer to exactly that query class — and they are currently invisible to crawlers.

---

## 11. What this means for Lextures — the five bets

1. **Fix retrievability first.** Nothing else compounds until every URL returns real HTML at
   `200 OK`. (→ [SEO.1](SEO.1-static-rendering-and-crawlability.md), [SEO.2](SEO.2-crawler-access-sitemaps-and-llms-txt.md))
2. **Become a resolvable entity.** Organization schema + `sameAs` + Wikidata + named authors is the
   cheapest available lever on AI citation. (→ [SEO.3](SEO.3-structured-data-and-entity-graph.md))
3. **Write for passage extraction, not for word count.** Answer-first blocks, 134–167-word
   self-contained units, outbound citations to primary sources. (→ [SEO.6](SEO.6-answer-first-content-system.md))
4. **Own bottom-of-funnel and the utility layer.** Comparisons, alternatives, integrations, glossary,
   calculators — the queries that convert and that AI assistants answer literally.
   (→ [SEO.9](SEO.9-comparison-alternatives-and-integration-pages.md), [SEO.10](SEO.10-programmatic-utility-pages.md))
5. **Manufacture mentions honestly.** Original research + disclosed community participation +
   directory/review presence, because 85% of the mentions that matter are not on our domain.
   (→ [SEO.12](SEO.12-original-research-and-data-program.md), [SEO.13](SEO.13-offsite-entity-mentions-and-digital-pr.md))

---

## 12. Sources

Trends & GEO
- [SEO.com — Rising GEO Trends for 2026](https://www.seo.com/blog/geo-trends/)
- [ZUMO — SEO Trends 2026: GEO, LLMO & AEO](https://www.zumoseo.ch/en/blog/seo-trends-2026)
- [Evergreen Media — SEO Trends 2026](https://www.evergreen.media/en/guide/seo-this-year/)

AI crawlers & indexing
- [Anagram — AI Crawlers Explained (2026)](https://www.anagram.ai/blog/ai-crawlers-explained-gptbot-claudebot-perplexitybot-and-how-to-let-them-in-2026)
- [DigitalApplied — AI Crawler Access Control: 2026 Decision Matrix](https://www.digitalapplied.com/blog/ai-crawler-access-control-2026-robots-llms-txt-decision-matrix)
- [AI Rank Lab — llms.txt Best Practices 2026](https://www.airanklab.com/blog/llms-txt-best-practices-ai-crawlers-index-content)
- [Subscribe PR — How to get indexed on Bing (2026)](https://subscribepr.com/blog/how-to-get-indexed-on-bing/)
- [HubSpot — How to get indexed by ChatGPT (2026)](https://blog.hubspot.com/marketing/how-to-get-indexed-by-chatgpt)

AI Overviews / citation factors
- [Stridec — AI Overview Ranking Factors](https://www.stridec.com/blog/google-ai-overview-ranking-factors/)
- [Wellows — AI Overviews Ranking Factors 2026](https://wellows.com/blog/google-ai-overviews-ranking-factors/)
- [SEOcrawl — AI Overview Ranking Factors](https://seocrawl.ai/blog/ai-overview-ranking-factors)
- [Digivate — What Gemini Really Looks For](https://digivate.com/blog/ai/how-to-rank-in-google-ai-overviews-2026)

Structured data
- [Google — Course list structured data](https://developers.google.com/search/docs/appearance/structured-data/course)
- [SEJ — Google clarifies Course structured data requirements](https://www.searchenginejournal.com/google-clarifies-course-structured-data-requirements/456806/)
- [SEO Strategy — FAQ schema after 7 May 2026](https://www.seostrategy.co.uk/learn/faq-schema-deprecation-2026-rich-result-vs-schema/)
- [JSON Schema App — Education schema markup](https://jsonschemaapp.com/education-schema-markup/)
- [Relevant Audience — Google structured data removals](https://www.relevantaudience.com/seo/google-removes-structured-data-2025-guide-for-websites/)

Entity SEO
- [DigitalApplied — Entity SEO & Knowledge Graph Optimization 2026](https://www.digitalapplied.com/blog/entity-seo-knowledge-graph-optimization-guide-2026)
- [Stackmatix — Organization schema & knowledge graph](https://www.stackmatix.com/blog/organization-schema-knowledge-graph)
- [ClickRank — Knowledge Graph SEO 2026](https://www.clickrank.ai/knowledge-graph-seo-guide/)

Core Web Vitals
- [DigitalApplied — Core Web Vitals 2026](https://www.digitalapplied.com/blog/core-web-vitals-2026-inp-lcp-cls-optimization-guide)
- [NitroPack — Most important Core Web Vitals metrics 2026](https://nitropack.io/blog/most-important-core-web-vitals-metrics/)

Content strategy & spam policy
- [Posicionament-Web — Topical authority: pillars & clusters 2026](https://posicionament-web.cat/en/blog/topical-authority-seo-clusters-2026)
- [ClickRank — Topical Authority SEO 2026](https://www.clickrank.ai/topical-authority/)
- [Optiseon — B2B SaaS SEO playbook 2026](https://optiseon.com/blog/b2b-saas-seo-playbook-2026/)
- [Averi — Programmatic SEO for B2B SaaS 2026](https://www.averi.ai/blog/programmatic-seo-for-b2b-saas-startups-the-complete-2026-playbook)
- [GenesysGrowth — Designing competitive comparison pages](https://genesysgrowth.com/blog/designing-competitive-comparison-pages)
- [DigitalApplied — Scaled content abuse](https://www.digitalapplied.com/blog/scaled-content-abuse-google-march-update-ai-pages-decimated)
- [PPC Land — Spam policies now cover AI Overviews & AI Mode](https://ppc.land/google-spam-policies-now-officially-cover-ai-overviews-and-ai-mode-in-search/)
- [Google — Guidance on gen-AI content](https://developers.google.com/search/docs/fundamentals/using-gen-ai-content)

Off-site & measurement
- [AirOps — The 2026 State of AI Search](https://www.airops.com/report/the-2026-state-of-ai-search)
- [AirOps — Reddit & Quora mentions as off-site signal](https://www.airops.com/blog/reddit-quora-ai-citation-potential)
- [Subscribe PR — Backlinks vs citations for AEO](https://subscribepr.com/blog/backlinks-vs-citations-for-aeo/)
- [OutReachFrog — Digital PR for backlinks 2026](https://outreachfrog.com/blog/digital-pr-for-backlinks-2026-data-stories)
- [ClickRank — AI Share of Voice](https://www.clickrank.ai/ai-share-of-voice-sov/)
- [AuthorityTech — AI traffic attribution](https://authoritytech.io/blog/ai-traffic-attribution-how-to-track-chatgpt-perplexity-gemini)
- [SE Ranking — AI traffic research study](https://seranking.com/blog/ai-traffic-research-study/)
- [Trakkr — AI search traffic index](https://trakkr.ai/ai-search-traffic)

Category
- [Teachfloor — Canvas LMS alternatives 2026](https://www.teachfloor.com/blog/canvas-lms-alternatives)
- [D2L — Moodle alternatives 2026](https://www.d2l.com/blog/moodle-alternatives/)
- [Edsby — Top Canvas alternatives for K-12 districts](https://www.edsby.com/top-canvas-alternatives-for-k12-districts/commentary/)
