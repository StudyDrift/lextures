# URL policy

Marketing URLs are lowercase, use hyphens between words, have no file extension, and have no trailing slash except `/`. Content paths may be at most three segments deep. Published slugs are permanent.

## Renaming a URL

In the same change:

1. Change the path in `src/lib/route-manifest.tsx` and its page-loader key.
2. Add the old path to `src/lib/redirects.ts` with a permanent status, date, and reason.
3. Point directly to the final manifest route. The build rejects chains, cycles, external targets, and missing targets.
4. Run `npm test && npm run build`. Confirm `dist/_redirects`, the canonical fallback stub, and `dist/.link-graph.json`.

Query parameters used for attribution (`utm_*`, `coupon`, and `ref`) never change the canonical URL. Existing procurement URLs—`/security`, `/accessibility`, `/privacy`, and `/terms`—are frozen.

`/parents` is intentionally distinct from `/homeschool`: it serves observers of a learner enrolled through an institution, while `/homeschool` serves the purchasing family.
