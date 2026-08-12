# SEO measurement runbook

The repository holds definitions and low-volume measurement data; production credentials remain in
the scheduler's secret store. Run `npm run measurement:validate` before changing a definition and
`npm run measurement:report` after importing source files. Never replace missing observations with
zero. The generated dashboard reports source freshness and marks missing/stale sources.

The fixed prompt instrument is `prompts.yaml` (60 prompts, version `2026-08-v1`). Changes require a
new version and retained historical file. Weekly observations contain `date`, `prompt_id`, `engine`,
`mentioned`, `answer_position`, `sentiment`, `cited_urls`, `competitors`, `raw_answer_hash`, and brand
`entity_accuracy`/`entity_errors`. Full answers are not retained.

Daily imports use JSONL files under `measurement/data/` (gitignored in production). Supported source
names are `gsc`, `bing`, `ga4`, `crux`, `crawl`, `crm`, `mentions`, and `ai_visibility`. Collectors
write a sidecar freshness record even on failure. Owners investigate a source after its cadence plus
one day. Crawl logs retain no IP address and expire after 30 days; access is restricted to Growth and
Security, and the inventory owner must record the processor and retention in the RoPA.

The Monday report covers indexed pages, non-brand clicks, movers, AI share of voice and competitors,
citations, referring domains, crawl status/first-crawl latency, CWV, conversions, and freshness.
AI SoV is mentions divided by observed answers and is shown per engine, overall, and as a four-week
rolling average. Monthly, copy the generated summary into `../performance.md`, add what shipped,
what moved, and next actions, then archive the source snapshot.

Recovery: fix credentials or schema, rerun only the missing date, and confirm freshness advances.
Never backfill an unavailable engine with a synthetic answer. Reverify GSC/Bing and resubmit sitemaps
after a host change. Alerts are: crawl volume down over 50% week over week, bot non-200 rate over 5%,
or any sitemap request returning non-200.
