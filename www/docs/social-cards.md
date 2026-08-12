# Social cards

Every indexable page receives a deterministic 1200×630 PNG during static generation. The title and route section form a content hash, so unchanged cards are reused and changed titles receive a new immutable URL under `/og/`.

Set `ogImage` on a route descriptor to override the generated card. Overrides must be an absolute PNG or JPEG URL with a 1200×630 source image and a maximum size of 300 KB. The generator falls back to `/assets/og-default.png` and logs a warning if card rendering fails.

To debug an unfurl, build the site, inspect the page's `og:image` and `og:image:alt`, then verify the referenced file in `dist/og`. Social networks cache previews independently, so use their refresh/debug tool after a deployment.
