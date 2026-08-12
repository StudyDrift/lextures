# Marketing-site internationalization

English URLs remain at the root. Other locales use lowercase subdirectories (`/es/pricing`, `/fr-ca/pricing`). Locale identifiers in metadata remain canonical BCP 47 (`es`, `fr-CA`, `en-GB`). Add locales to `src/lib/locales.ts`; a regional variant is justified only when pricing, regulation, terminology, or other substantive content differs—not for spelling alone.

Every localized URL is a route-manifest entry whose `translationOf` points to the unprefixed English route. It is routable only when its locale is `published` and its `translationStatus` is `complete`. The build rejects unsupported tags, malformed prefixes, missing English sources, and complete translations for planned locales.

The generator creates a self-canonical page and a reciprocal hreflang cluster in both HTML and the locale sitemap. Every cluster contains all complete variants, a self-reference, and `x-default` pointing to English. `npm run seo:check -- --only=hreflang` checks HTML/sitemap parity, reciprocity, and indexable targets.

Do not redirect by IP or `Accept-Language`. The optional banner suggests an available locale and a user choice persists in the first-party `lextures_locale` cookie. The URL always controls content. Locale values are read only through the allowlist.

The document `lang` and `dir` come from the route locale. RTL is supported by the locale registry. The switcher uses native-language labels and per-option `lang` attributes.

