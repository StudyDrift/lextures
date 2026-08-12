## Summary

<!-- What changed and why -->

## Checklist

- [ ] Tests added/updated as appropriate
- [ ] Changes under `www/src/blog` or `www/src/docs` follow the [answer-first content contract](www/docs/content-contract.md), cite numeric claims, and pass `npm run content:lint`
- [ ] Structure: new/changed code follows [ARCHITECTURE_CONVENTIONS.md](docs/ARCHITECTURE_CONVENTIONS.md) (`make lint-structure`)
- [ ] If moving a plan into `docs/completed/`, added a disposition in `e2e/coverage/completed-feature-manifest.json` and ran `npm run e2e:coverage:check` (E2E.4)
- [ ] Flagged features link settings-toggle / disabled / enabled / authz / dependency / rollback coverage (or an E2E.1–E2E.3 family)

## Marketing SEO (`www/` changes)

- [ ] Route is in `route-manifest.tsx` with unique title/description, canonical, parent/cluster, and internal links
- [ ] OG raster image and appropriate JSON-LD schema are present
- [ ] Content has an owner/author and `reviewDue`
- [ ] Crawler-policy, redirect, or `noindex` changes include their rationale here
