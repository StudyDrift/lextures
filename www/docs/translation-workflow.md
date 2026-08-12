# Translation workflow

1. Start from an English manifest route and record its current update date.
2. Adapt currency, education terminology, examples, and jurisdictional claims—not only strings.
3. Obtain review by a native speaker with education-domain knowledge. Trust and compliance pages also require jurisdiction-specific compliance sign-off.
4. Add the localized route as `in-progress`. It will not route, appear in hreflang, or enter a sitemap.
5. After review, record `sourceUpdatedAt`, set `translationStatus: complete`, and publish the locale in `src/lib/locales.ts` only after the product supports it end to end.
6. When the English source changes materially, set its translations to `stale`. The lifecycle report lists them; after 90 days, the page displays an English-update notice.

Machine translation may be used for a draft but must never be published without the human review above. Removing a published locale also requires a coordinated 410 response and removal of its entire hreflang cluster.

