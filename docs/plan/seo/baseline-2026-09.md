# SEO measurement baseline — September 2026

Status: scheduled, not yet observable. Baseline window: 2026-09-01 through 2026-09-21. This file was
created before the window and before the editorial measurement gate; it deliberately does not turn
unconnected sources into zeroes. Run `npm run measurement:report` from `www/` after the window closes,
replace each connection state with its sourced value, and record the export timestamp and owner.

| Metric | Baseline | Source |
|---|---|---|
| Google indexed pages | not connected | GSC Indexing / sitemap export |
| Bing indexed pages | not connected | Bing Webmaster Tools |
| Organic clicks / impressions / position by query and page | not connected | GSC daily export |
| Referring domains / backlinks | not connected | approved backlink provider |
| Brand / non-brand split | not connected | GSC export + `measurement/brand-terms.txt` |
| AI Share of Voice overall and by engine | not connected | 60 × 6 weekly prompt run |
| Third-party mentions | not connected | mentions export |
| Core Web Vitals field data | not connected | CrUX + GA4 web-vitals export |
| Conversions by channel | not connected | GA4 + CRM export |

Do not mark this snapshot captured until all rows have actual observations and source timestamps.
