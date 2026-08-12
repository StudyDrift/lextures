# SEO checks

Run `npm run build && npm run seo:check`. Each check writes structured results to `dist/.seo-check.json`; use `--only=titles,canonicals` to select checks or `--warn-only` during an explicitly documented rollout exception.

The suite verifies generated output and body content, unique metadata, self-canonicals, sitemap parity, JSON-LD IDs, link depth/orphans, internal links, redirects, raster OG images, and removals compared with `SEO_PREVIOUS_MANIFEST`. Fix the named manifest/content/redirect source rather than editing `dist`. Description-length findings remain rollout warnings for existing pages; duplicates and missing values fail.

Exceptions require a PR rationale, owner, expiry date, and lifecycle-log entry. Network-dependent checks belong in `seo:smoke`, not PR checks.
