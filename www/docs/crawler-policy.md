# Crawler policy (SEO.2)

How Lextures declares access for search engines and AI crawlers, and how the
build emits sitemaps, `llms.txt`, and IndexNow submissions.

## Stance

**Maximum retrievability.** We do not block training, retrieval, or live-fetch
agents. Access is an explicit, reviewed decision in one file — not an accident
of a wildcard `User-agent: *`.

If a future licensing decision requires blocking a job class (e.g. training),
change `allow: false` on the relevant agents in `src/lib/crawler-policy.ts` and
ship a PR. The `rationale` field is required so the policy stays auditable.

### Cloudflare edge

`lextures.com` sits behind Cloudflare in front of GitHub Pages. Two controls
must stay aligned with this file:

1. **Managed robots.txt** — Cloudflare may prepend a `# BEGIN Cloudflare
   Managed content` block that `Disallow: /`s training agents (`ClaudeBot`,
   `GPTBot`, …). Turn that managed robots injection / “Block AI bots” robots
   rewrite **off** (or allow the agents we list here). Otherwise the public
   `robots.txt` contradicts the origin policy below the marker.
2. **AI bot / Bot Fight policies** — allow at least `GPTBot`,
   `OAI-SearchBot`, `ClaudeBot`, `Claude-SearchBot`, `PerplexityBot`, and the
   `*-User` fetch agents. A `robots.txt` Allow is useless if the CDN returns
   403 before origin.

Post-deploy `seo:smoke` evaluates the origin policy (stripping the managed
preamble) and warns when CI IPs are edge-blocked while spoofing bot UAs.

## Three crawler jobs

| Job | Meaning | Examples |
|---|---|---|
| `retrieval` | Search / citation indexes | Googlebot, Bingbot, OAI-SearchBot, PerplexityBot |
| `training` | Model training corpora | GPTBot, ClaudeBot, Google-Extended, CCBot |
| `user-fetch` | On-demand agent browsing | ChatGPT-User, Claude-User, Perplexity-User |

## Source of truth

| Artefact | Source |
|---|---|
| Agent list + robots.txt | `www/src/lib/crawler-policy.ts` |
| Curated llms.txt | `www/src/lib/llms-catalog.ts` |
| IndexNow key | `www/src/lib/indexnow-key.ts` (public constant) |
| Generator | `www/scripts/generate-site.mjs` + `www/scripts/seo-artifacts.mjs` |
| Post-deploy submit | `www/scripts/submit-indexnow.mjs` (CI after Pages deploy) |

**Do not hand-edit** `dist/robots.txt` or keep a static `public/robots.txt`.
Vite no longer ships a committed robots file; the generator always writes it.

## Adding or removing an agent

1. Edit `CRAWLER_AGENTS` in `crawler-policy.ts`.
2. Provide `agent`, `job`, `allow`, and a one-line `rationale`.
3. Rebuild (`npm run build` in `www/`) and review the generated `dist/robots.txt`.
4. Open a PR — policy changes are intentional product decisions.

## Disallows

Only genuinely non-indexable paths:

- `/404`
- `/*?*` — query-string duplicates of canonical pages

We do **not** disallow `/assets/` — AI systems fetch images and CSS for multimodal context.

## Staging

When `SITE_ORIGIN` is not `https://lextures.com`, or `ROBOTS_DISALLOW_ALL=1`,
robots.txt is `Disallow: /` for every agent so staging never leaks into indexes.

## Sitemaps

- `/sitemap.xml` is a **sitemap index**.
- Section files under `/sitemaps/` (`pages`, `blog`, `docs`, `courses`, …) are
  created only when non-empty.
- Courses auto-shard at 50,000 URLs (`courses-1.xml`, …).
- `lastmod` is real (frontmatter `updated`/`date`, git commit date, or course
  timestamps). If unknown, `lastmod` is **omitted** — never build-date.
- Bidirectional parity with `.seo-manifest.json` fails the build on mismatch.

## llms.txt

- `/llms.txt` — curated ≤200 links with descriptions (questions each page answers).
- `/llms-full.txt` — concatenated help + blog markdown (≤5 MB), no legal-history / noindex.
- Content pages (`/docs/*`, `/blog/*`) also ship a `.md` sibling with
  `<link rel="alternate" type="text/markdown">` on the HTML page.
- `.md` responses should be `noindex` (via `_headers` where the host supports it).

## Index submission

Manual (ops):

1. **Google Search Console** — verify with DNS TXT; submit `https://lextures.com/sitemap.xml`.
2. **Bing Webmaster Tools** — verify (DNS or import from GSC); submit the same sitemap index.

Automatic (CI):

- After every successful production deploy, `index-submit` diffs
  `.seo-manifest.json` against the previous deploy and POSTs changed URLs to
  [IndexNow](https://www.indexnow.org/documentation) (batches of ≤10,000).
- Also pings Google’s legacy sitemap endpoint.
- Failures **warn only** — they never fail the deploy.

### Runbook: host migration

1. Keep the IndexNow key file path stable (`/{key}.txt`) or update Bing / IndexNow.
2. Re-verify DNS TXT for GSC and Bing.
3. Re-submit the sitemap index in both consoles.
4. Trigger a deploy; confirm IndexNow logs show 200/202 for changed URLs.

## Related

- [site-generation.md](./site-generation.md) — full SSG pipeline
- Plan: [SEO.2](../../docs/completed/seo/SEO.2-crawler-access-sitemaps-and-llms-txt.md)
