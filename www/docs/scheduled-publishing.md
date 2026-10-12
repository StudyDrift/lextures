# Scheduled publishing

Repository-native guides (`/guides/:slug`) and blog articles can be scheduled with a
`publishDate`. A page goes live on its `publishDate` in **America/Boise**; before that it is
left out of routes (no HTML is generated), the sitemap, and internal listings (the
`/resources/guides` list, the `/homeschool` guides block, and the Canvas comparison links).

## Cadence

Two pages a week, going live **Tuesday** and **Thursday**.

## Schedule a guide

1. Add the guide's metadata to `GUIDES` in `www/src/lib/guides-data.ts`, including
   `publishDate: 'YYYY-MM-DD'` (the Tuesday or Thursday it should go live) and `verifiedOn`.
2. Add the page body to `BODIES` in `www/src/pages/guide-page.tsx`.
3. Open and merge the PR any time before the date. Merging early is safe: the page stays
   hidden until its date.

Blog articles from the Marketing Content API honor an optional `publishDate` field the same
way (`www/src/lib/content-source.ts`). Articles without one publish as before.

## Preview a scheduled page locally

```bash
cd www
PUBLISH_DATE_OVERRIDE=2026-10-15 npm run build   # build as if today were Oct 15 (Boise)
ls dist/guides
```

## How it goes live

`.github/workflows/scheduled-publish-www.yml` runs daily at 12:00 UTC (6:00am MDT /
5:00am MST) and triggers `pages-www.yml` on `main`, the normal production deploy. The
build compares each `publishDate` with today's Boise date, so a page dated Tuesday is
published by that morning's run. `pages-www.yml` also rebuilds hourly, which acts as a
backstop.

The logic lives in `www/src/lib/publish-date.ts` and is tested in
`www/src/lib/publish-date.test.mjs`.
