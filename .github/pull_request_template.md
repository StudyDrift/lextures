## Summary

<!-- What changed and why -->

## Checklist

- [ ] Tests added/updated as appropriate
- [ ] Blog/help articles were authored in the Marketing Content workspace; this PR does not add file-based articles under `www/src`
- [ ] Structure: new/changed code follows [ARCHITECTURE_CONVENTIONS.md](docs/ARCHITECTURE_CONVENTIONS.md) (`make lint-structure`)
- [ ] If moving a plan into `docs/completed/`, added a disposition in `e2e/coverage/completed-feature-manifest.json` and ran `npm run e2e:coverage:check` (E2E.4)
- [ ] Flagged features link settings-toggle / disabled / enabled / authz / dependency / rollback coverage (or an E2E.1–E2E.3 family)

## Marketing SEO (`www/` changes)

- [ ] Route is in `route-manifest.tsx` with unique title/description, canonical, parent/cluster, and internal links
- [ ] OG raster image and appropriate JSON-LD schema are present
- [ ] Content has an owner/author and `reviewDue`
- [ ] Crawler-policy, redirect, or `noindex` changes include their rationale here
